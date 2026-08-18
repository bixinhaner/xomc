// Package license — Enforcer
//
// Enforcer is the business-rule gate for license capacity and expiry.
// Callers (e.g. device.Service.CreateDevice) inject an Enforcer and call
// EnforceCapacity / EnforceExpiry before any state-mutating operation that
// counts toward the license quota.
//
// F06 System License 重构（PRD F06-system-license-redesign, Step 3）：
// 内部已由老 multi-license 模型切到 singleton system_license 模型。
//
//   - 数据源：`SystemLicenseRepository.GetCurrent` — 全系统唯一 license 行；
//     空表 = fail-closed：受控业务返回 ErrLicenseUnavailable（对齐旧项目"无 License
//     安全拒绝"语义；进程不退出，仅受控业务全部拒绝）
//   - 容量：`SystemLicense.DevicesSupport` 是 map[device_type]int；
//     `EnforceCapacity(ctx, deviceType, additional)` 按**网元类型独立限额** ——
//     license 未显式授权该类型（map 查不到或配额<=0）一律拒绝（严格模式，
//     对齐 issue #316 "各网元最大接入设备数必须分别生效"）。大小写不敏感
//     匹配（license key `eNB`/`gNB` ↔ 设备 ne_type `ENB`/`GNB`）。例外是
//     容量分组（issue #318）：GSM 网元（2G BSC/BTS）与 eNB 共用容量，
//     见 capacity_group.go。
//   - 过期：`SystemLicense.ExpiryDate` 直接判断（新模型无 grace_period_days
//     / 无 perpetual 类型；NULL expiry_date 视为永不过期，对应老 perpetual 语义）
//   - 缓存：5min TTL，Replace（POST /system-license）后由 SystemLicenseService
//     调 Invalidate 显式失效
package license

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// Enforcer is the public contract for license enforcement. Consumers (e.g.
// device package) define their own narrower interface and accept this
// implementation; this avoids forcing every consumer to import the full
// license model.
//
// Step 3：接口签名 EnforceExpiry 完全保留向后兼容；EnforceCapacity 已升级为
// per-type gating（新增 deviceType 入参），device.LicenseEnforcer 同步更新。
type Enforcer interface {
	// EnforceCapacity returns ErrLicenseCapacityExceeded if adding `additional`
	// devices of `deviceType` would cross that device_type's quota in the
	// system-license DevicesSupport map. Returns ErrLicenseCapacityExceeded
	// (wrapped, "not authorized") when the license does not authorize the
	// given deviceType, and ErrLicenseUnavailable when no license is
	// configured (fail-closed). deviceType=="" rejects (unknown NE type).
	EnforceCapacity(ctx context.Context, deviceType string, additional int) error

	// EnforceExpiry returns ErrLicenseExpired if the current license is past
	// expiry_date, or ErrLicenseUnavailable when no license is configured
	// (fail-closed). Operation name is passed through for metric labels +
	// audit. Returns nil only when a license exists and expiry_date is NULL
	// (perpetual) or in the future.
	EnforceExpiry(ctx context.Context, operation string) error

	// ActiveLicense returns the current system license, or nil if none
	// configured (EnforceCapacity/EnforceExpiry translate nil into
	// ErrLicenseUnavailable — fail-closed).
	//
	// 注意：返回类型由老 License 改为 SystemLicense（Step 3 breaking 变更，
	// 但唯一 caller 是 enforcer 自身 / handler / test，已同步更新）。
	ActiveLicense(ctx context.Context) (*SystemLicense, error)

	// Quota returns the current enforcement state for /quota endpoint.
	Quota(ctx context.Context) (*Quota, error)

	// Invalidate clears the cached system license. Called by SystemLicenseService
	// after Update（POST /system-license）to force re-read on next enforcement.
	Invalidate()
}

// DefaultCacheTTL is the system-license cache lifetime. Five minutes balances
// responsiveness (operators see new license promptly) against PG load.
const DefaultCacheTTL = 5 * time.Minute

// nowFunc is overridden in tests. Production always uses time.Now.
var nowFunc = time.Now

