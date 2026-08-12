package device

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// runTimeTokenRegex matches TR069 run time tokens such as "40d", "4 hours",
// "58m" or "30秒". Some Baicells firmwares report vendor uptime as formatted
// text instead of the standard numeric Device.DeviceInfo.UpTime.
var runTimeTokenRegex = regexp.MustCompile(`(?i)(\d+)\s*(days?|d|天|hours?|hrs?|hr|h|小时|minutes?|mins?|min|m|分钟|seconds?|secs?|sec|s|秒)`)
var runTimeColonRegex = regexp.MustCompile(`^(\d+):(\d{1,2})(?::(\d{1,2}))?$`)

var lteBandwidthEnumMHz = map[string]float64{
	"n6":   1.4,
	"n15":  3,
	"n25":  5,
	"n50":  10,
	"n75":  15,
	"n100": 20,
}

// parseRunTimeToSeconds parses TR069 run time format to seconds.
// Supported formats: "40d 4h 58m", "2 days 5 hours", "58m",
// "1天2小时3分钟4秒", "49:03:04" and plain numeric seconds.
// Returns 0 if parsing fails.
func parseRunTimeToSeconds(val string) int64 {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return 0
	}

	if seconds, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return seconds
	}

	if matches := runTimeColonRegex.FindStringSubmatch(trimmed); matches != nil {
		first, _ := strconv.ParseInt(matches[1], 10, 64)
		second, _ := strconv.ParseInt(matches[2], 10, 64)
		if matches[3] == "" {
			return first*3600 + second*60
		}
		third, _ := strconv.ParseInt(matches[3], 10, 64)
		return first*3600 + second*60 + third
	}

	var totalSeconds int64
	matches := runTimeTokenRegex.FindAllStringSubmatch(trimmed, -1)
	for _, match := range matches {
		n, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(match[2]) {
		case "d", "day", "days", "天":
			totalSeconds += n * 86400
		case "h", "hr", "hrs", "hour", "hours", "小时":
			totalSeconds += n * 3600
		case "m", "min", "mins", "minute", "minutes", "分钟":
			totalSeconds += n * 60
		case "s", "sec", "secs", "second", "seconds", "秒":
			totalSeconds += n
		}
	}

	return totalSeconds
}

