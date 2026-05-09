// Package license — Monitor (cron-driven expiry + capacity checker)
//
// The Monitor runs three independent checks on a schedule:
//
//	daily 00:00 UTC  expiry checker  — flips licenses past expiry+grace to expired status
//	daily 01:00 UTC  expiring soon   — raises 30/7/1-day warnings for active subscription licenses
//	hourly 0 min     capacity check  — raises capacity threshold alerts (80/90/95%) with 6h dedup
//
// Alerts are dispatched via the AlertSink callback (defaults to no-op for
// dev/test). The Monitor never imports `internal/alarm` directly; the wiring
// happens in cmd/app/provider/modules.go via a sink adapter.
package license

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// AlertSeverity categorises a license alert. Severities map naturally to the
// alarm module's Warning/Major/Critical levels, but the mapping happens in
// the adapter so this package stays alarm-free.
type AlertSeverity string

const (
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityMajor    AlertSeverity = "major"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert describes a license event the Monitor wants to publish.
type Alert struct {
	LicenseID  uuid.UUID
	Identifier string // e.g. "license_expiring_30d", "license_capacity_80pct"
	Severity   AlertSeverity
	Summary    string
	Details    map[string]interface{}
}

// AlertSink delivers a license alert. Implementations should be non-blocking
// or short-running; the cron tick will block waiting on Send.
type AlertSink interface {
	Send(ctx context.Context, alert Alert) error
}

// NoopAlertSink discards alerts (useful for dev/test).
type NoopAlertSink struct{}

// Send implements AlertSink by doing nothing.
func (NoopAlertSink) Send(_ context.Context, _ Alert) error { return nil }

// CapacityDedupWindow is the minimum time between repeated alerts for the
// same threshold. Crossing into a new (higher) threshold short-circuits this.
const CapacityDedupWindow = 6 * time.Hour

// ExpiryWindowsDays defines the days-before-expiry breakpoints; ordered
// most-severe-first so the loop picks the smallest matching window (e.g.
// days=5 matches both {7} and {30} but should fire 7d-major).
var ExpiryWindowsDays = []struct {
	Days     int
	Severity AlertSeverity
	Suffix   string
}{
	{1, AlertSeverityCritical, "1d"},
	{7, AlertSeverityMajor, "7d"},
	{30, AlertSeverityWarning, "30d"},
}

// Monitor runs the periodic expiry+capacity checks.
type Monitor struct {
	repo             LicenseRepository
	sink             AlertSink
	metrics          *EnforcementMetrics
	logger           *zap.Logger
	logWriter        LogWriter // T-0100-P0：cron 触发的告警 / 自动过期审计
	archiver         *LogArchiver // T-0100-P4-B：license_logs 6 月归档；nil 时跳过
	archiveSchedule  string       // T-0100-P4-B：归档 cron expression；空时不注册

	cron   *cron.Cron
	cancel context.CancelFunc
}

// NewMonitor constructs a Monitor. sink defaults to NoopAlertSink when nil.
// metrics may be nil (no-op recording).
func NewMonitor(repo LicenseRepository, sink AlertSink, metrics *EnforcementMetrics, logger *zap.Logger) *Monitor {
	if sink == nil {
		sink = NoopAlertSink{}
	}
	return &Monitor{
		repo:      repo,
		sink:      sink,
		metrics:   metrics,
		logger:    logger.Named("license-monitor"),
		logWriter: NoopLogWriter{}, // 默认 noop；DI 通过 SetLogWriter 注入真实实现
	}
}

// SetLogWriter 注入真实的 LogWriter（T-0100-P0）。
// bootstrap 早期 / 测试不注入时保持 NoopLogWriter，写入路径无 nil 风险。
func (m *Monitor) SetLogWriter(w LogWriter) {
	if w == nil {
		w = NoopLogWriter{}
	}
	m.logWriter = w
}

// SetArchiver 注入 LogArchiver（T-0100-P4-B）。nil 等价于不注册归档 cron。
// schedule 为空时使用默认 "0 3 * * 0"（每周日 03:00 UTC）。
func (m *Monitor) SetArchiver(a *LogArchiver, schedule string) {
	m.archiver = a
	if schedule == "" {
		schedule = "0 3 * * 0"
	}
	m.archiveSchedule = schedule
}

// Start launches the cron schedule:
//
//   - daily 00:00 UTC  CheckExpiry
//   - daily 01:00 UTC  CheckExpiringSoon
//   - hourly @ 0 min   CheckCapacity
//
// The parent ctx scopes all check invocations; callers shut Monitor down
// via Stop() (typically wired to graceful shutdown).
func (m *Monitor) Start(ctx context.Context) error {
	scoped, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.cron = cron.New(cron.WithLocation(time.UTC))

	if _, err := m.cron.AddFunc("0 0 * * *", func() {
		c, c2 := context.WithTimeout(scoped, 5*time.Minute)
		defer c2()
		if err := m.CheckExpiry(c); err != nil {
			m.logger.Warn("daily expiry sweep failed", zap.Error(err))
		}
	}); err != nil {
		return fmt.Errorf("schedule expiry sweep: %w", err)
	}
	if _, err := m.cron.AddFunc("0 1 * * *", func() {
		c, c2 := context.WithTimeout(scoped, 5*time.Minute)
		defer c2()
		if err := m.CheckExpiringSoon(c); err != nil {
			m.logger.Warn("daily expiring-soon check failed", zap.Error(err))
		}
	}); err != nil {
		return fmt.Errorf("schedule expiring-soon: %w", err)
	}
	if _, err := m.cron.AddFunc("0 * * * *", func() {
		c, c2 := context.WithTimeout(scoped, 2*time.Minute)
		defer c2()
		if err := m.CheckCapacity(c); err != nil {
			m.logger.Warn("hourly capacity check failed", zap.Error(err))
		}
	}); err != nil {
		return fmt.Errorf("schedule capacity check: %w", err)
	}

	// T-0100-P4-B：周级归档 cron。仅在 SetArchiver 注入后注册；超时 30min（PRD
	// 没规定单 tick 上限，但批 10000 条 + gzip 压缩 + MinIO 上传 + DELETE 通常 <
	// 几分钟，30min 是 generous 上限）。
	if m.archiver != nil {
		if _, err := m.cron.AddFunc(m.archiveSchedule, func() {
			c, c2 := context.WithTimeout(scoped, 30*time.Minute)
			defer c2()
			if _, err := m.archiver.ArchiveOnce(c); err != nil {
				m.logger.Warn("license log archive run failed", zap.Error(err))
			}
		}); err != nil {
			return fmt.Errorf("schedule log archive: %w", err)
		}
	}

	m.cron.Start()
	logFields := []zap.Field{
		zap.String("expiry_schedule", "0 0 * * * UTC"),
		zap.String("expiring_soon_schedule", "0 1 * * * UTC"),
		zap.String("capacity_schedule", "0 * * * * (hourly)"),
	}
	if m.archiver != nil {
		logFields = append(logFields, zap.String("archive_schedule", m.archiveSchedule))
	}
	m.logger.Info("license monitor cron started", logFields...)
	return nil
}

// Stop gracefully shuts down the cron and cancels in-flight checks.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.cron != nil {
		m.cron.Stop()
	}
}

