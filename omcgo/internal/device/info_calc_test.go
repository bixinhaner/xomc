package device

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestCalcCellStatus(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   string
	}{
		{
			name:   "empty params",
			params: map[string]string{},
			want:   "inactive",
		},
		{
			name: "single cell inactive via strict LTE OpState path",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "0",
			},
			want: "inactive",
		},
		{
			name: "single cell active via strict LTE OpState path",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "1",
			},
			want: "normal",
		},
		{
			name: "LTE strict path true is active",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "true",
			},
			want: "normal",
		},
		{
			name: "NR strict path",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.1.NR.RAN.OpState": "1",
			},
			want: "normal",
		},
		{
			name: "non-strict NR path should be ignored",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.NR.RAN.OpState": "1",
			},
			want: "inactive",
		},
		{
			name: "multi-cell any active returns normal",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "0",
				"Device.Services.FAPService.2.FAPControl.LTE.OpState": "1",
			},
			want: "normal",
		},
		{
			name: "multi-cell all inactive returns inactive",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "0",
				"Device.Services.FAPService.2.FAPControl.LTE.OpState": "false",
			},
			want: "inactive",
		},
		{
			name: "GSM with InUse=true and OpState=1 returns normal",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.InUse":   "true",
				"Device.Services.GsmBTSCellDT.1.OpState": "1",
			},
			want: "normal",
		},
		{
			name: "GSM active but InUse=false is ignored when InUse exists",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.InUse":   "false",
				"Device.Services.GsmBTSCellDT.1.OpState": "1",
				"Device.Services.GsmBTSCellDT.2.InUse":   "true",
				"Device.Services.GsmBTSCellDT.2.OpState": "0",
			},
			want: "inactive",
		},
		{
			name: "GSM active without InUse is ignored",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.3.OpState": "1",
			},
			want: "inactive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcCellStatus(tt.params))
		})
	}
}

func TestCalcOpState(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   string
	}{
		{
			name:   "无 cell 数据 → 0(未激活)",
			params: map[string]string{},
			want:   "0",
		},
		{
			name: "单 cell active → 1",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "1",
			},
			want: "1",
		},
		{
			name: "单 cell inactive → 0",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "0",
			},
			want: "0",
		},
		{
			name: "OpState=true → 1（eNB 激活回归）",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "true",
			},
			want: "1",
		},
		{
			name: "多 cell 任一 active → 1(用户口径)",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "0",
				"Device.Services.FAPService.2.FAPControl.LTE.OpState": "1",
			},
			want: "1",
		},
		{
			name: "多 cell 全 inactive → 0",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "0",
				"Device.Services.FAPService.2.FAPControl.LTE.OpState": "false",
			},
			want: "0",
		},
		{
			name: "NR 严格路径同样起效",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.1.NR.RAN.OpState": "1",
			},
			want: "1",
		},
		{
			name: "LTE 严格路径 true 值",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "true",
			},
			want: "1",
		},
		{
			name: "GSM cell active(GsmBTSCellDT.{i}.OpState)— 走独立对象树",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.OpState": "1",
				"Device.Services.GsmBTSCellDT.1.InUse":   "true",
			},
			want: "1",
		},
		{
			name: "GSM cell 全 inactive — 设备未激活",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.OpState": "0",
				"Device.Services.GsmBTSCellDT.2.OpState": "0",
			},
			want: "0",
		},
		{
			name: "GSM 含 InUse 时仅统计 InUse=true 的 cell",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.OpState": "1", // InUse=false 不计
				"Device.Services.GsmBTSCellDT.1.InUse":   "false",
				"Device.Services.GsmBTSCellDT.2.OpState": "0",
				"Device.Services.GsmBTSCellDT.2.InUse":   "true",
			},
			want: "0",
		},
		{
			name: "GSM cell active 与 LTE/NR cell inactive 共存 — 任一制式 active 即激活",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.OpState": "0",
				"Device.Services.GsmBTSCellDT.1.OpState":              "1",
				"Device.Services.GsmBTSCellDT.1.InUse":                "true",
			},
			want: "1",
		},
		{
			name: "NR cellConfig index two under FAPService.1 is active",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.2.NR.RAN.OpState": "true",
			},
			want: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcOpState(tt.params))
		})
	}
}