func parseBandwidthMHz(val string) (float64, bool) {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return 0, false
	}
	if mhz, ok := lteBandwidthEnumMHz[strings.ToLower(trimmed)]; ok {
		return mhz, true
	}
	trimmed = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(trimmed), "mhz"), "m")
	value, err := strconv.ParseFloat(strings.TrimSpace(trimmed), 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// instanceAggregationRule 描述「同一逻辑列对应多个实例化 TR069 路径」的聚合规则。
// 模板中以 `{x}` 形式表示通配数字索引（如 `{f}` `{c}` `{t}`），实际占位字符无关，
// 仅用于阅读;运行期把所有 `\{[a-zA-Z]\}` 替换为 `(\d+)` 编译成正则匹配。
//
// 聚合语义（设计 #364-followup "多实例小区级聚合"）:
//   - 命中模板的所有 paramValues 路径 → 提取数字索引 → 数值升序排序 → 值去重 →
//     用 `,` 拼接覆盖 fields[column]。
//   - 同列允许有多模板(NR/LTE 双套兜底)，按列分组汇总;实际只一套有数据时不混淆。
//   - 在 SyncFromParameters 末尾、carrier mapping + universalInformMapping 之后执行，
//     确定性覆盖单实例写入。无命中则保留单值不动。
//   - 仅适用于 varchar 列；数值列(bandwidth/transmit_power/gps_satellites)若需多
//     实例显示需先单独 migration 改字段类型，此处不参与。
type instanceAggregationRule struct {
	column   string
	template string
	pattern  *regexp.Regexp
}

// universalInformInstanceMappings 列出小区级/多实例字段的 TR069 路径模板，按列分组。
// detail_assembler.go 渲染详情页时是逐 cell 遍历，list 页则统一聚合显示。
//
// 编辑时若新增模板，须确保通配符在 TR069 实际上报路径中确实是数字索引（如
// FAPService.{i} / CellConfig.{j} / NR.CN.TA.{k} / FAP.Ipsec.{i}）。
var universalInformInstanceMappings = []struct {
	column    string
	templates []string
}{
	// PCI（小区级，NR + LTE 兜底）
	{column: "pci", templates: []string{
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.RF.PhyCellID",
		"Device.Services.FAPService.{f}.CellConfig.LTE.RAN.RF.PhyCellID",
	}},
	// 频点 EARFCN / NRARFCNDL（小区级）
	{column: "freq_point", templates: []string{
		"Device.Services.GsmBTSCellDT.{g}.CurrentArfcn",
		"DeviceGSM.Bts.{b}.Trx.{t}.Arfcn",
		"Device.Services.FAPService.{f}.CellConfig.LTE.RAN.RF.EARFCNDL",
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.RF.NRARFCNDL",
		"Device.Services.FAPService.{f}.CellConfig.LTE.RAN.Common.EARFCNDL",
	}},
	// 上行 EARFCN / NRARFCNUL（小区级）
	{column: "ul_earfcn", templates: []string{
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.RF.NRARFCNUL",
		"Device.Services.FAPService.{f}.CellConfig.LTE.RAN.RF.EARFCNUL",
	}},
	// Band（小区级）
	{column: "band", templates: []string{
		"Device.Services.GsmBTSCellDT.{g}.GsmBtsBand",
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.{b}.FreqBandIndicatorNR",
		"Device.Services.FAPService.{f}.CellConfig.LTE.RAN.RF.FreqBandIndicator",
	}},
	// TAC（小区级，NR 走 CN.TA 子树，LTE 走 EPC.TAC）
	{column: "tac", templates: []string{
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.CN.TA.{t}.TAC",
		"Device.Services.FAPService.{f}.CellConfig.LTE.EPC.TAC",
	}},
	// LAC（GSM 小区级）
	{column: "lac", templates: []string{
		"Device.Services.GsmBTSCellDT.{g}.CurrLocAreaCode",
	}},
	// Cell ID / NR Cell Identity（小区级）
	{column: "cell_id", templates: []string{
		"Device.Services.GsmBTSCellDT.{g}.GsmCellID",
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.CN.TA.{t}.NrcellIdentity",
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.Common.CellLocalId",
		"Device.Services.FAPService.{f}.CellConfig.LTE.RAN.Common.CellIdentity",
	}},
	// NR Admin State（小区级 CellEnable.AdminState；与详情页小区表同源）
	{column: "admin_state", templates: []string{
		"Device.Services.FAPService.{f}.CellConfig.{c}.NR.RAN.CellEnable.AdminState",
	}},
	// LTE FAP AdminState（设备/FAPService 级 boolean，多 FAPService 实例时聚合）
	{column: "lock_status", templates: []string{
		"Device.Services.FAPService.{f}.FAPControl.LTE.AdminState",
	}},
	// IPSec 网关地址（多实例 IPSec 隧道，与详情页 IPSec 参数表同源）
	{column: "ipsec_addr", templates: []string{
		"Device.FAP.Ipsec.{i}.TUNNEL_GATEWAY",
	}},
}

// compiledInstanceAggregationRules 预编译形态（package init 期间生成）。
var compiledInstanceAggregationRules []instanceAggregationRule

// instancePlaceholderRegex 匹配模板字符串中的 `{x}` 通配符（QuoteMeta 后变成
// `\{x\}`），运行时替换为 `(\d+)` 形成捕获组。
var instancePlaceholderRegex = regexp.MustCompile(`\\\{[a-zA-Z_]\\\}`)

var defaultDeviceInfoVarcharLimits = map[string]int{
	"admin_state": 16,
	"band":        16,
	"cell_id":     64,
	"freq_point":  32,
	"ipsec_addr":  64,
	"lac":         16,
	"lock_status": 16,
	"pci":         64,
	"plmn":        40,
	"rf_status":   64,
	"tac":         16,
	"ul_earfcn":   32,
}

type deviceInfoStringLimitReader interface {
	GetStringFieldLimits(ctx context.Context) (map[string]int, error)
}

func init() {
	for _, m := range universalInformInstanceMappings {
		for _, tmpl := range m.templates {
			pattern := "^" + instancePlaceholderRegex.ReplaceAllString(regexp.QuoteMeta(tmpl), `(\d+)`) + "$"
			compiledInstanceAggregationRules = append(compiledInstanceAggregationRules, instanceAggregationRule{
				column:   m.column,
				template: tmpl,
				pattern:  regexp.MustCompile(pattern),
			})
		}
	}
}

// aggregateInstanceFields scans paramValues for every templated path and
// overrides fields[column] with the comma-joined values of all matched
// instances (sorted by numeric instance indices, duplicates removed).
//
// 若某列所有模板都无命中，则 fields[column] 维持上游 carrier/universal mapping
// 写入的单值（或空），不覆盖。这保证：单 cell / 单实例设备聚合后等价单值；多
// cell / 多 IPSec 隧道设备前端直接看到 "1,2,3" 这样的 csv 串。
func aggregateInstanceFields(paramValues map[string]string, fields map[string]interface{}) {
	type instanceValue struct {
		indices []int64
		value   string
	}
	byColumn := make(map[string][]instanceValue)

	for _, rule := range compiledInstanceAggregationRules {
		for path, val := range paramValues {
			if val == "" {
				continue
			}
			matches := rule.pattern.FindStringSubmatch(path)
			if matches == nil {
				continue
			}
			indices := make([]int64, 0, len(matches)-1)
			for _, s := range matches[1:] {
				n, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					indices = nil
					break
				}
				indices = append(indices, n)
			}
			if indices == nil {
				continue
			}
			byColumn[rule.column] = append(byColumn[rule.column], instanceValue{indices: indices, value: val})
		}
	}

	for col, list := range byColumn {
		if len(list) == 0 {
			continue
		}
		sort.SliceStable(list, func(i, j int) bool {
			a, b := list[i].indices, list[j].indices
			n := len(a)
			if len(b) < n {
				n = len(b)
			}
			for k := 0; k < n; k++ {
				if a[k] != b[k] {
					return a[k] < b[k]
				}
			}
			return len(a) < len(b)
		})
		seen := make(map[string]struct{}, len(list))
		vals := make([]string, 0, len(list))
		for _, v := range list {
			if _, ok := seen[v.value]; ok {
				continue
			}
			seen[v.value] = struct{}{}
			vals = append(vals, v.value)
		}
		if len(vals) == 0 {
			continue
		}
		fields[col] = strings.Join(vals, ",")
	}
}

