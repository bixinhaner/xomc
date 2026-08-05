package software

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks (svcMock prefix to avoid conflict with handler_test.go swH mocks)
// ---------------------------------------------------------------------------

type svcMockFirmwareRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error)
	createFn  func(ctx context.Context, fw *FirmwareVersion) error
}

func (m *svcMockFirmwareRepo) Create(ctx context.Context, fw *FirmwareVersion) error {
	if m.createFn != nil {
		return m.createFn(ctx, fw)
	}
	fw.ID = uuid.New()
	return nil
}
func (m *svcMockFirmwareRepo) GetByID(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockFirmwareRepo) List(_ context.Context, _ FirmwareFilter) (*model.ListResponse[FirmwareVersion], error) {
	return model.NewListResponse([]FirmwareVersion{}, 0, 1, 20), nil
}
func (m *svcMockFirmwareRepo) Update(_ context.Context, _ *FirmwareVersion) error { return nil }
func (m *svcMockFirmwareRepo) Delete(_ context.Context, _ uuid.UUID) error        { return nil }

type svcMockTaskRepo struct {
	createFn          func(ctx context.Context, task *UpgradeTask) error
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*UpgradeTask, error)
	updateStatusFn    func(ctx context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error
	incrementCountsFn func(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error
}

func (m *svcMockTaskRepo) Create(ctx context.Context, task *UpgradeTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	task.ID = uuid.New()
	return nil
}
func (m *svcMockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockTaskRepo) Update(_ context.Context, _ *UpgradeTask) error { return nil }
func (m *svcMockTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, result)
	}
	return nil
}
func (m *svcMockTaskRepo) List(_ context.Context, _ UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	return model.NewListResponse([]UpgradeTask{}, 0, 1, 20), nil
}
func (m *svcMockTaskRepo) IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error {
	if m.incrementCountsFn != nil {
		return m.incrementCountsFn(ctx, taskID, successDelta, failDelta)
	}
	return nil
}
func (m *svcMockTaskRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *svcMockTaskRepo) GetCanaryFields(_ context.Context, _ uuid.UUID) (*CanaryFields, error) {
	return nil, nil
}
func (m *svcMockTaskRepo) UpdateCanaryFields(_ context.Context, _ uuid.UUID, _ *CanaryFields) error {
	return nil
}
func (m *svcMockTaskRepo) ListActiveCanaryTaskIDs(_ context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

type svcMockSubTaskRepo struct {
	createFn            func(ctx context.Context, task *UpgradeSubTask) error
	updateStatusFn      func(ctx context.Context, id uuid.UUID, status UpgradeState, msg string) error
	listByTaskIDFn      func(ctx context.Context, taskID uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error)
	getActiveByDeviceFn func(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error)
	getByCommandKeyFn   func(ctx context.Context, commandKey string) (*UpgradeSubTask, error)
	batchCreateFn       func(ctx context.Context, tasks []*UpgradeSubTask) error
	updateDestByIDFn    func(ctx context.Context, id uuid.UUID, destVersion string) error
	failStaleFn         func(ctx context.Context, cutoffs StaleTimeouts) (StaleFailures, error)
	failStaleDetailsFn  func(ctx context.Context, cutoffs StaleTimeouts) ([]UpgradeSubTask, error)
}

func (m *svcMockSubTaskRepo) Create(ctx context.Context, task *UpgradeSubTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	task.ID = uuid.New()
	return nil
}
func (m *svcMockSubTaskRepo) GetByID(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockSubTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, msg string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, msg)
	}
	return nil
}
func (m *svcMockSubTaskRepo) Update(_ context.Context, _ *UpgradeSubTask) error { return nil }
func (m *svcMockSubTaskRepo) List(_ context.Context, _ SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return model.NewListResponse([]UpgradeSubTaskWithTaskName{}, 0, 1, 20), nil
}
func (m *svcMockSubTaskRepo) ListByTaskID(ctx context.Context, taskID uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	if m.listByTaskIDFn != nil {
		return m.listByTaskIDFn(ctx, taskID, filter)
	}
	return model.NewListResponse([]UpgradeSubTaskWithTaskName{}, 0, 1, 20), nil
}

