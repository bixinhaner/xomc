package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
)

// =============================================================
// Fixtures
// =============================================================

type memDLQ struct {
	mu      sync.Mutex
	entries []*dlq.DeadLetter
}

func (r *memDLQ) Insert(ctx context.Context, e *dlq.DeadLetter) error {
	if e == nil {
		return errors.New("nil entry")
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *e
	r.entries = append(r.entries, &cp)
	return nil
}

func (r *memDLQ) List(ctx context.Context, f dlq.Filter) (*model.ListResponse[dlq.DeadLetter], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]dlq.DeadLetter, 0, len(r.entries))
	for _, e := range r.entries {
		if f.Module != nil && *f.Module != "" && e.SourceModule != *f.Module {
			continue
		}
		if f.Subject != nil && *f.Subject != "" && e.SourceSubject != *f.Subject {
			continue
		}
		out = append(out, *e)
	}
	page := f.Page
	size := f.PageSize
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	return model.NewListResponse(out, int64(len(out)), page, size), nil
}

func (r *memDLQ) Get(ctx context.Context, id uuid.UUID) (*dlq.DeadLetter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if e.ID == id {
			cp := *e
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *memDLQ) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.entries[:0]
	for _, e := range r.entries {
		if e.ID != id {
			out = append(out, e)
		}
	}
	r.entries = out
	return nil
}

func (r *memDLQ) Count(ctx context.Context, mod string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if mod == "" {
		return int64(len(r.entries)), nil
	}
	var n int64
	for _, e := range r.entries {
		if e.SourceModule == mod {
			n++
		}
	}
	return n, nil
}

type stubReplayer struct {
	calls    []*dlq.DeadLetter
	failNext bool
}

func (s *stubReplayer) Replay(ctx context.Context, dl *dlq.DeadLetter) error {
	if s.failNext {
		s.failNext = false
		return errors.New("publish failed")
	}
	s.calls = append(s.calls, dl)
	return nil
}

func newRouter(t *testing.T, h *DeadLetterHandler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	h.RegisterRoutes(g)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	body := map[string]interface{}{}
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &body)
	}
	return w, body
}

// =============================================================
// Tests — V3, V4, V5 from PRD GWT
// =============================================================

// V3: 列表分页 + module 过滤
func TestDeadLetterHandler_List(t *testing.T) {
	repo := &memDLQ{}
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Insert(context.Background(), &dlq.DeadLetter{
			SourceModule: "pm", SourceSubject: "pm.file.received",
		}))
	}
	require.NoError(t, repo.Insert(context.Background(), &dlq.DeadLetter{
		SourceModule: "mr", SourceSubject: "mr.file.received",
	}))

	h := NewDeadLetterHandler(repo, nil)
	r := newRouter(t, h)

	w, _ := doJSON(t, r, http.MethodGet, "/api/v1/admin/dead-letters?module=pm&page=1&page_size=10")
	assert.Equal(t, http.StatusOK, w.Code)

	w2, body2 := doJSON(t, r, http.MethodGet, "/api/v1/admin/dead-letters")
	assert.Equal(t, http.StatusOK, w2.Code)
	data, ok := body2["data"].(map[string]interface{})
	require.True(t, ok)
	assert.EqualValues(t, 4, data["total"])
}

func TestDeadLetterHandler_GetSuccess(t *testing.T) {
	repo := &memDLQ{}
	dl := &dlq.DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received"}
	require.NoError(t, repo.Insert(context.Background(), dl))

	h := NewDeadLetterHandler(repo, nil)
	r := newRouter(t, h)

	w, body := doJSON(t, r, http.MethodGet, "/api/v1/admin/dead-letters/"+dl.ID.String())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, body["data"])
}

