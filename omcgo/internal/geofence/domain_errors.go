package geofence

import (
	"errors"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const (
	ErrCodeDefinitionNameInvalid   = 2601
	ErrCodeDefinitionNameDuplicate = 2602
	ErrCodeActiveBatchJobs         = 2603
	ErrCodeStaleLifecyclePreview   = 2604
)

func invalidDefinitionNameError() error {
	return commonerrors.NewBusinessError(
		ErrCodeDefinitionNameInvalid,
		"invalid geofence name",
		commonerrors.ErrInvalidInput,
	)
}

func duplicateDefinitionNameError(err error) error {
	return commonerrors.NewBusinessError(
		ErrCodeDefinitionNameDuplicate,
		"geofence name already exists for this carrier",
		errors.Join(commonerrors.ErrAlreadyExists, err),
	)
}
