package interop

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
)

// --- Mock DeviceRepository ---

type mockDeviceRepo struct {
	devices map[string]*model.Device
}

func newMockDeviceRepo(devs ...*model.Device) *mockDeviceRepo {
	m := &mockDeviceRepo{devices: make(map[string]*model.Device)}
	for _, d := range devs {
		m.devices[d.SerialNumber] = d
	}
	return m
}

func (m *mockDeviceRepo) Create(_ context.Context, _ *model.Device) error { return nil }
func (m *mockDeviceRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Device, error) {
	for _, d := range m.devices {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, fmt.Errorf("not found: %w", commonerrors.ErrNotFound)
}
func (m *mockDeviceRepo) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	d, ok := m.devices[sn]
	if !ok {
		return nil, fmt.Errorf("not found: %w", commonerrors.ErrNotFound)
	}
	return d, nil
}
func (m *mockDeviceRepo) Update(_ context.Context, _ *model.Device) error { return nil }
func (m *mockDeviceRepo) Delete(_ context.Context, _ uuid.UUID) error     { return nil }
func (m *mockDeviceRepo) List(_ context.Context, _ device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.DeviceStatus) error {
	return nil
}
func (m *mockDeviceRepo) UpdateLastInform(_ context.Context, _ string, _ time.Time, _ []string) error {
	return nil
}
func (m *mockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockDeviceRepo) CountByStatus(_ context.Context, _ *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return []model.Device{}, nil
}
func (m *mockDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *mockDeviceRepo) GetGeoStats(_ context.Context, _ []string) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *mockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *mockDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

// --- Mock DeviceParameterRepository ---

type mockParamRepo struct {
	params map[uuid.UUID][]model.DeviceParameter
}

func newMockParamRepo() *mockParamRepo {
	return &mockParamRepo{params: make(map[uuid.UUID][]model.DeviceParameter)}
}

func (m *mockParamRepo) BatchUpsert(_ context.Context, id uuid.UUID, params []model.DeviceParameter) error {
	m.params[id] = params
	return nil
}
func (m *mockParamRepo) GetByDevice(_ context.Context, id uuid.UUID) ([]model.DeviceParameter, error) {
	return m.params[id], nil
}
func (m *mockParamRepo) GetByPath(_ context.Context, id uuid.UUID, path string) (*model.DeviceParameter, error) {
	for _, p := range m.params[id] {
		if p.ParameterPath == path {
			return &p, nil
		}
	}
	return nil, nil
}
func (m *mockParamRepo) DeleteByDevice(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockParamRepo) GetByPathPrefix(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) CountByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int, error) {
	return 0, nil
}
func (m *mockParamRepo) SearchByKeyword(_ context.Context, _ uuid.UUID, _ string, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) GetDirectChildLeaves(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}

func (m *mockParamRepo) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

// --- Mock task.Enqueuer ---

type mockCmdQueue struct {
	queues map[string][]*task.Task
}

func newMockCmdQueue() *mockCmdQueue {
	return &mockCmdQueue{queues: make(map[string][]*task.Task)}
}

func (m *mockCmdQueue) CreateTask(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	t := task.NewTask(req)
	m.queues[req.DeviceSN] = append(m.queues[req.DeviceSN], t)
	return t, nil
}

func (m *mockCmdQueue) GetQueueLength(_ context.Context, deviceSN string) (int64, error) {
	return int64(len(m.queues[deviceSN])), nil
}

// --- Tests ---

func newTestDevice() *model.Device {
	now := time.Now()
	return &model.Device{
		ID:                   uuid.New(),
		SerialNumber:         "TEST-SN-001",
		OUI:                  "00AABB",
		ProductClass:         "SmallCell-LTE",
		Manufacturer:         "TestVendor",
		ModelName:            "SC-100",
		Carrier:              model.CarrierCMCC,
		Technology:           model.TechLTE,
		Status:               model.DeviceActive,
		FirmwareVersion:      "v2.1.0",
		IPAddress:            "192.168.1.100",
		ConnectionRequestURL: "http://192.168.1.100:7547/connreq",
		LastInformAt:         &now,
		LastInformEvents:     []string{"2 PERIODIC"},
		InformInterval:       300,
	}
}

