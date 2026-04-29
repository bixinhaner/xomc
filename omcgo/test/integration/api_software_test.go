package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Software Module E2E Tests — Sprint 6 (Upgrade & Rollback Enhancement)
// ---------------------------------------------------------------------------

// TestFirmwareVersionJSONFields verifies the FirmwareVersion JSON structure
// includes all new fields expected by frontend softwareApi.ts.
func TestFirmwareVersionJSONFields(t *testing.T) {
	fw := software.FirmwareVersion{
		ID:            uuid.New(),
		ProductClass:  "SmallCell-LTE",
		Version:       "V1.0.0",
		FileName:      "fw.bin",
		FileSize:      1024000,
		FileType:      software.FileTypeIMG,
		MinIOPath:     "firmware/img/cmcc/SmallCell-LTE/V1.0.0/fw.bin",
		CompatibleOUI: []string{"001A2B"},
		MD5Val:        "a1b2c3d4e5f6",
		Recommend:     true,
		Uploader:      "admin",
		Manufacturer:  "BaiCells",
		ReleaseNotes:  "Bug fixes",
		Description:   "Main firmware image",
		Status:        "active",
		CreatedAt:     software.JSONTime(time.Now()),
		UpdatedAt:     software.JSONTime(time.Now()),
	}

	data, err := json.Marshal(fw)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	requiredFields := []string{
		"id", "product_class", "version", "file_name", "file_size",
		"file_type", "minio_path", "compatible_oui", "md5_val", "recommend",
		"uploader", "manufacturer", "release_notes", "description", "status",
		"created_at", "updated_at",
	}
	for _, field := range requiredFields {
		assert.Contains(t, result, field, "FirmwareVersion must have field: %s", field)
	}

	// Verify new fields have correct values
	assert.Equal(t, float64(0), result["file_type"], "file_type IMG = 0")
	assert.Equal(t, true, result["recommend"])
	assert.Equal(t, "admin", result["uploader"])
	assert.Equal(t, "BaiCells", result["manufacturer"])
}

// TestFirmwareVersionFileTypeValues verifies file_type constants.
func TestFirmwareVersionFileTypeValues(t *testing.T) {
	assert.Equal(t, software.FileType(0), software.FileTypeIMG)
	assert.Equal(t, software.FileType(1), software.FileTypePATCH)
	assert.Equal(t, software.FileType(6), software.FileTypeFPGA)
}

