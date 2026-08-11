package northbound

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/topology"
	"github.com/omcgo/omcgo/internal/ufte"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeNBDeviceService struct {
	listFn       func(context.Context, device.DeviceFilter) (*model.ListResponse[device.DeviceWithInfo], error)
	getFn        func(context.Context, uuid.UUID) (*model.Device, error)
	getBySNFn    func(context.Context, string) (*model.Device, error)
	getInfoFn    func(context.Context, uuid.UUID) (*device.DeviceWithInfo, error)
	paramsFn     func(context.Context, uuid.UUID) ([]model.DeviceParameter, error)
	setParamsFn  func(context.Context, uuid.UUID, []device.ParameterValueItem, string) (string, error)
	syncParamsFn func(context.Context, uuid.UUID, string, []string) (*device.ManualParamSyncStart, *model.Device, error)
	renameFn     func(context.Context, uuid.UUID, string, string) (*device.RenameDeviceResult, error)
}

func (f *fakeNBDeviceService) ListDevicesWithInfo(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[device.DeviceWithInfo], error) {
	return f.listFn(ctx, filter)
}

func (f *fakeNBDeviceService) GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	return f.getFn(ctx, id)
}

func (f *fakeNBDeviceService) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	return f.getBySNFn(ctx, sn)
}

func (f *fakeNBDeviceService) GetDeviceWithInfo(ctx context.Context, id uuid.UUID) (*device.DeviceWithInfo, error) {
	return f.getInfoFn(ctx, id)
}

func (f *fakeNBDeviceService) GetDeviceParameters(ctx context.Context, id uuid.UUID) ([]model.DeviceParameter, error) {
	return f.paramsFn(ctx, id)
}

func (f *fakeNBDeviceService) SetParameters(ctx context.Context, id uuid.UUID, params []device.ParameterValueItem, creatorID string) (string, error) {
	return f.setParamsFn(ctx, id, params, creatorID)
}

func (f *fakeNBDeviceService) SyncDeviceParamsManualDetailed(ctx context.Context, id uuid.UUID, sourceID string, paths []string) (*device.ManualParamSyncStart, *model.Device, error) {
	return f.syncParamsFn(ctx, id, sourceID, paths)
}

func (f *fakeNBDeviceService) RenameDevice(ctx context.Context, id uuid.UUID, newName string, creatorID string) (*device.RenameDeviceResult, error) {
	return f.renameFn(ctx, id, newName, creatorID)
}

type fakeNBTaskService struct {
	createFn  func(context.Context, *task.CreateTaskRequest) (*task.Task, error)
	getFn     func(context.Context, string) (*task.Task, error)
	historyFn func(context.Context, string, *task.TaskHistoryOptions) (*task.TaskListResponse, error)
}

func (f *fakeNBTaskService) CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	return f.createFn(ctx, req)
}

func (f *fakeNBTaskService) GetTask(ctx context.Context, taskID string) (*task.Task, error) {
	return f.getFn(ctx, taskID)
}

func (f *fakeNBTaskService) GetTaskHistory(ctx context.Context, deviceSN string, opts *task.TaskHistoryOptions) (*task.TaskListResponse, error) {
	return f.historyFn(ctx, deviceSN, opts)
}

type fakeNBRegistrationService struct {
	registerFn func(context.Context, device.CreateRegistrationRequest, string) (*device.DeviceRegistration, error)
	listFn     func(context.Context, device.RegistrationFilter) (*model.ListResponse[device.DeviceRegistration], error)
	deleteFn   func(context.Context, uuid.UUID) error
}

func (f *fakeNBRegistrationService) Register(ctx context.Context, req device.CreateRegistrationRequest, operator string) (*device.DeviceRegistration, error) {
	return f.registerFn(ctx, req, operator)
}

func (f *fakeNBRegistrationService) List(ctx context.Context, filter device.RegistrationFilter) (*model.ListResponse[device.DeviceRegistration], error) {
	return f.listFn(ctx, filter)
}

func (f *fakeNBRegistrationService) Delete(ctx context.Context, id uuid.UUID) error {
	return f.deleteFn(ctx, id)
}

type fakeNBGroupService struct {
	treeFn    func(context.Context) ([]topology.DeviceGroup, error)
	createFn  func(context.Context, topology.CreateGroupRequest, string) (*topology.DeviceGroup, error)
	updateFn  func(context.Context, uuid.UUID, topology.UpdateGroupRequest, string) (*topology.DeviceGroup, error)
	deleteFn  func(context.Context, uuid.UUID) error
	addDevsFn func(context.Context, uuid.UUID, []uuid.UUID) (int64, error)
}

