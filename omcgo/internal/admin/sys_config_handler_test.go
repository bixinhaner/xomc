package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSysConfigTestRouter(repo SysConfigRepository) *gin.Engine {
	r := gin.New()
	h := NewSysConfigHandler(NewSysConfigService(repo))
	h.RegisterRoutes(r.Group("/admin"))
	h.RegisterPublicRoutes(r.Group(""))
	return r
}

func decodeResponseData(t *testing.T, recorder *httptest.ResponseRecorder) any {
	t.Helper()
	var envelope struct {
		Data any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	return envelope.Data
}

func TestSysConfigHandler_ListRedactsSecrets(t *testing.T) {
	repo := &stubSysConfigRepo{
		listFn: func(_ context.Context, _ string, _ bool) ([]SysConfig, error) {
			return []SysConfig{{
				ID:       uuid.New(),
				Category: "security",
				Key:      "defaultPasswd",
				Value:    "raw-default-password",
			}}, nil
		},
	}
	r := newSysConfigTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/sysConfig?category=security", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "raw-default-password")
	items := decodeResponseData(t, w).([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, "", item["value"])
	assert.Equal(t, true, item["is_secret"])
	assert.Equal(t, true, item["is_configured"])
}

func TestSysConfigHandler_EmailCategoryRequiresSuperAdmin(t *testing.T) {
	repoCalled := false
	repo := &stubSysConfigRepo{
		listFn: func(_ context.Context, _ string, _ bool) ([]SysConfig, error) {
			repoCalled = true
			return nil, nil
		},
	}
	r := newSysConfigTestRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/admin/sysConfig?category=notification.email", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, repoCalled)

	superRouter := gin.New()
	superRouter.Use(func(c *gin.Context) {
		c.Set(CtxKeyIsSuperAdmin, true)
		c.Next()
	})
	NewSysConfigHandler(NewSysConfigService(repo)).RegisterRoutes(superRouter.Group("/admin"))
	w = httptest.NewRecorder()
	superRouter.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, repoCalled)
}

func TestSysConfigHandler_GetRedactsSecrets(t *testing.T) {
	id := uuid.New()
	repo := &stubSysConfigRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*SysConfig, error) {
			assert.Equal(t, id, gotID)
			return &SysConfig{
				ID:       id,
				Category: "agent",
				Key:      "agent_studio_service_token",
				Value:    "raw-agent-token",
			}, nil
		},
	}
	r := newSysConfigTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/sysConfig/"+id.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "raw-agent-token")
	item := decodeResponseData(t, w).(map[string]any)
	assert.Equal(t, "", item["value"])
	assert.Equal(t, true, item["is_secret"])
	assert.Equal(t, true, item["is_configured"])
}

func TestSysConfigHandler_GetEmailConfigRequiresSuperAdmin(t *testing.T) {
	id := uuid.New()
	repo := &stubSysConfigRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*SysConfig, error) {
			return &SysConfig{
				ID:       id,
				Category: "notification.email",
				Key:      "host",
				Value:    "smtp.example.com",
			}, nil
		},
	}
	r := newSysConfigTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/sysConfig/"+id.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotContains(t, w.Body.String(), "smtp.example.com")
}

func TestSysConfigHandler_ListPublicUsesSafeDTO(t *testing.T) {
	id := uuid.New()
	repo := &stubSysConfigRepo{
		listFn: func(_ context.Context, _ string, _ bool) ([]SysConfig, error) {
			return []SysConfig{{ID: id, Category: "system", Key: "system_name", Value: "OMC"}}, nil
		},
	}
	r := newSysConfigTestRouter(repo)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		list   bool
	}{
		{name: "public list", method: http.MethodGet, path: "/admin/public/configs", list: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			data := decodeResponseData(t, w)
			var item map[string]any
			if tt.list {
				items := data.([]any)
				require.Len(t, items, 1)
				item = items[0].(map[string]any)
			} else {
				item = data.(map[string]any)
			}
			assert.Contains(t, item, "is_secret")
			assert.Contains(t, item, "is_configured")
			assert.Equal(t, true, item["is_public"])
		})
	}
}

func TestSysConfigHandlerRejectsDirectMutations(t *testing.T) {
	r := newSysConfigTestRouter(&stubSysConfigRepo{})
	id := uuid.New().String()
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/admin/sysConfig", `{"category":"system","key":"system_name","value":"OMC"}`},
		{http.MethodPut, "/admin/sysConfig/" + id, `{"value":"new"}`},
		{http.MethodDelete, "/admin/sysConfig/" + id, ""},
	} {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	}
}

func TestSysConfigHandler_CreateRejectsExplicitPublicFlagWith400(t *testing.T) {
	r := newSysConfigTestRouter(&stubSysConfigRepo{})
	req := httptest.NewRequest(http.MethodPost, "/admin/sysConfig",
		bytes.NewBufferString(`{"category":"system","key":"system_name","value":"OMC","is_public":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSysConfigHandler_BatchRejectsAgentTokenWith400(t *testing.T) {
	repoCalled := false
	r := newSysConfigTestRouter(&stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, _ []BatchItem) (int, error) {
			repoCalled = true
			return 1, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/sysConfig/batch",
		bytes.NewBufferString(`{"category":"agent","items":[{"key":"agent_studio_service_token","value":"secret"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, repoCalled)
}

func TestSysConfigHandler_BatchReturnsPersistedApplyStatus(t *testing.T) {
	r := newSysConfigTestRouter(&stubSysConfigRepo{})
	req := httptest.NewRequest(http.MethodPost, "/admin/sysConfig/batch",
		bytes.NewBufferString(`{"category":"device","items":[{"key":"periodicSyncEnabled","value":"true"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	data := decodeResponseData(t, w).(map[string]any)
	batch, ok := data["batch"].(map[string]any)
	require.True(t, ok, "保存响应必须返回可查询的应用批次，而不仅是已写入条数")
	assert.NotEmpty(t, batch["id"])
	assert.Equal(t, "applied", batch["status"])
	assert.Equal(t, "device", batch["category"])
}

func TestSysConfigHandler_BatchMarksPMRetentionAsPendingApplication(t *testing.T) {
	r := newSysConfigTestRouter(&stubSysConfigRepo{})
	req := httptest.NewRequest(http.MethodPost, "/admin/sysConfig/batch",
		bytes.NewBufferString(`{"category":"pm.retention","items":[{"key":"raw_15min_days","value":"30","value_type":"int"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	batch := decodeResponseData(t, w).(map[string]any)["batch"].(map[string]any)
	assert.Equal(t, "pending", batch["status"], "Timescale 策略尚未确认前不得显示已生效")
	targets := batch["targets"].([]any)
	require.Len(t, targets, 1)
	assert.Equal(t, "pm_retention", targets[0].(map[string]any)["target"])
	assert.Equal(t, "pending", targets[0].(map[string]any)["status"])
}

func TestSysConfigHandler_DeleteRejectsSecretWith400(t *testing.T) {
	id := uuid.New()
	deleted := false
	r := newSysConfigTestRouter(&stubSysConfigRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*SysConfig, error) {
			return &SysConfig{ID: id, Category: "security", Key: "defaultPasswd"}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/admin/sysConfig/"+id.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, deleted)
}