// DeviceCounter abstracts "how many registered devices exist". Defined at the
// enforcer consumer side so enforcer doesn't import the device package
// directly (avoids module cycle). Satisfied by PgLicenseRepository.CountDevices.
type DeviceCounter interface {
	// CountDevices 返回已纳管设备总数（不分类型），用于 monitor 容量阈值告警
	// 与 Quota 的总量展示。
	CountDevices(ctx context.Context) (int, error)
	// CountDevicesByType 按网元类型（products.alarm_ne_type，UPPER 归一化）分组
	// 计数，用于 EnforceCapacity 的 per-type 限额。key 为大写 ne_type。
	CountDevicesByType(ctx context.Context) (map[string]int, error)
}

// EnforcerImpl is the concrete Enforcer.
//
// Step 5 起：老 license_logs 表与 LogWriter 已删除；拒绝事件改纯 zap.Warn +
// Prometheus counter（recordEnforcement(…, "denied_capacity"/"denied_expired")）。
// 合规审计在生产部署里由 Loki/journald 抓 stdout 日志归档。
type EnforcerImpl struct {
	repo      SystemLicenseRepository
	devices   DeviceCounter
	logger    *zap.Logger
	metrics   *EnforcementMetrics
	usageRepo SystemLicenseUsageRepository
	alertSink AlertSink

	cacheMu      sync.RWMutex
	cachedActive *SystemLicense
	cachedAt     time.Time
	cacheTTL     time.Duration

	// capacityExhaustedAlert 去重（进程内时间窗）：避免每次 periodic Inform 拒绝
	// 都 Send 告警。Clear 由 monitor 负责（所有授权类型未满时），两者无状态共享，
	// 故用时间窗而非活跃标记——monitor Clear 后若仍满，最多 capacityExhaustedAlertDedup
	// 后重发。
	exhaustedAlertMu     sync.Mutex
	lastExhaustedAlertAt time.Time
}

// NewEnforcer constructs an EnforcerImpl with the default 5min cache TTL.
// metrics may be nil (degrades to no-op recording).
//
// repo 是 SystemLicenseRepository（singleton 模型）；devices 是独立的 CountDevices
// 提供方（DeviceCounter）— 与 license 表完全无关，由 caller wiring 自由选择
// （Step 5 起注入 device 模块的实现）。
func NewEnforcer(repo SystemLicenseRepository, devices DeviceCounter, logger *zap.Logger, metrics *EnforcementMetrics) *EnforcerImpl {
	return &EnforcerImpl{
		repo:      repo,
		devices:   devices,
		logger:    logger.Named("license-enforcer"),
		metrics:   metrics,
		alertSink: NoopAlertSink{},
		cacheTTL:  DefaultCacheTTL,
	}
}

// SetCacheTTL overrides the cache TTL. Test-only convenience.
func (e *EnforcerImpl) SetCacheTTL(ttl time.Duration) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	e.cacheTTL = ttl
}

// SetUsageRepo 注入累计使用时长仓储，启用累计时长 + 时间回拨检测。
// 不调用（nil）时 EnforceExpiry 只做日期过期校验（兼容旧测试/轻量部署）。
func (e *EnforcerImpl) SetUsageRepo(r SystemLicenseUsageRepository) {
	e.usageRepo = r
}

// SetAlertSink 注入告警 sink，启用容量满告警（warning）。
// 不调用（nil）时退化为 NoopAlertSink（不发告警，仅 zap + 指标）。
func (e *EnforcerImpl) SetAlertSink(sink AlertSink) {
	if sink == nil {
		sink = NoopAlertSink{}
	}
	e.alertSink = sink
}

// Invalidate clears the cached system license.
func (e *EnforcerImpl) Invalidate() {
	e.cacheMu.Lock()
	e.cachedActive = nil
	e.cachedAt = time.Time{}
	e.cacheMu.Unlock()
}

// ActiveLicense returns the cached current system license, refreshing if stale.
// Returns nil if no license is configured (callers EnforceCapacity/EnforceExpiry
// translate nil into ErrLicenseUnavailable — fail-closed).
func (e *EnforcerImpl) ActiveLicense(ctx context.Context) (*SystemLicense, error) {
	e.cacheMu.RLock()
	if !e.cachedAt.IsZero() && nowFunc().Sub(e.cachedAt) < e.cacheTTL {
		lic := e.cachedActive
		e.cacheMu.RUnlock()
		return lic, nil
	}
	e.cacheMu.RUnlock()

	// Slow path: refresh under write lock.
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	// Re-check after acquiring write lock (another goroutine may have
	// refreshed concurrently).
	if !e.cachedAt.IsZero() && nowFunc().Sub(e.cachedAt) < e.cacheTTL {
		return e.cachedActive, nil
	}

	lic, err := e.repo.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, ErrSystemLicenseNotFound) {
			// 空表 = fail-closed：缓存 nil 避免每次 Enforce 都打 DB；EnforceCapacity/
			// EnforceExpiry 会把 nil 翻译成 ErrLicenseUnavailable。
			e.cachedActive = nil
			e.cachedAt = nowFunc()
			return nil, nil
		}
		return nil, fmt.Errorf("load current system license: %w", err)
	}
	e.cachedActive = lic
	e.cachedAt = nowFunc()
	return lic, nil
}

