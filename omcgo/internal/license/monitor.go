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
//
// Clear 撤销此前用 Send 发出的、指定 identifier 的告警（条件解除时调用，
// 例如 license 重新上传后过期告警应消失）。实现应幂等：未找到匹配告警时
// 返回 nil 而非 error。
type AlertSink interface {
	Send(ctx context.Context, alert Alert) error
	Clear(ctx context.Context, identifier string) error
}

// NoopAlertSink discards alerts (useful for dev/test).
type NoopAlertSink struct{}

// Send implements AlertSink by doing nothing.
func (NoopAlertSink) Send(_ context.Context, _ Alert) error { return nil }

// Clear implements AlertSink by doing nothing.
func (NoopAlertSink) Clear(_ context.Context, _ string) error { return nil }

// CapacityDedupWindow is the minimum time between repeated alerts for the
// same threshold. Crossing into a new (higher) threshold short-circuits this.
const CapacityDedupWindow = 6 * time.Hour

// CapacityExhaustedIdentifier 是 OMC 告警库（data/alarm-definitions/OMC.xml）
// 中"License容量已满"告警的数字 identifier（跨 ne_type 全局唯一）。alarm engine
// 持久化后前端按 identifier JOIN alarm_definitions 取本地化名（cn_name/en_name）。
// enforcer 在 EnforceCapacity 因容量满拒绝设备时 Send（warning，1h 时间窗去重）；
// monitor 在所有授权类型都未满时 Clear。
const CapacityExhaustedIdentifier = "40"

// capacityExhaustedAlertDedup 是 enforcer 容量满告警的时间窗去重：1h 内不重发。
// monitor hourly Clear 后，若容量仍满，enforcer 最多 1h 后重新 Send。
const capacityExhaustedAlertDedup = 1 * time.Hour

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

	usageRepo SystemLicenseUsageRepository

	cron   *cron.Cron
	cancel context.CancelFunc

	// in-memory capacity dedup state（新模型无 last_capacity_alert_at 列，
	// 改为进程内 6h dedup；运维重启会重置，但 hourly tick 下一轮就重新触发，
	// 影响可接受）
	capacityDedupMu sync.Mutex
	lastCapAlertAt  time.Time
	lastCapAlertPct int

	// 告警恢复状态（进程内 best-effort）：记录当前是否有活跃告警，条件解除时
	// 调 sink.Clear 撤销。重启会丢失，但下一轮 tick 重新评估，最坏只多一次
	// 幂等 Clear 或漏清（与 dedup 状态同生命周期，可接受）。
	capAlertActive    bool // 受 capacityDedupMu 保护
	expiryAlertActive bool // 仅 CheckExpiringSoon 串行访问，无需额外锁

	cumulativeAlertActive bool // 仅 CheckCumulativeUsage 串行访问

	// exhaustedAlertActive 标记"容量满"告警是否活跃（受 capacityDedupMu 保护）。
	// enforcer 实时 raise（拒绝时），monitor hourly 兜底 raise + 负责恢复 clear；
	// 两者 Send 同 identifier，alarm engine 幂等。标志让 monitor 健康态不重复 Clear。
	exhaustedAlertActive bool
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

