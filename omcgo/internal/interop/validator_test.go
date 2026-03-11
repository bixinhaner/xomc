package interop

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/model"
	"github.com/omcgo/omcgo/internal/config/datamodel"
)

// newTestDataModel creates a DataModel with a parameter tree for testing.
func newTestDataModel(t *testing.T) *datamodel.DataModel {
	t.Helper()

	params := []datamodel.Parameter{
		{Path: "Device.DeviceInfo.Manufacturer", Type: "string", Writable: false},
		{Path: "Device.DeviceInfo.ModelName", Type: "string", Writable: false},
		{Path: "Device.DeviceInfo.SerialNumber", Type: "string", Writable: false},
		{Path: "Device.ManagementServer.PeriodicInformInterval", Type: "unsignedInt", Writable: true},
		{Path: "Device.ManagementServer.ConnectionRequestURL", Type: "string", Writable: false},
	}

	tree, err := json.Marshal(params)
	require.NoError(t, err)

	return &datamodel.DataModel{
		ID:            uuid.New(),
		Carrier:       model.CarrierCMCC,
		Technology:    model.TechLTE,
		Version:       "v1.0.0",
		OUI:           "00AABB",
		ProductClass:  "SmallCell-LTE",
		Scope:         model.ScopeProduct,
		Status:        datamodel.StatusActive,
		IsActive:      true,
		RootObject:    "Device.",
		ParameterTree: tree,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// mockDataModelRegistry wraps a single test DataModel and implements the
// resolution interface expected by the validator. Because DataModelRegistry is a
// concrete struct (not an interface), we test the validator indirectly by
// constructing a real registry with a mock repository.
type mockDataModelRepo struct {
	dm *datamodel.DataModel
}

func (m *mockDataModelRepo) Create(_ context.Context, _ *datamodel.DataModel) error { return nil }
func (m *mockDataModelRepo) GetByID(_ context.Context, id uuid.UUID) (*datamodel.DataModel, error) {
	if m.dm != nil && m.dm.ID == id {
		return m.dm, nil
	}
	return nil, nil
}
func (m *mockDataModelRepo) Update(_ context.Context, _ *datamodel.DataModel) error { return nil }
func (m *mockDataModelRepo) Delete(_ context.Context, _ uuid.UUID) error             { return nil }
func (m *mockDataModelRepo) List(_ context.Context, _ datamodel.DataModelFilter) (*model.ListResponse[datamodel.DataModel], error) {
	return model.NewListResponse([]datamodel.DataModel{}, 0, 1, 20), nil
}
func (m *mockDataModelRepo) FindActive(_ context.Context, carrier model.CarrierCode, tech model.Technology,
	oui, productClass string, scope model.DataModelScope) (*datamodel.DataModel, error) {
	if m.dm == nil {
		return nil, nil
	}
	// Return the model if scope matches or as carrier_default fallback.
	if scope == m.dm.Scope {
		return m.dm, nil
	}
	return nil, nil
}
func (m *mockDataModelRepo) Activate(_ context.Context, _ uuid.UUID) error    { return nil }
func (m *mockDataModelRepo) Deprecate(_ context.Context, _ uuid.UUID) error   { return nil }
func (m *mockDataModelRepo) Statistics(_ context.Context) (*datamodel.DataModelStats, error) {
	return &datamodel.DataModelStats{}, nil
}

func newTestValidator(t *testing.T, dev *model.Device, dm *datamodel.DataModel, deviceParams []model.DeviceParameter) *DataModelValidator {
	t.Helper()

	logger := zap.NewNop()

	dmRepo := &mockDataModelRepo{dm: dm}
	// Pass nil for cache; the registry has nil-cache guards and will fall
	// through directly to the DB (mock repo) on every resolve call.
	dmRegistry := datamodel.NewDataModelRegistry(dmRepo, nil, logger)

	devRepo := newMockDeviceRepo(dev)
	paramRepo := newMockParamRepo()
	if deviceParams != nil {
		paramRepo.params[dev.ID] = deviceParams
	}

	return NewDataModelValidator(dmRegistry, paramRepo, devRepo, logger)
}

func TestValidateDevice_AllMatch(t *testing.T) {
	dev := newTestDevice()
	dm := newTestDataModel(t)

	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.PeriodicInformInterval", ParameterType: model.ParamUint, Writable: true},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.ConnectionRequestURL", ParameterType: model.ParamString, Writable: false},
	}

	validator := newTestValidator(t, dev, dm, deviceParams)

	report, err := validator.ValidateDevice(context.Background(), dev.ID, dev.Carrier, dev.Technology)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, 5, report.TotalParams)
	assert.Equal(t, 5, report.MatchedParams)
	assert.Empty(t, report.MissingParams)
	assert.Empty(t, report.MismatchParams)
	assert.Empty(t, report.ExtraParams)
	assert.Equal(t, 100.0, report.Score)
}

