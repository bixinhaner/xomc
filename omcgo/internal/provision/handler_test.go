package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// stubDeviceChecker satisfies deviceExistenceChecker for handler tests.
// GetFn lets each case decide whether the device exists (non-nil),
// is absent (nil, nil — the PgDeviceRepository semantics for a missing row),
// or the lookup errored.
type stubDeviceChecker struct {
	GetFn func(ctx context.Context, id uuid.UUID) (*model.Device, error)
}

func (s *stubDeviceChecker) GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if s.GetFn != nil {
		return s.GetFn(ctx, id)
	}
	return nil, nil
}

type retryDeviceService struct {
	device *model.Device
}

func (s *retryDeviceService) GetDevice(context.Context, uuid.UUID) (*model.Device, error) {
	return s.device, nil
}

func (s *retryDeviceService) ListDevices(context.Context, device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{*s.device}, 1, 1, 20), nil
}

type retryXMLRepository struct {
	file          *ProvisioningXMLFile
	rotatedToken  uuid.UUID
	createInvoked bool
}

func (r *retryXMLRepository) CreateXML(context.Context, *ProvisioningXMLFile) error {
	r.createInvoked = true
	return nil
}

func (r *retryXMLRepository) GetXML(context.Context, uuid.UUID) (*ProvisioningXMLFile, error) {
	clone := *r.file
	return &clone, nil
}

func (r *retryXMLRepository) RotateXMLDownloadToken(context.Context, uuid.UUID) (uuid.UUID, error) {
	return r.rotatedToken, nil
}

type retryTaskEnqueuer struct {
	request *devtask.CreateTaskRequest
}

func (e *retryTaskEnqueuer) CreateTask(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
	e.request = req
	return &devtask.Task{ID: uuid.NewString()}, nil
}

type recordingXMLObjectStore struct {
	bucket, object, content string
}

func (s *recordingXMLObjectStore) PutObject(
	_ context.Context,
	bucket, object string,
	reader io.Reader,
	_ int64,
	_ minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	s.bucket, s.object, s.content = bucket, object, string(content)
	return minio.UploadInfo{}, nil
}

type executePolicyRepository struct {
	policy   *PlugAndPlayPolicy
	policies []PlugAndPlayPolicy
}

func (r *executePolicyRepository) CreatePolicy(context.Context, *PlugAndPlayPolicy) error {
	return nil
}
func (r *executePolicyRepository) GetPolicy(context.Context, uuid.UUID) (*PlugAndPlayPolicy, error) {
	return r.policy, nil
}
func (r *executePolicyRepository) ListPolicies(context.Context, PolicyFilter) ([]PlugAndPlayPolicy, int64, error) {
	return r.policies, int64(len(r.policies)), nil
}
func (r *executePolicyRepository) UpdatePolicy(context.Context, *PlugAndPlayPolicy) error {
	return nil
}
func (r *executePolicyRepository) DeletePolicy(context.Context, uuid.UUID) error {
	return nil
}

type executePolicyDeviceService struct {
	devices map[uuid.UUID]*model.Device
}

func (s *executePolicyDeviceService) GetDevice(_ context.Context, id uuid.UUID) (*model.Device, error) {
	return s.devices[id], nil
}
func (s *executePolicyDeviceService) ListDevices(context.Context, device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}

type detectPolicyDeviceService struct {
	filter device.DeviceFilter
	item   model.Device
}

func (s *detectPolicyDeviceService) GetDevice(context.Context, uuid.UUID) (*model.Device, error) {
	return nil, nil
}

func (s *detectPolicyDeviceService) ListDevices(_ context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	s.filter = filter
	if filter.ProductClass != nil {
		for _, productClass := range device.SplitCSV(*filter.ProductClass) {
			if productClass == s.item.ProductClass {
				return model.NewListResponse([]model.Device{s.item}, 1, filter.Page, filter.PageSize), nil
			}
		}
	}
	return model.NewListResponse([]model.Device{}, 0, filter.Page, filter.PageSize), nil
}

