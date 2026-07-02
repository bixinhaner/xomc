package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
	"github.com/omcgo/omcgo/internal/mr"
)

// Cleaner 实现 PRD §3 AC-7：每日清理超期 MR 文件的 mr_files PG 元数据行。
//
// #798：MinIO 对象本体的过期删除已并入「原始件 ILM」（sys_configs
// minio.retention.raw_object_days，由 cmd/app/provider 的 minio_ilm 模块对 pm-files/
// mr-files 两个桶统一下发 MinIO 生命周期规则，由 MinIO 服务端自动过期，界面改动即时生效、
// 不需要重启进程）——Cleaner 自身不再直接调用 MinIO ListObjects/RemoveObject。
// 本 Cleaner 现在只做一件事：@daily 按同一保留天数动态计算 cutoff，同步删除 mr_files 表
// 中 collect_time 早于 cutoff 的行，避免这些行长期指向已被 MinIO 自动删除的对象、成为死记录。
type Cleaner struct {
	store         mr.MRStore // 可 nil；nil 时 Start 直接禁用（无表可清）
	bucket        string     // 仅用于日志/指标标识，不再参与任何 MinIO API 调用
	retentionDays RetentionDaysLookup
	logger        *zap.Logger
	metrics       *Metrics // 可 nil

	cron    *cron.Cron
	entryID cron.EntryID
	mu      sync.Mutex
	started bool
}

// SetMetrics 注入 Prometheus 指标（可选）。
func (c *Cleaner) SetMetrics(m *Metrics) { c.metrics = m }

// RetentionDaysLookup 返回当前生效的 MinIO 原始件 ILM 保留天数（sys_configs
// minio.retention.raw_object_days），供 Cleaner 动态计算 mr_files PG 清理 cutoff（#798）。
// 由调用方（cmd/app/provider wiring）负责读 sys_configs 并做非法值兜底，与
// cmd/app/provider/minio_ilm.go 的 readMinIORetentionDays 读同一份配置、同一套校验规则，
// 保证两端天数一致。返回 <=0 时 Cleaner 自行回落 minioinfra.DefaultRawFileRetentionDays。
type RetentionDaysLookup func(ctx context.Context) int

// CleanerConfig 控制 cleaner 行为。
type CleanerConfig struct {
	Bucket   string
	Schedule string // cron 表达式，默认 "@daily"
}

// Defaults 给出 CleanerConfig 兜底值。
func (c CleanerConfig) Defaults() CleanerConfig {
	out := c
	if out.Schedule == "" {
		out.Schedule = "@daily"
	}
	return out
}

// NewCleaner 创建 cleaner。store 为 nil 时 Start 立即返回 nil（禁用清理）。
// retentionDays 为 nil 或返回非正数时，RunOnce 回落 minioinfra.DefaultRawFileRetentionDays。
func NewCleaner(store mr.MRStore, cfg CleanerConfig, retentionDays RetentionDaysLookup, logger *zap.Logger) *Cleaner {
	if logger == nil {
		logger = zap.NewNop()
	}
	c := cfg.Defaults()
	return &Cleaner{
		store:         store,
		bucket:        c.Bucket,
		retentionDays: retentionDays,
		logger:        logger.Named("mr-cleaner"),
	}
}

// Start 注册 cron。幂等。store 缺失时禁用并返回 nil。
func (c *Cleaner) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return nil
	}
	if c.store == nil {
		c.logger.Warn("mr cleaner disabled: store missing")
		return nil
	}
	c.cron = cron.New(cron.WithLocation(time.UTC))
	id, err := c.cron.AddFunc("@daily", func() {
		removed, err := c.RunOnce(ctx)
		if err != nil {
			c.logger.Error("mr cleaner tick failed", zap.Error(err))
			return
		}
		c.logger.Info("mr cleaner tick completed", zap.Int("removed", removed))
	})
	if err != nil {
		c.cron = nil
		return fmt.Errorf("register mr-cleaner cron: %w", err)
	}
	c.entryID = id
	c.cron.Start()
	c.started = true
	c.logger.Info("mr cleaner started", zap.String("bucket", c.bucket))
	return nil
}

// Stop 停止 cron。幂等。
func (c *Cleaner) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started || c.cron == nil {
		return
	}
	stopCtx := c.cron.Stop()
	<-stopCtx.Done()
	c.cron = nil
	c.started = false
}

// resolveRetentionDays 返回当前生效的保留天数：优先取注入的 lookup（实时读 sys_configs
// minio.retention.raw_object_days），lookup 为 nil 或返回非正数时回落默认值 60 天。
func (c *Cleaner) resolveRetentionDays(ctx context.Context) int {
	if c.retentionDays != nil {
		if days := c.retentionDays(ctx); days > 0 {
			return days
		}
	}
	return minioinfra.DefaultRawFileRetentionDays
}

// RunOnce 触发一次清理：按当前生效保留天数计算 cutoff，删除 mr_files 表中
// collect_time 早于 cutoff 的行。返回成功删除的行数。
//
// #798 前该方法还负责 MinIO 对象本体的 ListObjects/RemoveObject；那部分职责已移交 MinIO
// 生命周期规则（cmd/app/provider/minio_ilm.go），本方法不再直接操作 MinIO。
func (c *Cleaner) RunOnce(ctx context.Context) (int, error) {
	if c.store == nil {
		return 0, nil
	}
	days := c.resolveRetentionDays(ctx)
	cutoff := time.Now().UTC().AddDate(0, 0, -days)

	deleted, err := c.store.DeleteFilesBefore(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete mr_files PG rows: %w", err)
	}
	if deleted > 0 {
		c.logger.Info("deleted mr_files PG rows",
			zap.Int64("count", deleted), zap.Time("cutoff", cutoff), zap.Int("retention_days", days))
		c.metrics.AddFilesCleaned("pg", float64(deleted))
	}
	return int(deleted), nil
}
