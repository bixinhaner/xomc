// Package router 提供"按设备 → 产品 → 平台公式"的 KPI 路由（T-0164-P1 / G1）。
//
// 设计依据：docs/project/plan-T-0164-P1-kpi-routing.md / docs/design/pm-kpi-pipeline-improvements.md §4.1。
//
// 路由链条：
//
//	device.serial_number
//	    ↓ devices.product_class
//	ProductRegistry.MatchProductClass(productClass)
//	    ↓ Product { IndicatorPlatform, IndicatorDeviceType }
//	rela_platform_indicator_formula_{enb,gsm,gnb} → KPI 子集
//	perf_indicators_{enb,gsm,gnb}                 → counter / kpi 元数据
//
// 缓存层级：
//   - L1 sync 内存（hashicorp/golang-lru），按 product_id 索引；
//   - L2 Redis（[[redis_cache]] 文件，cache_version 协议跨实例失效）；
//   - DB 兜底（formulas + indicators 两次 SQL）。
//
// 这样改造后，"基站只算它该算的指标"由 DB 元数据声明决定，
// 不再依赖 carrier 适配器代码里的硬编码 KPI 列表。
package router

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
	"github.com/omcgo/omcgo/internal/product"
)

// ── 路由结果类型 ─────────────────────────────────────────────────────────

// KPIRoute 是 LookupByDevice 返回的不可变路由快照。
type KPIRoute struct {
	ProductID           uuid.UUID
	IndicatorPlatform   string
	IndicatorDeviceType indicator.DeviceType
	Counters            []CounterDef
	KPIs                []KPIDef
}

// CounterDef 描述一个 PM 计数器（is_counter='1' 的 perf_indicators 行）。
type CounterDef struct {
	IndicatorID string
	Name        string // perf_indicators_*.en_name
	ReportKey   string // perf_indicators_*.report_key（PM-P2：解析侧白名单据此建键，把上报名翻成编号 IndicatorID）
	StatisType  string // sum/avg/max/pct/...
}

// KPIDef 描述一个 KPI（is_counter='0' 的 perf_indicators 行 + arithmetic 公式）。
type KPIDef struct {
	IndicatorID  string
	Name         string   // perf_indicators_*.en_name（亦用作 model.KPIValue.KPIName）
	StatisType   string
	Formula      string   // PM-P3：perf_indicators_*.arithmetic（编号公式，如 (C000060011+C000060022)/1000）
	Dependencies []string // 公式里引用的 counter 编号（与编号化后 pm_metrics.metric_path 一致，用于 KPIEngine 取数）
}

// ── 错误信号 ─────────────────────────────────────────────────────────────

// ErrProductNotMatched 设备 productClass 未匹配任何 product_class_patterns。
// 调用侧（KPIEngine）应捕获并跳过该设备 KPI 计算（log warn）。
var ErrProductNotMatched = errors.New("device productClass not matched to any product")

// ErrInvalidProductMetadata product 缺 IndicatorPlatform / IndicatorDeviceType 配置。
// 通常意味着 products.xml 配置不完整；视作"该设备无 KPI 可算"，返回空路由。
var ErrInvalidProductMetadata = errors.New("product indicator_platform / indicator_device_type missing or invalid")

// ── 依赖契约（mock 友好）─────────────────────────────────────────────────

// DeviceLookup 按 SN 反查设备。最小消费 model.Device.ProductClass 字段。
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// ProductMatcher 按 productClass 正则匹配产品装配件。
//
// 真实实现是 *product.Registry；返回 product.ErrOrphan 时 Router 转译为 ErrProductNotMatched。
type ProductMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// IndicatorLister 按 ID 集合反查 perf_indicators_{dt} 行。
type IndicatorLister interface {
	ListByIDs(ctx context.Context, dt indicator.DeviceType, ids []string) ([]*indicator.PerfIndicator, error)
}

// FormulaLister 按 platform_name 反查 rela_platform_indicator_formula_{dt} 行。
type FormulaLister interface {
	ListByPlatform(ctx context.Context, dt indicator.DeviceType, platformName string) ([]*indicator.PlatformFormula, error)
}

// L2Cache 是 KPIRoute 的分布式缓存。
//
// 命中 → 返回 (*KPIRoute, nil)；miss → 返回 (nil, nil)；错误 → (nil, err)（Router 容忍并降级到 DB）。
type L2Cache interface {
	Get(ctx context.Context, productID uuid.UUID) (*KPIRoute, error)
	Put(ctx context.Context, route *KPIRoute) error
}

// ── Router ──────────────────────────────────────────────────────────────

const defaultL1Size = 1024