// TestUpgradeTaskJSONFields verifies the main UpgradeTask JSON structure
// matches frontend BackendUpgradeTask expectations.
func TestUpgradeTaskJSONFields(t *testing.T) {
	task := software.UpgradeTask{
		ID:            uuid.New(),
		TaskName:      "Batch upgrade test",
		TaskType:      software.TaskTypeUpgrade,
		Status:        software.TaskInProgress,
		Result:        "",
		ProductClass:  "SmallCell-LTE",
		IsKeepConfig:  true,
		CreateStatus:  "active",
		CreateUser:    "admin",
		TotalCount:    10,
		SuccessCount:  3,
		FailCount:     1,
		MaxConcurrent: 5,
		CreatedAt:     software.JSONTime(time.Now()),
		UpdatedAt:     software.JSONTime(time.Now()),
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	requiredFields := []string{
		"id", "task_name", "task_type", "status",
		"product_class", "is_keep_config", "create_status", "create_user",
		"total_count", "success_count", "fail_count", "max_concurrent",
		"created_at", "updated_at",
	}
	for _, field := range requiredFields {
		assert.Contains(t, result, field, "UpgradeTask must have field: %s", field)
	}

	assert.Equal(t, float64(1), result["task_type"])
	assert.Equal(t, "in_progress", result["status"])
	assert.Equal(t, float64(10), result["total_count"])
	assert.Equal(t, float64(3), result["success_count"])
}

// TestUpgradeSubTaskJSONFields verifies the UpgradeSubTask JSON structure
// matches frontend BackendUpgradeSubTask expectations.
func TestUpgradeSubTaskJSONFields(t *testing.T) {
	taskID := uuid.New()
	fwID := uuid.New()
	subTask := software.UpgradeSubTask{
		ID:              uuid.New(),
		TaskID:          taskID,
		DeviceID:        uuid.New(),
		FirmwareID:      &fwID,
		Status:          software.UpgradeDownloading,
		DeviceSN:        "ENB00001",
		OriVersion:      "V1.0.0",
		DestVersion:     "V1.1.0",
		CommandKey:      "cmd-123",
		FailureReason:  "none",
		PreSuspendStatus: "downloading",
		RetryCount:      0,
		MaxRetries:      3,
		CreatedAt:       software.JSONTime(time.Now()),
		UpdatedAt:       software.JSONTime(time.Now()),
	}

	data, err := json.Marshal(subTask)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	requiredFields := []string{
		"id", "task_id", "device_id", "firmware_id", "status",
		"device_sn", "ori_version", "dest_version", "command_key",
		"failure_reason", "pre_suspend_status", "retry_count", "max_retries",
		"created_at", "updated_at",
	}
	for _, field := range requiredFields {
		assert.Contains(t, result, field, "UpgradeSubTask must have field: %s", field)
	}

	assert.Equal(t, "downloading", result["status"])
	assert.Equal(t, "ENB00001", result["device_sn"])
	assert.Equal(t, "V1.0.0", result["ori_version"])
	assert.Equal(t, "V1.1.0", result["dest_version"])
}

// TestBatchUpgradeRequestJSON verifies the BatchUpgradeRequest JSON binding.
func TestBatchUpgradeRequestJSON(t *testing.T) {
	body := `{
		"device_ids": ["00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"],
		"firmware_id": "00000000-0000-0000-0000-000000000003",
		"task_name": "Test batch upgrade",
		"task_type": 1,
		"is_keep_config": true,
		"concurrency": 5
	}`

	var req software.BatchUpgradeRequest
	err := json.Unmarshal([]byte(body), &req)
	require.NoError(t, err)

	assert.Equal(t, 2, len(req.DeviceIDs))
	assert.Equal(t, "Test batch upgrade", req.TaskName)
	assert.Equal(t, software.TaskType(1), req.TaskType)
	assert.True(t, req.IsKeepConfig)
}

// TestRollbackRequestJSON verifies the RollbackRequest JSON binding.
func TestRollbackRequestJSON(t *testing.T) {
	body := `{
		"device_ids": ["00000000-0000-0000-0000-000000000001"],
		"task_name": "Rollback test",
		"create_user": "admin"
	}`

	var req software.RollbackRequest
	err := json.Unmarshal([]byte(body), &req)
	require.NoError(t, err)

	assert.Equal(t, 1, len(req.DeviceIDs))
	assert.Equal(t, "Rollback test", req.TaskName)
	assert.Equal(t, "admin", req.CreateUser)
}

// TestHandler_UpgradeTaskEndpoints verifies all upgrade task route registrations.
func TestHandler_UpgradeTaskEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Verify routes are registered by creating a handler and registering routes
	taskRepo := &swHTaskRepoStub{}
	subTaskRepo := &swHSubTaskRepoStub{}
	fwRepo := &swHFirmwareRepoStub{}

	h := software.NewHandler(nil, fwRepo, taskRepo, subTaskRepo, zap.NewNop())
	r := gin.New()
	h.RegisterRoutes(r.Group(""))

	routes := r.Routes()
	routeMap := make(map[string]string)
	for _, route := range routes {
		routeMap[route.Method+" "+route.Path] = route.Path
	}

	expectedRoutes := []string{
		"GET /firmware",
		"POST /firmware",
		"GET /firmware/:id",
		"DELETE /firmware/:id",
		"PUT /firmware/:id/recommend",
		"GET /upgrade-tasks",
		"GET /upgrade-tasks/:id",
		"POST /upgrade-tasks",
		"PUT /upgrade-tasks/:id/suspend",
		"PUT /upgrade-tasks/:id/resume",
		"PUT /upgrade-tasks/:id/terminate",
		"POST /upgrade-tasks/:id/retry",
		"POST /upgrade-tasks/rollback",
		"GET /upgrade-tasks/:id/tasks",
		"GET /upgrade-sub-tasks/:id",
	}

	for _, expected := range expectedRoutes {
		assert.Contains(t, routeMap, expected, "Missing route: %s", expected)
	}
}

