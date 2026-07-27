package handbookgen

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Generate(options Options) (Manifest, error) {
	if strings.TrimSpace(options.OutputDir) == "" {
		return Manifest{}, fmt.Errorf("output directory is required")
	}
	if options.Routes.TotalRoutes != len(options.Routes.Routes) {
		return Manifest{}, fmt.Errorf("route export total %d does not match routes %d", options.Routes.TotalRoutes, len(options.Routes.Routes))
	}

	openAPI, err := loadOpenAPI(options.OpenAPIPath)
	if err != nil {
		return Manifest{}, err
	}
	sources, err := newGoSourceAnalyzer(options.SourceRoot)
	if err != nil {
		return Manifest{}, err
	}

	routes := append([]Route(nil), options.Routes.Routes...)
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Category == routes[j].Category {
			return routes[i].OperationID < routes[j].OperationID
		}
		return routes[i].Category < routes[j].Category
	})
	seenOperationIDs := make(map[string]struct{}, len(routes))
	seenMethodPaths := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		if strings.TrimSpace(route.OperationID) == "" {
			return Manifest{}, fmt.Errorf("route %s %s has an empty operationId", route.Method, route.Path)
		}
		if _, exists := seenOperationIDs[route.OperationID]; exists {
			return Manifest{}, fmt.Errorf("duplicate operationId %s", route.OperationID)
		}
		seenOperationIDs[route.OperationID] = struct{}{}
		key := strings.ToUpper(route.Method) + " " + route.Path
		if _, exists := seenMethodPaths[key]; exists {
			return Manifest{}, fmt.Errorf("duplicate route %s", key)
		}
		seenMethodPaths[key] = struct{}{}
	}

	referencesDir := filepath.Join(options.OutputDir, "references")
	docsDir := filepath.Join(referencesDir, "api-docs")
	categoriesDir := filepath.Join(referencesDir, "api-categories")
	if err := os.RemoveAll(docsDir); err != nil {
		return Manifest{}, fmt.Errorf("remove generated API documents: %w", err)
	}
	if err := os.RemoveAll(categoriesDir); err != nil {
		return Manifest{}, fmt.Errorf("remove generated API categories: %w", err)
	}
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return Manifest{}, fmt.Errorf("create API document directory: %w", err)
	}
	if err := os.MkdirAll(categoriesDir, 0o755); err != nil {
		return Manifest{}, fmt.Errorf("create API category directory: %w", err)
	}

	categoryDocuments := make(map[string][]OperationSummary)
	categoryTitles := make(map[string]string)
	allSummaries := make([]OperationSummary, 0, len(routes))
	documents := make([]OperationDocument, 0, len(routes))
	for _, route := range routes {
		documents = append(documents, buildOperationDocument(route, openAPI.operation(route.Method, route.Path), sources.analyze(route.Handler)))
	}
	applyRelatedOperations(documents)
	if err := validateOperationDocuments(documents); err != nil {
		return Manifest{}, err
	}
	for index, route := range routes {
		doc := documents[index]
		filename := route.OperationID + ".json"
		if err := writeJSON(filepath.Join(docsDir, filename), doc); err != nil {
			return Manifest{}, err
		}
		category := route.Category
		if category == "" {
			category = "other"
		}
		categoryTitles[category] = chooseCategoryTitle(categoryTitles[category], doc.Title, category)
		summary := operationSummary(doc, "api-docs/"+filename)
		categorySummary := summary
		categorySummary.SearchTerms = nil
		categoryDocuments[category] = append(categoryDocuments[category], categorySummary)
		allSummaries = append(allSummaries, summary)
	}
	sort.Slice(allSummaries, func(i, j int) bool { return allSummaries[i].OperationID < allSummaries[j].OperationID })
	if err := writeJSONLines(filepath.Join(referencesDir, "api-index.jsonl"), allSummaries); err != nil {
		return Manifest{}, err
	}

	categoryIDs := make([]string, 0, len(categoryDocuments))
	for category := range categoryDocuments {
		categoryIDs = append(categoryIDs, category)
	}
	sort.Strings(categoryIDs)
	manifest := Manifest{
		SchemaVersion:   SchemaVersion,
		CatalogVersion:  options.Routes.CatalogVersion,
		TotalOperations: len(routes),
		SearchIndex:     "api-index.jsonl",
		Categories:      make([]CategoryManifest, 0, len(categoryIDs)),
	}
	for _, category := range categoryIDs {
		operations := categoryDocuments[category]
		sort.Slice(operations, func(i, j int) bool { return operations[i].OperationID < operations[j].OperationID })
		filename := category + ".json"
		index := CategoryIndex{
			SchemaVersion: SchemaVersion,
			Category:      category,
			Title:         categoryTitles[category],
			Operations:    operations,
		}
		if err := writeJSON(filepath.Join(categoriesDir, filename), index); err != nil {
			return Manifest{}, err
		}
		manifest.Categories = append(manifest.Categories, CategoryManifest{
			ID:    category,
			Title: categoryTitles[category],
			Count: len(operations),
			File:  "api-categories/" + filename,
		})
	}
	if err := writeJSON(filepath.Join(referencesDir, "manifest.json"), manifest); err != nil {
		return Manifest{}, err
	}
	if err := Check(options.OutputDir, options.Routes); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func buildOperationDocument(route Route, openAPI openAPIOperation, handler handlerContract) OperationDocument {
	method := strings.ToUpper(route.Method)
	summary := firstNonEmpty(openAPI.Summary, handler.Summary, route.Summary, route.Title, method+" "+route.Path)
	description := firstNonEmpty(openAPI.Description, route.Description, summary)
	if openAPI.Description == "" && handler.Summary != "" && handler.Description != "" {
		description = handler.Description
	}
	sources := []string{"runtime-route"}
	if openAPI.Found {
		sources = append(sources, "openapi")
	}
	if handler.Found {
		sources = append(sources, "go-handler")
		if handler.Description != "" && !strings.Contains(strings.ToLower(description), strings.ToLower(handler.Description)) {
			description += " Handler: " + handler.Description
		}
	}

	pathParams := mergeParameters(openAPI.PathParams, route.PathParams)
	pathParams = mergeParameters(pathParams, inferPathParameters(route.Path))
	queryParams := openAPI.QueryParams
	formParams := openAPI.FormParams
	if handler.Found {
		queryParams = mergeAuthoritativeParameters(handler.QueryParams, openAPI.QueryParams)
		formParams = mergeAuthoritativeParameters(handler.FormParams, openAPI.FormParams)
	}
	requestBody := openAPI.RequestBody
	if len(requestBody) == 0 {
		requestBody = handler.RequestBody
	}
	responses := openAPI.Responses
	responseCoverage := "openapi"
	if len(responses) == 0 {
		responses = standardEnvelopeResponses()
		responseCoverage = "standard-envelope"
	}
	requestCoverage := "path-only"
	if openAPI.Found && (len(openAPI.PathParams) > 0 || len(openAPI.QueryParams) > 0 || len(openAPI.FormParams) > 0 || len(openAPI.RequestBody) > 0) {
		requestCoverage = "openapi"
	} else if handler.Found && (len(handler.QueryParams) > 0 || len(handler.FormParams) > 0 || len(handler.RequestBody) > 0) {
		requestCoverage = "go-handler"
	}
	risk := firstNonEmpty(route.Risk, riskForMethod(method))
	read := isReadMethod(method)

	return OperationDocument{
		SchemaVersion:        SchemaVersion,
		OperationID:          route.OperationID,
		Method:               method,
		Path:                 route.Path,
		Handler:              route.Handler,
		Category:             firstNonEmpty(route.Category, "other"),
		Title:                firstNonEmpty(openAPI.Title, handler.Summary, route.Title, summary),
		Summary:              summary,
		Description:          description,
		Intents:              operationIntents(route, summary, description),
		Tags:                 uniqueStrings(append(route.Tags, openAPI.Tags...)),
		Risk:                 risk,
		ConfirmationRequired: !read,
		Idempotent:           method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions || method == http.MethodPut || method == http.MethodDelete,
		SideEffects:          sideEffects(method, summary),
		PathParams:           nonNilParameters(pathParams),
		QueryParams:          nonNilParameters(queryParams),
		FormParams:           nonNilParameters(formParams),
		RequestBody:          requestBody,
		Responses:            responses,
		ReferencedSchemas:    openAPI.ReferencedSchemas,
		ContractCoverage: map[string]string{
			"request":  requestCoverage,
			"response": responseCoverage,
		},
		EmptyResult: emptyResultGuidance(method),
		Sources:     sources,
	}
}

