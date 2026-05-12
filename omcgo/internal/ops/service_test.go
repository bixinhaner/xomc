package ops

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ---------------------------------------------------------------------------
// Mock repositories (function-field pattern)
// ---------------------------------------------------------------------------

type mockTemplateRepo struct {
	createFn            func(ctx context.Context, t *OpsTemplate) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*OpsTemplate, error)
	updateFn            func(ctx context.Context, t *OpsTemplate) error
	deleteFn            func(ctx context.Context, id uuid.UUID) error
	listFn              func(ctx context.Context, filter TemplateFilter) (*model.ListResponse[OpsTemplate], error)
	incrementUseCountFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockTemplateRepo) Create(ctx context.Context, t *OpsTemplate) error {
	if m.createFn != nil {
		return m.createFn(ctx, t)
	}
	return nil
}

func (m *mockTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*OpsTemplate, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTemplateRepo) Update(ctx context.Context, t *OpsTemplate) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, t)
	}
	return nil
}

func (m *mockTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTemplateRepo) List(ctx context.Context, filter TemplateFilter) (*model.ListResponse[OpsTemplate], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockTemplateRepo) IncrementUseCount(ctx context.Context, id uuid.UUID) error {
	if m.incrementUseCountFn != nil {
		return m.incrementUseCountFn(ctx, id)
	}
	return nil
}

type mockTaskRepo struct {
	createFn         func(ctx context.Context, task *OpsTask) error
	getByIDFn        func(ctx context.Context, id uuid.UUID) (*OpsTask, error)
	updateStatusFn   func(ctx context.Context, task *OpsTask) error
	listFn           func(ctx context.Context, filter TaskFilter) (*model.ListResponse[OpsTask], error)
	updateApprovalFn func(ctx context.Context, taskID, approverID uuid.UUID, approve bool, decidedAt time.Time) error
}

func (m *mockTaskRepo) Create(ctx context.Context, task *OpsTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*OpsTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskRepo) UpdateStatus(ctx context.Context, task *OpsTask) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[OpsTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockTaskRepo) UpdateApproval(ctx context.Context, taskID, approverID uuid.UUID, approve bool, decidedAt time.Time) error {
	if m.updateApprovalFn != nil {
		return m.updateApprovalFn(ctx, taskID, approverID, approve, decidedAt)
	}
	return nil
}

type mockCmdRepo struct {
	createFn func(ctx context.Context, record *OpsCommandRecord) error
	listFn   func(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error)
}

func (m *mockCmdRepo) Create(ctx context.Context, record *OpsCommandRecord) error {
	if m.createFn != nil {
		return m.createFn(ctx, record)
	}
	return nil
}

