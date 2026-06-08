package parammodel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/product"
)

// productGetter 是 Registry 反查 product → paramModelID 的最小依赖。
//
// 生产环境由 ProductRegistry 满足；测试可注入 stub 以避免起 product 包。
// 通过接口而非直接依赖 *product.Registry，便于单元测试隔离。
type productGetter interface {
	GetProductByID(ctx context.Context, id uuid.UUID) (*product.Product, error)
}

// discoveredKey 是 (productID, swVersion) 的复合 L1 键。
type discoveredKey struct {
	productID uuid.UUID
	swVersion string
}

// Registry 是参数映射的查询入口（设计 §1.4 / §1.5）。
//
// 缓存层级：
//   - L1 sync.Map（按 paramModelID 与 (productID, swVersion) 各一）
//   - L2 Cache（Redis）
//   - DB（param_mappings / discovered_param_mappings）
//
// 查询路径：
//   - GetByProduct(productID, swVersion)：discovered 三层 → 未命中降级 default 三层 → ErrNoMapping
//   - GetByParamModel(paramModelID)：仅 default 三层 → ErrNoMapping
//
// 注意：本 Registry **不消费** AppConfig.ParamRegistry.UseNew flag——dual-stack
// 期间 Registry 始终可用；旧/新栈切换由消费者（P2-04..08）按需读 flag 决定。
type Registry struct {
	repo     Repository
	cache    Cache
	products productGetter
	logger   *zap.Logger
	metrics  *registryMetrics

	defaultByModel     sync.Map // map[uuid.UUID][]ParamMapping
	discoveredByDevice sync.Map // map[discoveredKey][]ParamMapping

	// cacheVersion 是本进程已采纳的 L2 cache_version 快照。
	// 热路径 ensureFresh 比对它与 Cache.GetVersion()：版本前进即丢弃本进程 L1，
	// 这是 ACS（独立进程）能"及时"看到 app 侧增删改映射的关键（与 product.Registry 同协议）。
	cacheVersion atomic.Int64
}

// NewRegistry 构造一个 Registry。
//
//   - cache 为 nil 等价于注入 NopCache（退化为纯 L1 + DB）
//   - metrics 为 nil 注册到匿名 prometheus.Registry，可用但不可观测
//   - logger 为 nil 使用 zap.NewNop()
//   - products 为 nil 时 GetByProduct/Translator 返回 ErrProductGetterUnset；
//     GetByParamModel 仍可用（典型场景：admin handler 单测）
func NewRegistry(repo Repository, cache Cache, products productGetter, metrics *registryMetrics, logger *zap.Logger) *Registry {
	if cache == nil {
		cache = NopCache{}
	}
	if metrics == nil {
		metrics = NewRegistryMetrics(nil)
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Registry{
		repo:     repo,
		cache:    cache,
		products: products,
		logger:   logger.Named("parammodel.registry"),
		metrics:  metrics,
	}
}

// GetByProduct 按 (productID, swVersion) 取映射集，discovered 优先，未命中降级 default。
//
// 返回的 MappingSet.Source 标记本次实际命中的来源；调用方可据此打日志/打指标。
// product.ParamModelID 为 nil → ErrNoParamModel。
// discovered 与 default 均为空 → ErrNoMapping。
func (r *Registry) GetByProduct(ctx context.Context, productID uuid.UUID, swVersion string) (*MappingSet, error) {
	t0 := time.Now()
	defer func() { r.metrics.lookupDuration.Observe(time.Since(t0).Seconds()) }()

	// 热路径版本自检：别的进程（典型：app 改了映射并 BumpVersion）一旦推进版本，
	// 这里就丢弃本进程全部 L1，使后续查询 read-through 到最新 L2/DB。
	r.ensureFresh(ctx)

	if r.products == nil {
		return nil, ErrProductGetterUnset
	}

	// Step 1: discovered（L1 → L2 → DB）
	discovered, err := r.loadDiscovered(ctx, productID, swVersion)
	if err != nil {
		return nil, err
	}
	if len(discovered) > 0 {
		r.metrics.lookupDiscovered()
		// 反查 product 仅为回填 ParamModelID，便于上游链路追踪。
		// product 不存在或 paramModelID 为 nil 都不阻塞 discovered 路径。
		paramModelID := r.lookupParamModelID(ctx, productID)
		return &MappingSet{
			ProductID:       productID,
			ParamModelID:    paramModelID,
			SoftwareVersion: swVersion,
			Source:          MappingSourceDiscovered,
			Mappings:        discovered,
		}, nil
	}

	// Step 2: 降级 default。需先反查 product.ParamModelID
	prod, err := r.products.GetProductByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("lookup product %s: %w", productID, err)
	}
	if prod == nil {
		r.metrics.lookupMiss()
		return nil, ErrNoMapping
	}
	if prod.ParamModelID == nil {
		r.metrics.lookupMiss()
		return nil, ErrNoParamModel
	}

	defaults, err := r.loadDefault(ctx, *prod.ParamModelID)
	if err != nil {
		return nil, err
	}
	if len(defaults) == 0 {
		r.metrics.lookupMiss()
		return nil, ErrNoMapping
	}

	r.metrics.lookupDefault()
	r.logger.Info("ParamRegistry fallback to default",
		zap.String("product_id", productID.String()),
		zap.String("software_version", swVersion),
		zap.String("param_model_id", prod.ParamModelID.String()),
	)
	return &MappingSet{
		ProductID:       productID,
		ParamModelID:    *prod.ParamModelID,
		SoftwareVersion: swVersion,
		Source:          MappingSourceDefault,
		Mappings:        defaults,
	}, nil
}

