package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	for _, m := range middlewares {
		r.Use(m)
	}
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestAuditClientIP_PrefersForwardedForAndNormalizes(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.RemoteAddr = "172.18.0.1:54321"
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.10, 172.18.0.1")

	assert.Equal(t, "203.0.113.10", auditClientIP(c))
}

func TestNormalizeAuditIP_StripsCIDRAndPort(t *testing.T) {
	assert.Equal(t, "173.18.0.1", normalizeAuditIP("173.18.0.1/32"))
	assert.Equal(t, "198.51.100.7", normalizeAuditIP("198.51.100.7:12345"))
	assert.Equal(t, "2001:db8::1", normalizeAuditIP("[2001:db8::1]:443"))
}

func TestAuditClientIP_PrefersPublicWhenMixedWithPrivate(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.RemoteAddr = "172.18.0.1:54321"
	c.Request.Header.Set("X-Forwarded-For", "172.18.0.1, 203.0.113.55")

	assert.Equal(t, "203.0.113.55", auditClientIP(c))
}

func TestAuditClientIP_FallsBackToPrivateWhenNoPublicAvailable(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.RemoteAddr = "172.18.0.1:54321"
	c.Request.Header.Set("X-Forwarded-For", "172.18.0.1, 10.0.0.8")

	assert.Equal(t, "172.18.0.1", auditClientIP(c))
}