func (m *mockCmdRepo) List(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestService(tmplRepo *mockTemplateRepo, taskRepo *mockTaskRepo, cmdRepo *mockCmdRepo) *Service {
	return NewService(tmplRepo, taskRepo, cmdRepo, zap.NewNop())
}

// ---------------------------------------------------------------------------
// Template Tests
// ---------------------------------------------------------------------------

func TestService_CreateTemplate(t *testing.T) {
	var saved *OpsTemplate
	tmplRepo := &mockTemplateRepo{
		createFn: func(_ context.Context, tmpl *OpsTemplate) error {
			saved = tmpl
			return nil
		},
	}

	svc := newTestService(tmplRepo, &mockTaskRepo{}, &mockCmdRepo{})

	tmpl := &OpsTemplate{
		ID:           uuid.New(),
		TemplateName: "Firmware Upgrade",
		Description:  "Standard firmware upgrade procedure",
		Category:     "upgrade",
		Creator:      "admin",
	}

	result, err := svc.CreateTemplate(context.Background(), tmpl)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Defaults applied
	assert.Equal(t, 0, saved.UseCount, "UseCount should be forced to 0")
	assert.Equal(t, json.RawMessage("[]"), saved.TargetDeviceTypes, "nil TargetDeviceTypes should default to []")
	assert.Equal(t, json.RawMessage("[]"), saved.Steps, "nil Steps should default to []")
	assert.Equal(t, json.RawMessage("[]"), saved.Tags, "nil Tags should default to []")
	assert.Equal(t, "Firmware Upgrade", saved.TemplateName)
}

func TestService_GetTemplate(t *testing.T) {
	id := uuid.New()
	expected := &OpsTemplate{ID: id, TemplateName: "Test Template"}

	tmplRepo := &mockTemplateRepo{
		getByIDFn: func(_ context.Context, qID uuid.UUID) (*OpsTemplate, error) {
			assert.Equal(t, id, qID)
			return expected, nil
		},
	}

	svc := newTestService(tmplRepo, &mockTaskRepo{}, &mockCmdRepo{})
	result, err := svc.GetTemplate(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestService_UpdateTemplate(t *testing.T) {
	id := uuid.New()
	existing := &OpsTemplate{
		ID:                id,
		TemplateName:      "Old Name",
		Description:       "Old desc",
		Category:          "old",
		TargetDeviceTypes: json.RawMessage(`["lte"]`),
		Steps:             json.RawMessage(`[{"cmd":"reboot"}]`),
		EstimatedDuration: 30,
		Creator:           "old-admin",
		Tags:              json.RawMessage(`["old-tag"]`),
		UseCount:          5,
	}

	var updated *OpsTemplate
	tmplRepo := &mockTemplateRepo{
		getByIDFn: func(_ context.Context, qID uuid.UUID) (*OpsTemplate, error) {
			assert.Equal(t, id, qID)
			return existing, nil
		},
		updateFn: func(_ context.Context, tmpl *OpsTemplate) error {
			updated = tmpl
			return nil
		},
	}

	svc := newTestService(tmplRepo, &mockTaskRepo{}, &mockCmdRepo{})

	patch := &OpsTemplate{
		TemplateName:      "New Name",
		Description:       "New desc",
		Category:          "maintenance",
		TargetDeviceTypes: json.RawMessage(`["nr"]`),
		Steps:             json.RawMessage(`[{"cmd":"upgrade"}]`),
		EstimatedDuration: 60,
		Creator:           "new-admin",
		Tags:              json.RawMessage(`["new-tag"]`),
	}

	result, err := svc.UpdateTemplate(context.Background(), id, patch)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Name", updated.TemplateName)
	assert.Equal(t, "New desc", updated.Description)
	assert.Equal(t, "maintenance", updated.Category)
	assert.Equal(t, json.RawMessage(`["nr"]`), updated.TargetDeviceTypes)
	assert.Equal(t, json.RawMessage(`[{"cmd":"upgrade"}]`), updated.Steps)
	assert.Equal(t, 60, updated.EstimatedDuration)
	assert.Equal(t, "new-admin", updated.Creator)
	assert.Equal(t, json.RawMessage(`["new-tag"]`), updated.Tags)
	// UseCount should be preserved from existing
	assert.Equal(t, 5, updated.UseCount)
}

func TestService_DeleteTemplate(t *testing.T) {
	id := uuid.New()
	var deletedID uuid.UUID

	tmplRepo := &mockTemplateRepo{
		deleteFn: func(_ context.Context, qID uuid.UUID) error {
			deletedID = qID
			return nil
		},
	}

	svc := newTestService(tmplRepo, &mockTaskRepo{}, &mockCmdRepo{})
	err := svc.DeleteTemplate(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, id, deletedID, "should delegate correct ID to repo")
}

// ---------------------------------------------------------------------------
// Task Tests
// ---------------------------------------------------------------------------

func TestService_CreateTask(t *testing.T) {
	var saved *OpsTask
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, task *OpsTask) error {
			saved = task
			return nil
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})

	deviceSNs := json.RawMessage(`["SN001","SN002","SN003"]`)
	task := &OpsTask{
		ID:        uuid.New(),
		TaskName:  "Batch Reboot",
		DeviceSNs: deviceSNs,
		Creator:   "admin",
	}

	result, err := svc.CreateTask(context.Background(), task)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, OpsTaskPending, saved.Status, "status should be set to pending")
	assert.Equal(t, 0, saved.Progress, "progress should start at 0")
	assert.Equal(t, 0, saved.CurrentStep, "current step should start at 0")
	assert.Equal(t, 0, saved.SuccessCount, "success count should start at 0")
	assert.Equal(t, 0, saved.FailCount, "fail count should start at 0")
	assert.Equal(t, 3, saved.TotalCount, "total count should equal number of device SNs")
}

