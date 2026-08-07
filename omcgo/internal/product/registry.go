package product

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
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

// ErrInactiveParamModel 表示 productClass 命中了产品规则，但产品关联的参数模型已停用。
// 它包装 ErrOrphan，使既有设备接入调用方继续把该设备视为未识别；需要禁止透传的
// 调用方（例如 MML）可优先判断本错误并采取更严格的处理。
var ErrInactiveParamModel = fmt.Errorf("%w: matched product parameter model is inactive", ErrOrphan)

// compiledPattern 是 product_class_patterns 行的运行时表示。
//
// regex 在 Refresh 时一次性编译；编译失败的行会被 WARN 跳过（避免单条坏正则瘫痪 Registry，
// 设计决策：S2 设计备忘 #4）。
type compiledPattern struct {
	regex              *regexp.Regexp
	productID          uuid.UUID
	raw                string
	sortOrder          int
	paramModelInactive bool
}

// Registry 是 productClass → Product 路由的核心组件（设计 §4.3.1）。
//
// 缓存层级：
//   - patterns slice：atomic.Pointer 持有不可变快照，Refresh 整体替换；零锁热路径
//   - product 详情：sync.Map L1（按 ID）+ Cache L2（Redis）+ DB read-through
//   - productClass 路由：sync.Map L1（按 class 字面量）+ Cache L2（Redis byProductClass）+
//     全表 regex 扫描兜底（T-0173）
type Registry struct {
	repo    Repository
	cache   Cache
	logger  *zap.Logger
	metrics *registryMetrics

	patterns       atomic.Pointer[[]compiledPattern] // 全局 sort_order 升序
	productByID    sync.Map                          // map[uuid.UUID]*Product
	productClassL1 sync.Map                          // map[string]*ProductClassCacheEntry — T-0173
	cacheVersion   atomic.Int64
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

func (r *Registry) ensureFresh(ctx context.Context) error {
	if r.cache == nil {
		return nil
	}
	version, err := r.cache.GetVersion(ctx)
	if err != nil {
		r.logger.Warn("get cache version failed; using current registry snapshot", zap.Error(err))
		return nil
	}
	if version == 0 || version == r.cacheVersion.Load() {
		return nil
	}
	if err := r.refresh(ctx, false); err != nil {
		return fmt.Errorf("refresh after cache version change %d->%d: %w", r.cacheVersion.Load(), version, err)
	}
	r.cacheVersion.Store(version)
	return nil
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
	return r.refresh(ctx, true)
}

func (r *Registry) refresh(ctx context.Context, bumpVersion bool) error {
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
			// #17: 坏正则不再静默吞掉 —— WARN 日志 + 计数器，便于告警与排障。
			// 仍跳过该行（避免单条坏正则瘫痪整个 Registry），但留下可观测痕迹。
			r.logger.Warn("skip pattern: regex compile failed",
				zap.String("pattern", row.ProductClass),
				zap.Int("sort_order", row.SortOrder),
				zap.String("product_id", row.ProductID.String()),
				zap.Error(err))
			r.metrics.patternSkip()
			skipped++
			continue
		}
		compiled = append(compiled, compiledPattern{
			regex:              re,
			productID:          row.ProductID,
			raw:                row.ProductClass,
			sortOrder:          row.SortOrder,
			paramModelInactive: row.ParamModelInactive,
		})
	}
	r.patterns.Store(&compiled)

	// drop L1 product details — 后续读会按需 read-through 重建
	r.productByID.Range(func(k, _ any) bool { r.productByID.Delete(k); return true })
	// drop L1 productClass entries — 即便不主动删，旧 entry 的 CacheVersion 也会与新版本
	// 不匹配而被识别为 stale；这里同步清空降低后续 stale 探测的开销（T-0173）。
	r.productClassL1.Range(func(k, _ any) bool { r.productClassL1.Delete(k); return true })

	if bumpVersion {
		if _, err := r.cache.BumpVersion(ctx); err != nil {
			// 版本递增失败不阻塞 Refresh — patterns 已 swap 完毕；其他实例下次心跳会发现版本未变跳过 Refresh。
			// 这是可接受的弱一致：单实例无影响，多实例最差表现是这次失效信号丢失。
			r.logger.Warn("bump cache version failed (non-fatal)", zap.Error(err))
		} else if version, err := r.cache.GetVersion(ctx); err != nil {
			r.logger.Warn("read cache version after bump failed (non-fatal)", zap.Error(err))
		} else {
			r.cacheVersion.Store(version)
		}
	}

	r.metrics.refreshOK()
	// #17: 有坏正则被跳过时整体抬到 WARN，确保聚合信息不被 Info 噪声淹没；
	// 逐条 pattern 的细节已在上面的循环里 WARN + 计数。
	logRefresh := r.logger.Info
	if skipped > 0 {
		logRefresh = r.logger.Warn
	}
	logRefresh("ProductRegistry refreshed",
		zap.Int("patterns_loaded", len(compiled)),
		zap.Int("patterns_skipped", skipped),
		zap.Duration("duration", time.Since(t0)))
	return nil
}