// projectValidCellIDs 使用详情页相同的小区组装口径，投影当前有效的小区标识。
// 列表的 cell_id 是设备级快照，不能直接保留所有历史/无效实例的聚合值。
func projectValidCellIDs(deviceID uuid.UUID, paramValues map[string]string, tech model.Technology, productClass string) string {
	params := make([]model.DeviceParameter, 0, len(paramValues))
	for path, value := range paramValues {
		params = append(params, model.DeviceParameter{
			DeviceID:       deviceID,
			ParameterPath:  path,
			ParameterValue: value,
		})
	}

	var cells []CellInfo
	if strings.EqualFold(string(tech), "gsm") {
		cells = AssembleGSMCells(params)
	} else {
		cells = AssembleCells(params, CalcNumOfCells(paramValues), productClass)
		configuredCellCountRaw := strings.TrimSpace(paramValues["Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells"])
		if configuredCellCountRaw == "" {
			configuredCellCountRaw = strings.TrimSpace(paramValues["Device.Services.FAPService.1.CellConfig.NR.RAN.CA.PARAMS.NumOfCells"])
		}
		// NumOfCells 未上报时，不能使用 CalcNumOfCells 的默认值 1；
		// BM 等站型通过实际 FAPService 实例数量表达有效小区数。
		if configuredCellCountRaw != "" {
			configuredCellCount := CalcNumOfCells(paramValues)
			filtered := cells[:0]
			for _, cell := range cells {
				if cell.Index <= configuredCellCount {
					filtered = append(filtered, cell)
				}
			}
			cells = filtered
		}
	}

	ids := make([]string, 0, len(cells))
	seen := make(map[string]struct{}, len(cells))
	for _, cell := range cells {
		id := strings.TrimSpace(cell.CellID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return strings.Join(ids, ",")
}

func fitCSVWithinLimit(value string, maxLen int) string {
	if utf8.RuneCountInString(value) <= maxLen {
		return value
	}

	parts := strings.Split(value, ",")
	kept := make([]string, 0, len(parts))
	currentLen := 0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		nextLen := currentLen + utf8.RuneCountInString(part)
		if len(kept) > 0 {
			nextLen++
		}
		if nextLen > maxLen {
			break
		}
		kept = append(kept, part)
		currentLen = nextLen
	}
	return strings.Join(kept, ",")
}

func enforceDeviceInfoFieldSizeLimits(fields map[string]interface{}) {
	enforceDeviceInfoFieldSizeLimitsWithSchema(fields, defaultDeviceInfoVarcharLimits, nil, uuid.Nil)
}

func enforceDeviceInfoFieldSizeLimitsWithSchema(fields map[string]interface{}, limits map[string]int, logger *zap.Logger, deviceID uuid.UUID) {
	for column, maxLen := range limits {
		if maxLen <= 0 {
			continue
		}
		value, ok := fields[column].(string)
		if !ok || value == "" || utf8.RuneCountInString(value) <= maxLen {
			continue
		}

		if strings.Contains(value, ",") {
			trimmed := fitCSVWithinLimit(value, maxLen)
			if trimmed != "" {
				fields[column] = trimmed
				if logger != nil {
					logger.Warn("trimmed overlong device_info CSV field before update",
						zap.String("device_id", deviceID.String()),
						zap.String("column", column),
						zap.Int("max_chars", maxLen),
						zap.String("value", value),
						zap.String("trimmed", trimmed))
				}
				continue
			}
		}

		if logger != nil {
			logger.Warn("dropped overlong device_info field before update",
				zap.String("device_id", deviceID.String()),
				zap.String("column", column),
				zap.Int("max_chars", maxLen),
				zap.String("value", value))
		}
		delete(fields, column)
	}
}

func copyStringIntMap(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergeStringIntMaps(base, override map[string]int) map[string]int {
	out := copyStringIntMap(base)
	for k, v := range override {
		if v > 0 {
			out[k] = v
		}
	}
	return out
}

func (s *InfoSyncer) deviceInfoStringLimits(ctx context.Context) map[string]int {
	s.stringLimitsMu.Lock()
	defer s.stringLimitsMu.Unlock()
	if s.stringLimits != nil {
		return s.stringLimits
	}

	limits := defaultDeviceInfoVarcharLimits
	if reader, ok := s.infoRepo.(deviceInfoStringLimitReader); ok {
		dbLimits, err := reader.GetStringFieldLimits(ctx)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("load device_info string field limits failed; using defaults", zap.Error(err))
			}
		} else if len(dbLimits) > 0 {
			limits = mergeStringIntMaps(defaultDeviceInfoVarcharLimits, dbLimits)
		}
	}

	s.stringLimits = copyStringIntMap(limits)
	return s.stringLimits
}

func (s *InfoSyncer) enforceDeviceInfoFieldSizeLimits(ctx context.Context, deviceID uuid.UUID, fields map[string]interface{}) {
	if len(fields) == 0 {
		return
	}
	enforceDeviceInfoFieldSizeLimitsWithSchema(fields, s.deviceInfoStringLimits(ctx), s.logger, deviceID)
}