func TestCalcMMEStatus(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   string
	}{
		{
			name: "lte gateway mme status connected",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus": "connected",
			},
			want: "connected",
		},
		{
			name: "lte gateway mme status true",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus": "true",
			},
			want: "connected",
		},
		{
			name: "lte gateway mme status disconnected",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus": "disconnected",
			},
			want: "disconnected",
		},
		{
			name: "lte gateway mme status has priority over legacy pool",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus":                   "disconnected",
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Status": "1",
			},
			want: "disconnected",
		},
		{
			name:   "no MME parameters returns empty",
			params: map[string]string{},
			want:   "",
		},
		{
			name: "BLQ legacy LTE path with all observed MME active returns connected",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status": "1",
			},
			want: "connected",
		},
		{
			name: "EPC and legacy paths for one instance count once",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
				"Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status":     "1",
			},
			want: "connected",
		},
		{
			name: "empty EPC placeholder falls back to legacy LTE path",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": " ",
				"Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status":     "1",
			},
			want: "connected",
		},
		{
			name: "fallback one configured active MME",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
			},
			want: "connected",
		},
		{
			name: "fallback two active MMEs",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Status": "1",
			},
			want: "connected",
		},
		{
			name: "fallback mixed active and inactive is connected when any MME is active",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Status": "0",
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.3.MME1Status": "1",
			},
			want: "connected",
		},
		{
			name: "MLN pool path is connected when any MME is active",
			params: map[string]string{
				"Device.Services.FAPService.MmePoolConfigParam.1.MMEStatus": "0",
				"Device.Services.FAPService.MmePoolConfigParam.2.MMEStatus": "1",
			},
			want: "connected",
		},
		{
			name: "MLN pool path is disconnected when all MMEs are inactive",
			params: map[string]string{
				"Device.Services.FAPService.MmePoolConfigParam.1.MMEStatus": "0",
				"Device.Services.FAPService.MmePoolConfigParam.2.MMEStatus": "0",
			},
			want: "disconnected",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcMMEStatus(tt.params))
		})
	}
}

func TestCalcMMEStatus_GatewayPartialNormalizesToConnected(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus": "partial",
	}

	assert.Equal(t, "connected", CalcMMEStatus(params))
}

func TestCalcCoreNetworkStatusByTechnology(t *testing.T) {
	params := map[string]string{
		"Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus": "connected",
		amfsStatusPath: "0.0.0.0=0;0.0.0.1=1",
	}

	assert.Equal(t, "connected", CalcCoreNetworkStatus(params, model.TechLTE))
	assert.Equal(t, "connected", CalcCoreNetworkStatus(params, model.TechNR))
	assert.Empty(t, CalcCoreNetworkStatus(params, model.TechGSM))
}

func TestCalcLicenseStatus(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   string
	}{
		{
			name:   "no license data",
			params: map[string]string{},
			want:   "expired",
		},
		{
			name: "active license with plenty of time",
			params: map[string]string{
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.State":           "1",
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod": "365",
			},
			want: "active",
		},
		{
			name: "license expiring soon",
			params: map[string]string{
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.State":           "1",
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod": "15",
			},
			want: "expiring",
		},
		{
			name: "license at boundary (30 days)",
			params: map[string]string{
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.State":           "1",
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod": "30",
			},
			want: "expiring",
		},
		{
			name: "license at boundary (31 days)",
			params: map[string]string{
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.State":           "1",
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod": "31",
			},
			want: "active",
		},
		{
			name: "multiple capacities, one expiring",
			params: map[string]string{
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.State":           "1",
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod": "365",
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.2.State":           "1",
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.2.RemainingPeriod": "10",
			},
			want: "expiring",
		},
		{
			name: "inactive capacities only",
			params: map[string]string{
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.State":           "0",
				"Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod": "365",
			},
			want: "expired",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcLicenseStatus(tt.params))
		})
	}
}