// MatchProductClass 按全局 sort_order 升序遍历 patterns，首次命中返回 product。
// 全部未命中 → 返回 ErrOrphan。
//
// 命中路径（T-0173 二级缓存）：
//  1. productClassL1：按 class 字面量查 sync.Map；CacheVersion 匹配 → materialize 返回
//  2. cache.GetProductClass：Redis L2；CacheVersion 匹配 → 填回 L1 后 materialize 返回
//  3. matchPatternsSlow：全表 regex 扫描；写回 L1 + L2（hit TTL 1h / orphan TTL 5min）
//
// stale 探测：L1/L2 entry.CacheVersion ≠ r.cacheVersion 即旁路；不主动 DEL。
func (r *Registry) MatchProductClass(ctx context.Context, productClass string) (*MatchResult, error) {
	t0 := time.Now()
	defer func() { r.metrics.matchDuration.Observe(time.Since(t0).Seconds()) }()
	if err := r.ensureFresh(ctx); err != nil {
		return nil, err
	}

	currentVer := r.cacheVersion.Load()

	// L1
	if v, ok := r.productClassL1.Load(productClass); ok {
		entry := v.(*ProductClassCacheEntry)
		if entry.CacheVersion == currentVer {
			r.metrics.classCacheHit("L1")
			return r.materializeFromEntry(ctx, entry)
		}
	}

	// L2
	if r.cache != nil {
		entry, err := r.cache.GetProductClass(ctx, productClass)
		if err != nil {
			r.logger.Warn("L2 productClass cache get failed; falling back to slow path",
				zap.String("product_class", productClass),
				zap.Error(err))
		} else if entry != nil && entry.CacheVersion == currentVer {
			r.productClassL1.Store(productClass, entry)
			r.metrics.classCacheHit("L2")
			return r.materializeFromEntry(ctx, entry)
		}
	}

	// Slow path：全表 regex 扫描
	r.metrics.classCacheHit("miss")
	entry, result, err := r.matchPatternsSlow(ctx, productClass, currentVer)
	if err != nil {
		return nil, err
	}

	// 回填 L1 + L2
	r.productClassL1.Store(productClass, entry)
	if r.cache != nil {
		ttl := ProductClassHitTTL
		if entry.Orphan {
			ttl = ProductClassOrphanTTL
		}
		if err := r.cache.SetProductClass(ctx, productClass, entry, ttl); err != nil {
			// 与 SetProduct 一致：L2 写失败不影响 L1，仅 WARN
			r.logger.Warn("L2 productClass cache set failed (non-fatal)",
				zap.String("product_class", productClass),
				zap.Error(err))
		}
	}

	if entry.Orphan {
		r.metrics.matchOrphan()
		return nil, ErrOrphan
	}
	if entry.ParamModelInactive {
		return result, ErrInactiveParamModel
	}
	r.metrics.matchHit()
	return result, nil
}

// materializeFromEntry 把 cache entry 还原为 MatchResult（hit）或 ErrOrphan。
// hit 路径会调 GetProductByID 复用 product 详情的三级缓存。
//
// 若 entry 标记 hit 但 productByID 查不到（dangling cache entry），降级为 slow path
// 重算 —— 这与 dangling pattern 的容忍策略一致：宁可慢一次也不阻塞业务。
func (r *Registry) materializeFromEntry(ctx context.Context, entry *ProductClassCacheEntry) (*MatchResult, error) {
	if entry.Orphan {
		r.metrics.matchOrphan()
		return nil, ErrOrphan
	}
	prod, err := r.GetProductByID(ctx, entry.MatchedProductID)
	if err != nil {
		return nil, fmt.Errorf("load product %s from cached match: %w", entry.MatchedProductID, err)
	}
	if prod == nil {
		// dangling cache：product 被删/未同步。Drop L1 entry 让下次走 slow path 重算。
		r.productClassL1.Delete(entry.MatchedPattern)
		r.logger.Warn("cached match references missing product; will recompute on next call",
			zap.String("product_id", entry.MatchedProductID.String()),
			zap.String("matched_pattern", entry.MatchedPattern))
		return nil, fmt.Errorf("cached match references missing product %s", entry.MatchedProductID)
	}
	result := &MatchResult{
		Product:        prod,
		MatchedPattern: entry.MatchedPattern,
		GlobalOrder:    entry.GlobalSortOrder,
	}
	if entry.ParamModelInactive {
		return result, ErrInactiveParamModel
	}
	r.metrics.matchHit()
	return result, nil
}

