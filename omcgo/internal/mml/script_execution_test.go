package mml

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type scriptExecutionTaskRepo struct {
	tasks       []*MMLTask
	createCalls int
	updateCalls int
}

func (r *scriptExecutionTaskRepo) ClaimDueTasks(_ context.Context, _ time.Time, _ int) ([]*MMLTask, error) {
	out := append([]*MMLTask(nil), r.tasks...)
	r.tasks = nil
	return out, nil
}
func (r *scriptExecutionTaskRepo) RecordPeriodicChild(context.Context, *MMLTask, *MMLTask, *time.Time) error {
	return nil
}
func (r *scriptExecutionTaskRepo) FinalizePeriodicParent(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func (r *scriptExecutionTaskRepo) Create(_ context.Context, task *MMLTask) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	r.createCalls++
	r.tasks = append(r.tasks, task)
	return nil
}
func (r *scriptExecutionTaskRepo) GetByID(_ context.Context, id uuid.UUID) (*MMLTask, error) {
	for _, task := range r.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return nil, commonerrors.ErrNotFound
}
func (r *scriptExecutionTaskRepo) GetByRequestID(_ context.Context, creator, requestID string) (*MMLTask, error) {
	for _, task := range r.tasks {
		if task.Creator == creator && task.RequestID == requestID {
			return task, nil
		}
	}
	return nil, commonerrors.ErrNotFound
}
func (r *scriptExecutionTaskRepo) Update(_ context.Context, task *MMLTask) error {
	r.updateCalls++
	return nil
}
func (r *scriptExecutionTaskRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ TaskStatus) error {
	return nil
}
func (r *scriptExecutionTaskRepo) IncrementStats(context.Context, uuid.UUID, int, int) error {
	return nil
}
func (r *scriptExecutionTaskRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (r *scriptExecutionTaskRepo) List(context.Context, TaskFilter) (*model.ListResponse[MMLTask], error) {
	return nil, nil
}
func (r *scriptExecutionTaskRepo) ListByScriptID(context.Context, uuid.UUID, model.ListRequest) (*model.ListResponse[MMLTask], error) {
	return nil, nil
}
func (r *scriptExecutionTaskRepo) UpdateExportAggregate(context.Context, uuid.UUID, string, time.Time) error {
	return nil
}
func (r *scriptExecutionTaskRepo) UpdateExportDevice(context.Context, uuid.UUID, string, string, time.Time) error {
	return nil
}

type scriptExecutionValidator struct {
	result *ScriptValidationResult
}

func (v scriptExecutionValidator) Validate(context.Context, *ParsedScript, ValidationActor) (*ScriptValidationResult, error) {
	return v.result, nil
}

func newScriptExecutionService(script *MMLScript, tasks TaskRepository) *Service {
	return &Service{
		scriptRepo: &mockScriptRepo{getByIDFn: func(context.Context, uuid.UUID) (*MMLScript, error) { return script, nil }},
		taskRepo:   tasks,
		cmdRepo:    &mockCommandRepo{},
		logger:     zap.NewNop(),
	}
}

func scriptExecutionFixture() *MMLScript {
	id := uuid.New()
	return &MMLScript{
		ID: id, Creator: "alice", ScriptName: "巡检", ContentSHA256: "sha-a", ValidationVersion: ValidationVersion,
		PlanItems: []MMLPlanItem{{LineNo: 2, DeviceSN: "SN1", Order: 1, RawLine: "LST DEVICE_INFO;SN1", Command: map[string]interface{}{
			"command_code": "LST DEVICE_INFO", "operation_type": "LST", "rpc_method": "GetParameterValues", "parameters": map[string]interface{}{},
		}}},
		ValidationSummary: JSONMap{"summary": ScriptValidationSummary{TotalLines: 1, ValidLines: 1, DeviceCount: 1}},
	}
}

func TestCreateScriptExecution_UsesStoredPlanNotClientCommands(t *testing.T) {
	script := scriptExecutionFixture()
	tasks := &scriptExecutionTaskRepo{}
	svc := newScriptExecutionService(script, tasks)
	task, validation, err := svc.CreateScriptExecution(context.Background(), script.ID, "alice", ScriptExecutionRequest{TaskName: "巡检", ExecuteType: ExecuteImmediate})
	require.NoError(t, err)
	require.NotNil(t, validation)
	require.Empty(t, validation.Issues)
	require.Equal(t, script.PlanItems, task.PlanItems)
	require.Equal(t, "sha-a", task.ScriptContentSHA256)
	require.Equal(t, ValidationVersion, task.ScriptValidationVersion)
	require.Equal(t, 1, tasks.createCalls)
}

func TestCreateScriptExecution_RepeatedRequestIDReturnsExistingTask(t *testing.T) {
	script := scriptExecutionFixture()
	tasks := &scriptExecutionTaskRepo{}
	svc := newScriptExecutionService(script, tasks)
	req := ScriptExecutionRequest{TaskName: "巡检", ExecuteType: ExecuteImmediate, RequestID: "exec-1"}

	first, _, err := svc.CreateScriptExecution(context.Background(), script.ID, "alice", req)
	require.NoError(t, err)
	second, _, err := svc.CreateScriptExecution(context.Background(), script.ID, "alice", req)
	require.NoError(t, err)

	require.Equal(t, first.ID, second.ID)
	require.Equal(t, 1, tasks.createCalls)
}

func TestCreateScriptExecution_WarningRequiresConfirmation(t *testing.T) {
	script := scriptExecutionFixture()
	tasks := &scriptExecutionTaskRepo{}
	svc := newScriptExecutionService(script, tasks)
	svc.SetScriptExecutionValidator(scriptExecutionValidator{result: &ScriptValidationResult{
		PlanItems: script.PlanItems,
		Issues:    []ScriptIssue{{Code: "MML_DEVICE_OFFLINE", Severity: IssueWarning, LineNo: 2}},
	}})
	_, _, err := svc.CreateScriptExecution(context.Background(), script.ID, "alice", ScriptExecutionRequest{TaskName: "巡检", ExecuteType: ExecuteImmediate})
	require.ErrorIs(t, err, ErrScriptExecutionWarnings)
	require.Equal(t, 0, tasks.createCalls)
}

func TestCreateScriptExecution_DynamicErrorsBlock(t *testing.T) {
	script := scriptExecutionFixture()
	tasks := &scriptExecutionTaskRepo{}
	svc := newScriptExecutionService(script, tasks)
	svc.SetScriptExecutionValidator(scriptExecutionValidator{result: &ScriptValidationResult{
		PlanItems: script.PlanItems,
		Issues:    []ScriptIssue{{Code: "MML_COMMAND_DISABLED", Severity: IssueError, LineNo: 2}},
	}})
	_, result, err := svc.CreateScriptExecution(context.Background(), script.ID, "alice", ScriptExecutionRequest{TaskName: "巡检", ExecuteType: ExecuteImmediate})
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	require.NotNil(t, result)
	require.Equal(t, 0, tasks.createCalls)
}

func TestScheduler_PreflightErrorMarksTaskFailed(t *testing.T) {
	script := scriptExecutionFixture()
	task := &MMLTask{ID: uuid.New(), ScriptID: &script.ID, Creator: "alice", ExecuteType: ExecuteScheduled, Status: TaskRunning, PlanItems: cloneScriptPlanItems(script.PlanItems), Commands: []map[string]interface{}{{"command_code": "LST DEVICE_INFO", "rpc_method": "GetParameterValues"}}, DeviceSNs: []string{"SN1"}}
	tasks := &scriptExecutionTaskRepo{tasks: []*MMLTask{task}}
	svc := newScriptExecutionService(script, tasks)
	svc.SetScriptExecutionValidator(scriptExecutionValidator{result: &ScriptValidationResult{Issues: []ScriptIssue{{Code: "MML_DEVICE_NOT_FOUND", Severity: IssueError, LineNo: 2}}}})
	scheduler := NewScheduler(svc, tasks, &fakeClock{now: time.Now()}, time.Minute, zap.NewNop())
	scheduler.runOnce(context.Background())
	require.Equal(t, TaskFailed, task.Status)
	require.NotNil(t, task.Result)
	require.Equal(t, ResultFailed, *task.Result)
	require.NotEmpty(t, task.Results)
	require.GreaterOrEqual(t, tasks.updateCalls, 1)
}
