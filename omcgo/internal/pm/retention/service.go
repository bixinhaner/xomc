package retention

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// SysConfigReader 是本包对 admin.SysConfigRepository 的最小依赖窄接口，
// 只取 GetByKey 一个方法，便于单测用 stub 实现。
//
// 真实实现：omcgo/internal/admin/sys_config.go 的 PgSysConfigRepository。
type SysConfigReader interface {
	// GetByKey 返回 (category, key) 对应的配置；不存在时返回 (nil, nil) 或 (nil, error)，
	// 由调用方按"缺失走默认"语义处理。
	GetByKey(ctx context.Context, category, key string) (*SysConfigRow, error)
}

// SysConfigRow 是 retention 包内的轻量 DTO，仅含 retention 需要的字段。
// 避免直接依赖 internal/admin 包导致循环引用。
//
// adapter wiring 时通过简单 mapper 从 admin.SysConfig 转换：
//
//	type adminRepoAdapter struct{ inner admin.SysConfigRepository }
//	func (a adminRepoAdapter) GetByKey(ctx, cat, key string) (*retention.SysConfigRow, error) {
//	    row, err := a.inner.GetByKey(ctx, cat, key)
//	    if err != nil || row == nil { return nil, err }
//	    return &retention.SysConfigRow{Value: row.Value, ValueType: row.ValueType}, nil
//	}
type SysConfigRow struct {
	Value     string
	ValueType string
}

// ReloadListener 在 5 键之一保存后触发的回调，listener 自行判定哪些值变化、
// 是否需要 alter_job 重新挂 policy 或通知 cron。
//
// G3/G5 实施时把 alter_compression_policy / alter_retention_policy 的 SQL
// 触发逻辑挂在这里（避免本包硬依赖 TimescaleDB / pgx）。
type ReloadListener func(ctx context.Context, current map[PolicyKey]int, changed []PolicyKey)

// Service 是 PM 保留策略的运行时管理者：
//   - 从 sys_configs 读 5 个键到内存缓存（first-Get 时懒加载，或 wiring 时主动 Reload）
//   - 提供 Get(key) → 天数（缺失走默认）
//   - 通过 admin.SysConfigService.RegisterSavedHook(svc.OnSysConfigSaved) 挂监听，
//     category="pm.retention" 时触发 Reload + 通知所有 ReloadListener
//
// 并发安全：cache 用 sync.RWMutex 保护；listeners 仅在 wiring 阶段注册，运行期只读。
type Service struct {
	reader    SysConfigReader
	logger    *zap.Logger
	mu        sync.RWMutex
	cache     map[PolicyKey]int
	listeners []ReloadListener
}

// NewService 创建 retention.Service。
// wiring 完成后调用方应立即调一次 Reload(ctx) 把缓存预热到当前 sys_configs 值；
// 缺失值走 DefaultDays。
func NewService(reader SysConfigReader, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		reader: reader,
		logger: logger,
		cache:  make(map[PolicyKey]int, len(AllKeys())),
	}
}

// RegisterListener 注册 reload 回调，仅在进程启动 wiring 阶段调用，无并发保护。
func (s *Service) RegisterListener(l ReloadListener) {
	if l == nil {
		return
	}
	s.listeners = append(s.listeners, l)
}

// Get 返回指定 key 当前生效的保留天数。
// 优先级：cache → sys_configs → DefaultDays。
// cache 未命中时尝试现场读 sys_configs（懒加载），仍未命中走默认。
func (s *Service) Get(ctx context.Context, key PolicyKey) int {
	s.mu.RLock()
	if v, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	// cache miss，尝试现场读
	v, err := s.readOne(ctx, key)
	if err != nil {
		s.logger.Warn("retention.Get fallback to default (read sys_configs failed)",
			zap.String("key", string(key)), zap.Error(err))
		return DefaultDays[key]
	}
	s.mu.Lock()
	s.cache[key] = v
	s.mu.Unlock()
	return v
}

// GetAll 返回所有 5 键当前生效的值。
func (s *Service) GetAll(ctx context.Context) map[PolicyKey]int {
	out := make(map[PolicyKey]int, len(AllKeys()))
	for _, k := range AllKeys() {
		out[k] = s.Get(ctx, k)
	}
	return out
}