type recordingFileWorkflow struct {
	upgradeCalls int
	licenseCalls int
	callOrder    []string
	version      string
	preserve     bool
	operator     string
}

func TestPlugAndPlayXMLFileTypeUsesAutoStartFileOutsideGNB(t *testing.T) {
	assert.Equal(t, "Auto Start File", plugAndPlayXMLFileType(&model.Device{Technology: model.TechLTE}))
	assert.Equal(t, "Auto Start File", plugAndPlayXMLFileType(&model.Device{Technology: model.TechGSM}))
	assert.Equal(t, "103 Base Station Startup File", plugAndPlayXMLFileType(&model.Device{Technology: model.TechNR}))
}

func TestHandlerListBindsPolicyOnly(t *testing.T) {
	var captured ProvisioningTaskFilter
	h := NewHandler(&mockTaskRepo{
		ListFn: func(_ context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error) {
			captured = filter
			return nil, 0, nil
		},
	}, nil)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/provisioning/tasks?policy_only=true", nil)

	h.List(c)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, captured.PolicyOnly)
}

func (w *recordingFileWorkflow) ExecuteUpgrade(
	_ context.Context,
	_ string,
	version string,
	preserve bool,
	_ []*model.Device,
	operator string,
) (uuid.UUID, error) {
	w.upgradeCalls++
	w.callOrder = append(w.callOrder, "software_upgrade")
	w.version, w.preserve, w.operator = version, preserve, operator
	return uuid.MustParse("11111111-1111-1111-1111-111111111111"), nil
}

func (w *recordingFileWorkflow) ExecuteLicense(
	_ context.Context,
	_ string,
	_ []*model.Device,
	operator string,
) (uuid.UUID, error) {
	w.licenseCalls++
	w.callOrder = append(w.callOrder, "license")
	w.operator = operator
	return uuid.MustParse("22222222-2222-2222-2222-222222222222"), nil
}

// newCreateRequest builds a POST /provisioning/tasks gin context for the handler.
func newCreateRequest(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provisioning/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, rec
}

func TestBindPolicyAllowsMultipleEnabledModules(t *testing.T) {
	c, rec := newCreateRequest(t, `{
		"name":"multi-module",
		"product_class":"FAP/TEST",
		"execute_type":"manual",
		"upgrade_enabled":true,
		"target_version":"V2.0",
		"license_enabled":true,
		"self_config_enabled":true,
		"config":{}
	}`)

	policy, ok := bindPolicy(c)

	require.True(t, ok, rec.Body.String())
	require.NotNil(t, policy)
	assert.True(t, policy.UpgradeEnabled)
	assert.True(t, policy.LicenseEnabled)
	assert.True(t, policy.SelfConfigEnabled)
}

func TestBindPolicyAcceptsMultipleProductClasses(t *testing.T) {
	c, rec := newCreateRequest(t, `{
		"name":"multi-product",
		"product_class":"FAP/OLD",
		"product_classes":[" FAP/A ","FAP/B","FAP/A"],
		"execute_type":"auto",
		"config":{}
	}`)

	policy, ok := bindPolicy(c)

	require.True(t, ok, rec.Body.String())
	require.NotNil(t, policy)
	assert.Equal(t, []string{"FAP/A", "FAP/B"}, policy.ProductClasses)
	assert.Equal(t, "FAP/A", policy.ProductClass)
}

func TestDetectDevicesIncludesEveryPolicyProductClass(t *testing.T) {
	policyID := uuid.New()
	devices := &detectPolicyDeviceService{item: model.Device{
		ID: uuid.New(), SerialNumber: "SN-SECOND", ProductClass: "FAP/B",
	}}
	h := NewHandler(nil, nil)
	h.policyRepo = &executePolicyRepository{policy: &PlugAndPlayPolicy{
		ID: policyID, ProductClass: "FAP/A", ProductClasses: []string{"FAP/A", "FAP/B"},
	}}
	h.deviceService = devices

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: policyID.String()}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/provisioning/policies/"+policyID.String()+"/devices", nil)

	h.DetectDevices(c)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, devices.filter.ProductClass)
	assert.Equal(t, "FAP/A,FAP/B", *devices.filter.ProductClass)
	assert.Contains(t, rec.Body.String(), "SN-SECOND")
}