func (f *fakeNBGroupService) GetTree(ctx context.Context) ([]topology.DeviceGroup, error) {
	return f.treeFn(ctx)
}

func (f *fakeNBGroupService) CreateGroup(ctx context.Context, req topology.CreateGroupRequest, operator string) (*topology.DeviceGroup, error) {
	return f.createFn(ctx, req, operator)
}

func (f *fakeNBGroupService) UpdateGroup(ctx context.Context, id uuid.UUID, req topology.UpdateGroupRequest, operator string) (*topology.DeviceGroup, error) {
	return f.updateFn(ctx, id, req, operator)
}

func (f *fakeNBGroupService) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	return f.deleteFn(ctx, id)
}

func (f *fakeNBGroupService) BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	return f.addDevsFn(ctx, groupID, deviceIDs)
}

type fakeNBTransferTaskService struct {
	createFn func(context.Context, ufte.CreateTaskRequest, string, []uuid.UUID) (*ufte.Task, error)
	getFn    func(context.Context, string, []uuid.UUID) (*ufte.Task, error)
	listFn   func(context.Context, ufte.TaskListFilter, []uuid.UUID) (*model.ListResponse[ufte.Task], error)
}

func (f *fakeNBTransferTaskService) CreateTask(ctx context.Context, req ufte.CreateTaskRequest, createUser string, visibleGroups []uuid.UUID) (*ufte.Task, error) {
	return f.createFn(ctx, req, createUser, visibleGroups)
}

func (f *fakeNBTransferTaskService) GetTask(ctx context.Context, id string, visibleGroups []uuid.UUID) (*ufte.Task, error) {
	return f.getFn(ctx, id, visibleGroups)
}

func (f *fakeNBTransferTaskService) ListTasks(ctx context.Context, filter ufte.TaskListFilter, visibleGroups []uuid.UUID) (*model.ListResponse[ufte.Task], error) {
	return f.listFn(ctx, filter, visibleGroups)
}

func setupDeviceFacadeRouter(deviceSvc northboundDeviceService, taskSvc northboundTaskService) *gin.Engine {
	return setupDeviceFacadeRouterWithExtras(deviceSvc, taskSvc, nil, nil, nil)
}

func setupDeviceFacadeRouterWithExtras(
	deviceSvc northboundDeviceService,
	taskSvc northboundTaskService,
	regSvc northboundRegistrationService,
	groupSvc northboundDeviceGroupService,
	transferSvc northboundTransferTaskService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := &NorthboundService{logger: zap.NewNop()}
	router := NewRouter(svc)
	router.pageConfigService = nil
	router.pageConfigHandler = nil
	router.SetDeviceTaskServices(deviceSvc, taskSvc)
	router.SetDeviceRegistrationService(regSvc)
	router.SetDeviceGroupService(groupSvc)
	router.SetTransferTaskService(transferSvc)
	engine := gin.New()
	router.RegisterPublicRoutes(engine.Group("/api/v1"))
	return engine
}

