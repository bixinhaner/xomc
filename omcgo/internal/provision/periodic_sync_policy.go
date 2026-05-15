package provision

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// SysConfigLookup 是 PeriodicSyncPolicy 唯一的外部依赖：按 (category, key) 读
// sys_configs 行的 value 列。消费者驱动小接口（不引 admin 包），由 wiring 层
// 提供 admin.PgSysConfigRepository.GetByKey 的适配器。
//
// 返回 (value, true) 表示读到；(_, false) 表示 key 不存在 / 读失败 — 调用方
// 退化到 default。
type SysConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// PeriodicSyncPolicy 缓存 sys_configs (category='device') 周期性参数同步设置
// 的运行时值。设计同 admin.SecurityPolicy：atomic.Pointer 无锁读 + 30s 缓存 +
// nil 安全（未注入 lookup 时全部走 default）。
//
// FE 表单字段映射（pages/system/SystemConfig/DeviceSettings.tsx）：
//
//	periodicSyncEnabled              bool   总开关
//	periodicSyncIntervalHours        int    扫描周期（小时）
//	periodicSyncBatchSize            int    单轮最多挑多少台设备
//	periodicSyncMaxConcurrent        int    入队并发度
//	periodicSyncStaggerWindowMinutes int    打散窗口（分钟，0=关）
type PeriodicSyncPolicy struct {
	lookup  SysConfigLookup
	logger  *zap.Logger
	cacheMu sync.Mutex
	cache   atomic.Pointer[PeriodicSyncSnapshot]
}

// PeriodicSyncSnapshot 是某一时刻的运行时配置快照。
type PeriodicSyncSnapshot struct {
	Enabled       bool
	Interval      time.Duration
	BatchSize     int
	MaxConcurrent int
	StaggerWindow time.Duration

	expiresAt time.Time
}

// 默认值（与 docs/周期性参数同步配置项说明.md §二 对齐）。
const (
	defaultPeriodicSyncEnabled       = false
	defaultPeriodicSyncInterval      = 24 * time.Hour
	defaultPeriodicSyncBatchSize     = 200
	defaultPeriodicSyncMaxConcurrent = 10
	defaultPeriodicSyncStaggerWindow = 0 * time.Second

	periodicSyncCacheTTL = 30 * time.Second
)

// NewPeriodicSyncPolicy creates a new policy with an injected SysConfigLookup.
// nil-safe：lookup=nil 时所有字段返 default（适合单测 / 极简部署）。
func NewPeriodicSyncPolicy(lookup SysConfigLookup, logger *zap.Logger) *PeriodicSyncPolicy {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PeriodicSyncPolicy{lookup: lookup, logger: logger}
}

// Snapshot 返回当前配置快照。30s 缓存命中时无锁。
func (p *PeriodicSyncPolicy) Snapshot(ctx context.Context) PeriodicSyncSnapshot {
	if cached := p.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return *cached
	}

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	if cached := p.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return *cached
	}

	snap := defaultPeriodicSyncSnapshot()
	snap.expiresAt = time.Now().Add(periodicSyncCacheTTL)
	if p.lookup != nil {
		p.loadFromSysConfig(ctx, &snap)
	}
	p.cache.Store(&snap)
	return snap
}

// InvalidateCache 让下一次 Snapshot 重新加载（FE 保存"设备设置"后可主动调用）。
func (p *PeriodicSyncPolicy) InvalidateCache() {
	p.cache.Store(nil)
}

// defaultPeriodicSyncSnapshot 返默认值的 snapshot（不含 expiresAt）。
func defaultPeriodicSyncSnapshot() PeriodicSyncSnapshot {
	return PeriodicSyncSnapshot{
		Enabled:       defaultPeriodicSyncEnabled,
		Interval:      defaultPeriodicSyncInterval,
		BatchSize:     defaultPeriodicSyncBatchSize,
		MaxConcurrent: defaultPeriodicSyncMaxConcurrent,
		StaggerWindow: defaultPeriodicSyncStaggerWindow,
	}
}

// loadFromSysConfig 把 sys_configs 中可读到的字段覆盖到 default 之上。
// 单字段读 / 解析失败不影响其它字段（部分配置仍可生效）。
func (p *PeriodicSyncPolicy) loadFromSysConfig(ctx context.Context, s *PeriodicSyncSnapshot) {
	const cat = "device"

	if v, ok := p.lookup(ctx, cat, "periodicSyncEnabled"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			s.Enabled = b
		}
	}
	if v, ok := p.lookup(ctx, cat, "periodicSyncIntervalHours"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.Interval = time.Duration(n) * time.Hour
		}
	}
	if v, ok := p.lookup(ctx, cat, "periodicSyncBatchSize"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.BatchSize = n
		}
	}
	if v, ok := p.lookup(ctx, cat, "periodicSyncMaxConcurrent"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.MaxConcurrent = n
		}
	}
	if v, ok := p.lookup(ctx, cat, "periodicSyncStaggerWindowMinutes"); ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			s.StaggerWindow = time.Duration(n) * time.Minute
		}
	}
}
