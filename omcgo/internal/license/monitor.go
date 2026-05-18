// Package license — Monitor (cron-driven expiry + capacity checker)
//
// F06 System License 重构 Step 3 起：监视器从扫多张老 licenses 行改为扫
// system_license 唯一一行。
//
// 调度（保持与老 monitor 一致，便于运维认知不变）：
//
//	daily 00:00 UTC  expiring soon  — 30/7/1d 过期阈值告警（已过期同 1d critical）
//	hourly 0 min     capacity check — 80/90/95% 容量阈值告警，6h dedup
//
// 老 daily-01:00 expiry-sweep（把过期 license 翻状态）在新模型不再需要：
// system_license 没有"status"字段，过期 = expiry_date < now() 直接判断，
// 翻转动作纯冗余。已移除。
//
// Alerts 通过 AlertSink 派发（dev/test 默认 NoopAlertSink）；Monitor 不直接
// 引用 internal/alarm，wiring 在 cmd/app/provider/modules.go 走 adapter。
package license

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
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
//
// 新模型 system_license 主键是 UUID，告警里 LicenseID 用 SystemLicense.ID。
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

// DefaultCapacityThresholds — system_license 不再像老 licenses 表那样按行
// 配置阈值，统一用 80/90/95%（PRD F06-license-enforcement 留存的运维约定）。
var DefaultCapacityThresholds = []int{80, 90, 95}

// Monitor runs the periodic expiry+capacity checks.
//
// Step 5 起：老 license_logs 表与 LogArchiver 已删除；告警事件改纯 zap.Warn +
// AlertSink（通过 adapter 路由到 alarm 模块）。
type Monitor struct {
	repo    SystemLicenseRepository
	devices DeviceCounter
	sink    AlertSink
	metrics *EnforcementMetrics
	logger  *zap.Logger

	cron   *cron.Cron
	cancel context.CancelFunc

	// in-memory capacity dedup state（新模型无 last_capacity_alert_at 列，
	// 改为进程内 6h dedup；运维重启会重置，但 hourly tick 下一轮就重新触发，
	// 影响可接受）
	capacityDedupMu sync.Mutex
	lastCapAlertAt  time.Time
	lastCapAlertPct int
}

// NewMonitor constructs a Monitor. sink defaults to NoopAlertSink when nil.
// metrics may be nil (no-op recording).
//
// repo 是 SystemLicenseRepository；devices 是独立的 CountDevices 提供方。
func NewMonitor(repo SystemLicenseRepository, devices DeviceCounter, sink AlertSink, metrics *EnforcementMetrics, logger *zap.Logger) *Monitor {
	if sink == nil {
		sink = NoopAlertSink{}
	}
	return &Monitor{
		repo:    repo,
		devices: devices,
		sink:    sink,
		metrics: metrics,
		logger:  logger.Named("license-monitor"),
	}
}

