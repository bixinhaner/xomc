package handbookgen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateBuildsCompleteProgressiveHandbook(t *testing.T) {
	temp := t.TempDir()
	sourceRoot := filepath.Join(temp, "source")
	output := filepath.Join(temp, "skill")
	require.NoError(t, os.MkdirAll(filepath.Join(sourceRoot, "internal", "device"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(output, "references", "api-docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(output, "references", "api-docs", "stale.json"), []byte(`{"stale":true}`), 0o644))

	handlerSource := `package device

import "github.com/gin-gonic/gin"

type Handler struct{}

type CreateDeviceRequest struct {
	Name string ` + "`json:\"name\" binding:\"required\"`" + ` // User-facing device name.
	Enabled bool ` + "`json:\"enabled\"`" + ` // Whether the device is enabled.
}

// Create registers one managed device.
func (h *Handler) Create(c *gin.Context) {
	var req CreateDeviceRequest
	_ = c.ShouldBindJSON(&req)
	_ = c.DefaultQuery("tenant", "default")
}
`
	require.NoError(t, os.WriteFile(filepath.Join(sourceRoot, "internal", "device", "handler.go"), []byte(handlerSource), 0o644))

	openAPIPath := filepath.Join(temp, "openapi.yaml")
	openAPI := `openapi: 3.0.3
paths:
  /api/v1/devices/{id}:
    get:
      operationId: getDevice
      summary: 查询设备详情
      description: 按设备 ID 返回设备基础信息和当前状态。
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
        - name: verbose
          in: query
          required: false
          description: 是否返回扩展信息。
          schema:
            type: boolean
            default: false
      responses:
        "200":
          description: 设备详情
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Device"
components:
  schemas:
    Device:
      type: object
      required: [id, name]
      properties:
        id:
          type: string
        name:
          type: string
`
	require.NoError(t, os.WriteFile(openAPIPath, []byte(openAPI), 0o644))

	exported := RouteExport{
		SchemaVersion:  "1.0",
		CatalogVersion: "catalog-123",
		TotalRoutes:    2,
		Routes: []Route{
			{
				OperationID: "get.devices.by_id",
				Method:      "GET",
				Path:        "/api/v1/devices/:id",
				Handler:     "github.com/omcgo/omcgo/internal/device.(*Handler).Get-fm",
				Title:       "设备 - 详情",
				Summary:     "设备 - 详情",
				Description: "查询指定设备详情",
				Category:    "devices",
				Risk:        "read",
				Tags:        []string{"devices", "detail"},
			},
			{
				OperationID: "post.devices",
				Method:      "POST",
				Path:        "/api/v1/devices",
				Handler:     "github.com/omcgo/omcgo/internal/device.(*Handler).Create-fm",
				Title:       "设备 - 创建",
				Summary:     "设备 - 创建",
				Description: "创建设备",
				Category:    "devices",
				Risk:        "low",
				Tags:        []string{"devices", "create"},
			},
		},
	}

	manifest, err := Generate(Options{
		Routes:      exported,
		OpenAPIPath: openAPIPath,
		SourceRoot:  sourceRoot,
		OutputDir:   output,
	})
	require.NoError(t, err)
	require.Equal(t, "1.0", manifest.SchemaVersion)
	require.Equal(t, "catalog-123", manifest.CatalogVersion)
	require.Equal(t, 2, manifest.TotalOperations)
	require.Len(t, manifest.Categories, 1)
	require.Equal(t, 2, manifest.Categories[0].Count)
	require.Equal(t, "api-index.jsonl", manifest.SearchIndex)
	require.NoFileExists(t, filepath.Join(output, "references", "api-docs", "stale.json"))
	indexLines := nonEmptyLines(t, filepath.Join(output, "references", "api-index.jsonl"))
	require.Len(t, indexLines, 2)
	var firstIndexEntry OperationSummary
	require.NoError(t, json.Unmarshal([]byte(indexLines[0]), &firstIndexEntry))
	require.Equal(t, "get.devices.by_id", firstIndexEntry.OperationID)
	require.Contains(t, firstIndexEntry.SearchTerms, "verbose")
	require.Contains(t, firstIndexEntry.SearchTerms, "Device")
	require.Contains(t, firstIndexEntry.SearchTerms, "name")

	var getDoc map[string]any
	readJSONFile(t, filepath.Join(output, "references", "api-docs", "get.devices.by_id.json"), &getDoc)
	require.Equal(t, "查询设备详情", getDoc["summary"])
	require.Equal(t, false, getDoc["confirmationRequired"])
	require.Equal(t, true, getDoc["idempotent"])
	require.Equal(t, "read", getDoc["risk"])
	require.Contains(t, getDoc["sources"], "openapi")
	queryParams := getDoc["queryParams"].([]any)
	require.Len(t, queryParams, 1)
	require.Equal(t, "verbose", queryParams[0].(map[string]any)["name"])
	pathParams := getDoc["pathParams"].([]any)
	require.Len(t, pathParams, 1)
	require.Equal(t, "id", pathParams[0].(map[string]any)["name"])
	referencedSchemas := getDoc["referencedSchemas"].(map[string]any)
	require.Contains(t, referencedSchemas, "Device")
	getCoverage := getDoc["contractCoverage"].(map[string]any)
	require.Equal(t, "openapi", getCoverage["response"])

	var postDoc map[string]any
	readJSONFile(t, filepath.Join(output, "references", "api-docs", "post.devices.json"), &postDoc)
	require.Equal(t, true, postDoc["confirmationRequired"])
	require.Equal(t, false, postDoc["idempotent"])
	require.Contains(t, postDoc["description"], "registers one managed device")
	require.Contains(t, postDoc["sources"], "go-handler")
	postQuery := postDoc["queryParams"].([]any)
	require.Len(t, postQuery, 1)
	require.Equal(t, "tenant", postQuery[0].(map[string]any)["name"])
	require.Equal(t, "default", postQuery[0].(map[string]any)["default"])
	body := postDoc["requestBody"].(map[string]any)
	schema := body["schema"].(map[string]any)
	require.Equal(t, []any{"name"}, schema["required"])
	properties := schema["properties"].(map[string]any)
	require.Equal(t, "string", properties["name"].(map[string]any)["type"])
	require.Equal(t, "boolean", properties["enabled"].(map[string]any)["type"])
	postResponses := postDoc["responses"].(map[string]any)
	require.Contains(t, postResponses, "2xx")
	postCoverage := postDoc["contractCoverage"].(map[string]any)
	require.Equal(t, "go-handler", postCoverage["request"])
	require.Equal(t, "standard-envelope", postCoverage["response"])

	var category map[string]any
	readJSONFile(t, filepath.Join(output, "references", "api-categories", "devices.json"), &category)
	operations := category["operations"].([]any)
	require.Len(t, operations, 2)
	require.Equal(t, "get.devices.by_id", operations[0].(map[string]any)["operationId"])
	require.Equal(t, "api-docs/get.devices.by_id.json", operations[0].(map[string]any)["document"])
	require.NotContains(t, operations[0].(map[string]any), "searchTerms")

	require.NoError(t, Check(output, exported))
	firstHash := treeHash(t, filepath.Join(output, "references"))
	_, err = Generate(Options{Routes: exported, OpenAPIPath: openAPIPath, SourceRoot: sourceRoot, OutputDir: output})
	require.NoError(t, err)
	require.Equal(t, firstHash, treeHash(t, filepath.Join(output, "references")))
}

func TestGoSourceSummaryAndSearchTermsImproveGenericRoutes(t *testing.T) {
	temp := t.TempDir()
	sourceRoot := filepath.Join(temp, "source")
	output := filepath.Join(temp, "skill")
	require.NoError(t, os.MkdirAll(filepath.Join(sourceRoot, "internal", "transfer"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceRoot, "internal", "transfer", "handler.go"), []byte(`package transfer

import "github.com/gin-gonic/gin"

type Handler struct{}
type Filter struct {
	Status string `+"`form:\"status\"`"+` // Device execution status such as failed.
	TypeCode string `+"`form:\"typeCode\"`"+` // Runtime log collection type RUNTIME_LOG_COLLECT.
}

// ListDevices handles transfer task execution details.
//
// @Summary 查询传输任务设备执行明细与失败原因
// @Description Failed records include failureReason and failureDetail for log collection diagnosis.
func (h *Handler) ListDevices(c *gin.Context) {
	var filter Filter
	_ = c.ShouldBindQuery(&filter)
}
`), 0o644))

	routes := RouteExport{
		SchemaVersion: SchemaVersion, CatalogVersion: "catalog", TotalRoutes: 1,
		Routes: []Route{{
			OperationID: "get.transfer.devices", Method: "GET", Path: "/api/v1/transfer/devices",
			Handler: "github.com/omcgo/omcgo/internal/transfer.(*Handler).ListDevices-fm",
			Title:   "transfer - 列表", Summary: "transfer - 列表", Description: "transfer：按筛选条件查询资源列表",
			Category: "transfer", Risk: "read",
		}},
	}
	_, err := Generate(Options{Routes: routes, SourceRoot: sourceRoot, OutputDir: output})
	require.NoError(t, err)

	var document OperationDocument
	readJSONFile(t, filepath.Join(output, "references", "api-docs", "get.transfer.devices.json"), &document)
	require.Equal(t, "查询传输任务设备执行明细与失败原因", document.Summary)
	require.Equal(t, "Failed records include failureReason and failureDetail for log collection diagnosis.", document.Description)

	lines := nonEmptyLines(t, filepath.Join(output, "references", "api-index.jsonl"))
	var summary OperationSummary
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &summary))
	require.Contains(t, summary.SearchTerms, "status")
	require.Contains(t, summary.SearchTerms, "Device execution status such as failed.")
	require.Contains(t, summary.SearchTerms, "typeCode")
}

func TestGenerateRejectsDuplicateOperationIDs(t *testing.T) {
	exported := RouteExport{
		SchemaVersion:  "1.0",
		CatalogVersion: "catalog-duplicate",
		TotalRoutes:    2,
		Routes: []Route{
			{OperationID: "get.devices", Method: "GET", Path: "/api/v1/devices", Category: "devices"},
			{OperationID: "get.devices", Method: "GET", Path: "/api/v1/device-list", Category: "devices"},
		},
	}

	_, err := Generate(Options{Routes: exported, OutputDir: t.TempDir()})
	require.ErrorContains(t, err, "duplicate operationId get.devices")
}

func TestCheckRejectsMissingAndExtraDocuments(t *testing.T) {
	output := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(output, "references", "api-docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(output, "references", "manifest.json"), []byte(`{"schemaVersion":"1.0","catalogVersion":"catalog","totalOperations":1,"categories":[]}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(output, "references", "api-docs", "extra.json"), []byte(`{}`), 0o644))

	err := Check(output, RouteExport{
		SchemaVersion:  "1.0",
		CatalogVersion: "catalog",
		TotalRoutes:    1,
		Routes:         []Route{{OperationID: "get.devices", Method: "GET", Path: "/api/v1/devices"}},
	})
	require.ErrorContains(t, err, "missing document get.devices")
	require.ErrorContains(t, err, "extra document extra")
}

func TestDecodeRouteExportAcceptsOMCEnvelopeAndDirectDocument(t *testing.T) {
	direct := []byte(`{"schemaVersion":"1.0","catalogVersion":"catalog","totalRoutes":1,"routes":[{"operationId":"get.devices","method":"GET","path":"/api/v1/devices"}]}`)
	exported, err := DecodeRouteExport(direct)
	require.NoError(t, err)
	require.Equal(t, "catalog", exported.CatalogVersion)
	require.Len(t, exported.Routes, 1)

	envelope := []byte(`{"ret":1,"msg":"ok","data":{"schemaVersion":"1.0","catalogVersion":"catalog","totalRoutes":1,"routes":[{"operationId":"get.devices","method":"GET","path":"/api/v1/devices"}]}}`)
	exported, err = DecodeRouteExport(envelope)
	require.NoError(t, err)
	require.Equal(t, "get.devices", exported.Routes[0].OperationID)

	_, err = DecodeRouteExport([]byte(`{"ret":0,"msg":"denied","data":null}`))
	require.ErrorContains(t, err, "route export response is not successful")
}

func TestGoSourceAnalyzerResolvesNestedAndEmbeddedQueryBindings(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "internal", "inventory")
	require.NoError(t, os.MkdirAll(packageDir, 0o755))
	source := `package inventory

import "github.com/gin-gonic/gin"

type Handler struct{}

type ListRequest struct {
	Page int ` + "`form:\"page\" binding:\"min=1\"`" + `
	PageSize int ` + "`form:\"page_size\" binding:\"min=1,max=1000\"`" + `
}

type Filter struct {
	ListRequest ListRequest
}

type EmbeddedFilter struct {
	ListRequest
	Status string ` + "`form:\"status\"`" + `
}

func (h *Handler) Nested(c *gin.Context) {
	filter := Filter{}
	_ = c.ShouldBindQuery(&filter.ListRequest)
}

func (h *Handler) Embedded(c *gin.Context) {
	filter := EmbeddedFilter{}
	_ = c.ShouldBindQuery(&filter)
}

// List godoc
// @Summary Query inventory
// @Param status query string false "Device status"
func (h *Handler) Documented(c *gin.Context) {}
`
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, "handler.go"), []byte(source), 0o644))
	analyzer, err := newGoSourceAnalyzer(root)
	require.NoError(t, err)

	nested := analyzer.analyze("github.com/omcgo/omcgo/internal/inventory.(*Handler).Nested-fm")
	require.Equal(t, []string{"page", "page_size"}, parameterNames(nested.QueryParams))

	embedded := analyzer.analyze("github.com/omcgo/omcgo/internal/inventory.(*Handler).Embedded-fm")
	require.Equal(t, []string{"page", "page_size", "status"}, parameterNames(embedded.QueryParams))

	documented := analyzer.analyze("github.com/omcgo/omcgo/internal/inventory.(*Handler).Documented-fm")
	require.Equal(t, "Query inventory", documented.Summary)
	require.Empty(t, documented.Description, "structured Swagger directives must not leak into the description")
}

