package device

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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
	matches := runTimeRegex.FindStringSubmatch(val)
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
var universalInformMapping = map[string]string{
	"Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus": "alarm_severity",
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
