package provider

import (
	"context"
	"strconv"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
)

// issue #319：MinIO 原始件 ILM 保留期可配 wiring。
//
// MinIO 原始文件桶（pm-files / mr-files）的过期天数此前硬编码 14 天，与 DB 侧可配保留期
// （pm.retention.raw_15min_days）不同源——DB 调到 60 天，MinIO 仍 14 天就把原始件删了。
// 本模块把 MinIO ILM 天数接到 sys_configs（category=minio.retention, key=raw_object_days，
// 默认 60），启动期按实配值幂等重设 lifecycle，并经 SysConfigSavedHook 热加载：前端/脚本改
// sys_configs 后无需重启即对 pm-files/mr-files 重新 SetBucketLifecycle（仿 pm-retention 模式）。
//
// 注：EnsureBuckets 启动期已用 DefaultRawFileRetentionDays 兜底设过一次；本模块随后覆盖成
// sys_configs 实配值。acs/worker 进程无本模块（无 SysConfigSvc），沿用默认值即可。

const (
	minioRetentionCategory    = "minio.retention"
	minioRetentionRawDaysKey  = "raw_object_days"
	minioRetentionDefaultDays = minioinfra.DefaultRawFileRetentionDays
	minioRetentionMinDays     = 1
	minioRetentionMaxDays     = 3650
)

// initMinIOILMModule 在 ModuleGraph 中作为 "minio-ilm" 模块初始化，依赖 admin（拿 SysConfigSvc）。
func initMinIOILMModule(c *Container) error {
	logger := c.Logger.Named("minio-ilm")

	if c.MinIO == nil {
		logger.Warn("MinIO client not wired; skip ILM retention config")
		return nil
	}

	sysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	buckets := []string{c.Cfg.MinIO.Buckets.PMFiles, c.Cfg.MinIO.Buckets.MRFiles}

	apply := func(ctx context.Context) {
		days := readMinIORetentionDays(ctx, sysConfigRepo, logger)
		if err := minioinfra.ApplyRawFileLifecycle(ctx, c.MinIO, buckets, days); err != nil {
			logger.Warn("apply raw-file lifecycle failed", zap.Int("days", days), zap.Error(err))
			return
		}
		logger.Info("MinIO raw-file ILM applied", zap.Int("days", days), zap.Strings("buckets", buckets))
	}

	// 启动期按 sys_configs 实配值重设（EnsureBuckets 已用默认值兜底，这里覆盖成实配值）。
	apply(context.Background())

	// 热加载：category=minio.retention 保存后对原始件桶重新应用 lifecycle。
	if c.SysConfigSvc != nil {
		c.SysConfigSvc.RegisterSavedHook(func(ctx context.Context, category string) {
			if category != minioRetentionCategory {
				return
			}
			apply(ctx)
		})
	} else {
		logger.Warn("SysConfigSvc not wired; MinIO ILM won't auto-reload on sys_configs save")
	}
	return nil
}

// readMinIORetentionDays 读取 sys_configs 的 MinIO 原始件保留天数，缺失/非法回落默认值（1..3650）。
func readMinIORetentionDays(ctx context.Context, repo admin.SysConfigRepository, logger *zap.Logger) int {
	row, err := repo.GetByKey(ctx, minioRetentionCategory, minioRetentionRawDaysKey)
	if err != nil {
		logger.Warn("read minio.retention.raw_object_days failed; using default",
			zap.Int("default", minioRetentionDefaultDays), zap.Error(err))
		return minioRetentionDefaultDays
	}
	if row == nil {
		return minioRetentionDefaultDays
	}
	days, perr := strconv.Atoi(row.Value)
	if perr != nil || days < minioRetentionMinDays || days > minioRetentionMaxDays {
		logger.Warn("invalid minio.retention.raw_object_days; using default",
			zap.String("value", row.Value), zap.Int("default", minioRetentionDefaultDays))
		return minioRetentionDefaultDays
	}
	return days
}
