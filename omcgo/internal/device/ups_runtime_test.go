package device

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUPSRuntimeProjection_NewUPS(t *testing.T) {
	now := time.Date(2026, 8, 19, 10, 20, 30, 0, time.UTC)
	device := &model.Device{
		ID:              uuid.New(),
		SerialNumber:    "UPS-SN-001",
		OUI:             "ABCDEF",
		ProductClass:    "UPS_M3_BMU",
		FirmwareVersion: "fallback-version",
		LastInformAt:    &now,
	}
	inform := &tr069.InformMessage{
		DeviceId: tr069.DeviceId{
			Manufacturer: "Baicells",
			OUI:          "ABCDEF",
			ProductClass: "UPS_M3_BMU",
			SerialNumber: "UPS-SN-001",
		},
		ParameterList: []tr069.ParameterValueStruct{
			pv(igdSoftwareVersionPath, "UPS-1.0.3"),
			pv(igdUPSExternalIPAddressPath, "192.0.2.10"),
			pv("InternetGatewayDevice.DeviceInfo.HardwareVersion", "HW-A"),
			pv("InternetGatewayDevice.DeviceInfo.UpTime", "3600"),
			pv("InternetGatewayDevice.BMSInfo.Voltage", "53.4"),
			pv("InternetGatewayDevice.BMSInfo.Temperature", "31"),
			pv("InternetGatewayDevice.BMSInfo.Current", "12.1"),
			pv("InternetGatewayDevice.BMSInfo.Charging", "CHARGING"),
			pv("InternetGatewayDevice.ChargerInfo.ACPower", "ON"),
			pv("InternetGatewayDevice.ChargerInfo.DCVoltage", "53.1"),
			pv("InternetGatewayDevice.ChargerInfo.DCCurrent", "11.8"),
			pv("InternetGatewayDevice.ChargerInfo.BoardTemperature", "35"),
			pv("InternetGatewayDevice.ChargerInfo.SFPState", "PLUGIN"),
			pv("InternetGatewayDevice.ChargerInfo.PORT0State", "UP"),
			pv("InternetGatewayDevice.ChargerInfo.PORT1State", "DOWN"),
			pv("InternetGatewayDevice.ChargerInfo.PORT2State", "DOWN"),
			pv("InternetGatewayDevice.ChargerInfo.PORT3State", "FAULT"),
			pv("InternetGatewayDevice.ChargerInfo.AverageSOC", "96"),
			pv("InternetGatewayDevice.ChargerInfo.PackCounts", "2"),
			pv("InternetGatewayDevice.BMSInfo.1.SerialNumber", "BAT-001"),
			pv("InternetGatewayDevice.BMSInfo.1.SOC", "95"),
			pv("InternetGatewayDevice.BMSInfo.1.Voltage", "26.7"),
			pv("InternetGatewayDevice.BMSInfo.1.Temperature", "30"),
			pv("InternetGatewayDevice.BMSInfo.1.Current", "5.9"),
			pv("InternetGatewayDevice.BMSInfo.1.Status", "NORMAL"),
			pv("InternetGatewayDevice.BMSInfo.1.RecycleCount", "7"),
			pv("InternetGatewayDevice.BMSInfo.1.Charging", "CHARGING"),
			pv("InternetGatewayDevice.BMSInfo.1.Model", "BAT-A"),
			pv("InternetGatewayDevice.BMSInfo.1.SoftwareVersion", "BMS-1"),
			pv("InternetGatewayDevice.BMSInfo.2.SerialNumber", "N/A"),
			pv("InternetGatewayDevice.BMSInfo.2.SOC", "90"),
		},
	}

	runtime, batteries, ok := buildUPSRuntimeProjection(device, inform)

	require.True(t, ok)
	require.NotNil(t, runtime)
	assert.Equal(t, device.ID, runtime.DeviceID)
	assert.Equal(t, "UPS-SN-001", runtime.DeviceSerialNumber)
	assert.Equal(t, "192.0.2.10", runtime.ExternalIP)
	assert.Equal(t, "UPS-1.0.3", runtime.SoftwareVersion)
	assert.Equal(t, "HW-A", runtime.HardwareVersion)
	assert.Equal(t, "ON", runtime.ACPower)
	assert.Equal(t, "35", runtime.BoardTemperature)
	assert.Equal(t, "FAULT", runtime.Port3State)
	require.NotNil(t, runtime.AverageSOC)
	assert.Equal(t, 96, *runtime.AverageSOC)
	assert.Equal(t, "96", runtime.AverageSOCValues)
	require.NotNil(t, runtime.PackCounts)
	assert.Equal(t, 2, *runtime.PackCounts)
	require.NotNil(t, runtime.UpTimeSeconds)
	assert.Equal(t, int64(3600), *runtime.UpTimeSeconds)

	require.Len(t, batteries, 2)
	assert.Equal(t, 1, batteries[0].PackIndex)
	require.NotNil(t, batteries[0].SerialNumber)
	assert.Equal(t, "BAT-001", *batteries[0].SerialNumber)
	require.NotNil(t, batteries[0].SOC)
	assert.Equal(t, 95, *batteries[0].SOC)
	assert.Equal(t, "95", batteries[0].SOCValues)
	require.NotNil(t, batteries[0].RecycleCount)
	assert.Equal(t, 7, *batteries[0].RecycleCount)
	assert.Equal(t, 2, batteries[1].PackIndex)
	require.NotNil(t, batteries[1].SerialNumber)
	assert.Equal(t, "--", *batteries[1].SerialNumber)
	require.NotNil(t, batteries[1].SOC)
	assert.Equal(t, 90, *batteries[1].SOC)
	assert.Equal(t, "90", batteries[1].SOCValues)
}