// matchPatternsSlow 走全表 regex 扫描的兜底路径，返回缓存条目 + MatchResult（仅 hit）。
// orphan / dangling-pattern fall-through 与原 MatchProductClass 行为完全一致。
func (r *Registry) matchPatternsSlow(ctx context.Context, productClass string, currentVer int64) (*ProductClassCacheEntry, *MatchResult, error) {
	patternsPtr := r.patterns.Load()
	if patternsPtr == nil || len(*patternsPtr) == 0 {
		return &ProductClassCacheEntry{Orphan: true, CacheVersion: currentVer}, nil, nil
	}
	for _, p := range *patternsPtr {
		if !p.regex.MatchString(productClass) {
			continue
		}
		prod, err := r.GetProductByID(ctx, p.productID)
		if err != nil {
			return nil, nil, fmt.Errorf("load product %s for matched pattern %q: %w", p.productID, p.raw, err)
		}
		if prod == nil {
			// 数据不一致：pattern 引用了不存在的 product。容忍但 WARN —
			// 继续遍历，让兜底正则有机会接住（设计 §4.3.1）。
			r.logger.Warn("matched pattern points to missing product; continuing",
				zap.String("pattern", p.raw),
				zap.String("product_id", p.productID.String()))
			continue
		}
		entry := &ProductClassCacheEntry{
			MatchedProductID:   prod.ID,
			Orphan:             false,
			ParamModelInactive: p.paramModelInactive,
			MatchedPattern:     p.raw,
			GlobalSortOrder:    p.sortOrder,
			CacheVersion:       currentVer,
		}
		return entry, &MatchResult{
			Product:        prod,
			MatchedPattern: p.raw,
			GlobalOrder:    p.sortOrder,
		}, nil
	}
	return &ProductClassCacheEntry{Orphan: true, CacheVersion: currentVer}, nil, nil
}

// GetProductByID 按 L1 → L2 → DB 顺序加载 Product 详情。命中后回填上层缓存。
// 不存在 → (nil, nil)。
func (r *Registry) GetProductByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	if err := r.ensureFresh(ctx); err != nil {
		return nil, err
	}
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

// GetProductByName 按产品名称加载产品。产品名称是管理面策略等业务功能的
// 稳定展示/选择口径；设备南向上报的 product_class 仍由 MatchProductClass
// 负责路由到同一产品。
func (r *Registry) GetProductByName(ctx context.Context, name string) (*Product, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	if err := r.ensureFresh(ctx); err != nil {
		return nil, err
	}
	products, err := r.repo.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products by name: %w", err)
	}
	for _, p := range products {
		if p != nil && strings.EqualFold(strings.TrimSpace(p.Name), name) {
			return p, nil
		}
	}
	return nil, nil
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

// GetPatternsByProductID 返回某产品当前 active 的全部 product_class 模式字面量。
// 用于 file-manager 列表页按「产品名称」过滤时,把 product_id 解析为该产品的
// pattern 集合,SQL 层再 IN (...) 过滤。无匹配返回空切片(调用方可据此返回空结果)。
func (r *Registry) GetPatternsByProductID(ctx context.Context, productID uuid.UUID) ([]string, error) {
	if err := r.ensureFresh(ctx); err != nil {
		return nil, err
	}
	p := r.patterns.Load()
	if p == nil {
		return nil, nil
	}
	out := make([]string, 0, 4)
	for _, cp := range *p {
		if cp.productID == productID {
			out = append(out, cp.raw)
		}
	}
	return out, nil
}

// PatternCount 返回当前快照中已编译的 pattern 数量；供 healthz / debug 端点使用。
func (r *Registry) PatternCount() int {
	if p := r.patterns.Load(); p != nil {
		return len(*p)
	}
	return 0
}