func operationSummary(document OperationDocument, path string) OperationSummary {
	return OperationSummary{
		OperationID: document.OperationID,
		Method:      document.Method,
		Path:        document.Path,
		Title:       document.Title,
		Summary:     document.Summary,
		Description: document.Description,
		Intents:     document.Intents,
		SearchTerms: operationSearchTerms(document),
		Risk:        document.Risk,
		Document:    path,
	}
}

func operationSearchTerms(document OperationDocument) []string {
	values := append([]string{document.Category}, document.Tags...)
	for _, parameters := range [][]Parameter{document.PathParams, document.QueryParams, document.FormParams} {
		for _, parameter := range parameters {
			values = append(values, parameter.Name, parameter.Description)
			appendSearchLiterals(parameter.Enum, &values)
			collectSchemaSearchTerms(parameter.Schema, &values, true)
		}
	}
	collectSchemaSearchTerms(document.RequestBody, &values, true)
	collectSchemaSearchTerms(document.Responses, &values, false)
	collectSchemaSearchTerms(document.ReferencedSchemas, &values, true)
	return uniqueStrings(values)
}

func collectSchemaSearchTerms(value any, output *[]string, includeDescriptions bool) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			child := typed[key]
			switch key {
			case "description", "summary", "title", "name":
				if text, ok := child.(string); ok && includeDescriptions {
					*output = append(*output, text)
				}
			case "enum":
				appendSearchLiterals(child, output)
			case "properties":
				if properties, ok := child.(map[string]any); ok {
					propertyNames := make([]string, 0, len(properties))
					for property := range properties {
						propertyNames = append(propertyNames, property)
					}
					sort.Strings(propertyNames)
					*output = append(*output, propertyNames...)
				}
			case "$ref":
				if reference, ok := child.(string); ok {
					parts := strings.Split(reference, "/")
					*output = append(*output, parts[len(parts)-1])
				}
			}
			collectSchemaSearchTerms(child, output, includeDescriptions)
		}
	case []any:
		for _, item := range typed {
			collectSchemaSearchTerms(item, output, includeDescriptions)
		}
	}
}

