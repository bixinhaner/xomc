package device

import (
	"context"
	"fmt"
	"regexp"
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
	"Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus":                      "alarm_severity",
	// TR-181 Ethernet 标准 path（取代 CMCC X_CMCC_MACAddress，多数 CPE 上报此 path）
	"Device.Ethernet.Interface.MACAddress": "mac",
	// 发射功率：CPE 实际上报 MaxTxPower（Capabilities 子树）而非 ReferenceSignalPower
	"Device.Services.FAPService.1.Capabilities.MaxTxPower": "transmit_power",
	// LTE 小区配置（device_info 表 Phase 2 新增列）
	"Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC":                  "tac",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.FreqBandIndicator": "band",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNUL":          "ul_earfcn",
	"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment":    "subframe_assignment",
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

// TR069 parameter paths for run_time with priority
const (
	// ParamUpTime is the standard TR069 UpTime parameter (unit: seconds, direct storage)
	ParamUpTime = "Device.DeviceInfo.UpTime"
	// ParamStationRunTime is the vendor-specific run time parameter (format: "40d 4h 58m")
	ParamStationRunTime = "Device.DeviceInfo.X_COM_STATION_RUN_Time"
)

// InfoSyncer extracts key TR069 parameters from device_parameters
// and updates the corresponding device_info columns for fast query access.
type InfoSyncer struct {
	infoRepo        DeviceInfoRepository
	paramRepo       DeviceParameterRepository
	carrierRegistry *carrier.CarrierRegistry
	logger          *zap.Logger
}

// NewInfoSyncer creates a new InfoSyncer.
func NewInfoSyncer(
	infoRepo DeviceInfoRepository,
	paramRepo DeviceParameterRepository,
	carrierRegistry *carrier.CarrierRegistry,
	logger *zap.Logger,
) *InfoSyncer {
	return &InfoSyncer{
		infoRepo:        infoRepo,
		paramRepo:       paramRepo,
		carrierRegistry: carrierRegistry,
		logger:          logger,
	}
}

// SyncFromParameters reads the device's stored TR069 parameters and updates
// the corresponding device_info columns based on the carrier's mapping.
func (s *InfoSyncer) SyncFromParameters(ctx context.Context, deviceID uuid.UUID, carrierCode model.CarrierCode, tech model.Technology) error {
	c, err := s.carrierRegistry.Get(carrierCode)
	if err != nil {
		return fmt.Errorf("get carrier adapter: %w", err)
	}

	mapping := c.GetInfoParamMapping(tech)
	if len(mapping) == 0 {
		return nil
	}

	// Get all parameters for this device
	params, err := s.paramRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device parameters: %w", err)
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

	if len(fields) == 0 {
		return nil
	}

	if err := s.infoRepo.UpdateSyncFields(ctx, deviceID, fields); err != nil {
		return fmt.Errorf("update device info sync fields: %w", err)
	}

	s.logger.Debug("synced device info from parameters",
		zap.String("device_id", deviceID.String()),
		zap.Int("fields_synced", len(fields)),
	)

	return nil
}

// RecordOffline updates the last_offline_time when a device goes offline.
func (s *InfoSyncer) RecordOffline(ctx context.Context, deviceID uuid.UUID) error {
	return s.infoRepo.UpdateSyncFields(ctx, deviceID, map[string]interface{}{
		"last_offline_time": time.Now(),
	})
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
