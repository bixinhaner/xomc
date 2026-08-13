package device

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
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
//
// Priority:
//  1. LTE strict path `Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus`
//  2. For each pool instance, prefer the EPC path and fall back across legacy
//     LTE/MLN variants (`MME1Status` or `MMEStatus`) when EPC is empty.
//
// Returns only device-level states: any active MME means connected; all observed
// MMEs inactive means disconnected. The ambiguous "partial" state is normalized
// to connected because the device still has a usable core-network connection.
func CalcMMEStatus(params map[string]string) string {
	if gatewayStatus := strings.TrimSpace(params["Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeStatus"]); gatewayStatus != "" {
		if isConnectedMMEStatus(gatewayStatus) || strings.EqualFold(gatewayStatus, "partial") {
			return "connected"
		}
		return "disconnected"
	}

	hasStatus := false
	for i := 1; i <= 16; i++ {
		status := firstNonEmpty(
			params[fmt.Sprintf("Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.%d.MME1Status", i)],
			params[fmt.Sprintf("Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.%d.MME1Status", i)],
			params[fmt.Sprintf("Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.%d.MME1Status", i)],
			params[fmt.Sprintf("Device.Services.FAPService.MmePoolConfigParam.%d.MMEStatus", i)],
		)
		if status == "" {
			continue
		}

		hasStatus = true
		if isConnectedMMEStatus(status) {
			return "connected"
		}
	}
	if hasStatus {
		return "disconnected"
	}
	return ""
}

func isConnectedMMEStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "1", "true", "connected", "active", "up", "on":
		return true
	default:
		return false
	}
}

// CalcCoreNetworkStatus computes the technology-appropriate core-network
// quick-query status. LTE uses MME paths; NR uses the AMF status report.
func CalcCoreNetworkStatus(params map[string]string, tech model.Technology) string {
	switch tech {
	case model.TechLTE:
		return CalcMMEStatus(params)
	case model.TechNR:
		return normalizeAMFStatus(params[amfsStatusPath])
	default:
		return ""
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

type RFStatusProjectionState string

const (
	RFStatusValid        RFStatusProjectionState = "valid"
	RFStatusUnknown      RFStatusProjectionState = "unknown"
	RFStatusUnsupported  RFStatusProjectionState = "unsupported"
	RFStatusInconsistent RFStatusProjectionState = "inconsistent"
)

type RFStatusProjection struct {
	Status        string
	State         RFStatusProjectionState
	Reason        string
	ExpectedCount int
}

type rfStatusPathFamily struct {
	name                  string
	pattern               *regexp.Regexp
	physicalIndex         func(string) (int, bool)
	requiresExpectedCount bool
}

var (
	nrCellRFStatusPattern     = regexp.MustCompile(`^Device\.Services\.FAPService\.\d+\.CellConfig\.(\d+)\.NR\.RAN\.rftxEnable$`)
	sasCellRFStatusPattern    = regexp.MustCompile(`^Device\.DeviceInfo\.CellConfig\.(\d+)\.SAS\.RadioEnable$`)
	lteFAPRFStatusPattern     = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.FAPControl\.LTE\.RFTxStatus$`)
	ranRFStatusPattern        = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.(?:LTE|NR)\.RAN\.RF\.(?:X_COM_RadioEnable|AdminCellState)$`)
	ltePrivateRFStatusPattern = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.LTE\.RAN\.RF\.(?:X_COM_RadioEnable|AdminCellState)$`)
	gsmCellRFStatusPattern    = regexp.MustCompile(`^Device\.Services\.GsmBTSCellDT\.(\d+)\.RfState$`)
	sasRFStatusPattern        = regexp.MustCompile(`^Device\.DeviceInfo\.SAS\.RadioEnable(\d*)$`)
	ruRFStatusPattern         = regexp.MustCompile(`^Device\.DeviceInfo\.(?:EU\.\d+\.)?RU\.(\d+)\.RFTxStatus$`)
	legacyRFStatusPattern     = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.CellConfig\.(?:LTE|NR)\.RAN\.RF\.RFTxStatus$`)
)