// GetByParamModel 直接按 paramModelID 取默认映射集（仅 default 三层），
// 用于 P3-02 admin handler 等不带 device context 的查询。
func (r *Registry) GetByParamModel(ctx context.Context, paramModelID uuid.UUID) (*MappingSet, error) {
	t0 := time.Now()
	defer func() { r.metrics.lookupDuration.Observe(time.Since(t0).Seconds()) }()

	r.ensureFresh(ctx)

	defaults, err := r.loadDefault(ctx, paramModelID)
	if err != nil {
		return nil, err
	}
	if len(defaults) == 0 {
		r.metrics.lookupMiss()
		return nil, ErrNoMapping
	}
	r.metrics.lookupDefault()
	return &MappingSet{
		ParamModelID: paramModelID,
		Source:       MappingSourceDefault,
		Mappings:     defaults,
	}, nil
}

// Translator 是 GetByProduct + NewTranslator 的组合捷径；调用方典型用法。
func (r *Registry) Translator(ctx context.Context, productID uuid.UUID, swVersion string) (*Translator, error) {
	set, err := r.GetByProduct(ctx, productID, swVersion)
	if err != nil {
		return nil, err
	}
	return NewTranslator(set, r.metrics, r.logger), nil
}

// InvalidateProduct 清除某 (productID, swVersion) 的 L1 + L2 缓存。
//
// 由 P2-03 IntersectService 在写完 discovered_param_mappings 后调用，
// 确保下次 GetByProduct 拿到最新 discovered。
func (r *Registry) InvalidateProduct(ctx context.Context, productID uuid.UUID, swVersion string) error {
	r.discoveredByDevice.Delete(discoveredKey{productID: productID, swVersion: swVersion})
	if err := r.cache.InvalidateDiscovered(ctx, productID, swVersion); err != nil {
		// L2 失败不致命：L1 已删，下次必然击穿到 DB
		r.logger.Warn("L2 invalidate discovered failed (non-fatal)",
			zap.String("product_id", productID.String()),
			zap.String("software_version", swVersion),
			zap.Error(err))
		return err
	}
	// 跨进程失效信号：让 ACS 等独立进程在热路径 ensureFresh 后丢弃各自 L1。
	r.bumpVersionLocally(ctx)
	return nil
}

