package stationlog

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

// issue #320：基站日志保留。此前仅有「故障日志全局 20 文件 FIFO 配额」（无时间维度），
// 现按用户要求两者并存：
//   - 按时间保留（默认 60 天）：worker 每日 cron 清理 station_fault_logs + station_running_logs
//     中 created_at 早于 cutoff 的记录（删 MinIO 对象 + 软删 PG 行）。
//   - 文件数配额（默认 20，可配 / 0=禁用）：事件驱动的 enforceFaultLogQuota 兜底，防单设备
//     短时间刷爆。
//
// 配置全部落 sys_configs（category=stationlog.retention），TTL 缓存避免每文件查库。

const (
	// RetentionCategory 是 sys_configs 中基站日志保留策略的 category。
	RetentionCategory = "stationlog.retention"
	// KeyMaxRetentionDays 时间保留天数键；KeyMaxFileCount 故障日志文件数配额键（0=不限）。
	KeyMaxRetentionDays = "max_retention_days"
	KeyMaxFileCount     = "max_file_count"

	// DefaultMaxRetentionDays 默认按时间保留 60 天（用户场景）。
	DefaultMaxRetentionDays = 60
	// DefaultMaxFileCount 默认文件数配额沿用历史 FaultLogMaxCount=20。
	DefaultMaxFileCount = FaultLogMaxCount

	minRetentionDays = 1
	maxRetentionDays = 3650
	maxFileCount     = 1_000_000

	policyTTL = 60 * time.Second
)

// ConfigLookup 读 sys_configs 单值（value, found）。
type ConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// RetentionPolicy 提供基站日志保留参数（天数 + 文件数配额），从 sys_configs 读取并 TTL 缓存。
// 并发安全。零值不可用，须经 NewRetentionPolicy 构造。
type RetentionPolicy struct {
	lookup ConfigLookup
	logger *zap.Logger

	mu       sync.Mutex
	days     int
	count    int
	loadedAt time.Time
}

// NewRetentionPolicy 构造保留策略。lookup 为 nil 时恒返回默认值。
func NewRetentionPolicy(lookup ConfigLookup, logger *zap.Logger) *RetentionPolicy {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RetentionPolicy{
		lookup: lookup,
		logger: logger.Named("stationlog.retention"),
		days:   DefaultMaxRetentionDays,
		count:  DefaultMaxFileCount,
	}
}

// MaxRetentionDays 返回当前生效的时间保留天数。
func (p *RetentionPolicy) MaxRetentionDays(ctx context.Context) int {
	d, _ := p.get(ctx)
	return d
}

// MaxFileCount 返回当前生效的故障日志文件数配额（0 表示不限）。
func (p *RetentionPolicy) MaxFileCount(ctx context.Context) int {
	_, c := p.get(ctx)
	return c
}

func (p *RetentionPolicy) get(ctx context.Context) (days, count int) {
	// 快路径：仅短暂持锁读缓存，命中即返回（不在锁内做 DB IO）。
	p.mu.Lock()
	if !p.loadedAt.IsZero() && time.Since(p.loadedAt) < policyTTL {
		days, count = p.days, p.count
		p.mu.Unlock()
		return days, count
	}
	p.mu.Unlock()

	// 慢路径：锁外读 sys_configs（DB IO），避免持锁阻塞其他并发 caller
	// （enforceFaultLogQuota 在事件回调里并发调用）。边界期可能两 caller 同时读库，
	// 是良性冗余，远好于持锁串行化。
	newDays := p.readInt(ctx, KeyMaxRetentionDays, DefaultMaxRetentionDays, minRetentionDays, maxRetentionDays)
	newCount := p.readInt(ctx, KeyMaxFileCount, DefaultMaxFileCount, 0, maxFileCount)

	p.mu.Lock()
	p.days, p.count, p.loadedAt = newDays, newCount, time.Now()
	days, count = p.days, p.count
	p.mu.Unlock()
	return days, count
}

