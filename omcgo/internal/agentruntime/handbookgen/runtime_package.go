package handbookgen

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

const maxRuntimeTemplateBytes = 64 * 1024 * 1024

// BuildRuntimePackage assembles a handbook for the routes registered by this
// OMC instance. Rich build-time documents are reused when present; routes that
// only exist in the running instance receive the same baseline contract shape.
func BuildRuntimePackage(templateArchive []byte, routes RouteExport) ([]byte, error) {
	if routes.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("runtime route schema %q is not supported", routes.SchemaVersion)
	}
	if routes.TotalRoutes != len(routes.Routes) || routes.TotalRoutes == 0 {
		return nil, fmt.Errorf("runtime route export total %d does not match routes %d", routes.TotalRoutes, len(routes.Routes))
	}
	templateFiles, err := readPackageFiles(templateArchive)
	if err != nil {
		return nil, err
	}
	contracts, err := loadContractAssets(templateFiles)
	if err != nil {
		return nil, err
	}

	templateDocuments := make(map[string]OperationDocument)
	for name, raw := range templateFiles {
		if !strings.HasPrefix(name, handbookPackageRoot+"/api-docs/") || !strings.HasSuffix(name, ".json") {
			continue
		}
		var document OperationDocument
		if err := json.Unmarshal(raw, &document); err != nil {
			return nil, fmt.Errorf("decode template document %s: %w", name, err)
		}
		templateDocuments[document.OperationID] = document
	}

	runtimeRoutes := append([]Route(nil), routes.Routes...)
	sort.Slice(runtimeRoutes, func(i, j int) bool {
		if runtimeRoutes[i].Category == runtimeRoutes[j].Category {
			return runtimeRoutes[i].OperationID < runtimeRoutes[j].OperationID
		}
		return runtimeRoutes[i].Category < runtimeRoutes[j].Category
	})

	files := make(map[string][]byte, routes.TotalRoutes+128)
	if common, ok := templateFiles[handbookPackageRoot+"/common-operations.md"]; ok {
		files[handbookPackageRoot+"/common-operations.md"] = common
	} else {
		return nil, fmt.Errorf("handbook template has no common-operations.md")
	}
	categoryDocuments := make(map[string][]OperationSummary)
	categoryTitles := make(map[string]string)
	allSummaries := make([]OperationSummary, 0, len(runtimeRoutes))
	seenOperations := make(map[string]struct{}, len(runtimeRoutes))
	documents := make([]OperationDocument, 0, len(runtimeRoutes))

	for _, route := range runtimeRoutes {
		if route.OperationID == "" || route.OperationID == "." || route.OperationID == ".." || strings.ContainsAny(route.OperationID, `/\\`) {
			return nil, fmt.Errorf("runtime route %s %s has invalid operationId %q", route.Method, route.Path, route.OperationID)
		}
		if _, exists := seenOperations[route.OperationID]; exists {
			return nil, fmt.Errorf("runtime routes contain duplicate operationId %s", route.OperationID)
		}
		seenOperations[route.OperationID] = struct{}{}
		template, found := templateDocuments[route.OperationID]
		if found && (template.Method != strings.ToUpper(route.Method) || template.Path != route.Path) {
			found = false
		}
		document := template
		if len(contracts.Handlers) > 0 || contracts.OpenAPI.operations != nil {
			openAPI := contracts.OpenAPI.operation(route.Method, route.Path)
			handler := contracts.Handlers[route.Handler]
			generated := buildOperationDocument(route, openAPI, handler)
			document = mergeRuntimeDocument(generated, template, found, openAPI.Found || handler.Found)
		} else if !found {
			document = buildOperationDocument(route, openAPIOperation{}, handlerContract{})
		}
		if isReadMethod(document.Method) {
			document.EmptyResult = emptyResultGuidance(document.Method)
		}
		documents = append(documents, document)
	}
	applyRelatedOperations(documents)
	if err := validateOperationDocuments(documents); err != nil {
		return nil, fmt.Errorf("validate runtime operation contracts: %w", err)
	}

	for index, route := range runtimeRoutes {
		document := documents[index]
		documentRaw, err := marshalHandbookJSON(document)
		if err != nil {
			return nil, fmt.Errorf("encode runtime document %s: %w", route.OperationID, err)
		}
		files[handbookPackageRoot+"/api-docs/"+route.OperationID+".json"] = documentRaw

		category := route.Category
		if category == "" {
			category = "other"
		}
		categoryTitles[category] = chooseCategoryTitle(categoryTitles[category], document.Title, category)
		summary := operationSummary(document, "api-docs/"+route.OperationID+".json")
		categorySummary := summary
		categorySummary.SearchTerms = nil
		categoryDocuments[category] = append(categoryDocuments[category], categorySummary)
		allSummaries = append(allSummaries, summary)
	}

	sort.Slice(allSummaries, func(i, j int) bool { return allSummaries[i].OperationID < allSummaries[j].OperationID })
	var index strings.Builder
	for _, summary := range allSummaries {
		raw, err := json.Marshal(summary)
		if err != nil {
			return nil, fmt.Errorf("encode runtime search index: %w", err)
		}
		index.Write(raw)
		index.WriteByte('\n')
	}
	files[handbookPackageRoot+"/api-index.jsonl"] = []byte(index.String())

	categoryIDs := make([]string, 0, len(categoryDocuments))
	for category := range categoryDocuments {
		categoryIDs = append(categoryIDs, category)
	}
	sort.Strings(categoryIDs)
	manifest := Manifest{
		SchemaVersion:   SchemaVersion,
		CatalogVersion:  routes.CatalogVersion,
		TotalOperations: len(runtimeRoutes),
		SearchIndex:     "api-index.jsonl",
		Categories:      make([]CategoryManifest, 0, len(categoryIDs)),
	}
	for _, category := range categoryIDs {
		operations := categoryDocuments[category]
		sort.Slice(operations, func(i, j int) bool { return operations[i].OperationID < operations[j].OperationID })
		filename := category + ".json"
		categoryRaw, err := marshalHandbookJSON(CategoryIndex{
			SchemaVersion: SchemaVersion,
			Category:      category,
			Title:         categoryTitles[category],
			Operations:    operations,
		})
		if err != nil {
			return nil, fmt.Errorf("encode runtime category %s: %w", category, err)
		}
		files[handbookPackageRoot+"/api-categories/"+filename] = categoryRaw
		manifest.Categories = append(manifest.Categories, CategoryManifest{
			ID: category, Title: categoryTitles[category], Count: len(operations), File: "api-categories/" + filename,
		})
	}
	manifestRaw, err := marshalHandbookJSON(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode runtime manifest: %w", err)
	}
	files[handbookPackageRoot+"/manifest.json"] = manifestRaw

	entries := make([]string, 0, len(files))
	for name := range files {
		entries = append(entries, name)
	}
	sort.Strings(entries)
	return buildPackageFiles(entries, files)
}

