package querytemplate

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/indicator"
)

// stubRepo 是 Repository 的内存实现，专供 handler 单测使用。
type stubRepo struct {
	mu    sync.Mutex
	rows  map[uuid.UUID]*Template
	dupNS map[string]bool // creator_id+name 唯一性模拟
}

func newStubRepo() *stubRepo {
	return &stubRepo{rows: map[uuid.UUID]*Template{}, dupNS: map[string]bool{}}
}

func (s *stubRepo) Create(_ context.Context, req CreateRequest) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := req.CreatorID.String() + "|" + req.Name
	if s.dupNS[key] {
		return uuid.Nil, ErrDuplicate
	}
	id := uuid.New()
	s.rows[id] = &Template{
		ID:          id,
		Name:        req.Name,
		Visibility:  req.Visibility,
		CreatorID:   req.CreatorID,
		Description: req.Description,
		Payload:     req.Payload,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	s.dupNS[key] = true
	return id, nil
}

func (s *stubRepo) Get(_ context.Context, id uuid.UUID) (*Template, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.rows[id]; ok {
		clone := *t
		return &clone, nil
	}
	return nil, ErrNotFound
}

func (s *stubRepo) List(_ context.Context, filter ListFilter) ([]Template, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Template, 0)
	for _, t := range s.rows {
		if !filter.CallerIsSuperAdmin {
			if t.Visibility != VisibilityPublic && t.CreatorID != filter.CallerID {
				continue
			}
		}
		if filter.Visibility != nil && t.Visibility != *filter.Visibility {
			continue
		}
		out = append(out, *t)
	}
	return out, len(out), nil
}

func (s *stubRepo) Update(_ context.Context, id uuid.UUID, req UpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.rows[id]
	if !ok {
		return ErrNotFound
	}
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Payload != nil {
		t.Payload = req.Payload
	}
	if req.Visibility != nil {
		t.Visibility = *req.Visibility
	}
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *stubRepo) Delete(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rows[id]; !ok {
		return ErrNotFound
	}
	delete(s.rows, id)
	return nil
}

// ─────────────────────────────────────────────────────────────
// Test helpers
// ─────────────────────────────────────────────────────────────

// setupRouter 创建带固定 user/super 注入的 router + handler。
// repo 由调用方提供，便于多 case 复用同一仓库。
func setupRouter(repo Repository, callerID uuid.UUID, isSuperAdmin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(repo, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if callerID != uuid.Nil {
			c.Set("user_id", callerID)
		}
		c.Set("is_super_admin", isSuperAdmin)
		c.Next()
	})
	rg := r.Group("/api/v1")
	h.RegisterRoutes(rg)
	return r
}

func setupRouterWithEnabledRepo(repo Repository, callerID uuid.UUID, isSuperAdmin bool, enabledRepo indicator.EnabledIndicatorRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(repo, nil).
		WithEnabledMetricPayloadService(NewEnabledMetricPayloadService(enabledRepo))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if callerID != uuid.Nil {
			c.Set("user_id", callerID)
		}
		c.Set("is_super_admin", isSuperAdmin)
		c.Next()
	})
	rg := r.Group("/api/v1")
	h.RegisterRoutes(rg)
	return r
}

type enabledRepoStub struct {
	enabled map[indicator.DeviceType][]string
}

func (s enabledRepoStub) List(_ context.Context, dt indicator.DeviceType, _ string) ([]string, error) {
	return s.enabled[dt], nil
}

func (s enabledRepoStub) BatchCreate(context.Context, indicator.DeviceType, string, []string, pgx.Tx) error {
	return nil
}

func (s enabledRepoStub) BatchDelete(context.Context, indicator.DeviceType, string, []string, pgx.Tx) error {
	return nil
}

func (s enabledRepoStub) Exists(context.Context, indicator.DeviceType, string, string) (bool, error) {
	return false, nil
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	rr := httptest.NewRecorder()
	httpReq, err := http.NewRequest(method, path, buf)
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rr, httpReq)
	return rr
}

// ─────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────

func TestCreate_PrivateByNormalUser_OK(t *testing.T) {
	alice := uuid.New()
	r := setupRouter(newStubRepo(), alice, false)
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "my-template",
		"visibility": "private",
		"payload":    map[string]any{"device_sns": []string{"SN-1"}},
	})
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCreate_RejectsInvalidRegularReportConfiguration(t *testing.T) {
	alice := uuid.New()
	r := setupRouter(newStubRepo(), alice, false)
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "bad-report",
		"visibility": "private",
		"payload": map[string]any{
			"device_sns":   []string{"SN-1"},
			"metric_paths": []string{"K1"},
			"regular_report": map[string]any{
				"enabled": true, "send_time": "08:30", "periods": []string{"daily"},
				"email_enabled": true, "recipients": []string{},
			},
		},
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "recipients")
}

