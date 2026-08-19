package device

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestAssembleMMEPool(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Address", ParameterValue: "10.0.0.1"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.PLMNID", ParameterValue: "46000"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Status", ParameterValue: "0"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.2.MME1Address", ParameterValue: "10.0.0.2"},
	}

	entries := AssembleMMEPool(params)
	assert.Len(t, entries, 2)

	// Find active entry
	var active, inactive *MMEEntry
	for i := range entries {
		if entries[i].Index == 1 {
			active = &entries[i]
		} else if entries[i].Index == 2 {
			inactive = &entries[i]
		}
	}
	assert.NotNil(t, active)
	assert.Equal(t, "active", active.Status)
	assert.Equal(t, "10.0.0.1", active.IP)
	assert.Equal(t, "46000", active.PLMNID)

	assert.NotNil(t, inactive)
	assert.Equal(t, "inactive", inactive.Status)
}

func TestAssembleMMEPool_Empty(t *testing.T) {
	entries := AssembleMMEPool(nil)
	assert.Empty(t, entries)
}

// TestAssembleMMEPool_MMEIp1Path verifies Phase 4 mapper fix —
// Baicells BaiBLQ 等实际上报路径 `MmePoolConfigParam.{N}.MMEIp1` 被正确识别。
// 设计文档 §3.3。
func TestAssembleMMEPool_MMEIp1Path(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MMEIp1", ParameterValue: "172.23.224.88"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.PLMNID", ParameterValue: "46068"},
	}
	entries := AssembleMMEPool(params)
	assert.Len(t, entries, 1)
	assert.Equal(t, "active", entries[0].Status)
	assert.Equal(t, "172.23.224.88", entries[0].IP)
	assert.Equal(t, "46068", entries[0].PLMNID)
}

func TestAssembleMMEPool_MLNPath(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.MmePoolConfigParam.1.MMEStatus", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.MmePoolConfigParam.1.MMEIp", ParameterValue: "172.24.224.88"},
	}

	entries := AssembleMMEPool(params)

	assert.Len(t, entries, 1)
	assert.Equal(t, "active", entries[0].Status)
	assert.Equal(t, "172.24.224.88", entries[0].IP)
}

func TestAssembleMMEPool_XCOMPoolStatusFallback(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.MmePool1List", ParameterValue: "172.24.224.88"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.MmePool1Status", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.MmePool2List", ParameterValue: "172.24.224.91"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_MmePool.MmePool2Status", ParameterValue: "0"},
	}

	assert.Equal(t, []MMEEntry{
		{Index: 1, IP: "172.24.224.88", Status: "active"},
		{Index: 2, IP: "172.24.224.91", Status: "inactive"},
	}, AssembleMMEPool(params))
}

func TestAssembleLicenseDetail(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.LicenseCode", ParameterValue: "ABC-123"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.GenerateDate", ParameterValue: "2024-01-15"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.Description", ParameterValue: "LTE Band 40"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.State", ParameterValue: "1"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.ValidPeriod", ParameterValue: "365"},
		{ParameterPath: "Device.DeviceInfo.X_COM_LICENSE.Capacity.1.RemainingPeriod", ParameterValue: "200"},
	}

	detail := AssembleLicenseDetail(params)
	assert.NotNil(t, detail)
	assert.Equal(t, "ABC-123", detail.Code)
	assert.Equal(t, "2024-01-15", detail.GenerateDate)
	assert.Len(t, detail.Capacities, 1)
	assert.Equal(t, "LTE Band 40", detail.Capacities[0].Description)
	assert.Equal(t, 200, detail.Capacities[0].RemainingPeriod)
}

func TestAssembleLicenseDetail_Empty(t *testing.T) {
	detail := AssembleLicenseDetail(nil)
	assert.Nil(t, detail)
}