// universalInformMapping maps TR069 parameter paths to device_info columns
// for parameters that are identical across all carriers (not carrier-specific).
// Note: run_time is handled separately with priority logic (UpTime > X_COM_STATION_RUN_Time)
//
// Phase 3 (设计文档 §4.2 Layer B)：扩充 TR-181 标准 path 映射，覆盖
// device_info 表新增的 9 个字段。这些 path 与运营商无关（CMCC/CTCC/CUCC
// CPE 上报的字段名相同），统一在此 mapping 表，避免三个 adapter 冗余。
// carrier-specific 路径仍保留在各自 adapter.GetInfoParamMapping 中（如
// X_CMCC_MACAddress 兜底）— InfoSyncer 用 universal 后写覆盖 carrier
// 的语义：标准 path 优先（若 CPE 同时上报两种，标准胜出）。
//
// 小区级字段（pci/tac/band/freq_point/ul_earfcn/cell_id/admin_state/lock_status）
// 与多实例字段（ipsec_addr）的实例化路径不在此表，统一走 universalInformInstanceMappings
// 聚合（见 aggregateInstanceFields），单实例设备聚合后等价此处保留的单值。
var universalInformMapping = map[string]string{
	"Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus": "alarm_severity",
	"Device.DeviceInfo.FAP_adminstate":                                  "admin_state",
	// TR-181 Ethernet 标准 path（取代 CMCC X_CMCC_MACAddress，多数 CPE 上报此 path）
	"Device.Ethernet.Interface.MACAddress": "mac",
	// #362: 发射功率不再走 universal 的 Capabilities.MaxTxPower。MaxTxPower 是
	// 硬件最大能力上限(READ_ONLY)，而前端"发射功率"列与 LMT 口径是参考信号功率
	// (ReferenceSignalPower，RW，小区实际工作功率)。原映射在此 universal 循环里
	// 串行覆盖了 cmcc/ctcc adapter 的 ReferenceSignalPower→transmit_power（universal
	// 循环在 carrier 循环之后执行，确定性覆盖），导致列表/详情恒显能力上限值、与 LMT
	// 不一致。删除此条后 transmit_power 的唯一权威来源 = carrier adapter 的
	// ReferenceSignalPower→transmit_power（见 cmcc/ctcc adapter.go GetInfoParamMapping）。
	// LTE 设备特有 PHY 参数（不参与小区聚合 — 这些列前端按单值显示且只 cell-1 有意义）
	// GSM 位置区码：补全 device_groups LAC 匹配模式所需的设备侧数据源（T-2026-05-25）。
	// 与 TAC 平行，CPE 同时上报时 LAC 多见于双模 / GSM 设备。
	// 兼容两种 CPE 命名：BTS.* 是 Baicells BaiBS_AGS 旧固件；GSM.* 是 sNBS1200/9200 实测路径。
	"Device.DeviceInfo.BTS.CurrentLac":   "lac",
	"Device.DeviceInfo.GSM.CurrentLac":   "lac",
	"Device.DeviceInfo.BTS.CurrentArfcn": "freq_point",
	"Device.DeviceInfo.GSM.CurrentArfcn": "freq_point",
	// migration 000003：GSM/BTS 专属（DeviceGSM.* TR069 路径同步）
	"DeviceGSM.BscSelect":      "bsc_select",
	"DeviceGSM.OmlRemoteIp":    "oml_remote_ip",
	"DeviceGSM.OmlRemoteIpBak": "oml_remote_ip_bak",
	"DeviceGSM.IpaUnitId":      "ipa_unit_id",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment":      "subframe_assignment",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns": "special_subframe",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig":  "root_index",
	// GPS（卫星数 + 高度）
	"Device.FAP.GPS.NumberOfSatellites": "gps_satellites",
	// 小区级 / 多实例字段（PCI/TAC/Band/EARFCN/Cell ID/AdminState/IPSec 等）已搬到
	// universalInformInstanceMappings 走聚合管线，详情见 aggregateInstanceFields。
}

// gpsHeightCandidatePaths lists possible TR069 paths for GPS height.
// CPE 上报的字段名有拼写差异（`altidute` 是 Baicells 固件字面值），按顺序
// 优先匹配第一个有值的 path。Phase 3 fallback 链；后续固件统一后可裁剪。
var gpsHeightCandidatePaths = []string{
	"Device.FAP.GPS.altidute", // Baicells BaiBLQ 实测拼写
	"Device.FAP.GPS.Altitude", // TR-181 spec 标准拼写
	"Device.FAP.GPS.LockedAltitude",
	"Device.FAP.Synchronization.Altitude",
	"Device.FAP.GPS.Height",
}

var gpsSatelliteCountCandidatePaths = []string{
	"Device.FAP.GPS.NumberOfSatellites",
	"Device.DeviceInfo.SAS.FAP.GPS.NumberOfSatellites",
}

const (
	gpsLockedLatitudePath  = "Device.FAP.GPS.LockedLatitude"
	gpsLockedLongitudePath = "Device.FAP.GPS.LockedLongitude"
	gpsCoordinateScale     = 1_000_000
)