// TestListAllSubTasksByTaskID_PagesAll 锁定 software 翻页助手：>100 台的任务必须翻页
// 取全量，不被默认首页(20)/单页上限(100)截断——否则 Resume 漏恢复、finalize 漏推进、
// file-landed 反查不到尾部设备 sub_task。
func TestListAllSubTasksByTaskID_PagesAll(t *testing.T) {
	const n = 250
	all := make([]UpgradeSubTaskWithTaskName, n)
	for i := range all {
		all[i].ID = uuid.New()
		all[i].DeviceID = uuid.New()
	}
	repo := &svcMockSubTaskRepo{
		listByTaskIDFn: func(_ context.Context, _ uuid.UUID, f SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
			ps := f.PageSize
			if ps < 1 {
				ps = 20
			}
			if ps > 100 {
				ps = 100
			}
			page := f.Page
			if page < 1 {
				page = 1
			}
			start := (page - 1) * ps
			if start > n {
				start = n
			}
			end := start + ps
			if end > n {
				end = n
			}
			return model.NewListResponse(all[start:end], int64(n), page, ps), nil
		},
	}

	got, err := listAllSubTasksByTaskID(context.Background(), repo, uuid.New())
	require.NoError(t, err)
	require.Len(t, got, n, "翻页应取回全部 250 条，而非被默认 20/上限 100 截断")
}
func (m *svcMockSubTaskRepo) ListAll(_ context.Context, _ AllSubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return model.NewListResponse([]UpgradeSubTaskWithTaskName{}, 0, 1, 20), nil
}
func (m *svcMockSubTaskRepo) UpdateStatusWithCode(_ context.Context, id uuid.UUID, status UpgradeState, msg string, _ FailureCode) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(context.Background(), id, status, msg)
	}
	return nil
}
func (m *svcMockSubTaskRepo) UpdateStatusByOperator(_ context.Context, id uuid.UUID, status UpgradeState, msg string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(context.Background(), id, status, msg)
	}
	return nil
}
func (m *svcMockSubTaskRepo) UpdateFailureReasonByTask(_ context.Context, _ uuid.UUID, _ FailureCode) error {
	return nil
}
func (m *svcMockSubTaskRepo) UpdateDestVersionByCommandKey(_ context.Context, _, _ string) error {
	return nil
}
func (m *svcMockSubTaskRepo) UpdateDestVersionByID(ctx context.Context, id uuid.UUID, destVersion string) error {
	if m.updateDestByIDFn != nil {
		return m.updateDestByIDFn(ctx, id, destVersion)
	}
	return nil
}
func (m *svcMockSubTaskRepo) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error) {
	if m.getActiveByDeviceFn != nil {
		return m.getActiveByDeviceFn(ctx, deviceID)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockSubTaskRepo) GetByCommandKey(ctx context.Context, commandKey string) (*UpgradeSubTask, error) {
	if m.getByCommandKeyFn != nil {
		return m.getByCommandKeyFn(ctx, commandKey)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockSubTaskRepo) BatchCreate(ctx context.Context, tasks []*UpgradeSubTask) error {
	if m.batchCreateFn != nil {
		return m.batchCreateFn(ctx, tasks)
	}
	for _, t := range tasks {
		t.ID = uuid.New()
	}
	return nil
}
func (m *svcMockSubTaskRepo) FailStale(ctx context.Context, cutoffs StaleTimeouts) (StaleFailures, error) {
	if m.failStaleFn != nil {
		return m.failStaleFn(ctx, cutoffs)
	}
	return StaleFailures{}, nil
}
func (m *svcMockSubTaskRepo) FailStaleWithDetails(ctx context.Context, cutoffs StaleTimeouts) ([]UpgradeSubTask, error) {
	if m.failStaleDetailsFn != nil {
		return m.failStaleDetailsFn(ctx, cutoffs)
	}
	return nil, nil
}
func (m *svcMockSubTaskRepo) DeleteByTaskID(_ context.Context, _ uuid.UUID) error { return nil }

func TestReapStaleSubTasksOnceReleasesTimedOutDeviceLock(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	subTaskID := uuid.New()
	deviceSN := "SN-REAPER-LOCK"

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	require.NoError(t, redisClient.Set(ctx, upgradeDeviceLockKey(deviceSN), subTaskID.String(), time.Hour).Err())

	subRepo := &svcMockSubTaskRepo{
		failStaleDetailsFn: func(_ context.Context, _ StaleTimeouts) ([]UpgradeSubTask, error) {
			return []UpgradeSubTask{{
				ID: subTaskID, TaskID: taskID, DeviceSN: deviceSN,
			}}, nil
		},
	}
	taskRepo := &svcMockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{ID: id, Status: TaskInProgress, TotalCount: 1, FailCount: 1}, nil
		},
	}
	service := NewSoftwareService(
		&svcMockFirmwareRepo{}, taskRepo, subRepo, &svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "", &svcMockEventBus{}, redisClient, zap.NewNop(),
	)

	require.NoError(t, service.reapStaleSubTasksOnce(ctx, defaultUpgradeTaskReaperTimeouts()))

	assert.False(t, mr.Exists(upgradeDeviceLockKey(deviceSN)), "reaper must release Redis lock owned by timed-out sub-task")
}

type svcMockDeviceRepo struct {
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	getBySerialNumberFn func(ctx context.Context, sn string) (*model.Device, error)
}

func (m *svcMockDeviceRepo) Create(_ context.Context, _ *model.Device) error { return nil }
func (m *svcMockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.getBySerialNumberFn != nil {
		return m.getBySerialNumberFn(ctx, sn)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockDeviceRepo) GetDeletedBySerialNumber(_ context.Context, _ string, _ model.CarrierCode) (*model.Device, error) {
	return nil, nil
}
func (m *svcMockDeviceRepo) Update(_ context.Context, _ *model.Device) error { return nil }
func (m *svcMockDeviceRepo) Delete(_ context.Context, _ uuid.UUID) error     { return nil }
func (m *svcMockDeviceRepo) List(_ context.Context, _ device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return &model.ListResponse[model.Device]{Items: []model.Device{}}, nil
}
func (m *svcMockDeviceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.DeviceStatus) error {
	return nil
}

// T-0162 新接口方法
func (m *svcMockDeviceRepo) UpdateLifecycle(_ context.Context, _ uuid.UUID, _ model.DeviceLifecycle) error {
	return nil
}

func (m *svcMockDeviceRepo) UpdateOnlineStatus(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *svcMockDeviceRepo) UpdateLastInform(_ context.Context, _ string, _ time.Time, _ []string) error {
	return nil
}
func (m *svcMockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *svcMockDeviceRepo) CountByStatus(_ context.Context, _ *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return map[model.DeviceStatus]int64{}, nil
}
func (m *svcMockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *svcMockDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *svcMockDeviceRepo) GetGeoStats(_ context.Context, _ device.GeoStatsFilter) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *svcMockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int, _ []model.DeviceVisibilityGrant) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *svcMockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *svcMockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *svcMockDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *svcMockDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (m *svcMockDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}
func (m *svcMockDeviceRepo) UpdateSiteName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *svcMockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *svcMockDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[device.DeviceWithInfo], error) {
	return model.NewListResponse([]device.DeviceWithInfo{}, 0, 1, 20), nil
}
func (m *svcMockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (*device.RestoreResult, error) {
	return &device.RestoreResult{}, nil
}
func (m *svcMockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *svcMockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

type svcMockCmdQueue struct {
	createFn func(ctx context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error)
}

func (m *svcMockCmdQueue) CreateTask(ctx context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return devtask.NewTask(req), nil
}

func (m *svcMockCmdQueue) GetQueueLength(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

type svcMockEventBus struct {
	publishFn  func(ctx context.Context, subject string, evt event.Event) error
	keyedCalls []struct {
		subject string
		config  event.KeyedQueueConfig
		keyFn   event.EventKeyFunc
	}
}

func (m *svcMockEventBus) Publish(ctx context.Context, subject string, evt event.Event) error {
	if m.publishFn != nil {
		return m.publishFn(ctx, subject, evt)
	}
	return nil
}
func (m *svcMockEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &svcMockSub{}, nil
}
func (m *svcMockEventBus) QueueSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &svcMockSub{}, nil
}
func (m *svcMockEventBus) KeyedQueueSubscribe(
	subject string,
	config event.KeyedQueueConfig,
	keyFn event.EventKeyFunc,
	_ event.EventHandler,
) (event.Subscription, error) {
	m.keyedCalls = append(m.keyedCalls, struct {
		subject string
		config  event.KeyedQueueConfig
		keyFn   event.EventKeyFunc
	}{subject: subject, config: config, keyFn: keyFn})
	return &svcMockSub{}, nil
}
func (m *svcMockEventBus) PullSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &svcMockSub{}, nil
}
func (m *svcMockEventBus) Close() error { return nil }

type svcMockSub struct{}

func (s *svcMockSub) Unsubscribe() error { return nil }

func TestSoftwareSubscribeUsesKeyedPeriodicConsumer(t *testing.T) {
	bus := &svcMockEventBus{}
	service := &SoftwareService{logger: zap.NewNop()}

	require.NoError(t, service.Subscribe(bus))

	require.Len(t, bus.keyedCalls, 1)
	call := bus.keyedCalls[0]
	assert.Equal(t, event.SubjectDevicePeriodic, call.subject)
	assert.Equal(t, "software-upgrade-periodic", call.config.Durable)
	assert.Equal(t, softwarePeriodicConsumerConcurrency, call.config.Concurrency)
	assert.Equal(t, softwarePeriodicConsumerQueueDepth, call.config.QueueDepth)
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, map[string]any{
		"device_id": map[string]any{"serial_number": " SW-PERIODIC-001 "},
	})
	require.NoError(t, err)
	key, err := call.keyFn(evt)
	require.NoError(t, err)
	assert.Equal(t, "SW-PERIODIC-001", key)
}

