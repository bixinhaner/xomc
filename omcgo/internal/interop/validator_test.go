package interop

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// T-0098 P5-01：旧 datamodel.DataModelRegistry 路径已删除。本组测试不再 mock
// parammodel.Registry / product.Registry（它们带 pgxpool/redis 依赖，单测内
// 难以构造），改为直接调用包内 compare() 验证比对核心逻辑。

func newDirectValidator(t *testing.T, dev *model.Device, deviceParams []model.DeviceParameter) (*DataModelValidator, *mockDeviceRepo, *mockParamRepo) {
	t.Helper()

	devRepo := newMockDeviceRepo(dev)
	paramRepo := newMockParamRepo()
	if deviceParams != nil {
		paramRepo.params[dev.ID] = deviceParams
	}
	v := NewDataModelValidator(nil, nil, paramRepo, devRepo, zap.NewNop())
	return v, devRepo, paramRepo
}

func defaultExpected() []validatorExpectedParam {
	return []validatorExpectedParam{
		{Path: "Device.DeviceInfo.Manufacturer", Type: "string", Writable: false},
		{Path: "Device.DeviceInfo.ModelName", Type: "string", Writable: false},
		{Path: "Device.DeviceInfo.SerialNumber", Type: "string", Writable: false},
		{Path: "Device.ManagementServer.PeriodicInformInterval", Type: "unsignedInt", Writable: true},
		{Path: "Device.ManagementServer.ConnectionRequestURL", Type: "string", Writable: false},
	}
}

func TestCompare_AllMatch(t *testing.T) {
	dev := newTestDevice()

	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.PeriodicInformInterval", ParameterType: model.ParamUint, Writable: true},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.ConnectionRequestURL", ParameterType: model.ParamString, Writable: false},
	}

	v, _, _ := newDirectValidator(t, dev, deviceParams)
	report, err := v.compare(context.Background(), dev.ID, dev, defaultExpected(), "test-v1")
	require.NoError(t, err)

	assert.Equal(t, 5, report.TotalParams)
	assert.Equal(t, 5, report.MatchedParams)
	assert.Empty(t, report.MissingParams)
	assert.Empty(t, report.MismatchParams)
	assert.Empty(t, report.ExtraParams)
	assert.Equal(t, 100.0, report.Score)
}

func TestCompare_MissingParams(t *testing.T) {
	dev := newTestDevice()
	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
	}

	v, _, _ := newDirectValidator(t, dev, deviceParams)
	report, err := v.compare(context.Background(), dev.ID, dev, defaultExpected(), "test-v1")
	require.NoError(t, err)

	assert.Equal(t, 5, report.TotalParams)
	assert.Equal(t, 2, report.MatchedParams)
	assert.Len(t, report.MissingParams, 3)
	assert.Empty(t, report.MismatchParams)
}

func TestCompare_TypeMismatch(t *testing.T) {
	dev := newTestDevice()
	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterType: model.ParamString, Writable: false},
		// 类型错配：期望 unsignedInt，实际 string
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.PeriodicInformInterval", ParameterType: model.ParamString, Writable: true},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.ConnectionRequestURL", ParameterType: model.ParamString, Writable: false},
	}

	v, _, _ := newDirectValidator(t, dev, deviceParams)
	report, err := v.compare(context.Background(), dev.ID, dev, defaultExpected(), "test-v1")
	require.NoError(t, err)

	assert.Equal(t, 5, report.TotalParams)
	assert.Len(t, report.MismatchParams, 1)
	assert.Equal(t, "Device.ManagementServer.PeriodicInformInterval", report.MismatchParams[0].Path)
	assert.Equal(t, "type_mismatch", report.MismatchParams[0].Type)
}

func TestCompare_ExtraParams(t *testing.T) {
	dev := newTestDevice()
	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.PeriodicInformInterval", ParameterType: model.ParamUint, Writable: true},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.ConnectionRequestURL", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.Vendor.X_CUSTOM_PARAM", ParameterType: model.ParamString, Writable: true},
	}

	v, _, _ := newDirectValidator(t, dev, deviceParams)
	report, err := v.compare(context.Background(), dev.ID, dev, defaultExpected(), "test-v1")
	require.NoError(t, err)

	assert.Equal(t, 5, report.MatchedParams)
	assert.Len(t, report.ExtraParams, 1)
	assert.Contains(t, report.ExtraParams, "Device.Vendor.X_CUSTOM_PARAM")
}

func TestCompare_WritableMismatch(t *testing.T) {
	dev := newTestDevice()
	deviceParams := []model.DeviceParameter{
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.Manufacturer", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.ModelName", ParameterType: model.ParamString, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterType: model.ParamString, Writable: false},
		// writable 错配：期望 true，实际 false
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.PeriodicInformInterval", ParameterType: model.ParamUint, Writable: false},
		{DeviceID: dev.ID, ParameterPath: "Device.ManagementServer.ConnectionRequestURL", ParameterType: model.ParamString, Writable: false},
	}

	v, _, _ := newDirectValidator(t, dev, deviceParams)
	report, err := v.compare(context.Background(), dev.ID, dev, defaultExpected(), "test-v1")
	require.NoError(t, err)

	assert.Len(t, report.MismatchParams, 1)
	assert.Equal(t, "writable_mismatch", report.MismatchParams[0].Type)
}

func TestValidateDevice_NoRegistry(t *testing.T) {
	dev := newTestDevice()
	v, _, _ := newDirectValidator(t, dev, nil)

	_, err := v.ValidateDevice(context.Background(), dev.ID, dev.Carrier, dev.Technology)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "param registry not configured")
}

// TestValidateDevice_DeviceNotFound is the #125 regression: deviceRepo.GetByID
// returns (nil, nil) for an unknown-but-well-formed UUID (production
// PgDeviceRepository folds pgx.ErrNoRows to nil,nil). ValidateDevice must
// short-circuit with an ErrNotFound-wrapped error so the handler yields 404,
// rather than falling into resolveExpectedParams and surfacing the misleading
// "device productClass missing" as a 500.
func TestValidateDevice_DeviceNotFound(t *testing.T) {
	v := NewDataModelValidator(nil, nil, newMockParamRepo(), newNilDeviceRepo(), zap.NewNop())

	_, err := v.ValidateDevice(context.Background(), uuid.New(), model.CarrierCMCC, model.TechLTE)
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrNotFound,
		"missing device must wrap ErrNotFound so handler maps 404")
	assert.Equal(t, http.StatusNotFound, commonerrors.HTTPStatusFromError(err))
	assert.NotContains(t, err.Error(), "productClass missing",
		"must not surface the misleading downstream error")
}