// TestUpgradeTaskStatusConstants verifies status and result constants match frontend expectations.
func TestUpgradeTaskStatusConstants(t *testing.T) {
	// Main task status
	assert.Equal(t, software.TaskStatus("pending"), software.TaskPending)
	assert.Equal(t, software.TaskStatus("in_progress"), software.TaskInProgress)
	assert.Equal(t, software.TaskStatus("suspended"), software.TaskSuspended)
	assert.Equal(t, software.TaskStatus("ended"), software.TaskEnded)

	// Main task result
	assert.Equal(t, software.TaskResult("success"), software.TaskResultSuccess)
	assert.Equal(t, software.TaskResult("partial"), software.TaskResultPartial)
	assert.Equal(t, software.TaskResult("failed"), software.TaskResultFailed)
	assert.Equal(t, software.TaskResult("terminated"), software.TaskResultTerminated)

	// Sub-task status
	assert.Equal(t, software.UpgradeState("pending"), software.UpgradePending)
	assert.Equal(t, software.UpgradeState("downloading"), software.UpgradeDownloading)
	assert.Equal(t, software.UpgradeState("rebooting"), software.UpgradeRebooting)
	assert.Equal(t, software.UpgradeState("verifying"), software.UpgradeVerifying)
	assert.Equal(t, software.UpgradeState("completed"), software.UpgradeCompleted)
	assert.Equal(t, software.UpgradeState("failed"), software.UpgradeFailed)
	assert.Equal(t, software.UpgradeState("suspended"), software.UpgradeSuspended)
	assert.Equal(t, software.UpgradeState("terminated"), software.UpgradeTerminated)

	// Task types
	assert.Equal(t, software.TaskType(1), software.TaskTypeUpgrade)
	assert.Equal(t, software.TaskType(2), software.TaskTypeRollback)
	assert.Equal(t, software.TaskType(4), software.TaskTypePatch)
	assert.Equal(t, software.TaskType(6), software.TaskTypeFPGA)
	assert.Equal(t, software.TaskType(8), software.TaskTypeReserved)
}

// TestListResponseJSONFormat verifies the paginated list response format
// matches frontend PageResponse expectations.
func TestListResponseJSONFormat(t *testing.T) {
	tasks := []software.UpgradeTask{
		{ID: uuid.New(), TaskName: "task-1", Status: software.TaskPending},
		{ID: uuid.New(), TaskName: "task-2", Status: software.TaskInProgress},
	}
	resp := model.NewListResponse(tasks, 2, 1, 20)

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "items")
	assert.Contains(t, result, "total")
	assert.Contains(t, result, "page")
	assert.Contains(t, result, "page_size")
	assert.Contains(t, result, "total_pages")

	items, ok := result["items"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, len(items))
}

// ---------------------------------------------------------------------------
// Minimal stubs for handler route test
// ---------------------------------------------------------------------------

type swHTaskRepoStub struct{}

func (s *swHTaskRepoStub) Create(_ context.Context, task *software.UpgradeTask) error {
	task.ID = uuid.New()
	return nil
}
func (s *swHTaskRepoStub) GetByID(_ context.Context, _ uuid.UUID) (*software.UpgradeTask, error) {
	return nil, nil
}
func (s *swHTaskRepoStub) Update(_ context.Context, _ *software.UpgradeTask) error { return nil }
func (s *swHTaskRepoStub) UpdateStatus(_ context.Context, _ uuid.UUID, _ software.TaskStatus, _ software.TaskResult) error {
	return nil
}
func (s *swHTaskRepoStub) List(_ context.Context, _ software.UpgradeTaskFilter) (*model.ListResponse[software.UpgradeTask], error) {
	return model.NewListResponse([]software.UpgradeTask{}, 0, 1, 20), nil
}
func (s *swHTaskRepoStub) IncrementCounts(_ context.Context, _ uuid.UUID, _, _ int) error {
	return nil
}
func (s *swHTaskRepoStub) Delete(_ context.Context, _ uuid.UUID) error { return nil }

