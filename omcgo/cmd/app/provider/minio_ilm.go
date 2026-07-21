package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// issue #319：MinIO 原始件 ILM 保留期可配 wiring。
//
// MinIO 原始文件桶（pm-files / mr-files）的过期天数此前硬编码 14 天，与 DB 侧可配保留期
// （pm.retention.raw_15min_days）不同源——DB 调到 60 天，MinIO 仍 14 天就把原始件删了。
// 本模块把 MinIO ILM 天数接到 sys_configs（category=minio.retention, key=raw_object_days，
// 默认 60），启动期按实配值幂等重设 lifecycle，并经 SysConfigSavedHook 热加载：前端/脚本改
// sys_configs 后无需重启即对 pm-files/mr-files 重新 SetBucketLifecycle，并记录可重试状态。
//
// 本模块是 lifecycle 的唯一写入者；acs/worker 的通用建桶流程不得覆盖这里的实配值。

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

	sysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	buckets := []string{c.Cfg.MinIO.Buckets.PMFiles, c.Cfg.MinIO.Buckets.MRFiles}

	apply := func(ctx context.Context) (int, error) {
		if c.MinIO == nil {
			return 0, fmt.Errorf("MinIO client is not wired")
		}
		days, err := readMinIORetentionDays(ctx, sysConfigRepo)
		if err != nil {
			return 0, err
		}
		if err := minioinfra.ApplyRawFileLifecycle(ctx, c.MinIO, buckets, days); err != nil {
			return 0, fmt.Errorf("apply raw-file lifecycle: %w", err)
		}
		logger.Info("MinIO raw-file ILM applied", zap.Int("days", days), zap.Strings("buckets", buckets))
		return days, nil
	}

	// 启动期按 sys_configs 实配值设置；缺省时使用 readMinIORetentionDays 的产品默认值。
	if c.MinIO != nil {
		if _, err := apply(context.Background()); err != nil {
			logger.Warn("initial MinIO raw-file ILM apply failed", zap.Error(err))
		}
	} else {
		logger.Warn("MinIO client not wired; ILM saves will remain failed until the dependency recovers")
	}

	// 热加载：category=minio.retention 保存后对原始件桶重新应用 lifecycle。
	if c.SysConfigSvc != nil {
		c.SysConfigSvc.RegisterValidator(minioRetentionCategory, minioRetentionRawDaysKey, validateMinIORetentionDays)
		c.SysConfigSvc.RegisterApplyHandler(minioRetentionCategory, "minio_raw_file_lifecycle", func(ctx context.Context, _ admin.ConfigApplyWork) (map[string]any, error) {
			days, err := apply(ctx)
			if err != nil {
				return nil, err
			}
			return map[string]any{"raw_object_days": days}, nil
		})
	} else {
		logger.Warn("SysConfigSvc not wired; MinIO ILM won't auto-reload on sys_configs save")
	}
	return nil
}

// readMinIORetentionDays reads the configured lifecycle duration. A missing
// value is an intentional default; malformed values and read failures must be
// surfaced to the persistent apply workflow.
func readMinIORetentionDays(ctx context.Context, repo admin.SysConfigRepository) (int, error) {
	row, err := repo.GetByKey(ctx, minioRetentionCategory, minioRetentionRawDaysKey)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			return minioRetentionDefaultDays, nil
		}
		return 0, fmt.Errorf("read minio retention days: %w", err)
	}
	if row == nil {
		return minioRetentionDefaultDays, nil
	}
	days, err := strconv.Atoi(row.Value)
	if err != nil {
		return 0, fmt.Errorf("parse minio retention days: %w", err)
	}
	if err := validateMinIORetentionDayValue(days); err != nil {
		return 0, err
	}
	return days, nil
}

// readMinIORetentionDaysOrDefault keeps the cleaner's periodic job resilient
// to a transient configuration-store error. Configuration saves use the strict
// function above; the already-running cleaner may safely retain its default
// schedule until the next cycle.
func readMinIORetentionDaysOrDefault(ctx context.Context, repo admin.SysConfigRepository, logger *zap.Logger) int {
	days, err := readMinIORetentionDays(ctx, repo)
	if err == nil {
		return days
	}
	logger.Warn("read minio retention days for cleaner failed; using default",
		zap.Int("default", minioRetentionDefaultDays), zap.Error(err))
	return minioRetentionDefaultDays
}

func validateMinIORetentionDays(value string) error {
	days, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("MinIO raw object retention must be an integer number of days: %w", err)
	}
	return validateMinIORetentionDayValue(days)
}

func validateMinIORetentionDayValue(days int) error {
	if days < minioRetentionMinDays || days > minioRetentionMaxDays {
		return fmt.Errorf("MinIO raw object retention days must be between %d and %d", minioRetentionMinDays, minioRetentionMaxDays)
	}
	return nil
}
