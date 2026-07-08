package backup

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks (prefixed with exec to avoid collision with service_test.go)
// ---------------------------------------------------------------------------

type execTaskRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*BackupTask, error)
	updateFn  func(ctx context.Context, task *BackupTask) error
}

func (m *execTaskRepo) Create(_ context.Context, _ *BackupTask) error { return nil }
func (m *execTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *execTaskRepo) Update(ctx context.Context, task *BackupTask) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, task)
	}
	return nil
}
func (m *execTaskRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *execTaskRepo) List(_ context.Context, _ TaskFilter) (*model.ListResponse[BackupTask], error) {
	return nil, nil
}
func (m *execTaskRepo) UpdateFilePath(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *execTaskRepo) FindByIDPrefix(_ context.Context, _ string, _ int) ([]*BackupTask, error) {
	return nil, nil
}
func (m *execTaskRepo) CleanupOldRows(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
	return nil, 0, nil
}
func (m *execTaskRepo) MarkComplete(_ context.Context, _ uuid.UUID, _ TaskStatus, _ int16, _ time.Time, _ string) error {
	return nil
}

type execDeviceRepo struct {
	getBySNFn func(ctx context.Context, sn string) (*model.Device, error)
}

func (m *execDeviceRepo) Create(_ context.Context, _ *model.Device) error { return nil }
func (m *execDeviceRepo) Update(_ context.Context, _ *model.Device) error { return nil }
func (m *execDeviceRepo) Delete(_ context.Context, _ uuid.UUID) error     { return nil }
func (m *execDeviceRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Device, error) {
	return nil, nil
}
func (m *execDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.getBySNFn != nil {
		return m.getBySNFn(ctx, sn)
	}
	return nil, nil
}
func (m *execDeviceRepo) GetDeletedBySerialNumber(_ context.Context, _ string, _ model.CarrierCode) (*model.Device, error) {
	return nil, nil
}
func (m *execDeviceRepo) List(_ context.Context, _ device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return nil, nil
}
func (m *execDeviceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.DeviceStatus) error {
	return nil
}

// T-0162 新接口方法
func (m *execDeviceRepo) UpdateLifecycle(_ context.Context, _ uuid.UUID, _ model.DeviceLifecycle) error {
	return nil
}

func (m *execDeviceRepo) UpdateOnlineStatus(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *execDeviceRepo) UpdateLastInform(_ context.Context, _ string, _ time.Time, _ []string) error {
	return nil
}
func (m *execDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *execDeviceRepo) CountByStatus(_ context.Context, _ *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return nil, nil
}
func (m *execDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *execDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *execDeviceRepo) GetGeoStats(_ context.Context, _ device.GeoStatsFilter) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *execDeviceRepo) SearchDevices(_ context.Context, _ string, _ int, _ []model.DeviceVisibilityGrant) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *execDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *execDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *execDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *execDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (m *execDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}
func (m *execDeviceRepo) UpdateSiteName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *execDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *execDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[device.DeviceWithInfo], error) {
	return model.NewListResponse([]device.DeviceWithInfo{}, 0, 1, 20), nil
}
func (m *execDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (*device.RestoreResult, error) {
	return &device.RestoreResult{}, nil
}
func (m *execDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *execDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

// execCmdQueue is the test stand-in for task.Enqueuer: it captures
// CreateTask calls so assertions can inspect what was enqueued.
type execCmdQueue struct {
	pushed []struct {
		DeviceSN string
		Req      *devtask.CreateTaskRequest
	}
}

func (m *execCmdQueue) CreateTask(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
	m.pushed = append(m.pushed, struct {
		DeviceSN string
		Req      *devtask.CreateTaskRequest
	}{req.DeviceSN, req})
	return devtask.NewTask(req), nil
}

func (m *execCmdQueue) GetQueueLength(_ context.Context, _ string) (int64, error) {
	return int64(len(m.pushed)), nil
}

// execTransferProvider 是测试用的 transfercfg.Provider mock。
type execTransferProvider struct {
	upload transfercfg.UploadSettings
}

func (m *execTransferProvider) Snapshot(_ context.Context) transfercfg.Snapshot {
	return transfercfg.Snapshot{Upload: m.upload}
}

type execPolicyGetter struct {
	policy *BackupPolicy
	err    error
}

func (m *execPolicyGetter) Get(_ context.Context) (*BackupPolicy, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.policy, nil
}

type execFTPConfigRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*FTPConfig, error)
}