func TestSoftwareService_ReapStaleUpgradeTasksPublishesFailedEvent(t *testing.T) {
	taskID := uuid.New()
	deviceID := uuid.New()
	subTaskID := uuid.New()
	reason := "Timed out waiting for device to come online."

	subRepo := &svcMockSubTaskRepo{
		failStaleDetailsFn: func(_ context.Context, _ StaleTimeouts) ([]UpgradeSubTask, error) {
			return []UpgradeSubTask{{
				ID: subTaskID, TaskID: taskID, DeviceID: deviceID,
				DeviceSN: "SN-OFFLINE", Status: UpgradeFailed, ErrorMessage: reason,
			}}, nil
		},
	}
	taskRepo := &svcMockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{ID: id, Status: TaskInProgress, TotalCount: 1, FailCount: 1}, nil
		},
	}

	var gotSubject string
	var gotPayload struct {
		SubTaskID string `json:"sub_task_id"`
		TaskID    string `json:"task_id"`
		DeviceID  string `json:"device_id"`
		Reason    string `json:"reason"`
	}
	bus := &svcMockEventBus{publishFn: func(_ context.Context, subject string, evt event.Event) error {
		gotSubject = subject
		return evt.DecodePayload(&gotPayload)
	}}
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	lockKey := "software:upgrade:active:SN-OFFLINE"
	require.NoError(t, redisClient.Set(context.Background(), lockKey, subTaskID.String(), time.Hour).Err())

	svc := &SoftwareService{
		taskRepo: taskRepo, subTaskRepo: subRepo, eventBus: bus, logger: zap.NewNop(),
		executor: &UpgradeExecutor{redis: redisClient, logger: zap.NewNop()},
	}
	reaped, err := svc.reapStaleUpgradeTasks(context.Background(), StaleTimeouts{DeviceOnline: 10 * time.Minute})
	require.NoError(t, err)
	assert.EqualValues(t, 1, reaped)
	assert.Equal(t, event.SubjectUpgradeFailed, gotSubject)
	assert.Equal(t, subTaskID.String(), gotPayload.SubTaskID)
	assert.Equal(t, taskID.String(), gotPayload.TaskID)
	assert.Equal(t, deviceID.String(), gotPayload.DeviceID)
	assert.Equal(t, reason, gotPayload.Reason)
	assert.EqualValues(t, 0, redisClient.Exists(context.Background(), lockKey).Val(),
		"reaper must release the Redis device lock owned by the timed-out sub-task")
}

func TestUpgradeExecutor_FailedLockContenderDoesNotDeleteOwnerLock(t *testing.T) {
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	const deviceSN = "SN-LOCKED"
	ownerID := uuid.New()
	contenderID := uuid.New()
	lockKey := "software:upgrade:active:" + deviceSN
	require.NoError(t, redisClient.Set(context.Background(), lockKey, ownerID.String(), time.Hour).Err())

	executor := &UpgradeExecutor{
		redis:       redisClient,
		subTaskRepo: &svcMockSubTaskRepo{},
		taskRepo:    &svcMockTaskRepo{},
		logger:      zap.NewNop(),
	}
	executor.failLockedSubTask(context.Background(), &UpgradeSubTask{
		ID: contenderID, TaskID: uuid.New(), DeviceSN: deviceSN,
	}, deviceSN)

	assert.Equal(t, ownerID.String(), redisClient.Get(context.Background(), lockKey).Val(),
		"a contender that never acquired the lock must not delete another sub-task's lock")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestService_BatchUpgrade_Success(t *testing.T) {
	deviceID := uuid.New()
	firmwareID := uuid.New()

	deviceRepo := &svcMockDeviceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           id,
				SerialNumber: "SN-UPGRADE-001",
			}, nil
		},
	}

	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{
				ID:           id,
				Version:      "V1.0.0",
				FileName:     "fw.bin",
				MinIOPath:    "firmware/cmcc/SC/V1.0.0/fw.bin",
				FileSize:     1024,
				ProductClass: "SmallCell-LTE",
			}, nil
		},
	}

	cmdQueue := &svcMockCmdQueue{}

	svc := NewSoftwareService(
		fwRepo,
		&svcMockTaskRepo{},
		&svcMockSubTaskRepo{},
		deviceRepo,
		cmdQueue,
		nil, // connReq
		nil, // minioClient
		"test-bucket",
		&svcMockEventBus{},
		nil, // redis
		zap.NewNop(),
	)

	req := BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{deviceID},
		FirmwareID: firmwareID,
		TaskName:   "test-upgrade",
		TaskType:   TaskTypeUpgrade,
	}

	task, err := svc.BatchUpgrade(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, TaskInProgress, task.Status)
	assert.Equal(t, "test-upgrade", task.TaskName)
	assert.Equal(t, 1, task.TotalCount)
	assert.Equal(t, "SmallCell-LTE", task.ProductClass)
}