// SetUsageRepo 注入累计使用时长仓储，启用周期累计推进 + 超限告警。
// 不调用（nil）时跳过累计检查（兼容无 time_limit 的部署）。
func (m *Monitor) SetUsageRepo(r SystemLicenseUsageRepository) {
	m.usageRepo = r
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

	// 每 5 分钟推进累计使用时长，使 system_license_usage.total_used 保持当前，
	// 让页面/菜单的 is_expired 反映真实状态（enforcement 点自己的推进不受影响）。
	if m.usageRepo != nil {
		if _, err := m.cron.AddFunc("*/5 * * * *", func() {
			c, c2 := context.WithTimeout(scoped, time.Minute)
			defer c2()
			if err := m.CheckCumulativeUsage(c); err != nil {
				m.logger.Warn("cumulative usage check failed", zap.Error(err))
			}
		}); err != nil {
			return fmt.Errorf("schedule cumulative usage check: %w", err)
		}
	}

	m.cron.Start()
	m.logger.Info("license monitor cron started (system_license model)",
		zap.String("expiring_soon_schedule", "0 1 * * * UTC"),
		zap.String("capacity_schedule", "0 * * * * (hourly)"),
		zap.String("cumulative_schedule", "*/5 * * * * (every 5min)"),
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
		m.clearExpiryAlerts(ctx)
		return nil
	}
	now := nowFunc()
	days := int(lic.ExpiryDate.Sub(now).Hours() / 24)
	m.metrics.SetExpiryDaysRemaining("system", days)

	// 过期或在 30/7/1d 窗口内：发告警。已过期视为 1d critical。
	if days > 30 {
		m.clearExpiryAlerts(ctx)
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
	m.expiryAlertActive = true
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
	// 无论走哪个分支，退出前都评估"容量满"告警：检测到某授权类型已满则兜底
	// raise（enforcer 实时 raise 的补充），全部未满则 Clear。hourly 频率可接受。
	defer m.evaluateCapacityExhausted(ctx, lic)
	if lic == nil {
		m.metrics.SetCapacity(0, 0, 0)
		m.clearCapacityAlerts(ctx)
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
		m.clearCapacityAlerts(ctx)
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
		m.clearCapacityAlerts(ctx)
		return nil
	}

	// 6h dedup（进程内）：相同档位 6h 内不重复发；跨档位短路重发。
	now := nowFunc()
	m.capacityDedupMu.Lock()
	if m.lastCapAlertPct == highestCrossed &&
		!m.lastCapAlertAt.IsZero() &&
		now.Sub(m.lastCapAlertAt) < CapacityDedupWindow {
		// dedup 抑制重发，但告警仍活跃（保持 capAlertActive=true）。
		m.capAlertActive = true
		m.capacityDedupMu.Unlock()
		return nil
	}
	m.lastCapAlertPct = highestCrossed
	m.lastCapAlertAt = now
	m.capAlertActive = true
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

// clearExpiryAlerts 在过期告警条件解除时（license 有效 / 永久 / 无 license）
// 撤销此前发出的所有过期告警。expiration 窗口逐级收紧（30d→7d→1d）会产生
// 不同 identifier 的告警，故遍历所有窗口 suffix 一并清；Clear 幂等，未匹配
// 返 nil。仅在 expiryAlertActive=true 时调用，避免健康态每轮空跑 DB。
func (m *Monitor) clearExpiryAlerts(ctx context.Context) {
	if !m.expiryAlertActive {
		return
	}
	for _, w := range ExpiryWindowsDays {
		identifier := "license_expiring_" + w.Suffix
		if err := m.sink.Clear(ctx, identifier); err != nil {
			m.logger.Warn("clear license expiry alert failed",
				zap.String("identifier", identifier), zap.Error(err))
		}
	}
	m.logger.Info("license expiry alerts cleared on recovery")
	m.expiryAlertActive = false
}

// clearCapacityAlerts 在容量告警条件解除时（低于阈值 / 无 license / 无容量）
// 撤销此前发出的所有容量告警。跨档位升级（80→90→95）会产生多个 identifier，
// 故遍历所有阈值一并清。仅在 capAlertActive=true 时调用。
func (m *Monitor) clearCapacityAlerts(ctx context.Context) {
	m.capacityDedupMu.Lock()
	if !m.capAlertActive {
		m.capacityDedupMu.Unlock()
		return
	}
	m.capAlertActive = false
	m.lastCapAlertPct = 0
	m.lastCapAlertAt = time.Time{}
	m.capacityDedupMu.Unlock()

	for _, pct := range DefaultCapacityThresholds {
		identifier := fmt.Sprintf("license_capacity_%dpct", pct)
		if err := m.sink.Clear(ctx, identifier); err != nil {
			m.logger.Warn("clear license capacity alert failed",
				zap.String("identifier", identifier), zap.Error(err))
		}
	}
	m.logger.Info("license capacity alerts cleared on recovery")
}

// evaluateCapacityExhausted 评估 per-type 容量满告警的 raise/clear，与 enforcer
// 互补：enforcer 在 EnforceCapacity 拒绝时实时 raise（1h 去重），本方法 hourly
// 兜底 raise（覆盖 enforcer 去重空窗）+ 负责恢复 clear。两者 Send 同 identifier，
// alarm engine 按 (DeviceSN, AlarmIdentifier) 幂等去重。exhaustedAlertActive 标志
// 保证健康态不重复 Clear。
func (m *Monitor) evaluateCapacityExhausted(ctx context.Context, lic *SystemLicense) {
	if lic == nil || len(lic.DevicesSupport) == 0 {
		m.clearExhaustedAlert(ctx)
		return
	}
	usedByType, err := m.devices.CountDevicesByType(ctx)
	if err != nil {
		m.logger.Warn("count devices by type for exhausted check failed", zap.Error(err))
		return
	}
	anyExhausted := false
	var sampleType string
	for t, max := range lic.DevicesSupport {
		// 容量分组（issue #318）：用量按组合计（eNB 配额的用量含 GSM 设备）。
		if max > 0 && capacityGroupUsage(usedByType, upperKey(t)) >= max {
			anyExhausted = true
			sampleType = t
			break
		}
	}

	m.capacityDedupMu.Lock()
	active := m.exhaustedAlertActive
	m.capacityDedupMu.Unlock()

	if anyExhausted {
		if active {
			return // 已活跃，不重复 raise
		}
		alert := Alert{
			LicenseID:  lic.ID,
			Identifier: CapacityExhaustedIdentifier,
			Severity:   AlertSeverityWarning,
			Summary:    fmt.Sprintf("License capacity exhausted for %s, new device rejected", sampleType),
			Details: map[string]interface{}{
				"license_id":      lic.LicenseID,
				"device_type":     sampleType,
				"devices_support": lic.DevicesSupport,
			},
		}
		if err := m.sink.Send(ctx, alert); err != nil {
			m.logger.Warn("send capacity-exhausted alert failed", zap.Error(err))
			return
		}
		m.capacityDedupMu.Lock()
		m.exhaustedAlertActive = true
		m.capacityDedupMu.Unlock()
		m.logger.Warn("license capacity exhausted",
			zap.String("audit", "capacity_alert"),
			zap.String("identifier", CapacityExhaustedIdentifier),
			zap.String("device_type", sampleType))
		return
	}
	m.clearExhaustedAlert(ctx)
}

// clearExhaustedAlert 撤销容量满告警（仅在 exhaustedAlertActive=true 时）。
func (m *Monitor) clearExhaustedAlert(ctx context.Context) {
	m.capacityDedupMu.Lock()
	if !m.exhaustedAlertActive {
		m.capacityDedupMu.Unlock()
		return
	}
	m.exhaustedAlertActive = false
	m.capacityDedupMu.Unlock()

	if err := m.sink.Clear(ctx, CapacityExhaustedIdentifier); err != nil {
		m.logger.Warn("clear capacity exhausted alert failed", zap.Error(err))
		return
	}
	m.logger.Info("license capacity exhausted alert cleared (all authorized types recovered)")
}

// CheckCumulativeUsage 每 5min 推进累计使用时长并检查是否超限。
//
// 设计要点：
//   - 推进逻辑与 enforcer.enforceCumulative 一致（按 now - last_visited 累加），
//     但不拒绝业务——仅推进 + 告警。enforcement 点（device.create/inform）仍有
//     自己的即时推进 + 拒绝。
//   - 推进使 system_license_usage.total_used 保持当前，页面/菜单据此判断 is_expired。
//   - 无 license / time_limit_hours<=0 → noop（同时清历史告警）。
//   - 时间回拨 → 发 critical 告警，不推进（避免负值污染累计）。
func (m *Monitor) CheckCumulativeUsage(ctx context.Context) error {
	if m.usageRepo == nil {
		return nil
	}
	lic, err := m.getCurrentOrNil(ctx)
	if err != nil {
		return err
	}
	if lic == nil {
		m.clearCumulativeAlerts(ctx)
		return nil
	}
	timeLimit := extractTimeLimitHours(lic.FeatureList)
	if timeLimit <= 0 {
		m.clearCumulativeAlerts(ctx)
		return nil
	}

	lastVisited, err := m.usageRepo.LastVisited(ctx)
	if err != nil {
		m.logger.Warn("cumulative usage load failed, skip", zap.Error(err))
		return nil
	}
	now := nowFunc()
	elapsed := now.Sub(lastVisited)
	if elapsed < 0 {
		if !m.cumulativeAlertActive {
			alert := Alert{
				LicenseID:  lic.ID,
				Identifier: "license_cumulative_rollback",
				Severity:   AlertSeverityCritical,
				Summary:    "System license time rollback detected",
				Details: map[string]interface{}{
					"license_id":   lic.LicenseID,
					"now":          now.UTC().Format(time.RFC3339),
					"last_visited": lastVisited.UTC().Format(time.RFC3339),
				},
			}
			m.sink.Send(ctx, alert)
			m.cumulativeAlertActive = true
		}
		m.logger.Warn("license time rollback detected during cumulative check",
			zap.Time("now", now), zap.Time("last_visited", lastVisited))
		return nil
	}

	total, err := m.usageRepo.Advance(ctx, elapsed.Hours())
	if err != nil {
		m.logger.Warn("cumulative usage advance failed, skip", zap.Error(err))
		return nil
	}

	if total >= float64(timeLimit) {
		if !m.cumulativeAlertActive {
			alert := Alert{
				LicenseID:  lic.ID,
				Identifier: "license_cumulative_exceeded",
				Severity:   AlertSeverityCritical,
				Summary:    fmt.Sprintf("System license cumulative usage %.1fh >= limit %dh", total, timeLimit),
				Details: map[string]interface{}{
					"license_id":  lic.LicenseID,
					"used_hours":  total,
					"limit_hours": timeLimit,
				},
			}
			m.sink.Send(ctx, alert)
			m.cumulativeAlertActive = true
		}
		m.logger.Warn("license cumulative usage exceeded",
			zap.Float64("used_hours", total), zap.Int("limit_hours", timeLimit))
	} else {
		m.clearCumulativeAlerts(ctx)
	}
	return nil
}

// clearCumulativeAlerts 撤销累计超限/回拨告警（恢复或换 license 后调用）。
func (m *Monitor) clearCumulativeAlerts(ctx context.Context) {
	if !m.cumulativeAlertActive {
		return
	}
	for _, id := range []string{"license_cumulative_exceeded", "license_cumulative_rollback"} {
		if err := m.sink.Clear(ctx, id); err != nil {
			m.logger.Warn("clear cumulative alert failed", zap.String("identifier", id), zap.Error(err))
		}
	}
	m.cumulativeAlertActive = false
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
