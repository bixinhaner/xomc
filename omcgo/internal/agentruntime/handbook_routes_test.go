package agentruntime

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandbookRouteExportCoversEveryAPIV1Route(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/healthz", func(c *gin.Context) {})
	router.POST("/api/v1/devices", func(c *gin.Context) {})
	router.GET("/api/v1/devices/:id", func(c *gin.Context) {})
	router.DELETE("/api/v1/devices/:id", func(c *gin.Context) {})

	executor := NewToolExecutor(router, router, nil)
	exported := executor.HandbookRouteExport()

	require.Equal(t, handbookSchemaVersion, exported.SchemaVersion)
	require.Equal(t, 3, exported.TotalRoutes)
	require.NotEmpty(t, exported.CatalogVersion)
	require.Len(t, exported.Routes, 3)
	require.Equal(t, http.MethodPost, exported.Routes[0].Method)
	require.Equal(t, "/api/v1/devices", exported.Routes[0].Path)
	require.Equal(t, http.MethodDelete, exported.Routes[1].Method)
	require.Equal(t, "/api/v1/devices/:id", exported.Routes[1].Path)
	require.Equal(t, "delete.devices.by_id", exported.Routes[1].OperationID)
	require.Equal(t, "high", exported.Routes[1].Risk)
	require.NotEmpty(t, exported.Routes[1].Handler)
	require.Equal(t, http.MethodGet, exported.Routes[2].Method)
	require.Equal(t, "/api/v1/devices/:id", exported.Routes[2].Path)
	require.Len(t, exported.Routes[2].PathParams, 1)
	require.Equal(t, "id", exported.Routes[2].PathParams[0].Name)

	repeated := executor.HandbookRouteExport()
	require.Equal(t, exported.CatalogVersion, repeated.CatalogVersion)
	require.Equal(t, exported.Routes, repeated.Routes)
}

func TestHandbookRouteExportReturnsEmptyManifestWithoutRouteProvider(t *testing.T) {
	executor := NewToolExecutor(nil, nil, nil)
	exported := executor.HandbookRouteExport()

	require.Equal(t, handbookSchemaVersion, exported.SchemaVersion)
	require.Zero(t, exported.TotalRoutes)
	require.Empty(t, exported.Routes)
	require.NotEmpty(t, exported.CatalogVersion)
}
