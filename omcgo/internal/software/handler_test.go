package software

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks (swH prefix to avoid conflict with state_machine_test.go)
// ---------------------------------------------------------------------------

type swHFirmwareRepo struct {
	listFn    func(ctx context.Context, filter FirmwareFilter) (*model.ListResponse[FirmwareVersion], error)
	getByIDFn func(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error)
	createFn  func(ctx context.Context, fw *FirmwareVersion) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	updateFn  func(ctx context.Context, fw *FirmwareVersion) error
}

func (m *swHFirmwareRepo) Create(ctx context.Context, fw *FirmwareVersion) error {
	if m.createFn != nil {
		return m.createFn(ctx, fw)
	}
	fw.ID = uuid.New()
	return nil
}
func (m *swHFirmwareRepo) GetByID(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *swHFirmwareRepo) List(ctx context.Context, filter FirmwareFilter) (*model.ListResponse[FirmwareVersion], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]FirmwareVersion{}, 0, 1, 20), nil
}
func (m *swHFirmwareRepo) Update(ctx context.Context, fw *FirmwareVersion) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, fw)
	}
	return nil
}
func (m *swHFirmwareRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type swHTaskRepo struct {
	listFn    func(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error)
	getByIDFn func(ctx context.Context, id uuid.UUID) (*UpgradeTask, error)
}

func (m *swHTaskRepo) Create(_ context.Context, task *UpgradeTask) error {
	task.ID = uuid.New()
	return nil
}
func (m *swHTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *swHTaskRepo) Update(_ context.Context, _ *UpgradeTask) error { return nil }
func (m *swHTaskRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ TaskStatus, _ TaskResult) error {
	return nil
}
func (m *swHTaskRepo) List(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]UpgradeTask{}, 0, 1, 20), nil
}
func (m *swHTaskRepo) IncrementCounts(_ context.Context, _ uuid.UUID, _, _ int) error {
	return nil
}
func (m *swHTaskRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *swHTaskRepo) GetCanaryFields(_ context.Context, _ uuid.UUID) (*CanaryFields, error) {
	return nil, nil
}
func (m *swHTaskRepo) UpdateCanaryFields(_ context.Context, _ uuid.UUID, _ *CanaryFields) error {
	return nil
}
func (m *swHTaskRepo) ListActiveCanaryTaskIDs(_ context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

type swHSubTaskRepo struct{}

func (m *swHSubTaskRepo) Create(_ context.Context, task *UpgradeSubTask) error {
	task.ID = uuid.New()
	return nil
}
func (m *swHSubTaskRepo) GetByID(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *swHSubTaskRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string) error {
	return nil
}
func (m *swHSubTaskRepo) Update(_ context.Context, _ *UpgradeSubTask) error { return nil }
func (m *swHSubTaskRepo) List(_ context.Context, _ SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return model.NewListResponse([]UpgradeSubTaskWithTaskName{}, 0, 1, 20), nil
}
func (m *swHSubTaskRepo) ListByTaskID(_ context.Context, _ uuid.UUID, _ SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return model.NewListResponse([]UpgradeSubTaskWithTaskName{}, 0, 1, 20), nil
}
func (m *swHSubTaskRepo) ListAll(_ context.Context, _ AllSubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return model.NewListResponse([]UpgradeSubTaskWithTaskName{}, 0, 1, 20), nil
}
func (m *swHSubTaskRepo) UpdateStatusWithCode(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string, _ FailureCode) error {
	return nil
}
func (m *swHSubTaskRepo) UpdateFailureReasonByTask(_ context.Context, _ uuid.UUID, _ FailureCode) error {
	return nil
}
func (m *swHSubTaskRepo) GetActiveByDeviceID(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *swHSubTaskRepo) GetByCommandKey(_ context.Context, _ string) (*UpgradeSubTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *swHSubTaskRepo) BatchCreate(_ context.Context, tasks []*UpgradeSubTask) error {
	for _, t := range tasks {
		t.ID = uuid.New()
	}
	return nil
}
func (m *swHSubTaskRepo) FailStale(_ context.Context, _ time.Time) (map[uuid.UUID]int64, error) {
	return nil, nil
}
func (m *swHSubTaskRepo) DeleteByTaskID(_ context.Context, _ uuid.UUID) error { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func swHSetupRouter(fwRepo FirmwareRepository, taskRepo TaskRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &SoftwareService{firmwareRepo: fwRepo, logger: zap.NewNop()}
	h := NewHandler(svc, fwRepo, taskRepo, &swHSubTaskRepo{}, zap.NewNop())
	h.RegisterRoutes(r.Group(""))
	return r
}

func swHSampleFirmware(id uuid.UUID) *FirmwareVersion {
	return &FirmwareVersion{
		ID:        id,
		Version:   "V1.0.0",
		FileName:  "firmware.bin",
		FileSize:  1024000,
		MinIOPath: "firmware/SmallCell/V1.0.0/firmware.bin",
		Status:    "active",
		CreatedAt: JSONTime(time.Now()),
		UpdatedAt: JSONTime(time.Now()),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSwHandler_ListFirmware_Default(t *testing.T) {
	fwRepo := &swHFirmwareRepo{
		listFn: func(_ context.Context, _ FirmwareFilter) (*model.ListResponse[FirmwareVersion], error) {
			return model.NewListResponse([]FirmwareVersion{
				*swHSampleFirmware(uuid.New()),
				*swHSampleFirmware(uuid.New()),
			}, 2, 1, 20), nil
		},
	}
	router := swHSetupRouter(fwRepo, &swHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/firmware?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[FirmwareVersion]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
}

func TestSwHandler_GetFirmware_Success(t *testing.T) {
	id := uuid.New()
	fwRepo := &swHFirmwareRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*FirmwareVersion, error) {
			assert.Equal(t, id, gotID)
			return swHSampleFirmware(id), nil
		},
	}
	router := swHSetupRouter(fwRepo, &swHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/firmware/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp FirmwareVersion
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, id, resp.ID)
}

func TestSwHandler_GetFirmware_NotFound(t *testing.T) {
	router := swHSetupRouter(&swHFirmwareRepo{}, &swHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/firmware/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSwHandler_DeleteFirmware_Success(t *testing.T) {
	id := uuid.New()
	deleteCalled := false
	fwRepo := &swHFirmwareRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*FirmwareVersion, error) {
			return swHSampleFirmware(gotID), nil
		},
		deleteFn: func(_ context.Context, gotID uuid.UUID) error {
			assert.Equal(t, id, gotID)
			deleteCalled = true
			return nil
		},
	}
	router := swHSetupRouter(fwRepo, &swHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/firmware/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, deleteCalled)
}

func TestSwHandler_ListUpgradeTasks_Default(t *testing.T) {
	taskRepo := &swHTaskRepo{
		listFn: func(_ context.Context, _ UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
			return model.NewListResponse([]UpgradeTask{
				{ID: uuid.New(), Status: TaskPending, TaskName: "test-task"},
			}, 1, 1, 20), nil
		},
	}
	router := swHSetupRouter(&swHFirmwareRepo{}, taskRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/upgrade-tasks?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[UpgradeTask]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
}

func TestSwHandler_GetUpgradeTask_Success(t *testing.T) {
	id := uuid.New()
	taskRepo := &swHTaskRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{ID: gotID, Status: TaskInProgress, TaskName: "test"}, nil
		},
	}
	router := swHSetupRouter(&swHFirmwareRepo{}, taskRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/upgrade-tasks/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp UpgradeTask
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, id, resp.ID)
}

func TestSwHandler_GetUpgradeTask_NotFound(t *testing.T) {
	router := swHSetupRouter(&swHFirmwareRepo{}, &swHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/upgrade-tasks/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