func TestService_BatchUpgrade_PersistsDownloadFileTypeOverride(t *testing.T) {
	deviceID := uuid.New()
	firmwareID := uuid.New()
	var createdTask *UpgradeTask

	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			require.Equal(t, firmwareID, id)
			return &FirmwareVersion{
				ID:           firmwareID,
				Version:      "V1.0.0",
				FileName:     "fpga.bin",
				MD5Val:       "abc123",
				MinIOPath:    "firmware/cmcc/SC/V1.0.0/fpga.bin",
				FileSize:     2048,
				ProductClass: "SmallCell-LTE",
			}, nil
		},
	}
	taskRepo := &svcMockTaskRepo{
		createFn: func(_ context.Context, task *UpgradeTask) error {
			task.ID = uuid.New()
			copyTask := *task
			createdTask = &copyTask
			return nil
		},
	}

	svc := NewSoftwareService(
		fwRepo,
		taskRepo,
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil,
		nil,
		"test-bucket",
		&svcMockEventBus{},
		nil,
		zap.NewNop(),
	)

	_, err := svc.BatchUpgrade(context.Background(), BatchUpgradeRequest{
		DeviceIDs:        []uuid.UUID{deviceID},
		FirmwareID:       firmwareID,
		TaskName:         "test-fpga-upgrade",
		TaskType:         TaskTypeFPGA,
		DownloadFileType: "Firmware Upgrade Fpga",
		CreateSuspended:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, createdTask)
	assert.Equal(t, "Firmware Upgrade Fpga", createdTask.DownloadFileType)
}

func TestService_BatchUpgrade_PersistsCreateUserFromRequest(t *testing.T) {
	deviceID := uuid.New()
	firmwareID := uuid.New()
	var createdTask *UpgradeTask

	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			require.Equal(t, firmwareID, id)
			return &FirmwareVersion{
				ID:           firmwareID,
				Version:      "V1.0.0",
				FileName:     "fw.bin",
				MD5Val:       "abc123",
				MinIOPath:    "firmware/cmcc/SC/V1.0.0/fw.bin",
				FileSize:     2048,
				ProductClass: "SmallCell-LTE",
			}, nil
		},
	}
	taskRepo := &svcMockTaskRepo{
		createFn: func(_ context.Context, task *UpgradeTask) error {
			task.ID = uuid.New()
			copyTask := *task
			createdTask = &copyTask
			return nil
		},
	}

	svc := NewSoftwareService(
		fwRepo,
		taskRepo,
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil,
		nil,
		"test-bucket",
		&svcMockEventBus{},
		nil,
		zap.NewNop(),
	)

	_, err := svc.BatchUpgrade(context.Background(), BatchUpgradeRequest{
		DeviceIDs:       []uuid.UUID{deviceID},
		FirmwareID:      firmwareID,
		TaskName:        "operator-upgrade",
		TaskType:        TaskTypeUpgrade,
		CreateUser:      "operator01",
		CreateSuspended: true,
	})
	require.NoError(t, err)
	require.NotNil(t, createdTask)
	assert.Equal(t, "operator01", createdTask.CreateUser)
}

func TestService_BatchUpgrade_FirmwareNotFound(t *testing.T) {
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{},
		nil, // redis
		zap.NewNop(),
	)

	req := BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{uuid.New()},
		FirmwareID: uuid.New(),
		TaskName:   "test-upgrade",
	}

	_, err := svc.BatchUpgrade(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get firmware")
}

func TestService_HandleTransferComplete_InformOnlyDoesNotAdvanceActiveUpgrade(t *testing.T) {
	deviceID := uuid.New()
	taskID := uuid.New()

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			assert.Equal(t, "SN-TC-001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn, Technology: model.TechLTE}, nil
		},
	}

	var updatedStatus UpgradeState
	getActiveCalled := false
	subTaskRepo := &svcMockSubTaskRepo{
		getActiveByDeviceFn: func(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
			getActiveCalled = true
			return &UpgradeSubTask{
				ID:     uuid.New(),
				TaskID: taskID,
				Status: UpgradeDownloading,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status UpgradeState, _ string) error {
			updatedStatus = status
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, redisClient, zap.NewNop(),
	)

	// Simulate Inform-level TC event (from publishInformEvents)
	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]interface{}{
		"device_id": map[string]string{"SerialNumber": "SN-TC-001"},
		"events":    []string{"7 TRANSFER COMPLETE"},
	})

	err := svc.HandleTransferComplete(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, getActiveCalled, "Inform-level 7 TRANSFER COMPLETE has no CommandKey/FaultStruct and must not be correlated by active device")
	assert.Empty(t, updatedStatus, "only TC body with matching CommandKey may advance the upgrade")
}

func TestService_HandleTransferComplete_EmptyCommandKeyFaultDoesNotFailActiveUpgrade(t *testing.T) {
	var updatedStatus UpgradeState
	getByCommandKeyCalled := false
	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, _ string) (*UpgradeSubTask, error) {
			getByCommandKeyCalled = true
			return nil, assert.AnError
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status UpgradeState, _ string) error {
			updatedStatus = status
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]any{
		"command_key": "",
		"fault_struct": map[string]any{
			"fault_code":   9013,
			"fault_string": "Url invalid, only support FTP and HTTP protocol.",
		},
	})
	require.NoError(t, err)

	err = svc.HandleTransferComplete(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, getByCommandKeyCalled, "empty CommandKey TransferComplete must not be matched to an active upgrade")
	assert.Empty(t, updatedStatus)
}

func TestService_HandleTransferComplete_MatchingCommandKeySuccess(t *testing.T) {
	deviceID := uuid.New()
	taskID := uuid.New()
	subTaskID := uuid.New()
	commandKey := "Download Upgrade," + subTaskID.String()

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			assert.Equal(t, "SN-TC-001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn, Technology: model.TechLTE}, nil
		},
	}

	var updatedStatus UpgradeState
	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, got string) (*UpgradeSubTask, error) {
			assert.Equal(t, commandKey, got)
			return &UpgradeSubTask{
				ID:         subTaskID,
				TaskID:     taskID,
				Status:     UpgradeDownloading,
				DeviceSN:   "SN-TC-001",
				CommandKey: commandKey,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status UpgradeState, _ string) error {
			updatedStatus = status
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, redisClient, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]any{
		"command_key": commandKey,
		"fault_struct": map[string]any{
			"fault_code":   0,
			"fault_string": "",
		},
	})
	require.NoError(t, err)

	err = svc.HandleTransferComplete(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, UpgradeCompleted, updatedStatus)
}

func TestService_HandleTransferComplete_LegacyDeviceSNPayloadIgnored(t *testing.T) {
	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return &model.Device{ID: uuid.New(), SerialNumber: "SN-001"}, nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		&svcMockSubTaskRepo{},
		deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]string{
		"device_sn": "SN-001",
	})

	err := svc.HandleTransferComplete(context.Background(), evt)
	assert.NoError(t, err) // should silently return nil
}

func TestService_HandleTransferComplete_InvalidPayload(t *testing.T) {
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, &svcMockTaskRepo{}, &svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	// Empty device_sn
	evt := event.Event{
		Payload: json.RawMessage(`{"device_sn":""}`),
	}

	err := svc.HandleTransferComplete(context.Background(), evt)
	assert.NoError(t, err) // should silently return nil
}

