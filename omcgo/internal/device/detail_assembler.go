package device

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

const nrMultiPlmnEnablePath = "Device.Services.FAPService.1.FAPControl.NR.MultiPlmnEnable"

const (
	amfsStatusPath          = "Device.Services.FAPService.1.AmfsStatus"
	gpsSoftVersionPath      = "Device.FAP.GPS.SoftVersion"
	ppsTimeModePath         = "Device.FAP.Synchronization.PpsTimeMode"
	systemBackupVersionPath = "Device.SoftwareCtrl.SystemBackupVersion"
)

var ethernetInterfaceStatusPath = regexp.MustCompile(`^Device\.Ethernet\.Interface\.(\d+)\.Status$`)
var ipInterfaceStatusPath = regexp.MustCompile(`^Device\.IP\.Interface\.(\d+)\.Status$`)

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

// AssembleMultiPlmnEnable normalizes the NR Multi PLMN switch from device_parameters.
// Supported raw values: 1/true/enabled/on and 0/false/disabled/off.
func AssembleMultiPlmnEnable(params []model.DeviceParameter) string {
	for _, p := range params {
		if p.ParameterPath != nrMultiPlmnEnablePath {
			continue
		}
		return normalizeBinaryState(p.ParameterValue)
	}
	return ""
}

func AssembleAMFStatus(params []model.DeviceParameter) string {
	for _, p := range params {
		if p.ParameterPath == amfsStatusPath && p.ParameterValue != "" {
			return normalizeAMFStatus(p.ParameterValue)
		}
	}
	return ""
}

func AssembleGPSVersion(params []model.DeviceParameter) string {
	for _, p := range params {
		if p.ParameterPath == gpsSoftVersionPath && p.ParameterValue != "" {
			return p.ParameterValue
		}
	}
	return ""
}

func AssemblePPSTimeMode(params []model.DeviceParameter) string {
	for _, p := range params {
		if p.ParameterPath == ppsTimeModePath && p.ParameterValue != "" {
			return p.ParameterValue
		}
	}
	return ""
}

func AssembleRollbackVersion(params []model.DeviceParameter) string {
	for _, p := range params {
		if p.ParameterPath == systemBackupVersionPath && p.ParameterValue != "" {
			return p.ParameterValue
		}
	}
	return ""
}

func AssembleWANStatus(params []model.DeviceParameter) string {
	paramValues := make(map[string]string, len(params))
	for _, p := range params {
		if p.ParameterValue == "" {
			continue
		}
		paramValues[p.ParameterPath] = p.ParameterValue
	}

	bestIndex := ""
	bestScore := -1
	bestStatus := ""
	for path, value := range paramValues {
		index := ""
		score := -1

		if matches := ethernetInterfaceStatusPath.FindStringSubmatch(path); matches != nil {
			index = matches[1]
			score = scoreEthernetInterfaceForDetail(index, paramValues)
		} else if matches := ipInterfaceStatusPath.FindStringSubmatch(path); matches != nil {
			index = matches[1]
			score = scoreIPInterfaceForDetail(index, paramValues)
		} else {
			continue
		}

		if score > bestScore || (score == bestScore && bestIndex != "" && index < bestIndex) || bestIndex == "" {
			bestIndex = index
			bestScore = score
			bestStatus = normalizeWANStatus(value)
		}
	}

	return bestStatus
}

func scoreEthernetInterfaceForDetail(index string, paramValues map[string]string) int {
	score := 0
	prefix := "Device.Ethernet.Interface." + index + "."

	for _, suffix := range []string{"Name", "UserLabel", "PortLocation"} {
		if containsWANKeywordForDetail(paramValues[prefix+suffix]) {
			score += 20
		}
	}

	for path, value := range paramValues {
		if value == "" || !strings.HasPrefix(path, prefix) {
			continue
		}
		if strings.HasSuffix(path, ".PortType") && containsWANKeywordForDetail(value) {
			score += 100
			continue
		}
		if strings.HasSuffix(path, ".IPAddress") || strings.HasSuffix(path, ".DefaultGateway") {
			score += 5
		}
	}

	return score
}