func (m *execFTPConfigRepo) Create(_ context.Context, _ *FTPConfig) error { return nil }
func (m *execFTPConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*FTPConfig, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *execFTPConfigRepo) Update(_ context.Context, _ *FTPConfig) error { return nil }
func (m *execFTPConfigRepo) Delete(_ context.Context, _ uuid.UUID) error  { return nil }
func (m *execFTPConfigRepo) List(_ context.Context, _ FTPConfigFilter) (*model.ListResponse[FTPConfig], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func newTestExecutor(taskRepo *execTaskRepo, deviceRepo *execDeviceRepo, cmdQ *execCmdQueue) *BackupExecutor {
	return &BackupExecutor{
		taskRepo:   taskRepo,
		deviceRepo: deviceRepo,
		taskSvc:    cmdQ,
		connReq:    nil,
		eventBus:   event.NewChannelEventBus(16, zap.NewNop()),
		logger:     zap.NewNop(),
	}
}

func TestHandleTask_PendingToRunning(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN001"},
	}

	var statusUpdates []TaskStatus
	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*BackupTask, error) {
			assert.Equal(t, taskID, id)
			return task, nil
		},
		updateFn: func(_ context.Context, t *BackupTask) error {
			statusUpdates = append(statusUpdates, t.Status)
			return nil
		},
	}

	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn}, nil
		},
	}

	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{
		ID:        uuid.New().String(),
		Subject:   event.SubjectBackupTaskCreated,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	// First update: running, last update: completed
	require.GreaterOrEqual(t, len(statusUpdates), 2)
	assert.Equal(t, TaskRunning, statusUpdates[0])
	assert.Equal(t, TaskCompleted, statusUpdates[len(statusUpdates)-1])
	assert.NotNil(t, task.StartedAt)
}

func TestHandleTask_PushUploadCommand(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN001", "SN002"},
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn:  func(_ context.Context, _ *BackupTask) error { return nil },
	}
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn, OUI: "0000B9", ProductClass: "FAP/BLQ/SC"}, nil
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)
	executor.SetTransferProvider(&execTransferProvider{
		upload: transfercfg.UploadSettings{
			BaseURL: "http://acs:7557",
			Path:    "/smallcell/FileUploadService",
		},
	})

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	// Should have pushed 2 Upload commands
	require.Len(t, cmdQ.pushed, 2)
	assert.Equal(t, "SN001", cmdQ.pushed[0].DeviceSN)
	assert.Equal(t, "Upload", cmdQ.pushed[0].Req.Method)
	assert.Equal(t, "SN002", cmdQ.pushed[1].DeviceSN)
	assert.Equal(t, "Upload", cmdQ.pushed[1].Req.Method)

	// Verify file_type：标准平台（非 NV）应为 "10 {OUI} Configuration File"
	var params map[string]interface{}
	_ = json.Unmarshal(cmdQ.pushed[0].Req.Params, &params)
	assert.Equal(t, "10 0000B9 Configuration File", params["file_type"])

	// url 字段应包含 CONFIGBACKUP_XML 和设备 SN
	assert.Contains(t, params["url"], "CONFIGBACKUP_XML")
	assert.Contains(t, params["url"], "SN001")

	// CommandKey + SourceID linkage for TransferComplete correlation.
	// M2: CommandKey 格式含 _BACKUP_ 标记
	assert.Equal(t, task.ID.String(), cmdQ.pushed[0].Req.SourceID)
	assert.Contains(t, cmdQ.pushed[0].Req.CommandKey, "_BACKUP_")
}

