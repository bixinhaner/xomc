package agentruntime

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentconfig"
)

func TestToolExecutorExecutesLocalReadAPIWithUserToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc, err := admin.NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	router := gin.New()
	router.GET("/api/v1/devices", func(c *gin.Context) {
		require.NotEmpty(t, c.GetHeader("Authorization"))
		require.Equal(t, "online", c.Query("status"))
		c.JSON(http.StatusOK, gin.H{"total": 1, "items": []string{"device-1"}})
	})

	executor := NewToolExecutor(router, router, jwtSvc)
	result := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-1",
		Input: ToolRequestBody{
			Method: http.MethodGet,
			Path:   "/api/v1/devices",
			Query:  map[string]any{"status": "online"},
		},
	}, agentconfig.RuntimePolicy{
		AllowedMethods:     []string{http.MethodGet},
		ToolTimeoutSeconds: 30,
		MaxResponseBytes:   262144,
	})

	require.Equal(t, "ok", result.Status)
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, output["total"])
}

func TestToolExecutorRejectsMethodsOutsidePolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc, err := admin.NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	router := gin.New()
	router.POST("/api/v1/devices", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	executor := NewToolExecutor(router, router, jwtSvc)
	result := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-1",
		Input: ToolRequestBody{
			Method: http.MethodPost,
			Path:   "/api/v1/devices",
		},
	}, agentconfig.RuntimePolicy{
		AllowedMethods:     []string{http.MethodGet},
		ToolTimeoutSeconds: 30,
		MaxResponseBytes:   262144,
	})

	require.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	require.Contains(t, result.Error.Message, "method POST is not enabled")
}

func TestToolExecutorCatalogRespectsPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/devices", func(c *gin.Context) {})
	router.POST("/api/v1/devices", func(c *gin.Context) {})
	router.GET("/api/v1/auth/profile", func(c *gin.Context) {})

	executor := NewToolExecutor(router, router, nil)
	result := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-1",
		Input: ToolRequestBody{
			Method: http.MethodGet,
			Path:   "/api/v1/agent/catalog",
			Query:  map[string]any{"q": "devices"},
		},
	}, agentconfig.RuntimePolicy{
		AllowedMethods:      []string{http.MethodGet},
		BlockedPathPrefixes: []string{"/api/v1/auth/*"},
		ToolTimeoutSeconds:  30,
		MaxResponseBytes:    262144,
	})

	require.Equal(t, "ok", result.Status)
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	items, ok := output["items"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	require.Equal(t, "GET", items[0]["method"])
	require.Equal(t, "/api/v1/devices", items[0]["path"])
}

func TestToolExecutorCatalogHonorsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/devices", func(c *gin.Context) {})
	router.GET("/api/v1/device-groups", func(c *gin.Context) {})
	router.GET("/api/v1/dashboard/device-status", func(c *gin.Context) {})

	executor := NewToolExecutor(router, router, nil)
	result := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-1",
		Input: ToolRequestBody{
			Method: http.MethodGet,
			Path:   "/api/v1/agent/catalog",
			Query:  map[string]any{"q": "device", "limit": "2"},
		},
	}, agentconfig.RuntimePolicy{
		AllowedMethods:     []string{http.MethodGet},
		ToolTimeoutSeconds: 30,
		MaxResponseBytes:   262144,
	})

	require.Equal(t, "ok", result.Status)
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	items, ok := output["items"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	require.Equal(t, 2, output["returned"])
	require.Equal(t, 3, output["total"])
}