// CheckExpiry flips active licenses past expiry+grace to expired status.
// Idempotent: safe to call repeatedly.
func (m *Monitor) CheckExpiry(ctx context.Context) error {
	licenses, err := m.repo.ListActiveLicenses(ctx)
	if err != nil {
		return fmt.Errorf("list active licenses: %w", err)
	}
	now := nowFunc()
	expired := 0
	for _, lic := range licenses {
		if lic.LicenseType == TypePerpetual || lic.ExpiryDate == nil {
			continue
		}
		grace := lic.ExpiryDate.Add(time.Duration(lic.GracePeriodDays) * 24 * time.Hour)
		if now.After(grace) {
			if err := m.repo.MarkExpired(ctx, lic.ID); err != nil {
				m.logger.Error("mark expired failed",
					zap.String("license_id", lic.ID.String()),
					zap.Error(err))
				continue
			}
			expired++
			m.logger.Info("license marked expired",
				zap.String("license_id", lic.ID.String()),
				zap.Time("expiry_date", *lic.ExpiryDate),
				zap.Int("grace_period_days", lic.GracePeriodDays),
			)
			// 审计：自动过期事件落 license_logs（T-0100-P0 / R-109）
			licID := lic.ID
			m.logWriter.Write(ctx, LicenseLogEntry{
				LicenseID: &licID,
				LogType:   LogTypeAutoExpire,
				Result:    LogResultSuccess,
				Details: map[string]any{
					"summary":           "license auto-expired by daily cron",
					"license_code":      lic.LicenseCode,
					"expiry_date":       lic.ExpiryDate.Format(time.RFC3339),
					"grace_period_days": lic.GracePeriodDays,
				},
			})
		}
	}
	if expired > 0 {
		m.logger.Info("expiry sweep complete", zap.Int("expired_count", expired))
	}
	return nil
}

