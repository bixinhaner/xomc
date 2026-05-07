package interop

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/product"
)

// DataModelValidator compares a device's actual reported parameters against its
// resolved data model definition to produce a conformance report.
//
// T-0098 P2-08：双栈期 dataModelReg 与 paramRegistry 共存。当 paramRegistryEnabled
// 且 product.MatchProductClass 命中且 ParamRegistry.GetByProduct 命中时，
// 走 ParamMapping 元属性校验；否则降级 dataModelReg。
type DataModelValidator struct {
	dataModelReg         *datamodel.DataModelRegistry
	paramRegistry        *parammodel.Registry
	productRegistry      *product.Registry
	paramRegistryEnabled bool
	paramRepo            device.DeviceParameterRepository
	deviceRepo           device.DeviceRepository
	logger               *zap.Logger
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

// WithParamRegistry 启用 T-0098 P2-08 dual-stack 模式。
func (v *DataModelValidator) WithParamRegistry(paramReg *parammodel.Registry, prodReg *product.Registry, enabled bool) *DataModelValidator {
	v.paramRegistry = paramReg
	v.productRegistry = prodReg
	v.paramRegistryEnabled = enabled && paramReg != nil && prodReg != nil
	return v
}

// validatorExpectedParam 是 datamodel.Parameter 与 parammodel.ParamMapping 的最小公共视图。
type validatorExpectedParam struct {
	Path     string
	Type     string
	Writable bool
}

// resolveExpectedParams 在 dual-stack 启用时优先走 ParamRegistry；返回 (params, source, version, nil)
// 表示成功；返回 (nil, "", "", nil) 表示未启用或未命中，调用方降级 dataModelReg。
func (v *DataModelValidator) resolveExpectedParams(ctx context.Context, dev *model.Device) ([]validatorExpectedParam, string, string, error) {
	if !v.paramRegistryEnabled || v.paramRegistry == nil || v.productRegistry == nil {
		return nil, "", "", nil
	}
	if dev == nil || dev.ProductClass == "" {
		return nil, "", "", nil
	}
	match, err := v.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil {
		return nil, "", "", err
	}
	set, err := v.paramRegistry.GetByProduct(ctx, match.Product.ID, dev.FirmwareVersion)
	if err != nil || set == nil {
		return nil, "", "", err
	}
	out := make([]validatorExpectedParam, 0, len(set.Mappings))
	for _, m := range set.Mappings {
		if m.EntryType != "parameter" {
			continue
		}
		out = append(out, validatorExpectedParam{
			Path:     m.PrivatePath,
			Type:     m.DataType,
			Writable: parammodel.IsAccessWritable(m.Access),
		})
	}
	return out, string(set.Source), dev.FirmwareVersion, nil
}

// ValidateDevice resolves the data model for the given device and compares
// the device's actual parameters against the model's expected parameter tree.
//
// T-0098 P2-08：dual-stack 启用时优先 ParamRegistry，未命中或失败降级 dataModelReg。
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

	// 优先尝试 ParamRegistry 路径；source 与 modelVersion 区分 default/discovered。
	if expected, source, fwVersion, _ := v.resolveExpectedParams(ctx, dev); expected != nil {
		return v.compare(ctx, deviceID, dev, expected, fmt.Sprintf("paramRegistry:%s@%s", source, fwVersion))
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
	expected := make([]validatorExpectedParam, 0, len(expectedParams))
	for _, p := range expectedParams {
		expected = append(expected, validatorExpectedParam{Path: p.Path, Type: p.Type, Writable: p.Writable})
	}
	return v.compare(ctx, deviceID, dev, expected, dm.Version)
}

// compare 公共比对路径——两栈聚合到同一逻辑，只在前置取数路径上分叉。
func (v *DataModelValidator) compare(
	ctx context.Context,
	deviceID uuid.UUID,
	dev *model.Device,
	expectedParams []validatorExpectedParam,
	modelVersion string,
) (*ValidationReport, error) {
	actualParams, err := v.paramRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device parameters: %w", err)
	}

	actualByPath := make(map[string]model.DeviceParameter, len(actualParams))
	for _, p := range actualParams {
		actualByPath[p.ParameterPath] = p
	}

	expectedByPath := make(map[string]validatorExpectedParam, len(expectedParams))
	for _, p := range expectedParams {
		expectedByPath[p.Path] = p
	}

	report := &ValidationReport{
		DeviceID:     deviceID.String(),
		DeviceSN:     dev.SerialNumber,
		ModelVersion: modelVersion,
		TotalParams:  len(expectedParams),
		CreatedAt:    time.Now(),
	}

	matched := 0
	for _, ep := range expectedParams {
		ap, found := actualByPath[ep.Path]
		if !found {
			report.MissingParams = append(report.MissingParams, ep.Path)
			continue
		}
		if string(ap.ParameterType) != ep.Type {
			report.MismatchParams = append(report.MismatchParams, ParamMismatch{
				Path:     ep.Path,
				Expected: ep.Type,
				Actual:   string(ap.ParameterType),
				Type:     "type_mismatch",
			})
			continue
		}
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
		zap.String("model_version", modelVersion),
		zap.Int("total", report.TotalParams),
		zap.Int("matched", report.MatchedParams),
		zap.Int("missing", len(report.MissingParams)),
		zap.Int("mismatches", len(report.MismatchParams)),
		zap.Int("extra", len(report.ExtraParams)),
		zap.Float64("score", report.Score),
	)

	return report, nil
}
