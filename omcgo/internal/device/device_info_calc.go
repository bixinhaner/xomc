package device

import (
	"fmt"
	"strconv"
	"strings"
)

// CalcCellStatus computes the cell_status quick-query column from device_parameters.
// Three-dimensional judgment: FAP AdminState + OpState + CellOpState.
//
// Logic:
//
//	AdminState=false                                     → "inactive"
//	AdminState=true && OpState=false                     → "fault"
//	AdminState=true && OpState=true && CellOpState="0"   → "decommissioned"
//	AdminState=true && OpState=true && CellOpState="1"   → "normal"
func CalcCellStatus(params map[string]string) string {
	adminState := params["Device.Services.FAPService.1.FAPControl.LTE.AdminState"]
	if adminState == "" {
		adminState = params["Device.Services.FAPService.1.FAPControl.NR.AdminState"]
	}
	opState := params["Device.Services.FAPService.1.FAPControl.LTE.OpState"]
	if opState == "" {
		opState = params["Device.Services.FAPService.1.FAPControl.NR.OpState"]
	}
	cellOpState := params["Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellOpState"]
	if cellOpState == "" {
		cellOpState = params["Device.Services.FAPService.1.CellConfig.NR.RAN.Common.CellOpState"]
	}

	if !isTrueValue(adminState) {
		return "inactive"
	}
	if !isTrueValue(opState) {
		return "fault"
	}
	if cellOpState == "0" || strings.EqualFold(cellOpState, "false") {
		return "decommissioned"
	}
	if cellOpState == "1" || strings.EqualFold(cellOpState, "true") {
		return "normal"
	}
	// Default when CellOpState is absent
	return "normal"
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
func CalcSyncStatus(params map[string]string) string {
	if nrSyncStatus := firstNonEmpty(
		params["Device.Services.FAPService.1.FAPControl.PLLSyncState"],
		params["Device.FAP.Synchronization.ClockSourceSyncState"],
	); nrSyncStatus != "" {
		return nrSyncStatus
	}

	tfcsSync := params["Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState"]
	if tfcsSync == "" {
		tfcsSync = params["Device.Services.FAPService.1.FAPControl.NR.Gateway.X_COM_tfcsSyncState"]
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