func TestBuildUPSRuntimeProjection_LegacyUPSWithoutPackCounts(t *testing.T) {
	device := &model.Device{
		ID:           uuid.New(),
		SerialNumber: "UPS-SN-LEGACY",
		ProductClass: "UPS_LEGACY",
	}
	inform := &tr069.InformMessage{
		DeviceId: tr069.DeviceId{ProductClass: "UPS_LEGACY"},
		ParameterList: []tr069.ParameterValueStruct{
			pv("InternetGatewayDevice.BMSInfo.Voltage", "51.8"),
			pv("InternetGatewayDevice.BMSInfo.Temperature", "29"),
			pv("InternetGatewayDevice.BMSInfo.Current", "8.2"),
			pv("InternetGatewayDevice.BMSInfo.Charging", "STANDBY"),
		},
	}

	runtime, batteries, ok := buildUPSRuntimeProjection(device, inform)

	require.True(t, ok)
	require.NotNil(t, runtime)
	assert.Nil(t, runtime.PackCounts)
	require.Len(t, batteries, 1)
	assert.Equal(t, 1, batteries[0].PackIndex)
	require.NotNil(t, batteries[0].SerialNumber)
	assert.Equal(t, "--", *batteries[0].SerialNumber)
	assert.Equal(t, "51.8", batteries[0].Voltage)
	assert.Equal(t, "STANDBY", batteries[0].Charging)
	assert.Equal(t, "--", batteries[0].Model)
	assert.Equal(t, "--", batteries[0].SoftwareVersion)
}

