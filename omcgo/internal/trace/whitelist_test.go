package trace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

// mockRepo 用于 whitelist 测试；只实现 ListRunningSNs，其他方法返回 errUnused。
type mockRepo struct {
	snapshots []map[string]uuid.UUID
	calls     int
}

var errUnused = errors.New("mockRepo: method not exercised by this test")

func (m *mockRepo) ListRunningSNs(_ context.Context) (map[string]uuid.UUID, error) {
	if len(m.snapshots) == 0 {
		return map[string]uuid.UUID{}, nil
	}
	idx := m.calls
	if idx >= len(m.snapshots) {
		idx = len(m.snapshots) - 1
	}
	m.calls++
	return m.snapshots[idx], nil
}

func (m *mockRepo) CreateTask(_ context.Context, _ *Task) error                                        { return errUnused }
func (m *mockRepo) GetTask(_ context.Context, _ uuid.UUID) (*Task, error)                              { return nil, errUnused }
func (m *mockRepo) GetRunningTaskBySN(_ context.Context, _ string) (*Task, error)                      { return nil, errUnused }
func (m *mockRepo) ListTasks(_ context.Context, _ TaskFilter) (*model.ListResponse[Task], error)       { return nil, errUnused }
func (m *mockRepo) UpdateTaskStatus(_ context.Context, _ uuid.UUID, _ TaskStatus) error                { return errUnused }
func (m *mockRepo) IncrementMessageCount(_ context.Context, _ uuid.UUID, _ int) error                  { return errUnused }
func (m *mockRepo) ListExpired(_ context.Context, _ int) ([]Task, error)                               { return nil, errUnused }
func (m *mockRepo) PurgeTaskMessages(_ context.Context, _ uuid.UUID) error                             { return errUnused }
func (m *mockRepo) InsertMessage(_ context.Context, _ *Message) error                                  { return errUnused }
func (m *mockRepo) InsertMessages(_ context.Context, _ []*Message) error                               { return errUnused }
func (m *mockRepo) ListMessages(_ context.Context, _ MessageFilter) (*model.ListResponse[Message], error) {
	return nil, errUnused
}
func (m *mockRepo) GetMessage(_ context.Context, _, _ uuid.UUID) (*Message, error) {
	return nil, errUnused
}
func (m *mockRepo) CreateExportJob(_ context.Context, _ *ExportJob) error { return errUnused }
func (m *mockRepo) GetExportJob(_ context.Context, _ uuid.UUID) (*ExportJob, error) {
	return nil, errUnused
}
func (m *mockRepo) UpdateExportJob(_ context.Context, _ *ExportJob) error { return errUnused }
func (m *mockRepo) StorageStats(_ context.Context) (int64, int64, error)  { return 0, 0, errUnused }

// 编译期断言：mockRepo 实现 Repository
var _ Repository = (*mockRepo)(nil)

func TestWhitelistCache_AddRemoveLookup(t *testing.T) {
	repo := &mockRepo{snapshots: []map[string]uuid.UUID{{}}}
	wl := NewWhitelistCache(repo, WhitelistConfig{RefreshInterval: time.Hour}, nil)

	id1 := uuid.New()
	wl.Add("SN-001", id1)
	got, ok := wl.Lookup("SN-001")
	require.True(t, ok)
	assert.Equal(t, id1, got)

	wl.Remove("SN-001")
	_, ok = wl.Lookup("SN-001")
	assert.False(t, ok)
}

func TestWhitelistCache_RefreshReplaces(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	repo := &mockRepo{
		snapshots: []map[string]uuid.UUID{
			{"SN-A": id1, "SN-B": id2}, // 首次加载
			{"SN-A": id1},              // 第二次：SN-B 已不在 running
		},
	}
	wl := NewWhitelistCache(repo, WhitelistConfig{RefreshInterval: time.Hour}, nil)
	require.NoError(t, wl.refresh(context.Background()))
	assert.Equal(t, 2, wl.Size())

	// 第二次 refresh 应剔除 SN-B
	require.NoError(t, wl.refresh(context.Background()))
	assert.Equal(t, 1, wl.Size())
	_, ok := wl.Lookup("SN-A")
	assert.True(t, ok)
	_, ok = wl.Lookup("SN-B")
	assert.False(t, ok)
}

func TestWhitelistCache_LookupEmpty(t *testing.T) {
	wl := NewWhitelistCache(&mockRepo{}, WhitelistConfig{}, nil)
	_, ok := wl.Lookup("")
	assert.False(t, ok)
	_, ok = wl.Lookup("SN-NEVER")
	assert.False(t, ok)
}