// TestAssembleLicenseDetail_ValueField verifies Phase 4 mapper fix —
// Baicells 实际上报 Capacity.{N}.Value 字段（容量数值），不上报 State；
// 此前 mapper 只取 State 导致 capacity 数据全空。设计文档 §3.3。
func TestAssembleLicenseDetail_ValueField(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.Description", ParameterValue: "Max served UE num"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.Value", ParameterValue: "100"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.ValidPeriod", ParameterValue: "180"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.RemainingPeriod", ParameterValue: "0"},
	}
	detail := AssembleLicenseDetail(params)
	assert.NotNil(t, detail)
	assert.Len(t, detail.Capacities, 1)
	assert.Equal(t, "Max served UE num", detail.Capacities[0].Description)
	assert.Equal(t, "100", detail.Capacities[0].Value)
	assert.Equal(t, "", detail.Capacities[0].State) // CPE 不上报 → 仍空
	assert.Equal(t, 180, detail.Capacities[0].ValidPeriod)
	assert.Equal(t, 0, detail.Capacities[0].RemainingPeriod)
}

func TestAssembleAntennaInfo(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.Azimuth", ParameterValue: "120"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.Gain", ParameterValue: "15"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.Height", ParameterValue: "30"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.AntennaInfo.Downtilt", ParameterValue: "6"},
	}

	info := AssembleAntennaInfo(params)
	assert.NotNil(t, info)
	assert.Equal(t, "120", info.Azimuth)
	assert.Equal(t, "15", info.Gain)
	assert.Equal(t, "30", info.Height)
	assert.Equal(t, "6", info.Downtilt)
}

func TestAssembleAntennaInfo_Empty(t *testing.T) {
	info := AssembleAntennaInfo(nil)
	assert.Nil(t, info)
}

func TestAssembleMultiPlmnEnable(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: nrMultiPlmnEnablePath, ParameterValue: "0"},
	}
	assert.Equal(t, "disabled", AssembleMultiPlmnEnable(params))
}

func TestAssembleMultiPlmnEnable_TrueValue(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: nrMultiPlmnEnablePath, ParameterValue: "true"},
	}
	assert.Equal(t, "enabled", AssembleMultiPlmnEnable(params))
}

func TestAssembleAMFStatus(t *testing.T) {
	params := []model.DeviceParameter{{ParameterPath: amfsStatusPath, ParameterValue: "connected"}}
	assert.Equal(t, "connected", AssembleAMFStatus(params))
}

func TestAssembleAMFStatus_AnyPeerConnected(t *testing.T) {
	params := []model.DeviceParameter{{ParameterPath: amfsStatusPath, ParameterValue: "0.0.0.0=0;0.0.0.1=1"}}
	assert.Equal(t, "connected", AssembleAMFStatus(params))
}

func TestAssembleAMFStatus_AllPeersDisconnected(t *testing.T) {
	params := []model.DeviceParameter{{ParameterPath: amfsStatusPath, ParameterValue: "0.0.0.0=0;0.0.0.1=0"}}
	assert.Equal(t, "disconnected", AssembleAMFStatus(params))
}

func TestAssembleGPSVersion(t *testing.T) {
	params := []model.DeviceParameter{{ParameterPath: gpsSoftVersionPath, ParameterValue: "GPS_2.0.1"}}
	assert.Equal(t, "GPS_2.0.1", AssembleGPSVersion(params))
}

func TestAssemblePPSTimeMode(t *testing.T) {
	params := []model.DeviceParameter{{ParameterPath: ppsTimeModePath, ParameterValue: "GPS"}}
	assert.Equal(t, "GPS", AssemblePPSTimeMode(params))
}

func TestAssembleRollbackVersion(t *testing.T) {
	params := []model.DeviceParameter{{ParameterPath: systemBackupVersionPath, ParameterValue: "DENGYO_BNQ_2.6.12"}}
	assert.Equal(t, "DENGYO_BNQ_2.6.12", AssembleRollbackVersion(params))
}

func TestAssembleWANStatus(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Ethernet.Interface.1.Status", ParameterValue: "DOWN"},
		{ParameterPath: "Device.Ethernet.Interface.2.Status", ParameterValue: "UP"},
		{ParameterPath: "Device.Ethernet.Interface.2.Name", ParameterValue: "WAN"},
	}
	assert.Equal(t, "up", AssembleWANStatus(params))
}

