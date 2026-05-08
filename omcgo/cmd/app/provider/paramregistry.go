package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/dictloader"
)

// initParamRegistryModule 构造 ParamRegistry 并 Refresh（T-0098 P2-02）。
//
// 启动语义：
//  1. 构造 PgRepository（读 param_mappings / discovered_param_mappings）
//  2. 构造 RedisCache（L2）—— Container.Redis 为 nil 时退化 NopCache（开发/单进程）
//  3. 注册 Prometheus 指标
//  4. Refresh：清 L1 + BumpVersion 通知其他实例。**不预热**：mappings 数据量适中（~5K rows
//     按 9 paramModel 分布），按需 read-through 即可，避免启动期长尾
//
// 依赖关系：
//   - dictload：保证 param_mappings / discovered_param_mappings 表结构 + 默认数据已就绪
//   - productregistry：注入为 productGetter，供 ParamRegistry 反查 product.ParamModelID
//
// Feature flag：`appconfig.ParamRegistry.UseNew`
//   - false（默认）：ParamRegistry 仍构造启动，但消费者继续走旧 datamodel 栈
//   - true：消费者切换到 parammodel.Registry + Translator
//   - 切换由 P2-04..08 各自的消费者改造负责，本模块只确保 Registry 可用
func initParamRegistryModule(c *Container) error {
	logger := c.Logger.Named("parammodel")

	repo := parammodel.NewPgRepository(c.PgPool)

	var cache parammodel.Cache = parammodel.NopCache{}
	if c.Redis != nil {
		cache = parammodel.NewRedisCacheWithTTL(
			c.Redis,
			c.Cfg.ParamRegistry.DefaultTTL,
			c.Cfg.ParamRegistry.DiscoveredTTL,
		)
	}

	metrics := parammodel.NewRegistryMetrics(c.MetricsReg)

	// ProductRegistry 满足 productGetter 接口（GetProductByID）。
	// 启动顺序：dictload → productregistry → paramregistry，由 ModuleGraph 保证。
	if c.ProductRegistry == nil {
		return fmt.Errorf("paramregistry: ProductRegistry 未初始化（依赖 productregistry 先于 paramregistry）")
	}
	registry := parammodel.NewRegistry(repo, cache, c.ProductRegistry, metrics, logger)

	// 启动期 Refresh —— 仅清 L1 + BumpVersion；本进程 L1 本就空，BumpVersion 用于
	// 多实例环境通知其他实例对齐版本号。30s 超时足够。
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t0 := time.Now()
	if err := registry.Refresh(ctx); err != nil {
		// 多实例下 BumpVersion 失败不致命（其他实例本次错过失效信号）；
		// 本实例 Registry 已可用，记 WARN 后继续启动。
		logger.Warn("ParamRegistry initial refresh failed (non-fatal)",
			zap.Duration("duration", time.Since(t0)),
			zap.Error(err))
	} else {
		logger.Info("ParamRegistry loaded",
			zap.Duration("duration", time.Since(t0)),
			zap.Bool("use_new_flag", c.Cfg.ParamRegistry.UseNew))
	}

	c.ParamRegistry = registry

	// T-0098 P2-03：IntersectService 与 Registry 共用同一个 PgRepository（读+写）
	// 与同一个 productGetter；写完后通过 Registry.InvalidateProduct 失效缓存。
	intersectMetrics := parammodel.NewIntersectMetrics(c.MetricsReg)
	c.ParamIntersect = parammodel.NewIntersectService(
		repo,
		repo,
		c.ProductRegistry,
		registry,
		intersectMetrics,
		logger,
	)

	// T-0098 P3-02：ParamModel handler — 共用 PgRepository 走 CRUD/查询，registry
	// 用于写后 Invalidate / Translate。Reloader 适配 dictloader.Registry.ReloadOne。
	c.ParamModelRepo = repo
	c.ParamModelHandler = parammodel.NewHandler(
		repo,
		registry,
		&paramModelReloader{reg: c.DictLoaderRegistry},
		logger,
	)
	return nil
}

// paramModelReloader 把 dictloader.Registry.ReloadOne(...)(Report, error)
// 适配为 parammodel.Reloader 期望的 ReloadOne(ctx, name) error。
type paramModelReloader struct {
	reg *dictloader.Registry
}

func (r *paramModelReloader) ReloadOne(ctx context.Context, name string) error {
	if r.reg == nil {
		return errors.New("dictloader registry not wired")
	}
	_, err := r.reg.ReloadOne(ctx, name)
	return err
}