func TestGoSourceAnalyzerResolvesImportedQueryStruct(t *testing.T) {
	root := t.TempDir()
	modelDir := filepath.Join(root, "internal", "core", "model")
	handlerDir := filepath.Join(root, "internal", "inventory")
	require.NoError(t, os.MkdirAll(modelDir, 0o755))
	require.NoError(t, os.MkdirAll(handlerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modelDir, "pagination.go"), []byte(`package model
type ListRequest struct {
	Page int `+"`form:\"page\" binding:\"min=1\"`"+`
	PageSize int `+"`form:\"page_size\" binding:\"min=1,max=1000\"`"+`
}

`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(handlerDir, "handler.go"), []byte(`package inventory
import (
	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/core/model"
)
type Handler struct{}
type Filter struct { model.ListRequest }
func (h *Handler) List(c *gin.Context) {
	filter := Filter{}
	_ = c.ShouldBindQuery(&filter.ListRequest)
}
`), 0o644))

	analyzer, err := newGoSourceAnalyzer(root)
	require.NoError(t, err)
	contract := analyzer.analyze("github.com/omcgo/omcgo/internal/inventory.(*Handler).List-fm")
	require.Equal(t, []string{"page", "page_size"}, parameterNames(contract.QueryParams))
	require.EqualValues(t, 1, contract.QueryParams[0].Schema["minimum"])
	require.EqualValues(t, 1, contract.QueryParams[1].Schema["minimum"])
	require.EqualValues(t, 1000, contract.QueryParams[1].Schema["maximum"])
}

