package provider

import (
	"context"
	"fmt"
	"strconv"

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

	// 小时策略应用到 pm_hourly_anchors / pm_hourly_values /
	// pm_group_metrics_hourly；兼容视图不能传给 TimescaleDB policy API。
	// 原始稀疏表由 worker 做水位安全清理，daily/weekly/monthly 由
	// cleanup_runner cron 按天数清理，均不经此 applier。
	// 运行态应用改由 SysConfigService 的有错误返回应用器驱动，不能沿用只写 warn 的 listener。
	applier := retention.NewPMRetentionApplier(c.TsPool, logger)

	// 启动期预热缓存并主动对齐 policy。失败只影响启动期同步，后续保存/重试仍会纠正。
	ctx := context.Background()
	if err := svc.Reload(ctx); err != nil {
		logger.Warn("retention initial Reload failed (using defaults)", zap.Error(err))
	}
	if err := applier.ApplyAllWithError(ctx, svc.GetAll(ctx), retention.AllKeys()); err != nil {
		logger.Warn("retention initial policy apply failed", zap.Error(err))
	}

	// 保存后的 PM policy 必须经有错误返回的应用器确认；失败状态由 SysConfigService
	// 持久化并由后台执行器重试，不允许用缓存失效 hook 报告“已生效”。
	if c.SysConfigSvc != nil {
		for _, key := range retention.AllKeys() {
			key := key
			c.SysConfigSvc.RegisterValidator(retention.Category, string(key), func(value string) error {
				days, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("%s must be an integer number of days: %w", key, err)
				}
				return retention.ValidateDays(days)
			})
		}
		c.SysConfigSvc.RegisterApplyHandler(retention.Category, "pm_retention", func(ctx context.Context, _ admin.ConfigApplyWork) (map[string]any, error) {
			current, err := svc.ReloadStrict(ctx)
			if err != nil {
				return nil, err
			}
			if err := applier.ApplyAllWithError(ctx, current, retention.AllKeys()); err != nil {
				return nil, err
			}
			return map[string]any{
				"raw_15min_days": current[retention.KeyRaw15MinDays],
				"hourly_days":    current[retention.KeyHourlyDays],
			}, nil
		})
	} else {
		logger.Warn("SysConfigSvc not wired, retention policy won't apply on sys_configs save")
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
