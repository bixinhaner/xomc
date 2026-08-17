package definition

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Registry 提供告警定义的 O(1) identifier → ResolvedDefinition 查询能力（设计 §3.3）。
//
// 数据规模 ~443 行 alarm_definitions + 4 行 severity，可一次性全量装内存，
// 故不引入 L2 Redis（与 ParamRegistry / ProductRegistry 不同——后者数据规模更大）。
//
// 启动期 Refresh 一次：Loader 加载 XML → DB → 本 Registry 从 DB 重读到 sync.Map。
// 运行期：API CRUD 改 alarm_definitions 后调 Registry.Refresh 同步 in-memory cache。
type Registry struct {
	repo    Repository
	logger  *zap.Logger
	metrics *registryMetrics

	byIdentifier sync.Map // map[string]ResolvedDefinition
	loaded       atomic_bool
}

// NewRegistry 构造 Registry。
//
//   - logger 为 nil 使用 zap.NewNop()
//   - metrics 为 nil 注册到匿名 prometheus.Registry
func NewRegistry(repo Repository, metrics *registryMetrics, logger *zap.Logger) *Registry {
	if metrics == nil {
		metrics = NewRegistryMetrics(nil)
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Registry{
		repo:    repo,
		logger:  logger.Named("alarmdef.registry"),
		metrics: metrics,
	}
}

// Lookup 按 identifier 查告警定义。返回 nil + ErrUnknownIdentifier 表示未命中（fallback 入口）。
func (r *Registry) Lookup(_ context.Context, identifier string) (*ResolvedDefinition, error) {
	v, ok := r.byIdentifier.Load(identifier)
	if !ok {
		r.metrics.lookupMiss()
		return nil, ErrUnknownIdentifier
	}
	r.metrics.lookupHit()
	rd := v.(ResolvedDefinition)
	return &rd, nil
}

// Refresh 重新装载全量 alarm_definitions 到 sync.Map。
//
// 加载语义：build → swap → 旧条目自然 GC（Range Delete 后 Store）。
// 调用方典型路径：
//   - 启动期 Provider 调一次
//   - 运行期 alarm_definitions API CRUD 后调一次
func (r *Registry) Refresh(ctx context.Context) error {
	t0 := time.Now()
	defs, err := r.repo.ListAll(ctx)
	if err != nil {
		r.metrics.refreshErr()
		return fmt.Errorf("list alarm_definitions: %w", err)
	}

	// 先收集旧 keys，遍历完成后再删（避免在 Range 内 Delete 影响迭代正确性）
	oldKeys := make([]string, 0)
	r.byIdentifier.Range(func(k, _ any) bool {
		oldKeys = append(oldKeys, k.(string))
		return true
	})
	for _, k := range oldKeys {
		r.byIdentifier.Delete(k)
	}

	for _, d := range defs {
		r.byIdentifier.Store(d.Identifier, d)
	}

	r.loaded.set(true)
	r.metrics.refreshOK()
	r.logger.Info("AlarmDefRegistry refreshed",
		zap.Int("count", len(defs)),
		zap.Duration("duration", time.Since(t0)),
	)
	return nil
}

// Loaded 返回是否已成功 Refresh 至少一次。供监控 / 健康检查使用。
func (r *Registry) Loaded() bool { return r.loaded.get() }

// Count 返回当前已加载 identifier 数量（Range 一次，O(N)；调试用）。
func (r *Registry) Count() int {
	n := 0
	r.byIdentifier.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}

// ── atomic_bool 简易封装（避免引入 sync/atomic 多返回值 + 跨平台差异） ──

type atomic_bool struct {
	mu sync.RWMutex
	v  bool
}

func (a *atomic_bool) set(b bool) {
	a.mu.Lock()
	a.v = b
	a.mu.Unlock()
}

func (a *atomic_bool) get() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.v
}