func TestHandler_ExecutePolicyReusesFileWorkflowWithoutXMLWhenSelfConfigDisabled(t *testing.T) {
	policyID, deviceID := uuid.New(), uuid.New()
	workflow := &recordingFileWorkflow{}
	var created, updated *ProvisioningTask
	h := NewHandler(&mockTaskRepo{
		CreateFn: func(_ context.Context, task *ProvisioningTask) error {
			created = task
			return nil
		},
		UpdateFn: func(_ context.Context, task *ProvisioningTask) error {
			updated = task
			return nil
		},
	}, nil)
	h.policyRepo = &executePolicyRepository{policy: &PlugAndPlayPolicy{
		ID: policyID, Name: "P1", Enabled: true, ProductClass: "FAP/TEST",
		UpgradeEnabled: true, TargetVersion: "V2.0", LicenseEnabled: false,
		SelfConfigEnabled: false, Config: json.RawMessage(`{"preserveSetting":true}`),
	}}
	h.deviceService = &executePolicyDeviceService{devices: map[uuid.UUID]*model.Device{
		deviceID: {
			ID: deviceID, SerialNumber: "SN-001", ProductClass: "FAP/TEST",
		},
	}}
	h.SetPlugAndPlayFileWorkflow(workflow)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: policyID.String()}}
	c.Set("username", "pnp-operator")
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/provisioning/policies/"+policyID.String()+"/execute",
		strings.NewReader(fmt.Sprintf(`{"device_ids":[%q]}`, deviceID)),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExecutePolicy(c)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, 1, workflow.upgradeCalls)
	assert.Zero(t, workflow.licenseCalls)
	assert.Equal(t, "V2.0", workflow.version)
	assert.True(t, workflow.preserve)
	assert.Equal(t, "pnp-operator", workflow.operator)
	require.NotNil(t, created)
	require.NotNil(t, updated)
	assert.Equal(t, StateConfiguring, updated.Status)
	assert.Equal(t, "software_upgrade_task_submitted", updated.CurrentStepName)
	assert.Equal(t, 0, updated.CurrentStep)
	assert.Equal(t, 0, updated.MaxRetries)
	assert.JSONEq(t, fmt.Sprintf(
		`{"ret":1,"msg":"ok","data":{"items":[%s],"total":1,"file_task_id":"11111111-1111-1111-1111-111111111111"}}`,
		mustJSON(t, updated),
	), rec.Body.String())
}

func TestExecuteAutomaticPolicyMatchesRegisteredDeviceFirmware(t *testing.T) {
	deviceID, policyID := uuid.New(), uuid.New()
	workflow := &recordingFileWorkflow{}
	var created *ProvisioningTask
	h := NewHandler(&mockTaskRepo{
		CreateFn: func(_ context.Context, task *ProvisioningTask) error {
			copy := *task
			created = &copy
			return nil
		},
	}, nil)
	h.policyRepo = &executePolicyRepository{policies: []PlugAndPlayPolicy{
		{
			ID: policyID, Name: "auto-p1", Enabled: true, ExecuteType: "auto",
			ProductClass: "FAP/TEST", UpgradeEnabled: true, TargetVersion: "V2.0",
			Config: json.RawMessage(`{"originalVersion":["V1.0"]}`),
		},
	}}
	h.deviceService = &executePolicyDeviceService{devices: map[uuid.UUID]*model.Device{
		deviceID: {ID: deviceID, SerialNumber: "SN-AUTO", ProductClass: "FAP/TEST", FirmwareVersion: "V1.0"},
	}}
	h.SetPlugAndPlayFileWorkflow(workflow)

	require.NoError(t, h.ExecuteAutomaticPolicy(context.Background(), deviceID))
	assert.Equal(t, 1, workflow.upgradeCalls)
	require.NotNil(t, created)
	assert.Equal(t, policyID, *created.PolicyID)
	assert.Equal(t, 0, created.MaxRetries)
}