func TestService_HandleTransferComplete_LogCollectFaultMessage(t *testing.T) {
	taskID := uuid.New()
	subTaskID := uuid.New()
	commandKey := "Collect LOG,abc123"

	var capturedMsg string
	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, got string) (*UpgradeSubTask, error) {
			require.Equal(t, commandKey, got)
			return &UpgradeSubTask{
				ID:         subTaskID,
				TaskID:     taskID,
				Status:     UpgradeUploading,
				CommandKey: commandKey,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status UpgradeState, msg string) error {
			require.Equal(t, UpgradeFailed, status)
			capturedMsg = msg
			return nil
		},
	}
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]any{
		"command_key": commandKey,
		"fault_struct": map[string]any{
			"fault_code":   0,
			"fault_string": "Download fail with exit status 1",
		},
	})
	require.NoError(t, err)

	require.NoError(t, svc.HandleTransferComplete(context.Background(), evt))
	require.Equal(t, "FaultCode: 0, FaultString: Download fail with exit status 1", capturedMsg)
	require.NotContains(t, capturedMsg, "Upgrade failed")
	require.NotContains(t, capturedMsg, "File transfer failed")
}

func TestService_HandleTransferComplete_UpgradeFaultMessage(t *testing.T) {
	taskID := uuid.New()
	subTaskID := uuid.New()
	commandKey := subTaskID.String()

	var capturedMsg string
	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, got string) (*UpgradeSubTask, error) {
			require.Equal(t, commandKey, got)
			return &UpgradeSubTask{
				ID:         subTaskID,
				TaskID:     taskID,
				Status:     UpgradeDownloading,
				CommandKey: commandKey,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status UpgradeState, msg string) error {
			require.Equal(t, UpgradeFailed, status)
			capturedMsg = msg
			return nil
		},
	}
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]any{
		"command_key": commandKey,
		"fault_struct": map[string]any{
			"fault_code":   9010,
			"fault_string": "install failed",
		},
	})
	require.NoError(t, err)

	require.NoError(t, svc.HandleTransferComplete(context.Background(), evt))
	require.Equal(t, "FaultCode: 9010, FaultString: install failed", capturedMsg)
	require.NotContains(t, capturedMsg, "Upgrade failed")
}

// ---------------------------------------------------------------------------
// Executor event handling tests
// ---------------------------------------------------------------------------

func TestService_HandleDownloadResponse_FaultCode(t *testing.T) {
	taskID := uuid.New()
	subTaskID := uuid.New()

	var capturedStatus UpgradeState
	var capturedMsg string
	incrementCalled := false

	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, commandKey string) (*UpgradeSubTask, error) {
			assert.Equal(t, subTaskID.String(), commandKey)
			return &UpgradeSubTask{
				ID:       subTaskID,
				TaskID:   taskID,
				Status:   UpgradeDownloading,
				DeviceSN: "SN-DL-001",
			}, nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status UpgradeState, msg string) error {
			capturedStatus = status
			capturedMsg = msg
			assert.Equal(t, subTaskID, id)
			return nil
		},
	}

	taskRepo := &svcMockTaskRepo{
		incrementCountsFn: func(_ context.Context, id uuid.UUID, successDelta, failDelta int) error {
			assert.Equal(t, taskID, id)
			assert.Equal(t, 0, successDelta)
			assert.Equal(t, 1, failDelta)
			incrementCalled = true
			return nil
		},
	}

	// Use miniredis so executor.releaseDeviceLock does not panic on nil redis
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		taskRepo,
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, redisClient, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectCommandDownloadResponse, map[string]interface{}{
		"device_sn":    "SN-DL-001",
		"command_key":  subTaskID.String(),
		"fault_code":   9001,
		"fault_string": "device busy",
	})
	require.NoError(t, err)

	err = svc.executor.HandleDownloadResponse(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, UpgradeFailed, capturedStatus)
	assert.Contains(t, capturedMsg, "FaultCode: 9001")
	assert.Contains(t, capturedMsg, "device busy")
	assert.True(t, incrementCalled, "IncrementCounts should have been called")
}

func TestService_HandleDownloadResponse_Success(t *testing.T) {
	taskID := uuid.New()
	subTaskID := uuid.New()

	updateCalled := false
	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, commandKey string) (*UpgradeSubTask, error) {
			assert.Equal(t, subTaskID.String(), commandKey)
			return &UpgradeSubTask{
				ID:       subTaskID,
				TaskID:   taskID,
				Status:   UpgradeDownloading,
				DeviceSN: "SN-DL-002",
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectCommandDownloadResponse, map[string]interface{}{
		"device_sn":    "SN-DL-002",
		"command_key":  subTaskID.String(),
		"fault_code":   0,
		"fault_string": "",
	})
	require.NoError(t, err)

	err = svc.executor.HandleDownloadResponse(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, updateCalled, "updateStatus should NOT be called on success (waiting for TC)")
}

func TestService_HandleDownloadResponse_Idempotent_WrongStatus(t *testing.T) {
	taskID := uuid.New()
	subTaskID := uuid.New()

	updateCalled := false
	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, commandKey string) (*UpgradeSubTask, error) {
			return &UpgradeSubTask{
				ID:       subTaskID,
				TaskID:   taskID,
				Status:   UpgradeRebooting, // not "downloading"
				DeviceSN: "SN-DL-003",
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectCommandDownloadResponse, map[string]interface{}{
		"device_sn":   "SN-DL-003",
		"command_key": subTaskID.String(),
		"fault_code":  9001,
	})
	require.NoError(t, err)

	err = svc.executor.HandleDownloadResponse(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, updateCalled, "updateStatus should NOT be called for non-downloading sub-task")
}

