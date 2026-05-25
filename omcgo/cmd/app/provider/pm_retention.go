package provider

import (
	"context"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/pm/retention"
)

// T-0164-P2 / G2 PM 保留策略 wiring。
//
// 装配链：
//   1. admin.PgSysConfigRepository（已有，复用）
//   2. adminSysConfigReader adapter（本文件，把 admin.SysConfigRepository 适配为 retention.SysConfigReader 窄接口）
//   3. retention.NewService(reader, logger) → c.PMRetentionSvc
//   4. Reload(ctx) 预热缓存
//   5. c.SysConfigSvc.RegisterSavedHook(svc.OnSysConfigSaved) 挂监听（chenbo01 已留 API）
//
// G3/G5 实施时通过 c.PMRetentionSvc.RegisterListener(...) 挂 alter_compression_policy /
// alter_retention_policy 触发器（避免本包硬依赖 TimescaleDB / pgx）。

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

	// 启动期预热缓存。缺失走默认（DefaultDays），不阻塞启动。
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
