package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/deviceaccess"
)

type candidateOwnershipRegistrar struct {
	repository *device.PgRegistrationRepository
}

func (r candidateOwnershipRegistrar) EnsureCandidateOwnership(
	ctx context.Context,
	tx pgx.Tx,
	request deviceaccess.CandidateOwnershipRegistration,
) error {
	if r.repository == nil {
		return fmt.Errorf("candidate ownership registration repository is not configured")
	}
	defaultGroupID := uuid.MustParse(global.DefaultLevel2GroupID)
	err := r.repository.EnsureCandidateOwnershipTx(ctx, tx, &device.DeviceRegistration{
		SerialNumber: request.SerialNumber,
		GroupID:      &defaultGroupID,
		Carrier:      model.CarrierCode(request.Carrier),
		Remark:       "device access candidate approved",
		CreatedBy:    request.CreatedBy,
	})
	if errors.Is(err, device.ErrRegistrationOwnershipConflict) {
		return fmt.Errorf("candidate identity is already owned: %w", deviceaccess.ErrAssetOwnershipConflict)
	}
	return err
}

var _ deviceaccess.CandidateOwnershipRegistrar = candidateOwnershipRegistrar{}
