package license

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Service provides business logic for license management.
type Service struct {
	repo     LicenseRepository
	logRepo  LicenseLogRepository // T-0100-P2 Summary 卡 enforcement_hits_7d 用；nil 时退化为 0
	logger   *zap.Logger
	enforcer Enforcer // optional; nil during bootstrap before Enforcer is wired
}

// NewService creates a new license Service.
func NewService(repo LicenseRepository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.Named("license"),
	}
}

// SetEnforcer wires an Enforcer so the Service can call Invalidate after
// state-mutating operations (Import / Activate / Revoke). Optional.
func (s *Service) SetEnforcer(e Enforcer) {
	s.enforcer = e
}

// SetLogRepo 注入 LicenseLogRepository（T-0100-P2），让 Summary 卡片能拉
// 近 7 天 enforcement 命中数。
func (s *Service) SetLogRepo(repo LicenseLogRepository) {
	s.logRepo = repo
}

// invalidateEnforcerCache calls Invalidate on the Enforcer if wired.
func (s *Service) invalidateEnforcerCache() {
	if s.enforcer != nil {
		s.enforcer.Invalidate()
	}
}

// Quota is a passthrough to Enforcer.Quota for the handler. Returns an
// empty quota with HasActiveLicense=false when no Enforcer is wired.
func (s *Service) Quota(ctx context.Context) (*Quota, error) {
	if s.enforcer == nil {
		return &Quota{HasActiveLicense: false}, nil
	}
	return s.enforcer.Quota(ctx)
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
//
// EnforcementHits7d 在 logRepo 可用时填充近 7 天 result='denied' 计数；
// logRepo 未注入或查询失败均退化为 0（warn 日志），不阻断 Summary 主体。
func (s *Service) Summary(ctx context.Context) (*LicenseSummary, error) {
	summary, err := s.repo.Summary(ctx)
	if err != nil {
		return nil, err
	}
	if s.logRepo != nil {
		since := nowFunc().Add(-7 * 24 * time.Hour)
		hits, qErr := s.logRepo.CountDenialsSince(ctx, since)
		if qErr != nil {
			s.logger.Warn("count enforcement denials for summary failed (degraded to 0)",
				zap.Error(qErr))
		} else {
			summary.EnforcementHits7d = hits
		}
	}
	return summary, nil
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

	s.invalidateEnforcerCache()

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

	s.invalidateEnforcerCache()

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

	s.invalidateEnforcerCache()

	s.logger.Info("license imported",
		zap.String("license_id", lic.ID.String()),
		zap.String("license_code", lic.LicenseCode),
	)

	return lic, nil
}