func TestAssembleWANStatus_IPInterfaceFallback(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.IP.Interface.1.Status", ParameterValue: "Down"},
		{ParameterPath: "Device.IP.Interface.1.IPv4Address.1.Status", ParameterValue: "Disabled"},
	}
	assert.Equal(t, "down", AssembleWANStatus(params))
}

func TestAssembleCells(t *testing.T) {
	params := []model.DeviceParameter{
		// Cell 1 (LTE)
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "12345"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", ParameterValue: "100"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL", ParameterValue: "38950"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "true"},
		// Cell 2 (LTE CA)
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "12346"},
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.PhyCellID", ParameterValue: "101"},
		{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.OpState", ParameterValue: "false"},
	}

	cells := AssembleCells(params, 2)
	assert.Len(t, cells, 2)

	assert.Equal(t, 1, cells[0].Index)
	assert.Equal(t, "12345", cells[0].CellID)
	assert.Equal(t, "12345", cells[0].ECI)
	assert.Equal(t, "100", cells[0].PCI)
	assert.Equal(t, "38950", cells[0].FreqPoint)
	assert.Equal(t, "true", cells[0].OpState)

	assert.Equal(t, 2, cells[1].Index)
	assert.Equal(t, "12346", cells[1].CellID)
	assert.Equal(t, "12346", cells[1].ECI)
	assert.Equal(t, "false", cells[1].OpState)
}

func TestAssembleCells_SingleCell(t *testing.T) {
	cells := AssembleCells(nil, 1)
	assert.Len(t, cells, 1)
	assert.Equal(t, 1, cells[0].Index)
}

func TestAssembleCells_SCDoesNotReturnPhantomCarriers(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "210922753"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "false"},
		{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.OpState", ParameterValue: "false"},
		{ParameterPath: "Device.Services.FAPService.3.FAPControl.LTE.RFTxStatus", ParameterValue: "false"},
	}

	cells := AssembleCells(params, 3, "FAP/MLN/SC")

	assert.Len(t, cells, 1)
	assert.Equal(t, 1, cells[0].Index)
}

// TestAssembleCells_FAPControlPaths verifies strict OpState mapping + RF/Admin mapping —
// 实际上报路径：
//   - OpState 来自 `FAPControl.LTE.OpState`
//   - RFTxStatus 来自 `FAPControl.LTE.RFTxStatus`（非 RAN.RF 子树）
//
// 设计文档 §3.3。
func TestAssembleCells_FAPControlPaths(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "654321"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", ParameterValue: "2"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL", ParameterValue: "1500"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "false"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "false"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.AdminState", ParameterValue: "false"},
	}
	cells := AssembleCells(params, 1)
	assert.Len(t, cells, 1)
	assert.Equal(t, "654321", cells[0].CellID)
	assert.Equal(t, "654321", cells[0].ECI)
	assert.Equal(t, "2", cells[0].PCI)
	assert.Equal(t, "1500", cells[0].FreqPoint)
	assert.Equal(t, "false", cells[0].OpState)
	assert.Equal(t, "false", cells[0].RFTxStatus)
	assert.Equal(t, "false", cells[0].AdminState)
}

// TestAssembleCells_LTEOpStatePath verifies strict LTE OpState path.
func TestAssembleCells_LTEOpStatePath(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "true"},
	}
	cells := AssembleCells(params, 1)
	assert.Len(t, cells, 1)
	assert.Equal(t, "true", cells[0].OpState)
}

func TestAssembleCells_RFTxStatusFallsBackToUniformRUValue(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "210922753"},
		{ParameterPath: "Device.DeviceInfo.RU.1.RFTxStatus", ParameterValue: "false"},
		{ParameterPath: "Device.DeviceInfo.RU.2.RFTxStatus", ParameterValue: "false"},
	}

	cells := AssembleCells(params, 1)
	assert.Len(t, cells, 1)
	assert.Equal(t, "false", cells[0].RFTxStatus)
}

func TestAssembleCells_RFTxStatusDoesNotFallbackWhenRUValuesConflict(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "210922753"},
		{ParameterPath: "Device.DeviceInfo.RU.1.RFTxStatus", ParameterValue: "false"},
		{ParameterPath: "Device.DeviceInfo.RU.2.RFTxStatus", ParameterValue: "true"},
	}

	cells := AssembleCells(params, 1)
	assert.Len(t, cells, 1)
	assert.Equal(t, "", cells[0].RFTxStatus)
}