func appendSearchLiterals(value any, output *[]string) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			appendSearchLiterals(item, output)
		}
	case []string:
		*output = append(*output, typed...)
	case string:
		if len(typed) <= 160 {
			*output = append(*output, typed)
		}
	case bool, int, int32, int64, float32, float64:
		*output = append(*output, fmt.Sprint(typed))
	}
}

func Check(outputDir string, routes RouteExport) error {
	var manifest Manifest
	if err := readJSON(filepath.Join(outputDir, "references", "manifest.json"), &manifest); err != nil {
		return err
	}
	var failures []error
	if manifest.SchemaVersion != SchemaVersion {
		failures = append(failures, fmt.Errorf("manifest schemaVersion %q does not equal %q", manifest.SchemaVersion, SchemaVersion))
	}
	if manifest.CatalogVersion != routes.CatalogVersion {
		failures = append(failures, fmt.Errorf("manifest catalogVersion %q does not equal route export %q", manifest.CatalogVersion, routes.CatalogVersion))
	}
	if manifest.TotalOperations != len(routes.Routes) {
		failures = append(failures, fmt.Errorf("manifest totalOperations %d does not equal route count %d", manifest.TotalOperations, len(routes.Routes)))
	}
	if manifest.SearchIndex == "" {
		failures = append(failures, fmt.Errorf("manifest searchIndex is empty"))
	} else {
		indexRaw, err := os.ReadFile(filepath.Join(outputDir, "references", manifest.SearchIndex))
		if err != nil {
			failures = append(failures, fmt.Errorf("read search index: %w", err))
		} else {
			indexed := map[string]struct{}{}
			for lineNumber, line := range strings.Split(string(indexRaw), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				var summary OperationSummary
				if err := json.Unmarshal([]byte(line), &summary); err != nil {
					failures = append(failures, fmt.Errorf("decode search index line %d: %w", lineNumber+1, err))
					continue
				}
				indexed[summary.OperationID] = struct{}{}
			}
			if len(indexed) != len(routes.Routes) {
				failures = append(failures, fmt.Errorf("search index contains %d operations, expected %d", len(indexed), len(routes.Routes)))
			}
			for _, route := range routes.Routes {
				if _, ok := indexed[route.OperationID]; !ok {
					failures = append(failures, fmt.Errorf("search index missing operation %s", route.OperationID))
				}
			}
		}
	}

	expected := make(map[string]Route, len(routes.Routes))
	for _, route := range routes.Routes {
		expected[route.OperationID] = route
	}
	docsDir := filepath.Join(outputDir, "references", "api-docs")
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		failures = append(failures, fmt.Errorf("read API documents: %w", err))
		return errors.Join(failures...)
	}
	actual := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		operationID := strings.TrimSuffix(entry.Name(), ".json")
		actual[operationID] = struct{}{}
		if _, ok := expected[operationID]; !ok {
			failures = append(failures, fmt.Errorf("extra document %s", operationID))
			continue
		}
		var doc OperationDocument
		if err := readJSON(filepath.Join(docsDir, entry.Name()), &doc); err != nil {
			failures = append(failures, err)
			continue
		}
		route := expected[operationID]
		if doc.Method != strings.ToUpper(route.Method) || doc.Path != route.Path {
			failures = append(failures, fmt.Errorf("document %s route is %s %s, expected %s %s", operationID, doc.Method, doc.Path, strings.ToUpper(route.Method), route.Path))
		}
		if len(doc.Responses) == 0 {
			failures = append(failures, fmt.Errorf("document %s has no response contract", operationID))
		}
		if doc.ContractCoverage["response"] == "" {
			failures = append(failures, fmt.Errorf("document %s has no response coverage source", operationID))
		}
	}
	for operationID := range expected {
		if _, ok := actual[operationID]; !ok {
			failures = append(failures, fmt.Errorf("missing document %s", operationID))
		}
	}
	categoryTotal := 0
	for _, category := range manifest.Categories {
		categoryTotal += category.Count
		var index CategoryIndex
		if err := readJSON(filepath.Join(outputDir, "references", category.File), &index); err != nil {
			failures = append(failures, err)
			continue
		}
		if len(index.Operations) != category.Count {
			failures = append(failures, fmt.Errorf("category %s contains %d operations, expected %d", category.ID, len(index.Operations), category.Count))
		}
	}
	if categoryTotal != len(routes.Routes) {
		failures = append(failures, fmt.Errorf("category count %d does not equal route count %d", categoryTotal, len(routes.Routes)))
	}
	return errors.Join(failures...)
}