var rfStatusPathFamilies = []rfStatusPathFamily{
	// 产品模型会把 BaiBNQ 私有 rftxEnable 和 LTE 私有 RadioEnable
	// 归一为以下标准路径，优先读取小区级直接状态。
	{name: "nr_cell", pattern: nrCellRFStatusPattern, physicalIndex: regexRFPhysicalIndex(nrCellRFStatusPattern)},
	{name: "sas_cell", pattern: sasCellRFStatusPattern, physicalIndex: regexRFPhysicalIndex(sasCellRFStatusPattern)},
	{name: "lte_fap", pattern: lteFAPRFStatusPattern, physicalIndex: regexRFPhysicalIndex(lteFAPRFStatusPattern), requiresExpectedCount: true},
	{name: "ran_radio", pattern: ranRFStatusPattern, physicalIndex: regexRFPhysicalIndex(ranRFStatusPattern), requiresExpectedCount: true},
	{name: "gsm_cell", pattern: gsmCellRFStatusPattern, physicalIndex: regexRFPhysicalIndex(gsmCellRFStatusPattern)},
	{name: "sas_radio", pattern: sasRFStatusPattern, physicalIndex: sasRFPhysicalIndex, requiresExpectedCount: true},
	{name: "ru", pattern: ruRFStatusPattern, physicalIndex: regexRFPhysicalIndex(ruRFStatusPattern)},
	// 兼容历史已落库的非标准 RAN.RF 路径。
	{name: "legacy_ran_radio", pattern: legacyRFStatusPattern, physicalIndex: regexRFPhysicalIndex(legacyRFStatusPattern), requiresExpectedCount: true},
}

var rfStatusPathIndex = regexp.MustCompile(`\.(\d+)(?:\.|$)`)
var carrierModeSuffix = regexp.MustCompile(`(?i)/(SC|DC|CA)$`)

// CalcRFStatus computes the rf_status quick-query column from device_parameters.
// It preserves three distinct states: explicit on, explicit off/error, and unknown.
// Missing or unrecognized parameters must stay empty instead of being fabricated as off.
func CalcRFStatus(params map[string]string, tech model.Technology, productClass string) RFStatusProjection {
	if strings.EqualFold(strings.TrimSpace(productClass), "FAP/PGSM") {
		return RFStatusProjection{State: RFStatusUnsupported, Reason: "BSC has no device-level RF"}
	}

	expectedCount, countKnown, countReason, countInconsistent := resolveExpectedPhysicalCarrierCount(params, productClass)
	if countInconsistent {
		return RFStatusProjection{
			State:         RFStatusInconsistent,
			Reason:        countReason,
			ExpectedCount: expectedCount,
		}
	}

	observedAny := false
	lastReason := countReason
	for _, family := range rfStatusPathFamilies {
		paths := make([]string, 0)
		for path := range params {
			if family.pattern.MatchString(path) {
				paths = append(paths, path)
			}
		}
		if len(paths) == 0 {
			continue
		}
		observedAny = true

		if family.requiresExpectedCount && !countKnown {
			lastReason = fmt.Sprintf("%s RF paths require a reliable carrier count", family.name)
			continue
		}

		sort.SliceStable(paths, func(i, j int) bool {
			return lessRFStatusPath(paths[i], paths[j])
		})

		statusesByIndex := make(map[int]string, len(paths))
		conflict := false
		for _, path := range paths {
			index, indexed := family.physicalIndex(path)
			if !indexed || index < 1 {
				continue
			}
			if countKnown && index > expectedCount {
				// FAPService may contain logical/phantom instances outside the
				// configured physical carrier range. They do not enter the denominator.
				continue
			}
			status, normalized := normalizeRFStatusValue(params[path])
			if !normalized {
				continue
			}
			if previous, exists := statusesByIndex[index]; exists && previous != status {
				conflict = true
				lastReason = fmt.Sprintf("%s RF index %d has conflicting values", family.name, index)
				break
			}
			statusesByIndex[index] = status
		}
		if conflict {
			return RFStatusProjection{
				State:         RFStatusInconsistent,
				Reason:        lastReason,
				ExpectedCount: expectedCount,
			}
		}

		if countKnown {
			statuses := make([]string, 0, expectedCount)
			complete := true
			for index := 1; index <= expectedCount; index++ {
				status, exists := statusesByIndex[index]
				if !exists {
					complete = false
					lastReason = fmt.Sprintf("%s RF index %d is missing", family.name, index)
					break
				}
				statuses = append(statuses, status)
			}
			if complete {
				return RFStatusProjection{
					Status:        strings.Join(statuses, ","),
					State:         RFStatusValid,
					ExpectedCount: expectedCount,
				}
			}
			continue
		}

		indices := make([]int, 0, len(statusesByIndex))
		for index := range statusesByIndex {
			indices = append(indices, index)
		}
		sort.Ints(indices)
		if len(indices) > 0 {
			statuses := make([]string, 0, len(indices))
			for _, index := range indices {
				statuses = append(statuses, statusesByIndex[index])
			}
			return RFStatusProjection{Status: strings.Join(statuses, ","), State: RFStatusValid}
		}
	}

	if observedAny && countKnown {
		return RFStatusProjection{
			State:         RFStatusInconsistent,
			Reason:        lastReason,
			ExpectedCount: expectedCount,
		}
	}
	return RFStatusProjection{State: RFStatusUnknown, Reason: lastReason, ExpectedCount: expectedCount}
}

