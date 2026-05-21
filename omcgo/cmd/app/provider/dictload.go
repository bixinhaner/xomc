package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	alarmdef "github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader"
	"github.com/omcgo/omcgo/internal/core/dictloader"
	"github.com/omcgo/omcgo/internal/mml/catalogloader"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/quicksettings"

	"path/filepath"
)

// initDictLoadModule 初始化 T-0098 P1-06 字典加载模块。
//
// 启动语义（设计 §7.2）：
//   1. 创建 4 个 Loader（paramModel / indicator / alarmDefinition / product）
//   2. 注册到 dictloader.Registry，供未来管理 API 触发 ReloadOne（Phase 3）
//   3. 启动期 LoadOnce 编排 — 三引用字典并行（Phase 1），products 串行后置（Phase 2）确保引用校验有数据
//
// 依赖关系（实施计划 §3.1）：
//   ParamModel ┐
//   Indicator  ├→ Product（校验三引用：paramModel.name / indicator platform / alarm ne_type）
//   AlarmDef   ┘
//
// 失败语义：
//   - DictLoaderConfig.AutoLoadOnStartup=false 时跳过 LoadOnce，仅创建+注册 Loader
//   - 任一字典 Loader 失败 → 启动失败（DoD：字典是 RC 必备）
//   - Product Loader 单 product 校验失败 → 跳过 + ERROR 日志（设计 §4.5），不阻塞启动
func initDictLoadModule(c *Container) error {
	logger := c.Logger.Named("dictload")

	registry := dictloader.NewRegistry(c.Cfg.DictLoader.LoadConcurrency, logger)
	c.DictLoaderRegistry = registry

	baseDir := c.Cfg.DictLoader.XMLBaseDir
	if baseDir == "" {
		baseDir = "data" // 兼容缺省 — 各 Loader 子配置 Directory 仍相对此根
	}

	paramLoader := parammodel.NewLoader(c.PgPool, c.Cfg.DictLoader.ParamModel, baseDir, logger)
	indicatorLoader := indicator.NewLoader(c.PgPool, c.Cfg.DictLoader.Indicator, baseDir, logger)
	alarmLoader := alarmdef.NewLoader(c.PgPool, c.Cfg.DictLoader.AlarmDefinition, baseDir, logger)
	productLoader := product.NewLoader(c.PgPool, c.Cfg.DictLoader.Product, baseDir, logger)

	// T-0138：「快速设置」分组元数据 Loader（进程内存 cache，无 DB）
	quickSettingsRegistry := quicksettings.NewRegistry()
	c.QuickSettingsRegistry = quickSettingsRegistry
	quickSettingsLoader := quicksettings.NewLoader(c.Cfg.DictLoader.QuickSettings, baseDir, quickSettingsRegistry, logger)

	// MML 控制台 v2.3 catalog Loader（dictloader 模式第 5 个 Loader）
	// 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.7
	mmlCatalogDir := c.Cfg.DictLoader.MMLCatalog.Directory
	if mmlCatalogDir == "" {
		mmlCatalogDir = catalogloader.DefaultDirectory
	}
	// MML_V2_SCHEMA=true 切到 v2 路径（spec v2.3 §R-1：18 章节顶层 + chapter:* group_code +
	// tree_node_refs + link_health）。默认不设，保持 v1 行为。注意：此 env 同时驱动
	// modules.go 的 mml.WithV2Mode（BuildTree 过滤），两处必须共用同一 toggle，
	// 否则 Loader 写 v2 行而 BuildTree 仍按 v1 视图过滤，UI 会看到混乱。
	mmlUseV2Schema := os.Getenv("MML_V2_SCHEMA") == "true"
	c.MMLV2Schema = mmlUseV2Schema
	mmlLoaderOpts := []catalogloader.Option{}
	if mmlUseV2Schema {
		mmlLoaderOpts = append(mmlLoaderOpts, catalogloader.WithV2Schema(true))
		logger.Info("mml-catalog: v2 schema ENABLED (MML_V2_SCHEMA=true)")
	}
	mmlCatalogLoader := catalogloader.NewLoader(c.PgPool, filepath.Join(baseDir, mmlCatalogDir), logger, mmlLoaderOpts...)
	c.MMLCatalogLoader = mmlCatalogLoader
	// T-0123-P0：mmlstandardloader 启动期注册下线。
	// 改为一次性 SQL seed 导入（migrations/seed/000096_mml_standard_params_import.sql，
	// 由 `omcctl mml import-standard-xml` 离线生成）。
	// 后续 catalog 通过 admin UI 增删改（migrations/000095 + admin_*.go）维护，DB 是单一权威源。
	// 包 mmlstandardloader/ 整包保留作为 omcctl 工具的解析引擎，不再 ModuleGraph 注册。
	// 关联：docs/design/mml-restore-old-interaction-plan-20260514.md §M.6 / R-206 mitigation
	_ = mmlstandardloader.LoaderName // 显式引用防止 import 被 goimports 移除

	for _, ld := range []dictloader.Loader{paramLoader, indicatorLoader, alarmLoader, productLoader, quickSettingsLoader, mmlCatalogLoader} {
		if err := registry.Register(ld); err != nil {
			return fmt.Errorf("register %s loader: %w", ld.Name(), err)
		}
	}

	if !c.Cfg.DictLoader.AutoLoadOnStartup {
		logger.Info("dictload skipped startup load (AutoLoadOnStartup=false)")
		return nil
	}

	// 启动期加载：5 分钟超时为安全上限（实际 1.5 万行级别 < 30s）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTotal := time.Now()

	// Phase 1：三引用字典并行
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		t0 := time.Now()
		rep, err := paramLoader.LoadOnce(gctx)
		logger.Info("param-model load done",
			zap.Int("rows", rep.RowsAffected),
			zap.Int("files_loaded", rep.FilesLoaded),
			zap.Int("files_skipped", rep.FilesSkipped),
			zap.Duration("duration", time.Since(t0)),
			zap.Error(err))
		return err
	})
	g.Go(func() error {
		t0 := time.Now()
		rep, err := indicatorLoader.LoadOnce(gctx)
		logger.Info("indicator load done",
			zap.Int("rows", rep.RowsAffected),
			zap.Int("files_loaded", rep.FilesLoaded),
			zap.Int("files_skipped", rep.FilesSkipped),
			zap.Duration("duration", time.Since(t0)),
			zap.Error(err))
		return err
	})
	g.Go(func() error {
		t0 := time.Now()
		rep, err := alarmLoader.LoadOnce(gctx)
		logger.Info("alarm-definition load done",
			zap.Int("rows", rep.RowsAffected),
			zap.Int("files_loaded", rep.FilesLoaded),
			zap.Int("files_skipped", rep.FilesSkipped),
			zap.Duration("duration", time.Since(t0)),
			zap.Error(err))
		return err
	})
	g.Go(func() error {
		t0 := time.Now()
		rep, err := quickSettingsLoader.LoadOnce(gctx)
		logger.Info("quick-settings load done",
			zap.Int("rows", rep.RowsAffected),
			zap.Int("files_loaded", rep.FilesLoaded),
			zap.Int("files_skipped", rep.FilesSkipped),
			zap.Duration("duration", time.Since(t0)),
			zap.Error(err))
		return err
	})
	g.Go(func() error {
		t0 := time.Now()
		rep, err := mmlCatalogLoader.LoadOnce(gctx)
		logger.Info("mml-catalog load done",
			zap.Int("rows", rep.RowsAffected),
			zap.Int("files_loaded", rep.FilesLoaded),
			zap.Int("non_fatal_errors", len(rep.Errors)),
			zap.Duration("duration", time.Since(t0)),
			zap.Error(err))
		return err
	})
	// T-0123-P0：mml-standard 启动期 LoadOnce 下线。
	// MML 命令字典通过 migrations/seed/000096_mml_standard_params_import.sql 一次性 DB 导入；
	// 后续 admin UI 维护。详 docs/design/mml-restore-old-interaction-plan-20260514.md §M.6。
	if err := g.Wait(); err != nil {
		return fmt.Errorf("dictload phase 1 (3 dicts in parallel): %w", err)
	}

	// Phase 2：products（依赖 Phase 1 结果做引用校验）
	t0 := time.Now()
	rep, err := productLoader.LoadOnce(ctx)
	logger.Info("product load done",
		zap.Int("rows", rep.RowsAffected),
		zap.Int("files_loaded", rep.FilesLoaded),
		zap.Int("non_fatal_errors", len(rep.Errors)),
		zap.Duration("duration", time.Since(t0)),
		zap.Error(err))
	if err != nil {
		// product loader 内部已对单 product 校验失败做 skip+ERROR；这里到达说明 fatal（解析失败 / DB 错误）
		return fmt.Errorf("dictload phase 2 (product): %w", err)
	}

	logger.Info("dictload completed",
		zap.Duration("total_duration", time.Since(startTotal)),
		zap.Strings("loaders", registry.Names()))
	return nil
}