func TestCalcSyncStatus(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   string
	}{
		{
			name:   "no sync data",
			params: map[string]string{},
			want:   "error",
		},
		{
			name: "NR PLL sync state takes precedence",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.PLLSyncState":                   "LOCKED",
				"Device.FAP.Synchronization.ClockSourceSyncState":                        "HOLDOVER",
				"Device.Services.FAPService.1.FAPControl.NR.Gateway.X_COM_tfcsSyncState": "1",
			},
			want: "LOCKED",
		},
		{
			name: "NR clock source sync state fallback",
			params: map[string]string{
				"Device.FAP.Synchronization.ClockSourceSyncState": "SYNCED",
			},
			want: "SYNCED",
		},
		{
			name: "BaiBNQ clock source sync state takes precedence over GPS status",
			params: map[string]string{
				"Device.FAP.Synchronization.ClockSourceSyncState": "Synchronized",
				"Device.DeviceInfo.GPS_Status":                    "Synchronized",
			},
			want: "Synchronized",
		},
		{
			name: "LTE management server tfcs sync state",
			params: map[string]string{
				"Device.ManagementServer.tfcsSyncState": "1",
				"Device.DeviceInfo.X_COM_GPS_Status":    "1",
			},
			want: "gps",
		},
		{
			name: "LTE management server raw textual sync state is preserved",
			params: map[string]string{
				"Device.ManagementServer.tfcsSyncState": "LOCKED",
				"Device.DeviceInfo.X_COM_GPS_Status":    "0",
				"Device.DeviceInfo.X_COM_BDS_Status":    "0",
				"Device.DeviceInfo.X_COM_1588_Status":   "0",
			},
			want: "LOCKED",
		},
		{
			name: "GPS synced",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState": "1",
				"Device.DeviceInfo.X_COM_GPS_Status":                                      "1",
			},
			want: "gps",
		},
		{
			name: "standard GPS status synchronized",
			params: map[string]string{
				"Device.DeviceInfo.GPS_Status": "Synchronized",
			},
			want: "gps",
		},
		{
			name: "BDS synced",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState": "1",
				"Device.DeviceInfo.X_COM_BDS_Status":                                      "1",
			},
			want: "beidou",
		},
		{
			name: "1588 synced",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState": "1",
				"Device.DeviceInfo.X_COM_1588_Status":                                     "1",
			},
			want: "ntp",
		},
		{
			name: "GPS available but not locked",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState": "0",
				"Device.DeviceInfo.X_COM_GPS_Status":                                      "1",
			},
			want: "gps",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcSyncStatus(tt.params))
		})
	}
}

func TestCalcGPSStatus(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   string
	}{
		{
			name:   "GPS active",
			params: map[string]string{"Device.DeviceInfo.X_COM_GPS_Status": "1"},
			want:   "normal",
		},
		{
			name:   "GPS inactive",
			params: map[string]string{"Device.DeviceInfo.X_COM_GPS_Status": "0"},
			want:   "abnormal",
		},
		{
			name:   "GPS no data",
			params: map[string]string{},
			want:   "no_signal",
		},
		{
			name:   "GPS true string",
			params: map[string]string{"Device.DeviceInfo.X_COM_GPS_Status": "true"},
			want:   "normal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcGPSStatus(tt.params))
		})
	}
}