// Start launches the cron schedule:
//
//   - daily 01:00 UTC  CheckExpiringSoon
//   - hourly @ 0 min   CheckCapacity
//   - weekly (optional) license_logs archive
func (m *Monitor) Start(ctx context.Context) error {
	scoped, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.cron = cron.New(cron.WithLocation(time.UTC))

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

	m.cron.Start()
	m.logger.Info("license monitor cron started (system_license model)",
		zap.String("expiring_soon_schedule", "0 1 * * * UTC"),
		zap.String("capacity_schedule", "0 * * * * (hourly)"),
	)
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

// CheckExpiringSoon 检查当前 system_license 是否进入 30/7/1d 过期窗口；
// 已过期（days < 0）同样触发 1d critical 告警。
//
// 无 license（表空）→ noop（保留 expiry_days_remaining=-1 指标）。
// expiry_date 为 NULL（永久）→ noop。
func (m *Monitor) CheckExpiringSoon(ctx context.Context) error {
	lic, err := m.getCurrentOrNil(ctx)
	if err != nil {
		return err
	}
	if lic == nil || lic.ExpiryDate == nil {
		m.metrics.SetExpiryDaysRemaining("system", -1)
		return nil
	}
	now := nowFunc()
	days := int(lic.ExpiryDate.Sub(now).Hours() / 24)
	m.metrics.SetExpiryDaysRemaining("system", days)

	// 过期或在 30/7/1d 窗口内：发告警。已过期视为 1d critical。
	if days > 30 {
		return nil
	}
	severity := AlertSeverityCritical
	suffix := "1d"
	for _, w := range ExpiryWindowsDays {
		if days <= w.Days {
			severity = w.Severity
			suffix = w.Suffix
			break
		}
	}

	alert := Alert{
		LicenseID:  lic.ID,
		Identifier: "license_expiring_" + suffix,
		Severity:   severity,
		Summary:    fmt.Sprintf("System license %s expires in %d days", lic.LicenseID, days),
		Details: map[string]interface{}{
			"license_id":     lic.LicenseID,
			"license_pk":     lic.ID.String(),
			"expiry_date":    lic.ExpiryDate.Format(time.RFC3339),
			"days_remaining": days,
		},
	}
	if sendErr := m.sink.Send(ctx, alert); sendErr != nil {
		m.logger.Warn("send expiry alert failed",
			zap.String("license_id", lic.LicenseID),
			zap.Error(sendErr))
	}
	m.logger.Warn("license expiring",
		zap.String("audit", "expiry_alert"),
		zap.String("identifier", alert.Identifier),
		zap.String("severity", string(alert.Severity)),
		zap.String("license_id", lic.LicenseID),
		zap.Int("days_remaining", days),
		zap.Time("expiry_date", *lic.ExpiryDate),
	)
	return nil
}

// CheckCapacity 容量阈值告警（80/90/95% 默认）+ 6h dedup。
//
// 无 license → SetCapacity(0,0,0) + noop（max=0 暗示 unprotected mode）。
func (m *Monitor) CheckCapacity(ctx context.Context) error {
	lic, err := m.getCurrentOrNil(ctx)
	if err != nil {
		return err
	}
	if lic == nil {
		m.metrics.SetCapacity(0, 0, 0)
		return nil
	}

	used, err := m.devices.CountDevices(ctx)
	if err != nil {
		return fmt.Errorf("count devices: %w", err)
	}

	maxDevices := totalCapacity(lic)
	var ratio float64
	if maxDevices > 0 {
		ratio = float64(used) / float64(maxDevices)
	}
	m.metrics.SetCapacity(used, maxDevices, ratio)

	if maxDevices == 0 {
		return nil
	}

	// 按 DefaultCapacityThresholds（80/90/95）寻最高已穿越档位。
	thresholds := append([]int(nil), DefaultCapacityThresholds...)
	sort.Ints(thresholds)
	usagePct := used * 100 / maxDevices
	highestCrossed := -1
	for _, t := range thresholds {
		if usagePct >= t {
			highestCrossed = t
		}
	}
	if highestCrossed == -1 {
		return nil
	}

	// 6h dedup（进程内）：相同档位 6h 内不重复发；跨档位短路重发。
	now := nowFunc()
	m.capacityDedupMu.Lock()
	if m.lastCapAlertPct == highestCrossed &&
		!m.lastCapAlertAt.IsZero() &&
		now.Sub(m.lastCapAlertAt) < CapacityDedupWindow {
		m.capacityDedupMu.Unlock()
		return nil
	}
	m.lastCapAlertPct = highestCrossed
	m.lastCapAlertAt = now
	m.capacityDedupMu.Unlock()

	severity := capacitySeverity(highestCrossed, thresholds)
	alert := Alert{
		LicenseID:  lic.ID,
		Identifier: fmt.Sprintf("license_capacity_%dpct", highestCrossed),
		Severity:   severity,
		Summary:    fmt.Sprintf("System license capacity %d%% (%d/%d)", usagePct, used, maxDevices),
		Details: map[string]interface{}{
			"license_id":   lic.LicenseID,
			"license_pk":   lic.ID.String(),
			"used_devices": used,
			"max_devices":  maxDevices,
			"usage_ratio":  ratio,
			"threshold":    highestCrossed,
		},
	}
	if sendErr := m.sink.Send(ctx, alert); sendErr != nil {
		m.logger.Warn("send capacity alert failed",
			zap.String("license_id", lic.LicenseID),
			zap.Error(sendErr))
		return nil
	}

	m.logger.Warn("license capacity threshold crossed",
		zap.String("audit", "capacity_alert"),
		zap.String("identifier", alert.Identifier),
		zap.String("severity", string(alert.Severity)),
		zap.String("license_id", lic.LicenseID),
		zap.Int("used_devices", used),
		zap.Int("max_devices", maxDevices),
		zap.Float64("usage_ratio", ratio),
		zap.Int("threshold", highestCrossed),
	)
	return nil
}

// getCurrentOrNil 拉当前 system_license；表空返 (nil, nil)；其它错误透传。
func (m *Monitor) getCurrentOrNil(ctx context.Context) (*SystemLicense, error) {
	lic, err := m.repo.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, ErrSystemLicenseNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get current system license: %w", err)
	}
	return lic, nil
}

// capacitySeverity picks Warning/Major/Critical based on threshold position
// among the configured thresholds. Highest threshold => critical.
func capacitySeverity(threshold int, thresholds []int) AlertSeverity {
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
