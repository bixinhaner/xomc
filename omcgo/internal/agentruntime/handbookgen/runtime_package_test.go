package handbookgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRuntimePackageUsesActualRoutesAndPreservesTemplateDetails(t *testing.T) {
	templateRoot := t.TempDir()
	references := filepath.Join(templateRoot, handbookPackageRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(references, "api-docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(references, "common-operations.md"), []byte("# Common\n"), 0o644))

	templateDocument := OperationDocument{
		SchemaVersion: SchemaVersion,
		OperationID:   "get.devices",
		Method:        "GET",
		Path:          "/api/v1/devices",
		Category:      "devices",
		Title:         "设备列表",
		Summary:       "读取设备",
		Description:   "构建阶段生成的详细设备说明。",
		Intents:       []string{"设备查询"},
		Tags:          []string{"devices"},
		Risk:          "read",
		Idempotent:    true,
		PathParams:    []Parameter{},
		QueryParams: []Parameter{{
			Name: "status", In: "query", Type: "string", Description: "设备状态",
		}},
		FormParams:       []Parameter{},
		Responses:        standardEnvelopeResponses(),
		ContractCoverage: map[string]string{"request": "openapi", "response": "openapi"},
		Sources:          []string{"runtime-route", "openapi", "go-handler"},
	}
	templateRaw, err := marshalHandbookJSON(templateDocument)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(references, "api-docs", "get.devices.json"), templateRaw, 0o644))
	manifestRaw, err := marshalHandbookJSON(Manifest{
		SchemaVersion: SchemaVersion, CatalogVersion: "template", TotalOperations: 1, SearchIndex: "api-index.jsonl",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(references, "manifest.json"), manifestRaw, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(references, "api-index.jsonl"), []byte("{}\n"), 0o644))

	templateArchive, err := BuildPackage(templateRoot)
	require.NoError(t, err)
	routes := RouteExport{
		SchemaVersion: SchemaVersion, CatalogVersion: "runtime-version", TotalRoutes: 2,
		Routes: []Route{
			{OperationID: "get.devices", Method: "GET", Path: "/api/v1/devices", Category: "devices", Title: "设备列表"},
			{OperationID: "get.optional.status", Method: "GET", Path: "/api/v1/optional/status", Category: "optional", Title: "可选模块状态"},
		},
	}

	first, err := BuildRuntimePackage(templateArchive, routes)
	require.NoError(t, err)
	second, err := BuildRuntimePackage(templateArchive, routes)
	require.NoError(t, err)
	require.Equal(t, first, second, "the same runtime routes must produce an identical package")

	files, err := readPackageFiles(first)
	require.NoError(t, err)
	var manifest Manifest
	require.NoError(t, json.Unmarshal(files[handbookPackageRoot+"/manifest.json"], &manifest))
	require.Equal(t, "runtime-version", manifest.CatalogVersion)
	require.Equal(t, 2, manifest.TotalOperations)
	require.Len(t, manifest.Categories, 2)
	require.Len(t, strings.Split(strings.TrimSpace(string(files[handbookPackageRoot+"/api-index.jsonl"])), "\n"), 2)

	var preserved OperationDocument
	require.NoError(t, json.Unmarshal(files[handbookPackageRoot+"/api-docs/get.devices.json"], &preserved))
	require.Equal(t, templateDocument.Description, preserved.Description)
	require.Equal(t, templateDocument.QueryParams, preserved.QueryParams)
	require.Equal(t, templateDocument.Sources, preserved.Sources)
	var indexEntries []OperationSummary
	for _, line := range strings.Split(strings.TrimSpace(string(files[handbookPackageRoot+"/api-index.jsonl"])), "\n") {
		var entry OperationSummary
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		indexEntries = append(indexEntries, entry)
	}
	require.Contains(t, indexEntries[0].SearchTerms, "status")

	var generated OperationDocument
	require.NoError(t, json.Unmarshal(files[handbookPackageRoot+"/api-docs/get.optional.status.json"], &generated))
	require.Equal(t, "/api/v1/optional/status", generated.Path)
	require.NotEmpty(t, generated.Responses)
	require.Equal(t, "standard-envelope", generated.ContractCoverage["response"])
	require.Contains(t, generated.Sources, "runtime-route")
}

func TestBuildRuntimePackageRejectsUnsafeOperationID(t *testing.T) {
	templateRoot := t.TempDir()
	references := filepath.Join(templateRoot, handbookPackageRoot)
	require.NoError(t, os.MkdirAll(references, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(references, "common-operations.md"), []byte("# Common\n"), 0o644))
	templateArchive, err := BuildPackage(templateRoot)
	require.NoError(t, err)

	_, err = BuildRuntimePackage(templateArchive, RouteExport{
		SchemaVersion:  SchemaVersion,
		CatalogVersion: "runtime",
		TotalRoutes:    1,
		Routes:         []Route{{OperationID: "../escape", Method: "GET", Path: "/api/v1/test"}},
	})
	require.ErrorContains(t, err, "invalid operationId")
}

func TestBuildRuntimePackageUsesGeneratedContractForNewRoute(t *testing.T) {
	templateRoot := t.TempDir()
	references := filepath.Join(templateRoot, handbookPackageRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(references, contractsDirectory), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(references, "common-operations.md"), []byte("# Common\n"), 0o644))
	handlerName := "github.com/omcgo/omcgo/internal/devices.(*Handler).Failures-fm"
	contractsRaw, err := marshalHandbookJSON(contractAssets{
		SchemaVersion: SchemaVersion,
		Handlers: map[string]handlerContract{
			handlerName: {
				Found:       true,
				Summary:     "查询设备失败明细",
				Description: "返回设备执行失败原因。",
				QueryParams: []Parameter{{Name: "status", In: "query", Type: "string"}},
			},
		},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(references, contractsDirectory, contractsFile), contractsRaw, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(references, contractsDirectory, contractsOpenAPI), []byte("openapi: 3.0.3\npaths: {}\n"), 0o644))
	templateArchive, err := BuildPackage(templateRoot)
	require.NoError(t, err)

	runtimeArchive, err := BuildRuntimePackage(templateArchive, RouteExport{
		SchemaVersion: SchemaVersion, CatalogVersion: "runtime", TotalRoutes: 1,
		Routes: []Route{{
			OperationID: "get.devices.failures", Method: "GET", Path: "/api/v1/devices/failures",
			Handler: handlerName, Category: "devices", Title: "devices - 列表",
		}},
	})
	require.NoError(t, err)
	files, err := readPackageFiles(runtimeArchive)
	require.NoError(t, err)
	var document OperationDocument
	require.NoError(t, json.Unmarshal(files[handbookPackageRoot+"/api-docs/get.devices.failures.json"], &document))
	require.Equal(t, "查询设备失败明细", document.Summary)
	require.Equal(t, "返回设备执行失败原因。", document.Description)
	require.Len(t, document.QueryParams, 1)
	require.Equal(t, "status", document.QueryParams[0].Name)
	require.Contains(t, document.Sources, "go-handler")
}