// CalcRFStatusFromDeviceParameters reconciles equivalent standard/private RF
// observations before projecting the list column. A newer private false must
// not be hidden by an older standard true. Equal-time contradictory aliases
// are reported as inconsistent instead of guessing.
func CalcRFStatusFromDeviceParameters(
	parameters []model.DeviceParameter,
	tech model.Technology,
	productClass string,
) RFStatusProjection {
	type aliasObservation struct {
		path       string
		rawValue   string
		status     string
		statusOK   bool
		observedAt time.Time
	}
	values := make(map[string]string, len(parameters))
	aliasGroups := make(map[string][]aliasObservation)
	for _, parameter := range parameters {
		path := strings.TrimSpace(parameter.ParameterPath)
		if path == "" {
			continue
		}
		values[path] = parameter.ParameterValue
		matches := lteFAPRFStatusPattern.FindStringSubmatch(path)
		if len(matches) != 2 {
			matches = ltePrivateRFStatusPattern.FindStringSubmatch(path)
		}
		if len(matches) != 2 {
			continue
		}
		canonicalPath := fmt.Sprintf(
			"Device.Services.FAPService.%s.FAPControl.LTE.RFTxStatus", matches[1],
		)
		status, statusOK := normalizeRFStatusValue(parameter.ParameterValue)
		aliasGroups[canonicalPath] = append(aliasGroups[canonicalPath], aliasObservation{
			path: path, rawValue: parameter.ParameterValue, status: status,
			statusOK: statusOK, observedAt: parameter.LastUpdatedAt,
		})
	}
	for canonicalPath, observations := range aliasGroups {
		if len(observations) < 2 {
			continue
		}
		var newestAt time.Time
		for index := range observations {
			observation := &observations[index]
			delete(values, observation.path)
			if observation.statusOK && observation.observedAt.After(newestAt) {
				newestAt = observation.observedAt
			}
		}
		var newest *aliasObservation
		for index := range observations {
			observation := &observations[index]
			if !observation.statusOK || !observation.observedAt.Equal(newestAt) {
				continue
			}
			if newest != nil && observation.status != newest.status {
				return RFStatusProjection{
					State: RFStatusInconsistent,
					Reason: fmt.Sprintf(
						"RF aliases for %s have contradictory values at %s",
						canonicalPath, observation.observedAt.UTC().Format(time.RFC3339Nano),
					),
				}
			}
			newest = observation
		}
		if newest != nil {
			values[canonicalPath] = newest.rawValue
		}
	}
	return CalcRFStatus(values, tech, productClass)
}

func regexRFPhysicalIndex(pattern *regexp.Regexp) func(string) (int, bool) {
	return func(path string) (int, bool) {
		matches := pattern.FindStringSubmatch(path)
		if len(matches) != 2 {
			return 0, false
		}
		index, err := strconv.Atoi(matches[1])
		return index, err == nil && index > 0
	}
}

func sasRFPhysicalIndex(path string) (int, bool) {
	matches := sasRFStatusPattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return 0, false
	}
	if matches[1] == "" || matches[1] == "1" {
		return 1, true
	}
	index, err := strconv.Atoi(matches[1])
	return index, err == nil && index > 0
}

