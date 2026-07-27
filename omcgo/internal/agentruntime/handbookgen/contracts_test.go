package handbookgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncAndCheckContractsTrackGoHandlersAndOpenAPI(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	skillRoot := filepath.Join(root, "skill")
	packagePath := filepath.Join(root, "handbook.tar.gz")
	require.NoError(t, os.MkdirAll(filepath.Join(sourceRoot, "internal", "devices"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillRoot, "references"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillRoot, "references", "common-operations.md"), []byte("# Common\n"), 0o644))

	handlerPath := filepath.Join(sourceRoot, "internal", "devices", "handler.go")
	handler := `package devices

import "github.com/gin-gonic/gin"

type Handler struct{}
type Filter struct { Status string ` + "`form:\"status\"`" + ` }

// List returns managed devices.
// @Summary 查询设备列表
func (h *Handler) List(c *gin.Context) {
	var filter Filter
	_ = c.ShouldBindQuery(&filter)
}
`
	require.NoError(t, os.WriteFile(handlerPath, []byte(handler), 0o644))
	openAPIPath := filepath.Join(root, "openapi.yaml")
	require.NoError(t, os.WriteFile(openAPIPath, []byte("openapi: 3.0.3\npaths: {}\n"), 0o644))
	options := ContractOptions{SourceRoot: sourceRoot, OpenAPIPath: openAPIPath, OutputDir: skillRoot, PackagePath: packagePath}

	require.NoError(t, SyncContracts(options))
	require.NoError(t, CheckContracts(options))
	var assets contractAssets
	require.NoError(t, readJSON(filepath.Join(skillRoot, "references", "contracts", "handlers.json"), &assets))
	contract, found := assets.Handlers["github.com/omcgo/omcgo/internal/devices.(*Handler).List-fm"]
	require.True(t, found)
	require.Equal(t, "查询设备列表", contract.Summary)
	require.Len(t, contract.QueryParams, 1)
	require.Equal(t, "status", contract.QueryParams[0].Name)

	updatedHandler := strings.Replace(handler, "查询设备列表", "查询设备状态列表", 1)
	require.NoError(t, os.WriteFile(handlerPath, []byte(updatedHandler), 0o644))
	require.ErrorContains(t, CheckContracts(options), "handler contracts is stale")
	require.NoError(t, SyncContracts(options))
	require.NoError(t, os.WriteFile(openAPIPath, []byte("openapi: 3.0.3\ninfo:\n  title: changed\n  version: 1\npaths: {}\n"), 0o644))
	require.ErrorContains(t, CheckContracts(options), "OpenAPI contract source is stale")
}
