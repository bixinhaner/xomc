package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// fakeRepo 是 Repository 的轻量内存实现，用于 service 层单测。
// 不涉及真实 PG/squirrel，专注于业务规则验证。
type fakeRepo struct {
	tasks       map[uuid.UUID]*Task
	targets     map[uuid.UUID][]CellTarget
	createError error

	lastProgressDispatch *progressDispatchUpdate
}

type progressDispatchUpdate struct {
	taskID        uuid.UUID
	smallCellCode string
	status        ProgressStatus
	faultCode     *string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		tasks:   make(map[uuid.UUID]*Task),
		targets: make(map[uuid.UUID][]CellTarget),
	}
}

func (f *fakeRepo) CreateTask(_ context.Context, t *Task) error {
	if f.createError != nil {
		return f.createError
	}
	f.tasks[t.TaskID] = t
	return nil
}

func (f *fakeRepo) InsertProgressRows(_ context.Context, id uuid.UUID, ts []CellTarget) error {
	f.targets[id] = append(f.targets[id], ts...)
	return nil
}

func (f *fakeRepo) GetTask(_ context.Context, id uuid.UUID) (*Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	return t, nil
}

func (f *fakeRepo) ListTasks(_ context.Context, filter TaskListFilter) (*model.ListResponse[Task], error) {
	items := make([]Task, 0, len(f.tasks))
	for _, t := range f.tasks {
		items = append(items, *t)
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	size := filter.PageSize
	if size < 1 {
		size = 20
	}
	return model.NewListResponse(items, int64(len(items)), page, size), nil
}

func (f *fakeRepo) UpdateTaskStatus(_ context.Context, id uuid.UUID, status TaskStatus, _ *string) error {
	t, ok := f.tasks[id]
	if !ok {
		return commonerrors.ErrNotFound
	}
	t.TaskStatus = status
	return nil
}

func (f *fakeRepo) DeleteTask(_ context.Context, id uuid.UUID) error {
	if _, ok := f.tasks[id]; !ok {
		return commonerrors.ErrNotFound
	}
	delete(f.tasks, id)
	delete(f.targets, id)
	return nil
}

func (f *fakeRepo) ListProgress(_ context.Context, _ ProgressListFilter) (*model.ListResponse[Progress], error) {
	return model.NewListResponse([]Progress{}, 0, 1, 20), nil
}
func (f *fakeRepo) UpdateProgressDispatch(_ context.Context, taskID uuid.UUID, smallCellCode string, status ProgressStatus, faultCode *string) error {
	f.lastProgressDispatch = &progressDispatchUpdate{
		taskID:        taskID,
		smallCellCode: smallCellCode,
		status:        status,
		faultCode:     faultCode,
	}
	return nil
}
func (f *fakeRepo) TouchHeartbeat(_ context.Context, _ string) (bool, error) { return true, nil }
func (f *fakeRepo) ListActiveTasksByCell(_ context.Context, _ string) ([]Task, error) {
	return nil, nil
}
func (f *fakeRepo) ListDueWaitingTasks(_ context.Context, _ int) ([]Task, error) { return nil, nil }
func (f *fakeRepo) ListDueOnTasks(_ context.Context, _ int) ([]Task, error)      { return nil, nil }
func (f *fakeRepo) IncrementMissedHeartbeat(_ context.Context, _ string, _ int) (bool, int, bool, error) {
	return false, 0, false, nil
}

// ---------- 测试 ----------

func newTestService() (Service, *fakeRepo) {
	repo := newFakeRepo()
	svc := NewService(repo, zap.NewNop())
	return svc, repo
}

func validCreateInput() CreateTaskInput {
	return CreateTaskInput{
		TaskName:        "test-task",
		StartTime:       time.Now().Add(time.Hour),
		Creator:         "alice",
		TargetDeviceSNs: []string{"SN001"},
	}
}

func TestService_Create_FillsDefaultsAndStatus(t *testing.T) {
	svc, repo := newTestService()
	in := validCreateInput()
	task, err := svc.Create(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, "MRS,MRE,MRO", task.MRType)
	assert.Equal(t, "5120", task.StatisPeriod)
	assert.Equal(t, "15", task.ReportPeriod)
	assert.Equal(t, StatusWaiting, task.TaskStatus)
	assert.NotEqual(t, uuid.Nil, task.TaskID)

	// time 落地前已转 UTC
	assert.Equal(t, time.UTC, task.StartTime.Location())

	// 仓储里写入了对应行（简化模型：CreateTask 不写 targets，scheduler 后续动态写）
	assert.Len(t, repo.tasks, 1)
}

func TestService_Create_ValidationErrors(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		mutate  func(*CreateTaskInput)
		wantMsg string
	}{
		{
			name:    "missing task_name",
			mutate:  func(in *CreateTaskInput) { in.TaskName = "" },
			wantMsg: "task_name",
		},
		{
			name:    "mr_type missing MRO",
			mutate:  func(in *CreateTaskInput) { in.MRType = "MRS,MRE" },
			wantMsg: "MRS, MRE and MRO",
		},
		{
			name:    "invalid statis_period",
			mutate:  func(in *CreateTaskInput) { in.StatisPeriod = "999" },
			wantMsg: "statis_period",
		},
		{
			name:    "invalid report_period",
			mutate:  func(in *CreateTaskInput) { in.ReportPeriod = "45" },
			wantMsg: "report_period",
		},
		{
			name: "end_time before start_time",
			mutate: func(in *CreateTaskInput) {
				before := now.Add(-time.Hour)
				in.EndTime = &before
			},
			wantMsg: "end_time must be after start_time",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService()
			in := validCreateInput()
			tt.mutate(&in)
			_, err := svc.Create(context.Background(), in)
			require.Error(t, err)
			assert.True(t, IsValidationError(err), "expected validation error, got %T", err)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestService_Stop_StateTransitions(t *testing.T) {
	tests := []struct {
		name         string
		from         TaskStatus
		wantErr      bool
		wantToState  TaskStatus
		wantConflict bool
	}{
		{name: "waitting → termination", from: StatusWaiting, wantToState: StatusTermination},
		{name: "on → termination", from: StatusOn, wantToState: StatusTermination},
		{name: "off cannot stop", from: StatusOff, wantErr: true, wantConflict: true},
		{name: "termination cannot re-stop", from: StatusTermination, wantErr: true, wantConflict: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			id := uuid.New()
			repo.tasks[id] = &Task{
				TaskID:     id,
				TaskName:   "t",
				TaskStatus: tt.from,
				Creator:    "alice",
				StartTime:  time.Now(),
			}
			err := svc.Stop(context.Background(), id)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantConflict {
					assert.True(t, IsStateError(err), "expected state error, got %T", err)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantToState, repo.tasks[id].TaskStatus)
		})
	}
}

// 多租户已下线（2026-05-25）：测试改为 "Stop 不存在的任务"
func TestService_Stop_NotFoundReturnsNotFound(t *testing.T) {
	svc, _ := newTestService()
	err := svc.Stop(context.Background(), uuid.New())
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
}

func TestService_Delete_OnlyAllowedFromTerminalStates(t *testing.T) {
	tests := []struct {
		name    string
		from    TaskStatus
		wantErr bool
	}{
		{name: "off allowed", from: StatusOff, wantErr: false},
		{name: "termination allowed", from: StatusTermination, wantErr: false},
		{name: "waitting blocked", from: StatusWaiting, wantErr: true},
		{name: "on blocked", from: StatusOn, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			id := uuid.New()
			repo.tasks[id] = &Task{TaskID: id, TaskStatus: tt.from}
			err := svc.Delete(context.Background(), id)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, IsStateError(err))
				assert.Contains(t, repo.tasks, id) // 未删除
				return
			}
			require.NoError(t, err)
			assert.NotContains(t, repo.tasks, id) // 已删除
		})
	}
}