func TestLegacyFacadeDeviceQueryUsesCurrentDeviceService(t *testing.T) {
	deviceID := uuid.New()
	deviceSvc := &fakeNBDeviceService{
		listFn: func(_ context.Context, filter device.DeviceFilter) (*model.ListResponse[device.DeviceWithInfo], error) {
			require.NotNil(t, filter.SN)
			require.Equal(t, "SN001", *filter.SN)
			require.NotNil(t, filter.Technology)
			require.Equal(t, model.TechLTE, *filter.Technology)
			return model.NewListResponse([]device.DeviceWithInfo{{
				Device: model.Device{ID: deviceID, SerialNumber: "SN001", Technology: model.TechLTE, IsOnline: true},
			}}, 1, 1, 20), nil
		},
	}

	router := setupDeviceFacadeRouter(deviceSvc, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/v1/device/query", bytes.NewBufferString(`{"sn":"SN001","technology":"LTE","page":1,"rows":20}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "SN001")
}

func TestDeviceFacadeSetParametersReturnsLegacyJobFields(t *testing.T) {
	deviceID := uuid.New()
	deviceSvc := &fakeNBDeviceService{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			require.Equal(t, "SN001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
		setParamsFn: func(_ context.Context, id uuid.UUID, params []device.ParameterValueItem, _ string) (string, error) {
			require.Equal(t, deviceID, id)
			require.Len(t, params, 1)
			require.Equal(t, "Device.DeviceInfo.X", params[0].Path)
			return "task-001", nil
		},
	}

	router := setupDeviceFacadeRouter(deviceSvc, nil)
	body := bytes.NewBufferString(`{"parameters":[{"path":"Device.DeviceInfo.X","value":"1","type":"xsd:string"}]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/v1/device/parameters/SN001", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, "task-001", envelope.Data["task_id"])
	require.Equal(t, "task-001", envelope.Data["jobId"])
	require.Equal(t, "SN001", envelope.Data["sn"])
	require.Equal(t, float64(5), envelope.Data["waitTime"])
}

func TestDeviceFacadeTaskDetailReturnsLegacyStatus(t *testing.T) {
	taskID := uuid.NewString()
	taskSvc := &fakeNBTaskService{
		getFn: func(_ context.Context, gotTaskID string) (*task.Task, error) {
			require.Equal(t, taskID, gotTaskID)
			return &task.Task{ID: gotTaskID, DeviceSN: "SN001", Method: "SetParameterValues", Status: task.TaskStatusCompleted}, nil
		},
	}

	router := setupDeviceFacadeRouter(nil, taskSvc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/v1/job/result/"+taskID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"legacy_status":"2"`)
	require.Contains(t, rec.Body.String(), `"jobId":"`+taskID+`"`)
}

func TestLegacyFacadeGetParametersReturnsSnapshotMap(t *testing.T) {
	deviceID := uuid.New()
	deviceSvc := &fakeNBDeviceService{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			require.Equal(t, "SN001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
		paramsFn: func(_ context.Context, id uuid.UUID) ([]model.DeviceParameter, error) {
			require.Equal(t, deviceID, id)
			return []model.DeviceParameter{{
				DeviceID:       id,
				ParameterPath:  "Device.DeviceInfo.SoftwareVersion",
				ParameterValue: "BaiBS_QRTB_2.9",
				ParameterType:  model.ParamString,
			}}, nil
		},
	}

	router := setupDeviceFacadeRouter(deviceSvc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/v1/device/parameters/SN001", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"sn":"SN001"`)
	require.Contains(t, rec.Body.String(), `"Device.DeviceInfo.SoftwareVersion":"BaiBS_QRTB_2.9"`)
}

func TestLegacyFacadeRebootReturnsJobFields(t *testing.T) {
	deviceID := uuid.New()
	deviceSvc := &fakeNBDeviceService{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			require.Equal(t, "SN001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}
	taskSvc := &fakeNBTaskService{
		createFn: func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
			require.Equal(t, "SN001", req.DeviceSN)
			require.Equal(t, "Reboot", req.Method)
			return &task.Task{ID: "task-reboot", DeviceSN: req.DeviceSN, Method: req.Method, Status: task.TaskStatusPending}, nil
		},
	}

	router := setupDeviceFacadeRouter(deviceSvc, taskSvc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/v1/device/reboot/SN001", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Contains(t, rec.Body.String(), `"jobId":"task-reboot"`)
	require.Contains(t, rec.Body.String(), `"sn":"SN001"`)
}

func TestLegacyFacadeCPERoutesAreNotExposed(t *testing.T) {
	router := setupDeviceFacadeRouter(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/v1/cpe/status/SN001", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLegacyFacadeRegistrationCreateUsesCurrentService(t *testing.T) {
	groupID := uuid.New()
	regSvc := &fakeNBRegistrationService{
		registerFn: func(_ context.Context, req device.CreateRegistrationRequest, _ string) (*device.DeviceRegistration, error) {
			require.Equal(t, "SN001", req.SerialNumber)
			require.Equal(t, groupID.String(), req.GroupID)
			require.Equal(t, "CMCC", req.Carrier)
			return &device.DeviceRegistration{ID: uuid.New(), SerialNumber: req.SerialNumber, GroupID: &groupID, Carrier: model.CarrierCMCC}, nil
		},
	}

	router := setupDeviceFacadeRouterWithExtras(nil, nil, regSvc, nil, nil)
	body := bytes.NewBufferString(`{"serialNumber":"SN001","groupId":"` + groupID.String() + `","carrier":"CMCC"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/v1/device/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"serial_number":"SN001"`)
}

func TestLegacyFacadeGroupAddDevicesAcceptsSNS(t *testing.T) {
	groupID := uuid.New()
	deviceID := uuid.New()
	deviceSvc := &fakeNBDeviceService{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			require.Equal(t, "SN001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}
	groupSvc := &fakeNBGroupService{
		addDevsFn: func(_ context.Context, gotGroupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
			require.Equal(t, groupID, gotGroupID)
			require.Equal(t, []uuid.UUID{deviceID}, deviceIDs)
			return 1, nil
		},
	}

	router := setupDeviceFacadeRouterWithExtras(deviceSvc, nil, nil, groupSvc, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/v1/device/group/"+groupID.String()+"/devices", bytes.NewBufferString(`{"sns":["SN001"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"affected":1`)
}

func TestLegacyFacadeResetReturnsJobFields(t *testing.T) {
	deviceID := uuid.New()
	deviceSvc := &fakeNBDeviceService{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}
	taskSvc := &fakeNBTaskService{
		createFn: func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
			require.Equal(t, "SN001", req.DeviceSN)
			require.Equal(t, "FactoryReset", req.Method)
			return &task.Task{ID: "task-reset", DeviceSN: req.DeviceSN, Method: req.Method, Status: task.TaskStatusPending}, nil
		},
	}

	router := setupDeviceFacadeRouterWithExtras(deviceSvc, taskSvc, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/v1/device/reset/SN001", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Contains(t, rec.Body.String(), `"jobId":"task-reset"`)
}

func TestLegacyFacadeLogCollectUsesUFTE(t *testing.T) {
	deviceID := uuid.New()
	deviceSvc := &fakeNBDeviceService{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn, ProductClass: "FAP-LTE-100"}, nil
		},
	}
	transferSvc := &fakeNBTransferTaskService{
		createFn: func(_ context.Context, req ufte.CreateTaskRequest, _ string, visibleGroups []uuid.UUID) (*ufte.Task, error) {
			require.Nil(t, visibleGroups)
			require.Equal(t, "RUNTIME_LOG_COLLECT", req.TypeCode)
			require.Equal(t, []uuid.UUID{deviceID}, req.DeviceIDs)
			require.Equal(t, "immediate", req.ExecutionMode)
			return &ufte.Task{ID: "ufte-log-task", TaskName: req.TaskName, TypeCode: req.TypeCode, Status: "in_progress", ExecutionMode: req.ExecutionMode}, nil
		},
	}

	router := setupDeviceFacadeRouterWithExtras(deviceSvc, nil, nil, nil, transferSvc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/v1/device/log/collect/SN001", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Contains(t, rec.Body.String(), `"jobId":"ufte-log-task"`)
	require.Contains(t, rec.Body.String(), `"typeCode":"RUNTIME_LOG_COLLECT"`)
}

func TestLegacyFacadeTaskDetailFallsBackToUFTE(t *testing.T) {
	taskID := uuid.NewString()
	transferSvc := &fakeNBTransferTaskService{
		getFn: func(_ context.Context, id string, visibleGroups []uuid.UUID) (*ufte.Task, error) {
			require.Equal(t, taskID, id)
			require.Nil(t, visibleGroups)
			return &ufte.Task{ID: id, TaskName: "log collect", TypeCode: "RUNTIME_LOG_COLLECT", Category: "station_log", Status: "ended", Progress: 100}, nil
		},
	}

	router := setupDeviceFacadeRouterWithExtras(nil, nil, nil, nil, transferSvc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/v1/job/result/"+taskID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"jobId":"`+taskID+`"`)
	require.Contains(t, rec.Body.String(), `"legacy_status":"2"`)
}

func TestLegacyFacadeTaskDetailInvalidStorageIDReturnsBadRequest(t *testing.T) {
	called := false
	taskSvc := &fakeNBTaskService{
		getFn: func(_ context.Context, _ string) (*task.Task, error) {
			called = true
			return nil, errors.New(`get task from repository: ERROR: invalid input syntax for type uuid: "not-a-uuid" (SQLSTATE 22P02)`)
		},
	}

	router := setupDeviceFacadeRouter(nil, taskSvc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/v1/job/result/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.False(t, called)
	require.Contains(t, rec.Body.String(), `"msg":"invalid task_id"`)
}

func TestLegacyFacadeListTasksWithoutSNUsesUFTE(t *testing.T) {
	transferSvc := &fakeNBTransferTaskService{
		listFn: func(_ context.Context, filter ufte.TaskListFilter, visibleGroups []uuid.UUID) (*model.ListResponse[ufte.Task], error) {
			require.Nil(t, visibleGroups)
			require.Equal(t, "station_log", filter.Category)
			require.Equal(t, 2, filter.Page)
			require.Equal(t, 5, filter.PageSize)
			return model.NewListResponse([]ufte.Task{{
				ID:       uuid.NewString(),
				TaskName: "northbound log collect",
				Category: "station_log",
				Status:   "ended",
			}}, 1, 2, 5), nil
		},
	}

	router := setupDeviceFacadeRouterWithExtras(nil, nil, nil, nil, transferSvc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/v1/job/result/page", bytes.NewBufferString(`{"page":2,"rows":5}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"northbound log collect"`)
}
