package device

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

// AssembleMMEPool builds MME pool entries from device_parameters with the MME prefix.
// params should be the result of GetByPathPrefix("...MmePoolConfigParam.").
func AssembleMMEPool(params []model.DeviceParameter) []MMEEntry {
	// Group params by index: extract the index from path like ...MmePoolConfigParam.{N}.{field}
	type mmeRaw struct {
		ip     string
		status string
		plmnID string
	}
	grouped := make(map[int]*mmeRaw)

	for _, p := range params {
		idx, field := extractIndexAndField(p.ParameterPath, "MmePoolConfigParam.")
		if idx == 0 {
			continue
		}
		if grouped[idx] == nil {
			grouped[idx] = &mmeRaw{}
		}
		switch {
		case strings.HasSuffix(field, "MME1Status"):
			grouped[idx].status = p.ParameterValue
		// MMEIp1 / MMEIp2: Baicells BaiBLQ 等设备实际上报路径
		//   Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.{N}.MMEIp1
		// MME1Address / MME1IP: 部分设备/早期固件使用的路径（保留向后兼容）
		case strings.HasSuffix(field, "MMEIp1") ||
			strings.HasSuffix(field, "MME1Address") ||
			strings.HasSuffix(field, "MME1IP"):
			grouped[idx].ip = p.ParameterValue
		case strings.HasSuffix(field, "PLMNID"):
			grouped[idx].plmnID = p.ParameterValue
		}
	}

	var entries []MMEEntry
	for idx, raw := range grouped {
		if raw.ip == "" && raw.status == "" {
			continue
		}
		status := "inactive"
		if raw.status == "1" {
			status = "active"
		}
		entries = append(entries, MMEEntry{
			Index:  idx,
			IP:     raw.ip,
			Status: status,
			PLMNID: raw.plmnID,
		})
	}
	return entries
}

// AssembleLicenseDetail builds license detail from device_parameters with the License prefix.
// params should be the result of GetByPathPrefix("...X_COM_LICENSE.").
func AssembleLicenseDetail(params []model.DeviceParameter) *LicenseDetail {
	if len(params) == 0 {
		return nil
	}

	detail := &LicenseDetail{}
	capMap := make(map[int]*LicenseCapacity)

	for _, p := range params {
		path := p.ParameterPath
		if strings.Contains(path, "LicenseCode") {
			detail.Code = p.ParameterValue
			continue
		}
		if strings.Contains(path, "GenerateDate") {
			detail.GenerateDate = p.ParameterValue
			continue
		}

		idx, field := extractIndexAndField(path, "Capacity.")
		if idx == 0 {
			continue
		}
		if capMap[idx] == nil {
			capMap[idx] = &LicenseCapacity{Index: idx}
		}
		cap := capMap[idx]
		switch {
		case strings.HasSuffix(field, "Description"):
			cap.Description = p.ParameterValue
		case strings.HasSuffix(field, "State"):
			cap.State = p.ParameterValue
		// Baicells BaiBLQ 等设备实际上报 Capacity.{N}.Value 字段（容量数值），
		// 不上报 State；前端凭 RemainingPeriod>0 派生 active/expired 文案。
		// 设计文档 §3.3。
		case strings.HasSuffix(field, "Value"):
			cap.Value = p.ParameterValue
		case strings.HasSuffix(field, "ValidPeriod"):
			cap.ValidPeriod, _ = strconv.Atoi(p.ParameterValue)
		case strings.HasSuffix(field, "RemainingPeriod"):
			cap.RemainingPeriod, _ = strconv.Atoi(p.ParameterValue)
		}
	}

	for _, cap := range capMap {
		detail.Capacities = append(detail.Capacities, *cap)
	}

	if detail.Code == "" && len(detail.Capacities) == 0 {
		return nil
	}
	return detail
}

// AssembleAntennaInfo builds antenna info from device_parameters with the AntennaInfo prefix.
// params should be the result of GetByPathPrefix("...AntennaInfo.").
func AssembleAntennaInfo(params []model.DeviceParameter) *AntennaInfo {
	if len(params) == 0 {
		return nil
	}

	info := &AntennaInfo{}
	for _, p := range params {
		path := p.ParameterPath
		switch {
		case strings.HasSuffix(path, "Azimuth"):
			info.Azimuth = p.ParameterValue
		case strings.HasSuffix(path, "Beamwidth"):
			info.Beamwidth = p.ParameterValue
		case strings.HasSuffix(path, "Downtilt"):
			info.Downtilt = p.ParameterValue
		case strings.HasSuffix(path, "Gain"):
			info.Gain = p.ParameterValue
		case strings.HasSuffix(path, "Height"):
			info.Height = p.ParameterValue
		case strings.HasSuffix(path, "HeightType"):
			info.HeightType = p.ParameterValue
		}
	}

	if info.Azimuth == "" && info.Gain == "" && info.Height == "" {
		return nil
	}
	return info
}