func TestExecuteAutomaticPolicyMatchesAnyConfiguredProductClass(t *testing.T) {
	deviceID, policyID := uuid.New(), uuid.New()
	workflow := &recordingFileWorkflow{}
	h := NewHandler(&mockTaskRepo{}, nil)
	h.policyRepo = &executePolicyRepository{policies: []PlugAndPlayPolicy{{
		ID: policyID, Name: "auto-multi", Enabled: true, ExecuteType: "auto",
		ProductClass: "FAP/OTHER", ProductClasses: []string{"FAP/OTHER", "FAP/TEST"},
		UpgradeEnabled: true, TargetVersion: "V2.0",
	}}}
	h.deviceService = &executePolicyDeviceService{devices: map[uuid.UUID]*model.Device{
		deviceID: {ID: deviceID, SerialNumber: "SN-MULTI", ProductClass: "FAP/TEST"},
	}}
	h.SetPlugAndPlayFileWorkflow(workflow)

	require.NoError(t, h.ExecuteAutomaticPolicy(context.Background(), deviceID))
	assert.Equal(t, 1, workflow.upgradeCalls)
}

func TestExecuteAutomaticPolicySkipsFirmwareMismatch(t *testing.T) {
	deviceID := uuid.New()
	workflow := &recordingFileWorkflow{}
	h := NewHandler(&mockTaskRepo{}, nil)
	h.policyRepo = &executePolicyRepository{policies: []PlugAndPlayPolicy{{
		ID: uuid.New(), Enabled: true, ExecuteType: "auto", ProductClass: "FAP/TEST",
		UpgradeEnabled: true, TargetVersion: "V2.0",
		Config: json.RawMessage(`{"originalVersion":["V0.9"]}`),
	}}}
	h.deviceService = &executePolicyDeviceService{devices: map[uuid.UUID]*model.Device{
		deviceID: {ID: deviceID, ProductClass: "FAP/TEST", FirmwareVersion: "V1.0"},
	}}
	h.SetPlugAndPlayFileWorkflow(workflow)

	require.NoError(t, h.ExecuteAutomaticPolicy(context.Background(), deviceID))
	assert.Zero(t, workflow.upgradeCalls)
}

func TestHandler_ExecutePolicyRejectsUpgradeWithoutTargetVersion(t *testing.T) {
	policyID, deviceID := uuid.New(), uuid.New()
	workflow := &recordingFileWorkflow{}
	h := NewHandler(&mockTaskRepo{}, nil)
	h.policyRepo = &executePolicyRepository{policy: &PlugAndPlayPolicy{
		ID: policyID, Name: "P1", Enabled: true, ProductClass: "FAP/TEST",
		UpgradeEnabled: true,
	}}
	h.deviceService = &executePolicyDeviceService{devices: map[uuid.UUID]*model.Device{
		deviceID: {ID: deviceID, SerialNumber: "SN-001", ProductClass: "FAP/TEST"},
	}}
	h.SetPlugAndPlayFileWorkflow(workflow)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: policyID.String()}}
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/provisioning/policies/"+policyID.String()+"/execute",
		strings.NewReader(fmt.Sprintf(`{"device_ids":[%q]}`, deviceID)),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExecutePolicy(c)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Zero(t, workflow.upgradeCalls)
	assert.Zero(t, workflow.licenseCalls)
}