func operationIntents(route Route, summary, description string) []string {
	values := []string{route.Title, summary, description}
	values = append(values, route.Tags...)
	for _, segment := range strings.Split(strings.Trim(route.Path, "/"), "/") {
		if segment == "" || segment == "api" || strings.HasPrefix(segment, "v") || strings.HasPrefix(segment, ":") {
			continue
		}
		values = append(values, strings.ReplaceAll(segment, "-", " "))
	}
	return uniqueStrings(values)
}

func mergeParameters(primary, secondary []Parameter) []Parameter {
	result := append([]Parameter(nil), primary...)
	positions := make(map[string]int, len(result))
	for i, parameter := range result {
		positions[parameter.In+"\x00"+parameter.Name] = i
	}
	for _, parameter := range secondary {
		key := parameter.In + "\x00" + parameter.Name
		if index, exists := positions[key]; exists {
			if result[index].Description == "" {
				result[index].Description = parameter.Description
			}
			if result[index].Type == "" {
				result[index].Type = parameter.Type
			}
			if result[index].Default == nil {
				result[index].Default = parameter.Default
			}
			if len(result[index].Enum) == 0 {
				result[index].Enum = parameter.Enum
			}
			if len(result[index].Schema) == 0 {
				result[index].Schema = parameter.Schema
			}
			result[index].Required = result[index].Required || parameter.Required
			continue
		}
		positions[key] = len(result)
		result = append(result, parameter)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].In == result[j].In {
			return result[i].Name < result[j].Name
		}
		return result[i].In < result[j].In
	})
	return result
}