// AssembleCells builds cell info for multi-carrier devices from device_parameters.
// numOfCells determines how many FAPService instances to look for.
// params should be the full device parameters (or a broad prefix query result).
func AssembleCells(params []model.DeviceParameter, numOfCells int) []CellInfo {
	if numOfCells <= 0 {
		numOfCells = 1
	}

	// Build a lookup map for fast access
	paramMap := make(map[string]string, len(params))
	for _, p := range params {
		paramMap[p.ParameterPath] = p.ParameterValue
	}

	var cells []CellInfo
	for i := 1; i <= numOfCells; i++ {
		// Try LTE paths first, then NR
		ltePrefix := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.LTE.", i)
		nrPrefix := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.NR.", i)
		ctrlPrefix := fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.", i)

		cell := CellInfo{Index: i}

		// ECI
		cell.ECI = paramMap[ltePrefix+"RAN.Common.CellIdentity"]
		if cell.ECI == "" {
			cell.ECI = paramMap[nrPrefix+"RAN.Common.CellLocalId"]
		}

		// PCI
		cell.PCI = paramMap[ltePrefix+"RAN.RF.PhyCellID"]
		if cell.PCI == "" {
			cell.PCI = paramMap[nrPrefix+"RAN.RF.NRPCI"]
		}

		// FreqPoint
		cell.FreqPoint = paramMap[ltePrefix+"RAN.Common.EARFCNDL"]
		if cell.FreqPoint == "" {
			cell.FreqPoint = paramMap[nrPrefix+"RAN.Common.NRARFCN"]
		}

		// Bandwidth
		cell.Bandwidth = paramMap[ltePrefix+"RAN.RF.DLBandwidth"]
		if cell.Bandwidth == "" {
			cell.Bandwidth = paramMap[nrPrefix+"RAN.RF.ChannelBandwidth"]
		}

		// OpState — Baicells 等 BaiBLQ 设备实际上报 CellOpState（FAPControl 子树）
		// 老 OpState 后缀保留作为向后兼容（部分设备/早期固件可能仅有 OpState）
		// 设计文档 §3.3。
		cell.OpState = paramMap[ctrlPrefix+"LTE.CellOpState"]
		if cell.OpState == "" {
			cell.OpState = paramMap[ctrlPrefix+"LTE.OpState"]
		}
		if cell.OpState == "" {
			cell.OpState = paramMap[ctrlPrefix+"NR.CellOpState"]
		}
		if cell.OpState == "" {
			cell.OpState = paramMap[ctrlPrefix+"NR.OpState"]
		}

		// RFTxStatus — 实际位于 FAPControl 子树（非 RAN.RF）
		// 设计文档 §3.3。
		cell.RFTxStatus = paramMap[ctrlPrefix+"LTE.RFTxStatus"]
		if cell.RFTxStatus == "" {
			cell.RFTxStatus = paramMap[ctrlPrefix+"NR.RFTxStatus"]
		}
		if cell.RFTxStatus == "" {
			// 老路径兜底（保留兼容）
			cell.RFTxStatus = paramMap[ltePrefix+"RAN.RF.RFTxStatus"]
		}
		if cell.RFTxStatus == "" {
			cell.RFTxStatus = paramMap[nrPrefix+"RAN.RF.RFTxStatus"]
		}

		// AdminState
		cell.AdminState = paramMap[ctrlPrefix+"LTE.AdminState"]
		if cell.AdminState == "" {
			cell.AdminState = paramMap[ctrlPrefix+"NR.AdminState"]
		}

		cells = append(cells, cell)
	}
	return cells
}

// extractIndexAndField parses a TR069 path to extract an instance index and the remaining field.
// For example, given path "...MmePoolConfigParam.3.MME1Status" and marker "MmePoolConfigParam.",
// returns (3, "MME1Status").
func extractIndexAndField(path, marker string) (int, string) {
	idx := strings.Index(path, marker)
	if idx < 0 {
		return 0, ""
	}
	rest := path[idx+len(marker):]
	dotIdx := strings.Index(rest, ".")
	if dotIdx < 0 {
		return 0, ""
	}
	n, err := strconv.Atoi(rest[:dotIdx])
	if err != nil {
		return 0, ""
	}
	return n, rest[dotIdx+1:]
}