func TestService_CreateTask_WithTemplate(t *testing.T) {
	tmplID := uuid.New()
	var incrementedID uuid.UUID

	tmplRepo := &mockTemplateRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTemplate, error) {
			return &OpsTemplate{
				ID:           tmplID,
				TemplateName: "Upgrade Template",
				Steps:        json.RawMessage(`[{"cmd":"download"},{"cmd":"install"},{"cmd":"reboot"}]`),
			}, nil
		},
		incrementUseCountFn: func(_ context.Context, id uuid.UUID) error {
			incrementedID = id
			return nil
		},
	}

	var saved *OpsTask
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, task *OpsTask) error {
			saved = task
			return nil
		},
	}

	svc := newTestService(tmplRepo, taskRepo, &mockCmdRepo{})

	task := &OpsTask{
		ID:         uuid.New(),
		TaskName:   "Template Task",
		TemplateID: &tmplID,
		DeviceSNs:  json.RawMessage(`["SN001"]`),
		Creator:    "admin",
	}

	result, err := svc.CreateTask(context.Background(), task)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, saved.TotalSteps, "total steps should be resolved from template")
	assert.Equal(t, 1, saved.TotalCount, "total count should match device list")
	assert.Equal(t, tmplID, incrementedID, "should increment use count for the correct template")
}

func TestService_CancelTask_FromPending(t *testing.T) {
	taskID := uuid.New()
	var statusUpdated *OpsTask

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Status: OpsTaskPending}, nil
		},
		updateStatusFn: func(_ context.Context, task *OpsTask) error {
			statusUpdated = task
			return nil
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.CancelTask(context.Background(), taskID)

	require.NoError(t, err)
	require.NotNil(t, statusUpdated)
	assert.Equal(t, OpsTaskCancelled, statusUpdated.Status)
	assert.NotNil(t, statusUpdated.CompletedAt, "CompletedAt should be set")
}

func TestService_CancelTask_FromCompleted(t *testing.T) {
	taskID := uuid.New()

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Status: OpsTaskSuccess}, nil
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.CancelTask(context.Background(), taskID)

	require.Error(t, err)
	var bErr *commonerrors.BusinessError
	require.ErrorAs(t, err, &bErr)
	assert.Equal(t, 8100, bErr.Code)
}

func TestService_PauseTask_FromRunning(t *testing.T) {
	taskID := uuid.New()
	var statusUpdated *OpsTask

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Status: OpsTaskRunning}, nil
		},
		updateStatusFn: func(_ context.Context, task *OpsTask) error {
			statusUpdated = task
			return nil
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.PauseTask(context.Background(), taskID)

	require.NoError(t, err)
	require.NotNil(t, statusUpdated)
	assert.Equal(t, OpsTaskPaused, statusUpdated.Status)
}

func TestService_PauseTask_FromPending(t *testing.T) {
	taskID := uuid.New()

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Status: OpsTaskPending}, nil
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.PauseTask(context.Background(), taskID)

	require.Error(t, err)
	var bErr *commonerrors.BusinessError
	require.ErrorAs(t, err, &bErr)
	assert.Equal(t, 8101, bErr.Code)
}

func TestService_ResumeTask_FromPaused(t *testing.T) {
	taskID := uuid.New()
	var statusUpdated *OpsTask

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Status: OpsTaskPaused, StartedAt: nil}, nil
		},
		updateStatusFn: func(_ context.Context, task *OpsTask) error {
			statusUpdated = task
			return nil
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.ResumeTask(context.Background(), taskID)

	require.NoError(t, err)
	require.NotNil(t, statusUpdated)
	assert.Equal(t, OpsTaskRunning, statusUpdated.Status)
	assert.NotNil(t, statusUpdated.StartedAt, "StartedAt should be set when nil")
}

func TestService_ResumeTask_FromRunning(t *testing.T) {
	taskID := uuid.New()

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Status: OpsTaskRunning}, nil
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.ResumeTask(context.Background(), taskID)

	require.Error(t, err)
	var bErr *commonerrors.BusinessError
	require.ErrorAs(t, err, &bErr)
	assert.Equal(t, 8102, bErr.Code)
}

// ---------------------------------------------------------------------------
// Command Record Tests
// ---------------------------------------------------------------------------

