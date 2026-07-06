package device

import (
	"fmt"
	"strconv"
	"strings"
)

// CalcCellStatus computes the list-page activation summary from device_parameters.
// It intentionally follows device detail page semantics: iterate all cells/FAPService
// instances and treat the device as active when any cell OpState is active.
//
// Logic:
//
//	Any cell OpState active   → "normal"
//	All observed cells inactive → "inactive"
//	No cell data                → "inactive"
//
// 覆盖制式（严格路径）:
//   - LTE: Device.Services.FAPService.{i}.FAPControl.LTE.OpState
//   - NR:  Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.OpState
//   - GSM: Device.Services.GsmBTSCellDT.{i}.OpState
func CalcCellStatus(params map[string]string) string {
	// GSM 制式并行检查——走 GsmBTSCellDT.{i}.* 对象树,与 FAPService 体系独立,
	// 任一 GSM cell active 也算设备激活(不依赖 FAPService maxIndex)。
	if gsmActive, ok := hasActiveGSMCell(params); ok && gsmActive {
		return "normal"
	}

	maxIndex := detectMaxFAPServiceIndexFromMap(params)
	if count := CalcNumOfCells(params); count > maxIndex {
		maxIndex = count
	}
	if maxIndex <= 0 {
		maxIndex = 1
	}

	for i := 1; i <= maxIndex; i++ {
		isActive, ok := isCellActiveForIndex(params, i)
		if !ok {
			continue
		}
		if isActive {
			return "normal"
		}
	}
	return "inactive"
}

// CalcOpState 派生设备级“激活状态”（前端设备列表 op_state 列专用）。
//
// 口径与 CalcCellStatus 同源（同一轮 isCellActiveForIndex 遍历），仅取值不同：
//
//	CalcCellStatus 返 "normal"  → CalcOpState 返 "1" （激活）
//	其它一切                        → CalcOpState 返 "0" （未激活，含无 cell 数据）
//
// 该口径取代了 model.DeriveOpStateActivated(first_online_time) 这种“一次性持久”语义、
// 表达“设备当前是否在运营”。InfoSyncer 在 sync_from_parameters 中同步 device_info.op_state。
func CalcOpState(params map[string]string) string {
	if CalcCellStatus(params) == "normal" {
		return "1"
	}
	return "0"
}

// CalcMMEStatus computes the mme_status quick-query column from device_parameters.
// Iterates MmePoolConfigParam.{1-16}.MME1Status, counts active connections.
//
// Returns:
//
//	"disconnected" — no active MME
//	"partial"      — 1 active MME
//	"connected"    — 2+ active MMEs
func CalcMMEStatus(params map[string]string) string {
	activeCount := 0
	for i := 1; i <= 16; i++ {
		prefix := fmt.Sprintf("Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.%d.", i)
		if params[prefix+"MME1Status"] == "1" {
			activeCount++
		}
		// Also check NR path
		nrPrefix := fmt.Sprintf("Device.Services.FAPService.1.CellConfig.NR.Core.MmePoolConfigParam.%d.", i)
		if params[nrPrefix+"MME1Status"] == "1" {
			activeCount++
		}
	}
	switch {
	case activeCount == 0:
		return "disconnected"
	case activeCount < 2:
		return "partial"
	default:
		return "connected"
	}
}

// CalcLicenseStatus computes the license_status quick-query column from device_parameters.
// Iterates X_COM_LICENSE.Capacity.{1-32}, checks State and RemainingPeriod.
//
// Returns:
//
//	"expired"  — no active license capacity
//	"expiring" — min remaining period <= 30 days
//	"active"   — all capacities healthy
func CalcLicenseStatus(params map[string]string) string {
	hasActive := false
	minRemain := int(^uint(0) >> 1) // max int
	for i := 1; i <= 32; i++ {
		prefix := fmt.Sprintf("Device.DeviceInfo.X_COM_LICENSE.Capacity.%d.", i)
		state := params[prefix+"State"]
		if state == "1" || strings.EqualFold(state, "active") {
			hasActive = true
			if remainStr := params[prefix+"RemainingPeriod"]; remainStr != "" {
				if remain, err := strconv.Atoi(remainStr); err == nil && remain < minRemain {
					minRemain = remain
				}
			}
		}
	}
	switch {
	case !hasActive:
		return "expired"
	case minRemain <= 30:
		return "expiring"
	default:
		return "active"
	}
}

