package device

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// universalInformMapping maps TR069 parameter paths to device_info columns
// for parameters that are identical across all carriers (not carrier-specific).
var universalInformMapping = map[string]string{
	"Device.DeviceInfo.X_COM_STATION_RUN_Time":                          "run_time",
	"Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus": "alarm_severity",
}

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