func mergeAuthoritativeParameters(authoritative, descriptive []Parameter) []Parameter {
	allowed := make(map[string]struct{}, len(authoritative))
	for _, parameter := range authoritative {
		allowed[parameter.In+"\x00"+parameter.Name] = struct{}{}
	}
	filtered := make([]Parameter, 0, len(descriptive))
	for _, parameter := range descriptive {
		if _, ok := allowed[parameter.In+"\x00"+parameter.Name]; ok {
			filtered = append(filtered, parameter)
		}
	}
	return mergeParameters(authoritative, filtered)
}

func applyRelatedOperations(documents []OperationDocument) {
	type candidate struct {
		index int
		score int
	}
	for index := range documents {
		current := documents[index]
		currentTokens := operationPathTokens(current.Path)
		candidates := make([]candidate, 0, len(documents))
		for otherIndex := range documents {
			if index == otherIndex {
				continue
			}
			other := documents[otherIndex]
			shared := sharedStringCount(currentTokens, operationPathTokens(other.Path))
			sameCategory := current.Category != "" && current.Category == other.Category
			if shared == 0 || (!sameCategory && shared < 2) {
				continue
			}
			score := shared * 4
			if sameCategory {
				score += 3
			}
			if pathContains(current.Path, other.Path) {
				score += 3
			}
			if isReadMethod(current.Method) == isReadMethod(other.Method) {
				score++
			}
			candidates = append(candidates, candidate{index: otherIndex, score: score})
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].score == candidates[j].score {
				return documents[candidates[i].index].OperationID < documents[candidates[j].index].OperationID
			}
			return candidates[i].score > candidates[j].score
		})
		if len(candidates) > 5 {
			candidates = candidates[:5]
		}
		related := make([]RelatedOperation, 0, len(candidates))
		for _, item := range candidates {
			other := documents[item.index]
			relation := "same-domain"
			if pathContains(current.Path, other.Path) {
				relation = "same-resource"
			}
			related = append(related, RelatedOperation{
				OperationID: other.OperationID,
				Method:      other.Method,
				Path:        other.Path,
				Relation:    relation,
			})
		}
		documents[index].RelatedOperations = related
	}
}

func operationPathTokens(value string) []string {
	var tokens []string
	for _, token := range strings.Split(strings.Trim(value, "/"), "/") {
		token = strings.TrimSpace(strings.ToLower(token))
		if token == "" || token == "api" || strings.HasPrefix(token, "v") || strings.HasPrefix(token, ":") || strings.HasPrefix(token, "*") {
			continue
		}
		tokens = append(tokens, token)
	}
	return uniqueStrings(tokens)
}

func sharedStringCount(left, right []string) int {
	values := make(map[string]struct{}, len(left))
	for _, value := range left {
		values[value] = struct{}{}
	}
	count := 0
	for _, value := range right {
		if _, ok := values[value]; ok {
			count++
		}
	}
	return count
}

func pathContains(left, right string) bool {
	left = strings.TrimSuffix(left, "/")
	right = strings.TrimSuffix(right, "/")
	return strings.HasPrefix(left+"/", right+"/") || strings.HasPrefix(right+"/", left+"/")
}

