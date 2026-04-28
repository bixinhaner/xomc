package license

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Service provides business logic for license management.
type Service struct {
	repo   LicenseRepository
	logger *zap.Logger
}

// NewService creates a new license Service.
func NewService(repo LicenseRepository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.Named("license"),
	}
}

// List returns a paginated list of licenses.
func (s *Service) List(ctx context.Context, filter LicenseFilter) (*model.ListResponse[License], error) {
	return s.repo.List(ctx, filter)
}

// GetByID retrieves a license by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*License, error) {
	return s.repo.GetByID(ctx, id)
}

// Summary returns aggregated license statistics.
func (s *Service) Summary(ctx context.Context) (*LicenseSummary, error) {
	return s.repo.Summary(ctx)
}

// Activate finds a license by code and sets its status to active.
func (s *Service) Activate(ctx context.Context, licenseCode string) (*License, error) {
	lic, err := s.repo.GetByCode(ctx, licenseCode)
	if err != nil {
		return nil, fmt.Errorf("get license by code: %w", err)
	}

	if lic.Status == StatusActive {
		return lic, nil
	}

	if lic.Status == StatusRevoked {
		return nil, commonerrors.NewBusinessError(9100, "cannot activate a revoked license", commonerrors.ErrInvalidInput)
	}

	lic.Status = StatusActive
	if err := s.repo.Update(ctx, lic); err != nil {
		return nil, fmt.Errorf("activate license: %w", err)
	}

	s.logger.Info("license activated",
		zap.String("license_id", lic.ID.String()),
		zap.String("license_code", licenseCode),
	)

	return lic, nil
}

// Revoke sets a license status to revoked.
func (s *Service) Revoke(ctx context.Context, id uuid.UUID) error {
	lic, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get license: %w", err)
	}

	if lic.Status == StatusRevoked {
		return nil
	}

	lic.Status = StatusRevoked
	if err := s.repo.Update(ctx, lic); err != nil {
		return fmt.Errorf("revoke license: %w", err)
	}

	s.logger.Info("license revoked",
		zap.String("license_id", id.String()),
	)

	return nil
}

// Import creates a new license record.
//
// Returns:
//   - commonerrors.ErrInvalidInput (→ 400) when required fields are missing.
//   - commonerrors.ErrAlreadyExists (→ 409) when license_code already exists.
func (s *Service) Import(ctx context.Context, lic *License) (*License, error) {
	if lic == nil {
		return nil, fmt.Errorf("license payload is nil: %w", commonerrors.ErrInvalidInput)
	}
	if lic.LicenseCode == "" {
		return nil, fmt.Errorf("license_code is required: %w", commonerrors.ErrInvalidInput)
	}
	if lic.LicenseName == "" {
		return nil, fmt.Errorf("license_name is required: %w", commonerrors.ErrInvalidInput)
	}
	if lic.ProductName == "" {
		return nil, fmt.Errorf("product_name is required: %w", commonerrors.ErrInvalidInput)
	}

	if lic.Status == "" {
		lic.Status = StatusPending
	}
	if lic.Features == nil {
		lic.Features = []byte("[]")
	}

	// Pre-check: if a license with the same code already exists, return 409
	// instead of letting the unique-violation surface as 500.
	existing, err := s.repo.GetByCode(ctx, lic.LicenseCode)
	if err != nil && !errors.Is(err, commonerrors.ErrNotFound) {
		return nil, fmt.Errorf("check existing license: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("license_code %q already exists: %w", lic.LicenseCode, commonerrors.ErrAlreadyExists)
	}

	if err := s.repo.Create(ctx, lic); err != nil {
		// Repository may also detect unique-violation as a safety net.
		return nil, fmt.Errorf("import license: %w", err)
	}

	s.logger.Info("license imported",
		zap.String("license_id", lic.ID.String()),
		zap.String("license_code", lic.LicenseCode),
	)

	return lic, nil
}
