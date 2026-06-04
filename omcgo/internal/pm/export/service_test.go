package export

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

// ── stubs ───────────────────────────────────────────────────────────────────

type stubRepo struct {
	createID     uuid.UUID
	createErr    error
	created      *CreateRequest
	deleted      []uuid.UUID
	getTask      *Task
	getErr       error
	listResult   []Task
	listFilter   ListFilter
	listErr      error
	markRunErr   error
	markRunCalls []uuid.UUID
}

func (s *stubRepo) Create(_ context.Context, req CreateRequest) (uuid.UUID, error) {
	r := req
	s.created = &r
	return s.createID, s.createErr
}
func (s *stubRepo) Get(_ context.Context, id uuid.UUID) (*Task, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getTask != nil {
		return s.getTask, nil
	}
	return &Task{ID: id, Status: StatusPending}, nil
}
func (s *stubRepo) List(_ context.Context, f ListFilter) ([]Task, error) {
	s.listFilter = f
	return s.listResult, s.listErr
}
func (s *stubRepo) Delete(_ context.Context, id uuid.UUID) error {
	s.deleted = append(s.deleted, id)
	return nil
}
func (s *stubRepo) MarkRunning(_ context.Context, id uuid.UUID) error {
	s.markRunCalls = append(s.markRunCalls, id)
	return s.markRunErr
}
func (s *stubRepo) MarkSucceeded(_ context.Context, _ uuid.UUID, _, _ string, _, _ int64) error {
	return nil
}
func (s *stubRepo) MarkFailed(_ context.Context, _ uuid.UUID, _ string) error { return nil }

type stubEnqueuer struct {
	inserted []asyncjob.InsertRequest
	insertID uuid.UUID
	insertErr error
}

func (s *stubEnqueuer) Insert(_ context.Context, req asyncjob.InsertRequest) (uuid.UUID, error) {
	s.inserted = append(s.inserted, req)
	return s.insertID, s.insertErr
}

// ── tests ───────────────────────────────────────────────────────────────────

func TestService_Create_Success_LandsRowAndEnqueues(t *testing.T) {
	taskID := uuid.New()
	repo := &stubRepo{createID: taskID, getTask: &Task{ID: taskID, Status: StatusPending, SourceType: SourceDashboard}}
	enq := &stubEnqueuer{insertID: uuid.New()}
	svc := NewService(repo, enq)

	task, err := svc.Create(context.Background(), CreateRequest{
		SourceType: SourceDashboard,
		Params:     []byte(`{"metric_type":"kpi"}`),
		CreateUser: "alice",
	})
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, taskID, task.ID)

	// 落表参数透传
	require.NotNil(t, repo.created)
	assert.Equal(t, SourceDashboard, repo.created.SourceType)
	assert.Equal(t, "alice", repo.created.CreateUser)

	// 入队一条 pm_kpi_export job，payload 带 export_task_id
	require.Len(t, enq.inserted, 1)
	assert.Equal(t, JobType, enq.inserted[0].JobType)
	var p JobPayload
	require.NoError(t, json.Unmarshal(enq.inserted[0].Payload, &p))
	assert.Equal(t, taskID.String(), p.ExportTaskID)

	// 成功路径不回滚
	assert.Empty(t, repo.deleted)
}

func TestService_Create_InvalidSourceType(t *testing.T) {
	repo := &stubRepo{}
	enq := &stubEnqueuer{}
	svc := NewService(repo, enq)

	_, err := svc.Create(context.Background(), CreateRequest{SourceType: SourceType("bogus")})
	require.ErrorIs(t, err, ErrInvalidSourceType)
	// 非法来源不落表、不入队
	assert.Nil(t, repo.created)
	assert.Empty(t, enq.inserted)
}

func TestService_Create_NilJobRepo_Rejects(t *testing.T) {
	repo := &stubRepo{}
	svc := NewService(repo, nil)

	_, err := svc.Create(context.Background(), CreateRequest{SourceType: SourceDashboard})
	require.ErrorIs(t, err, ErrJobRepoNotWired)
	assert.Nil(t, repo.created)
}

func TestService_Create_EnqueueFails_RollsBackRow(t *testing.T) {
	taskID := uuid.New()
	repo := &stubRepo{createID: taskID}
	enq := &stubEnqueuer{insertErr: errors.New("queue down")}
	svc := NewService(repo, enq)

	_, err := svc.Create(context.Background(), CreateRequest{SourceType: SourceAdhoc})
	require.Error(t, err)
	// 入队失败后删除已落表的孤儿任务行
	require.Len(t, repo.deleted, 1)
	assert.Equal(t, taskID, repo.deleted[0])
}

func TestService_ListFiles_OnlyReadyFilterApplied(t *testing.T) {
	// 经 handler 设置 OnlyReady=true 后透传到 repo（此处直接验 service.List 透传 filter）。
	repo := &stubRepo{listResult: nil}
	svc := NewService(repo, &stubEnqueuer{})
	_, err := svc.List(context.Background(), ListFilter{OnlyReady: true})
	require.NoError(t, err)
	assert.True(t, repo.listFilter.OnlyReady)
}