func (p *RetentionPolicy) readInt(ctx context.Context, key string, def, lo, hi int) int {
	if p.lookup == nil {
		return def
	}
	v, found := p.lookup(ctx, RetentionCategory, key)
	if !found {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < lo || n > hi {
		p.logger.Warn("invalid stationlog retention value; using default",
			zap.String("key", key), zap.String("value", v), zap.Int("default", def))
		return def
	}
	return n
}

// cleanupStore 是 CleanupRunner 需要的仓库子集（便于单测注入 fake）。*PgRepository 满足。
type cleanupStore interface {
	ListExpired(ctx context.Context, cutoff time.Time, limit int) ([]*LogFile, error)
	MarkDeleted(ctx context.Context, id uuid.UUID) error
}

// objectRemover 是 CleanupRunner 删 MinIO 对象需要的子集。*minio.Client 满足。
type objectRemover interface {
	RemoveObject(ctx context.Context, bucket, object string, opts minio.RemoveObjectOptions) error
}

// JobTypeStationLogCleanup 是基站日志时间清理任务的 asyncjob 类型主键。
const JobTypeStationLogCleanup = "stationlog_retention_cleanup"

// CleanupRunner 是基站日志按时间保留清理任务（asyncjob.JobRunner，#320）。每日 cron 触发，
// 删除 fault + running 两表中 created_at 早于 cutoff 的记录（MinIO 对象 + PG 软删）。
// 失败只 warn、不阻塞其它记录/表。
type CleanupRunner struct {
	fault   cleanupStore
	running cleanupStore
	remover objectRemover
	policy  *RetentionPolicy
	logger  *zap.Logger
}

// NewCleanupRunner 构造基站日志清理 runner。
func NewCleanupRunner(fault, running cleanupStore, remover objectRemover, policy *RetentionPolicy, logger *zap.Logger) *CleanupRunner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CleanupRunner{
		fault:   fault,
		running: running,
		remover: remover,
		policy:  policy,
		logger:  logger.Named("stationlog.retention.cleanup"),
	}
}

// JobType 实现 asyncjob.JobRunner。
func (r *CleanupRunner) JobType() string { return JobTypeStationLogCleanup }

// Run 清理两表过保留期记录。
func (r *CleanupRunner) Run(ctx context.Context, _ *asyncjob.Job) (json.RawMessage, error) {
	days := r.policy.MaxRetentionDays(ctx)
	if days < minRetentionDays {
		r.logger.Warn("retention days too small; skip cleanup", zap.Int("days", days))
		return json.Marshal(map[string]any{"skipped": true, "days": days})
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	faultDel := r.cleanupTable(ctx, "station_fault_logs", r.fault, cutoff)
	runningDel := r.cleanupTable(ctx, "station_running_logs", r.running, cutoff)

	r.logger.Info("stationlog retention cleanup done",
		zap.Int("days", days), zap.Time("cutoff", cutoff),
		zap.Int("fault_deleted", faultDel), zap.Int("running_deleted", runningDel))
	return json.Marshal(map[string]any{
		"days": days, "fault_deleted": faultDel, "running_deleted": runningDel,
	})
}

// cleanupTable 分批清理单表过期记录，防长事务。每批先删 MinIO 对象再软删 PG 行；
// ListExpired 已过滤 is_deleted=false，软删后下批不再返回，故循环收敛。maxBatches 兜底极端积压。
func (r *CleanupRunner) cleanupTable(ctx context.Context, table string, store cleanupStore, cutoff time.Time) int {
	if store == nil {
		return 0
	}
	const batchSize = 500
	const maxBatches = 40

	total := 0
	for batch := 0; batch < maxBatches; batch++ {
		select {
		case <-ctx.Done():
			return total
		default:
		}
		expired, err := store.ListExpired(ctx, cutoff, batchSize)
		if err != nil {
			r.logger.Warn("list expired failed; stop table", zap.String("table", table), zap.Error(err))
			return total
		}
		if len(expired) == 0 {
			return total
		}
		progressed := 0
		for _, old := range expired {
			if r.remover != nil && old.Bucket != "" && old.ObjectPath != "" {
				if rmErr := r.remover.RemoveObject(ctx, old.Bucket, old.ObjectPath, minio.RemoveObjectOptions{}); rmErr != nil {
					r.logger.Warn("remove expired log object",
						zap.String("table", table), zap.String("id", old.ID.String()),
						zap.String("path", old.ObjectPath), zap.Error(rmErr))
				}
			}
			if mErr := store.MarkDeleted(ctx, old.ID); mErr != nil {
				r.logger.Warn("mark expired log deleted",
					zap.String("table", table), zap.String("id", old.ID.String()), zap.Error(mErr))
				continue
			}
			total++
			progressed++
		}
		// 全批 MarkDeleted 都失败（progressed==0）→ 下批必返回同样记录，避免空转直接停。
		if progressed == 0 || len(expired) < batchSize {
			return total
		}
	}
	r.logger.Warn("stationlog cleanup hit max batches; remaining cleaned next run", zap.String("table", table))
	return total
}
