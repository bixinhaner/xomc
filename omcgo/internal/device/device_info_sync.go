package device

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// runTimeRegex matches TR069 run time format like "40d 4h 58m" or "4h 58m" or "58m"
// Captures: days, hours, minutes, seconds (each optional)
var runTimeRegex = regexp.MustCompile(`(?:(\d+)d\s*)?(?:(\d+)h\s*)?(?:(\d+)m\s*)?(?:(\d+)s)?`)

// parseRunTimeToSeconds parses TR069 run time format to seconds.
// Supported formats: "40d 4h 58m", "4h 58m", "58m", "40d 4h 58m 30s"
// Returns 0 if parsing fails.
func parseRunTimeToSeconds(val string) int64 {
	matches := runTimeRegex.FindStringSubmatch(strings.TrimSpace(val))
	if matches == nil {
		return 0
	}

	var totalSeconds int64
	// matches[0] is the full match, matches[1-4] are days, hours, minutes, seconds
	if matches[1] != "" {
		if days, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			totalSeconds += days * 86400
		}
	}
	if matches[2] != "" {
		if hours, err := strconv.ParseInt(matches[2], 10, 64); err == nil {
			totalSeconds += hours * 3600
		}
	}
	if matches[3] != "" {
		if minutes, err := strconv.ParseInt(matches[3], 10, 64); err == nil {
			totalSeconds += minutes * 60
		}
	}
	if matches[4] != "" {
		if seconds, err := strconv.ParseInt(matches[4], 10, 64); err == nil {
			totalSeconds += seconds
		}
	}

	return totalSeconds
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
var universalInformMapping = map[string]string{
	"Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus": "alarm_severity",
	// TR-181 Ethernet 标准 path（取代 CMCC X_CMCC_MACAddress，多数 CPE 上报此 path）
	"Device.Ethernet.Interface.MACAddress": "mac",
	// 发射功率：CPE 实际上报 MaxTxPower（Capabilities 子树）而非 ReferenceSignalPower
	"Device.Services.FAPService.1.Capabilities.MaxTxPower": "transmit_power",
	// LTE 小区配置（device_info 表 Phase 2 新增列）
	"Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC": "tac",
	// GSM 位置区码：补全 device_groups LAC 匹配模式所需的设备侧数据源（T-2026-05-25）。
	// 与 TAC 平行，CPE 同时上报时 LAC 多见于双模 / GSM 设备。
	"Device.DeviceInfo.BTS.CurrentLac":                                                     "lac",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.FreqBandIndicator":                 "band",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNUL":                          "ul_earfcn",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment":      "subframe_assignment",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns": "special_subframe",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig":  "root_index",
	// GPS（卫星数 + 高度）
	"Device.FAP.GPS.NumberOfSatellites": "gps_satellites",
	// 锁状态 = FAP AdminState（"true"=已激活/unlocked, "false"=锁定）
	"Device.Services.FAPService.1.FAPControl.LTE.AdminState": "lock_status",
}

// gpsHeightCandidatePaths lists possible TR069 paths for GPS height.
// CPE 上报的字段名有拼写差异（`altidute` 是 Baicells 固件字面值），按顺序
// 优先匹配第一个有值的 path。Phase 3 fallback 链；后续固件统一后可裁剪。
var gpsHeightCandidatePaths = []string{
	"Device.FAP.GPS.altidute", // Baicells BaiBLQ 实测拼写
	"Device.FAP.GPS.Altitude", // TR-181 spec 标准拼写
	"Device.FAP.GPS.Height",
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

// InfoSyncer extracts key TR069 parameters from device_parameters
// and updates the corresponding device_info columns for fast query access.
type DeviceCoordinateWriter interface {
	UpdateCoordinates(ctx context.Context, id uuid.UUID, latitude, longitude float64) error
}

type InfoSyncer struct {
	infoRepo         DeviceInfoRepository
	paramRepo        DeviceParameterRepository
	coordinateWriter DeviceCoordinateWriter
	carrierRegistry  *carrier.CarrierRegistry
	logger           *zap.Logger
}

// NewInfoSyncer creates a new InfoSyncer.
func NewInfoSyncer(
	infoRepo DeviceInfoRepository,
	paramRepo DeviceParameterRepository,
	coordinateWriter DeviceCoordinateWriter,
	carrierRegistry *carrier.CarrierRegistry,
	logger *zap.Logger,
) *InfoSyncer {
	return &InfoSyncer{
		infoRepo:         infoRepo,
		paramRepo:        paramRepo,
		coordinateWriter: coordinateWriter,
		carrierRegistry:  carrierRegistry,
		logger:           logger,
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
func (s *InfoSyncer) SyncFromParameters(ctx context.Context, deviceID uuid.UUID, carrierCode model.CarrierCode, tech model.Technology) ([]string, error) {
	c, err := s.carrierRegistry.Get(carrierCode)
	if err != nil {
		return nil, fmt.Errorf("get carrier adapter: %w", err)
	}

	mapping := c.GetInfoParamMapping(tech)
	if len(mapping) == 0 {
		return nil, nil
	}

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
	if tech == model.TechNR {
		if mac, ok := lookupWANMAC(paramValues); ok {
			fields["mac"] = mac
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
	fields["mme_status"] = CalcMMEStatus(paramValues)
	fields["sync_status"] = CalcSyncStatus(paramValues)
	fields["rf_status"] = CalcRFStatus(paramValues)
	fields["gps_status"] = CalcGPSStatus(paramValues)
	fields["num_of_cells"] = CalcNumOfCells(paramValues)
	fields["license_status"] = CalcLicenseStatus(paramValues)

	// Phase 3 派生字段（设计文档 §4.2 Layer C）：
	if v, ok := lookupGPSHeight(paramValues); ok {
		fields["gps_height"] = v
	}
	if eci, ok := fields["eci"].(string); ok {
		if enb, ok := deriveEnbID(eci); ok {
			fields["enb_id"] = enb
		}
	}
	if model, ok := deriveNetworkModel(paramValues); ok {
		fields["network_model"] = model
	}

	latitude, longitude, hasCoordinates := lookupGPSCoordinates(paramValues)

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

	if hasCoordinates && s.coordinateWriter != nil {
		if err := s.coordinateWriter.UpdateCoordinates(ctx, deviceID, latitude, longitude); err != nil {
			return nil, fmt.Errorf("update device coordinates: %w", err)
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
	if err == nil && info.FirstOnlineTime == nil {
		fields["first_online_time"] = now
	}

	return s.infoRepo.UpdateSyncFields(ctx, deviceID, fields)
}
