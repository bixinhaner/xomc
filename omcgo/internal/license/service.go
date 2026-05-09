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

// SameDimensionConflictError 表示同 (device_type, region) 维度已存在 active license。
//
// T-0100-P3 / PRD §5.3.2：业务约束规定同维度最多一个 active；Activate 检测到
// 冲突且 force=false 时返回此错误，handler 翻成 HTTP 409 + ConflictingLicenses
// 列表，前端弹 Modal 二次确认；force=true 再次调用进入自动 revoke + activate
// 串行流程。
type SameDimensionConflictError struct {
	DeviceType  *string
	Region      *string
	Conflicting []*License
}

// Error implements error.
func (e *SameDimensionConflictError) Error() string {
	return fmt.Sprintf(
		"same-dimension active license already exists (device_type=%s, region=%s, count=%d): %s",
		strPtrOrEmpty(e.DeviceType), strPtrOrEmpty(e.Region), len(e.Conflicting),
		commonerrors.ErrAlreadyExists.Error(),
	)
}

// Unwrap exposes ErrAlreadyExists so commonerrors.HTTPStatusFromError → 409
// （与"license_code 已存在"复用 409 语义；前端通过响应 body 的 code/conflict
// 字段区分两种冲突来源）。
func (e *SameDimensionConflictError) Unwrap() error { return commonerrors.ErrAlreadyExists }

func strPtrOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ActivateResult 是 Activate 执行结果，便于 handler 知道是否做了 auto-revoke
// 以便额外写一条 license_log（log_type=auto_revoke_by_activate）。
type ActivateResult struct {
	Activated *License // 新激活 license
	// AutoRevoked 列出本次 activate 自动 revoke 的同维度旧 license（可能多条）。
	// 仅 force=true 路径非空；handler 据此为每条写 auto_revoke_by_activate 日志。
	AutoRevoked []*License
}

// Activate finds a license by code and sets its status to active.
//
// force=false（默认）：检测同 (device_type, region) 维度是否已有 active；
// 若有，返回 SameDimensionConflictError → handler 翻 409 + 冲突列表，前端弹
// Modal 二次确认。若无冲突，直接激活。
//
// force=true：跳过冲突预检，先把同维度旧 active 全部 revoke（写日志的责任在
// handler，避免 service 层耦合 LogWriter），再激活目标 license。
func (s *Service) Activate(ctx context.Context, licenseCode string, force bool) (*ActivateResult, error) {
	lic, err := s.repo.GetByCode(ctx, licenseCode)
	if err != nil {
		return nil, fmt.Errorf("get license by code: %w", err)
	}

	if lic.Status == StatusActive {
		return &ActivateResult{Activated: lic}, nil
	}
	if lic.Status == StatusRevoked {
		return nil, commonerrors.NewBusinessError(9100, "cannot activate a revoked license", commonerrors.ErrInvalidInput)
	}

	conflicts, err := s.repo.ListActiveByDimension(ctx, lic.DeviceType, lic.Region)
	if err != nil {
		return nil, fmt.Errorf("list active by dimension: %w", err)
	}
	// 排除目标 license 自身（理论上 status != active 不会撞上，但保险一层）。
	conflicts = filterOutByID(conflicts, lic.ID)

	if len(conflicts) > 0 && !force {
		return nil, &SameDimensionConflictError{
			DeviceType:  lic.DeviceType,
			Region:      lic.Region,
			Conflicting: conflicts,
		}
	}

	autoRevoked := make([]*License, 0, len(conflicts))
	for _, old := range conflicts {
		old.Status = StatusRevoked
		if err := s.repo.Update(ctx, old); err != nil {
			return nil, fmt.Errorf("auto-revoke conflicting license %s: %w", old.ID, err)
		}
		s.logger.Info("license auto-revoked by activate",
			zap.String("revoked_id", old.ID.String()),
			zap.String("revoked_code", old.LicenseCode),
			zap.String("activated_code", licenseCode),
		)
		autoRevoked = append(autoRevoked, old)
	}

	lic.Status = StatusActive
	if err := s.repo.Update(ctx, lic); err != nil {
		return nil, fmt.Errorf("activate license: %w", err)
	}

	s.invalidateEnforcerCache()

	s.logger.Info("license activated",
		zap.String("license_id", lic.ID.String()),
		zap.String("license_code", licenseCode),
		zap.Int("auto_revoked_count", len(autoRevoked)),
	)

	return &ActivateResult{Activated: lic, AutoRevoked: autoRevoked}, nil
}

// filterOutByID drops any *License whose ID matches the given UUID.
func filterOutByID(in []*License, id uuid.UUID) []*License {
	out := make([]*License, 0, len(in))
	for _, lic := range in {
		if lic.ID != id {
			out = append(out, lic)
		}
	}
	return out
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