func TestBuildUPSRuntimeProjection_IndexedBatteryWithBlankSerialNumber(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 14, 26, 0, time.UTC)
	device := &model.Device{
		ID:           uuid.New(),
		SerialNumber: "20202001019999",
		ProductClass: "UPS/FSU2024",
		LastInformAt: &now,
	}
	inform := &tr069.InformMessage{
		DeviceId: tr069.DeviceId{ProductClass: "UPS/FSU2024", SerialNumber: "20202001019999"},
		ParameterList: []tr069.ParameterValueStruct{
			pv("InternetGatewayDevice.DeviceInfo.SoftwareVersion", "DTU200E_1020260722"),
			pv("InternetGatewayDevice.DeviceInfo.UpTime", "6911.00"),
			pv("InternetGatewayDevice.BMSInfo.1.SerialNumber", ""),
			pv("InternetGatewayDevice.BMSInfo.1.SOC", "34,34,34"),
			pv("InternetGatewayDevice.BMSInfo.1.SOH", "100"),
			pv("InternetGatewayDevice.BMSInfo.1.Voltage", "49190,3280,3279"),
			pv("InternetGatewayDevice.BMSInfo.1.Temperature", "24.85,24.85,24.85,24.85"),
			pv("InternetGatewayDevice.BMSInfo.1.Current", "0"),
			pv("InternetGatewayDevice.BMSInfo.1.Status", "NORMAL"),
			pv("InternetGatewayDevice.BMSInfo.1.RecycleCount", "2"),
			pv("InternetGatewayDevice.BMSInfo.1.Charging", "STANDBY"),
			pv("InternetGatewayDevice.BMSInfo.1.Model", ""),
			pv("InternetGatewayDevice.BMSInfo.1.SoftwareVersion", ""),
			pv("InternetGatewayDevice.ChargerInfo.AverageSOC", "34"),
			pv("InternetGatewayDevice.ChargerInfo.PackCounts", "1"),
			pv("InternetGatewayDevice.ChargerInfo.Port0State", "UP"),
		},
	}

	runtime, batteries, ok := buildUPSRuntimeProjection(device, inform)

	require.True(t, ok)
	require.NotNil(t, runtime)
	assert.Equal(t, "49190,3280,3279", runtime.TotalVoltage)
	assert.Equal(t, "24.85,24.85,24.85,24.85", runtime.TotalTemperature)
	assert.Equal(t, "0", runtime.TotalCurrent)
	assert.Equal(t, "STANDBY", runtime.BMSCharging)
	assert.Equal(t, "UP", runtime.Port0State)
	require.NotNil(t, runtime.UpTimeSeconds)
	assert.Equal(t, int64(6911), *runtime.UpTimeSeconds)
	require.NotNil(t, runtime.AverageSOC)
	assert.Equal(t, 34, *runtime.AverageSOC)
	assert.Equal(t, "34", runtime.AverageSOCValues)

	require.Len(t, batteries, 1)
	assert.Equal(t, 1, batteries[0].PackIndex)
	require.NotNil(t, batteries[0].SerialNumber)
	assert.Equal(t, "--", *batteries[0].SerialNumber)
	require.NotNil(t, batteries[0].SOC)
	assert.Equal(t, 34, *batteries[0].SOC)
	assert.Equal(t, "34,34,34", batteries[0].SOCValues)
	assert.Equal(t, "49190,3280,3279", batteries[0].Voltage)
	assert.Equal(t, "24.85,24.85,24.85,24.85", batteries[0].Temperature)
	assert.Equal(t, "NORMAL", batteries[0].Status)
	require.NotNil(t, batteries[0].RecycleCount)
	assert.Equal(t, 2, *batteries[0].RecycleCount)
	assert.Equal(t, "--", batteries[0].Model)
	assert.Equal(t, "--", batteries[0].SoftwareVersion)
}

func TestBuildUPSRuntimeProjection_NonUPSIgnored(t *testing.T) {
	device := &model.Device{ID: uuid.New(), SerialNumber: "ENB-SN-001", ProductClass: "FAP/BAIBLQ/SC"}
	inform := &tr069.InformMessage{DeviceId: tr069.DeviceId{ProductClass: "FAP/BAIBLQ/SC"}}

	runtime, batteries, ok := buildUPSRuntimeProjection(device, inform)

	assert.False(t, ok)
	assert.Nil(t, runtime)
	assert.Nil(t, batteries)
}

func TestUPSInformFallbackHelpers(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		pv(igdSoftwareVersionPath, "UPS-2.0.0"),
		pv(igdConnectionRequestURLPath, "http://192.0.2.20:7547/"),
		pv(igdUDPConnectionRequestAddressPath, "192.0.2.20:7547"),
		pv(igdUPSExternalIPAddressPath, "not-an-ip"),
	}

	assert.Equal(t, "UPS-2.0.0", informSoftwareVersion(params))
	assert.Equal(t, "http://192.0.2.20:7547/", informConnectionRequestURL(params))
	assert.Equal(t, "192.0.2.20:7547", deriveUDPConnectionRequestAddress(params))
	assert.Empty(t, informUPSExternalIPAddress(params))
}

func pv(name, value string) tr069.ParameterValueStruct {
	return tr069.ParameterValueStruct{Name: name, Value: value}
}
