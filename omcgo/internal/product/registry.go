package product

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ErrOrphan 由 MatchProductClass 在 productClass 未匹配任何 active pattern 时返回。
//
// 调用侧（设备 Bootstrap 路径）应捕获此错误，把设备视为孤儿待人工绑定（设计 §4.3.1）。
var ErrOrphan = errors.New("productClass matched no pattern (orphan device)")

// compiledPattern 是 product_class_patterns 行的运行时表示。
//
// regex 在 Refresh 时一次性编译；编译失败的行会被 WARN 跳过（避免单条坏正则瘫痪 Registry，
// 设计决策：S2 设计备忘 #4）。
type compiledPattern struct {
	regex     *regexp.Regexp
	productID uuid.UUID
	raw       string
	sortOrder int
}

// Registry 是 productClass → Product 路由的核心组件（设计 §4.3.1）。
//
// 缓存层级：
//   - patterns slice：atomic.Pointer 持有不可变快照，Refresh 整体替换；零锁热路径
//   - product 详情：sync.Map L1（按 ID）+ Cache L2（Redis）+ DB read-through
type Registry struct {
	repo     Repository
	cache    Cache
	logger   *zap.Logger
	metrics  *registryMetrics

	patterns    atomic.Pointer[[]compiledPattern] // 全局 sort_order 升序
	productByID sync.Map                          // map[uuid.UUID]*Product
}

// NewRegistry 构造一个未加载状态的 Registry。
// 调用方必须在投入服务前显式调用 Refresh(ctx) 以加载 patterns。
//
// cache 传入 nil 等价于注入 NopCache（退化为纯 L1 + DB）。
// metrics 传入 nil 等价于注入匿名 Registry 上的指标（不影响业务但失去可观测性）。
func NewRegistry(repo Repository, cache Cache, metrics *registryMetrics, logger *zap.Logger) *Registry {
	if cache == nil {
		cache = NopCache{}
	}
	if metrics == nil {
		metrics = NewRegistryMetrics(nil)
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	r := &Registry{
		repo:    repo,
		cache:   cache,
		logger:  logger.Named("product.registry"),
		metrics: metrics,
	}
	empty := make([]compiledPattern, 0)
	r.patterns.Store(&empty)
	return r
}

// Refresh 重新加载 patterns 全集 + 清空 product L1 + 触发 cache 版本递增。
//
// 调用顺序（无锁）：
//  1. repo.ListActivePatterns 读出全集（已按 sort_order ASC 排序）
//  2. 编译每条正则；失败 WARN 跳过
//  3. atomic.Swap patterns slice
//  4. drop L1 productByID（避免读到旧 product 详情）
//  5. cache.BumpVersion 通知其他实例
func (r *Registry) Refresh(ctx context.Context) error {
	t0 := time.Now()
	rows, err := r.repo.ListActivePatterns(ctx)
	if err != nil {
		r.metrics.refreshErr()
		return fmt.Errorf("list active patterns: %w", err)
	}

	compiled := make([]compiledPattern, 0, len(rows))
	skipped := 0
	for _, row := range rows {
		re, err := regexp.Compile(row.ProductClass)
		if err != nil {
			r.logger.Warn("skip pattern: regex compile failed",
				zap.String("pattern", row.ProductClass),
				zap.Int("sort_order", row.SortOrder),
				zap.String("product_id", row.ProductID.String()),
				zap.Error(err))
			skipped++
			continue
		}
		compiled = append(compiled, compiledPattern{
			regex:     re,
			productID: row.ProductID,
			raw:       row.ProductClass,
			sortOrder: row.SortOrder,
		})
	}
	r.patterns.Store(&compiled)

	// drop L1 product details — 后续读会按需 read-through 重建
	r.productByID.Range(func(k, _ any) bool { r.productByID.Delete(k); return true })

	if _, err := r.cache.BumpVersion(ctx); err != nil {
		// 版本递增失败不阻塞 Refresh — patterns 已 swap 完毕；其他实例下次心跳会发现版本未变跳过 Refresh。
		// 这是可接受的弱一致：单实例无影响，多实例最差表现是这次失效信号丢失。
		r.logger.Warn("bump cache version failed (non-fatal)", zap.Error(err))
	}

	r.metrics.refreshOK()
	r.logger.Info("ProductRegistry refreshed",
		zap.Int("patterns_loaded", len(compiled)),
		zap.Int("patterns_skipped", skipped),
		zap.Duration("duration", time.Since(t0)))
	return nil
}

// MatchProductClass 按全局 sort_order 升序遍历 patterns，首次命中返回 product。
// 全部未命中 → 返回 ErrOrphan。
func (r *Registry) MatchProductClass(ctx context.Context, productClass string) (*MatchResult, error) {
	t0 := time.Now()
	defer func() { r.metrics.matchDuration.Observe(time.Since(t0).Seconds()) }()

	patternsPtr := r.patterns.Load()
	if patternsPtr == nil || len(*patternsPtr) == 0 {
		r.metrics.matchOrphan()
		return nil, ErrOrphan
	}
	for _, p := range *patternsPtr {
		if p.regex.MatchString(productClass) {
			prod, err := r.GetProductByID(ctx, p.productID)
			if err != nil {
				return nil, fmt.Errorf("load product %s for matched pattern %q: %w", p.productID, p.raw, err)
			}
			if prod == nil {
				// 数据不一致：pattern 引用了不存在的 product。容忍但 WARN —
				// 继续遍历，让兜底正则有机会接住（设计 §4.3.1 一次匹配，但此场景属异常恢复）。
				r.logger.Warn("matched pattern points to missing product; continuing",
					zap.String("pattern", p.raw),
					zap.String("product_id", p.productID.String()))
				continue
			}
			r.metrics.matchHit()
			return &MatchResult{
				Product:        prod,
				MatchedPattern: p.raw,
				GlobalOrder:    p.sortOrder,
			}, nil
		}
	}
	r.metrics.matchOrphan()
	return nil, ErrOrphan
}

// GetProductByID 按 L1 → L2 → DB 顺序加载 Product 详情。命中后回填上层缓存。
// 不存在 → (nil, nil)。
func (r *Registry) GetProductByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	// L1
	if v, ok := r.productByID.Load(id); ok {
		r.metrics.cacheHit("L1")
		return v.(*Product), nil
	}
	// L2
	if r.cache != nil {
		if p, err := r.cache.GetProduct(ctx, id); err != nil {
			r.logger.Warn("L2 cache get failed; falling back to DB",
				zap.String("product_id", id.String()),
				zap.Error(err))
		} else if p != nil {
			r.productByID.Store(id, p)
			r.metrics.cacheHit("L2")
			return p, nil
		}
	}
	// DB
	p, err := r.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("repo get product %s: %w", id, err)
	}
	if p == nil {
		r.metrics.cacheHit("miss")
		return nil, nil
	}
	r.productByID.Store(id, p)
	if err := r.cache.SetProduct(ctx, p); err != nil {
		// L2 写失败不影响 L1 命中；记录 WARN 让运维介入
		r.logger.Warn("L2 cache set failed (non-fatal)",
			zap.String("product_id", id.String()),
			zap.Error(err))
	}
	r.metrics.cacheHit("miss")
	return p, nil
}

