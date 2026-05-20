// T-0161: trace 任务删除单元测试。
package trace

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// deleteMockRepo 给删除测试用的可控 mock：可注入任务存在性 / 状态 / 错误。
type deleteMockRepo struct {
	mu sync.Mutex

	tasks map[uuid.UUID]*Task

	purgedMessages []uuid.UUID // 记录 PurgeTaskMessages 调用顺序
	deletedTasks   []uuid.UUID // 记录 DeleteTask 调用顺序

	// 错误注入
	purgeErr  error
	deleteErr error
}

func newDeleteMockRepo() *deleteMockRepo {
	return &deleteMockRepo{tasks: map[uuid.UUID]*Task{}}
}

func (m *deleteMockRepo) addTask(t *Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t
	m.tasks[t.ID] = &cp
}

func (m *deleteMockRepo) GetTask(_ context.Context, id uuid.UUID) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		cp := *t
		return &cp, nil
	}
	return nil, commonerrors.ErrNotFound
}

func (m *deleteMockRepo) PurgeTaskMessages(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.purgeErr != nil {
		return m.purgeErr
	}
	m.purgedMessages = append(m.purgedMessages, id)
	return nil
}

func (m *deleteMockRepo) DeleteTask(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.tasks[id]; !ok {
		return commonerrors.ErrNotFound
	}
	delete(m.tasks, id)
	m.deletedTasks = append(m.deletedTasks, id)
	return nil
}

// 其余方法为本测试不关心，返回 errUnused 让误用立即暴露。
func (m *deleteMockRepo) CreateTask(_ context.Context, _ *Task) error { return errUnused }
func (m *deleteMockRepo) GetRunningTaskBySN(_ context.Context, _ string) (*Task, error) {
	return nil, errUnused
}
func (m *deleteMockRepo) ListTasks(_ context.Context, _ TaskFilter) (*model.ListResponse[Task], error) {
	return nil, errUnused
}
func (m *deleteMockRepo) UpdateTaskStatus(_ context.Context, _ uuid.UUID, _ TaskStatus) error {
	return errUnused
}
func (m *deleteMockRepo) IncrementMessageCount(_ context.Context, _ uuid.UUID, _ int) error {
	return errUnused
}
func (m *deleteMockRepo) ListRunningSNs(_ context.Context) (map[string]uuid.UUID, error) {
	return map[string]uuid.UUID{}, nil
}
func (m *deleteMockRepo) ListExpired(_ context.Context, _ int) ([]Task, error) { return nil, errUnused }
func (m *deleteMockRepo) InsertMessage(_ context.Context, _ *Message) error    { return errUnused }
func (m *deleteMockRepo) InsertMessages(_ context.Context, _ []*Message) error { return errUnused }
func (m *deleteMockRepo) ListMessages(_ context.Context, _ MessageFilter) (*model.ListResponse[Message], error) {
	return nil, errUnused
}
func (m *deleteMockRepo) GetMessage(_ context.Context, _, _ uuid.UUID) (*Message, error) {
	return nil, errUnused
}
func (m *deleteMockRepo) BatchDeleteTasks(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, errUnused
}
func (m *deleteMockRepo) CreateExportJob(_ context.Context, _ *ExportJob) error { return errUnused }
func (m *deleteMockRepo) GetExportJob(_ context.Context, _ uuid.UUID) (*ExportJob, error) {
	return nil, errUnused
}
func (m *deleteMockRepo) UpdateExportJob(_ context.Context, _ *ExportJob) error { return errUnused }
func (m *deleteMockRepo) StorageStats(_ context.Context) (int64, int64, error)  { return 0, 0, errUnused }

var _ Repository = (*deleteMockRepo)(nil)

func TestService_DeleteTask_Success(t *testing.T) {
	repo := newDeleteMockRepo()
	id := uuid.New()
	repo.addTask(&Task{
		ID:        id,
		DeviceSN:  "SN-001",
		Status:    TaskStatusStopped,
		StartTime: time.Now().Add(-10 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	})

	svc := NewService(repo, DefaultConfig(), nil)
	got, err := svc.DeleteTask(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "SN-001", got.DeviceSN)
	// PurgeTaskMessages 先于 DeleteTask 调用
	assert.Equal(t, []uuid.UUID{id}, repo.purgedMessages)
	assert.Equal(t, []uuid.UUID{id}, repo.deletedTasks)
}

func TestService_DeleteTask_RunningRejected(t *testing.T) {
	repo := newDeleteMockRepo()
	id := uuid.New()
	repo.addTask(&Task{
		ID:       id,
		DeviceSN: "SN-001",
		Status:   TaskStatusRunning, // running → 必须拒绝
	})

	svc := NewService(repo, DefaultConfig(), nil)
	_, err := svc.DeleteTask(context.Background(), id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput),
		"expected ErrInvalidInput got %v", err)
	// 不应触发 messages 清理
	assert.Empty(t, repo.purgedMessages)
	assert.Empty(t, repo.deletedTasks)
}

func TestService_DeleteTask_NotFound(t *testing.T) {
	repo := newDeleteMockRepo()
	svc := NewService(repo, DefaultConfig(), nil)
	_, err := svc.DeleteTask(context.Background(), uuid.New())
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
}

func TestService_BatchDeleteTasks_MixedSuccessFailure(t *testing.T) {
	repo := newDeleteMockRepo()

	okID := uuid.New()
	runningID := uuid.New()
	missingID := uuid.New()
	repo.addTask(&Task{ID: okID, DeviceSN: "OK", Status: TaskStatusStopped})
	repo.addTask(&Task{ID: runningID, DeviceSN: "RUN", Status: TaskStatusRunning})

	svc := NewService(repo, DefaultConfig(), nil)
	res := svc.BatchDeleteTasks(context.Background(), []uuid.UUID{okID, runningID, missingID})

	assert.Equal(t, 3, res.Total)
	assert.Equal(t, 1, res.Succeeded)
	assert.Equal(t, 2, res.Failed)
	require.Len(t, res.Errors, 2)
	// 仅 ok 被删除
	assert.Equal(t, []uuid.UUID{okID}, repo.deletedTasks)
}

func TestService_BatchDeleteTasks_Empty(t *testing.T) {
	repo := newDeleteMockRepo()
	svc := NewService(repo, DefaultConfig(), nil)
	res := svc.BatchDeleteTasks(context.Background(), nil)
	assert.Equal(t, 0, res.Total)
	assert.Equal(t, 0, res.Succeeded)
	assert.Equal(t, 0, res.Failed)
	assert.Empty(t, res.Errors)
}
