package provider

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	alarmdef "github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader"
	"github.com/omcgo/omcgo/internal/core/dictloader"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/quicksettings"
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
//   - DictLoaderConfig.AutoLoadOnStartup=false 时跳过 DB 字典 LoadOnce，仅创建+注册 Loader；
//     quicksettings 因为只存在进程内存中，仍需在启动时加载
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

	// MML 命令目录(mml_commands/groups/sub_fields)+ standard_params 为 seed-once、DB 单一权威源:
	// catalogloader 已下线删除;mmlstandardloader 仅作为 omcctl 离线解析引擎(一次性 SQL seed 导入,
	// 不在启动期/Registry 加载)。详见 CLAUDE §5.3、seed/000029-000030。
	_ = mmlstandardloader.LoaderName // 显式引用防止 import 被 goimports 移除

	// 仅注册下列 5 个 dictloader Loader（MML 不在其列）。
	for _, ld := range []dictloader.Loader{paramLoader, indicatorLoader, alarmLoader, productLoader, quickSettingsLoader} {
		if err := registry.Register(ld); err != nil {
			return fmt.Errorf("register %s loader: %w", ld.Name(), err)
		}
	}

	// 启动期加载：5 分钟超时为安全上限（实际 1.5 万行级别 < 30s）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if !c.Cfg.DictLoader.AutoLoadOnStartup {
		t0 := time.Now()
		rep, err := quickSettingsLoader.LoadOnce(ctx)
		logger.Info("dictload skipped DB startup load; quick-settings loaded only",
			zap.Int("rows", rep.RowsAffected),
			zap.Int("files_loaded", rep.FilesLoaded),
			zap.Int("files_skipped", rep.FilesSkipped),
			zap.Duration("duration", time.Since(t0)),
			zap.Error(err))
		if err != nil {
			return fmt.Errorf("load quick-settings on startup: %w", err)
		}
		return nil
	}

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
	// v2.3 catalog 单源化：mml-catalog Loader 启动注入已下线，
	// 数据由 seed/000152 + seed/000155 唯一注入，避免与 Loader 并行撕裂。
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