// ValidateReferences 全表扫描 products，校验 indicator_platform / alarm_ne_type 软引用。
//
// 与 P1-06 product Loader 的启动期一次性校验互补：本方法用于运行时管理 API（P3-01）
// 在保存 product 前预校验，或运维主动巡检。paramModel.id 由 PG FK 强一致已保证，本方法不再校验。
func (r *Registry) ValidateReferences(ctx context.Context) (*ValidationReport, error) {
	products, err := r.repo.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	alarmNeTypes, err := r.repo.FetchAlarmNeTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch alarm ne_types: %w", err)
	}

	// indicator platform 按 deviceType 分桶查表 — 同 deviceType 的多个 product 共享一次查询
	platformsByDevice := make(map[string]map[string]struct{}, 4)
	report := &ValidationReport{TotalProducts: len(products)}

	for _, p := range products {
		// alarm_ne_type
		if p.AlarmNeType != "" {
			if _, ok := alarmNeTypes[p.AlarmNeType]; !ok {
				report.WarnedProducts = append(report.WarnedProducts, ValidationIssue{
					ProductID:   p.ID,
					ProductName: p.Name,
					Field:       "alarm_ne_type",
					Reason:      fmt.Sprintf("ne_type %q not found in alarm_definitions", p.AlarmNeType),
				})
			}
		}
		// indicator_platform
		if p.IndicatorDeviceType != "" && p.IndicatorPlatform != "" {
			platforms, cached := platformsByDevice[p.IndicatorDeviceType]
			if !cached {
				platforms, err = r.repo.FetchIndicatorPlatformsByDeviceType(ctx, p.IndicatorDeviceType)
				if err != nil {
					return nil, fmt.Errorf("fetch indicator platforms for %s: %w", p.IndicatorDeviceType, err)
				}
				platformsByDevice[p.IndicatorDeviceType] = platforms
			}
			if _, ok := platforms[p.IndicatorPlatform]; !ok {
				report.WarnedProducts = append(report.WarnedProducts, ValidationIssue{
					ProductID:   p.ID,
					ProductName: p.Name,
					Field:       "indicator_platform",
					Reason: fmt.Sprintf("platform %q not found in perf_indicators_%s",
						p.IndicatorPlatform, p.IndicatorDeviceType),
				})
			}
		}
	}
	if len(report.WarnedProducts) > 0 {
		r.logger.Warn("ProductRegistry reference validation found soft-ref gaps",
			zap.Int("warned_count", len(report.WarnedProducts)))
	}
	return report, nil
}

// PatternCount 返回当前快照中已编译的 pattern 数量；供 healthz / debug 端点使用。
func (r *Registry) PatternCount() int {
	if p := r.patterns.Load(); p != nil {
		return len(*p)
	}
	return 0
}