func TestAssembleCells_AdminStateFallsBackToUniformFAPControlValue(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "210922753"},
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity", ParameterValue: "210922754"},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.AdminState", ParameterValue: "false"},
	}

	cells := AssembleCells(params, 2)
	assert.Len(t, cells, 2)
	assert.Equal(t, "false", cells[0].AdminState)
	assert.Equal(t, "false", cells[1].AdminState)
}

func TestAssembleCells_DetectsHigherFAPServiceIndex(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", ParameterValue: "11"},
		{ParameterPath: "Device.Services.FAPService.6.CellConfig.LTE.RAN.RF.PhyCellID", ParameterValue: "66"},
		{ParameterPath: "Device.Services.FAPService.6.FAPControl.LTE.OpState", ParameterValue: "1"},
	}

	// 设备表 num_of_cells 即使是 1，也应按参数路径探测到 6 个实例。
	cells := AssembleCells(params, 1)
	assert.Len(t, cells, 6)
	assert.Equal(t, "11", cells[0].PCI)
	assert.Equal(t, "66", cells[5].PCI)
	assert.Equal(t, "1", cells[5].OpState)
}

func TestAssembleCells_NRIndexedPaths(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.NrcellIdentity", ParameterValue: "1153"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID", ParameterValue: "21"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.NRARFCNDL", ParameterValue: "513000"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.1.FreqBandIndicatorNR", ParameterValue: "41"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.OpState", ParameterValue: "0"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.rftxEnable", ParameterValue: "0"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.CellEnable.AdminState", ParameterValue: "1"},
	}
	cells := AssembleCells(params, 1)
	assert.Len(t, cells, 1)
	assert.Equal(t, "1153", cells[0].CellID)
	assert.Equal(t, "21", cells[0].PCI)
	assert.Equal(t, "513000", cells[0].FreqPoint)
	assert.Equal(t, "41", cells[0].Band)
	assert.Equal(t, "0", cells[0].OpState)
	assert.Equal(t, "0", cells[0].RFTxStatus)
	assert.Equal(t, "1", cells[0].AdminState)
	assert.Equal(t, "1153", cells[0].ECI)
	assert.Equal(t, "", cells[0].Bandwidth)
}

func TestAssembleCells_NRStrictPathForIndexTwo(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.OpState", ParameterValue: "true"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.CN.TA.1.NrcellIdentity", ParameterValue: "2202"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.Common.CellLocalId", ParameterValue: "2202"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.RF.PhyCellID", ParameterValue: "22"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.RF.NRARFCNDL", ParameterValue: "630000"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.RF.ChannelBandwidth", ParameterValue: "100"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.1.FreqBandIndicatorNR", ParameterValue: "78"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.rftxEnable", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.CellEnable.AdminState", ParameterValue: "1"},
	}

	cells := AssembleCells(params, 2)
	assert.Len(t, cells, 2)
	assert.Equal(t, "", cells[0].OpState)
	assert.Equal(t, "2202", cells[1].CellID)
	assert.Equal(t, "2202", cells[1].ECI)
	assert.Equal(t, "22", cells[1].PCI)
	assert.Equal(t, "630000", cells[1].FreqPoint)
	assert.Equal(t, "100", cells[1].Bandwidth)
	assert.Equal(t, "78", cells[1].Band)
	assert.Equal(t, "true", cells[1].OpState)
	assert.Equal(t, "1", cells[1].RFTxStatus)
	assert.Equal(t, "1", cells[1].AdminState)
}

func TestAssembleCells_NRLegacyPathIgnoredForIndexTwo(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.NR.RAN.OpState", ParameterValue: "true"},
	}

	cells := AssembleCells(params, 2)
	assert.Len(t, cells, 2)
	assert.Equal(t, "", cells[1].OpState)
}