func TestGoSourceAnalyzerExtractsRealHandlerRequestBody(t *testing.T) {
	analyzer, err := newGoSourceAnalyzer(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	contract := analyzer.analyze("github.com/omcgo/omcgo/internal/admin.(*Handler).CreateApiEndpoint-fm")
	require.True(t, contract.Found)
	require.NotEmpty(t, contract.RequestBody)
	schema := contract.RequestBody["schema"].(map[string]any)
	properties := schema["properties"].(map[string]any)
	require.Contains(t, properties, "path")
	require.Contains(t, properties, "method")
	require.Contains(t, properties, "name")
}

func TestGoSourceAnalyzerFollowsHandlerBindingHelper(t *testing.T) {
	analyzer, err := newGoSourceAnalyzer(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	contract := analyzer.analyze("github.com/omcgo/omcgo/internal/agentconfig.(*Handler).Save-fm")
	require.True(t, contract.Found)
	require.NotEmpty(t, contract.RequestBody)
	schema := contract.RequestBody["schema"].(map[string]any)
	properties := schema["properties"].(map[string]any)
	require.Contains(t, properties, "agentStudioBaseUrl")
	require.Contains(t, properties, "allowedMethods")
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, target))
}

func treeHash(t *testing.T, root string) string {
	t.Helper()
	var files []string
	require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	}))
	sort.Strings(files)
	hash := sha256.New()
	for _, file := range files {
		relative, err := filepath.Rel(root, file)
		require.NoError(t, err)
		raw, err := os.ReadFile(file)
		require.NoError(t, err)
		_, _ = hash.Write([]byte(relative))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(raw)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func parameterNames(parameters []Parameter) []string {
	names := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		names = append(names, parameter.Name)
	}
	return names
}

func nonEmptyLines(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var lines []string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