// Options 构造 Router 的可选参数。所有字段允许零值（采用默认）。
type Options struct {
	L1Size  int     // L1 LRU 容量，<= 0 取 defaultL1Size
	L2Cache L2Cache // nil → 纯 L1 + DB
	Metrics *Metrics
	Logger  *zap.Logger
}

// Router 是 LookupByDevice 的入口对象。线程安全，可被多 goroutine 共享。
type Router struct {
	devices    DeviceLookup
	products   ProductMatcher
	indicators IndicatorLister
	formulas   FormulaLister
	l1         *lru.Cache[uuid.UUID, *KPIRoute]
	l2         L2Cache
	metrics    *Metrics
	logger     *zap.Logger
}

// New 构造 Router。前 4 个参数必填；opts 控制 L1/L2/metrics/logger。
func New(
	devices DeviceLookup,
	products ProductMatcher,
	indicators IndicatorLister,
	formulas FormulaLister,
	opts Options,
) (*Router, error) {
	if devices == nil || products == nil || indicators == nil || formulas == nil {
		return nil, errors.New("router.New: nil dependency")
	}
	size := opts.L1Size
	if size <= 0 {
		size = defaultL1Size
	}
	cache, err := lru.New[uuid.UUID, *KPIRoute](size)
	if err != nil {
		return nil, fmt.Errorf("router.New: build L1 cache: %w", err)
	}
	metrics := opts.Metrics
	if metrics == nil {
		metrics = NewMetrics(nil)
	}
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Router{
		devices:    devices,
		products:   products,
		indicators: indicators,
		formulas:   formulas,
		l1:         cache,
		l2:         opts.L2Cache,
		metrics:    metrics,
		logger:     logger.Named("kpi.router"),
	}, nil
}

// LookupByDevice 返回设备所属产品声明支持的 KPI 子集。
//
// 错误语义：
//   - ErrProductNotMatched：设备 productClass 未配 pattern；KPIEngine 应跳过该设备
//   - ErrInvalidProductMetadata：product 元数据残缺；KPIEngine 应跳过
//   - 其它 error：Repo/DB 异常；调用侧按需重试或记账
func (r *Router) LookupByDevice(ctx context.Context, deviceSN string) (*KPIRoute, error) {
	dev, err := r.devices.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return nil, fmt.Errorf("lookup device %q: %w", deviceSN, err)
	}
	if dev == nil {
		return nil, fmt.Errorf("device %q not found", deviceSN)
	}

	match, err := r.products.MatchProductClass(ctx, dev.ProductClass)
	if err != nil {
		if errors.Is(err, product.ErrOrphan) {
			r.metrics.miss("orphan")
			return nil, ErrProductNotMatched
		}
		return nil, fmt.Errorf("match productClass %q: %w", dev.ProductClass, err)
	}
	prod := match.Product
	if prod == nil {
		return nil, ErrInvalidProductMetadata
	}

	// L1
	if route, ok := r.l1.Get(prod.ID); ok {
		r.metrics.hit("L1")
		return route, nil
	}

	// L2
	if r.l2 != nil {
		route, err := r.l2.Get(ctx, prod.ID)
		if err != nil {
			r.logger.Warn("L2 get failed; falling back to DB",
				zap.String("product_id", prod.ID.String()),
				zap.Error(err))
		} else if route != nil {
			r.l1.Add(prod.ID, route)
			r.metrics.hit("L2")
			return route, nil
		}
	}

	// DB 兜底
	route, err := r.loadFromDB(ctx, prod)
	if err != nil {
		return nil, err
	}
	r.l1.Add(prod.ID, route)
	if r.l2 != nil {
		if err := r.l2.Put(ctx, route); err != nil {
			r.logger.Warn("L2 put failed (non-fatal)",
				zap.String("product_id", prod.ID.String()),
				zap.Error(err))
		}
	}
	r.metrics.hit("DB")
	return route, nil
}

// InvalidateAll 清空 L1。L2 失效由 cache.BumpVersion 跨实例广播（见 redis_cache.go）。
// 仅供运维 / 管理 API 调用。
func (r *Router) InvalidateAll() { r.l1.Purge() }

// loadFromDB 拉两次 SQL（formulas + indicators）并装配 KPIRoute。
func (r *Router) loadFromDB(ctx context.Context, prod *product.Product) (*KPIRoute, error) {
	if prod.IndicatorPlatform == "" || prod.IndicatorDeviceType == "" {
		r.metrics.miss("invalid_metadata")
		r.logger.Warn("product missing indicator metadata; returning empty route",
			zap.String("product_id", prod.ID.String()),
			zap.String("product_name", prod.Name))
		return &KPIRoute{ProductID: prod.ID}, nil
	}
	dt, err := indicator.ParseDeviceType(prod.IndicatorDeviceType)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProductMetadata, err)
	}

	formulas, err := r.formulas.ListByPlatform(ctx, dt, prod.IndicatorPlatform)
	if err != nil {
		return nil, fmt.Errorf("list formulas by platform %q: %w", prod.IndicatorPlatform, err)
	}

	indicatorIDs := uniqueIndicatorIDs(formulas)
	indicators, err := r.indicators.ListByIDs(ctx, dt, indicatorIDs)
	if err != nil {
		return nil, fmt.Errorf("list indicators by IDs (%d): %w", len(indicatorIDs), err)
	}

	return assembleRoute(prod, dt, indicators, r.logger), nil
}

