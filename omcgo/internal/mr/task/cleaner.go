package task

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/mr"
)

// Cleaner 实现 PRD §3 AC-7：每日清理超期 MR 文件。
//
// MR 文件在 MinIO 中的实际 key 形如 `{carrier}/yyyy/MM/dd/{deviceSN}/{filename}`
// （由 storage.ObjectPath 生成，category=""）。本 cleaner：
//   - 列出 MR bucket 下所有对象
//   - 按 key 的日期段（yyyy/MM/dd）筛出 cutoff 之前的文件
//   - 删 MinIO 对象 + 同步删 mr_files PG 行
//
// 设计原则：
//   - 单 tick 删除数有上限（maxRemovePerRun）防止短时大量 MinIO API 调用
//   - 删除按 batch 调 RemoveObjects（minio 批量 API）减少 round-trip
//   - mr_files 表删除目前为占位（需要 mr.MRStore.DeleteFilesBefore 方法支持）；
//     Phase 4 接入后取消 TODO
type Cleaner struct {
	minio    *minio.Client
	bucket   string
	saveDays int
	store    mr.MRStore // 可 nil；nil 时只清 MinIO 不清 PG（向后兼容单进程部署）
	logger   *zap.Logger
	metrics  *Metrics // 可 nil

	cron    *cron.Cron
	entryID cron.EntryID
	mu      sync.Mutex
	started bool
}

// SetMetrics 注入 Prometheus 指标（可选）。
func (c *Cleaner) SetMetrics(m *Metrics) { c.metrics = m }

// CleanerConfig 控制 cleaner 行为。
type CleanerConfig struct {
	Bucket   string
	SaveDays int
	Schedule string // cron 表达式，默认 "@daily"
}

// Defaults 给出 CleanerConfig 兜底值。
func (c CleanerConfig) Defaults() CleanerConfig {
	out := c
	if out.SaveDays <= 0 {
		out.SaveDays = 3
	}
	if out.Schedule == "" {
		out.Schedule = "@daily"
	}
	return out
}

// NewCleaner 创建 cleaner。minio + bucket 为空时 Start 立即返回 nil（禁用清理）。
// store 可 nil（向后兼容）；非 nil 时同步清 mr_files PG 行。
func NewCleaner(m *minio.Client, store mr.MRStore, cfg CleanerConfig, logger *zap.Logger) *Cleaner {
	if logger == nil {
		logger = zap.NewNop()
	}
	c := cfg.Defaults()
	return &Cleaner{
		minio:    m,
		bucket:   c.Bucket,
		saveDays: c.SaveDays,
		store:    store,
		logger:   logger.Named("mr-cleaner"),
	}
}

// Start 注册 cron。幂等。bucket / minio 任一缺失时禁用并返回 nil。
func (c *Cleaner) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return nil
	}
	if c.minio == nil || c.bucket == "" {
		c.logger.Warn("mr cleaner disabled: minio or bucket missing")
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
	c.logger.Info("mr cleaner started",
		zap.String("bucket", c.bucket),
		zap.Int("save_days", c.saveDays),
	)
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

// 单次清理上限。文档 §11 默认保留 3 天，正常稳态新增/淘汰每天约等量；
// 设 5000 / 天上限足以覆盖 10 万站规模 × 4 文件 / 站 / 周期的极端突发。
const maxRemovePerRun = 5000

// objectDateRe — DEPRECATED 2026-05-26：MR 存储路径已改为 {sn}/{filename}（无日期段）。
// 历史文件可能仍带 yyyy/MM/dd 路径，正则保留用于"历史文件兜底"。新文件走 LastModified 判定。
var objectDateRe = regexp.MustCompile(`(?:^|/)(\d{4})/(\d{2})/(\d{2})/`)

// RunOnce 触发一次清理。返回成功删除的对象数。
//
// 出错策略：
//   - ListObjects 出错：直接返回 err（无法继续）
//   - RemoveObject 单个出错：log + 计数，继续下一个（错误不向上抛）
//   - 解析 key 日期失败：跳过该对象（不计错误，因为可能是后来加的非标 key）
func (c *Cleaner) RunOnce(ctx context.Context) (int, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -c.saveDays)
	removed := 0

	objCh := c.minio.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{
		Recursive: true,
	})
	for obj := range objCh {
		if obj.Err != nil {
			return removed, fmt.Errorf("list mr bucket objects: %w", obj.Err)
		}
		if removed >= maxRemovePerRun {
			c.logger.Info("mr cleaner reached per-run limit; remainder picked next tick",
				zap.Int("limit", maxRemovePerRun))
			break
		}
		// 2026-05-26：新 MR 存储路径 {sn}/{filename} 不含日期段；以 obj.LastModified
		// 为准（MinIO 自带，可靠）。历史文件路径含 yyyy/MM/dd 时也仍能正确判定，
		// 因 LastModified 与上传时间一致。
		mtime := obj.LastModified
		if mtime.IsZero() {
			// 兜底：MinIO 返空（理论上不应发生），尝试从 path 解析
			if dt, ok := parseDateFromKey(obj.Key); ok {
				mtime = dt
			} else {
				continue
			}
		}
		if !mtime.Before(cutoff) {
			continue
		}
		if err := c.minio.RemoveObject(ctx, c.bucket, obj.Key,
			minio.RemoveObjectOptions{}); err != nil {
			c.logger.Warn("remove expired MR object failed",
				zap.String("key", obj.Key), zap.Error(err))
			continue
		}
		removed++
	}
	c.metrics.AddFilesCleaned("minio", float64(removed))

	// 同步删 PG 端 mr_files 元数据行。失败仅记 warn，不抹掉 MinIO 删除成果。
	// mr_records（TimescaleDB hypertable）由独立 retention policy 管，不在这里删。
	if c.store != nil {
		deleted, err := c.store.DeleteFilesBefore(ctx, cutoff)
		if err != nil {
			c.logger.Warn("delete mr_files PG rows failed (MinIO already cleaned)",
				zap.Time("cutoff", cutoff), zap.Error(err))
		} else if deleted > 0 {
			c.logger.Info("deleted mr_files PG rows",
				zap.Int64("count", deleted), zap.Time("cutoff", cutoff))
			c.metrics.AddFilesCleaned("pg", float64(deleted))
		}
	}

	return removed, nil
}

// parseDateFromKey 从 key 里提取 yyyy/MM/dd 并转为 UTC midnight time。
// 找不到合法日期返回 (zero, false)。
func parseDateFromKey(key string) (time.Time, bool) {
	m := objectDateRe.FindStringSubmatch(key)
	if len(m) != 4 {
		return time.Time{}, false
	}
	dt, err := time.Parse("2006/01/02", fmt.Sprintf("%s/%s/%s", m[1], m[2], m[3]))
	if err != nil {
		return time.Time{}, false
	}
	return dt, true
}