// totalCapacity 返回 license.DevicesSupport map 所有 type 容量之和。
//
// 仅用于 Quota 端点的总量展示与 monitor 告警阈值；EnforceCapacity 走 per-type
// 独立限额（见 lookupCapacity），不再用总和做拦截判断。
func totalCapacity(lic *SystemLicense) int {
	if lic == nil {
		return 0
	}
	total := 0
	for _, v := range lic.DevicesSupport {
		if v > 0 {
			total += v
		}
	}
	return total
}

// upperKey 把网元类型归一化为大写键，用于 license 配额与设备 ne_type 的
// 大小写不敏感匹配（license key 形如 "eNB"/"gNB"，设备 alarm_ne_type 形如
// "ENB"/"GNB"）。
func upperKey(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// lookupCapacity 在 license.DevicesSupport 中大小写不敏感地查找 deviceType 的
// 配额。返回 (quota, true) 表示该类型被显式授权；返回 (0, false) 表示未授权。
func lookupCapacity(ds DevicesSupport, deviceType string) (int, bool) {
	target := upperKey(deviceType)
	if target == "" {
		return 0, false
	}
	for k, v := range ds {
		if upperKey(k) == target {
			return v, true
		}
	}
	return 0, false
}

// EnforceCapacity rejects (per-type gating) when:
//   - deviceType=="" → unknown NE type (product not registered): reject;
//   - no license configured → fail-closed (ErrLicenseUnavailable);
//   - license does not authorize deviceType's capacity group (absent from
//     DevicesSupport or quota<=0) → reject (strict, issue #316);
//   - current count of the capacity group + additional exceeds the group's
//     quota → reject.
//
// 容量分组（issue #318）：GSM 归入 eNB 容量组——授权与用量都看 eNB 配额，
// 组内 eNB+GSM 在线数合计控制；其余类型各占独立容量组。
func (e *EnforcerImpl) EnforceCapacity(ctx context.Context, deviceType string, additional int) error {
	if additional < 0 {
		return fmt.Errorf("additional must be >= 0: %w", commonerrors.ErrInvalidInput)
	}

	target := upperKey(deviceType)
	if target == "" {
		e.recordEnforcement("capacity", "denied_unknown_type")
		e.logger.Warn("license capacity denied: device network-element type unknown (product not registered)",
			zap.String("audit", "enforcement_capacity"),
		)
		return fmt.Errorf("device ne_type is empty (product not registered): %w", commonerrors.ErrLicenseCapacityExceeded)
	}

	lic, err := e.ActiveLicense(ctx)
	if err != nil {
		return err
	}
	if lic == nil {
		e.recordEnforcement("capacity", "denied_no_license")
		e.logger.Warn("license capacity enforcement denied: no system license configured",
			zap.String("audit", "enforcement_capacity"),
		)
		return fmt.Errorf("no system license configured: %w", commonerrors.ErrLicenseUnavailable)
	}

	quotaKey := capacityGroupKey(target)
	maxForType, authorized := lookupCapacity(lic.DevicesSupport, quotaKey)
	if !authorized || maxForType <= 0 {
		e.recordEnforcement("capacity", "denied_unauthorized_type")
		e.logger.Warn("license capacity denied: device type not authorized by license",
			zap.String("audit", "enforcement_capacity"),
			zap.String("license_id", lic.LicenseID),
			zap.String("device_type", deviceType),
			zap.String("capacity_group", quotaKey),
			zap.Any("devices_support", lic.DevicesSupport),
		)
		return fmt.Errorf("device type %q not authorized by license (capacity group %q has no quota for it): %w",
			deviceType, quotaKey, commonerrors.ErrLicenseCapacityExceeded)
	}

	usedByType, err := e.devices.CountDevicesByType(ctx)
	if err != nil {
		return fmt.Errorf("count devices by type for enforcement: %w", err)
	}
	used := capacityGroupUsage(usedByType, quotaKey)
	if used+additional > maxForType {
		e.recordEnforcement("capacity", "denied_capacity")
		e.logger.Warn("license capacity exceeded — device.create denied",
			zap.String("audit", "enforcement_capacity"),
			zap.String("license_id", lic.LicenseID),
			zap.String("device_type", deviceType),
			zap.String("capacity_group", quotaKey),
			zap.Int("type_capacity", maxForType),
			zap.Int("type_used", used),
			zap.Int("additional", additional),
			zap.Any("devices_support", lic.DevicesSupport),
		)
		e.raiseCapacityExhaustedAlert(ctx, lic, quotaKey, used, maxForType)
		return fmt.Errorf("type=%s used=%d, max=%d, additional=%d: %w",
			quotaKey, used, maxForType, additional, commonerrors.ErrLicenseCapacityExceeded)
	}

	e.recordEnforcement("capacity", "allowed")
	return nil
}

// raiseCapacityExhaustedAlert 在容量满拒绝时上报一条 warning 告警（issue #316）。
// 用 capacityExhaustedAlertDedup 时间窗去重，避免每次 periodic Inform 拒绝都 Send。
// 告警清除由 Monitor.CheckCapacity 负责（所有授权类型未满时 Clear），本方法不做 Clear。
// alarm engine 按 (DeviceSN=OMC-SYSTEM, AlarmIdentifier) 幂等去重，故 Send 安全。
func (e *EnforcerImpl) raiseCapacityExhaustedAlert(ctx context.Context, lic *SystemLicense, deviceType string, used, max int) {
	e.exhaustedAlertMu.Lock()
	if nowFunc().Sub(e.lastExhaustedAlertAt) < capacityExhaustedAlertDedup {
		e.exhaustedAlertMu.Unlock()
		return
	}
	e.lastExhaustedAlertAt = nowFunc()
	e.exhaustedAlertMu.Unlock()

	alert := Alert{
		LicenseID:  lic.ID,
		Identifier: CapacityExhaustedIdentifier,
		Severity:   AlertSeverityWarning,
		Summary:    fmt.Sprintf("License capacity exhausted for %s (%d/%d), new device rejected", deviceType, used, max),
		Details: map[string]interface{}{
			"license_id":       lic.LicenseID,
			"device_type":      deviceType,
			"type_used":        used,
			"type_capacity":    max,
			"devices_support":  lic.DevicesSupport,
			"recoverable_hint": "delete devices / upload a larger license to free capacity",
		},
	}
	if err := e.alertSink.Send(ctx, alert); err != nil {
		e.logger.Warn("send capacity-exhausted alert failed", zap.Error(err))
	}
}

// EnforceExpiry rejects when the current license is past expiry_date, when
// cumulative usage exceeds time_limit_hours, on system time rollback, or when
// no license is configured (fail-closed → ErrLicenseUnavailable). NULL
// expiry_date (perpetual) skips the date check but cumulative/rollback still apply.
func (e *EnforcerImpl) EnforceExpiry(ctx context.Context, operation string) error {
	lic, err := e.ActiveLicense(ctx)
	if err != nil {
		return err
	}
	if lic == nil {
		e.recordEnforcement(operation, "denied_no_license")
		e.logger.Warn("license expiry enforcement denied: no system license configured",
			zap.String("audit", "enforcement_expiry"),
			zap.String("operation", operation),
		)
		return fmt.Errorf("no system license configured: %w", commonerrors.ErrLicenseUnavailable)
	}
	if lic.ExpiryDate != nil && nowFunc().After(*lic.ExpiryDate) {
		e.recordEnforcement(operation, "denied_expired")
		e.logger.Warn("license expired — operation denied",
			zap.String("audit", "enforcement_expiry"),
			zap.String("license_id", lic.LicenseID),
			zap.String("operation", operation),
			zap.Time("expiry_date", *lic.ExpiryDate),
		)
		return fmt.Errorf("system license %s expired at %s: %w",
			lic.LicenseID, lic.ExpiryDate.Format(time.RFC3339),
			commonerrors.ErrLicenseExpired)
	}
	if err := e.enforceCumulative(ctx, lic); err != nil {
		return err
	}
	e.recordEnforcement(operation, "allowed")
	return nil
}

// enforceCumulative 复刻旧项目 checkOMCTimeUsed/checkLicenseExpire 的累计时长 +
// 时间回拨检测（compute-on-read：按 now - last_visited 累加）。time_limit_hours<=0
// 或未注入 usageRepo 时跳过。持久化错误 fail-open（仅告警），避免 DB 抖动锁死全部业务。
func (e *EnforcerImpl) enforceCumulative(ctx context.Context, lic *SystemLicense) error {
	if e.usageRepo == nil {
		return nil
	}
	timeLimit := extractTimeLimitHours(lic.FeatureList)
	if timeLimit <= 0 {
		return nil
	}
	lastVisited, err := e.usageRepo.LastVisited(ctx)
	if err != nil {
		e.logger.Warn("license cumulative usage load failed, skip", zap.Error(err))
		return nil
	}
	now := nowFunc()
	elapsed := now.Sub(lastVisited)
	if elapsed < 0 {
		e.recordEnforcement("expiry", "denied_rollback")
		e.logger.Warn("license time rollback detected — operation denied",
			zap.String("audit", "enforcement_expiry"),
			zap.Time("now", now), zap.Time("last_visited", lastVisited))
		return fmt.Errorf("system time rollback (now=%s < last_visited=%s): %w",
			now.UTC().Format(time.RFC3339), lastVisited.UTC().Format(time.RFC3339),
			commonerrors.ErrLicenseExpired)
	}
	total, err := e.usageRepo.Advance(ctx, elapsed.Hours())
	if err != nil {
		e.logger.Warn("license cumulative usage advance failed, skip", zap.Error(err))
		return nil
	}
	if total >= float64(timeLimit) {
		e.recordEnforcement("expiry", "denied_cumulative")
		e.logger.Warn("license cumulative usage exceeded — operation denied",
			zap.String("audit", "enforcement_expiry"),
			zap.Float64("used_hours", total), zap.Int("limit_hours", timeLimit))
		return fmt.Errorf("license cumulative usage %.1fh >= limit %dh: %w",
			total, timeLimit, commonerrors.ErrLicenseExpired)
	}
	return nil
}

// Quota returns the current enforcement state. has_active_license=false
// when no license is configured.
//
// 兼容老字段（MaxDevices/UsedDevices/UsageRatio/DaysRemaining）按总和填写，
// 新增 PerType map 暴露每个 device_type 的子配额。Used 在 Phase 7 per-type
// gating 上线前固定 0（精细化时再按 type 计数）。
func (e *EnforcerImpl) Quota(ctx context.Context) (*Quota, error) {
	lic, err := e.ActiveLicense(ctx)
	if err != nil {
		return nil, err
	}
	if lic == nil {
		return &Quota{HasActiveLicense: false}, nil
	}

	used, err := e.devices.CountDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("count devices for quota: %w", err)
	}

	maxDevices := totalCapacity(lic)
	q := &Quota{
		HasActiveLicense: true,
		MaxDevices:       maxDevices,
		UsedDevices:      used,
		LicenseType:      string(lic.LicenseType),
		GracePeriodDays:  0, // 新模型无 grace；保留字段为 JSON 兼容
		DaysRemaining:    -1,
	}
	if maxDevices > 0 {
		q.UsageRatio = float64(used) / float64(maxDevices)
	}
	if lic.ExpiryDate != nil {
		days := int(lic.ExpiryDate.Sub(nowFunc()).Hours() / 24)
		q.DaysRemaining = days
	}
	if len(lic.DevicesSupport) > 0 {
		usedByType, err := e.devices.CountDevicesByType(ctx)
		if err != nil {
			return nil, fmt.Errorf("count devices by type for quota: %w", err)
		}
		q.PerType = make(map[string]TypeQuotaItem, len(lic.DevicesSupport))
		for t, m := range lic.DevicesSupport {
			// 容量分组（issue #318）：用量按组合计（eNB 卡片的 Used 含 GSM 设备）。
			used := capacityGroupUsage(usedByType, upperKey(t))
			item := TypeQuotaItem{Max: m, Used: used}
			if m > 0 {
				item.Ratio = float64(used) / float64(m)
			}
			q.PerType[t] = item
		}
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
	case errors.Is(err, commonerrors.ErrLicenseUnavailable):
		return "no_license", true
	default:
		return "", false
	}
}