func scoreIPInterfaceForDetail(index string, paramValues map[string]string) int {
	score := 1
	prefix := "Device.IP.Interface." + index + "."

	for path := range paramValues {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		if strings.Contains(path, ".IPv4Address.") || strings.Contains(path, ".IPv6Address.") {
			score += 5
		}
		if strings.HasSuffix(path, ".DefaultGateway") {
			score += 5
		}
	}

	return score
}

func containsWANKeywordForDetail(val string) bool {
	if val == "" {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(val))
	for _, keyword := range []string{"wan", "uplink", "up-link", "internet", "external"} {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func normalizeWANStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "up", "connected", "1", "true":
		return "up"
	case "down", "disconnected", "0", "false":
		return "down"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func normalizeAMFStatus(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if strings.Contains(trimmed, ";") || strings.Contains(trimmed, "=") {
		allZero := true
		for _, segment := range strings.Split(trimmed, ";") {
			segment = strings.TrimSpace(segment)
			if segment == "" {
				continue
			}

			statusValue := segment
			if idx := strings.LastIndex(segment, "="); idx >= 0 {
				statusValue = strings.TrimSpace(segment[idx+1:])
			}

			switch strings.ToLower(statusValue) {
			case "1", "true", "connected", "up":
				return "connected"
			case "0", "false", "disconnected", "down":
				continue
			default:
				allZero = false
			}
		}
		if allZero {
			return "disconnected"
		}
	}

	switch strings.ToLower(trimmed) {
	case "1", "true", "connected", "up":
		return "connected"
	case "0", "false", "disconnected", "down":
		return "disconnected"
	default:
		return trimmed
	}
}

func normalizeBinaryState(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "enabled", "on":
		return "enabled"
	case "0", "false", "disabled", "off":
		return "disabled"
	default:
		return ""
	}
}

// AssembleCells builds cell info for multi-carrier devices from device_parameters.
// numOfCells determines how many FAPService instances to look for.
// params should be the full device parameters (or a broad prefix query result).
func AssembleCells(params []model.DeviceParameter, numOfCells int) []CellInfo {
	if numOfCells <= 0 {
		numOfCells = 1
	}

	// 某些设备的 num_of_cells 可能滞后于实际上报参数（例如仍为 1，
	// 但 device_parameters 已存在 FAPService.2~N）；这里按参数路径探测最大实例号兜底。
	if detected := detectMaxFAPServiceIndex(params); detected > numOfCells {
		numOfCells = detected
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
		nrIndexedPrefix := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.1.NR.", i)
		ctrlPrefix := fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.", i)

		cell := CellInfo{Index: i}

		// ECI
		cell.CellID = paramMap[ltePrefix+"RAN.Common.CellIdentity"]
		cell.ECI = cell.CellID
		if cell.ECI == "" {
			cell.CellID = paramMap[nrIndexedPrefix+"CN.TA.1.NrcellIdentity"]
			if cell.CellID == "" {
				cell.CellID = paramMap[nrIndexedPrefix+"RAN.Common.CellLocalId"]
			}
			if cell.CellID == "" {
				cell.CellID = paramMap[nrPrefix+"RAN.Common.CellLocalId"]
			}
			cell.ECI = cell.CellID
		}

		// PCI
		cell.PCI = paramMap[ltePrefix+"RAN.RF.PhyCellID"]
		if cell.PCI == "" {
			cell.PCI = paramMap[nrIndexedPrefix+"RAN.RF.PhyCellID"]
		}
		if cell.PCI == "" {
			cell.PCI = paramMap[nrPrefix+"RAN.RF.NRPCI"]
		}

		// FreqPoint
		cell.FreqPoint = paramMap[ltePrefix+"RAN.RF.EARFCNDL"]
		if cell.FreqPoint == "" {
			cell.FreqPoint = paramMap[ltePrefix+"RAN.Common.EARFCNDL"]
		}
		if cell.FreqPoint == "" {
			cell.FreqPoint = paramMap[nrIndexedPrefix+"RAN.RF.NRARFCNDL"]
		}
		if cell.FreqPoint == "" {
			cell.FreqPoint = paramMap[nrPrefix+"RAN.Common.NRARFCN"]
		}

		// Bandwidth
		cell.Bandwidth = paramMap[ltePrefix+"RAN.RF.DLBandwidth"]
		if cell.Bandwidth == "" {
			cell.Bandwidth = paramMap[nrIndexedPrefix+"RAN.RF.ChannelBandwidth"]
		}
		if cell.Bandwidth == "" {
			cell.Bandwidth = paramMap[nrPrefix+"RAN.RF.ChannelBandwidth"]
		}

		// Band
		cell.Band = paramMap[ltePrefix+"RAN.RF.FreqBandIndicator"]
		if cell.Band == "" {
			cell.Band = paramMap[nrIndexedPrefix+"RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.1.FreqBandIndicatorNR"]
		}

		// OpState — Baicells 等 BaiBLQ 设备实际上报 CellOpState（FAPControl 子树）
		// 老 OpState 后缀保留作为向后兼容（部分设备/早期固件可能仅有 OpState）
		// 设计文档 §3.3。
		cell.OpState = paramMap[ctrlPrefix+"LTE.CellOpState"]
		if cell.OpState == "" {
			cell.OpState = paramMap[ctrlPrefix+"LTE.OpState"]
		}
		if cell.OpState == "" {
			cell.OpState = paramMap[nrIndexedPrefix+"RAN.OpState"]
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
			cell.RFTxStatus = paramMap[nrIndexedPrefix+"RAN.rftxEnable"]
		}
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
		if cell.RFTxStatus == "" {
			// 某些 LTE 设备只上报 RU 级射频状态，且无更细粒度 cell 关联；
			// 仅当所有 RU 值一致时才回填到 cell，避免多 RU 状态冲突时误导前端。
			cell.RFTxStatus = uniformDeviceInfoRUValue(paramMap, "RFTxStatus")
		}

		// AdminState
		cell.AdminState = paramMap[ctrlPrefix+"LTE.AdminState"]
		if cell.AdminState == "" {
			cell.AdminState = paramMap[nrIndexedPrefix+"RAN.CellEnable.AdminState"]
		}
		if cell.AdminState == "" {
			cell.AdminState = paramMap[ctrlPrefix+"NR.AdminState"]
		}
		if cell.AdminState == "" {
			// 某些 LTE 设备只上报一个全局 AdminState（如 FAPService.1），
			// 多小区共享该状态时回填到所有 cell 行，避免后续小区显示为空。
			cell.AdminState = uniformFAPControlValue(paramMap, "LTE.AdminState")
		}

		cells = append(cells, cell)
	}
	return cells
}

// AssembleGSMCells builds BM GSM cell info from Device.Services.GsmBTSCellDT.{i}.*
// object parameters. The returned rows preserve the TR-069 instance index so the
// frontend can align them with quick-settings visibility rules.
func AssembleGSMCells(params []model.DeviceParameter) []CellInfo {
	maxIdx := detectMaxGSMBTSCellIndex(params)
	if maxIdx <= 0 {
		return nil
	}

	paramMap := make(map[string]string, len(params))
	hasAnyGSMInUse := false
	for _, p := range params {
		paramMap[p.ParameterPath] = p.ParameterValue
		if strings.HasSuffix(p.ParameterPath, ".InUse") && strings.Contains(p.ParameterPath, "Device.Services.GsmBTSCellDT.") {
			hasAnyGSMInUse = true
		}
	}

	var cells []CellInfo
	for i := 1; i <= maxIdx; i++ {
		prefix := fmt.Sprintf("Device.Services.GsmBTSCellDT.%d.", i)
		if hasAnyGSMInUse && normalizeBinaryState(paramMap[prefix+"InUse"]) != "enabled" {
			continue
		}
		cell := CellInfo{
			Index:      i,
			CellID:     paramMap[prefix+"GsmCellID"],
			Band:       paramMap[prefix+"GsmBtsBand"],
			OpState:    paramMap[prefix+"OpState"],
			RFTxStatus: paramMap[prefix+"RfState"],
			LAC:        paramMap[prefix+"CurrLocAreaCode"],
			ARFCN:      paramMap[prefix+"CurrentArfcn"],
		}
		if trxNum := strings.TrimSpace(paramMap[prefix+"TrxNum"]); trxNum != "" {
			if parsed, err := strconv.Atoi(trxNum); err == nil {
				cell.BTSNum = parsed
			}
		}
		if !hasGSMCellData(cell) {
			continue
		}
		cells = append(cells, cell)
	}

	return cells
}

func uniformFAPControlValue(paramMap map[string]string, field string) string {
	prefix := "Device.Services.FAPService."
	marker := ".FAPControl."
	wantSuffix := "." + field
	var value string
	matched := false

	for path, current := range paramMap {
		if !strings.HasPrefix(path, prefix) || !strings.Contains(path, marker) || !strings.HasSuffix(path, wantSuffix) {
			continue
		}
		if !matched {
			value = current
			matched = true
			continue
		}
		if current != value {
			return ""
		}
	}

	if !matched {
		return ""
	}
	return value
}

func uniformDeviceInfoRUValue(paramMap map[string]string, field string) string {
	prefix := "Device.DeviceInfo.RU."
	wantSuffix := "." + field
	var value string
	matched := false

	for path, current := range paramMap {
		if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, wantSuffix) {
			continue
		}
		if !matched {
			value = current
			matched = true
			continue
		}
		if current != value {
			return ""
		}
	}

	if !matched {
		return ""
	}
	return value
}