var ethernetInterfaceMACPath = regexp.MustCompile(`^Device\.Ethernet\.Interface\.(\d+)\.MACAddress$`)

// deriveEnbID 从 LTE ECI 派生 eNodeB ID。
//
// TR-36.413 / 3GPP 标准：28-bit ECI = 20-bit eNB-ID + 8-bit Cell-ID。
// 即 enb_id = eci >> 8。
//
// 返回 (派生值字符串, ok)。ok=false 表示输入无效（空 / 非数字 / 0）。
func deriveEnbID(eci string) (string, bool) {
	if eci == "" {
		return "", false
	}
	eciInt, err := strconv.ParseInt(eci, 10, 64)
	if err != nil || eciInt <= 0 {
		return "", false
	}
	return strconv.FormatInt(eciInt>>8, 10), true
}

func lookupGPSSatelliteCount(paramValues map[string]string) *int {
	for _, path := range gpsSatelliteCountCandidatePaths {
		raw := strings.TrimSpace(paramValues[path])
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			continue
		}
		return &value
	}
	return nil
}

// deriveNetworkModel 根据 PHY 子帧 path 是否存在判定 TDD / FDD。
//
// LTE 帧结构由 SubFrameAssignment / SpecialSubframePatterns 标识 TDD；
// FDDFrame 子树标识 FDD。两者互斥。
//
// 返回 (模式字符串, ok)。ok=false 表示无法判定（参数缺失）。
func deriveNetworkModel(paramValues map[string]string) (string, bool) {
	if _, ok := paramValues["Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment"]; ok {
		return "TDD", true
	}
	if _, ok := paramValues["Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.FDDFrame.SubFrameAssignment"]; ok {
		return "FDD", true
	}
	return "", false
}

// lookupGPSHeight 按候选 path 优先级查找 GPS 高度值。
//
// 命中第一个非空值即返回；全部未命中返回 ("", false)。
func lookupGPSHeight(paramValues map[string]string) (string, bool) {
	for _, p := range gpsHeightCandidatePaths {
		if v, ok := paramValues[p]; ok && v != "" {
			return v, true
		}
	}
	return "", false
}

func lookupGPSCoordinates(paramValues map[string]string) (float64, float64, bool) {
	latitude, ok := parseGPSCoordinate(paramValues[gpsLockedLatitudePath], 90)
	if !ok {
		return 0, 0, false
	}

	longitude, ok := parseGPSCoordinate(paramValues[gpsLockedLongitudePath], 180)
	if !ok {
		return 0, 0, false
	}

	return latitude, longitude, true
}

func parseGPSCoordinate(raw string, maxAbs float64) (float64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}

	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, false
	}

	absValue := math.Abs(value)
	if absValue <= maxAbs {
		return value, true
	}
	if absValue <= maxAbs*gpsCoordinateScale {
		return value / gpsCoordinateScale, true
	}

	return 0, false
}

type ethernetInterfaceCandidate struct {
	index string
	mac   string
	score int
}

func lookupDeviceInfoMAC(paramValues map[string]string) (string, bool) {
	if v, ok := paramValues["Device.DeviceInfo.X_COM_MACAddress"]; ok && v != "" {
		return v, true
	}

	return "", false
}

func lookupWANMAC(paramValues map[string]string) (string, bool) {
	if v, ok := paramValues["Device.Ethernet.Interface.MACAddress"]; ok && v != "" {
		return v, true
	}

	candidates := make([]ethernetInterfaceCandidate, 0)
	for path, mac := range paramValues {
		if mac == "" {
			continue
		}
		matches := ethernetInterfaceMACPath.FindStringSubmatch(path)
		if matches == nil {
			continue
		}

		index := matches[1]
		candidates = append(candidates, ethernetInterfaceCandidate{
			index: index,
			mac:   mac,
			score: scoreEthernetInterface(index, paramValues),
		})
	}

	if len(candidates) == 0 {
		return "", false
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		left, errLeft := strconv.Atoi(candidates[i].index)
		right, errRight := strconv.Atoi(candidates[j].index)
		switch {
		case errLeft == nil && errRight == nil:
			return left < right
		case errLeft == nil:
			return true
		case errRight == nil:
			return false
		default:
			return candidates[i].index < candidates[j].index
		}
	})

	return candidates[0].mac, true
}

func lookupTransmitPower(paramValues map[string]string, mappedValue string) (string, bool) {
	for _, suffix := range []string{
		".BtsRfPower",
		".GsmBtsRFPower",
		".X_COM_MaxTxPowerExpanded",
		".PowerModify",
		".ReferenceSignalPower",
	} {
		paths := make([]string, 0, len(paramValues))
		for path, value := range paramValues {
			if value == "" || !strings.HasSuffix(path, suffix) {
				continue
			}
			paths = append(paths, path)
		}
		if len(paths) == 0 {
			continue
		}
		sort.Strings(paths)
		return strings.TrimSpace(paramValues[paths[0]]), true
	}

	if mappedValue != "" {
		return mappedValue, true
	}

	return "", false
}