func TestHandler_ExecutePolicyDefersLicenseUntilUpgradeCompletes(t *testing.T) {
	policyID, deviceID := uuid.New(), uuid.New()
	workflow := &recordingFileWorkflow{}
	var created, updated []*ProvisioningTask
	h := NewHandler(&mockTaskRepo{
		CreateFn: func(_ context.Context, task *ProvisioningTask) error {
			created = append(created, task)
			return nil
		},
		UpdateFn: func(_ context.Context, task *ProvisioningTask) error {
			updated = append(updated, task)
			return nil
		},
	}, nil)
	h.policyRepo = &executePolicyRepository{policy: &PlugAndPlayPolicy{
		ID: policyID, Name: "P1", Enabled: true, ProductClass: "FAP/TEST",
		UpgradeEnabled: true, TargetVersion: "V2.0", LicenseEnabled: true,
	}}
	h.deviceService = &executePolicyDeviceService{devices: map[uuid.UUID]*model.Device{
		deviceID: {ID: deviceID, SerialNumber: "SN-001", ProductClass: "FAP/TEST"},
	}}
	h.SetPlugAndPlayFileWorkflow(workflow)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: policyID.String()}}
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/provisioning/policies/"+policyID.String()+"/execute",
		strings.NewReader(fmt.Sprintf(`{"device_ids":[%q]}`, deviceID)),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExecutePolicy(c)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, 1, workflow.upgradeCalls)
	assert.Zero(t, workflow.licenseCalls)
	assert.Equal(t, []string{"software_upgrade"}, workflow.callOrder)
	require.Len(t, created, 1)
	require.Len(t, updated, 1)
	assert.Equal(t, "software_upgrade_task_submitted", updated[0].CurrentStepName)
	assert.Equal(t, StateConfiguring, updated[0].Status)

	require.NoError(t, h.ContinuePolicy(context.Background(), policyID, deviceID, policyModuleUpgrade))
	assert.Equal(t, 1, workflow.licenseCalls)
	assert.Equal(t, []string{"software_upgrade", "license"}, workflow.callOrder)
	require.Len(t, created, 2)
	require.Len(t, updated, 2)
	assert.Equal(t, "license_task_submitted", updated[1].CurrentStepName)
	assert.Equal(t, StateConfiguring, updated[1].Status)
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return string(raw)
}

