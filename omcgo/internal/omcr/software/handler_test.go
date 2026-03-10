package software

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
func (m *swHFirmwareRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type swHUpgradeRepo struct {
	listFn    func(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error)
	getByIDFn func(ctx context.Context, id uuid.UUID) (*UpgradeTask, error)
}

func (m *swHUpgradeRepo) Create(_ context.Context, task *UpgradeTask) error {
	task.ID = uuid.New()
	return nil
}
func (m *swHUpgradeRepo) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *swHUpgradeRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string) error {
	return nil
}
func (m *swHUpgradeRepo) List(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]UpgradeTask{}, 0, 1, 20), nil
}
func (m *swHUpgradeRepo) GetActiveByDeviceID(_ context.Context, _ uuid.UUID) (*UpgradeTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *swHUpgradeRepo) CountByBatchStatus(_ context.Context, _ uuid.UUID) (map[UpgradeState]int64, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func swHSetupRouter(fwRepo FirmwareRepository, upgradeRepo UpgradeTaskRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Pass nil service — only test endpoints that use repos directly
	h := NewHandler(nil, fwRepo, upgradeRepo, zap.NewNop())
	h.RegisterRoutes(r.Group(""))
	return r
}

func swHSampleFirmware(id uuid.UUID) *FirmwareVersion {
	return &FirmwareVersion{
		ID:        id,
		Carrier:   "cmcc",
		Version:   "V1.0.0",
		FileName:  "firmware.bin",
		FileSize:  1024000,
		MinIOPath: "firmware/cmcc/SmallCell/V1.0.0/firmware.bin",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
	router := swHSetupRouter(fwRepo, &swHUpgradeRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/firmware?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[FirmwareVersion]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
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
	router := swHSetupRouter(fwRepo, &swHUpgradeRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/firmware/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp FirmwareVersion
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, id, resp.ID)
}

func TestSwHandler_GetFirmware_NotFound(t *testing.T) {
	router := swHSetupRouter(&swHFirmwareRepo{}, &swHUpgradeRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/firmware/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSwHandler_DeleteFirmware_Success(t *testing.T) {
	deleteCalled := false
	fwRepo := &swHFirmwareRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	router := swHSetupRouter(fwRepo, &swHUpgradeRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/firmware/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, deleteCalled)
}

func TestSwHandler_ListUpgradeTasks_Default(t *testing.T) {
	upgradeRepo := &swHUpgradeRepo{
		listFn: func(_ context.Context, _ UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
			return model.NewListResponse([]UpgradeTask{
				{ID: uuid.New(), Status: UpgradePending},
			}, 1, 1, 20), nil
		},
	}
	router := swHSetupRouter(&swHFirmwareRepo{}, upgradeRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/upgrade-tasks?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[UpgradeTask]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Total)
}

func TestSwHandler_GetUpgradeTask_Success(t *testing.T) {
	id := uuid.New()
	upgradeRepo := &swHUpgradeRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{ID: gotID, Status: UpgradeDownloading}, nil
		},
	}
	router := swHSetupRouter(&swHFirmwareRepo{}, upgradeRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/upgrade-tasks/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp UpgradeTask
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, id, resp.ID)
}

func TestSwHandler_GetUpgradeTask_NotFound(t *testing.T) {
	router := swHSetupRouter(&swHFirmwareRepo{}, &swHUpgradeRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/upgrade-tasks/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
