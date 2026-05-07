package provider

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

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
	return nil
}