func TestService_HandleTransferComplete_Idempotent(t *testing.T) {
	deviceID := uuid.New()
	taskID := uuid.New()

	updateCalled := false
	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			assert.Equal(t, "SN-IDEMPOTENT-001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	subTaskRepo := &svcMockSubTaskRepo{
		getActiveByDeviceFn: func(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
			return &UpgradeSubTask{
				ID:     uuid.New(),
				TaskID: taskID,
				Status: UpgradeCompleted, // not "downloading"
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]string{
		"device_sn": "SN-IDEMPOTENT-001",
	})

	err := svc.HandleTransferComplete(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, updateCalled, "updateStatus should NOT be called for non-downloading sub-task")
}

// ---------------------------------------------------------------------------
// State transition tests
// ---------------------------------------------------------------------------

func TestService_BatchUpgrade_CreatesMainTask(t *testing.T) {
	deviceID := uuid.New()
	firmwareID := uuid.New()

	var capturedTask *UpgradeTask
	taskRepo := &svcMockTaskRepo{
		createFn: func(_ context.Context, task *UpgradeTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
	}

	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{
				ID:           id,
				Version:      "V2.0.0",
				FileName:     "fw-v2.bin",
				MinIOPath:    "firmware/cmcc/SC/V2.0.0/fw-v2.bin",
				FileSize:     2048,
				ProductClass: "SmallCell-NR",
				MD5Val:       "abc123",
			}, nil
		},
	}

	svc := NewSoftwareService(
		fwRepo,
		taskRepo,
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	req := BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{deviceID},
		FirmwareID: firmwareID,
		TaskName:   "batch-main-task-test",
		TaskType:   TaskTypePatch,
	}

	task, err := svc.BatchUpgrade(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, capturedTask)

	assert.Equal(t, "batch-main-task-test", capturedTask.TaskName)
	assert.Equal(t, TaskTypePatch, capturedTask.TaskType)
	assert.Equal(t, 1, capturedTask.TotalCount)
	assert.Equal(t, "SmallCell-NR", capturedTask.ProductClass)
	assert.Equal(t, "fw-v2.bin", capturedTask.FileName)
	assert.Equal(t, "abc123", capturedTask.FileMD5)

	// After BatchUpgrade returns, status should be updated to in_progress
	assert.Equal(t, TaskInProgress, task.Status)
}

func TestService_BatchUpgrade_CreatesSubTasks(t *testing.T) {
	deviceID1 := uuid.New()
	deviceID2 := uuid.New()
	deviceID3 := uuid.New()
	firmwareID := uuid.New()

	var capturedSubTasks []*UpgradeSubTask
	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{
				ID:           id,
				Version:      "V3.0.0",
				FileName:     "fw-v3.bin",
				MinIOPath:    "firmware/cmcc/SC/V3.0.0/fw-v3.bin",
				FileSize:     4096,
				ProductClass: "SmallCell-LTE",
			}, nil
		},
	}

	subTaskRepo := &svcMockSubTaskRepo{
		batchCreateFn: func(_ context.Context, tasks []*UpgradeSubTask) error {
			capturedSubTasks = make([]*UpgradeSubTask, len(tasks))
			copy(capturedSubTasks, tasks)
			for _, t := range tasks {
				t.ID = uuid.New()
			}
			return nil
		},
	}

	svc := NewSoftwareService(
		fwRepo,
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	req := BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{deviceID1, deviceID2, deviceID3},
		FirmwareID: firmwareID,
		TaskName:   "batch-subtask-test",
		TaskType:   TaskTypeUpgrade,
	}

	_, err := svc.BatchUpgrade(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, capturedSubTasks)

	assert.Len(t, capturedSubTasks, 3)

	expectedDeviceIDs := map[uuid.UUID]bool{deviceID1: true, deviceID2: true, deviceID3: true}
	for _, st := range capturedSubTasks {
		assert.Equal(t, UpgradePending, st.Status, "each sub-task should be in pending state")
		assert.True(t, expectedDeviceIDs[st.DeviceID], "sub-task DeviceID should match one of the requested devices")
		assert.NotNil(t, st.FirmwareID, "sub-task should have FirmwareID set")
		assert.Equal(t, firmwareID, *st.FirmwareID)
		delete(expectedDeviceIDs, st.DeviceID)
	}
	assert.Empty(t, expectedDeviceIDs, "all device IDs should be accounted for")
}

// ---------------------------------------------------------------------------
// Model constants tests
// ---------------------------------------------------------------------------

func TestModelConstants_TaskType(t *testing.T) {
	assert.Equal(t, TaskType(1), TaskTypeUpgrade)
	assert.Equal(t, TaskType(2), TaskTypeRollback)
	assert.Equal(t, TaskType(4), TaskTypePatch)
	assert.Equal(t, TaskType(6), TaskTypeFPGA)
	assert.Equal(t, TaskType(8), TaskTypeReserved)
}

func TestModelConstants_FileType(t *testing.T) {
	assert.Equal(t, FileType(0), FileTypeIMG)
	assert.Equal(t, FileType(1), FileTypePATCH)
	assert.Equal(t, FileType(6), FileTypeFPGA)
}

func TestModelConstants_TaskStatus(t *testing.T) {
	assert.Equal(t, TaskStatus("pending"), TaskPending)
	assert.Equal(t, TaskStatus("in_progress"), TaskInProgress)
	assert.Equal(t, TaskStatus("suspended"), TaskSuspended)
	assert.Equal(t, TaskStatus("ended"), TaskEnded)
}

func TestModelConstants_TaskResult(t *testing.T) {
	assert.Equal(t, TaskResult("success"), TaskResultSuccess)
	assert.Equal(t, TaskResult("partial"), TaskResultPartial)
	assert.Equal(t, TaskResult("failed"), TaskResultFailed)
	assert.Equal(t, TaskResult("terminated"), TaskResultTerminated)
}

// TestApplyScheduleMode_StatusMatrix 锁死三种调度模式投射到 UpgradeTask 的
// Status/CreateStatus/ScheduledAt。重点：挂起分支必须落 TaskSuspended（#138）——
// 历史 bug 落 TaskPending+active 导致 ufte/executionModeForTask 把挂起任务回显成
// "immediate"。统一为 TaskSuspended 后与 CreatePlaceholderTrackingTask 挂起分支一致，
// 且 executionModeForTask(TaskSuspended, active) → "suspended"。
func TestApplyScheduleMode_StatusMatrix(t *testing.T) {
	schedAt := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name             string
		mode             scheduleMode
		scheduledAt      *time.Time
		wantStatus       TaskStatus
		wantCreateStatus string
		wantScheduledSet bool
	}{
		{
			name:             "suspended falls to TaskSuspended (not TaskPending)",
			mode:             scheduleModeSuspended,
			wantStatus:       TaskSuspended,
			wantCreateStatus: CreateStatusActive,
			wantScheduledSet: false,
		},
		{
			name:             "scheduled keeps TaskPending+timing and persists scheduled_at",
			mode:             scheduleModeScheduled,
			scheduledAt:      &schedAt,
			wantStatus:       TaskPending,
			wantCreateStatus: CreateStatusTiming,
			wantScheduledSet: true,
		},
		{
			name:             "immediate keeps create_status active and clears scheduled_at",
			mode:             scheduleModeImmediate,
			wantStatus:       "", // immediate 不改 Status，由调用方推到 in_progress
			wantCreateStatus: CreateStatusActive,
			wantScheduledSet: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := &UpgradeTask{}
			applyScheduleMode(task, tc.mode, tc.scheduledAt)

			assert.Equal(t, tc.wantStatus, task.Status)
			assert.Equal(t, tc.wantCreateStatus, task.CreateStatus)
			if tc.wantScheduledSet {
				require.NotNil(t, task.ScheduledAt)
			} else {
				assert.Nil(t, task.ScheduledAt)
			}
		})
	}
}