// Canary repository methods (added in T-0018; stub returns harmless zero values).
func (s *swHTaskRepoStub) GetCanaryFields(_ context.Context, _ uuid.UUID) (*software.CanaryFields, error) {
	return nil, nil
}
func (s *swHTaskRepoStub) UpdateCanaryFields(_ context.Context, _ uuid.UUID, _ *software.CanaryFields) error {
	return nil
}
func (s *swHTaskRepoStub) ListActiveCanaryTaskIDs(_ context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

type swHSubTaskRepoStub struct{}

func (s *swHSubTaskRepoStub) Create(_ context.Context, task *software.UpgradeSubTask) error {
	task.ID = uuid.New()
	return nil
}
func (s *swHSubTaskRepoStub) GetByID(_ context.Context, _ uuid.UUID) (*software.UpgradeSubTask, error) {
	return nil, nil
}
func (s *swHSubTaskRepoStub) UpdateStatus(_ context.Context, _ uuid.UUID, _ software.UpgradeState, _ string) error {
	return nil
}
func (s *swHSubTaskRepoStub) Update(_ context.Context, _ *software.UpgradeSubTask) error { return nil }
func (s *swHSubTaskRepoStub) List(_ context.Context, _ software.SubTaskFilter) (*model.ListResponse[software.UpgradeSubTask], error) {
	return model.NewListResponse([]software.UpgradeSubTask{}, 0, 1, 20), nil
}
func (s *swHSubTaskRepoStub) ListByTaskID(_ context.Context, _ uuid.UUID, _ software.SubTaskFilter) (*model.ListResponse[software.UpgradeSubTask], error) {
	return model.NewListResponse([]software.UpgradeSubTask{}, 0, 1, 20), nil
}
func (s *swHSubTaskRepoStub) GetActiveByDeviceID(_ context.Context, _ uuid.UUID) (*software.UpgradeSubTask, error) {
	return nil, nil
}
func (s *swHSubTaskRepoStub) GetByCommandKey(_ context.Context, _ string) (*software.UpgradeSubTask, error) {
	return nil, nil
}
func (s *swHSubTaskRepoStub) BatchCreate(_ context.Context, tasks []*software.UpgradeSubTask) error {
	for _, t := range tasks {
		t.ID = uuid.New()
	}
	return nil
}
func (s *swHSubTaskRepoStub) FailStale(_ context.Context, _ time.Time) (map[uuid.UUID]int64, error) {
	return nil, nil
}
func (s *swHSubTaskRepoStub) DeleteByTaskID(_ context.Context, _ uuid.UUID) error { return nil }
func (s *swHSubTaskRepoStub) ListAll(_ context.Context, _ software.AllSubTaskFilter) (*model.ListResponse[software.UpgradeSubTaskWithTaskName], error) {
	return model.NewListResponse([]software.UpgradeSubTaskWithTaskName{}, 0, 1, 20), nil
}
func (s *swHSubTaskRepoStub) UpdateStatusWithCode(_ context.Context, _ uuid.UUID, _ software.UpgradeState, _ string, _ software.FailureCode) error {
	return nil
}
func (s *swHSubTaskRepoStub) UpdateFailureReasonByTask(_ context.Context, _ uuid.UUID, _ software.FailureCode) error {
	return nil
}

type swHFirmwareRepoStub struct{}

func (s *swHFirmwareRepoStub) Create(_ context.Context, _ *software.FirmwareVersion) error { return nil }
func (s *swHFirmwareRepoStub) GetByID(_ context.Context, _ uuid.UUID) (*software.FirmwareVersion, error) {
	return nil, nil
}
func (s *swHFirmwareRepoStub) List(_ context.Context, _ software.FirmwareFilter) (*model.ListResponse[software.FirmwareVersion], error) {
	return model.NewListResponse([]software.FirmwareVersion{}, 0, 1, 20), nil
}
func (s *swHFirmwareRepoStub) Update(_ context.Context, _ *software.FirmwareVersion) error { return nil }
func (s *swHFirmwareRepoStub) Delete(_ context.Context, _ uuid.UUID) error                 { return nil }