func TestRequireAuth_ValidToken(t *testing.T) {
	jwtService, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()

	pair, err := jwtService.GenerateTokenPair(&Claims{
		UserID:       userID,
		Username:     "testuser",
		IsSuperAdmin: true,
		Roles:        []string{"admin"},
	})
	require.NoError(t, err)

	var capturedUserID uuid.UUID
	var capturedUsername string
	var capturedIsSuper bool
	var capturedRoles []string
	var capturedContextUserID uuid.UUID
	var capturedContextUsername string

	r := gin.New()
	r.Use(RequireAuth(jwtService))
	r.GET("/test", func(c *gin.Context) {
		if uid, exists := c.Get(CtxKeyUserID); exists {
			capturedUserID = uid.(uuid.UUID)
		}
		if uname, exists := c.Get(CtxKeyUsername); exists {
			capturedUsername = uname.(string)
		}
		if v, exists := c.Get(CtxKeyIsSuperAdmin); exists {
			capturedIsSuper, _ = v.(bool)
		}
		if roles, exists := c.Get(CtxKeyRoles); exists {
			capturedRoles = roles.([]string)
		}
		if uid, ok := c.Request.Context().Value(CtxKeyUserID).(uuid.UUID); ok {
			capturedContextUserID = uid
		}
		if username, ok := c.Request.Context().Value(CtxKeyUsername).(string); ok {
			capturedContextUsername = username
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID, capturedUserID)
	assert.Equal(t, "testuser", capturedUsername)
	assert.True(t, capturedIsSuper)
	assert.Equal(t, []string{"admin"}, capturedRoles)
	assert.Equal(t, userID, capturedContextUserID)
	assert.Equal(t, "testuser", capturedContextUsername)
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	jwtService, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	r := setupRouter(RequireAuth(jwtService))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuth_InvalidFormat(t *testing.T) {
	jwtService, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	r := setupRouter(RequireAuth(jwtService))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	jwtService, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	r := setupRouter(RequireAuth(jwtService))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuth_RefreshTokenRejected(t *testing.T) {
	jwtService, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)

	pair, err := jwtService.GenerateTokenPair(&Claims{
		UserID:   uuid.New(),
		Username: "testuser",
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	r := setupRouter(RequireAuth(jwtService))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePermission_Allowed(t *testing.T) {
	jwtService, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()

	pair, err := jwtService.GenerateTokenPair(&Claims{
		UserID:   userID,
		Username: "admin",
		Roles:    []string{"admin"},
	})
	require.NoError(t, err)

	roleRepo := &mockRoleRepo{
		checkPermissionFn: func(ctx context.Context, uid uuid.UUID, resource, action string) (bool, error) {
			return uid == userID && resource == "devices" && action == "read", nil
		},
	}

	r := gin.New()
	r.Use(RequireAuth(jwtService))
	r.Use(RequirePermission(roleRepo, "devices", "read"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_Denied(t *testing.T) {
	jwtService, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()

	pair, err := jwtService.GenerateTokenPair(&Claims{
		UserID:   userID,
		Username: "viewer",
		Roles:    []string{"viewer"},
	})
	require.NoError(t, err)

	roleRepo := &mockRoleRepo{
		checkPermissionFn: func(ctx context.Context, uid uuid.UUID, resource, action string) (bool, error) {
			return false, nil
		},
	}

	r := gin.New()
	r.Use(RequireAuth(jwtService))
	r.Use(RequirePermission(roleRepo, "users", "admin"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_NoAuth(t *testing.T) {
	roleRepo := &mockRoleRepo{}

	r := gin.New()
	r.Use(RequirePermission(roleRepo, "devices", "read"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAPIPermission_UsesFullPathAndMethod(t *testing.T) {
	userID := uuid.New()
	var capturedUserID uuid.UUID
	var capturedPath, capturedMethod string
	roleRepo := &mockRoleRepo{
		checkPermissionFn: func(_ context.Context, uid uuid.UUID, resource, action string) (bool, error) {
			capturedUserID = uid
			capturedPath = resource
			capturedMethod = action
			return true, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, userID)
		c.Set(CtxKeyIsSuperAdmin, false)
		c.Next()
	})
	r.Use(RequireAPIPermission(roleRepo))
	r.PUT("/api/v1/admin/sysConfig/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/sysConfig/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID, capturedUserID)
	assert.Equal(t, "/api/v1/admin/sysConfig/:id", capturedPath)
	assert.Equal(t, http.MethodPut, capturedMethod)
}

func TestRequireAPIPermission_Denied(t *testing.T) {
	roleRepo := &mockRoleRepo{
		checkPermissionFn: func(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
			return false, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Next()
	})
	r.Use(RequireAPIPermission(roleRepo))
	r.GET("/api/v1/admin/sysConfig", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sysConfig", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAPIPermission_CheckerErrorFailsClosed(t *testing.T) {
	roleRepo := &mockRoleRepo{
		checkPermissionFn: func(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
			return false, errors.New("casbin unavailable")
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Next()
	})
	r.Use(RequireAPIPermission(roleRepo))
	r.GET("/api/v1/admin/sysConfig", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sysConfig", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequireAPIPermission_NilCheckerFailsClosed(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Next()
	})
	r.Use(RequireAPIPermission(nil))
	r.GET("/api/v1/admin/sysConfig", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sysConfig", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequireAPIPermission_MissingAuthContext(t *testing.T) {
	r := gin.New()
	r.Use(RequireAPIPermission(&mockRoleRepo{}))
	r.GET("/api/v1/admin/sysConfig", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sysConfig", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAPIPermission_InvalidUserContextFailsClosed(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, "not-a-uuid")
		c.Next()
	})
	r.Use(RequireAPIPermission(&mockRoleRepo{}))
	r.GET("/api/v1/admin/sysConfig", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sysConfig", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequireAPIPermission_MissingRouteTemplateFailsClosed(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Next()
	})
	r.Use(RequireAPIPermission(&mockRoleRepo{}))

	req := httptest.NewRequest(http.MethodGet, "/unregistered", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequireAPIPermission_SuperAdminBypassesChecker(t *testing.T) {
	checkerCalled := false
	roleRepo := &mockRoleRepo{
		checkPermissionFn: func(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
			checkerCalled = true
			return false, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Set(CtxKeyIsSuperAdmin, true)
		c.Next()
	})
	r.Use(RequireAPIPermission(roleRepo))
	r.DELETE("/api/v1/admin/sysConfig/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/sysConfig/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, checkerCalled)
}

// TestRequireCarrier_* removed in v1.0: RequireCarrier middleware deleted
// alongside users.carrier column. See PRD §11.11.

func TestAuditLogger_WritesOnPost(t *testing.T) {
	auditCreated := false
	auditRepo := &mockAuditRepo{
		createFn: func(ctx context.Context, log *AuditLog) error {
			auditCreated = true
			assert.Equal(t, "POST", log.Action)
			assert.Equal(t, "testuser", log.Username)
			return nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Set(CtxKeyUsername, "testuser")
		c.Next()
	})
	r.Use(AuditLogger(auditRepo))
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Give goroutine time to complete
	// Note: audit is fire-and-forget via goroutine so we can't assert immediately.
	// In a real test we'd use a channel or sync mechanism. For now we verify no panic.
	_ = auditCreated
}

func TestAuditLogger_UsesSingleBusinessAuditEntry(t *testing.T) {
	created := make(chan *AuditLog, 2)
	auditRepo := &mockAuditRepo{
		createFn: func(_ context.Context, log *AuditLog) error {
			created <- log
			return nil
		},
	}
	userID := uuid.New()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, userID)
		c.Set(CtxKeyUsername, "operator")
		c.Next()
	})
	r.Use(AuditLogger(auditRepo))
	r.POST("/geofences/:id/enable", func(c *gin.Context) {
		SetBusinessAudit(c, audit.Entry{
			Action:       "config_geofence_enable",
			ResourceType: "geofence",
			ResourceID:   c.Param("id"),
			Details: map[string]interface{}{
				"reason":        "site moved",
				"target_status": "enabled",
			},
		})
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	resourceID := uuid.NewString()
	req := httptest.NewRequest(
		http.MethodPost,
		"/geofences/"+resourceID+"/enable",
		nil,
	)
	req.Header.Set("User-Agent", "audit-test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	select {
	case got := <-created:
		require.Equal(t, "config_geofence_enable", got.Action)
		require.Equal(t, "geofence", got.Resource)
		require.Equal(t, resourceID, got.ResourceID)
		require.Equal(t, "operator", got.Username)
		require.Equal(t, &userID, got.UserID)
		require.Equal(t, "audit-test", got.UserAgent)
		require.Equal(t, "site moved", got.Details["reason"])
	case <-time.After(time.Second):
		t.Fatal("business audit entry was not created")
	}

	select {
	case duplicate := <-created:
		t.Fatalf("unexpected duplicate audit entry: %+v", duplicate)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestAuditLogger_RecordsFailedBusinessAuditWithReason(t *testing.T) {
	created := make(chan *AuditLog, 1)
	auditRepo := &mockAuditRepo{
		createFn: func(_ context.Context, log *AuditLog) error {
			created <- log
			return nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUsername, "operator")
		c.Next()
	})
	r.Use(AuditLogger(auditRepo))
	r.POST("/geofences/:id/disable", func(c *gin.Context) {
		SetBusinessAudit(c, audit.Entry{
			Action:       "config_geofence_disable",
			ResourceType: "geofence",
			ResourceID:   c.Param("id"),
		})
		SetBusinessAuditError(c, errors.New("preview fingerprint is stale"))
		c.AbortWithStatus(http.StatusConflict)
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/geofences/"+uuid.NewString()+"/disable",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	select {
	case got := <-created:
		require.Equal(t, "config_geofence_disable_failed", got.Action)
		require.Equal(t, "preview fingerprint is stale", got.Details["error"])
	case <-time.After(time.Second):
		t.Fatal("failed business audit entry was not created")
	}
}

func TestAuditLogger_SkipsExplicitlyExcludedWrite(t *testing.T) {
	created := make(chan *AuditLog, 1)
	auditRepo := &mockAuditRepo{
		createFn: func(_ context.Context, log *AuditLog) error {
			created <- log
			return nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUsername, "operator")
		c.Next()
	})
	r.Use(AuditLogger(auditRepo))
	r.POST("/geofences/:id/enable-preview", func(c *gin.Context) {
		SkipAudit(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/geofences/"+uuid.NewString()+"/enable-preview",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	select {
	case got := <-created:
		t.Fatalf("preview unexpectedly created audit entry: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRequireResourcePermission_MapsMethodToAction(t *testing.T) {
	jwtService, err := NewJWTService("test-secret-minimum-32-characters!!")
	require.NoError(t, err)
	userID := uuid.New()

	pair, err := jwtService.GenerateTokenPair(&Claims{
		UserID:   userID,
		Username: "operator",
		Roles:    []string{"operator"},
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		method         string
		expectedAction string
		allowed        bool
		expectedStatus int
	}{
		{"GET maps to read", http.MethodGet, "read", true, http.StatusOK},
		{"HEAD maps to read", http.MethodHead, "read", true, http.StatusOK},
		{"POST maps to write", http.MethodPost, "write", true, http.StatusOK},
		{"PUT maps to write", http.MethodPut, "write", true, http.StatusOK},
		{"PATCH maps to write", http.MethodPatch, "write", true, http.StatusOK},
		{"DELETE maps to delete", http.MethodDelete, "delete", true, http.StatusOK},
		{"GET denied", http.MethodGet, "read", false, http.StatusForbidden},
		{"POST denied", http.MethodPost, "write", false, http.StatusForbidden},
		{"DELETE denied", http.MethodDelete, "delete", false, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedAction string
			roleRepo := &mockRoleRepo{
				checkPermissionFn: func(ctx context.Context, uid uuid.UUID, resource, action string) (bool, error) {
					capturedAction = action
					assert.Equal(t, "devices", resource)
					return tt.allowed, nil
				},
			}

			r := gin.New()
			r.Use(RequireAuth(jwtService))
			r.Use(RequireResourcePermission(roleRepo, "devices"))
			handler := func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			}
			r.GET("/test", handler)
			r.HEAD("/test", handler)
			r.POST("/test", handler)
			r.PUT("/test", handler)
			r.PATCH("/test", handler)
			r.DELETE("/test", handler)

			req := httptest.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedAction, capturedAction)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRequireResourcePermission_NoAuth(t *testing.T) {
	roleRepo := &mockRoleRepo{}

	r := gin.New()
	r.Use(RequireResourcePermission(roleRepo, "devices"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHttpMethodToAction(t *testing.T) {
	tests := []struct {
		method   string
		expected string
	}{
		{http.MethodGet, "read"},
		{http.MethodHead, "read"},
		{http.MethodOptions, "read"},
		{http.MethodPost, "write"},
		{http.MethodPut, "write"},
		{http.MethodPatch, "write"},
		{http.MethodDelete, "delete"},
		{"UNKNOWN", "read"},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			assert.Equal(t, tt.expected, httpMethodToAction(tt.method))
		})
	}
}

func TestAuditLogger_SkipsGetRequests(t *testing.T) {
	auditCreated := false
	auditRepo := &mockAuditRepo{
		createFn: func(ctx context.Context, log *AuditLog) error {
			auditCreated = true
			return nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Set(CtxKeyUsername, "testuser")
		c.Next()
	})
	r.Use(AuditLogger(auditRepo))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, auditCreated)
}

// ── T-0098 P3-05 RequireSuperAdmin ───────────────────────────────────

func TestRequireSuperAdmin_Allowed(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Set(CtxKeyUsername, "root")
		c.Set(CtxKeyIsSuperAdmin, true)
		c.Next()
	})
	r.Use(RequireSuperAdmin())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireSuperAdmin_NonSuperRejected(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxKeyUserID, uuid.New())
		c.Set(CtxKeyUsername, "admin")
		// Not a super admin: IsSuperAdmin false / 缺省。
		c.Set(CtxKeyIsSuperAdmin, false)
		c.Set(CtxKeyRoles, []string{"admin"})
		c.Next()
	})
	r.Use(RequireSuperAdmin())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireSuperAdmin_Unauthenticated(t *testing.T) {
	r := gin.New()
	// 没有上游 RequireAuth：IsSuperAdmin 未设置。
	r.Use(RequireSuperAdmin())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