// CheckExpiringSoon raises 30/7/1-day warnings for active subscription
// licenses approaching expiry.
func (m *Monitor) CheckExpiringSoon(ctx context.Context) error {
	licenses, err := m.repo.ListActiveLicenses(ctx)
	if err != nil {
		return fmt.Errorf("list active licenses: %w", err)
	}
	now := nowFunc()
	for _, lic := range licenses {
		if lic.LicenseType == TypePerpetual || lic.ExpiryDate == nil {
			m.metrics.SetExpiryDaysRemaining(lic.ID.String(), -1)
			continue
		}
		days := int(lic.ExpiryDate.Sub(now).Hours() / 24)
		m.metrics.SetExpiryDaysRemaining(lic.ID.String(), days)

		// Find the matching window; only emit one alert per check, the
		// most-severe (smallest-day) window we cross.
		for _, w := range ExpiryWindowsDays {
			if days <= w.Days && days >= 0 {
				alert := Alert{
					LicenseID:  lic.ID,
					Identifier: "license_expiring_" + w.Suffix,
					Severity:   w.Severity,
					Summary:    fmt.Sprintf("License %s expires in %d days", lic.LicenseCode, days),
					Details: map[string]interface{}{
						"license_id":     lic.ID.String(),
						"license_code":   lic.LicenseCode,
						"expiry_date":    lic.ExpiryDate.Format(time.RFC3339),
						"days_remaining": days,
					},
				}
				if sendErr := m.sink.Send(ctx, alert); sendErr != nil {
					m.logger.Warn("send expiry alert failed",
						zap.String("license_id", lic.ID.String()),
						zap.Error(sendErr))
				}
				// 审计：过期阈值告警事件落 license_logs（T-0100-P0 / R-109）
				licID := lic.ID
				m.logWriter.Write(ctx, LicenseLogEntry{
					LicenseID: &licID,
					LogType:   LogTypeExpiryAlert,
					Result:    LogResultWarning,
					Details: map[string]any{
						"summary":        alert.Summary,
						"identifier":     alert.Identifier,
						"severity":       alert.Severity,
						"license_code":   lic.LicenseCode,
						"expiry_date":    lic.ExpiryDate.Format(time.RFC3339),
						"days_remaining": days,
					},
				})
				break
			}
		}
	}
	return nil
}

