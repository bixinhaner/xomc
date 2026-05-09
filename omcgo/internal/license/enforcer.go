// Package license — Enforcer
//
// Enforcer is the business-rule gate for license capacity and expiry.
// Callers (e.g. device.Service.CreateDevice) inject an Enforcer and call
// EnforceCapacity / EnforceExpiry before any state-mutating operation that
// counts toward the license quota.
//
// Design (T-0015 / R-103, see PRD F06-license-enforcement.md):
//
//   - Multi-active-license rule: pick the license with the largest
//     MaxDevices as the canonical enforcement license. Other actives are
//     ignored for enforcement decisions (they remain queryable).
//   - No active license => default-allow + warn log + metric=0.
//     dev-friendly; prod operators are expected to import a license.
//   - Permanent licenses (TypePerpetual) skip expiry checks unconditionally.
//   - Cached active license with 5min TTL; invalidated on Import/Activate/
//     Revoke via Service hook.
package license

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// Enforcer is the public contract for license enforcement. Consumers (e.g.
// device package) define their own narrower interface and accept this
// implementation; this avoids forcing every consumer to import the full
// license model.
type Enforcer interface {
	// EnforceCapacity returns ErrLicenseCapacityExceeded if adding `additional`
	// devices would cross MaxDevices. Returns nil when no active license
	// exists (default-allow).
	EnforceCapacity(ctx context.Context, additional int) error

	// EnforceExpiry returns ErrLicenseExpired if the active license is past
	// expiry + grace_period_days. Operation name is passed through for
	// metric labels and audit. Returns nil for perpetual licenses or when
	// no active license exists.
	EnforceExpiry(ctx context.Context, operation string) error

	// ActiveLicense returns the canonical enforcement license (max
	// MaxDevices among actives), or nil if none exist.
	ActiveLicense(ctx context.Context) (*License, error)

	// Quota returns the current enforcement state for /quota endpoint.
	Quota(ctx context.Context) (*Quota, error)

	// Invalidate clears the cached active license. Called by Service after
	// Import / Activate / Revoke to force re-read on next enforcement.
	Invalidate()
}

// DefaultCacheTTL is the active-license cache lifetime. Five minutes balances
// responsiveness (operators see new licenses promptly) against PG load.
const DefaultCacheTTL = 5 * time.Minute

// nowFunc is overridden in tests. Production always uses time.Now.
var nowFunc = time.Now

// EnforcerImpl is the concrete Enforcer.
type EnforcerImpl struct {
	repo      LicenseRepository
	logger    *zap.Logger
	metrics   *EnforcementMetrics
	logWriter LogWriter // T-0100-P0：拒绝事件审计；nil 时退化为 NoopLogWriter

	cacheMu      sync.RWMutex
	cachedActive *License
	cachedAt     time.Time
	cacheTTL     time.Duration
}

// NewEnforcer constructs an EnforcerImpl with the default 5min cache TTL.
// metrics may be nil (degrades to no-op recording).
func NewEnforcer(repo LicenseRepository, logger *zap.Logger, metrics *EnforcementMetrics) *EnforcerImpl {
	return &EnforcerImpl{
		repo:      repo,
		logger:    logger.Named("license-enforcer"),
		metrics:   metrics,
		logWriter: NoopLogWriter{}, // 默认 noop；DI 通过 SetLogWriter 注入真实实现
		cacheTTL:  DefaultCacheTTL,
	}
}

// SetLogWriter 注入真实的 LogWriter（T-0100-P0）。
// bootstrap 早期 / 测试不注入时保持 NoopLogWriter，写入路径无 nil 风险。
func (e *EnforcerImpl) SetLogWriter(w LogWriter) {
	if w == nil {
		w = NoopLogWriter{}
	}
	e.logWriter = w
}

// SetCacheTTL overrides the cache TTL. Test-only convenience.
func (e *EnforcerImpl) SetCacheTTL(ttl time.Duration) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	e.cacheTTL = ttl
}

// Invalidate clears the cached active license.
func (e *EnforcerImpl) Invalidate() {
	e.cacheMu.Lock()
	e.cachedActive = nil
	e.cachedAt = time.Time{}
	e.cacheMu.Unlock()
}

// ActiveLicense returns the cached active license, refreshing if stale.
// Returns nil if no active license exists (default-allow scenario).
func (e *EnforcerImpl) ActiveLicense(ctx context.Context) (*License, error) {
	e.cacheMu.RLock()
	if !e.cachedAt.IsZero() && nowFunc().Sub(e.cachedAt) < e.cacheTTL {
		lic := e.cachedActive
		e.cacheMu.RUnlock()
		return lic, nil
	}
	e.cacheMu.RUnlock()

	// Slow path: refresh.
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	// Re-check after acquiring write lock (another goroutine may have
	// refreshed concurrently).
	if !e.cachedAt.IsZero() && nowFunc().Sub(e.cachedAt) < e.cacheTTL {
		return e.cachedActive, nil
	}

	lic, err := e.repo.GetActiveLicenseWithMaxDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active license: %w", err)
	}
	e.cachedActive = lic
	e.cachedAt = nowFunc()
	return lic, nil
}