func TestCreate_RejectsRegularReportWithoutMetrics(t *testing.T) {
	alice := uuid.New()
	r := setupRouter(newStubRepo(), alice, false)
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "report-without-metrics",
		"visibility": "private",
		"payload": map[string]any{
			"device_sns":   []string{"SN-1"},
			"metric_paths": []string{},
			"regular_report": map[string]any{
				"enabled": true, "send_time": "08:30", "periods": []string{"daily"},
				"email_enabled": true, "recipients": []string{"ops@example.com"},
			},
		},
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "metric_path")
}

func TestCreate_RejectsPayloadWithTooManyDevices(t *testing.T) {
	alice := uuid.New()
	r := setupRouter(newStubRepo(), alice, false)
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "too-many-devices",
		"visibility": "private",
		"payload": map[string]any{
			"device_sns":   makeStrings("SN", 51),
			"metric_paths": []string{"K1"},
		},
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreate_RejectsPayloadWithTooManyMetrics(t *testing.T) {
	alice := uuid.New()
	r := setupRouter(newStubRepo(), alice, false)
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "too-many-metrics",
		"visibility": "private",
		"payload": map[string]any{
			"device_sns":   []string{"SN-1"},
			"metric_paths": makeStrings("K", 51),
		},
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreate_RejectsPayloadWithDisabledMetrics(t *testing.T) {
	alice := uuid.New()
	r := setupRouterWithEnabledRepo(newStubRepo(), alice, false, enabledRepoStub{
		enabled: map[indicator.DeviceType][]string{indicator.DeviceTypeENB: {"K_ENABLED"}},
	})
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "disabled-metric",
		"visibility": "private",
		"payload": map[string]any{
			"device_type":  "ENB",
			"metric_paths": []string{"K_ENABLED", "K_DISABLED"},
		},
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "K_DISABLED")
}

func TestCreate_AllowsPayloadAtLimitAndMissingLimitFields(t *testing.T) {
	alice := uuid.New()
	r := setupRouter(newStubRepo(), alice, false)
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "at-limit",
		"visibility": "private",
		"payload": map[string]any{
			"device_sns":   makeStrings("SN", 50),
			"metric_paths": makeStrings("K", 50),
		},
	})
	assert.Equal(t, http.StatusOK, rr.Code)

	rr = doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "missing-limit-fields",
		"visibility": "private",
		"payload":    map[string]any{"granularity": "15min"},
	})
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCreate_PublicByNormalUser_Forbidden(t *testing.T) {
	alice := uuid.New()
	r := setupRouter(newStubRepo(), alice, false)
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "team-template",
		"visibility": "public",
		"payload":    map[string]any{},
	})
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCreate_PublicBySuperAdmin_OK(t *testing.T) {
	r := setupRouter(newStubRepo(), uuid.New(), true)
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", map[string]any{
		"name":       "official-template",
		"visibility": "public",
		"payload":    map[string]any{},
	})
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCreate_DuplicateNameByCreator_Conflict(t *testing.T) {
	alice := uuid.New()
	repo := newStubRepo()
	r := setupRouter(repo, alice, false)
	body := map[string]any{"name": "dup", "visibility": "private", "payload": map[string]any{}}
	rr := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", body)
	require.Equal(t, http.StatusOK, rr.Code)
	rr2 := doJSON(t, r, http.MethodPost, "/api/v1/pm/query-templates", body)
	assert.Equal(t, http.StatusConflict, rr2.Code)
}

func TestList_NormalUser_PublicAndOwnPrivate(t *testing.T) {
	alice := uuid.New()
	bob := uuid.New()
	repo := newStubRepo()

	_, _ = repo.Create(context.Background(), CreateRequest{
		Name: "shared", Visibility: VisibilityPublic, CreatorID: uuid.New(), Payload: []byte("{}"),
	})
	_, _ = repo.Create(context.Background(), CreateRequest{
		Name: "alice-priv", Visibility: VisibilityPrivate, CreatorID: alice, Payload: []byte("{}"),
	})
	_, _ = repo.Create(context.Background(), CreateRequest{
		Name: "bob-priv", Visibility: VisibilityPrivate, CreatorID: bob, Payload: []byte("{}"),
	})

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodGet, "/api/v1/pm/query-templates", nil)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Data struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Data.Total)
}