// CalcSyncStatus computes the sync_status quick-query column from device_parameters.
// Checks GPS, BDS, GLONASS, 1588v2, and tfcsSyncState to determine sync source.
// 对于设备直接上报的文本态（如 LOCKED / HOLDOVER / SYNCED），保留原始值，
// 避免被错误折叠成 error。
func CalcSyncStatus(params map[string]string) string {
	if nrSyncStatus := firstNonEmpty(
		params["Device.Services.FAPService.1.FAPControl.PLLSyncState"],
		params["Device.FAP.Synchronization.ClockSourceSyncState"],
	); nrSyncStatus != "" {
		return nrSyncStatus
	}

	tfcsSync := firstNonEmpty(
		params["Device.ManagementServer.tfcsSyncState"],
		params["Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState"],
	)
	if tfcsSync == "" {
		tfcsSync = params["Device.Services.FAPService.1.FAPControl.NR.Gateway.X_COM_tfcsSyncState"]
	}
	if tfcsSync != "" && !isBooleanLikeValue(tfcsSync) {
		return tfcsSync
	}

	gpsStatus := firstNonEmpty(
		params["Device.DeviceInfo.X_COM_GPS_Status"],
		params["Device.DeviceInfo.GPS_Status"],
		params["Device.FAP.GPS.SyncStatus"],
	)
	bdsStatus := firstNonEmpty(
		params["Device.DeviceInfo.X_COM_BDS_Status"],
		params["Device.DeviceInfo.BDS_Status"],
	)
	ieee1588Status := firstNonEmpty(
		params["Device.DeviceInfo.X_COM_1588_Status"],
		params["Device.DeviceInfo.1588_Status"],
		params["Device.FAP.PTP1588.SyncStatus"],
	)

	// If tfcsSync indicates locked/synced, determine the source
	if isTrueValue(tfcsSync) || tfcsSync == "1" {
		if isTrueValue(gpsStatus) || gpsStatus == "1" || isSynchronizedValue(gpsStatus) {
			return "gps"
		}
		if isTrueValue(bdsStatus) || bdsStatus == "1" || isSynchronizedValue(bdsStatus) {
			return "beidou"
		}
		if isTrueValue(ieee1588Status) || ieee1588Status == "1" || isSynchronizedValue(ieee1588Status) {
			return "ntp"
		}
		return "gps" // default sync source when locked
	}

	// Not synced — check individual sources for partial status
	if isTrueValue(gpsStatus) || gpsStatus == "1" || isSynchronizedValue(gpsStatus) {
		return "gps"
	}
	if isTrueValue(bdsStatus) || bdsStatus == "1" || isSynchronizedValue(bdsStatus) {
		return "beidou"
	}

	return "error"
}

// CalcGPSStatus computes the gps_status quick-query column from device_parameters.
func CalcGPSStatus(params map[string]string) string {
	status := params["Device.DeviceInfo.X_COM_GPS_Status"]
	switch {
	case status == "1" || strings.EqualFold(status, "true"):
		return "normal"
	case status == "0" || strings.EqualFold(status, "false"):
		return "abnormal"
	case status == "":
		return "no_signal"
	default:
		return "abnormal"
	}
}

// CalcRFStatus computes the rf_status quick-query column from device_parameters.
// Combines RFTxStatus and RadioEnable for a comprehensive RF status.
func CalcRFStatus(params map[string]string) string {
	rfTx := params["Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.RFTxStatus"]
	if rfTx == "" {
		rfTx = params["Device.Services.FAPService.1.CellConfig.NR.RAN.RF.RFTxStatus"]
	}
	radioEnable := params["Device.Services.FAPService.1.FAPControl.LTE.AdminState"]
	if radioEnable == "" {
		radioEnable = params["Device.Services.FAPService.1.FAPControl.NR.AdminState"]
	}

	if !isTrueValue(radioEnable) {
		return "off"
	}
	if isTrueValue(rfTx) || rfTx == "1" {
		return "on"
	}
	if rfTx == "0" || strings.EqualFold(rfTx, "false") {
		return "error"
	}
	// RadioEnable is on but no RFTxStatus data
	return "on"
}

// CalcNumOfCells extracts the number of cells (carriers) from device_parameters.
func CalcNumOfCells(params map[string]string) int {
	val := params["Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells"]
	if val == "" {
		val = params["Device.Services.FAPService.1.CellConfig.NR.RAN.CA.PARAMS.NumOfCells"]
	}
	if val == "" {
		return 1 // default single carrier
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// isTrueValue checks if a TR069 parameter value represents a boolean true.
func isTrueValue(v string) bool {
	return v == "1" || strings.EqualFold(v, "true")
}

func isSynchronizedValue(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), "synchronized") || strings.EqualFold(strings.TrimSpace(v), "synced")
}

