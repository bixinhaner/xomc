package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/dictloader"
	"github.com/omcgo/omcgo/internal/product"
)

// initProductRegistryModule 构造 ProductRegistry 并加载 patterns（T-0098 P2-01）。
//
// 启动语义：
//  1. 创建 PgRepository（读 products / product_class_patterns / 软引用查询表）
//  2. 创建 RedisCache（L2）— 若 Container.Redis 为 nil 则注入 NopCache 退化为纯 L1
//  3. 注册 Prometheus 指标
//  4. 调 Refresh 加载 patterns 全集 + 三引用启动期校验（WARN-only，不阻塞启动）
//
// 依赖关系：必须在 dictload 完成之后运行（products / alarm_definitions / perf_indicators_*
// 表必须已由 P1-06 Loader 写入 DB）。在 router.go 的 ModuleGraph 中显式 Depends ["dictload"]。
func initProductRegistryModule(c *Container) error {
	logger := c.Logger.Named("product")

	repo := product.NewPgRepository(c.PgPool)

	var cache product.Cache = product.NopCache{}
	if c.Redis != nil {
		cache = product.NewRedisCache(c.Redis)
	}

	metrics := product.NewRegistryMetrics(c.MetricsReg)

	registry := product.NewRegistry(repo, cache, metrics, logger)

	// 启动期 Refresh 设 30s 超时（patterns 表通常 < 100 行，绝大多数环境亚秒完成）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t0 := time.Now()
	if err := registry.Refresh(ctx); err != nil {
		return fmt.Errorf("ProductRegistry refresh: %w", err)
	}
	logger.Info("ProductRegistry loaded",
		zap.Int("patterns", registry.PatternCount()),
		zap.Duration("duration", time.Since(t0)))

	// 启动期软引用校验：WARN-only，不阻塞启动（与 P1-06 product Loader 一致）
	if rep, err := registry.ValidateReferences(ctx); err != nil {
		logger.Warn("ProductRegistry reference validation failed (non-fatal)", zap.Error(err))
	} else {
		logger.Info("ProductRegistry reference validation completed",
			zap.Int("total_products", rep.TotalProducts),
			zap.Int("warned_products", len(rep.WarnedProducts)))
	}

	c.ProductRegistry = registry

	// T-0098 P3-01：Product handler — DiscoveredCleaner 由 parammodel.PgRepository 实现
	// （DeleteDiscoveredAll 已在 P3-02 添加）；rematcher 暂用 handler 内置实现。
	c.ProductRepo = repo
	c.ProductHandler = product.NewHandler(
		repo,
		registry,
		&productDiscoveredCleaner{repo: parammodel.NewPgRepository(c.PgPool)},
		nil, // rematcher: nil → handler 走内置 fallback 用 Registry 重算
		&productReloader{reg: c.DictLoaderRegistry},
		logger,
	)
	return nil
}

// productDiscoveredCleaner 实现 product.DiscoveredCleaner（避免反向 import）。
type productDiscoveredCleaner struct {
	repo *parammodel.PgRepository
}

func (c *productDiscoveredCleaner) DeleteDiscoveredAll(ctx context.Context, productID uuid.UUID) (int64, error) {
	return c.repo.DeleteDiscoveredAll(ctx, productID)
}

// productReloader 把 dictloader.Registry.ReloadOne(ctx,name)(Report,error)
// 适配为 product.Reloader 期望的 ReloadOne(ctx,name) error。
type productReloader struct {
	reg *dictloader.Registry
}

func (r *productReloader) ReloadOne(ctx context.Context, name string) error {
	if r.reg == nil {
		return errors.New("dictloader registry not wired")
	}
	_, err := r.reg.ReloadOne(ctx, name)
	return err
}