// CheckCapacity emits capacity-threshold alerts (80/90/95% by default).
//
// Dedup rule: skip if the same threshold was alerted within
// CapacityDedupWindow. A higher threshold crossing always fires immediately
// (it's a more urgent signal).
func (m *Monitor) CheckCapacity(ctx context.Context) error {
	lic, err := m.repo.GetActiveLicenseWithMaxDevices(ctx)
	if err != nil {
		return fmt.Errorf("get active license: %w", err)
	}
	if lic == nil {
		m.metrics.SetActiveCount(0)
		return nil
	}
	m.metrics.SetActiveCount(1)

	used, err := m.repo.CountDevices(ctx)
	if err != nil {
		return fmt.Errorf("count devices: %w", err)
	}

	var ratio float64
	if lic.MaxDevices > 0 {
		ratio = float64(used) / float64(lic.MaxDevices)
	}
	m.metrics.SetCapacity(used, lic.MaxDevices, ratio)

	thresholds := parseThresholds(lic.CapacityAlertThresholds)
	if len(thresholds) == 0 {
		return nil
	}
	// Sort ascending so we can find the highest threshold currently crossed.
	sort.Ints(thresholds)

	// Find the highest threshold the current usage has crossed (% int compare).
	usagePct := 0
	if lic.MaxDevices > 0 {
		usagePct = used * 100 / lic.MaxDevices
	}
	highestCrossed := -1
	for _, t := range thresholds {
		if usagePct >= t {
			highestCrossed = t
		}
	}
	if highestCrossed == -1 {
		return nil
	}

	// Dedup: same threshold within window? Skip.
	now := nowFunc()
	if lic.LastCapacityAlertAt != nil &&
		lic.LastCapacityAlertThreshold != nil &&
		*lic.LastCapacityAlertThreshold == highestCrossed &&
		now.Sub(*lic.LastCapacityAlertAt) < CapacityDedupWindow {
		return nil
	}

	severity := capacitySeverity(highestCrossed, thresholds)
	alert := Alert{
		LicenseID:  lic.ID,
		Identifier: fmt.Sprintf("license_capacity_%dpct", highestCrossed),
		Severity:   severity,
		Summary:    fmt.Sprintf("License capacity %d%% (%d/%d)", usagePct, used, lic.MaxDevices),
		Details: map[string]interface{}{
			"license_id":   lic.ID.String(),
			"used_devices": used,
			"max_devices":  lic.MaxDevices,
			"usage_ratio":  ratio,
			"threshold":    highestCrossed,
		},
	}
	if sendErr := m.sink.Send(ctx, alert); sendErr != nil {
		m.logger.Warn("send capacity alert failed",
			zap.String("license_id", lic.ID.String()),
			zap.Error(sendErr))
		// Persist anyway? No — re-emit on next tick if dispatch failed.
		return nil
	}

	// 审计：容量阈值告警事件落 license_logs（T-0100-P0 / R-109）
	licID := lic.ID
	m.logWriter.Write(ctx, LicenseLogEntry{
		LicenseID: &licID,
		LogType:   LogTypeCapacityAlert,
		Result:    LogResultWarning,
		Details: map[string]any{
			"summary":      alert.Summary,
			"identifier":   alert.Identifier,
			"severity":     alert.Severity,
			"used_devices": used,
			"max_devices":  lic.MaxDevices,
			"usage_ratio":  ratio,
			"threshold":    highestCrossed,
		},
	})

	if updErr := m.repo.UpdateCapacityAlert(ctx, lic.ID, highestCrossed, now); updErr != nil {
		m.logger.Warn("update capacity alert dedup state failed",
			zap.String("license_id", lic.ID.String()),
			zap.Error(updErr))
	}
	return nil
}

// parseThresholds extracts an ascending []int from JSON like "[80, 90, 95]".
// Returns nil on malformed input (caller treats as "no thresholds").
func parseThresholds(raw json.RawMessage) []int {
	if len(raw) == 0 {
		return nil
	}
	var out []int
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// capacitySeverity picks Warning/Major/Critical based on threshold position
// among the configured thresholds. Highest threshold => critical.
func capacitySeverity(threshold int, thresholds []int) AlertSeverity {
	// thresholds is sorted ascending by caller.
	if len(thresholds) == 0 {
		return AlertSeverityWarning
	}
	last := thresholds[len(thresholds)-1]
	switch {
	case threshold >= last:
		return AlertSeverityCritical
	case threshold >= last-10:
		return AlertSeverityMajor
	default:
		return AlertSeverityWarning
	}
}