func uniqueIndicatorIDs(formulas []*indicator.PlatformFormula) []string {
	seen := make(map[string]struct{}, len(formulas))
	out := make([]string, 0, len(formulas))
	for _, f := range formulas {
		if _, dup := seen[f.IndicatorID]; dup {
			continue
		}
		seen[f.IndicatorID] = struct{}{}
		out = append(out, f.IndicatorID)
	}
	return out
}

// assembleRoute 把 indicators 拼成 KPIRoute。
//
// PM-P3：公式内容从标准名版 formula 表切到编号版 `perf_indicators_*.arithmetic`，
// 与编号化落库的计数器（pm_metrics.metric_path）全程编号、直接对上、零翻译。
// formulas 参数仍由 loadFromDB 用于决定「该平台支持哪些 KPI」的成员资格
// （uniqueIndicatorIDs → ListByIDs），故此处 indicators 只含该平台引用的指标；
// assembleRoute 不再读 formula 内容。
//
// 拆分规则（perf_indicators_*.is_counter）：
//   - is_counter='1'：CounterDef（无公式）
//   - is_counter='0' 且 arithmetic 非空：KPIDef（Formula = arithmetic 编号公式）
//   - is_counter='0' 但 arithmetic 为空：跳过 + WARN（platform 已选但编号公式缺失，运维问题）
func assembleRoute(
	prod *product.Product,
	dt indicator.DeviceType,
	indicators []*indicator.PerfIndicator,
	logger *zap.Logger,
) *KPIRoute {
	route := &KPIRoute{
		ProductID:           prod.ID,
		IndicatorPlatform:   prod.IndicatorPlatform,
		IndicatorDeviceType: dt,
		Counters:            make([]CounterDef, 0),
		KPIs:                make([]KPIDef, 0),
	}

	for _, ind := range indicators {
		statisType := derefStr(ind.StatisType)
		if ind.IsCounter == "1" {
			route.Counters = append(route.Counters, CounterDef{
				IndicatorID: ind.ID,
				Name:        ind.EnName,
				ReportKey:   derefStr(ind.ReportKey),
				StatisType:  statisType,
			})
			continue
		}
		// is_counter='0' → KPI，公式取编号版 arithmetic。
		arithmetic := derefStr(ind.Arithmetic)
		if arithmetic == "" {
			logger.Warn("indicator marked KPI but arithmetic (numbered formula) empty; skip",
				zap.String("indicator_id", ind.ID),
				zap.String("platform", prod.IndicatorPlatform),
				zap.String("device_type", string(dt)))
			continue
		}
		// Formula 与 Dependencies 同源：都从 arithmetic 提取（同一个 expr.Parse().Identifiers()），
		// 否则 KPIEngine 会按标准名取数 → 编号化后 pm_metrics 是编号 → 取不到 counter。
		deps := extractFormulaDeps(arithmetic)
		route.KPIs = append(route.KPIs, KPIDef{
			IndicatorID:  ind.ID,
			Name:         ind.EnName,
			StatisType:   statisType,
			Formula:      arithmetic,
			Dependencies: deps,
		})
	}
	return route
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// extractFormulaDeps 从公式字符串提取去重的 counter 标识符。
//
// 复用 pm/kpi/expr 的解析器作为标识符的单一真源：返回的依赖名与 KPIEngine 求值时
// 在 counter map 里查找的 key 一字不差，从而避免"提取规则"与"求值规则"漂移
// （历史 bug：手写 token 扫描把 3GPP 点分名 MAC.RachSuccess 切成 MAC / RachSuccess，
// 导致 QueryForKPI 用错误 metric_path 查不到 counter → 全库零 KPI）。
//
// expr 是零依赖叶子包，router 直接 import 不构成 pm/kpi → router → indicator → kpi 循环。
// 公式解析失败时返回 nil（该 KPI 在 KPIEngine 侧也会被同样的解析失败跳过并 log）。
func extractFormulaDeps(formula string) []string {
	f, err := expr.Parse(formula)
	if err != nil {
		return nil
	}
	return f.Identifiers()
}