func TestHandleTask_NVPlatform(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN_MLQ"},
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn:  func(_ context.Context, _ *BackupTask) error { return nil },
	}
	// MLQ 平台：ProductClass 以 "FAP/MLQ" 开头，应使用 NV 备份
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn, OUI: "0000B9", ProductClass: "FAP/MLQ/SC"}, nil
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)
	executor.SetTransferProvider(&execTransferProvider{
		upload: transfercfg.UploadSettings{
			BaseURL: "http://acs:7557",
			Path:    "/smallcell/FileUploadService",
		},
	})

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	require.Len(t, cmdQ.pushed, 1)
	var params map[string]interface{}
	_ = json.Unmarshal(cmdQ.pushed[0].Req.Params, &params)
	// NV 平台：FileType 应为 "12 {OUI} Configuration File"
	assert.Equal(t, "12 0000B9 Configuration File", params["file_type"])
	// URL 应包含 CONFIGBACKUP_NV
	assert.Contains(t, params["url"], "CONFIGBACKUP_NV")
}

// qa-614 #376: selectBackupType 前缀契约固化。
// 修复前 BM(FAP/BU1810) 永远兜底 XML、MLN 全系因 "FAP/MLN_SC"(下划线) 不命中而误落 XML。
func TestSelectBackupType_PrefixContract(t *testing.T) {
	tests := []struct {
		name         string
		productClass string
		wantType     string
		wantExt      string
		wantURLParam string
	}{
		// NV 平台
		{"MLQ/SC → NV", "FAP/MLQ/SC", "CONFIG_BACKUP_NV", ".nv", "CONFIGBACKUP_NV"},
		{"MLN/SC → NV（修复下划线漂移）", "FAP/MLN/SC", "CONFIG_BACKUP_NV", ".nv", "CONFIGBACKUP_NV"},
		{"MLN/CA → NV", "FAP/MLN/CA", "CONFIG_BACKUP_NV", ".nv", "CONFIGBACKUP_NV"},
		{"MLN/DC → NV", "FAP/MLN/DC", "CONFIG_BACKUP_NV", ".nv", "CONFIGBACKUP_NV"},
		{"BM/BU1810 → NV（新增登记）", "FAP/BU1810", "CONFIG_BACKUP_NV", ".nv", "CONFIGBACKUP_NV"},
		// XML 兜底平台（不回归）
		{"BLQ/SC → XML 兜底", "FAP/BLQ/SC", "CONFIG_BACKUP_XML", ".xml", "CONFIGBACKUP_XML"},
		{"QLS → XML 兜底", "FAP/QLS/SC", "CONFIG_BACKUP_XML", ".xml", "CONFIGBACKUP_XML"},
		{"BSC → XML 兜底", "FAP/BSC", "CONFIG_BACKUP_XML", ".xml", "CONFIGBACKUP_XML"},
		{"空 ProductClass → XML 兜底", "", "CONFIG_BACKUP_XML", ".xml", "CONFIGBACKUP_XML"},
		// 边界：曾错写的下划线串本身不应再命中 NV（确保前缀契约就是斜杠）
		{"FAP/MLN_SC（旧错误串）不再命中 NV", "FAP/MLN_SC", "CONFIG_BACKUP_XML", ".xml", "CONFIGBACKUP_XML"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := selectBackupType(tt.productClass)
			require.NotNil(t, spec)
			assert.Equal(t, tt.wantType, spec.TypeCode)
			assert.Equal(t, tt.wantExt, spec.FileExtension)
			assert.Equal(t, tt.wantURLParam, spec.URLFileTypeParam)
		})
	}
}