func scoreEthernetInterface(index string, paramValues map[string]string) int {
	score := 0
	prefix := "Device.Ethernet.Interface." + index + "."

	for _, suffix := range []string{"Name", "UserLabel", "PortLocation"} {
		if containsWANKeyword(paramValues[prefix+suffix]) {
			score += 20
		}
	}

	for path, value := range paramValues {
		if value == "" || !strings.HasPrefix(path, prefix) {
			continue
		}
		if strings.HasSuffix(path, ".PortType") && containsWANKeyword(value) {
			score += 100
			continue
		}
		if strings.HasSuffix(path, ".IPAddress") || strings.HasSuffix(path, ".DefaultGateway") {
			score += 5
		}
	}

	return score
}

func containsWANKeyword(val string) bool {
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

// TR069 parameter paths for run_time with priority
const (
	// ParamUpTime is the standard TR069 UpTime parameter (unit: seconds, direct storage)
	ParamUpTime = "Device.DeviceInfo.UpTime"
	// ParamStationRunTime is the vendor-specific run time parameter (format: "40d 4h 58m")
	ParamStationRunTime = "Device.DeviceInfo.X_COM_STATION_RUN_Time"
)

// existPlmnIdListPath matches FAPControl Gateway ExistPlmnidList paths across
// FAPService instances. Used as a PLMN fallback when carrier mapping
// (EPC.PLMNList) is empty — common on GSM / 旧固件 设备。
var existPlmnIdListPath = regexp.MustCompile(`^Device\.Services\.FAPService\.(\d+)\.FAPControl\.LTE\.Gateway\.ExistPlmnidList$`)

// lookupExistPlmnIdList scans for FAPControl ExistPlmnidList values and
// returns the first non-empty entry (FAPService 索引最小者优先)。值形如
// "314030" 或 "314030,460000"；多 PLMN 时取原值不再切分（与 EPC.PLMNList 单值映射对齐）。
func lookupExistPlmnIdList(paramValues map[string]string) string {
	bestIdx := -1
	best := ""
	for path, val := range paramValues {
		if val == "" {
			continue
		}
		matches := existPlmnIdListPath.FindStringSubmatch(path)
		if matches == nil {
			continue
		}
		idx, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if bestIdx == -1 || idx < bestIdx {
			bestIdx = idx
			best = val
		}
	}
	return best
}

// InfoSyncer extracts key TR069 parameters from device_parameters
// and updates the corresponding device_info columns for fast query access.
type DeviceCoordinateWriter interface {
	UpdateCoordinates(ctx context.Context, id uuid.UUID, latitude, longitude float64) error
}

type DeviceCoordinateReader interface {
	GetCoordinates(ctx context.Context, id uuid.UUID) (*Location, error)
}

type InfoSyncer struct {
	infoRepo                DeviceInfoRepository
	paramRepo               DeviceParameterRepository
	coordinateWriter        DeviceCoordinateWriter
	locationObservationRepo LocationObservationRepository
	carrierRegistry         *carrier.CarrierRegistry
	logger                  *zap.Logger
	stringLimitsMu          sync.Mutex
	stringLimits            map[string]int
}

// NewInfoSyncer creates a new InfoSyncer.
func NewInfoSyncer(
	infoRepo DeviceInfoRepository,
	paramRepo DeviceParameterRepository,
	coordinateWriter DeviceCoordinateWriter,
	carrierRegistry *carrier.CarrierRegistry,
	logger *zap.Logger,
	locationObservationRepos ...LocationObservationRepository,
) *InfoSyncer {
	var locationObservationRepo LocationObservationRepository
	if len(locationObservationRepos) > 0 {
		locationObservationRepo = locationObservationRepos[0]
	}
	return &InfoSyncer{
		infoRepo:                infoRepo,
		paramRepo:               paramRepo,
		coordinateWriter:        coordinateWriter,
		locationObservationRepo: locationObservationRepo,
		carrierRegistry:         carrierRegistry,
		logger:                  logger,
	}
}

// topologyAttributeColumns 列出会触发 device_groups 重匹配的 device_info 列。
// 当 SyncFromParameters 检测到这些列的实际值变化时，会把列名追加到返回的
// changedAttrs 切片中，由调用方决定是否 publish device.attributes.changed。
//
// 当前只覆盖 LAC / TAC（GroupMatchEngine 已支持的两种位置区匹配模式）。
// 后续若 device_groups 新增按其它属性匹配（如 PLMN / Band），把列名加进来即可。
var topologyAttributeColumns = []string{"lac", "tac"}

// SyncFromParameters reads the device's stored TR069 parameters and updates
// the corresponding device_info columns based on the carrier's mapping.
//
// 返回 changedAttrs：本次 sync 中 topologyAttributeColumns 列出的列**实际从旧值
// 变成了不同的新值**（NULL→有值 / 有值→不同新值 / 有值→NULL 全算变化）的列名
// 列表。调用方据此决定是否发 device.attributes.changed 事件触发分组重匹配。
// 当 sync 不涉及这些列、或值未变时，返回 nil（避免事件风暴）。
func (s *InfoSyncer) SyncFromParameters(ctx context.Context, deviceID uuid.UUID, carrierCode model.CarrierCode, tech model.Technology, productClass string) ([]string, error) {
	c, err := s.carrierRegistry.Get(carrierCode)
	if err != nil {
		return nil, fmt.Errorf("get carrier adapter: %w", err)
	}

	mapping := c.GetInfoParamMapping(tech)

	// Get all parameters for this device
	params, err := s.paramRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device parameters: %w", err)
	}

	// Build a lookup map: path → value
	paramValues := make(map[string]string, len(params))
	for _, p := range params {
		paramValues[p.ParameterPath] = p.ParameterValue
	}

	// Extract values that exist in both the device's parameters and the mapping
	fields := make(map[string]interface{})
	for paramPath, infoColumn := range mapping {
		if val, ok := paramValues[paramPath]; ok && val != "" {
			fields[infoColumn] = val
		}
	}

	// Universal Inform mapping (carrier-agnostic direct fields)
	for paramPath, infoColumn := range universalInformMapping {
		if val, ok := paramValues[paramPath]; ok && val != "" {
			fields[infoColumn] = val
		}
	}
	if raw, ok := fields["bandwidth"].(string); ok {
		if mhz, parsed := parseBandwidthMHz(raw); parsed {
			fields["bandwidth"] = mhz
		} else {
			delete(fields, "bandwidth")
		}
	}
	if _, exists := fields["bandwidth"]; !exists {
		if raw, ok := paramValues["Device.DeviceInfo.SAS.PreferredBandwidth"]; ok {
			if mhz, parsed := parseBandwidthMHz(raw); parsed {
				fields["bandwidth"] = mhz
			}
		}
	}
	if _, exists := fields["mac"]; !exists {
		if mac, ok := lookupDeviceInfoMAC(paramValues); ok {
			fields["mac"] = mac
		} else if mac, ok := lookupWANMAC(paramValues); ok {
			fields["mac"] = mac
		}
	}
	mappedTransmitPower, _ := fields["transmit_power"].(string)
	if txPower, ok := lookupTransmitPower(paramValues, mappedTransmitPower); ok {
		fields["transmit_power"] = txPower
	} else {
		delete(fields, "transmit_power")
	}

	// PLMN fallback：carrier mapping 一般用 EPC.PLMNList.{n}.PLMNID 路径，
	// 但部分 GSM / 旧固件设备只上报 FAPControl.LTE.Gateway.ExistPlmnidList
	// （实测 sNBS1200 sNBS9200）。仅在前述映射未命中时兜底取首个非空值，
	// 避免覆盖 carrier-specific 优先级。
	if _, exists := fields["plmn"]; !exists {
		if plmn := lookupExistPlmnIdList(paramValues); plmn != "" {
			fields["plmn"] = plmn
		}
	}

	// Handle run_time with priority: UpTime (seconds) > X_COM_STATION_RUN_Time (parsed format)
	// Priority 1: Device.DeviceInfo.UpTime (standard TR069, unit is seconds)
	if val, ok := paramValues[ParamUpTime]; ok && val != "" {
		if seconds, err := strconv.ParseInt(val, 10, 64); err == nil {
			fields["run_time"] = seconds
		}
	}
	// Priority 2: Device.DeviceInfo.X_COM_STATION_RUN_Time (vendor-specific, format "40d 4h 58m")
	// Only use if UpTime is not available or invalid
	if _, exists := fields["run_time"]; !exists {
		if val, ok := paramValues[ParamStationRunTime]; ok && val != "" {
			if seconds := parseRunTimeToSeconds(val); seconds > 0 {
				fields["run_time"] = seconds
			}
		}
	}

	// Computed quick-query columns from multiple parameters
	fields["cell_status"] = CalcCellStatus(paramValues)
	fields["op_state"] = CalcOpState(paramValues)
	fields["mme_status"] = CalcCoreNetworkStatus(paramValues, tech)
	fields["sync_status"] = CalcSyncStatus(paramValues)
	rfProjection := CalcRFStatusFromDeviceParameters(params, tech, productClass)
	fields["rf_status"] = rfProjection.Status
	if rfProjection.State == RFStatusInconsistent {
		s.logger.Warn("RF status projection is inconsistent",
			zap.String("device_id", deviceID.String()),
			zap.String("product_class", productClass),
			zap.Int("expected_count", rfProjection.ExpectedCount),
			zap.String("reason", rfProjection.Reason))
	}
	fields["gps_status"] = CalcGPSStatus(paramValues)
	fields["num_of_cells"] = CalcNumOfCells(paramValues)
	fields["license_status"] = CalcLicenseStatus(paramValues)
	fields["ue_count"] = CalcUECount(paramValues)

	// Phase 3 派生字段（设计文档 §4.2 Layer C）：
	var reportedGPSHeight *float64
	if v, ok := lookupGPSHeight(paramValues); ok {
		fields["gps_height"] = v
		if height, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && !math.IsNaN(height) && !math.IsInf(height, 0) {
			reportedGPSHeight = &height
		}
	}
	if eci, ok := fields["eci"].(string); ok {
		if enb, ok := deriveEnbID(eci); ok {
			fields["enb_id"] = enb
		}
	}
	if model, ok := deriveNetworkModel(paramValues); ok {
		fields["network_model"] = model
	}

	// 多实例聚合（#364-followup）：覆盖小区级 / 多实例字段为
	// "v1,v2,..." 形式，与详情页逐 cell 表格逻辑保持一致。
	// 必须在所有 carrier mapping + universalInformMapping 写入之后执行,
	// 单实例设备聚合结果与之前单值等价,多实例设备 list 直接显示 csv 串。
	aggregateInstanceFields(paramValues, fields)
	if validCellIDs := projectValidCellIDs(deviceID, paramValues, tech, productClass); validCellIDs != "" {
		fields["cell_id"] = validCellIDs
	}
	frequencyProjection := projectRadioFrequencyFields(paramValues, tech, productClass)
	if frequencyProjection.dlObserved {
		fields["freq_point"] = frequencyProjection.dlValue
	}
	if frequencyProjection.ulObserved {
		fields["ul_earfcn"] = frequencyProjection.ulValue
	}
	if !frequencyProjection.complete {
		s.logger.Warn("radio frequency projection is incomplete",
			zap.String("device_id", deviceID.String()),
			zap.String("product_class", productClass),
			zap.String("technology", string(tech)),
			zap.String("reason", frequencyProjection.reason))
	}
	s.enforceDeviceInfoFieldSizeLimits(ctx, deviceID, fields)

	latitude, longitude, sourcePath, hasCoordinates := LookupGPSCoordinates(paramValues)

	if len(fields) == 0 && !hasCoordinates {
		return nil, nil
	}

	// Topology diff：在 UPDATE 之前 SELECT 一次当前 LAC/TAC 旧值，与新 fields 对比，
	// 决定调用方是否要 publish device.attributes.changed。读 SELECT 只在 fields 真
	// 含 LAC/TAC 时跑，避免给纯指标列同步路径增加无用 IO。
	var changedAttrs []string
	if anyKey(fields, topologyAttributeColumns) {
		oldAttrs, err := s.infoRepo.GetTopologyAttributes(ctx, deviceID)
		if err != nil {
			s.logger.Warn("read old topology attributes failed; will skip change-event publish",
				zap.String("device_id", deviceID.String()),
				zap.Error(err))
		} else {
			for _, col := range topologyAttributeColumns {
				newVal, hasNew := fields[col].(string)
				oldVal, hasOld := oldAttrs[col]
				switch {
				case hasNew && hasOld && newVal != oldVal:
					changedAttrs = append(changedAttrs, col)
				case hasNew && !hasOld:
					changedAttrs = append(changedAttrs, col)
				}
			}
		}
	}

	if len(fields) > 0 {
		if err := s.infoRepo.UpdateSyncFields(ctx, deviceID, fields); err != nil {
			return nil, fmt.Errorf("update device info sync fields: %w", err)
		}
	}

	if hasCoordinates {
		receivedAt := time.Now().UTC()
		locationAccepted := true
		gpsLockStatus := CalcGPSStatus(paramValues)
		observation := ReportedLocation{
			Latitude:       latitude,
			Longitude:      longitude,
			GPSHeight:      reportedGPSHeight,
			GPSLockStatus:  &gpsLockStatus,
			SatelliteCount: lookupGPSSatelliteCount(paramValues),
			ObservedAt:     receivedAt,
			ReceivedAt:     receivedAt,
			SourcePath:     sourcePath,
		}
		if s.locationObservationRepo != nil {
			if _, err := s.locationObservationRepo.SaveLatestWithOutbox(
				ctx,
				deviceID,
				observation,
			); errors.Is(err, ErrLocationSourceNotAllowed) {
				locationAccepted = false
				s.logger.Debug(
					"ignored device GPS for external positioning mode",
					zap.String("device_id", deviceID.String()),
					zap.String("source_path", sourcePath),
				)
			} else if err != nil {
				return nil, fmt.Errorf("update reported GPS coordinates: %w", err)
			}
		}

		if locationAccepted && s.coordinateWriter != nil {
			var accepted *Location
			if reader, ok := s.coordinateWriter.(DeviceCoordinateReader); ok {
				var err error
				accepted, err = reader.GetCoordinates(ctx, deviceID)
				if err != nil {
					return nil, fmt.Errorf("read accepted device coordinates: %w", err)
				}
			}
			if accepted == nil {
				if err := s.coordinateWriter.UpdateCoordinates(ctx, deviceID, latitude, longitude); err != nil {
					return nil, fmt.Errorf("initialize device coordinates: %w", err)
				}
			}
		}
	}

	s.logger.Debug("synced device info from parameters",
		zap.String("device_id", deviceID.String()),
		zap.Int("fields_synced", len(fields)),
		zap.Strings("topology_changed", changedAttrs),
	)

	return changedAttrs, nil
}

// anyKey reports whether m contains any of the given keys.
func anyKey(m map[string]interface{}, keys []string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

// RecordOnline updates the last_online_time when a device comes online (from offline to active).
// It also sets first_online_time if this is the device's first online event.
func (s *InfoSyncer) RecordOnline(ctx context.Context, deviceID uuid.UUID) error {
	now := time.Now()
	fields := map[string]interface{}{
		"last_online_time": now,
	}

	// 检查是否首次上线，如果是则同时设置 first_online_time
	info, err := s.infoRepo.GetByDeviceID(ctx, deviceID)
	if err == nil && (info == nil || info.FirstOnlineTime == nil) {
		fields["first_online_time"] = now
	}

	return s.infoRepo.UpdateSyncFields(ctx, deviceID, fields)
}
