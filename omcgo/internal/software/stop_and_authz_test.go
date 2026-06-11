package software

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ===========================================================================
// #59 Problem 3 — service-level 紧急叫停（SuspendUpgrade / canary 阈值）止血
// ===========================================================================

// TestSuspendUpgrade_CancelsExecutionContext proves SuspendUpgrade fires the
// cancelable-context registry so in-flight goroutines see ctx.Done. We register
// an execution ctx for the task (as startExecution would), then assert Suspend
// cancels it.
func TestSuspendUpgrade_CancelsExecutionContext(t *testing.T) {
	taskID := uuid.New()
	taskRepo := &svcMockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{ID: id, Status: TaskInProgress}, nil
		},
	}
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, taskRepo, &svcMockSubTaskRepo{},
		&svcMockDeviceRepo{}, &svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	// Simulate an in-flight execution batch registered for this task.
	execCtx, done := svc.launchControlledExecution(taskID, 1)
	defer done()
	require.NoError(t, execCtx.Err(), "exec ctx live before suspend")

	require.NoError(t, svc.SuspendUpgrade(context.Background(), taskID))
	assert.Error(t, execCtx.Err(), "SuspendUpgrade must cancel the task's execution context")
}

// TestTerminateUpgrade_CancelsExecutionContext mirrors the above for Terminate.
func TestTerminateUpgrade_CancelsExecutionContext(t *testing.T) {
	taskID := uuid.New()
	taskRepo := &svcMockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{ID: id, Status: TaskInProgress}, nil
		},
	}
	subRepo := &svcMockSubTaskRepo{} // empty list → terminate loop is a no-op
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, taskRepo, subRepo,
		&svcMockDeviceRepo{}, &svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	execCtx, done := svc.launchControlledExecution(taskID, 1)
	defer done()

	require.NoError(t, svc.TerminateUpgrade(context.Background(), taskID))
	assert.Error(t, execCtx.Err(), "TerminateUpgrade must cancel the task's execution context")
}

// recordingSuspender records SuspendUpgrade calls so the test can assert the
// canary threshold path actually triggers a real suspend (#59 Problem 3.3).
type recordingSuspender struct {
	called []uuid.UUID
	retErr error
}

func (s *recordingSuspender) SuspendUpgrade(_ context.Context, taskID uuid.UUID) error {
	s.called = append(s.called, taskID)
	return s.retErr
}

// thresholdRepo builds a fakeCanaryRepo (reused from canary_monitor_test.go) with
// a single running canary task whose stage has crossed its failure threshold.
func thresholdRepo(id uuid.UUID) *fakeCanaryRepo {
	repo := newFakeCanaryRepo()
	repo.activeIDs = []uuid.UUID{id}
	repo.canary[id] = &CanaryFields{
		TaskID:       id,
		Strategy:     StrategyCanary,
		Stages:       DefaultCanaryStages,
		CurrentStage: 1,
		StageStatus:  StageStatusRunning,
		TotalCount:   100,
	}
	// stage 1 = 1 device (1%); 1 fail / 0 success → 100% > 5% threshold.
	repo.tasks[id] = &UpgradeTask{ID: id, FailCount: 1, SuccessCount: 0, TotalCount: 100}
	return repo
}

// TestCanaryMonitor_ThresholdExceeded_CallsSuspend verifies the canary monitor
// calls SuspendUpgrade (真正止血) when a stage crosses its failure threshold,
// instead of merely flipping StageStatus.
func TestCanaryMonitor_ThresholdExceeded_CallsSuspend(t *testing.T) {
	id := uuid.New()
	repo := thresholdRepo(id)
	monitor := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	suspender := &recordingSuspender{}
	monitor.SetSuspender(suspender)

	require.NoError(t, monitor.CheckCanaryTasks(context.Background()))

	require.Len(t, suspender.called, 1, "threshold exceeded must call SuspendUpgrade")
	assert.Equal(t, id, suspender.called[0])
	require.NotNil(t, repo.updatedWith[id])
	assert.Equal(t, StageStatusPaused, repo.updatedWith[id].StageStatus, "StageStatus still flips to paused")
}

// TestCanaryMonitor_ThresholdExceeded_SuspendErrorDoesNotBlock confirms the pause
// is still persisted even if the suspend call errors (best-effort止血).
func TestCanaryMonitor_ThresholdExceeded_SuspendErrorDoesNotBlock(t *testing.T) {
	id := uuid.New()
	repo := thresholdRepo(id)
	monitor := NewCanaryMonitor(repo, NewCanaryMetrics(nil), zap.NewNop())
	monitor.SetSuspender(&recordingSuspender{retErr: commonerrors.ErrInvalidInput})

	require.NoError(t, monitor.CheckCanaryTasks(context.Background()))
	require.NotNil(t, repo.updatedWith[id])
	assert.Equal(t, StageStatusPaused, repo.updatedWith[id].StageStatus)
}

// ===========================================================================
// #59 Problem 4 — per-device 归属校验（service + handler）
// ===========================================================================