func detectMaxFAPServiceIndex(params []model.DeviceParameter) int {
	const prefix = "Device.Services.FAPService."

	maxIdx := 0
	for _, p := range params {
		path := p.ParameterPath
		if !strings.HasPrefix(path, prefix) {
			continue
		}

		rest := strings.TrimPrefix(path, prefix)
		dot := strings.Index(rest, ".")
		if dot <= 0 {
			continue
		}

		idx, err := strconv.Atoi(rest[:dot])
		if err != nil || idx <= 0 {
			continue
		}

		if strings.Contains(path, ".CellConfig.LTE.") ||
			strings.Contains(path, ".CellConfig.NR.") ||
			strings.Contains(path, ".FAPControl.LTE.") ||
			strings.Contains(path, ".FAPControl.NR.") {
			if idx > maxIdx {
				maxIdx = idx
			}
		}
	}

	return maxIdx
}

func detectMaxGSMBTSCellIndex(params []model.DeviceParameter) int {
	const prefix = "Device.Services.GsmBTSCellDT."

	maxIdx := 0
	for _, p := range params {
		path := p.ParameterPath
		if !strings.HasPrefix(path, prefix) {
			continue
		}

		rest := strings.TrimPrefix(path, prefix)
		dot := strings.Index(rest, ".")
		if dot <= 0 {
			continue
		}

		idx, err := strconv.Atoi(rest[:dot])
		if err != nil || idx <= 0 {
			continue
		}
		if idx > maxIdx {
			maxIdx = idx
		}
	}

	return maxIdx
}

func hasGSMCellData(cell CellInfo) bool {
	return cell.CellID != "" ||
		cell.Band != "" ||
		cell.OpState != "" ||
		cell.RFTxStatus != "" ||
		cell.LAC != "" ||
		cell.ARFCN != "" ||
		cell.BTSNum > 0
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
