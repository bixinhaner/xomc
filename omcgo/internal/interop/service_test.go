package interop

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// --- Mocks for facade-level tests -------------------------------------------------

type mockCaseRunner struct {
	cases       map[TestCategory][]TestCase
	allResults  []TestResult
	allErr      error
	byCatResult map[TestCategory][]TestResult
	byCatErr    map[TestCategory]error
	calls       []string
}

func (m *mockCaseRunner) ListTestCases() map[TestCategory][]TestCase {
	m.calls = append(m.calls, "list")
	return m.cases
}

func (m *mockCaseRunner) RunAll(_ context.Context, deviceSN string) ([]TestResult, error) {
	m.calls = append(m.calls, "all:"+deviceSN)
	return m.allResults, m.allErr
}

func (m *mockCaseRunner) RunByCategory(_ context.Context, deviceSN string, cat TestCategory) ([]TestResult, error) {
	m.calls = append(m.calls, "cat:"+string(cat)+":"+deviceSN)
	if err, ok := m.byCatErr[cat]; ok {
		return nil, err
	}
	return m.byCatResult[cat], nil
}

type mockValidator struct {
	report *ValidationReport
	err    error
	calls  int
}

func (m *mockValidator) ValidateDevice(_ context.Context, _ uuid.UUID, _ model.CarrierCode, _ model.Technology) (*ValidationReport, error) {
	m.calls++
	return m.report, m.err
}

// --- Tests -----------------------------------------------------------------------

func TestService_ListCases(t *testing.T) {
	runner := &mockCaseRunner{
		cases: map[TestCategory][]TestCase{
			CategoryProtocol: {{ID: "p1", Name: "Inform parsing", Category: CategoryProtocol}},
		},
	}
	svc := NewService(runner, nil, zap.NewNop())

	got := svc.ListCases(context.Background())
	require.Len(t, got, 1)
	assert.Equal(t, "p1", got[CategoryProtocol][0].ID)
	assert.Contains(t, runner.calls, "list")
}

func TestService_RunCases_AllCategories(t *testing.T) {
	runner := &mockCaseRunner{
		allResults: []TestResult{
			{TestCaseID: "t1", Passed: true},
			{TestCaseID: "t2", Passed: false, Error: "boom"},
			{TestCaseID: "t3", Passed: true},
		},
	}
	svc := NewService(runner, nil, zap.NewNop())

	summary, err := svc.RunCases(context.Background(), "SN-1", nil)
	require.NoError(t, err)
	assert.Equal(t, "SN-1", summary.DeviceSN)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 2, summary.Passed)
	assert.Equal(t, 1, summary.Failed)
	assert.Equal(t, []string{"all:SN-1"}, runner.calls)
}

func TestService_RunCases_SpecificCategories(t *testing.T) {
	runner := &mockCaseRunner{
		byCatResult: map[TestCategory][]TestResult{
			CategoryProtocol:  {{TestCaseID: "p1", Passed: true}},
			CategoryDataModel: {{TestCaseID: "d1", Passed: false}, {TestCaseID: "d2", Passed: true}},
		},
		byCatErr: map[TestCategory]error{},
	}
	svc := NewService(runner, nil, zap.NewNop())

	summary, err := svc.RunCases(context.Background(), "SN-2", []TestCategory{CategoryProtocol, CategoryDataModel})
	require.NoError(t, err)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 2, summary.Passed)
	assert.Equal(t, 1, summary.Failed)
	// Each category should have triggered exactly one byCategory call.
	assert.Contains(t, runner.calls, "cat:protocol:SN-2")
	assert.Contains(t, runner.calls, "cat:datamodel:SN-2")
}

func TestService_RunCases_InvalidCategoryShortCircuits(t *testing.T) {
	runner := &mockCaseRunner{}
	svc := NewService(runner, nil, zap.NewNop())

	_, err := svc.RunCases(context.Background(), "SN-3", []TestCategory{"bogus"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid test category")
	// Runner must NOT have been touched if validation fails up front.
	assert.Empty(t, runner.calls)
}

func TestService_RunCases_EmptyDeviceSN(t *testing.T) {
	svc := NewService(&mockCaseRunner{}, nil, zap.NewNop())
	_, err := svc.RunCases(context.Background(), "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device_sn is required")
}

func TestService_RunCases_RunnerError(t *testing.T) {
	runner := &mockCaseRunner{allErr: errors.New("device offline")}
	svc := NewService(runner, nil, zap.NewNop())

	_, err := svc.RunCases(context.Background(), "SN-4", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device offline")
}

func TestService_RunByCategory(t *testing.T) {
	runner := &mockCaseRunner{
		byCatResult: map[TestCategory][]TestResult{
			CategoryRPC: {{TestCaseID: "r1", Passed: true}, {TestCaseID: "r2", Passed: true}},
		},
	}
	svc := NewService(runner, nil, zap.NewNop())

	summary, err := svc.RunByCategory(context.Background(), "SN-5", CategoryRPC)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 2, summary.Passed)
	assert.Equal(t, 0, summary.Failed)
}

func TestService_RunByCategory_InvalidCategory(t *testing.T) {
	svc := NewService(&mockCaseRunner{}, nil, zap.NewNop())
	_, err := svc.RunByCategory(context.Background(), "SN-6", "nonsense")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid test category")
}

func TestService_ValidateDevice_Success(t *testing.T) {
	val := &mockValidator{report: &ValidationReport{
		DeviceID: "abc", DeviceSN: "SN-7", TotalParams: 10, MatchedParams: 9, Score: 90.0,
	}}
	svc := NewService(nil, val, zap.NewNop())

	report, err := svc.ValidateDevice(context.Background(), uuid.New(), model.CarrierCMCC, model.TechLTE)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "SN-7", report.DeviceSN)
	assert.Equal(t, 1, val.calls)
}

func TestService_ValidateDevice_InvalidCarrier(t *testing.T) {
	val := &mockValidator{}
	svc := NewService(nil, val, zap.NewNop())

	_, err := svc.ValidateDevice(context.Background(), uuid.New(), model.CarrierCode("xxxx"), model.TechLTE)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid carrier")
	assert.Equal(t, 0, val.calls, "validator must not be called when input is invalid")
}

func TestService_ValidateDevice_ValidatorError(t *testing.T) {
	val := &mockValidator{err: errors.New("data model not found")}
	svc := NewService(nil, val, zap.NewNop())

	_, err := svc.ValidateDevice(context.Background(), uuid.New(), model.CarrierCMCC, model.TechNR)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data model not found")
}

func TestSummarize_PureFunction(t *testing.T) {
	results := []TestResult{
		{Passed: true}, {Passed: false}, {Passed: true}, {Passed: true},
	}
	got := summarize("SN-X", results)
	assert.Equal(t, "SN-X", got.DeviceSN)
	assert.Equal(t, 4, got.Total)
	assert.Equal(t, 3, got.Passed)
	assert.Equal(t, 1, got.Failed)
}

func TestService_NilDependenciesGuarded(t *testing.T) {
	svc := NewService(nil, nil, nil) // also exercises nil-logger fallback

	// ListCases on nil runner returns empty map, never panics.
	cases := svc.ListCases(context.Background())
	assert.Empty(t, cases)

	_, err := svc.RunCases(context.Background(), "SN-Z", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "case runner not configured")

	_, err = svc.ValidateDevice(context.Background(), uuid.New(), model.CarrierCMCC, model.TechLTE)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model validator not configured")
}