// testProtocolCases returns protocol test cases inline to avoid import cycle with cases package.
func testProtocolCases() []TestCase {
	return []TestCase{
		{
			ID: "PROTO-001", Name: "Inform Required Fields", Category: CategoryProtocol,
			Steps: []TestStep{
				{Order: 1, Action: "check_param", Params: map[string]interface{}{"field": "serial_number", "check": "not_empty"}},
				{Order: 2, Action: "check_param", Params: map[string]interface{}{"fields": []string{"oui", "manufacturer", "product_class"}, "check": "not_empty"}},
			},
		},
		{
			ID: "PROTO-002", Name: "Event Code Presence", Category: CategoryProtocol,
			Steps: []TestStep{
				{Order: 1, Action: "check_param", Params: map[string]interface{}{"field": "last_inform_events", "check": "not_empty"}},
			},
		},
		{
			ID: "PROTO-003", Name: "Session Parameters", Category: CategoryProtocol,
			Steps: []TestStep{
				{Order: 1, Action: "check_param", Params: map[string]interface{}{"field": "connection_request_url", "check": "not_empty"}},
				{Order: 2, Action: "check_param", Params: map[string]interface{}{"field": "inform_interval", "check": "positive"}},
			},
		},
	}
}

// testRPCCases returns RPC test cases inline to avoid import cycle.
func testRPCCases() []TestCase {
	methods := []string{
		"GetParameterValues", "SetParameterValues", "GetParameterNames",
		"SetParameterAttributes", "GetParameterAttributes", "AddObject",
		"DeleteObject", "Download", "Reboot",
	}
	var tc []TestCase
	for i, m := range methods {
		tc = append(tc, TestCase{
			ID: fmt.Sprintf("RPC-%03d", i+1), Name: m, Category: CategoryRPC,
			Steps: []TestStep{
				{Order: 1, Action: "send_rpc", Params: map[string]interface{}{"method": m}},
				{Order: 2, Action: "verify_response", Params: map[string]interface{}{"check": "command_queued"}},
			},
		})
	}
	return tc
}

func newTestRunner(devRepo *mockDeviceRepo, paramRepo *mockParamRepo, cmdQ *mockCmdQueue) *ConformanceTestRunner {
	logger := zap.NewNop()
	// T-0098 P5-01：paramRegistry / productRegistry 在单测中传 nil；
	// 期望参数集解析失败时 checkParam* 报错，对应 TestRun_PathPresent_NoModel 路径。
	runner := NewConformanceTestRunner(devRepo, paramRepo, nil, nil, cmdQ, logger)
	runner.RegisterCases(testProtocolCases())
	runner.RegisterCases(testRPCCases())
	return runner
}

func TestListTestCases(t *testing.T) {
	devRepo := newMockDeviceRepo(newTestDevice())
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	runner := newTestRunner(devRepo, paramRepo, cmdQ)

	tc := runner.ListTestCases()

	assert.Contains(t, tc, CategoryProtocol, "should contain protocol category")
	assert.Contains(t, tc, CategoryRPC, "should contain rpc category")
	assert.Len(t, tc[CategoryProtocol], 3, "should have 3 protocol test cases")
	assert.Len(t, tc[CategoryRPC], 9, "should have 9 RPC test cases")
}

func TestRunAll_ProtocolAndRPC(t *testing.T) {
	dev := newTestDevice()
	devRepo := newMockDeviceRepo(dev)
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	runner := newTestRunner(devRepo, paramRepo, cmdQ)

	results, err := runner.RunAll(context.Background(), dev.SerialNumber)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// All protocol tests should pass for our well-formed device.
	for _, r := range results {
		if r.Category == CategoryProtocol {
			assert.True(t, r.Passed, "protocol test %s should pass: %s", r.TestCaseID, r.Error)
		}
	}
}

func TestRunByCategory_Protocol(t *testing.T) {
	dev := newTestDevice()
	devRepo := newMockDeviceRepo(dev)
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	runner := newTestRunner(devRepo, paramRepo, cmdQ)

	results, err := runner.RunByCategory(context.Background(), dev.SerialNumber, CategoryProtocol)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	for _, r := range results {
		assert.Equal(t, CategoryProtocol, r.Category)
		assert.True(t, r.Passed, "test %s should pass: %s", r.TestCaseID, r.Error)
	}
}

func TestRunByCategory_RPC(t *testing.T) {
	dev := newTestDevice()
	devRepo := newMockDeviceRepo(dev)
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	runner := newTestRunner(devRepo, paramRepo, cmdQ)

	results, err := runner.RunByCategory(context.Background(), dev.SerialNumber, CategoryRPC)
	require.NoError(t, err)
	assert.Len(t, results, 9)

	// All RPC tests should pass because our mock queue accepts pushes.
	for _, r := range results {
		assert.Equal(t, CategoryRPC, r.Category)
		assert.True(t, r.Passed, "RPC test %s should pass: %s", r.TestCaseID, r.Error)
	}

	// Verify commands were actually enqueued.
	qLen, err := cmdQ.GetQueueLength(context.Background(), dev.SerialNumber)
	require.NoError(t, err)
	assert.Equal(t, int64(9), qLen, "should have 9 commands in the queue")
}