// Reload 强制重新读取 sys_configs 全部 5 键，更新缓存，并通知 listener 变化项。
// 在 wiring 阶段调一次预热缓存；在 SysConfigSaved hook（category=pm.retention）触发时再调。
//
// 读取某键失败时，优先保留上次已知好值（缓存命中），避免暂时性 DB 故障把用户配置的天数
// 重置为默认值并传播给 listener（进而错误地重写 TimescaleDB retention policy）。
// 仅在缓存为空（首次启动）时才降级到 DefaultDays。
func (s *Service) Reload(ctx context.Context) error {
	// 快照当前缓存，用于读取失败时的降级策略（不持锁做 readOne 避免长时间锁住）。
	s.mu.RLock()
	cacheSnapshot := make(map[PolicyKey]int, len(s.cache))
	for k, v := range s.cache {
		cacheSnapshot[k] = v
	}
	s.mu.RUnlock()

	newVals := make(map[PolicyKey]int, len(AllKeys()))
	for _, k := range AllKeys() {
		v, err := s.readOne(ctx, k)
		if err != nil {
			if cached, ok := cacheSnapshot[k]; ok {
				// 保留上次已知好值，不通知 listener，避免覆盖用户配置。
				s.logger.Warn("retention.Reload: preserving cached value on read error",
					zap.String("key", string(k)), zap.Int("cached_days", cached), zap.Error(err))
				v = cached
			} else {
				// 首次启动缓存为空，降级到默认值（TimescaleDB policy 尚未被用户配置过）。
				s.logger.Warn("retention.Reload: fallback to default (no cache yet)",
					zap.String("key", string(k)), zap.Int("default_days", DefaultDays[k]), zap.Error(err))
				v = DefaultDays[k]
			}
		}
		newVals[k] = v
	}

	s.mu.Lock()
	changed := make([]PolicyKey, 0, len(AllKeys()))
	for k, newV := range newVals {
		if oldV, ok := s.cache[k]; !ok || oldV != newV {
			changed = append(changed, k)
		}
	}
	s.cache = newVals
	listeners := s.listeners // 引用切片快照，调用时不持锁
	s.mu.Unlock()

	if len(changed) == 0 {
		return nil
	}
	s.logger.Info("retention policy reloaded", zap.Int("changed_keys", len(changed)))
	for _, l := range listeners {
		// listener panic 不破坏外层（已注释规避，实施 review 时 caller 自行包 recover）
		l(ctx, newVals, changed)
	}
	return nil
}

// ReloadStrict reloads every configured policy without defaulting or retaining
// cached values. It is used by the persistent configuration apply pipeline:
// reporting success after an invalid or unreadable value would otherwise claim
// that a policy changed when the old/default policy is still active.
func (s *Service) ReloadStrict(ctx context.Context) (map[PolicyKey]int, error) {
	newVals := make(map[PolicyKey]int, len(AllKeys()))
	for _, k := range AllKeys() {
		v, err := s.readOne(ctx, k)
		if err != nil {
			// A partial category update is supported by the generic API. A
			// missing sibling key is therefore an intentional default, unlike a
			// malformed value or a database read failure.
			if errors.Is(err, ErrUnknownPolicyKey) || errors.Is(err, commonerrors.ErrNotFound) {
				v = DefaultDays[k]
			} else {
				return nil, fmt.Errorf("reload strict %s: %w", k, err)
			}
		}
		newVals[k] = v
	}
	s.mu.Lock()
	s.cache = newVals
	s.mu.Unlock()
	return newVals, nil
}

// OnSysConfigSaved 是注册给 admin.SysConfigService.RegisterSavedHook 的回调。
// wiring 时挂：
//
//	sysCfgSvc.RegisterSavedHook(retentionSvc.OnSysConfigSaved)
//
// chenbo01 的 SavedHook 收到的 category 是本次保存的 category；
// 仅当 category == retention.Category 时才触发 Reload，避免无关 category 误触发。
func (s *Service) OnSysConfigSaved(ctx context.Context, category string) {
	if category != Category {
		return
	}
	if err := s.Reload(ctx); err != nil {
		s.logger.Warn("retention.OnSysConfigSaved Reload failed", zap.Error(err))
	}
}

// readOne 读取单个 key 的当前值并做合法性校验。
// 不存在 / 类型不对 / 越界 → 返回错误（调用方走默认）。
func (s *Service) readOne(ctx context.Context, key PolicyKey) (int, error) {
	row, err := s.reader.GetByKey(ctx, Category, string(key))
	if err != nil {
		return 0, fmt.Errorf("read sys_configs (%s,%s): %w", Category, key, err)
	}
	if row == nil {
		return 0, ErrUnknownPolicyKey
	}
	if row.ValueType != "int" {
		return 0, fmt.Errorf("%w (got %q)", ErrInvalidValueType, row.ValueType)
	}
	days, err := strconv.Atoi(row.Value)
	if err != nil {
		return 0, fmt.Errorf("parse days for %s: %w", key, err)
	}
	if err := ValidateDays(days); err != nil {
		return 0, fmt.Errorf("validate days for %s: %w", key, err)
	}
	return days, nil
}