func TestCalcRFStatus(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]string
		tech         model.Technology
		productClass string
		wantStatus   string
		wantState    RFStatusProjectionState
	}{
		{
			name:         "rf radio disabled",
			params:       map[string]string{"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable": "false"},
			tech:         model.TechLTE,
			productClass: "FAP/MLQ/SC",
			wantStatus:   "off",
			wantState:    RFStatusValid,
		},
		{
			name:         "rf radio enabled",
			tech:         model.TechLTE,
			productClass: "FAP/MLQ/SC",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable": "true",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name:         "rf radio enable does not depend on admin state",
			tech:         model.TechLTE,
			productClass: "FAP/MLQ/SC",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.AdminState":               "false",
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable": "true",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name:         "legacy RFTxStatus fallback transmitting",
			tech:         model.TechLTE,
			productClass: "FAP/MLQ/SC",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.RFTxStatus": "1",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name:         "legacy RFTxStatus fallback not transmitting",
			tech:         model.TechLTE,
			productClass: "FAP/MLQ/SC",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.RFTxStatus": "0",
			},
			wantStatus: "off",
			wantState:  RFStatusValid,
		},
		{
			name:         "standard LTE FAPControl status",
			tech:         model.TechLTE,
			productClass: "FAP/MLQ/SC",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": "true",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name: "standard NR cell status",
			tech: model.TechNR,
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.2.NR.RAN.rftxEnable": "1",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name:         "BaiBNQ SAS radio enable",
			tech:         model.TechNR,
			productClass: "FAP/BSC7040/SC",
			params: map[string]string{
				"Device.DeviceInfo.SAS.RadioEnable": "true",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name: "BaiBNQ cell SAS radio enable",
			tech: model.TechNR,
			params: map[string]string{
				"Device.DeviceInfo.CellConfig.2.SAS.RadioEnable": "true",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name:         "unknown direct status falls back to valid SAS status",
			tech:         model.TechNR,
			productClass: "FAP/BSC7040/SC",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.1.NR.RAN.rftxEnable": "not-reported",
				"Device.DeviceInfo.SAS.RadioEnable":                           "true",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name:         "GSM BTS RF state",
			tech:         model.TechGSM,
			productClass: "FAP/BTS",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.RfState": "1",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name: "multi cell NR status",
			tech: model.TechNR,
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.2.NR.RAN.rftxEnable": "0",
				"Device.Services.FAPService.1.CellConfig.1.NR.RAN.rftxEnable": "1",
			},
			wantStatus: "on,off",
			wantState:  RFStatusValid,
		},
		{
			name:         "BAIBLQ SC ignores logical FAPService outside physical carrier range",
			tech:         model.TechLTE,
			productClass: "FAP/BAIBLQ/SC",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": "true",
				"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus": "true",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name:         "DC uses configured physical carriers and ignores higher logical instances",
			tech:         model.TechLTE,
			productClass: "FAP/MLN/DC",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "2",
				"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus":               "true",
				"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus":               "false",
				"Device.Services.FAPService.3.FAPControl.LTE.RFTxStatus":               "true",
				"Device.Services.FAPService.4.FAPControl.LTE.RFTxStatus":               "true",
			},
			wantStatus: "on,off",
			wantState:  RFStatusValid,
		},
		{
			name:         "DC missing a physical carrier is inconsistent",
			tech:         model.TechLTE,
			productClass: "FAP/MLN/DC",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "2",
				"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus":               "true",
			},
			wantState: RFStatusInconsistent,
		},
		{
			name:         "reported count conflicting with SC mode is inconsistent",
			tech:         model.TechLTE,
			productClass: "FAP/MLQ/SC",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "2",
				"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus":               "true",
				"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus":               "true",
			},
			wantState: RFStatusInconsistent,
		},
		{
			name:         "CA without reported count stays unknown",
			tech:         model.TechLTE,
			productClass: "FAP/MLN/CA",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": "true",
				"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus": "true",
			},
			wantState: RFStatusUnknown,
		},
		{
			name:         "BTS exposes one own RF despite duplicate logical services",
			tech:         model.TechGSM,
			productClass: "FAP/BTS",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": "true",
				"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus": "true",
			},
			wantStatus: "on",
			wantState:  RFStatusValid,
		},
		{
			name:         "BSC never projects child RF as its own",
			tech:         model.TechGSM,
			productClass: "FAP/PGSM",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.RfState":                 "1",
				"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": "true",
			},
			wantState: RFStatusUnsupported,
		},
		{
			name:         "duplicate values for one physical carrier conflict",
			tech:         model.TechNR,
			productClass: "FAP/BSC7040/SC",
			params: map[string]string{
				"Device.DeviceInfo.SAS.RadioEnable":  "true",
				"Device.DeviceInfo.SAS.RadioEnable1": "false",
			},
			wantState: RFStatusInconsistent,
		},
		{
			name: "unknown RF value",
			tech: model.TechNR,
			params: map[string]string{
				"Device.DeviceInfo.SAS.RadioEnable": "not-reported",
			},
			wantState: RFStatusUnknown,
		},
		{
			name:      "empty params",
			params:    map[string]string{},
			tech:      model.TechLTE,
			wantState: RFStatusUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcRFStatus(tt.params, tt.tech, tt.productClass)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, tt.wantState, got.State)
		})
	}
}