// InvalidateParamModel 清除某 paramModelID 的 default 缓存（L1 + L2）。
// XML 重导入或 P3-02 改动 default mapping 后调用。
func (r *Registry) InvalidateParamModel(ctx context.Context, paramModelID uuid.UUID) error {
	r.defaultByModel.Delete(paramModelID)
	if err := r.cache.InvalidateDefault(ctx, paramModelID); err != nil {
		r.logger.Warn("L2 invalidate default failed (non-fatal)",
			zap.String("param_model_id", paramModelID.String()),
			zap.Error(err))
		return err
	}
	// 跨进程失效信号（同 InvalidateProduct）：增删改单条映射后，ACS 才能"及时"
	// 下发最新 privatePath，而非等进程重启。
	r.bumpVersionLocally(ctx)
	return nil
}

// Refresh 整体清空 L1（所有 default 与 discovered 缓存）+ BumpVersion 通知其他实例。
//
// 启动期由 Provider 调一次，用于多实例部署下的版本对齐；
// 运行期偶尔调（如管理 API 触发全局刷新）。
func (r *Registry) Refresh(ctx context.Context) error {
	r.dropL1()

	newVersion, err := r.cache.BumpVersion(ctx)
	if err != nil {
		r.logger.Warn("ParamRegistry bump cache version failed (non-fatal)", zap.Error(err))
		r.metrics.refreshErr()
		return err
	}
	// 采纳自己刚 bump 的版本，避免下次 ensureFresh 把它误判为"别人改的"而冗余重载。
	r.cacheVersion.Store(newVersion)
	r.metrics.refreshOK()
	r.logger.Info("ParamRegistry refreshed", zap.Int64("version", newVersion))
	return nil
}

// ── 跨进程一致性 ────────────────────────────────────────────────────

// ensureFresh 在热路径入口比对 L2 cache_version 与本进程已采纳版本。
//
// 版本前进 → 说明别的进程（典型：app 改了映射并 BumpVersion）已使本进程 L1 过时，
// 丢弃全部 L1，让后续查询 read-through 到最新 L2/DB。这是 ACS 独立进程能"及时"
// 看到 app 侧映射增删改的关键 —— 否则 L1 sync.Map 无 TTL、热路径短路返回旧映射，
// 要等进程重启才更新。与 product.Registry.ensureFresh 同协议。
//
// 降级：GetVersion 出错只 WARN 并沿用当前快照——翻译可用性优先于强一致，一次 Redis
// 抖动不应让 CPE 会话失败。NopCache 下 GetVersion 恒 0，本方法为 no-op（无跨进程介质）。
func (r *Registry) ensureFresh(ctx context.Context) {
	version, err := r.cache.GetVersion(ctx)
	if err != nil {
		r.logger.Warn("get cache version failed; using current L1 snapshot", zap.Error(err))
		return
	}
	if version == 0 || version == r.cacheVersion.Load() {
		return
	}
	r.dropL1()
	r.cacheVersion.Store(version)
	r.logger.Info("ParamRegistry L1 invalidated by cache version change",
		zap.Int64("new_version", version))
}

// dropL1 清空两张 L1 sync.Map（default 与 discovered）。
func (r *Registry) dropL1() {
	r.defaultByModel.Range(func(k, _ any) bool { r.defaultByModel.Delete(k); return true })
	r.discoveredByDevice.Range(func(k, _ any) bool { r.discoveredByDevice.Delete(k); return true })
}

// bumpVersionLocally 递增全局 cache_version 并把新值采纳到本进程，确保本进程不会把
// 自己刚 bump 的版本当成"别人改的"而在下次 ensureFresh 冗余重载整张 L1。失败仅 WARN：
// 失效信号丢失最多让远端晚一拍，不阻塞写操作。
func (r *Registry) bumpVersionLocally(ctx context.Context) {
	newVersion, err := r.cache.BumpVersion(ctx)
	if err != nil {
		r.logger.Warn("bump cache version after invalidate failed (non-fatal)", zap.Error(err))
		return
	}
	r.cacheVersion.Store(newVersion)
}

// ── 内部缓存路径 ────────────────────────────────────────────────────