// isCellActiveForIndex 仅处理 LTE/NR 两种 FAPService 路径；GSM 由 hasActiveGSMCell 独立判定。
func isCellActiveForIndex(params map[string]string, index int) (bool, bool) {
	ltePath := fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.OpState", index)
	nrPath := fmt.Sprintf("Device.Services.FAPService.1.CellConfig.%d.NR.RAN.OpState", index)

	lteOpStates := []string{params[ltePath]}
	nrOpStates := []string{params[nrPath]}

	lteActive, lteObserved := anyOpStateActive(lteOpStates)
	if lteActive {
		return true, true
	}

	nrActive, nrObserved := anyOpStateActive(nrOpStates)
	if nrActive {
		return true, true
	}

	if !(lteObserved || nrObserved) {
		return false, false
	}
	return false, true
}

func anyOpStateActive(values []string) (active bool, observed bool) {
	for _, cellOpState := range values {
		trimmed := strings.TrimSpace(cellOpState)
		if trimmed == "" {
			continue
		}
		observed = true
		if isTrueValue(trimmed) || strings.EqualFold(trimmed, "active") || strings.EqualFold(trimmed, "enabled") {
			return true, true
		}
	}
	return false, observed
}

// hasActiveGSMCell 遍历 GsmBTSCellDT.{i}.OpState 判定是否有 GSM cell active。
//
// GSM 制式在 BM (BaseManager) 形态下走 Device.Services.GsmBTSCellDT.{i}.OpState。
// 仅当 InUse=true 时，该 cell 才算有效并纳入判定；InUse=false 或缺失都不处理。
//
// 返回:
//   - active=true,observed=true   —— 至少一个 GSM cell active
//   - active=false,observed=true  —— 有 GSM cell 但全未激活
//   - active=false,observed=false —— 未观测到 GSM 参数 (非 GSM 设备)
func hasActiveGSMCell(params map[string]string) (active, observed bool) {
	const prefix = "Device.Services.GsmBTSCellDT."

	cellIdx := map[int]struct{}{}
	for path := range params {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		if !(strings.HasSuffix(path, ".InUse") || strings.HasSuffix(path, ".OpState")) {
			continue
		}
		rest := strings.TrimPrefix(path, prefix)
		dot := strings.Index(rest, ".")
		if dot <= 0 {
			continue
		}
		idx, err := strconv.Atoi(rest[:dot])
		if err == nil && idx > 0 {
			cellIdx[idx] = struct{}{}
		}
	}
	if len(cellIdx) == 0 {
		return false, false
	}

	observed = true
	for i := range cellIdx {
		cellPrefix := fmt.Sprintf("%s%d.", prefix, i)
		inUse := strings.TrimSpace(params[cellPrefix+"InUse"])
		if inUse == "" || (!isTrueValue(inUse) && !strings.EqualFold(inUse, "enabled")) {
			continue
		}
		cellOp := strings.TrimSpace(params[cellPrefix+"OpState"])
		if cellOp == "" {
			continue
		}
		if isTrueValue(cellOp) || strings.EqualFold(cellOp, "active") || strings.EqualFold(cellOp, "enabled") {
			return true, true
		}
	}
	return false, observed
}

func detectMaxFAPServiceIndexFromMap(params map[string]string) int {
	maxIndex := 0
	for path := range params {
		const prefix = "Device.Services.FAPService."
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		remainder := path[len(prefix):]
		dot := strings.IndexByte(remainder, '.')
		if dot <= 0 {
			continue
		}
		index, err := strconv.Atoi(remainder[:dot])
		if err == nil && index > maxIndex {
			maxIndex = index
		}

		const nrPrefix = "Device.Services.FAPService.1.CellConfig."
		if !strings.HasPrefix(path, nrPrefix) || !strings.Contains(path, ".NR.") {
			continue
		}
		remainder = path[len(nrPrefix):]
		dot = strings.IndexByte(remainder, '.')
		if dot <= 0 {
			continue
		}
		cellIndex, err := strconv.Atoi(remainder[:dot])
		if err == nil && cellIndex > maxIndex {
			maxIndex = cellIndex
		}
	}
	return maxIndex
}

func isBooleanLikeValue(v string) bool {
	return v == "1" || v == "0" || strings.EqualFold(v, "true") || strings.EqualFold(v, "false")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
// CalcUECount 从 device_parameters 读取当前接入 UE 数。
//
// 读取优先级：
//  1. Device.DeviceInfo.UE_Count          （通用标准路径，param_models 已收录）
//  2. Device.Services.FAPService.1.X_COM_ConnectedUECount  （LTE 厂商扩展路径）
//  3. 返回 0                              （参数缺失或解析失败）
//
// 返回值保证 >= 0；非数字字符串或负数均视为 0。
func CalcUECount(params map[string]string) int {
	candidates := []string{
		"Device.DeviceInfo.UE_Count",
		"Device.Services.FAPService.1.X_COM_ConnectedUECount",
	}
	for _, path := range candidates {
		v, ok := params[path]
		if !ok || v == "" {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || n < 0 {
			continue
		}
		return n
	}
	return 0
}