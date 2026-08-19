package northbound

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/northbound/pageconfig"
	"github.com/stretchr/testify/require"
)

func TestLegacyAPIUserPayloadOmitsPassword(t *testing.T) {
	payload := legacyAPIUserPayload(pageconfig.APIUser{
		ID:          "client-1",
		Username:    "oss",
		Password:    "plain-secret",
		PasswordSet: true,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, "46000")

	require.NotContains(t, payload, "password")
	require.NotContains(t, payload, "userPwd")
	require.Equal(t, true, payload["password_set"])
}

func TestLegacyNormalizeDeviceInfoForNorthboundUsesAccumulatedOnlineDuration(t *testing.T) {
	cumulative := int64(3600)
	currentOnline := int64(120)
	info := &device.DeviceWithInfo{
		Device:                   model.Device{IsOnline: true},
		CumulativeOnlineDuration: &cumulative,
		OnlineDuration:           &currentOnline,
	}

	legacyNormalizeDeviceInfoForNorthbound(info)

	require.Equal(t, int64(3720), *info.OnlineDuration)
	require.Equal(t, int64(3720), *info.CumulativeOnlineDuration)
}

func TestRefreshLegacyDeviceInfoFromParametersUsesCurrentRunTimeAndLicense(t *testing.T) {
	deviceID := uuid.New()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/northbound/v1/device/infos/SN-1", nil)
	router := &Router{deviceService: &fakeNBDeviceService{
		paramsFn: func(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
			return []model.DeviceParameter{
				{ParameterPath: device.ParamUpTime, ParameterValue: "1d 2h 3m 4s"},
				{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.Value", ParameterValue: "100"},
				{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod", ParameterValue: "45"},
			}, nil
		},
	}}
	info := &device.DeviceWithInfo{
		Device:        model.Device{ID: deviceID},
		LicenseStatus: stringPtr("expired"),
	}

	router.refreshLegacyDeviceInfoFromParameters(c, info)

	require.Equal(t, int64(1*86400+2*3600+3*60+4), *info.RunTime)
	require.Equal(t, "active", *info.LicenseStatus)
}

func TestRefreshLegacyDeviceInfoFromParametersUsesFAPServiceLicenseCapacity(t *testing.T) {
	deviceID := uuid.New()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/northbound/v1/device/infos/SN-1", nil)
	router := &Router{deviceService: &fakeNBDeviceService{
		paramsFn: func(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
			return []model.DeviceParameter{
				{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.State", ParameterValue: "1"},
				{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.RemainDays", ParameterValue: "15"},
			}, nil
		},
	}}
	info := &device.DeviceWithInfo{
		Device:        model.Device{ID: deviceID},
		LicenseStatus: stringPtr("expired"),
	}

	router.refreshLegacyDeviceInfoFromParameters(c, info)

	require.Equal(t, "expiring", *info.LicenseStatus)
}

func stringPtr(value string) *string {
	return &value
}