// loadDiscovered 按 L1 → L2 → DB 顺序取 discovered；命中后回填上层。
// 空切片是合法值（缓存"无 discovered"事实）。
func (r *Registry) loadDiscovered(ctx context.Context, productID uuid.UUID, swVersion string) ([]ParamMapping, error) {
	key := discoveredKey{productID: productID, swVersion: swVersion}
	if v, ok := r.discoveredByDevice.Load(key); ok {
		r.metrics.cacheHit("L1", "discovered")
		return v.([]ParamMapping), nil
	}
	if cached, err := r.cache.GetDiscovered(ctx, productID, swVersion); err != nil {
		r.logger.Warn("L2 get discovered failed; falling back to DB",
			zap.String("product_id", productID.String()),
			zap.String("software_version", swVersion),
			zap.Error(err))
	} else if cached != nil {
		r.discoveredByDevice.Store(key, cached)
		r.metrics.cacheHit("L2", "discovered")
		return cached, nil
	}

	rows, err := r.repo.ListDiscoveredMappings(ctx, productID, swVersion)
	if err != nil {
		return nil, fmt.Errorf("repo list discovered %s/%s: %w", productID, swVersion, err)
	}
	if rows == nil {
		rows = []ParamMapping{}
	}
	r.discoveredByDevice.Store(key, rows)
	if err := r.cache.SetDiscovered(ctx, productID, swVersion, rows); err != nil {
		r.logger.Warn("L2 set discovered failed (non-fatal)",
			zap.String("product_id", productID.String()),
			zap.String("software_version", swVersion),
			zap.Error(err))
	}
	r.metrics.cacheHit("DB", "discovered")
	if len(rows) > 0 {
		r.logger.Debug("ParamRegistry discovered mapping loaded",
			zap.String("product_id", productID.String()),
			zap.String("software_version", swVersion),
			zap.Int("count", len(rows)))
	}
	return rows, nil
}

// loadDefault 按 L1 → L2 → DB 顺序取 default；命中后回填上层。
func (r *Registry) loadDefault(ctx context.Context, paramModelID uuid.UUID) ([]ParamMapping, error) {
	if v, ok := r.defaultByModel.Load(paramModelID); ok {
		r.metrics.cacheHit("L1", "default")
		return v.([]ParamMapping), nil
	}
	if cached, err := r.cache.GetDefault(ctx, paramModelID); err != nil {
		r.logger.Warn("L2 get default failed; falling back to DB",
			zap.String("param_model_id", paramModelID.String()),
			zap.Error(err))
	} else if cached != nil {
		r.defaultByModel.Store(paramModelID, cached)
		r.metrics.cacheHit("L2", "default")
		return cached, nil
	}

	rows, err := r.repo.ListMappingsByParamModel(ctx, paramModelID)
	if err != nil {
		return nil, fmt.Errorf("repo list default %s: %w", paramModelID, err)
	}
	if rows == nil {
		rows = []ParamMapping{}
	}
	r.defaultByModel.Store(paramModelID, rows)
	if err := r.cache.SetDefault(ctx, paramModelID, rows); err != nil {
		r.logger.Warn("L2 set default failed (non-fatal)",
			zap.String("param_model_id", paramModelID.String()),
			zap.Error(err))
	}
	r.metrics.cacheHit("DB", "default")
	if len(rows) > 0 {
		r.logger.Info("ParamRegistry default mapping loaded",
			zap.String("param_model_id", paramModelID.String()),
			zap.Int("count", len(rows)))
	}
	return rows, nil
}

// lookupParamModelID 反查 product.ParamModelID；失败或为 nil 时返回 uuid.Nil。
// 仅供日志/元数据回填，不影响 discovered 命中路径。
func (r *Registry) lookupParamModelID(ctx context.Context, productID uuid.UUID) uuid.UUID {
	if r.products == nil {
		return uuid.Nil
	}
	prod, err := r.products.GetProductByID(ctx, productID)
	if err != nil || prod == nil || prod.ParamModelID == nil {
		return uuid.Nil
	}
	return *prod.ParamModelID
}
