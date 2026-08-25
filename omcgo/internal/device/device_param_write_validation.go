package device

import (
	"context"
	"strings"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
)

// parameterWriteModel keeps validation and translation on the same immutable
// MappingSet snapshot. Validation remains private-path based; translation is
// only an adapter for API callers that submit standard paths.
type parameterWriteModel struct {
	validator  *parammodel.MappingValidator
	translator *parammodel.Translator
}

type parameterWriteValidationResult struct {
	RebootRequired bool
	RebootTarget   int
	Errors         []*parammodel.MappingValidationError
}

func (h *ParameterTreeHandler) resolveParameterWriteModel(
	ctx context.Context,
	dev *model.Device,
) *parameterWriteModel {
	if h.paramRegistry == nil || h.productRegistry == nil || dev == nil || dev.ProductClass == "" {
		return nil
	}
	match, err := h.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil {
		return nil
	}
	set, err := h.paramRegistry.GetByProduct(ctx, match.Product.ID, dev.FirmwareVersion)
	if err != nil || set == nil {
		return nil
	}
	return &parameterWriteModel{
		validator:  parammodel.NewMappingValidator(set),
		translator: parammodel.NewTranslator(set, nil, h.logger),
	}
}

func validateParameterWrites(
	writeModel *parameterWriteModel,
	items []ParameterValueItem,
) parameterWriteValidationResult {
	var result parameterWriteValidationResult
	if writeModel == nil || writeModel.validator == nil {
		return result
	}

	for _, item := range items {
		validationPath := item.Path
		mapping := writeModel.validator.LookupParam(validationPath)
		if mapping == nil && writeModel.translator != nil {
			translated := writeModel.translator.ToPrivate(item.Path)
			if translated.Found {
				validationPath = translated.Translated
				mapping = writeModel.validator.LookupParam(validationPath)
			}
		}

		if validationError := writeModel.validator.ValidateValue(validationPath, item.Value); validationError != nil {
			errorCopy := *validationError
			errorCopy.Path = item.Path
			result.Errors = append(result.Errors, &errorCopy)
			continue
		}
		if mapping != nil &&
			(mapping.ChangeApplies == "RebootRequired" || mapping.ChangeApplies == "NotifyRequired") {
			result.RebootRequired = true
			result.RebootTarget = mergeRebootTarget(result.RebootTarget, rebootTargetForPath(validationPath))
		}
	}
	return result
}

func rebootTargetForPath(path string) int {
	if !strings.HasPrefix(path, "Device.ImsCore.") {
		return 0
	}
	if strings.HasPrefix(path, "Device.ImsCore.CoreInfoConfig.") {
		return 3
	}
	if path == "Device.ImsCore.BaseConfig.WEB_LOG_LEVEL" ||
		strings.HasPrefix(path, "Device.ImsCore.BaseConfig.WEB_SERV_PORT") ||
		strings.HasPrefix(path, "Device.ImsCore.BaseConfig.SYN_SERVER") ||
		strings.HasPrefix(path, "Device.ImsCore.BaseConfig.Web_HSS_IP") ||
		strings.HasPrefix(path, "Device.ImsCore.BaseConfig.HTTPS_LOGIN_FLAG") {
		return 2
	}
	return 1
}

func mergeRebootTarget(current, next int) int {
	if current == 0 {
		return next
	}
	if next == 0 || current == next {
		return current
	}
	return 3
}
