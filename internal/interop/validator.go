package interop

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/omcr/device"
)

// DataModelValidator compares a device's actual reported parameters against its
// resolved data model definition to produce a conformance report.
type DataModelValidator struct {
	dataModelReg *datamodel.DataModelRegistry
	paramRepo    device.DeviceParameterRepository
	deviceRepo   device.DeviceRepository
	logger       *zap.Logger
}

// NewDataModelValidator creates a new DataModelValidator.
func NewDataModelValidator(
	dataModelReg *datamodel.DataModelRegistry,
	paramRepo device.DeviceParameterRepository,
	deviceRepo device.DeviceRepository,
	logger *zap.Logger,
) *DataModelValidator {
	return &DataModelValidator{
		dataModelReg: dataModelReg,
		paramRepo:    paramRepo,
		deviceRepo:   deviceRepo,
		logger:       logger.Named("datamodel-validator"),
	}
}

// ValidateDevice resolves the data model for the given device and compares
// the device's actual parameters against the model's expected parameter tree.
func (v *DataModelValidator) ValidateDevice(
	ctx context.Context,
	deviceID uuid.UUID,
	carrier model.CarrierCode,
	tech model.Technology,
) (*ValidationReport, error) {
	// Fetch the device.
	dev, err := v.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", deviceID, err)
	}

	// Resolve the data model using the provided carrier/tech with the device's OUI/ProductClass.
	dm, err := v.dataModelReg.Resolve(ctx, carrier, tech, dev.OUI, dev.ProductClass)
	if err != nil {
		return nil, fmt.Errorf("resolve data model: %w", err)
	}
	if dm == nil {
		return nil, fmt.Errorf("no data model found for carrier=%s tech=%s oui=%s product=%s",
			carrier, tech, dev.OUI, dev.ProductClass)
	}

	// Parse the expected parameter tree.
	var expectedParams []datamodel.Parameter
	if err := json.Unmarshal(dm.ParameterTree, &expectedParams); err != nil {
		return nil, fmt.Errorf("parse data model parameter tree: %w", err)
	}

	// Fetch the device's actual parameters.
	actualParams, err := v.paramRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device parameters: %w", err)
	}

	// Build lookups.
	actualByPath := make(map[string]model.DeviceParameter, len(actualParams))
	for _, p := range actualParams {
		actualByPath[p.ParameterPath] = p
	}

	expectedByPath := make(map[string]datamodel.Parameter, len(expectedParams))
	for _, p := range expectedParams {
		expectedByPath[p.Path] = p
	}

	report := &ValidationReport{
		DeviceID:     deviceID.String(),
		DeviceSN:     dev.SerialNumber,
		ModelVersion: dm.Version,
		TotalParams:  len(expectedParams),
		CreatedAt:    time.Now(),
	}

	// Compare expected vs actual.
	matched := 0
	for _, ep := range expectedParams {
		ap, found := actualByPath[ep.Path]
		if !found {
			report.MissingParams = append(report.MissingParams, ep.Path)
			continue
		}

		// Check type mismatch.
		if string(ap.ParameterType) != ep.Type {
			report.MismatchParams = append(report.MismatchParams, ParamMismatch{
				Path:     ep.Path,
				Expected: ep.Type,
				Actual:   string(ap.ParameterType),
				Type:     "type_mismatch",
			})
			continue
		}

		// Check writable mismatch.
		if ap.Writable != ep.Writable {
			report.MismatchParams = append(report.MismatchParams, ParamMismatch{
				Path:     ep.Path,
				Expected: fmt.Sprintf("writable=%v", ep.Writable),
				Actual:   fmt.Sprintf("writable=%v", ap.Writable),
				Type:     "writable_mismatch",
			})
			continue
		}

		matched++
	}

	// Find extra parameters not in the data model.
	for _, ap := range actualParams {
		if _, found := expectedByPath[ap.ParameterPath]; !found {
			report.ExtraParams = append(report.ExtraParams, ap.ParameterPath)
		}
	}

	report.MatchedParams = matched
	if report.TotalParams > 0 {
		report.Score = float64(matched) / float64(report.TotalParams) * 100.0
	}

	v.logger.Info("device validation complete",
		zap.String("device_id", deviceID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("total", report.TotalParams),
		zap.Int("matched", report.MatchedParams),
		zap.Int("missing", len(report.MissingParams)),
		zap.Int("mismatches", len(report.MismatchParams)),
		zap.Int("extra", len(report.ExtraParams)),
		zap.Float64("score", report.Score),
	)

	return report, nil
}