func TestService_CreateCommandRecord(t *testing.T) {
	var saved *OpsCommandRecord
	cmdRepo := &mockCmdRepo{
		createFn: func(_ context.Context, record *OpsCommandRecord) error {
			saved = record
			return nil
		},
	}

	svc := newTestService(&mockTemplateRepo{}, &mockTaskRepo{}, cmdRepo)

	record := &OpsCommandRecord{
		ID:          uuid.New(),
		CommandText: "get InternetGatewayDevice.",
		DeviceSN:    "SN001",
		DeviceName:  "eNB-001",
		Operator:    "admin",
		Duration:    120,
		Success:     true,
		Output:      "OK",
	}

	result, err := svc.CreateCommandRecord(context.Background(), record)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, saved.ExecuteTime.IsZero(), "ExecuteTime should default to now when zero")
	assert.WithinDuration(t, time.Now(), saved.ExecuteTime, 2*time.Second)
}

// ---------------------------------------------------------------------------
// CreateTask error-path tests (T-0057.6: HTTP status mapping for ops/tasks)
// ---------------------------------------------------------------------------

// TestService_CreateTask_TemplateNotFound — missing template_id must surface as
// ErrNotFound (404) instead of being silently swallowed (which previously let
// the FK violation reach the DB and produce a 500).
func TestService_CreateTask_TemplateNotFound(t *testing.T) {
	missingTemplateID := uuid.New()

	tmplRepo := &mockTemplateRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTemplate, error) {
			assert.Equal(t, missingTemplateID, id)
			return nil, commonerrors.ErrNotFound
		},
	}

	taskCreated := false
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, _ *OpsTask) error {
			taskCreated = true
			return nil
		},
	}

	svc := newTestService(tmplRepo, taskRepo, &mockCmdRepo{})

	task := &OpsTask{
		TaskName:   "Task with missing template",
		TemplateID: &missingTemplateID,
		DeviceSNs:  json.RawMessage(`["SN001"]`),
		Creator:    "admin",
	}

	result, err := svc.CreateTask(context.Background(), task)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"missing template must propagate ErrNotFound (got: %v)", err)
	assert.False(t, taskCreated,
		"task must NOT be persisted when template is missing")

	// HTTP layer must map this to 404, not 500.
	assert.Equal(t, http.StatusNotFound, commonerrors.HTTPStatusFromError(err),
		"HTTPStatusFromError must return 404 for missing template")
}

// TestService_CreateTask_RepoFKViolation — when template_id passes the up-front
// check but a FK violation slips through (e.g. concurrent template deletion),
// the repo's mapPgError must translate it to ErrInvalidInput so the handler
// returns 400 instead of 500.
func TestService_CreateTask_RepoFKViolation(t *testing.T) {
	tmplID := uuid.New()

	tmplRepo := &mockTemplateRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTemplate, error) {
			return &OpsTemplate{ID: tmplID, Steps: json.RawMessage(`[]`)}, nil
		},
	}

	// Simulate the repo returning the same sentinel that mapPgError would
	// produce for a FK violation (PostgreSQL SQLSTATE 23503).
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, _ *OpsTask) error {
			return commonerrors.ErrInvalidInput
		},
	}

	svc := newTestService(tmplRepo, taskRepo, &mockCmdRepo{})

	task := &OpsTask{
		TaskName:   "Task hitting FK violation",
		TemplateID: &tmplID,
		DeviceSNs:  json.RawMessage(`["SN001"]`),
		Creator:    "admin",
	}

	_, err := svc.CreateTask(context.Background(), task)

	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput),
		"FK violation must surface as ErrInvalidInput (got: %v)", err)

	// HTTP layer must map this to 400, not 500.
	assert.Equal(t, http.StatusBadRequest, commonerrors.HTTPStatusFromError(err),
		"HTTPStatusFromError must return 400 for FK violation")
}

// TestService_CreateTask_TemplateLookupGenericError — non-NotFound errors from
// template lookup must propagate (not be swallowed) so operators can observe
// real infrastructure issues.
func TestService_CreateTask_TemplateLookupGenericError(t *testing.T) {
	tmplID := uuid.New()
	infraErr := errors.New("connection refused")

	tmplRepo := &mockTemplateRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTemplate, error) {
			return nil, infraErr
		},
	}

	svc := newTestService(tmplRepo, &mockTaskRepo{}, &mockCmdRepo{})

	task := &OpsTask{
		TaskName:   "Task with infra failure",
		TemplateID: &tmplID,
		DeviceSNs:  json.RawMessage(`["SN001"]`),
		Creator:    "admin",
	}

	_, err := svc.CreateTask(context.Background(), task)

	require.Error(t, err)
	assert.True(t, errors.Is(err, infraErr),
		"underlying infra error must be wrapped, not swallowed")
	// Generic error (no sentinel) → HTTP 500, which is correct here.
	assert.Equal(t, http.StatusInternalServerError, commonerrors.HTTPStatusFromError(err))
}