func TestHandler_Create_DeviceExistencePrecheck(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name           string
		body           string
		checker        deviceExistenceChecker
		wantStatus     int
		wantTaskCreate bool // whether repo.Create should have been invoked
	}{
		{
			name: "device exists -> 201 and task created",
			body: fmt.Sprintf(`{"device_id":%q}`, existingID),
			checker: &stubDeviceChecker{GetFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
				return &model.Device{ID: id, SerialNumber: "SN-EXIST"}, nil
			}},
			wantStatus:     http.StatusCreated,
			wantTaskCreate: true,
		},
		{
			name: "device absent -> 404 and no orphan task",
			body: fmt.Sprintf(`{"device_id":%q}`, uuid.New()),
			checker: &stubDeviceChecker{GetFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
				return nil, nil // PgDeviceRepository returns (nil, nil) for missing row
			}},
			wantStatus:     http.StatusNotFound,
			wantTaskCreate: false,
		},
		{
			name: "device lookup ErrNotFound -> 404 and no orphan task",
			body: fmt.Sprintf(`{"device_id":%q}`, uuid.New()),
			checker: &stubDeviceChecker{GetFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
				return nil, commonerrors.ErrNotFound
			}},
			wantStatus:     http.StatusNotFound,
			wantTaskCreate: false,
		},
		{
			name: "device lookup internal error -> 500 and no orphan task",
			body: fmt.Sprintf(`{"device_id":%q}`, uuid.New()),
			checker: &stubDeviceChecker{GetFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
				return nil, errors.New("db down")
			}},
			wantStatus:     http.StatusInternalServerError,
			wantTaskCreate: false,
		},
		{
			name:           "missing device_id -> 400 (binding required)",
			body:           `{}`,
			checker:        &stubDeviceChecker{},
			wantStatus:     http.StatusBadRequest,
			wantTaskCreate: false,
		},
		{
			name:           "malformed device_id -> 400",
			body:           `{"device_id":"not-a-uuid"}`,
			checker:        &stubDeviceChecker{},
			wantStatus:     http.StatusBadRequest,
			wantTaskCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var created bool
			repo := &mockTaskRepo{
				CreateFn: func(_ context.Context, _ *ProvisioningTask) error {
					created = true
					return nil
				},
			}
			h := NewHandler(repo, nil)
			h.SetDeviceChecker(tt.checker)

			c, rec := newCreateRequest(t, tt.body)
			h.Create(c)

			assert.Equal(t, tt.wantStatus, rec.Code, "HTTP status")
			assert.Equal(t, tt.wantTaskCreate, created, "repo.Create invocation")

			// On the non-existent-device path, ensure the response is the
			// standard failure envelope (ret=0) and not a 201 success.
			if tt.wantStatus == http.StatusNotFound {
				var env struct {
					Ret int `json:"ret"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
				assert.Equal(t, 0, env.Ret, "404 body must be failure envelope")
			}
		})
	}
}

func TestHandler_RetryRejectsPlugAndPlayModuleTask(t *testing.T) {
	taskID, policyID, xmlID, deviceID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	failed := &ProvisioningTask{
		ID: taskID, DeviceID: deviceID, PolicyID: &policyID, XMLFileID: &xmlID,
		Status: StateFailed, RetryCount: 1, MaxRetries: 3,
	}
	repo := &mockTaskRepo{
		GetByIDFn: func(context.Context, uuid.UUID) (*ProvisioningTask, error) {
			return failed, nil
		},
		CreateFn: func(_ context.Context, task *ProvisioningTask) error {
			t.Fatal("plug-and-play task retry must not create a task")
			return nil
		},
	}
	h := NewHandler(repo, nil)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/provisioning/tasks/"+taskID.String()+"/retry", nil)
	c.Request.Host = "omc.example.test"

	h.Retry(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "start a new policy execution")
}

func TestAutomaticStartFileName(t *testing.T) {
	generatedAt := time.Date(2026, 7, 30, 9, 8, 7, 0, time.UTC)
	tests := []struct {
		name       string
		technology model.Technology
		want       string
	}{
		{name: "LTE", technology: model.TechLTE, want: "auto_start_SN_001.xml"},
		{name: "GSM", technology: model.TechGSM, want: "auto_start_SN_001.xml"},
		{name: "NR", technology: model.TechNR, want: "SN_001_BAI_CELLS_v1.7_20260730090807.xml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := automaticStartFileName(&model.Device{
				SerialNumber: "SN/001", Manufacturer: "BAI CELLS", Technology: tt.technology,
			}, "v1.7", generatedAt)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHandler_RetryRejectsMaximumRetryCount(t *testing.T) {
	taskID := uuid.New()
	repo := &mockTaskRepo{
		GetByIDFn: func(context.Context, uuid.UUID) (*ProvisioningTask, error) {
			return &ProvisioningTask{
				ID: taskID, Status: StateFailed, RetryCount: 3, MaxRetries: 3,
			}, nil
		},
		CreateFn: func(context.Context, *ProvisioningTask) error {
			t.Fatal("retry at the configured maximum must not create a task")
			return nil
		},
	}
	h := NewHandler(repo, nil)
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/provisioning/tasks/"+taskID.String()+"/retry", nil)

	h.Retry(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestHandler_Create_NilChecker confirms the precheck is skipped when no checker
// is wired (defensive: keeps the handler usable even if wiring is absent),
// preserving prior behavior in that degenerate case.
func TestHandler_Create_NilChecker(t *testing.T) {
	var created bool
	repo := &mockTaskRepo{
		CreateFn: func(_ context.Context, _ *ProvisioningTask) error {
			created = true
			return nil
		},
	}
	h := NewHandler(repo, nil) // no SetDeviceChecker

	c, rec := newCreateRequest(t, fmt.Sprintf(`{"device_id":%q}`, uuid.New()))
	h.Create(c)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, created, "without a checker the task is created (precheck skipped)")
}