// authzFakeReader maps each device to a fixed set of groups.
type authzFakeReader struct {
	groups map[uuid.UUID][]uuid.UUID
}

func (r authzFakeReader) GetDeviceGroupIDs(_ context.Context, deviceID uuid.UUID) ([]uuid.UUID, error) {
	return r.groups[deviceID], nil
}

// authzFakeResolver returns fixed visible groups for the caller.
type authzFakeResolver struct {
	groups []uuid.UUID
	isSuper bool
}

func (r authzFakeResolver) GetUserVisibleGroupIDs(_ context.Context, _ uuid.UUID, _ bool) ([]uuid.UUID, error) {
	if r.isSuper {
		return nil, nil // super admin: nil → unrestricted
	}
	return r.groups, nil
}

func TestService_AuthorizeDevicesAccess(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()
	inScope := uuid.New()
	outScope := uuid.New()
	reader := authzFakeReader{groups: map[uuid.UUID][]uuid.UUID{
		inScope:  {g1},
		outScope: {g2},
	}}
	svc := &SoftwareService{logger: zap.NewNop()}
	svc.SetDeviceGroupReader(reader)

	t.Run("in-scope device allowed", func(t *testing.T) {
		err := svc.AuthorizeDevicesAccess(context.Background(), []uuid.UUID{g1}, inScope)
		assert.NoError(t, err)
	})
	t.Run("out-of-scope device forbidden", func(t *testing.T) {
		err := svc.AuthorizeDevicesAccess(context.Background(), []uuid.UUID{g1}, outScope)
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})
	t.Run("batch with one out-of-scope fails whole batch", func(t *testing.T) {
		err := svc.AuthorizeDevicesAccess(context.Background(), []uuid.UUID{g1}, inScope, outScope)
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})
	t.Run("super admin nil groups allowed", func(t *testing.T) {
		err := svc.AuthorizeDevicesAccess(context.Background(), nil, inScope, outScope)
		assert.NoError(t, err)
	})
	t.Run("empty groups fail-closed", func(t *testing.T) {
		err := svc.AuthorizeDevicesAccess(context.Background(), []uuid.UUID{}, inScope)
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})
	t.Run("reader not injected → nil-safe allow", func(t *testing.T) {
		bare := &SoftwareService{logger: zap.NewNop()}
		err := bare.AuthorizeDevicesAccess(context.Background(), []uuid.UUID{g2}, outScope)
		assert.NoError(t, err)
	})
}

// setupAuthzHandler builds a handler wired with a fake resolver + a service whose
// groupReader is the given reader. The router injects userID/isSuperAdmin into the
// gin ctx so the resolver path runs.
func setupAuthzHandler(t *testing.T, reader authzFakeReader, resolver authzFakeResolver, svc *SoftwareService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc.SetDeviceGroupReader(reader)
	h := NewHandler(svc, zap.NewNop())
	h.SetPermissionService(resolver)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, uuid.New())
		c.Set(admin.CtxKeyIsSuperAdmin, resolver.isSuper)
		c.Next()
	})
	h.RegisterRoutes(r.Group(""))
	return r
}

func postJSON(t *testing.T, r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestCreateUpgradeTask_ForbidsOutOfScopeDevice(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()
	outScope := uuid.New()
	reader := authzFakeReader{groups: map[uuid.UUID][]uuid.UUID{outScope: {g2}}}
	svc := &SoftwareService{logger: zap.NewNop()}
	r := setupAuthzHandler(t, reader, authzFakeResolver{groups: []uuid.UUID{g1}}, svc)

	w := postJSON(t, r, "/upgrade-tasks", BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{outScope},
		FirmwareID: uuid.New(),
		TaskName:   "t1",
	})
	assert.Equal(t, http.StatusForbidden, w.Code, "out-of-scope upgrade must be 403")
}

func TestCreateUpgradeTask_AllowsInScopeDevice(t *testing.T) {
	g1 := uuid.New()
	inScope := uuid.New()
	fwID := uuid.New()
	reader := authzFakeReader{groups: map[uuid.UUID][]uuid.UUID{inScope: {g1}}}

	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{ID: id, Version: "V1.0.0", FileName: "fw.bin",
				MinIOPath: "p/fw.bin", MD5Val: "abc", ProductClass: "SC"}, nil
		},
	}
	devRepo := &svcMockDeviceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: id, SerialNumber: "SN-1"}, nil
		},
	}
	// Full service so BatchUpgrade can persist + launch (no redis → execution
	// goroutine fails fast, but the create path returns 201 before that matters).
	svc := NewSoftwareService(
		fwRepo, &svcMockTaskRepo{}, &svcMockSubTaskRepo{}, devRepo, &svcMockCmdQueue{},
		nil, nil, "test-bucket", &svcMockEventBus{}, nil, zap.NewNop(),
	)
	r := setupAuthzHandler(t, reader, authzFakeResolver{groups: []uuid.UUID{g1}}, svc)

	w := postJSON(t, r, "/upgrade-tasks", BatchUpgradeRequest{
		DeviceIDs:       []uuid.UUID{inScope},
		FirmwareID:      fwID,
		TaskName:        "t1",
		CreateSuspended: true, // suspended: no execution goroutine, deterministic
	})
	assert.Equal(t, http.StatusCreated, w.Code, "in-scope upgrade must be 201")
}

