package dlq

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

// memRepository 是一个进程内 Map 实现，仅用于上层（runner / handler）单元测试。
// PG 实现的 SQL 正确性由 e2e_verify.sh 的 GET /admin/dead-letters 200/401 用例兜底。
type memRepository struct {
	mu      sync.Mutex
	entries map[uuid.UUID]*DeadLetter
	// failInsert / failList 注入路径用于错误传播测试
	failInsert error
	failList   error
	failGet    error
	failDelete error
	failCount  error
}

func newMemRepository() *memRepository {
	return &memRepository{entries: make(map[uuid.UUID]*DeadLetter)}
}

func (m *memRepository) Insert(ctx context.Context, e *DeadLetter) error {
	if m.failInsert != nil {
		return m.failInsert
	}
	if e == nil {
		return assertErr("nil entry")
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.LastAttemptAt.IsZero() {
		e.LastAttemptAt = now
	}
	e.Error = TruncateError(e.Error)
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *e
	m.entries[e.ID] = &cp
	return nil
}

func (m *memRepository) List(ctx context.Context, f Filter) (*model.ListResponse[DeadLetter], error) {
	if m.failList != nil {
		return nil, m.failList
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	all := make([]DeadLetter, 0, len(m.entries))
	for _, v := range m.entries {
		if f.Module != nil && *f.Module != "" && v.SourceModule != *f.Module {
			continue
		}
		if f.Subject != nil && *f.Subject != "" && v.SourceSubject != *f.Subject {
			continue
		}
		all = append(all, *v)
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page == 0 {
		f.Page = 1
	}
	return model.NewListResponse(all, int64(len(all)), f.Page, f.PageSize), nil
}

func (m *memRepository) Get(ctx context.Context, id uuid.UUID) (*DeadLetter, error) {
	if m.failGet != nil {
		return nil, m.failGet
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.entries[id]; ok {
		cp := *v
		return &cp, nil
	}
	return nil, nil
}

func (m *memRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.failDelete != nil {
		return m.failDelete
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, id)
	return nil
}

func (m *memRepository) Count(ctx context.Context, mod string) (int64, error) {
	if m.failCount != nil {
		return 0, m.failCount
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if mod == "" {
		return int64(len(m.entries)), nil
	}
	var n int64
	for _, v := range m.entries {
		if v.SourceModule == mod {
			n++
		}
	}
	return n, nil
}

type assertErrType string

func (a assertErrType) Error() string { return string(a) }
func assertErr(s string) error        { return assertErrType(s) }

// =============================================================
// Tests
// =============================================================

func TestTruncateError_BelowLimit(t *testing.T) {
	s := strings.Repeat("a", 100)
	got := TruncateError(s)
	assert.Equal(t, s, got)
}

func TestTruncateError_AboveLimit(t *testing.T) {
	s := strings.Repeat("a", MaxErrorLength+500)
	got := TruncateError(s)
	assert.LessOrEqual(t, len(got), MaxErrorLength)
	assert.Contains(t, got, "[truncated]")
}

func TestMemRepository_InsertAndGet(t *testing.T) {
	repo := newMemRepository()
	ctx := context.Background()

	dl := &DeadLetter{
		SourceModule:  "pm",
		SourceSubject: "pm.file.received",
		Payload:       []byte(`{"foo":"bar"}`),
		Error:         "boom",
		RetryCount:    3,
	}
	require.NoError(t, repo.Insert(ctx, dl))
	require.NotEqual(t, uuid.Nil, dl.ID, "Insert should auto-assign ID")
	require.False(t, dl.CreatedAt.IsZero())
	require.False(t, dl.LastAttemptAt.IsZero())

	got, err := repo.Get(ctx, dl.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "pm", got.SourceModule)
	assert.Equal(t, 3, got.RetryCount)
}

func TestMemRepository_GetNotFound(t *testing.T) {
	repo := newMemRepository()
	got, err := repo.Get(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, got, "missing record should be nil, not error")
}

func TestMemRepository_ListFilterByModule(t *testing.T) {
	repo := newMemRepository()
	ctx := context.Background()
	for _, m := range []string{"pm", "pm", "mr"} {
		require.NoError(t, repo.Insert(ctx, &DeadLetter{SourceModule: m, SourceSubject: m + ".file.received"}))
	}
	mod := "pm"
	got, err := repo.List(ctx, Filter{Module: &mod, ListRequest: model.ListRequest{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	assert.Equal(t, int64(2), got.Total)
}

func TestMemRepository_DeleteThenCount(t *testing.T) {
	repo := newMemRepository()
	ctx := context.Background()
	dl := &DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received"}
	require.NoError(t, repo.Insert(ctx, dl))

	c1, err := repo.Count(ctx, "pm")
	require.NoError(t, err)
	assert.Equal(t, int64(1), c1)

	require.NoError(t, repo.Delete(ctx, dl.ID))
	c2, err := repo.Count(ctx, "pm")
	require.NoError(t, err)
	assert.Equal(t, int64(0), c2)
}

func TestMemRepository_DeleteNotFoundIsNoop(t *testing.T) {
	repo := newMemRepository()
	require.NoError(t, repo.Delete(context.Background(), uuid.New()))
}

func TestMemRepository_ErrorTruncated(t *testing.T) {
	repo := newMemRepository()
	long := strings.Repeat("x", MaxErrorLength*2)
	dl := &DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received", Error: long}
	require.NoError(t, repo.Insert(context.Background(), dl))
	got, err := repo.Get(context.Background(), dl.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.LessOrEqual(t, len(got.Error), MaxErrorLength)
	assert.Contains(t, got.Error, "[truncated]")
}

func TestMemRepository_CountByModuleSeparate(t *testing.T) {
	repo := newMemRepository()
	ctx := context.Background()
	require.NoError(t, repo.Insert(ctx, &DeadLetter{SourceModule: "pm", SourceSubject: "pm.file.received"}))
	require.NoError(t, repo.Insert(ctx, &DeadLetter{SourceModule: "mr", SourceSubject: "mr.file.received"}))
	require.NoError(t, repo.Insert(ctx, &DeadLetter{SourceModule: "mr", SourceSubject: "mr.file.received"}))

	pmN, err := repo.Count(ctx, "pm")
	require.NoError(t, err)
	mrN, err := repo.Count(ctx, "mr")
	require.NoError(t, err)
	allN, err := repo.Count(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), pmN)
	assert.Equal(t, int64(2), mrN)
	assert.Equal(t, int64(3), allN)
}

func TestMemRepository_InsertNilReturnsError(t *testing.T) {
	repo := newMemRepository()
	err := repo.Insert(context.Background(), nil)
	assert.Error(t, err)
}