func mergeRuntimeDocument(generated, template OperationDocument, templateFound, freshContract bool) OperationDocument {
	if !templateFound {
		return generated
	}
	if !freshContract {
		return template
	}
	generated.PathParams = mergeParameters(generated.PathParams, template.PathParams)
	generated.QueryParams = mergeAuthoritativeParameters(generated.QueryParams, template.QueryParams)
	generated.FormParams = mergeAuthoritativeParameters(generated.FormParams, template.FormParams)
	if len(generated.RequestBody) == 0 {
		generated.RequestBody = template.RequestBody
	}
	if generated.ContractCoverage["response"] == "standard-envelope" && template.ContractCoverage["response"] == "openapi" {
		generated.Responses = template.Responses
		generated.ReferencedSchemas = template.ReferencedSchemas
		generated.ContractCoverage["response"] = "openapi"
	}
	generated.Intents = uniqueStrings(append(generated.Intents, template.Intents...))
	generated.Tags = uniqueStrings(append(generated.Tags, template.Tags...))
	generated.Sources = uniqueStrings(append(generated.Sources, template.Sources...))
	return generated
}

func readPackageFiles(archive []byte) (map[string][]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open handbook template: %w", err)
	}
	defer gzipReader.Close()
	files := make(map[string][]byte)
	totalBytes := int64(0)
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read handbook template: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("handbook template contains non-regular entry %s", header.Name)
		}
		name := path.Clean(header.Name)
		if name == "." || strings.HasPrefix(name, "../") || !strings.HasPrefix(name, handbookPackageRoot+"/") {
			return nil, fmt.Errorf("handbook template contains invalid path %s", header.Name)
		}
		if _, exists := files[name]; exists {
			return nil, fmt.Errorf("handbook template contains duplicate path %s", name)
		}
		totalBytes += header.Size
		if header.Size < 0 || totalBytes > maxRuntimeTemplateBytes {
			return nil, fmt.Errorf("handbook template exceeds extraction limit")
		}
		raw, err := io.ReadAll(io.LimitReader(tarReader, header.Size+1))
		if err != nil {
			return nil, fmt.Errorf("read handbook template file %s: %w", name, err)
		}
		if int64(len(raw)) != header.Size {
			return nil, fmt.Errorf("handbook template file %s has invalid size", name)
		}
		files[name] = raw
	}
	return files, nil
}

func marshalHandbookJSON(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