// issue #126 第 5 项：ListProgress 对不存在的 task_id 应返回 ErrNotFound（→ 404），
// 与 Get/Stop/Delete 一致，而非 200 空列表。
func TestService_ListProgress_NotFoundReturnsNotFound(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.ListProgress(context.Background(), ProgressListFilter{TaskID: uuid.New()})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"expected ErrNotFound for missing task, got %T", err)
}

// ListProgress 对存在的任务正常返回（不误判为 NotFound）。
func TestService_ListProgress_ExistingTaskSucceeds(t *testing.T) {
	svc, repo := newTestService()
	id := uuid.New()
	repo.tasks[id] = &Task{TaskID: id, TaskName: "t", TaskStatus: StatusWaiting}
	resp, err := svc.ListProgress(context.Background(), ProgressListFilter{TaskID: id})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestUploadPeriodSeconds(t *testing.T) {
	cases := map[string]struct {
		input string
		want  int
		ok    bool
	}{
		"15min":   {"15", 900, true},
		"30min":   {"30", 1800, true},
		"60min":   {"60", 3600, true},
		"invalid": {"45", 0, false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := UploadPeriodSeconds(c.input)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestHeartbeatTTLSeconds(t *testing.T) {
	// 文档 §7.3：900 → 1500, 1800 → 2700, 3600 → 5400
	assert.Equal(t, 1500, HeartbeatTTLSeconds(900))
	assert.Equal(t, 2700, HeartbeatTTLSeconds(1800))
	assert.Equal(t, 5400, HeartbeatTTLSeconds(3600))
	assert.Equal(t, 0, HeartbeatTTLSeconds(0))
}

func TestHasAllRequiredMRTypes(t *testing.T) {
	cases := map[string]bool{
		"MRS,MRE,MRO":      true,
		"MRO,MRS,MRE":      true,
		"MRS,MRE,MRO,MDT":  true,
		"mrs,mre,mro":      true, // 大小写不敏感
		" MRS , MRE , MRO": true, // 空格容错
		"MRS,MRE":          false,
		"MRO":              false,
		"":                 false,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, hasAllRequiredMRTypes(in))
		})
	}
}