// ---------------------------------------------------------------------------
// CancelTask / PauseTask / ResumeTask error-path tests
// (Verify ErrNotFound propagates through %w wrapping for handler 404 mapping)
// ---------------------------------------------------------------------------

func TestService_CancelTask_NotFound(t *testing.T) {
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return nil, commonerrors.ErrNotFound
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.CancelTask(context.Background(), uuid.New())

	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
	assert.Equal(t, http.StatusNotFound, commonerrors.HTTPStatusFromError(err))
}

func TestService_PauseTask_NotFound(t *testing.T) {
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return nil, commonerrors.ErrNotFound
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.PauseTask(context.Background(), uuid.New())

	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
	assert.Equal(t, http.StatusNotFound, commonerrors.HTTPStatusFromError(err))
}

func TestService_ResumeTask_NotFound(t *testing.T) {
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return nil, commonerrors.ErrNotFound
		},
	}

	svc := newTestService(&mockTemplateRepo{}, taskRepo, &mockCmdRepo{})
	err := svc.ResumeTask(context.Background(), uuid.New())

	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
	assert.Equal(t, http.StatusNotFound, commonerrors.HTTPStatusFromError(err))
}

// ---------------------------------------------------------------------------
// mapPgError unit tests (translate PostgreSQL SQLSTATE codes → sentinel errors)
// ---------------------------------------------------------------------------

func TestMapPgError_ForeignKeyViolation(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:           pgCodeForeignKeyViolation,
		ConstraintName: "ops_tasks_template_id_fkey",
		Message:        "insert or update on table \"ops_tasks\" violates foreign key constraint",
	}
	mapped := mapPgError(pgErr, "ops_task")

	require.Error(t, mapped)
	assert.True(t, errors.Is(mapped, commonerrors.ErrInvalidInput),
		"FK violation must map to ErrInvalidInput")
	assert.Equal(t, http.StatusBadRequest, commonerrors.HTTPStatusFromError(mapped))
}

func TestMapPgError_UniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:           pgCodeUniqueViolation,
		ConstraintName: "ops_templates_name_key",
	}
	mapped := mapPgError(pgErr, "ops_template")

	require.Error(t, mapped)
	assert.True(t, errors.Is(mapped, commonerrors.ErrAlreadyExists))
	assert.Equal(t, http.StatusConflict, commonerrors.HTTPStatusFromError(mapped))
}

func TestMapPgError_NotNullViolation(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:       pgCodeNotNullViolation,
		ColumnName: "task_name",
	}
	mapped := mapPgError(pgErr, "ops_task")

	require.Error(t, mapped)
	assert.True(t, errors.Is(mapped, commonerrors.ErrInvalidInput))
	assert.Equal(t, http.StatusBadRequest, commonerrors.HTTPStatusFromError(mapped))
}

func TestMapPgError_NonPgError_PassesThrough(t *testing.T) {
	plain := errors.New("network timeout")
	mapped := mapPgError(plain, "ops_task")

	// Non-pg errors must be returned unchanged so callers can apply their
	// own handling.
	assert.Same(t, plain, mapped, "non-PgError must pass through unchanged")
}

func TestMapPgError_Nil(t *testing.T) {
	assert.Nil(t, mapPgError(nil, "ops_task"))
}

// ---------------------------------------------------------------------------
// T-0101-d: ApprovalService 状态机集成测试
// ---------------------------------------------------------------------------
//
// 测试覆盖（5 用例）：
//   1. 4-eye self-approval forbidden（既有逻辑，反退化）
//   2. 已审批的任务不可再审（ApprovalState 非 pending）
//   3. approve → UpdateApproval 调用 + approve=true
//   4. reject → UpdateApproval 调用 + approve=false
//   5. UpdateApproval 报错时 Approve 错误传播