func TestValidateDevice_MissingParams(t *testing.T) {
	dev := newTestDevice()
	dm := newTestDataModel(t)

	// Only provide 2 of 5 expected parameters.
	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
	}

	validator := newTestValidator(t, dev, dm, deviceParams)

	report, err := validator.ValidateDevice(context.Background(), dev.ID, dev.Carrier, dev.Technology)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, 5, report.TotalParams)
	assert.Equal(t, 2, report.MatchedParams)
	assert.Len(t, report.MissingParams, 3)
	assert.Empty(t, report.MismatchParams)
}

func TestValidateDevice_TypeMismatch(t *testing.T) {
	dev := newTestDevice()
	dm := newTestDataModel(t)

	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterType: model.ParamString, Writable: false},
		// Type mismatch: expected unsignedInt, got string
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.PeriodicInformInterval", ParameterType: model.ParamString, Writable: true},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.ConnectionRequestURL", ParameterType: model.ParamString, Writable: false},
	}

	validator := newTestValidator(t, dev, dm, deviceParams)

	report, err := validator.ValidateDevice(context.Background(), dev.ID, dev.Carrier, dev.Technology)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, 5, report.TotalParams)
	assert.Len(t, report.MismatchParams, 1)
	assert.Equal(t, "Device.ManagementServer.PeriodicInformInterval", report.MismatchParams[0].Path)
	assert.Equal(t, "type_mismatch", report.MismatchParams[0].Type)
}

func TestValidateDevice_ExtraParams(t *testing.T) {
	dev := newTestDevice()
	dm := newTestDataModel(t)

	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.PeriodicInformInterval", ParameterType: model.ParamUint, Writable: true},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.ConnectionRequestURL", ParameterType: model.ParamString, Writable: false},
		// Extra params not in the data model.
		{DeviceID: dev.ID, ParameterPath: "Device.Vendor.X_CUSTOM_PARAM", ParameterType: model.ParamString, Writable: true},
	}

	validator := newTestValidator(t, dev, dm, deviceParams)

	report, err := validator.ValidateDevice(context.Background(), dev.ID, dev.Carrier, dev.Technology)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, 5, report.MatchedParams)
	assert.Len(t, report.ExtraParams, 1)
	assert.Contains(t, report.ExtraParams, "Device.Vendor.X_CUSTOM_PARAM")
}

func TestValidateDevice_WritableMismatch(t *testing.T) {
	dev := newTestDevice()
	dm := newTestDataModel(t)

	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterType: model.ParamString, Writable: false},
		// Writable mismatch: expected writable=true, got writable=false
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.PeriodicInformInterval", ParameterType: model.ParamUint, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.ConnectionRequestURL", ParameterType: model.ParamString, Writable: false},
	}

	validator := newTestValidator(t, dev, dm, deviceParams)

	report, err := validator.ValidateDevice(context.Background(), dev.ID, dev.Carrier, dev.Technology)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Len(t, report.MismatchParams, 1)
	assert.Equal(t, "writable_mismatch", report.MismatchParams[0].Type)
}

func TestValidateDevice_NoDataModel(t *testing.T) {
	dev := newTestDevice()

	validator := newTestValidator(t, dev, nil, nil)

	_, err := validator.ValidateDevice(context.Background(), dev.ID, dev.Carrier, dev.Technology)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data model found")
}
