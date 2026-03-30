package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ExtractFAPInstance(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{"FAPService.1 control", "Device.Services.FAPService.1.FAPControl.LTE.OpState", 1},
		{"FAPService.2 control", "Device.Services.FAPService.2.FAPControl.LTE.CellOpState", 2},
		{"FAPService.3 config", "Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.PhyCellID", 3},
		{"FAPService.Ipsec no digit", "Device.Services.FAPService.Ipsec.IPSEC_TUNNEL1_STATUS", 0},
		{"DeviceInfo no FAPService", "Device.DeviceInfo.HardwareVersion", 0},
		{"GPS no FAPService", "Device.FAP.GPS.LockedLatitude", 0},
		{"FaultMgmt no FAPService", "Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", 0},
		{"IP no FAPService", "Device.IP.Interface.1.IPv4Address.1.IPAddress", 0},
		{"ManagementServer", "Device.ManagementServer.ConnectionRequestURL", 0},
		{"empty path", "", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExtractFAPInstance(tc.path))
		})
	}
}

func Test_ClassifyParamGroup(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		// mme_pool — highest priority
		{"MmePoolConfigParam", "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status", "mme_pool"},
		{"X_COM_MmePool", "Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.Enable", "mme_pool"},

		// license
		{"LICENSE in FAPControl", "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Code", "license"},
		{"LICENSE Capacity", "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.State", "license"},

		// antenna
		{"AntennaInfo DeviceInfo", "Device.DeviceInfo.AntennaInfo.Height", "antenna"},
		{"AntennaInfo CellConfig", "Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.AntennaInfo.AntennaPortsCount", "antenna"},

		// alarm
		{"CurrentAlarm", "Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", "alarm"},
		{"CurrentAlarm 5", "Device.FaultMgmt.CurrentAlarm.5.AlarmIdentifier", "alarm"},

		// sync
		{"GPS Status", "Device.DeviceInfo.X_COM_GPS_Status", "sync"},
		{"BDS Status", "Device.DeviceInfo.X_COM_BDS_Status", "sync"},
		{"1588 Status", "Device.DeviceInfo.X_COM_1588_Status", "sync"},
		{"GLONASS", "Device.DeviceInfo.X_COM_GLONASS_Status", "sync"},
		{"FAP GPS", "Device.FAP.GPS.LockedLatitude", "sync"},
		{"tfcsSyncState", "Device.ManagementServer.tfcsSyncState", "sync"},
		{"tfcsManager", "Device.ManagementServer.tfcsManagerPrimsrc", "sync"},

		// radio
		{"RAN RF", "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", "radio"},
		{"RAN Common", "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", "radio"},
		{"RAN CA", "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", "radio"},
		{"EPC PLMN", "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID", "radio"},

		// fap_control — after mme_pool/license/sync priorities
		{"FAPControl OpState", "Device.Services.FAPService.1.FAPControl.LTE.OpState", "fap_control"},
		{"FAPControl CellOpState", "Device.Services.FAPService.1.FAPControl.LTE.CellOpState", "fap_control"},
		{"FAPControl AlarmStatus", "Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus", "fap_control"},
		{"FAPControl RFTxStatus", "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", "fap_control"},

		// device_info
		{"HardwareVersion", "Device.DeviceInfo.HardwareVersion", "device_info"},
		{"SoftwareVersion", "Device.DeviceInfo.SoftwareVersion", "device_info"},
		{"FAP_adminstate", "Device.DeviceInfo.FAP_adminstate", "device_info"},

		// management
		{"ConnectionRequestURL", "Device.ManagementServer.ConnectionRequestURL", "management"},

		// ipsec
		{"Ipsec tunnel", "Device.Services.FAPService.Ipsec.IPSEC_TUNNEL1_STATUS", "ipsec"},

		// network
		{"IP address", "Device.IP.Interface.1.IPv4Address.1.IPAddress", "network"},

		// software
		{"SystemBackupVersion", "Device.SoftwareCtrl.SystemBackupVersion", "software"},

		// other
		{"RootDataModelVersion", "Device.RootDataModelVersion", "other"},
		{"empty", "", "other"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ClassifyParamGroup(tc.path))
		})
	}
}

func Test_ClassifyParamGroup_Priority(t *testing.T) {
	// X_COM_LICENSE under FAPControl → license wins over fap_control
	assert.Equal(t, "license",
		ClassifyParamGroup("Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Code"))

	// MmePool under FAPControl → mme_pool wins over fap_control
	assert.Equal(t, "mme_pool",
		ClassifyParamGroup("Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.Enable"))

	// GPS in DeviceInfo → sync wins over device_info
	assert.Equal(t, "sync",
		ClassifyParamGroup("Device.DeviceInfo.X_COM_GPS_Status"))

	// AntennaInfo in CellConfig → antenna wins over radio
	assert.Equal(t, "antenna",
		ClassifyParamGroup("Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.AntennaInfo.AntennaPortsCount"))

	// tfcsSyncState in ManagementServer → sync wins over management
	assert.Equal(t, "sync",
		ClassifyParamGroup("Device.ManagementServer.tfcsSyncState"))
}
