package device

import (
	"context"

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
		}
	}
	return result
}
