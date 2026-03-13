package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/stretchr/testify/assert"
)

// TestCORSConfigurableOrigins verifies CORS middleware reads from config origins.
func TestCORSConfigurableOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		allowOrigins   []string
		requestOrigin  string
		expectAllowed  bool
	}{
		{
			name:          "allowed origin",
			allowOrigins:  []string{"http://localhost:3000", "http://example.com"},
			requestOrigin: "http://localhost:3000",
			expectAllowed: true,
		},
		{
			name:          "custom production origin",
			allowOrigins:  []string{"https://omc.example.com"},
			requestOrigin: "https://omc.example.com",
			expectAllowed: true,
		},
		{
			name:          "blocked origin",
			allowOrigins:  []string{"http://localhost:3000"},
			requestOrigin: "http://evil.com",
			expectAllowed: false,
		},
		{
			name:          "empty origin header",
			allowOrigins:  []string{"http://localhost:3000"},
			requestOrigin: "",
			expectAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(middleware.CORS(middleware.CORSConfig{
				AllowOrigins: tt.allowOrigins,
			}))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			acao := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectAllowed {
				assert.Equal(t, tt.requestOrigin, acao)
			} else {
				assert.Empty(t, acao)
			}
		})
	}
}

// TestCORSPreflightConfigurable verifies OPTIONS preflight with configurable origins.
func TestCORSPreflightConfigurable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"https://omc.production.com"},
	}))
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "https://omc.production.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://omc.production.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
}

// TestErrorCodeRanges verifies error code constants are in correct domain ranges.
func TestErrorCodeRanges(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		minRange int
		maxRange int
	}{
		// Device Management (1000-1999)
		{"DeviceNotFound", commonerrors.ErrCodeDeviceNotFound, 1000, 1999},
		{"DeviceDuplicate", commonerrors.ErrCodeDeviceDuplicate, 1000, 1999},
		// Data Model (2000-2999)
		{"DataModelNotFound", commonerrors.ErrCodeDataModelNotFound, 2000, 2999},
		{"TemplateNotFound", commonerrors.ErrCodeTemplateNotFound, 2000, 2999},
		// ACS (3000-3999)
		{"ACSSessionTimeout", commonerrors.ErrCodeACSSessionTimeout, 3000, 3999},
		// PM (4000-4999)
		{"PMCounterNotFound", commonerrors.ErrCodePMCounterNotFound, 4000, 4999},
		{"PMThresholdNotFound", commonerrors.ErrCodePMThresholdNotFound, 4000, 4999},
		// Alarm (5000-5999)
		{"AlarmNotFound", commonerrors.ErrCodeAlarmNotFound, 5000, 5999},
		{"AlarmRuleNotFound", commonerrors.ErrCodeAlarmRuleNotFound, 5000, 5999},
		// Provision (6000-6999)
		{"ProvisionTaskNotFound", commonerrors.ErrCodeProvisionTaskNotFound, 6000, 6999},
		// Admin/Auth (7000-7999)
		{"AuthInvalidCredentials", commonerrors.ErrCodeAuthInvalidCredentials, 7000, 7999},
		{"UserNotFound", commonerrors.ErrCodeUserNotFound, 7000, 7999},
		{"RoleNotFound", commonerrors.ErrCodeRoleNotFound, 7000, 7999},
		// Software (8000-8999)
		{"FirmwareNotFound", commonerrors.ErrCodeFirmwareNotFound, 8000, 8999},
		// Northbound (9000-9999)
		{"NBTargetNotFound", commonerrors.ErrCodeNBTargetNotFound, 9000, 9999},
		// Interop (10000-10999)
		{"InteropTestFailed", commonerrors.ErrCodeInteropTestFailed, 10000, 10999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.GreaterOrEqual(t, tt.code, tt.minRange, "code %d should be >= %d", tt.code, tt.minRange)
			assert.LessOrEqual(t, tt.code, tt.maxRange, "code %d should be <= %d", tt.code, tt.maxRange)
		})
	}
}

// TestBusinessErrorSerialization verifies BusinessError JSON output.
func TestBusinessErrorSerialization(t *testing.T) {
	err := commonerrors.NewBusinessError(
		commonerrors.ErrCodeDeviceDuplicate,
		"device already exists",
		commonerrors.ErrAlreadyExists,
	)

	assert.Equal(t, commonerrors.ErrCodeDeviceDuplicate, err.Code)
	assert.Equal(t, "device already exists", err.Message)
	assert.Contains(t, err.Error(), "1002")
	assert.Contains(t, err.Error(), "device already exists")

	// Verify Unwrap
	assert.ErrorIs(t, err, commonerrors.ErrAlreadyExists)
}

// TestHTTPStatusFromError verifies sentinel error → HTTP status mapping.
func TestHTTPStatusFromError(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{commonerrors.ErrNotFound, http.StatusNotFound},
		{commonerrors.ErrAlreadyExists, http.StatusConflict},
		{commonerrors.ErrInvalidInput, http.StatusBadRequest},
		{commonerrors.ErrUnauthorized, http.StatusUnauthorized},
		{commonerrors.ErrForbidden, http.StatusForbidden},
		{commonerrors.ErrTimeout, http.StatusGatewayTimeout},
		{commonerrors.ErrUnavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			assert.Equal(t, tt.expected, commonerrors.HTTPStatusFromError(tt.err))
		})
	}
}