func TestToolExecutorCatalogUsesTokenizedSearchAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/devices", func(c *gin.Context) {})
	router.GET("/api/v1/devices/stats", func(c *gin.Context) {})
	router.GET("/api/v1/devices/:id", func(c *gin.Context) {})

	executor := NewToolExecutor(router, router, nil)
	policy := agentconfig.RuntimePolicy{
		AllowedMethods:     []string{http.MethodGet},
		ToolTimeoutSeconds: 30,
		MaxResponseBytes:   262144,
	}
	search := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-search",
		Input: ToolRequestBody{
			Method: http.MethodGet,
			Path:   "/api/v1/agent/catalog",
			Query: map[string]any{
				"q":        "devices stats",
				"category": "devices",
				"limit":    1,
			},
		},
	}, policy)

	require.Equal(t, "ok", search.Status)
	searchOutput, ok := search.Output.(map[string]any)
	require.True(t, ok)
	searchItems, ok := searchOutput["items"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, searchItems, 1)
	require.Equal(t, "/api/v1/devices/stats", searchItems[0]["path"])
	require.Equal(t, "devices", searchOutput["category"])
	require.NotEmpty(t, searchOutput["catalogVersion"])

	firstPage := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-page-1",
		Input: ToolRequestBody{
			Method: http.MethodGet,
			Path:   "/api/v1/agent/catalog",
			Query:  map[string]any{"category": "devices", "limit": 2},
		},
	}, policy)

	require.Equal(t, "ok", firstPage.Status)
	firstOutput, ok := firstPage.Output.(map[string]any)
	require.True(t, ok)
	require.Equal(t, 3, firstOutput["total"])
	require.Equal(t, 2, firstOutput["returned"])
	require.Equal(t, true, firstOutput["hasMore"])
	require.Equal(t, 2, firstOutput["nextOffset"])

	secondPage := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-page-2",
		Input: ToolRequestBody{
			Method: http.MethodGet,
			Path:   "/api/v1/agent/catalog",
			Query:  map[string]any{"category": "devices", "limit": 2, "offset": 2},
		},
	}, policy)

	require.Equal(t, "ok", secondPage.Status)
	secondOutput, ok := secondPage.Output.(map[string]any)
	require.True(t, ok)
	require.Equal(t, 1, secondOutput["returned"])
	require.Equal(t, false, secondOutput["hasMore"])
	require.Equal(t, firstOutput["catalogVersion"], secondOutput["catalogVersion"])
}

func TestToolExecutorCatalogCategoriesCoverAllPolicyVisibleRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/devices", func(c *gin.Context) {})
	router.POST("/api/v1/devices", func(c *gin.Context) {})
	router.GET("/api/v1/alarms/active", func(c *gin.Context) {})
	router.GET("/api/v1/auth/profile", func(c *gin.Context) {})

	executor := NewToolExecutor(router, router, nil)
	result := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-categories",
		Input: ToolRequestBody{
			Method: http.MethodGet,
			Path:   "/api/v1/agent/catalog/categories",
		},
	}, agentconfig.RuntimePolicy{
		AllowedMethods:      []string{http.MethodGet, http.MethodPost},
		BlockedPathPrefixes: []string{"/api/v1/auth/*"},
		ToolTimeoutSeconds:  30,
		MaxResponseBytes:    262144,
	})

	require.Equal(t, "ok", result.Status)
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	items, ok := output["items"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	require.Equal(t, 3, output["totalRoutes"])
	require.Equal(t, 2, output["totalCategories"])
	require.NotEmpty(t, output["catalogVersion"])

	covered := 0
	for _, item := range items {
		covered += item["count"].(int)
	}
	require.Equal(t, output["totalRoutes"], covered)
}

func TestToolExecutorDescribeReturnsAPIHandbookEntry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/api/v1/devices/:id/reboot", func(c *gin.Context) {})
	router.DELETE("/api/v1/devices/:id", func(c *gin.Context) {})

	executor := NewToolExecutor(router, router, nil)
	result := executor.Execute(context.Background(), &admin.Claims{
		UserID:   uuid.New(),
		Username: "operator",
	}, ToolRequest{
		RunID:      "run-1",
		ToolCallID: "tool-1",
		Input: ToolRequestBody{
			Method: http.MethodGet,
			Path:   "/api/v1/agent/catalog/describe",
			Query:  map[string]any{"operationId": "put.devices.by_id.reboot"},
		},
	}, agentconfig.RuntimePolicy{
		AllowedMethods:      []string{http.MethodGet, http.MethodPut, http.MethodDelete},
		BlockedPathPrefixes: []string{},
		ToolTimeoutSeconds:  30,
		MaxResponseBytes:    262144,
	})

	require.Equal(t, "ok", result.Status)
	output, ok := result.Output.(apiRouteDoc)
	require.True(t, ok)
	require.Equal(t, "put.devices.by_id.reboot", output.OperationID)
	require.Equal(t, "PUT", output.Method)
	require.Equal(t, "/api/v1/devices/:id/reboot", output.Path)
	require.Equal(t, "low", output.Risk)
	require.Len(t, output.PathParams, 1)
	require.Equal(t, "id", output.PathParams[0].Name)
	require.Equal(t, "put.devices.by_id.reboot", output.RequestUsage.OperationID)
	require.NotNil(t, output.RequestUsage.Body)

	deleteDoc := apiCatalogDoc(gin.RouteInfo{Method: http.MethodDelete, Path: "/api/v1/devices/:id"})
	require.Equal(t, "high", deleteDoc.Risk)
}