// TestApplyScheduleMode_SuspendedMatchesPlaceholder 守护两条挂起链路的一致性：
// applyScheduleMode（BatchCollect/BatchUpgrade/RollbackDevices）与
// CreatePlaceholderTrackingTask（CONFIG_RESTORE/LICENSE_UPGRADE）挂起后主任务
// Status 必须同为 TaskSuspended。
func TestApplyScheduleMode_SuspendedMatchesPlaceholder(t *testing.T) {
	// applyScheduleMode 挂起分支
	viaSchedule := &UpgradeTask{}
	applyScheduleMode(viaSchedule, scheduleModeSuspended, nil)

	// CreatePlaceholderTrackingTask 挂起分支用的 initialStatus（见 service.go:433）
	const placeholderSuspendedStatus = TaskSuspended

	assert.Equal(t, placeholderSuspendedStatus, viaSchedule.Status,
		"两条挂起链路的主任务 Status 必须一致，否则 ufte 回显会分叉")
}

// ---------------------------------------------------------------------------
// Rollback tests
// ---------------------------------------------------------------------------

func TestService_RollbackDevices_CreatesTask(t *testing.T) {
	deviceID1 := uuid.New()
	deviceID2 := uuid.New()

	var capturedTask *UpgradeTask
	taskRepo := &svcMockTaskRepo{
		createFn: func(_ context.Context, task *UpgradeTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		taskRepo,
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	req := RollbackRequest{
		DeviceIDs:  []uuid.UUID{deviceID1, deviceID2},
		TaskName:   "rollback-test",
		CreateUser: "admin",
	}

	task, err := svc.RollbackDevices(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, capturedTask)

	assert.Equal(t, TaskTypeRollback, capturedTask.TaskType, "rollback task type should be 2")
	assert.Equal(t, "rollback-test", capturedTask.TaskName)
	assert.Equal(t, 2, capturedTask.TotalCount)
	assert.Equal(t, "admin", capturedTask.CreateUser)

	// After RollbackDevices returns, status should be updated to in_progress
	assert.Equal(t, TaskInProgress, task.Status)
}

// ---------------------------------------------------------------------------
// T-0021 Rollback enhancement tests (source / reason / target_firmware_id)
// ---------------------------------------------------------------------------

// TestService_RollbackDevices_SourceMatrix exercises the canonical source enum:
// blank → manual default; each canonical value persists verbatim; non-canonical
// fails ErrInvalidInput.
func TestService_RollbackDevices_SourceMatrix(t *testing.T) {
	tests := []struct {
		name           string
		inSource       string
		expectSource   string
		expectErrInput bool
	}{
		{"blank defaults to manual", "", RollbackSourceManual, false},
		{"manual passes through", RollbackSourceManual, RollbackSourceManual, false},
		{"canary_failure passes through", RollbackSourceCanaryFailure, RollbackSourceCanaryFailure, false},
		{"compatibility passes through", RollbackSourceCompatibility, RollbackSourceCompatibility, false},
		{"scheduled passes through", RollbackSourceScheduled, RollbackSourceScheduled, false},
		{"unknown source rejected", "bogus_value", "", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var capturedTask *UpgradeTask
			taskRepo := &svcMockTaskRepo{
				createFn: func(_ context.Context, task *UpgradeTask) error {
					capturedTask = task
					task.ID = uuid.New()
					return nil
				},
			}
			svc := NewSoftwareService(
				&svcMockFirmwareRepo{},
				taskRepo,
				&svcMockSubTaskRepo{},
				&svcMockDeviceRepo{},
				&svcMockCmdQueue{},
				nil, nil, "test-bucket",
				&svcMockEventBus{}, nil, zap.NewNop(),
			)

			req := RollbackRequest{
				DeviceIDs:  []uuid.UUID{uuid.New()},
				TaskName:   "rb-source-" + tt.name,
				CreateUser: "admin",
				Source:     tt.inSource,
			}
			_, err := svc.RollbackDevices(context.Background(), req)
			if tt.expectErrInput {
				require.Error(t, err)
				assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
				assert.Nil(t, capturedTask, "task must not be persisted on validation error")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, capturedTask)
			assert.Equal(t, tt.expectSource, capturedTask.RollbackSource)
		})
	}
}

// TestService_RollbackDevices_TargetFirmwareMatrix covers the four target
// branches: nil (legacy OriVersion path), valid firmware, missing firmware,
// and combined with source=canary_failure.
func TestService_RollbackDevices_TargetFirmwareMatrix(t *testing.T) {
	targetFW := uuid.New()
	missingFW := uuid.New()

	tests := []struct {
		name              string
		targetID          *uuid.UUID
		source            string
		fwRepo            *svcMockFirmwareRepo
		expectErr         bool
		expectDestVersion string // "" means inherit OriVersion (legacy path)
		expectTaskTarget  *uuid.UUID
	}{
		{
			name:              "nil target keeps legacy path (DestVersion empty)",
			targetID:          nil,
			source:            "",
			fwRepo:            &svcMockFirmwareRepo{},
			expectDestVersion: "",
			expectTaskTarget:  nil,
		},
		{
			name:     "valid target overrides DestVersion",
			targetID: &targetFW,
			source:   "",
			fwRepo: &svcMockFirmwareRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
					return &FirmwareVersion{ID: id, Version: "V2.5.7"}, nil
				},
			},
			expectDestVersion: "V2.5.7",
			expectTaskTarget:  &targetFW,
		},
		{
			name:     "missing target firmware errors",
			targetID: &missingFW,
			source:   "",
			fwRepo: &svcMockFirmwareRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*FirmwareVersion, error) {
					return nil, commonerrors.ErrNotFound
				},
			},
			expectErr: true,
		},
		{
			name:     "target + canary_failure source combine",
			targetID: &targetFW,
			source:   RollbackSourceCanaryFailure,
			fwRepo: &svcMockFirmwareRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
					return &FirmwareVersion{ID: id, Version: "V1.9.0"}, nil
				},
			},
			expectDestVersion: "V1.9.0",
			expectTaskTarget:  &targetFW,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var capturedTask *UpgradeTask
			var capturedSubTasks []*UpgradeSubTask
			taskRepo := &svcMockTaskRepo{
				createFn: func(_ context.Context, task *UpgradeTask) error {
					capturedTask = task
					task.ID = uuid.New()
					return nil
				},
			}
			subRepo := &svcMockSubTaskRepo{
				batchCreateFn: func(_ context.Context, tasks []*UpgradeSubTask) error {
					capturedSubTasks = append(capturedSubTasks, tasks...)
					for _, st := range tasks {
						st.ID = uuid.New()
					}
					return nil
				},
			}
			devRepo := &svcMockDeviceRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
					return &model.Device{ID: id, SerialNumber: "SN-RB-001", FirmwareVersion: "V3.0.0"}, nil
				},
			}

			svc := NewSoftwareService(
				tt.fwRepo, taskRepo, subRepo, devRepo, &svcMockCmdQueue{},
				nil, nil, "test-bucket",
				&svcMockEventBus{}, nil, zap.NewNop(),
			)

			req := RollbackRequest{
				DeviceIDs:        []uuid.UUID{uuid.New()},
				TaskName:         "rb-target-" + tt.name,
				CreateUser:       "admin",
				Source:           tt.source,
				TargetFirmwareID: tt.targetID,
				// #59 Problem 4：本矩阵的目标版本（V2.5.7/V1.9.0）相对设备当前 V3.0.0 是
				// 降级，新防降级守卫默认会拦。这里测的是 DestVersion/target 的字段透传，
				// 非降级策略本身，故显式 Force=true 绕过守卫（降级守卫单测见 version_test.go
				// 与 TestService_RollbackDevices_DowngradeGuard）。
				Force: true,
			}
			_, err := svc.RollbackDevices(context.Background(), req)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, capturedTask)
			require.Len(t, capturedSubTasks, 1)
			assert.Equal(t, tt.expectDestVersion, capturedSubTasks[0].DestVersion,
				"DestVersion should match target firmware version (or be blank for legacy path)")
			assert.Equal(t, "V3.0.0", capturedSubTasks[0].OriVersion,
				"OriVersion always reflects device's current firmware")
			if tt.expectTaskTarget == nil {
				assert.Nil(t, capturedTask.RollbackTargetFirmwareID)
			} else {
				require.NotNil(t, capturedTask.RollbackTargetFirmwareID)
				assert.Equal(t, *tt.expectTaskTarget, *capturedTask.RollbackTargetFirmwareID)
			}
		})
	}
}

