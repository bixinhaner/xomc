package device

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/global"
	"go.uber.org/zap"
)

// RegistrationService provides business logic for device pre-registration.
type RegistrationService struct {
	repo   RegistrationRepository
	logger *zap.Logger
}

// NewRegistrationService creates a new RegistrationService.
func NewRegistrationService(repo RegistrationRepository, logger *zap.Logger) *RegistrationService {
	return &RegistrationService{repo: repo, logger: logger.Named("registration-service")}
}

// Register creates a single device pre-registration entry.
func (s *RegistrationService) Register(ctx context.Context, req CreateRegistrationRequest, operator string) (*DeviceRegistration, error) {
	// Check if SN already pending.
	existing, err := s.repo.GetBySerialNumber(ctx, req.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("check existing registration: %w", err)
	}
	if existing != nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeRegistrationDuplicate, "serial number already registered", nil)
	}

	reg := &DeviceRegistration{
		SerialNumber: req.SerialNumber,
		Carrier:      model.CarrierCode(req.Carrier),
		SiteName:     req.SiteName,
		Longitude:    req.Longitude,
		Latitude:     req.Latitude,
		Remark:       req.Remark,
		CreatedBy:    operator,
	}

	if req.GroupID != "" {
		gid, err := uuid.Parse(req.GroupID)
		if err != nil {
			return nil, commonerrors.NewBusinessError(global.ErrCodeRegistrationInvalidSN, "invalid group_id", err)
		}
		reg.GroupID = &gid
	}

	if err := s.repo.Create(ctx, reg); err != nil {
		return nil, fmt.Errorf("create registration: %w", err)
	}

	return reg, nil
}

// BatchRegister creates multiple pre-registration entries.
func (s *RegistrationService) BatchRegister(ctx context.Context, regs []*DeviceRegistration) (*ImportResult, error) {
	result := &ImportResult{Total: len(regs)}

	created, err := s.repo.BatchCreate(ctx, regs)
	result.Success = created
	result.Failed = result.Total - created

	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	return result, nil
}

// List returns a paginated list of device registrations.
func (s *RegistrationService) List(ctx context.Context, filter RegistrationFilter) (*model.ListResponse[DeviceRegistration], error) {
	return s.repo.List(ctx, filter)
}

// Delete removes a registration entry.
func (s *RegistrationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// MarkOnline marks a registration as online (called when device first connects).
func (s *RegistrationService) MarkOnline(ctx context.Context, sn string) {
	reg, err := s.repo.GetBySerialNumber(ctx, sn)
	if err != nil || reg == nil {
		return
	}
	if err := s.repo.UpdateStatus(ctx, reg.ID, string(global.RegistrationOnline)); err != nil {
		s.logger.Warn("mark registration online", zap.String("sn", sn), zap.Error(err))
	}
}