func TestList_SuperAdmin_SeesAll(t *testing.T) {
	repo := newStubRepo()
	_, _ = repo.Create(context.Background(), CreateRequest{
		Name: "pub", Visibility: VisibilityPublic, CreatorID: uuid.New(), Payload: []byte("{}"),
	})
	_, _ = repo.Create(context.Background(), CreateRequest{
		Name: "priv-a", Visibility: VisibilityPrivate, CreatorID: uuid.New(), Payload: []byte("{}"),
	})
	_, _ = repo.Create(context.Background(), CreateRequest{
		Name: "priv-b", Visibility: VisibilityPrivate, CreatorID: uuid.New(), Payload: []byte("{}"),
	})

	r := setupRouter(repo, uuid.New(), true)
	rr := doJSON(t, r, http.MethodGet, "/api/v1/pm/query-templates", nil)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 3, resp.Data.Total)
}

func TestGet_OtherUserPrivate_Forbidden(t *testing.T) {
	alice := uuid.New()
	bob := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "bob-priv", Visibility: VisibilityPrivate, CreatorID: bob, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodGet, "/api/v1/pm/query-templates/"+id.String(), nil)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestGet_OwnPrivate_OK(t *testing.T) {
	alice := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "alice-priv", Visibility: VisibilityPrivate, CreatorID: alice, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodGet, "/api/v1/pm/query-templates/"+id.String(), nil)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestUpdate_OnlyOwnerOrSuperAdmin(t *testing.T) {
	alice := uuid.New()
	bob := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "bob-priv", Visibility: VisibilityPrivate, CreatorID: bob, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodPatch, "/api/v1/pm/query-templates/"+id.String(), map[string]any{
		"description": "alice tries",
	})
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestUpdate_VisibilityPromoteRequiresSuperAdmin(t *testing.T) {
	alice := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "my-priv", Visibility: VisibilityPrivate, CreatorID: alice, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodPatch, "/api/v1/pm/query-templates/"+id.String(), map[string]any{
		"visibility": "public",
	})
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestUpdate_RejectsPayloadWithTooManyMetrics(t *testing.T) {
	alice := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "my-priv", Visibility: VisibilityPrivate, CreatorID: alice, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodPatch, "/api/v1/pm/query-templates/"+id.String(), map[string]any{
		"payload": map[string]any{
			"device_sns":   []string{"SN-1"},
			"metric_paths": makeStrings("K", 51),
		},
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdate_RejectsPayloadWithDisabledMetrics(t *testing.T) {
	alice := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "my-priv", Visibility: VisibilityPrivate, CreatorID: alice, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouterWithEnabledRepo(repo, alice, false, enabledRepoStub{
		enabled: map[indicator.DeviceType][]string{indicator.DeviceTypeGNB: {"KGNB_ENABLED"}},
	})
	rr := doJSON(t, r, http.MethodPatch, "/api/v1/pm/query-templates/"+id.String(), map[string]any{
		"payload": map[string]any{
			"device_type":  "GNB",
			"metric_paths": []string{"KGNB_ENABLED", "KGNB_DISABLED"},
		},
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "KGNB_DISABLED")
}

func TestUpdate_RejectsPayloadWithTooManyDevices(t *testing.T) {
	alice := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "my-priv", Visibility: VisibilityPrivate, CreatorID: alice, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodPatch, "/api/v1/pm/query-templates/"+id.String(), map[string]any{
		"payload": map[string]any{
			"device_sns":   makeStrings("SN", 51),
			"metric_paths": []string{"K1"},
		},
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDelete_OtherUserPrivate_Forbidden(t *testing.T) {
	alice := uuid.New()
	bob := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "bob-only", Visibility: VisibilityPrivate, CreatorID: bob, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodDelete, "/api/v1/pm/query-templates/"+id.String(), nil)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestDelete_SuperAdmin_AnyTemplate(t *testing.T) {
	bob := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "bob-only", Visibility: VisibilityPrivate, CreatorID: bob, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, uuid.New(), true)
	rr := doJSON(t, r, http.MethodDelete, "/api/v1/pm/query-templates/"+id.String(), nil)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDelete_Owner_OK(t *testing.T) {
	alice := uuid.New()
	repo := newStubRepo()
	id, err := repo.Create(context.Background(), CreateRequest{
		Name: "mine", Visibility: VisibilityPrivate, CreatorID: alice, Payload: []byte("{}"),
	})
	require.NoError(t, err)

	r := setupRouter(repo, alice, false)
	rr := doJSON(t, r, http.MethodDelete, "/api/v1/pm/query-templates/"+id.String(), nil)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestValidVisibility(t *testing.T) {
	assert.True(t, ValidVisibility("public"))
	assert.True(t, ValidVisibility("private"))
	assert.False(t, ValidVisibility(""))
	assert.False(t, ValidVisibility("internal"))
}

func makeStrings(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = prefix + "-" + strconv.Itoa(i+1)
	}
	return out
}