// mockAuditLogRepo 是 AuditLogRepository 的 noop mock，仅满足 ApprovalService 依赖。
type mockAuditLogRepo struct{}

func (m *mockAuditLogRepo) Create(_ context.Context, _ *OpsAuditLog) error { return nil }
func (m *mockAuditLogRepo) List(_ context.Context, _ AuditLogFilter) (*model.ListResponse[OpsAuditLog], error) {
	return model.NewListResponse([]OpsAuditLog{}, 0, 1, 20), nil
}

func newTestApprovalService(taskRepo *mockTaskRepo) *ApprovalService {
	audit := NewAuditLogService(&mockAuditLogRepo{}, zap.NewNop())
	return NewApprovalService(taskRepo, audit, zap.NewNop())
}

func TestApprovalService_Approve_SelfApprovalForbidden(t *testing.T) {
	creator := uuid.New() // 同时充当 creator 和 approver
	taskID := uuid.New()
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Creator: creator.String(), ApprovalState: ApprovalPending}, nil
		},
	}
	svc := newTestApprovalService(taskRepo)

	err := svc.Approve(context.Background(), taskID, creator, true, "self")
	assert.ErrorIs(t, err, ErrSelfApprovalForbidden,
		"创建者审批自己应被 4-eye 阻止")
}

func TestApprovalService_Approve_NonPendingRejected(t *testing.T) {
	creator := uuid.New()
	approver := uuid.New()
	taskID := uuid.New()

	cases := []struct {
		name  string
		state ApprovalState
	}{
		{"already_approved", ApprovalApproved},
		{"already_rejected", ApprovalRejected},
		{"not_required", ApprovalNotRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			taskRepo := &mockTaskRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
					return &OpsTask{ID: taskID, Creator: creator.String(), ApprovalState: tc.state}, nil
				},
			}
			svc := newTestApprovalService(taskRepo)

			err := svc.Approve(context.Background(), taskID, approver, true, "")
			assert.ErrorIs(t, err, ErrApprovalNotPending,
				"非 pending 状态不应再走审批 (got state=%q)", tc.state)
		})
	}
}

func TestApprovalService_Approve_TransitsToRunning(t *testing.T) {
	creator := uuid.New()
	approver := uuid.New()
	taskID := uuid.New()

	var capturedApprove bool
	var capturedApprover uuid.UUID
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Creator: creator.String(), ApprovalState: ApprovalPending}, nil
		},
		updateApprovalFn: func(_ context.Context, _, aid uuid.UUID, approve bool, _ time.Time) error {
			capturedApprove = approve
			capturedApprover = aid
			return nil
		},
	}
	svc := newTestApprovalService(taskRepo)

	err := svc.Approve(context.Background(), taskID, approver, true, "looks good")
	require.NoError(t, err)
	assert.True(t, capturedApprove, "approve=true 应传到 UpdateApproval")
	assert.Equal(t, approver, capturedApprover, "approver_user_id 应透传")
}

func TestApprovalService_Approve_TransitsToCancelled(t *testing.T) {
	creator := uuid.New()
	approver := uuid.New()
	taskID := uuid.New()

	var capturedApprove bool
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Creator: creator.String(), ApprovalState: ApprovalPending}, nil
		},
		updateApprovalFn: func(_ context.Context, _, _ uuid.UUID, approve bool, _ time.Time) error {
			capturedApprove = approve
			return nil
		},
	}
	svc := newTestApprovalService(taskRepo)

	err := svc.Approve(context.Background(), taskID, approver, false, "too risky")
	require.NoError(t, err)
	assert.False(t, capturedApprove, "approve=false（拒绝）应传到 UpdateApproval")
}

func TestApprovalService_Approve_UpdateApprovalErrorPropagates(t *testing.T) {
	creator := uuid.New()
	approver := uuid.New()
	taskID := uuid.New()

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: taskID, Creator: creator.String(), ApprovalState: ApprovalPending}, nil
		},
		updateApprovalFn: func(_ context.Context, _, _ uuid.UUID, _ bool, _ time.Time) error {
			return errors.New("db unreachable")
		},
	}
	svc := newTestApprovalService(taskRepo)

	err := svc.Approve(context.Background(), taskID, approver, true, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persist approval decision",
		"持久化失败应 wrap 出明确错误信息")
}
