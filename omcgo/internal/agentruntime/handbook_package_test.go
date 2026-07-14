package agentruntime

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/agentconfig"
	"github.com/omcgo/omcgo/internal/agentruntime/handbookgen"
)

func TestHandbookPackageManifestAndChunksMatchRuntimeRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/devices", func(c *gin.Context) {})
	router.GET("/api/v1/alarms/active", func(c *gin.Context) {})
	executor := NewToolExecutor(router, router, nil)
	routes := executor.HandbookRouteExport()

	packaged := testHandbookPackage(t, routes)
	archive := packaged.archive
	executor.handbook = packaged
	executor.handbookErr = nil

	manifest, err := executor.handbookManifest()
	require.NoError(t, err)
	require.Equal(t, routes.CatalogVersion, manifest.CatalogVersion)
	require.Equal(t, routes.TotalRoutes, manifest.TotalOperations)
	require.Equal(t, len(archive), manifest.ArchiveBytes)
	require.NotEmpty(t, manifest.HandbookDigest)

	reconstructed := make([]byte, 0, len(archive))
	for index := range manifest.TotalChunks {
		chunk, err := executor.handbookChunk(index)
		require.NoError(t, err)
		require.Equal(t, index, chunk.Index)
		raw, err := base64.StdEncoding.DecodeString(chunk.Data)
		require.NoError(t, err)
		require.Equal(t, chunk.Bytes, len(raw))
		reconstructed = append(reconstructed, raw...)
	}
	require.Equal(t, archive, reconstructed)
	require.Equal(t, fmt.Sprintf("sha256:%x", sha256.Sum256(reconstructed)), manifest.HandbookDigest)
}

func TestHandbookPackageRejectsRuntimeVersionMismatch(t *testing.T) {
	packaged, err := loadEmbeddedHandbookPackage()
	require.NoError(t, err)

	_, err = packaged.validatedManifest(HandbookRouteExport{
		SchemaVersion:  handbookSchemaVersion,
		CatalogVersion: "different",
		TotalRoutes:    1,
	})
	require.ErrorContains(t, err, "does not match running API")
}

func TestToolExecutorServesHandbookPackageThroughBlockedAgentNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/devices", func(c *gin.Context) {})
	executor := NewToolExecutor(router, router, nil)
	routes := executor.HandbookRouteExport()

	executor.handbook = testHandbookPackage(t, routes)
	executor.handbookErr = nil
	policy := agentconfig.RuntimePolicy{
		AllowedMethods:      []string{http.MethodGet},
		BlockedPathPrefixes: []string{"/api/v1/agent/*"},
	}

	manifestResult := executor.Execute(t.Context(), nil, ToolRequest{
		RunID: "run", ToolCallID: "manifest",
		Input: ToolRequestBody{Method: http.MethodGet, Path: "/api/v1/agent/handbook/manifest"},
	}, policy)
	require.Equal(t, "ok", manifestResult.Status)

	chunkResult := executor.Execute(t.Context(), nil, ToolRequest{
		RunID: "run", ToolCallID: "chunk",
		Input: ToolRequestBody{Method: http.MethodGet, Path: "/api/v1/agent/handbook/chunks/0"},
	}, policy)
	require.Equal(t, "ok", chunkResult.Status)

	postResult := executor.Execute(t.Context(), nil, ToolRequest{
		RunID: "run", ToolCallID: "post",
		Input: ToolRequestBody{Method: http.MethodPost, Path: "/api/v1/agent/handbook/manifest"},
	}, agentconfig.RuntimePolicy{AllowedMethods: []string{http.MethodGet, http.MethodPost}})
	require.Equal(t, "error", postResult.Status)
	require.Contains(t, postResult.Error.Message, "only supports GET")
}

func testHandbookPackage(t testing.TB, routes HandbookRouteExport) *handbookPackage {
	t.Helper()
	skillRoot := t.TempDir()
	docsDir := filepath.Join(skillRoot, "references", "api-docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))
	manifestJSON := fmt.Sprintf(
		`{"schemaVersion":%q,"catalogVersion":%q,"totalOperations":%d,"searchIndex":"api-index.jsonl","categories":[]}`,
		routes.SchemaVersion,
		routes.CatalogVersion,
		routes.TotalRoutes,
	)
	require.NoError(t, os.WriteFile(filepath.Join(skillRoot, "references", "manifest.json"), []byte(manifestJSON), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillRoot, "references", "api-index.jsonl"), []byte("{}\n"), 0o644))
	for index := range routes.TotalRoutes {
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, fmt.Sprintf("operation-%d.json", index)), []byte("{}"), 0o644))
	}
	archive, err := handbookgen.BuildPackage(skillRoot)
	require.NoError(t, err)
	packaged, err := newHandbookPackage(archive)
	require.NoError(t, err)
	return packaged
}
