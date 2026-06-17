package device

import (
	"testing"

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
			name: "single cell inactive",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "0",
			},
			want: "inactive",
		},
		{
			name: "single cell active via fapcontrol path",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "1",
			},
			want: "normal",
		},
		{
			name: "single cell active via config path",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellOpState": "1",
			},
			want: "normal",
		},
		{
			name: "NR paths",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.CellOpState": "1",
			},
			want: "normal",
		},
		{
			name: "NR indexed ran op state path matches detail page semantics",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.1.NR.RAN.OpState": "1",
			},
			want: "normal",
		},
		{
			name: "multi-cell any active returns normal",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "0",
				"Device.Services.FAPService.2.FAPControl.LTE.CellOpState": "1",
			},
			want: "normal",
		},
		{
			name: "multi-cell all inactive returns inactive",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "0",
				"Device.Services.FAPService.2.FAPControl.LTE.CellOpState": "false",
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
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "1",
			},
			want: "1",
		},
		{
			name: "单 cell inactive → 0",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "0",
			},
			want: "0",
		},
		{
			name: "多 cell 任一 active → 1(用户口径)",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "0",
				"Device.Services.FAPService.2.FAPControl.LTE.CellOpState": "1",
			},
			want: "1",
		},
		{
			name: "多 cell 全 inactive → 0",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "0",
				"Device.Services.FAPService.2.FAPControl.LTE.CellOpState": "false",
			},
			want: "0",
		},
		{
			name: "NR 路径同样起效",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.CellOpState": "1",
			},
			want: "1",
		},
		{
			name: "兜底 FAPControl.LTE.OpState 老路径",
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
				"Device.Services.GsmBTSCellDT.1.InUse":   "true",
				"Device.Services.GsmBTSCellDT.2.OpState": "0",
				"Device.Services.GsmBTSCellDT.2.InUse":   "true",
			},
			want: "0",
		},
		{
			name: "GSM 设备含 InUse=false 的 cell 应被跳过 — 仅 InUse=true 的 cell 也 inactive 时设备未激活",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.OpState": "1", // 但 InUse=false 不计
				"Device.Services.GsmBTSCellDT.1.InUse":   "false",
				"Device.Services.GsmBTSCellDT.2.OpState": "0",
				"Device.Services.GsmBTSCellDT.2.InUse":   "true",
			},
			want: "0",
		},
		{
			name: "GSM 老快照(无 InUse 字段)— 任一 cell active 也算激活",
			params: map[string]string{
				"Device.Services.GsmBTSCellDT.1.OpState": "1",
			},
			want: "1",
		},
		{
			name: "GSM cell active 与 LTE/NR cell inactive 共存 — 任一制式 active 即激活",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.CellOpState": "0",
				"Device.Services.GsmBTSCellDT.1.OpState":                  "1",
				"Device.Services.GsmBTSCellDT.1.InUse":                    "true",
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
			name:   "no active MME",
			params: map[string]string{},
			want:   "disconnected",
		},
		{
			name: "one active MME",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
			},
			want: "partial",
		},
		{
			name: "two active MMEs",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Status": "1",
			},
			want: "connected",
		},
		{
			name: "mixed active and inactive",
			params: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Status": "0",
				"Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.3.MME1Status": "1",
			},
			want: "connected",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcMMEStatus(tt.params))
		})
	}
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
				"Device.Services.FAPService.1.FAPControl.PLLSyncState":         "LOCKED",
				"Device.FAP.Synchronization.ClockSourceSyncState":              "HOLDOVER",
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
			name: "LTE management server tfcs sync state",
			params: map[string]string{
				"Device.ManagementServer.tfcsSyncState": "1",
				"Device.DeviceInfo.X_COM_GPS_Status":  "1",
			},
			want: "gps",
		},
		{
			name: "LTE management server raw textual sync state is preserved",
			params: map[string]string{
				"Device.ManagementServer.tfcsSyncState": "LOCKED",
				"Device.DeviceInfo.X_COM_GPS_Status":  "0",
				"Device.DeviceInfo.X_COM_BDS_Status":  "0",
				"Device.DeviceInfo.X_COM_1588_Status": "0",
			},
			want: "LOCKED",
		},
		{
			name: "GPS synced",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState": "1",
				"Device.DeviceInfo.X_COM_GPS_Status": "1",
			},
			want: "gps",
		},
		{
			name: "BDS synced",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState": "1",
				"Device.DeviceInfo.X_COM_BDS_Status": "1",
			},
			want: "beidou",
		},
		{
			name: "1588 synced",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState": "1",
				"Device.DeviceInfo.X_COM_1588_Status": "1",
			},
			want: "ntp",
		},
		{
			name: "GPS available but not locked",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState": "0",
				"Device.DeviceInfo.X_COM_GPS_Status": "1",
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
		name   string
		params map[string]string
		want   string
	}{
		{
			name:   "radio disabled",
			params: map[string]string{"Device.Services.FAPService.1.FAPControl.LTE.AdminState": "false"},
			want:   "off",
		},
		{
			name: "radio on, RF transmitting",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.AdminState":       "true",
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.RFTxStatus": "1",
			},
			want: "on",
		},
		{
			name: "radio on, RF not transmitting",
			params: map[string]string{
				"Device.Services.FAPService.1.FAPControl.LTE.AdminState":       "true",
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.RFTxStatus": "0",
			},
			want: "error",
		},
		{
			name:   "empty params",
			params: map[string]string{},
			want:   "off",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalcRFStatus(tt.params))
		})
	}
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