func TestAssembleCells_NRStatusOnlyPathsDoNotFallbackToLTEOrRU(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", ParameterValue: "false"},
		{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.AdminState", ParameterValue: "false"},
		{ParameterPath: "Device.DeviceInfo.RU.1.RFTxStatus", ParameterValue: "0"},
		{ParameterPath: "Device.DeviceInfo.RU.2.RFTxStatus", ParameterValue: "0"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.rftxEnable", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.CellEnable.AdminState", ParameterValue: "1"},
	}

	cells := AssembleCells(params, 2)
	assert.Len(t, cells, 2)
	assert.Equal(t, "1", cells[1].RFTxStatus)
	assert.Equal(t, "1", cells[1].AdminState)
}

func TestAssembleCells_StrictNRPathLiftsDetectedIndex(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.OpState", ParameterValue: "true"},
	}

	cells := AssembleCells(params, 1)
	assert.Len(t, cells, 2)
	assert.Equal(t, "true", cells[1].OpState)
}

func TestAssembleGSMCells(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.InUse", ParameterValue: "1"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.GsmCellID", ParameterValue: "1001"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.CurrLocAreaCode", ParameterValue: "2001"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.CurrentArfcn", ParameterValue: "45"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.OpState", ParameterValue: "1"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.RfState", ParameterValue: "1"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.GsmBtsBand", ParameterValue: "GSM900"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.TrxNum", ParameterValue: "2"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.2.InUse", ParameterValue: "0"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.2.GsmCellID", ParameterValue: "1002"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.2.CurrLocAreaCode", ParameterValue: "2002"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.2.CurrentArfcn", ParameterValue: "47"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.2.OpState", ParameterValue: "1"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.3.GsmCellID", ParameterValue: "1003"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.3.InUse", ParameterValue: "true"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.3.CurrLocAreaCode", ParameterValue: "2003"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.3.CurrentArfcn", ParameterValue: "49"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.3.OpState", ParameterValue: "0"},
	}

	cells := AssembleGSMCells(params)
	assert.Len(t, cells, 2)
	assert.Equal(t, 1, cells[0].Index)
	assert.Equal(t, "1001", cells[0].CellID)
	assert.Equal(t, "2001", cells[0].LAC)
	assert.Equal(t, "45", cells[0].ARFCN)
	assert.Equal(t, "1", cells[0].OpState)
	assert.Equal(t, "1", cells[0].RFTxStatus)
	assert.Equal(t, "GSM900", cells[0].Band)
	assert.Equal(t, 2, cells[0].BTSNum)

	assert.Equal(t, 3, cells[1].Index)
	assert.Equal(t, "1003", cells[1].CellID)
	assert.Equal(t, "2003", cells[1].LAC)
	assert.Equal(t, "49", cells[1].ARFCN)
	assert.Equal(t, "0", cells[1].OpState)
}

func TestAssembleGSMCells_FallbackWithoutInUse(t *testing.T) {
	params := []model.DeviceParameter{
		{ParameterPath: "Device.Services.GsmBTSCellDT.1.GsmCellID", ParameterValue: "1001"},
		{ParameterPath: "Device.Services.GsmBTSCellDT.3.GsmCellID", ParameterValue: "1003"},
	}

	cells := AssembleGSMCells(params)
	assert.Len(t, cells, 2)
	assert.Equal(t, 1, cells[0].Index)
	assert.Equal(t, 3, cells[1].Index)
}

func TestExtractIndexAndField(t *testing.T) {
	tests := []struct {
		path      string
		marker    string
		wantIdx   int
		wantField string
	}{
		{
			path:      "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.3.MME1Status",
			marker:    "MmePoolConfigParam.",
			wantIdx:   3,
			wantField: "MME1Status",
		},
		{
			path:      "Device.DeviceInfo.X_COM_LICENSE.Capacity.12.RemainingPeriod",
			marker:    "Capacity.",
			wantIdx:   12,
			wantField: "RemainingPeriod",
		},
		{
			path:    "Device.DeviceInfo.SoftwareVersion",
			marker:  "Capacity.",
			wantIdx: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			idx, field := extractIndexAndField(tt.path, tt.marker)
			assert.Equal(t, tt.wantIdx, idx)
			if tt.wantIdx > 0 {
				assert.Equal(t, tt.wantField, field)
			}
		})
	}
}
