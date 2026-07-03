package provider

import (
	"context"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/pm/retention"
)

// T-0164-P2 / G2 + #825 PM 保留策略 wiring。
//
// 装配链：
//   1. admin.PgSysConfigRepository（已有，复用）
//   2. adminSysConfigReader adapter（本文件，把 admin.SysConfigRepository 适配为 retention.SysConfigReader 窄接口）
//   3. retention.NewService(reader, logger) → c.PMRetentionSvc
//   4. retention.NewPMRetentionApplier(c.TsPool, logger) 注册 listener，驱动 TimescaleDB retention policy 更新
//   5. Reload(ctx) 预热缓存（首次 Reload changed 包含全部键，ApplyAll 自动触发一次对齐）
//   6. c.SysConfigSvc.RegisterSavedHook(svc.OnSysConfigSaved) 挂热重载监听

// adminSysConfigReader 实现 retention.SysConfigReader 窄接口，
// 包裹 admin.SysConfigRepository 避免 retention 包反向依赖 admin。
type adminSysConfigReader struct {
	inner admin.SysConfigRepository
}

func (a *adminSysConfigReader) GetByKey(ctx context.Context, category, key string) (*retention.SysConfigRow, error) {
	row, err := a.inner.GetByKey(ctx, category, key)
	if err != nil || row == nil {
		return nil, err
	}
	return &retention.SysConfigRow{Value: row.Value, ValueType: row.ValueType}, nil
}

// initPMRetentionModule 在 ModuleGraph 中作为 "pm-retention" 模块初始化，
// 依赖 pm 和 admin 两个模块（pm 仅占位避免拓扑漏挂；admin 提供 SysConfigSvc）。
func initPMRetentionModule(c *Container) error {
	logger := c.Logger.Named("pm-retention")

	sysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	reader := &adminSysConfigReader{inner: sysConfigRepo}

	svc := retention.NewService(reader, logger)

	// 注册 PMRetentionApplier listener：UI 保存或进程启动时驱动 TimescaleDB retention policy 更新。
	// hypertable 只有 pm_metrics / pm_metrics_hourly / pm_group_metrics_hourly（#825）；
	// daily/weekly/monthly 为普通表，由 cleanup_runner cron 按天数清理，不经此 applier。
	applier := retention.NewPMRetentionApplier(c.TsPool, logger)
	svc.RegisterListener(applier.ApplyAll)

	// 启动期预热缓存。首次 Reload 时 cache 从空→有值，changed 包含全部 5 键，
	// ApplyAll 自动被触发，对齐 TimescaleDB policy 与 sys_configs，无需额外调用。
	ctx := context.Background()
	if err := svc.Reload(ctx); err != nil {
		logger.Warn("retention initial Reload failed (using defaults)", zap.Error(err))
	}

	// 挂 SavedHook：BatchUpsert 提交 category='pm.retention' 后触发 svc.OnSysConfigSaved → Reload。
	// 与 securityPolicy / periodicSyncPolicy 一致的模式（admin.go L150 / modules.go L385）。
	if c.SysConfigSvc != nil {
		c.SysConfigSvc.RegisterSavedHook(svc.OnSysConfigSaved)
	} else {
		logger.Warn("SysConfigSvc not wired, retention.Service won't auto-reload on sys_configs save")
	}

	c.PMRetentionSvc = svc

	// 装配可观测的当前值快照便于运维排查（不算 sensitive 数据，直接打日志）。
	current := svc.GetAll(ctx)
	logger.Info("PM retention service initialized",
		zap.Int("raw_15min_days", current[retention.KeyRaw15MinDays]),
		zap.Int("hourly_days", current[retention.KeyHourlyDays]),
		zap.Int("daily_days", current[retention.KeyDailyDays]),
		zap.Int("weekly_days", current[retention.KeyWeeklyDays]),
		zap.Int("monthly_days", current[retention.KeyMonthlyDays]),
	)
	return nil
}