func TestCalcRFStatusFromDeviceParametersUsesNewestAliasObservation(t *testing.T) {
	standardPath := "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	privatePath := "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable"
	base := time.Date(2026, 8, 12, 16, 40, 0, 0, time.UTC)

	t.Run("newer private false overrides stale standard true", func(t *testing.T) {
		got := CalcRFStatusFromDeviceParameters([]model.DeviceParameter{
			{ParameterPath: standardPath, ParameterValue: "true", LastUpdatedAt: base},
			{ParameterPath: privatePath, ParameterValue: "false", LastUpdatedAt: base.Add(time.Second)},
		}, model.TechLTE, "FAP/BAIBLQ/SC")

		assert.Equal(t, RFStatusValid, got.State)
		assert.Equal(t, "off", got.Status)
	})

	t.Run("newer standard true wins older private false", func(t *testing.T) {
		got := CalcRFStatusFromDeviceParameters([]model.DeviceParameter{
			{ParameterPath: standardPath, ParameterValue: "true", LastUpdatedAt: base.Add(time.Second)},
			{ParameterPath: privatePath, ParameterValue: "false", LastUpdatedAt: base},
		}, model.TechLTE, "FAP/BAIBLQ/SC")

		assert.Equal(t, RFStatusValid, got.State)
		assert.Equal(t, "on", got.Status)
	})

	t.Run("same-time contradictory aliases are inconsistent", func(t *testing.T) {
		got := CalcRFStatusFromDeviceParameters([]model.DeviceParameter{
			{ParameterPath: standardPath, ParameterValue: "true", LastUpdatedAt: base},
			{ParameterPath: privatePath, ParameterValue: "false", LastUpdatedAt: base},
		}, model.TechLTE, "FAP/BAIBLQ/SC")

		assert.Equal(t, RFStatusInconsistent, got.State)
		assert.Empty(t, got.Status)
	})

	t.Run("newest private alias is the single authority", func(t *testing.T) {
		got := CalcRFStatusFromDeviceParameters([]model.DeviceParameter{
			{ParameterPath: standardPath, ParameterValue: "true", LastUpdatedAt: base},
			{ParameterPath: privatePath, ParameterValue: "false", LastUpdatedAt: base.Add(time.Second)},
			{
				ParameterPath:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
				ParameterValue: "true",
				LastUpdatedAt:  base.Add(2 * time.Second),
			},
		}, model.TechLTE, "FAP/BLN/SC")

		assert.Equal(t, RFStatusValid, got.State)
		assert.Equal(t, "on", got.Status)
	})
}

func TestCalcNumOfCells(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   int
	}{
		{
			name:   "default single carrier",
			params: map[string]string{},
			want:   1,
		},
		{
			name: "CA with 2 cells",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "2",
			},
			want: 2,
		},
		{
			name: "TC with 3 cells",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "3",
			},
			want: 3,
		},
		{
			name: "invalid value",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells": "abc",
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcNumOfCells(tt.params))
		})
	}
}