func TestCreateRollback_ForbidsOutOfScopeDevice(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()
	outScope := uuid.New()
	reader := authzFakeReader{groups: map[uuid.UUID][]uuid.UUID{outScope: {g2}}}
	svc := &SoftwareService{logger: zap.NewNop()}
	r := setupAuthzHandler(t, reader, authzFakeResolver{groups: []uuid.UUID{g1}}, svc)

	w := postJSON(t, r, "/upgrade-tasks/rollback", RollbackRequest{
		DeviceIDs:  []uuid.UUID{outScope},
		TaskName:   "rb1",
		CreateUser: "admin",
	})
	assert.Equal(t, http.StatusForbidden, w.Code, "out-of-scope rollback must be 403")
}

func TestCreateUpgradeTask_SuperAdminAllowed(t *testing.T) {
	g2 := uuid.New()
	anyDevice := uuid.New()
	reader := authzFakeReader{groups: map[uuid.UUID][]uuid.UUID{anyDevice: {g2}}}
	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{ID: id, Version: "V1.0.0", FileName: "fw.bin",
				MinIOPath: "p/fw.bin", MD5Val: "abc"}, nil
		},
	}
	svc := NewSoftwareService(
		fwRepo, &svcMockTaskRepo{}, &svcMockSubTaskRepo{}, &svcMockDeviceRepo{}, &svcMockCmdQueue{},
		nil, nil, "test-bucket", &svcMockEventBus{}, nil, zap.NewNop(),
	)
	// Super admin: resolver returns nil → unrestricted, even for a device the
	// caller has no explicit group for.
	r := setupAuthzHandler(t, reader, authzFakeResolver{isSuper: true}, svc)

	w := postJSON(t, r, "/upgrade-tasks", BatchUpgradeRequest{
		DeviceIDs:       []uuid.UUID{anyDevice},
		FirmwareID:      uuid.New(),
		TaskName:        "t1",
		CreateSuspended: true,
	})
	assert.Equal(t, http.StatusCreated, w.Code, "super admin must bypass per-device authz")
}

// ===========================================================================
// #59 Problem 4.2 — rollback 降级守卫
// ===========================================================================

func TestService_RollbackDevices_DowngradeGuard(t *testing.T) {
	targetFW := uuid.New()
	deviceID := uuid.New()

	newSvc := func(targetVersion, deviceVersion string) *SoftwareService {
		fwRepo := &svcMockFirmwareRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
				return &FirmwareVersion{ID: id, Version: targetVersion}, nil
			},
		}
		devRepo := &svcMockDeviceRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
				return &model.Device{ID: id, SerialNumber: "SN-RB", FirmwareVersion: deviceVersion}, nil
			},
		}
		return NewSoftwareService(
			fwRepo, &svcMockTaskRepo{}, &svcMockSubTaskRepo{}, devRepo, &svcMockCmdQueue{},
			nil, nil, "test-bucket", &svcMockEventBus{}, nil, zap.NewNop(),
		)
	}

	t.Run("downgrade blocked without force", func(t *testing.T) {
		svc := newSvc("V2.0.0", "V3.0.0") // target older than current → downgrade
		_, err := svc.RollbackDevices(context.Background(), RollbackRequest{
			DeviceIDs:        []uuid.UUID{deviceID},
			TaskName:         "rb-down",
			CreateUser:       "admin",
			TargetFirmwareID: &targetFW,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	})

	t.Run("downgrade allowed with force", func(t *testing.T) {
		svc := newSvc("V2.0.0", "V3.0.0")
		task, err := svc.RollbackDevices(context.Background(), RollbackRequest{
			DeviceIDs:        []uuid.UUID{deviceID},
			TaskName:         "rb-down-forced",
			CreateUser:       "admin",
			TargetFirmwareID: &targetFW,
			Force:            true,
		})
		require.NoError(t, err)
		require.NotNil(t, task)
	})

	t.Run("same-or-newer target not blocked", func(t *testing.T) {
		svc := newSvc("V3.0.0", "V3.0.0") // equal → not a downgrade
		task, err := svc.RollbackDevices(context.Background(), RollbackRequest{
			DeviceIDs:        []uuid.UUID{deviceID},
			TaskName:         "rb-equal",
			CreateUser:       "admin",
			TargetFirmwareID: &targetFW,
		})
		require.NoError(t, err)
		require.NotNil(t, task)
	})

	t.Run("no explicit target → guard not applied (legacy own-bank rollback)", func(t *testing.T) {
		svc := newSvc("ignored", "V3.0.0")
		task, err := svc.RollbackDevices(context.Background(), RollbackRequest{
			DeviceIDs:  []uuid.UUID{deviceID},
			TaskName:   "rb-ownbank",
			CreateUser: "admin",
			// no TargetFirmwareID
		})
		require.NoError(t, err)
		require.NotNil(t, task)
	})
}