func TestDeadLetterHandler_GetNotFound(t *testing.T) {
	repo := &memDLQ{}
	h := NewDeadLetterHandler(repo, nil)
	r := newRouter(t, h)
	w, _ := doJSON(t, r, http.MethodGet, "/api/v1/admin/dead-letters/"+uuid.New().String())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeadLetterHandler_GetInvalidID(t *testing.T) {
	repo := &memDLQ{}
	h := NewDeadLetterHandler(repo, nil)
	r := newRouter(t, h)
	w, _ := doJSON(t, r, http.MethodGet, "/api/v1/admin/dead-letters/not-a-uuid")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// V5: Delete
func TestDeadLetterHandler_Delete(t *testing.T) {
	repo := &memDLQ{}
	dl := &dlq.DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received"}
	require.NoError(t, repo.Insert(context.Background(), dl))

	h := NewDeadLetterHandler(repo, nil)
	r := newRouter(t, h)

	w, _ := doJSON(t, r, http.MethodDelete, "/api/v1/admin/dead-letters/"+dl.ID.String())
	assert.Equal(t, http.StatusOK, w.Code)

	// Subsequent Get → 404
	w2, _ := doJSON(t, r, http.MethodGet, "/api/v1/admin/dead-letters/"+dl.ID.String())
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

// V4: Replay 成功 — record retained
func TestDeadLetterHandler_ReplaySuccess(t *testing.T) {
	repo := &memDLQ{}
	dl := &dlq.DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received", Payload: []byte(`{}`)}
	require.NoError(t, repo.Insert(context.Background(), dl))

	replayer := &stubReplayer{}
	h := NewDeadLetterHandler(repo, nil)
	h.SetReplayer("pm", replayer)
	r := newRouter(t, h)

	w, _ := doJSON(t, r, http.MethodPost, "/api/v1/admin/dead-letters/"+dl.ID.String()+"/replay")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, replayer.calls, 1)

	// Record retained — Get still 200
	w2, _ := doJSON(t, r, http.MethodGet, "/api/v1/admin/dead-letters/"+dl.ID.String())
	assert.Equal(t, http.StatusOK, w2.Code)
}

// V4 negative: replayer 失败 → 500
func TestDeadLetterHandler_ReplayPublisherFails(t *testing.T) {
	repo := &memDLQ{}
	dl := &dlq.DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received"}
	require.NoError(t, repo.Insert(context.Background(), dl))

	replayer := &stubReplayer{failNext: true}
	h := NewDeadLetterHandler(repo, nil)
	h.SetReplayer("pm", replayer)
	r := newRouter(t, h)

	w, _ := doJSON(t, r, http.MethodPost, "/api/v1/admin/dead-letters/"+dl.ID.String()+"/replay")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// 模块未注册 replayer → 503
func TestDeadLetterHandler_ReplayNoReplayerForModule(t *testing.T) {
	repo := &memDLQ{}
	dl := &dlq.DeadLetter{SourceModule: "unknown_mod", SourceSubject: "x.y"}
	require.NoError(t, repo.Insert(context.Background(), dl))

	h := NewDeadLetterHandler(repo, nil)
	// 不注册任何 replayer
	r := newRouter(t, h)

	w, _ := doJSON(t, r, http.MethodPost, "/api/v1/admin/dead-letters/"+dl.ID.String()+"/replay")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeadLetterHandler_ReplayNotFound(t *testing.T) {
	repo := &memDLQ{}
	h := NewDeadLetterHandler(repo, nil)
	h.SetReplayer("pm", &stubReplayer{})
	r := newRouter(t, h)

	w, _ := doJSON(t, r, http.MethodPost, "/api/v1/admin/dead-letters/"+uuid.New().String()+"/replay")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// SetReplayer 防御性输入：nil / 空 module 不 panic
func TestDeadLetterHandler_SetReplayerDefensive(t *testing.T) {
	h := NewDeadLetterHandler(&memDLQ{}, nil)
	require.NotPanics(t, func() {
		h.SetReplayer("", &stubReplayer{})
		h.SetReplayer("pm", nil)
	})
}