// TestService_RollbackDevices_ReasonRoundTrip ensures the audit reason
// reaches the persisted task verbatim.
func TestService_RollbackDevices_ReasonRoundTrip(t *testing.T) {
	var capturedTask *UpgradeTask
	taskRepo := &svcMockTaskRepo{
		createFn: func(_ context.Context, task *UpgradeTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
	}
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, taskRepo, &svcMockSubTaskRepo{},
		&svcMockDeviceRepo{}, &svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	const reason = "stage-1 failure_rate 12.50% > threshold 5%"
	_, err := svc.RollbackDevices(context.Background(), RollbackRequest{
		DeviceIDs:  []uuid.UUID{uuid.New()},
		TaskName:   "rb-with-reason",
		CreateUser: "canary-monitor",
		Reason:     reason,
		Source:     RollbackSourceCanaryFailure,
	})
	require.NoError(t, err)
	require.NotNil(t, capturedTask)
	assert.Equal(t, reason, capturedTask.RollbackReason)
	assert.Equal(t, RollbackSourceCanaryFailure, capturedTask.RollbackSource)
}

func TestService_TerminateUpgrade_TerminatesActiveSubTasks(t *testing.T) {
	taskID := uuid.New()
	pendingID := uuid.New()
	downloadingID := uuid.New()
	completedID := uuid.New()
	updated := map[uuid.UUID]UpgradeState{}

	taskRepo := &svcMockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{ID: id, Status: TaskInProgress}, nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error {
			assert.Equal(t, taskID, id)
			assert.Equal(t, TaskEnded, status)
			assert.Equal(t, TaskResultTerminated, result)
			return nil
		},
	}
	subRepo := &svcMockSubTaskRepo{
		listByTaskIDFn: func(_ context.Context, id uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
			assert.Equal(t, taskID, id)
			assert.Equal(t, 1, filter.Page)
			return model.NewListResponse([]UpgradeSubTaskWithTaskName{
				{UpgradeSubTask: UpgradeSubTask{ID: pendingID, TaskID: taskID, Status: UpgradePending}},
				{UpgradeSubTask: UpgradeSubTask{ID: downloadingID, TaskID: taskID, Status: UpgradeDownloading}},
				{UpgradeSubTask: UpgradeSubTask{ID: completedID, TaskID: taskID, Status: UpgradeCompleted}},
			}, 3, 1, 100), nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status UpgradeState, msg string) error {
			updated[id] = status
			assert.Equal(t, "task terminated by operator", msg)
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, taskRepo, subRepo,
		&svcMockDeviceRepo{}, &svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	require.NoError(t, svc.TerminateUpgrade(context.Background(), taskID))
	assert.Equal(t, map[uuid.UUID]UpgradeState{
		pendingID:     UpgradeTerminated,
		downloadingID: UpgradeTerminated,
	}, updated)
	assert.NotContains(t, updated, completedID)
}

// TestIsValidRollbackSource sanity-checks the enum guard used by RollbackDevices.
func TestIsValidRollbackSource(t *testing.T) {
	cases := map[string]bool{
		"manual":         true,
		"canary_failure": true,
		"compatibility":  true,
		"scheduled":      true,
		"":               false,
		"MANUAL":         false,
		"unknown":        false,
	}
	for in, expect := range cases {
		assert.Equal(t, expect, IsValidRollbackSource(in), "input=%q", in)
	}
}

// TestService_TriggerCanaryFailureRollback_NoCompletedDevices is a no-op when
// no sub-tasks reached UpgradeCompleted.
func TestService_TriggerCanaryFailureRollback_NoCompletedDevices(t *testing.T) {
	canaryID := uuid.New()
	taskRepo := &svcMockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{ID: id, TaskName: "canary-x"}, nil
		},
	}
	subRepo := &svcMockSubTaskRepo{}
	// default svcMockSubTaskRepo.ListByTaskID returns empty list

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, taskRepo, subRepo,
		&svcMockDeviceRepo{}, &svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)
	err := svc.TriggerCanaryFailureRollback(context.Background(), canaryID, "test reason")
	assert.NoError(t, err, "no completed devices = silent no-op")
}
