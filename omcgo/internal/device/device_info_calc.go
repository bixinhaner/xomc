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
func CalcCellStatus(params map[string]string) string {
	maxIndex := detectMaxFAPServiceIndexFromMap(params)
	if count := CalcNumOfCells(params); count > maxIndex {
		maxIndex = count
	}
	if maxIndex <= 0 {
		maxIndex = 1
	}

	hasObservedCell := false

	for i := 1; i <= maxIndex; i++ {
		isActive, ok := isCellActiveForIndex(params, i)
		if !ok {
			continue
		}
		hasObservedCell = true
		if isActive {
			return "normal"
		}
	}
	if !hasObservedCell {
		return "inactive"
	}
	return "inactive"
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

	gpsStatus := params["Device.DeviceInfo.X_COM_GPS_Status"]
	bdsStatus := params["Device.DeviceInfo.X_COM_BDS_Status"]
	ieee1588Status := params["Device.DeviceInfo.X_COM_1588_Status"]

	// If tfcsSync indicates locked/synced, determine the source
	if isTrueValue(tfcsSync) || tfcsSync == "1" {
		if isTrueValue(gpsStatus) || gpsStatus == "1" {
			return "gps"
		}
		if isTrueValue(bdsStatus) || bdsStatus == "1" {
			return "beidou"
		}
		if isTrueValue(ieee1588Status) || ieee1588Status == "1" {
			return "ntp"
		}
		return "gps" // default sync source when locked
	}

	// Not synced — check individual sources for partial status
	if isTrueValue(gpsStatus) || gpsStatus == "1" {
		return "gps"
	}
	if isTrueValue(bdsStatus) || bdsStatus == "1" {
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

func isCellActiveForIndex(params map[string]string, index int) (bool, bool) {
	ctrlPrefix := fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.", index)
	lteConfigPrefix := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.LTE.", index)
	nrConfigPrefix := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.NR.", index)
	nrIndexedConfigPrefix := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.1.NR.", index)

	cellOpState := firstNonEmpty(
		params[ctrlPrefix+"LTE.CellOpState"],
		params[ctrlPrefix+"LTE.OpState"],
		params[lteConfigPrefix+"RAN.Common.CellOpState"],
		params[nrIndexedConfigPrefix+"RAN.OpState"],
		params[nrConfigPrefix+"RAN.OpState"],
		params[ctrlPrefix+"NR.CellOpState"],
		params[ctrlPrefix+"NR.OpState"],
		params[nrConfigPrefix+"RAN.Common.CellOpState"],
	)

	if cellOpState == "" {
		return false, false
	}
	return isTrueValue(cellOpState) || strings.EqualFold(cellOpState, "active"), true
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
