package interop

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/product"
)

// DataModelValidator compares a device's actual reported parameters against its
// resolved param mapping definition to produce a conformance report.
//
// T-0098 P5-01：旧 dataModelReg 路径已删除；现仅走 paramRegistry + productRegistry。
// 设备 ProductClass 未命中 product 或 ParamRegistry.GetByProduct 失败 → 直接报错。
type DataModelValidator struct {
	paramRegistry   *parammodel.Registry
	productRegistry *product.Registry
	paramRepo       device.DeviceParameterRepository
	deviceRepo      device.DeviceRepository
	logger          *zap.Logger
}

// NewDataModelValidator creates a new DataModelValidator.
func NewDataModelValidator(
	paramReg *parammodel.Registry,
	prodReg *product.Registry,
	paramRepo device.DeviceParameterRepository,
	deviceRepo device.DeviceRepository,
	logger *zap.Logger,
) *DataModelValidator {
	return &DataModelValidator{
		paramRegistry:   paramReg,
		productRegistry: prodReg,
		paramRepo:       paramRepo,
		deviceRepo:      deviceRepo,
		logger:          logger.Named("datamodel-validator"),
	}
}

// validatorExpectedParam 是 parammodel.ParamMapping 的最小公共视图。
type validatorExpectedParam struct {
	Path     string
	Type     string
	Writable bool
}

// resolveExpectedParams 通过 ParamRegistry 获取期望参数集；
// 任意一步失败 → 返回 (nil, "", "", err)，调用方据此报错。
func (v *DataModelValidator) resolveExpectedParams(ctx context.Context, dev *model.Device) ([]validatorExpectedParam, string, string, error) {
	if v.paramRegistry == nil || v.productRegistry == nil {
		return nil, "", "", fmt.Errorf("param registry not configured")
	}
	if dev == nil || dev.ProductClass == "" {
		return nil, "", "", fmt.Errorf("device productClass missing")
	}
	match, err := v.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil {
		return nil, "", "", fmt.Errorf("match productClass: %w", err)
	}
	if match == nil || match.Product == nil {
		return nil, "", "", fmt.Errorf("no product match for productClass=%s", dev.ProductClass)
	}
	set, err := v.paramRegistry.GetByProduct(ctx, match.Product.ID, dev.FirmwareVersion)
	if err != nil {
		return nil, "", "", fmt.Errorf("paramRegistry.GetByProduct: %w", err)
	}
	if set == nil {
		return nil, "", "", fmt.Errorf("no param mapping for product %s @ %s", match.Product.ID, dev.FirmwareVersion)
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

// ValidateDevice resolves the param mapping for the given device and compares
// the device's actual parameters against the mapping's expected list.
//
// 旧签名兼容保留 carrier/tech 入参，但新栈不再消费（路由判定走 ProductClass）。
func (v *DataModelValidator) ValidateDevice(
	ctx context.Context,
	deviceID uuid.UUID,
	_ model.CarrierCode,
	_ model.Technology,
) (*ValidationReport, error) {
	dev, err := v.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", deviceID, err)
	}

	expected, source, fwVersion, err := v.resolveExpectedParams(ctx, dev)
	if err != nil {
		return nil, err
	}

	return v.compare(ctx, deviceID, dev, expected, fmt.Sprintf("paramRegistry:%s@%s", source, fwVersion))
}

// compare 公共比对路径。
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