func resolveExpectedPhysicalCarrierCount(params map[string]string, productClass string) (count int, known bool, reason string, inconsistent bool) {
	reportedCount, reported, reportedReason, reportedInvalid := strictReportedNumOfCells(params)
	if reportedInvalid {
		return 0, false, reportedReason, true
	}

	modeCount, fixedMode, carrierAggregation := productClassCarrierCount(productClass)
	if strings.EqualFold(strings.TrimSpace(productClass), "FAP/BTS") {
		modeCount, fixedMode = 1, true
	}

	if reported && fixedMode && reportedCount != modeCount {
		return reportedCount, true,
			fmt.Sprintf("reported NumOfCells %d conflicts with product carrier mode count %d", reportedCount, modeCount), true
	}
	if reported {
		return reportedCount, true, "", false
	}
	if fixedMode {
		return modeCount, true, "", false
	}
	if carrierAggregation {
		return 0, false, "CA product requires reported NumOfCells", false
	}
	return 0, false, "carrier count is unavailable", false
}

func strictReportedNumOfCells(params map[string]string) (count int, found bool, reason string, invalid bool) {
	paths := []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.CA.PARAMS.NumOfCells",
	}
	for _, path := range paths {
		raw := strings.TrimSpace(params[path])
		if raw == "" {
			continue
		}
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return 0, false, fmt.Sprintf("invalid NumOfCells at %s", path), true
		}
		if found && parsed != count {
			return count, true, "LTE and NR NumOfCells values conflict", true
		}
		count, found = parsed, true
	}
	return count, found, "", false
}

func productClassCarrierCount(productClass string) (count int, fixed bool, carrierAggregation bool) {
	matches := carrierModeSuffix.FindStringSubmatch(strings.TrimSpace(productClass))
	if len(matches) != 2 {
		return 0, false, false
	}
	switch strings.ToUpper(matches[1]) {
	case "SC":
		return 1, true, false
	case "DC":
		return 2, true, false
	case "CA":
		return 0, false, true
	default:
		return 0, false, false
	}
}

func normalizeRFStatusValue(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "3", "true", "on", "enabled":
		return "on", true
	case "0", "2", "false", "off", "disabled":
		return "off", true
	case "error", "abnormal", "failed", "fault":
		return "error", true
	default:
		return "", false
	}
}

func lessRFStatusPath(left, right string) bool {
	leftMatches := rfStatusPathIndex.FindAllStringSubmatch(left, -1)
	rightMatches := rfStatusPathIndex.FindAllStringSubmatch(right, -1)
	limit := len(leftMatches)
	if len(rightMatches) < limit {
		limit = len(rightMatches)
	}
	for i := 0; i < limit; i++ {
		leftIndex, _ := strconv.Atoi(leftMatches[i][1])
		rightIndex, _ := strconv.Atoi(rightMatches[i][1])
		if leftIndex != rightIndex {
			return leftIndex < rightIndex
		}
	}
	if len(leftMatches) != len(rightMatches) {
		return len(leftMatches) < len(rightMatches)
	}
	return left < right
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

var indexedUECountPathPattern = regexp.MustCompile(`^Device\.DeviceInfo\.(\d+)\.UE_Count$`)

// CalcUECount 从 device_parameters 汇总当前接入 UE 数。
//
// 标准路径按物理小区聚合：根路径表示小区 1，数字实例路径表示对应小区。
// 若根路径与 ".1" 同时存在，根路径优先，避免重复计数。仅当不存在任何合法
// 标准路径值时，才回退 LTE 厂商扩展路径。
//
// 返回值保证 >= 0；空值、非数字字符串或负数不参与聚合。
func CalcUECount(params map[string]string) int {
	const (
		rootPath     = "Device.DeviceInfo.UE_Count"
		fallbackPath = "Device.Services.FAPService.1.X_COM_ConnectedUECount"
	)

	countsByCell := make(map[int]int)
	if n, ok := parseNonNegativeInt(params[rootPath]); ok {
		countsByCell[1] = n
	}

	for path, value := range params {
		matches := indexedUECountPathPattern.FindStringSubmatch(path)
		if len(matches) != 2 {
			continue
		}
		cellIndex, err := strconv.Atoi(matches[1])
		if err != nil || cellIndex <= 0 {
			continue
		}
		if cellIndex == 1 {
			if _, rootExists := countsByCell[1]; rootExists {
				continue
			}
		}
		if n, ok := parseNonNegativeInt(value); ok {
			countsByCell[cellIndex] = n
		}
	}

	if len(countsByCell) > 0 {
		total := 0
		for _, count := range countsByCell {
			total += count
		}
		return total
	}
	if n, ok := parseNonNegativeInt(params[fallbackPath]); ok {
		return n
	}
	return 0
}

func parseNonNegativeInt(value string) (int, bool) {
	if strings.TrimSpace(value) == "" {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	return n, err == nil && n >= 0
}