func TestRunByCategory_InvalidCategory(t *testing.T) {
	devRepo := newMockDeviceRepo(newTestDevice())
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	runner := newTestRunner(devRepo, paramRepo, cmdQ)

	_, err := runner.RunByCategory(context.Background(), "TEST-SN-001", TestCategory("invalid"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid test category")
}

func TestRunAll_DeviceNotFound(t *testing.T) {
	devRepo := newMockDeviceRepo() // empty
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	runner := newTestRunner(devRepo, paramRepo, cmdQ)

	_, err := runner.RunAll(context.Background(), "NONEXISTENT")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lookup device")
}

func TestRunAll_ProtocolFailsOnEmptyField(t *testing.T) {
	dev := newTestDevice()
	dev.OUI = "" // Make Inform required fields test fail
	devRepo := newMockDeviceRepo(dev)
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	runner := newTestRunner(devRepo, paramRepo, cmdQ)

	results, err := runner.RunAll(context.Background(), dev.SerialNumber)
	require.NoError(t, err)

	// Find the PROTO-001 result (Inform Required Fields) - should fail due to empty OUI.
	var proto001 *TestResult
	for i, r := range results {
		if r.TestCaseID == "PROTO-001" {
			proto001 = &results[i]
			break
		}
	}

	require.NotNil(t, proto001, "should find PROTO-001 result")
	assert.False(t, proto001.Passed, "PROTO-001 should fail when OUI is empty")
	assert.Contains(t, proto001.Error, "oui")
}

func TestGetDeviceField(t *testing.T) {
	dev := newTestDevice()

	tests := []struct {
		field    string
		expected string
	}{
		{"serial_number", dev.SerialNumber},
		{"oui", dev.OUI},
		{"manufacturer", dev.Manufacturer},
		{"product_class", dev.ProductClass},
		{"firmware_version", dev.FirmwareVersion},
		{"ip_address", dev.IPAddress},
		{"connection_request_url", dev.ConnectionRequestURL},
		{"last_inform_events", "present"},
		{"unknown_field", ""},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := getDeviceField(dev, tt.field)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestTestCategory_IsValid(t *testing.T) {
	tests := []struct {
		cat   TestCategory
		valid bool
	}{
		{CategoryProtocol, true},
		{CategoryDataModel, true},
		{CategoryRPC, true},
		{CategoryInform, true},
		{TestCategory("invalid"), false},
		{TestCategory(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.cat), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.cat.IsValid())
		})
	}
}

// TestTestResult_JSON ensures JSON serialization works correctly.
func TestTestResult_JSON(t *testing.T) {
	result := TestResult{
		TestCaseID: "PROTO-001",
		TestName:   "Inform Format Validation",
		Category:   CategoryProtocol,
		Passed:     true,
		Duration:   5 * time.Millisecond,
		Details:    "All steps passed",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"test_case_id":"PROTO-001"`)
	assert.Contains(t, string(data), `"passed":true`)
}

// TestRunTestCase_NegativePath_ExpectedFailureObserved verifies that a TestCase
// with ExpectedOutcome="fail" returns Passed=true when a step actually fails,
// and Details surface the original error for downstream inspection.
//
// T-0030 V3: negative path 用例 — 验收 §3 V3 "失败路径可识别"。
func TestRunTestCase_NegativePath_ExpectedFailureObserved(t *testing.T) {
	devRepo := newMockDeviceRepo(newTestDevice())
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	logger := zap.NewNop()
	runner := NewConformanceTestRunner(devRepo, paramRepo, nil, nil, cmdQ, logger)

	negCase := TestCase{
		ID: "NEG-001", Name: "Negative — Unknown Check", Category: CategoryProtocol,
		ExpectedOutcome: "fail",
		Steps: []TestStep{
			{Order: 1, Action: "check_param", Params: map[string]interface{}{"check": "no_such_check_type"}},
		},
	}
	runner.RegisterCases([]TestCase{negCase})

	results, err := runner.RunByCategory(context.Background(), "TEST-SN-001", CategoryProtocol)
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.True(t, r.Passed, "negative case should be marked passed when step fails as expected")
	assert.Empty(t, r.Error, "Error should be cleared after expected failure")
	assert.Contains(t, r.Details, "Expected failure observed")
	assert.Contains(t, r.Details, "unknown check type")
	assert.Equal(t, "fail", r.ExpectedOutcome)
}

// TestRunTestCase_NegativePath_UnexpectedPassFails verifies that a TestCase
// with ExpectedOutcome="fail" is marked failed when all steps unexpectedly pass.
func TestRunTestCase_NegativePath_UnexpectedPassFails(t *testing.T) {
	devRepo := newMockDeviceRepo(newTestDevice())
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	logger := zap.NewNop()
	runner := NewConformanceTestRunner(devRepo, paramRepo, nil, nil, cmdQ, logger)

	negCase := TestCase{
		ID: "NEG-002", Name: "Negative — All Steps Pass", Category: CategoryProtocol,
		ExpectedOutcome: "fail",
		Steps: []TestStep{
			{Order: 1, Action: "check_param", Params: map[string]interface{}{"field": "serial_number", "check": "not_empty"}},
		},
	}
	runner.RegisterCases([]TestCase{negCase})

	results, err := runner.RunByCategory(context.Background(), "TEST-SN-001", CategoryProtocol)
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.False(t, r.Passed, "negative case should be marked failed when no step actually fails")
	assert.Contains(t, r.Error, "negative case unexpectedly passed")
	assert.Equal(t, "fail", r.ExpectedOutcome)
}

// TestRunTestCase_DefaultPassSemanticsUnchanged verifies that omitting
// ExpectedOutcome preserves the pre-T-0030 behavior (pass = all steps pass).
func TestRunTestCase_DefaultPassSemanticsUnchanged(t *testing.T) {
	devRepo := newMockDeviceRepo(newTestDevice())
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	logger := zap.NewNop()
	runner := NewConformanceTestRunner(devRepo, paramRepo, nil, nil, cmdQ, logger)
	runner.RegisterCases(testProtocolCases())

	results, err := runner.RunByCategory(context.Background(), "TEST-SN-001", CategoryProtocol)
	require.NoError(t, err)
	require.Len(t, results, 3)
	for _, r := range results {
		assert.True(t, r.Passed)
		assert.Empty(t, r.ExpectedOutcome, "default cases should have empty ExpectedOutcome")
	}
}

// TestRunner_FiltersByDeviceModel verifies TargetDeviceModels gating (T-0116).
// Cases with no filter run against any device; cases with non-empty filter
// run only when device.ModelName matches one of the listed values.
func TestRunner_FiltersByDeviceModel(t *testing.T) {
	devRepo := newMockDeviceRepo(newTestDevice()) // ModelName = "SC-100"
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	logger := zap.NewNop()

	tests := []struct {
		name              string
		targetModels      []string
		expectExecuted    bool
		runnerExpectCount int
	}{
		{"no filter (nil) runs", nil, true, 1},
		{"empty filter slice runs", []string{}, true, 1},
		{"matching model runs", []string{"SC-100"}, true, 1},
		{"matching among multiple runs", []string{"OTHER", "SC-100", "XYZ"}, true, 1},
		{"non-matching model skipped", []string{"OTHER-MODEL"}, false, 0},
		{"multiple non-matching skipped", []string{"OTHER", "ANOTHER"}, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewConformanceTestRunner(devRepo, paramRepo, nil, nil, cmdQ, logger)
			runner.RegisterCases([]TestCase{
				{
					ID: "MODEL-FILTER-001", Name: "Model filter probe", Category: CategoryProtocol,
					TargetDeviceModels: tt.targetModels,
					Steps: []TestStep{
						{Order: 1, Action: "check_param", Params: map[string]interface{}{"field": "serial_number", "check": "not_empty"}},
					},
				},
			})

			results, err := runner.RunByCategory(context.Background(), "TEST-SN-001", CategoryProtocol)
			require.NoError(t, err)
			assert.Len(t, results, tt.runnerExpectCount, "unexpected result count for %s", tt.name)
			if tt.expectExecuted {
				assert.True(t, results[0].Passed)
			}
		})
	}
}

// TestRunner_FiltersByDeviceModel_RunAll verifies the same filter applies in
// RunAll (across all categories).
func TestRunner_FiltersByDeviceModel_RunAll(t *testing.T) {
	devRepo := newMockDeviceRepo(newTestDevice()) // ModelName = "SC-100"
	paramRepo := newMockParamRepo()
	cmdQ := newMockCmdQueue()
	logger := zap.NewNop()
	runner := NewConformanceTestRunner(devRepo, paramRepo, nil, nil, cmdQ, logger)

	// Register two cases: one universal, one scoped to a non-matching model.
	runner.RegisterCases([]TestCase{
		{
			ID: "UNIV-001", Name: "Universal", Category: CategoryProtocol,
			Steps: []TestStep{{Order: 1, Action: "check_param", Params: map[string]interface{}{"field": "serial_number", "check": "not_empty"}}},
		},
		{
			ID: "SCOPED-001", Name: "Scoped to OTHER-MODEL", Category: CategoryProtocol,
			TargetDeviceModels: []string{"OTHER-MODEL"},
			Steps:              []TestStep{{Order: 1, Action: "check_param", Params: map[string]interface{}{"field": "serial_number", "check": "not_empty"}}},
		},
	})

	results, err := runner.RunAll(context.Background(), "TEST-SN-001")
	require.NoError(t, err)
	require.Len(t, results, 1, "only universal case should execute (scoped one filtered out)")
	assert.Equal(t, "UNIV-001", results[0].TestCaseID)
}