// qa-614 #376: BM 站(FAP/BU1810) 端到端应下发 NV 备份。
func TestHandleTask_BMPlatformNV(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN_BM"},
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn:  func(_ context.Context, _ *BackupTask) error { return nil },
	}
	// BM 站：真机 ProductClass=FAP/BU1810，应使用 NV 备份。
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn, OUI: "0000B9", ProductClass: "FAP/BU1810"}, nil
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)
	executor.SetTransferProvider(&execTransferProvider{
		upload: transfercfg.UploadSettings{
			BaseURL: "http://acs:7557",
			Path:    "/smallcell/FileUploadService",
		},
	})

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	require.Len(t, cmdQ.pushed, 1)
	var params map[string]interface{}
	_ = json.Unmarshal(cmdQ.pushed[0].Req.Params, &params)
	assert.Equal(t, "12 0000B9 Configuration File", params["file_type"])
	assert.Contains(t, params["url"], "CONFIGBACKUP_NV")
	// 目标文件名应以 .nv 结尾。
	assert.Contains(t, params["url"], ".nv")
}

// qa-614 #376: MLN/SC 端到端应下发 NV 备份（修复下划线 vs 斜杠潜伏 bug）。
func TestHandleTask_MLNPlatformNV(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN_MLN"},
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn:  func(_ context.Context, _ *BackupTask) error { return nil },
	}
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn, OUI: "0000B9", ProductClass: "FAP/MLN/SC"}, nil
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)
	executor.SetTransferProvider(&execTransferProvider{
		upload: transfercfg.UploadSettings{
			BaseURL: "http://acs:7557",
			Path:    "/smallcell/FileUploadService",
		},
	})

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	require.Len(t, cmdQ.pushed, 1)
	var params map[string]interface{}
	_ = json.Unmarshal(cmdQ.pushed[0].Req.Params, &params)
	assert.Equal(t, "12 0000B9 Configuration File", params["file_type"])
	assert.Contains(t, params["url"], "CONFIGBACKUP_NV")
	assert.Contains(t, params["url"], ".nv")
}

func TestHandleTask_RemoteFTPPolicyUsesRemoteURL(t *testing.T) {
	taskID := uuid.New()
	ftpID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN_REMOTE"},
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn:  func(_ context.Context, _ *BackupTask) error { return nil },
	}
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn, OUI: "0000B9", ProductClass: "FAP/BLQ/SC"}, nil
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)
	executor.SetPolicyEnforcement(&execPolicyGetter{policy: &BackupPolicy{
		StorageBackend: "ftp",
		FTPConfigID:    &ftpID,
	}}, nil)
	executor.SetFTPConfigRepository(&execFTPConfigRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FTPConfig, error) {
			require.Equal(t, ftpID, id)
			password := "ftp-pass"
			return &FTPConfig{
				ID:                id,
				Host:              "10.0.0.8",
				Port:              21,
				Username:          "ftp-user",
				PasswordEncrypted: &password,
				Protocol:          "FTP",
				RemotePath:        "/backup/omc",
				Enabled:           true,
			}, nil
		},
	})

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)
	require.Len(t, cmdQ.pushed, 1)

	var params map[string]interface{}
	require.NoError(t, json.Unmarshal(cmdQ.pushed[0].Req.Params, &params))
	assert.Equal(t, "ftp-user", params["username"])
	assert.Equal(t, "ftp-pass", params["password"])
	assert.Contains(t, params["url"], "ftp://10.0.0.8:21/backup/omc/")
	assert.NotContains(t, params["url"], "smallcell/FileUploadService")
}

func TestHandleTask_ProgressUpdate(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN001", "SN002", "SN003", "SN004"},
	}

	var progressValues []int
	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn: func(_ context.Context, t *BackupTask) error {
			progressValues = append(progressValues, t.Progress)
			return nil
		},
	}
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn}, nil
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	// Final progress should be 100
	assert.Equal(t, 100, progressValues[len(progressValues)-1])
}

func TestHandleTask_DeviceNotFound(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"MISSING_SN"},
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn:  func(_ context.Context, _ *BackupTask) error { return nil },
	}
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, assert.AnError
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	// No commands pushed
	assert.Len(t, cmdQ.pushed, 0)
	// Task should be failed
	assert.Equal(t, TaskFailed, task.Status)
	assert.NotNil(t, task.ErrorMessage)
}

func TestHandleTask_SkipsNonPending(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:     taskID,
		Status: TaskCompleted,
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, &execDeviceRepo{}, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	assert.Len(t, cmdQ.pushed, 0)
}