func validateOperationDocuments(documents []OperationDocument) error {
	operations := make(map[string]struct{}, len(documents))
	for _, document := range documents {
		if document.OperationID == "" || document.Method == "" || document.Path == "" {
			return fmt.Errorf("operation contract is incomplete: operationId=%q method=%q path=%q", document.OperationID, document.Method, document.Path)
		}
		if _, exists := operations[document.OperationID]; exists {
			return fmt.Errorf("operation contract %s is duplicated", document.OperationID)
		}
		operations[document.OperationID] = struct{}{}
		if document.ContractCoverage["request"] == "" || document.ContractCoverage["response"] == "" {
			return fmt.Errorf("operation contract %s has incomplete coverage metadata", document.OperationID)
		}
		if isReadMethod(document.Method) && strings.TrimSpace(document.EmptyResult) == "" {
			return fmt.Errorf("read operation contract %s has no empty-result semantics", document.OperationID)
		}
	}
	for _, document := range documents {
		for _, related := range document.RelatedOperations {
			if related.OperationID == document.OperationID {
				return fmt.Errorf("operation contract %s references itself", document.OperationID)
			}
			if _, exists := operations[related.OperationID]; !exists {
				return fmt.Errorf("operation contract %s references missing operation %s", document.OperationID, related.OperationID)
			}
			if related.Method == "" || related.Path == "" || related.Relation == "" {
				return fmt.Errorf("operation contract %s has incomplete relation to %s", document.OperationID, related.OperationID)
			}
		}
	}
	return nil
}

func inferPathParameters(path string) []Parameter {
	var parameters []Parameter
	for _, segment := range strings.Split(path, "/") {
		if !strings.HasPrefix(segment, ":") || len(segment) == 1 {
			continue
		}
		parameters = append(parameters, Parameter{Name: strings.TrimPrefix(segment, ":"), In: "path", Required: true, Type: "string"})
	}
	return parameters
}

func chooseCategoryTitle(current, title, fallback string) string {
	if current != "" {
		return current
	}
	if prefix, _, found := strings.Cut(title, " - "); found && strings.TrimSpace(prefix) != "" {
		return strings.TrimSpace(prefix)
	}
	return fallback
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func nonNilParameters(parameters []Parameter) []Parameter {
	if parameters == nil {
		return []Parameter{}
	}
	return parameters
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func riskForMethod(method string) string {
	if isReadMethod(method) {
		return "read"
	}
	if method == http.MethodDelete {
		return "high"
	}
	return "low"
}

func isReadMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func sideEffects(method, summary string) string {
	if isReadMethod(method) {
		return "无；该操作只读取当前用户可见数据。"
	}
	return "会修改业务状态或触发任务；执行前确认目标、范围和影响。操作语义：" + summary
}

func emptyResultGuidance(method string) string {
	if !isReadMethod(method) {
		return ""
	}
	return "空数组或 total=0 只说明该操作在当前用户权限和本次过滤条件下没有返回记录，不能单独证明业务对象、源数据或系统能力不存在。先核对参数、标识符、时间范围和权限；若用户目标仍未回答，再沿 relatedOperations 选择汇总、明细、原始或派生数据操作补充证据。"
}

func standardEnvelopeResponses() map[string]any {
	return map[string]any{
		"2xx": map[string]any{
			"description": "成功。JSON 接口使用 OMC 标准响应信封；文件或 SSE 接口按 Content-Type 返回二进制或事件流。",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{
						"type":     "object",
						"required": []string{"ret", "msg", "data"},
						"properties": map[string]any{
							"ret":  map[string]any{"type": "integer", "const": 1},
							"msg":  map[string]any{"type": "string", "example": "ok"},
							"data": map[string]any{"description": "该 Handler 返回的真实业务结果；以实际 API JSON 为准。"},
						},
					},
				},
			},
		},
		"4xx": map[string]any{
			"description": "请求参数、身份、权限或 Agent policy 不允许。不要绕过限制。",
		},
		"5xx": map[string]any{
			"description": "OMC 内部处理失败。不要编造结果或自动重试非幂等写操作。",
		},
	}
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeJSONLines(path string, values []OperationSummary) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	var output strings.Builder
	for _, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode %s: %w", path, err)
		}
		output.Write(raw)
		output.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(output.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