// EnforceCapacity rejects when current device count + additional exceeds
// MaxDevices. Returns nil if no active license (default-allow + log).
func (e *EnforcerImpl) EnforceCapacity(ctx context.Context, additional int) error {
	if additional < 0 {
		return fmt.Errorf("additional must be >= 0: %w", commonerrors.ErrInvalidInput)
	}

	lic, err := e.ActiveLicense(ctx)
	if err != nil {
		return err
	}
	if lic == nil {
		e.recordEnforcement("capacity", "no_active_license")
		e.logger.Warn("license enforcement skipped: no active license")
		return nil
	}

	used, err := e.repo.CountDevices(ctx)
	if err != nil {
		return fmt.Errorf("count devices for enforcement: %w", err)
	}

	if used+additional > lic.MaxDevices {
		e.recordEnforcement("capacity", "denied_capacity")
		e.logger.Warn("license capacity exceeded",
			zap.String("license_id", lic.ID.String()),
			zap.Int("max_devices", lic.MaxDevices),
			zap.Int("used_devices", used),
			zap.Int("additional", additional),
		)
		// 审计：拒绝事件落 license_logs（T-0100-P0 / R-109）
		licID := lic.ID
		e.logWriter.Write(ctx, LicenseLogEntry{
			LicenseID: &licID,
			LogType:   LogTypeEnforcementCapacity,
			Result:    LogResultDenied,
			Details: map[string]any{
				"summary":      "device.create denied: capacity exceeded",
				"max_devices":  lic.MaxDevices,
				"used_devices": used,
				"additional":   additional,
			},
		})
		return fmt.Errorf("used=%d, max=%d, additional=%d: %w",
			used, lic.MaxDevices, additional, commonerrors.ErrLicenseCapacityExceeded)
	}

	e.recordEnforcement("capacity", "allowed")
	return nil
}

// EnforceExpiry rejects when the active license is past expiry + grace.
// Permanent licenses always pass. No active license = default-allow.
func (e *EnforcerImpl) EnforceExpiry(ctx context.Context, operation string) error {
	lic, err := e.ActiveLicense(ctx)
	if err != nil {
		return err
	}
	if lic == nil {
		e.recordEnforcement(operation, "no_active_license")
		return nil
	}
	if lic.LicenseType == TypePerpetual {
		e.recordEnforcement(operation, "allowed")
		return nil
	}
	if lic.ExpiryDate == nil {
		// Subscription/trial without expiry date — treat as not-expired.
		e.recordEnforcement(operation, "allowed")
		return nil
	}

	graceDeadline := lic.ExpiryDate.Add(time.Duration(lic.GracePeriodDays) * 24 * time.Hour)
	if nowFunc().After(graceDeadline) || lic.Status == StatusExpired {
		e.recordEnforcement(operation, "denied_expired")
		e.logger.Warn("license expired blocking write operation",
			zap.String("license_id", lic.ID.String()),
			zap.String("operation", operation),
			zap.Time("expiry_date", *lic.ExpiryDate),
			zap.Int("grace_period_days", lic.GracePeriodDays),
		)
		// 审计：拒绝事件落 license_logs（T-0100-P0 / R-109）
		licID := lic.ID
		e.logWriter.Write(ctx, LicenseLogEntry{
			LicenseID: &licID,
			LogType:   LogTypeEnforcementExpiry,
			Result:    LogResultDenied,
			Details: map[string]any{
				"summary":           "write operation denied: license expired",
				"operation":         operation,
				"expiry_date":       lic.ExpiryDate.Format(time.RFC3339),
				"grace_period_days": lic.GracePeriodDays,
			},
		})
		return fmt.Errorf("license %s expired at %s (grace=%dd): %w",
			lic.ID, lic.ExpiryDate.Format(time.RFC3339), lic.GracePeriodDays,
			commonerrors.ErrLicenseExpired)
	}

	e.recordEnforcement(operation, "allowed")
	return nil
}

// Quota returns the current enforcement state. has_active_license=false
// when no active license exists.
func (e *EnforcerImpl) Quota(ctx context.Context) (*Quota, error) {
	lic, err := e.ActiveLicense(ctx)
	if err != nil {
		return nil, err
	}
	if lic == nil {
		return &Quota{HasActiveLicense: false}, nil
	}

	used, err := e.repo.CountDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("count devices for quota: %w", err)
	}

	q := &Quota{
		HasActiveLicense: true,
		MaxDevices:       lic.MaxDevices,
		UsedDevices:      used,
		LicenseType:      string(lic.LicenseType),
		GracePeriodDays:  lic.GracePeriodDays,
		DaysRemaining:    -1, // perpetual / no expiry
	}
	if lic.MaxDevices > 0 {
		q.UsageRatio = float64(used) / float64(lic.MaxDevices)
	}
	if lic.LicenseType != TypePerpetual && lic.ExpiryDate != nil {
		days := int(lic.ExpiryDate.Sub(nowFunc()).Hours() / 24)
		q.DaysRemaining = days
	}
	return q, nil
}

// recordEnforcement bumps the Prometheus counter when metrics are wired.
func (e *EnforcerImpl) recordEnforcement(operation, result string) {
	if e.metrics != nil {
		e.metrics.RecordEnforcement(operation, result)
	}
}

// Compile-time check that EnforcerImpl satisfies Enforcer.
var _ Enforcer = (*EnforcerImpl)(nil)

// EnforcementError reports whether an error is one of the enforcement
// sentinels (capacity exceeded or expired). Convenience for callers who
// want to log without coupling to commonerrors directly.
func EnforcementError(err error) (kind string, isEnforcement bool) {
	switch {
	case errors.Is(err, commonerrors.ErrLicenseCapacityExceeded):
		return "capacity_exceeded", true
	case errors.Is(err, commonerrors.ErrLicenseExpired):
		return "expired", true
	default:
		return "", false
	}
}
