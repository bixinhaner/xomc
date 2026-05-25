package asyncjob

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CronState 是 cron 调度器持久化的"上次触发"状态。
//
// 用途：worker 停机 / 重启后，调度器需要知道"上次成功触发到哪个 bucket 了"以补跑漏桶。
// robfig/cron 本身是内存调度器，停机期间所有触发完全丢失；本表是兜底持久化。
//
// 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.8 G8 cron 错过补跑
// 实施 plan：docs/project/plan-T-0164-followup-gaps.md G8-Gap-1
type CronState struct {
	JobType         string
	CronExpr        string
	LastTriggeredAt time.Time
	LastBucketEnd   *time.Time // 上次触发的 bucket 时间窗右界（半开区间 [start, end) 的 end）
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CronStateRepository 是 async_jobs_cron_state 表的持久化接口。
type CronStateRepository interface {
	// Get 按 job_type 取状态；不存在返 ErrNoCronState（首次启动场景）。
	Get(ctx context.Context, jobType string) (*CronState, error)

	// Upsert 写入 / 更新 cron 触发状态。多 worker 并发时单 SQL UPSERT 保证不冲突。
	Upsert(ctx context.Context, jobType, cronExpr string, lastTriggeredAt, lastBucketEnd time.Time) error
}

// ErrNoCronState job_type 未曾触发过（首次启动场景，调用方退化为"不补跑，只跑当前 bucket"）。
var ErrNoCronState = errors.New("asyncjob: no cron state for job_type")

// PgCronStateRepository 是 CronStateRepository 的 pgxpool 实现。
type PgCronStateRepository struct {
	pool *pgxpool.Pool
}

// NewPgCronStateRepository 创建 CronState 持久化。
func NewPgCronStateRepository(pool *pgxpool.Pool) *PgCronStateRepository {
	return &PgCronStateRepository{pool: pool}
}

var _ CronStateRepository = (*PgCronStateRepository)(nil)

func (r *PgCronStateRepository) Get(ctx context.Context, jobType string) (*CronState, error) {
	const q = `
SELECT job_type, cron_expr, last_triggered_at, last_bucket_end, created_at, updated_at
FROM async_jobs_cron_state WHERE job_type = $1`
	row := r.pool.QueryRow(ctx, q, jobType)
	var s CronState
	var bucketEnd *time.Time
	err := row.Scan(&s.JobType, &s.CronExpr, &s.LastTriggeredAt, &bucketEnd, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoCronState
		}
		return nil, fmt.Errorf("get cron state %s: %w", jobType, err)
	}
	s.LastBucketEnd = bucketEnd
	return &s, nil
}

func (r *PgCronStateRepository) Upsert(ctx context.Context, jobType, cronExpr string, lastTriggeredAt, lastBucketEnd time.Time) error {
	const q = `
INSERT INTO async_jobs_cron_state (job_type, cron_expr, last_triggered_at, last_bucket_end)
VALUES ($1, $2, $3, $4)
ON CONFLICT (job_type) DO UPDATE SET
    cron_expr         = EXCLUDED.cron_expr,
    last_triggered_at = EXCLUDED.last_triggered_at,
    last_bucket_end   = EXCLUDED.last_bucket_end`
	if _, err := r.pool.Exec(ctx, q, jobType, cronExpr, lastTriggeredAt, lastBucketEnd); err != nil {
		return fmt.Errorf("upsert cron state %s: %w", jobType, err)
	}
	return nil
}

// ── 补跑助手 ──────────────────────────────────────────────────────────────

// BucketAdvance 把上一个 bucket 的 end 推进到下一个 bucket 的 end。
//
// 不同粒度的"加一个 bucket"语义不同：
//   - hourly:  end + 1 hour
//   - daily:   end + 1 day
//   - weekly:  end + 7 days
//   - monthly: end + 1 month（按自然月，28/30/31 天变长度，必须用 AddDate(0,1,0)）
//
// 由调用方按 cron 类型注入。
type BucketAdvance func(prevEnd time.Time) time.Time

// MissedBucket 描述一个待补跑的 bucket [Start, End)。
type MissedBucket struct {
	Start time.Time
	End   time.Time
}

// CatchupMissedBuckets 列出 (lastBucketEnd, now] 区间内所有应触发的 bucket。
//
// 算法：从 lastBucketEnd 开始反复用 advance 推进到下一个 bucket 的 end，
// 只要 end <= now 就纳入补跑列表（end > now 表示"未来 bucket"，应该等下次正常 cron 触发）。
//
// 例：hourly cron，lastBucketEnd = 2026-05-23 10:00，now = 2026-05-23 14:30
//   返回 4 个 bucket：[10:00,11:00), [11:00,12:00), [12:00,13:00), [13:00,14:00)
//   (14:30 < 15:00，最后一个 [14:00,15:00) 不补，等下次 cron 自然触发)
//
// 极端情况：lastBucketEnd > now（时钟回退）或 advance 返回相同时间（不前进）→ 立即返回空切片防止死循环。
func CatchupMissedBuckets(lastBucketEnd time.Time, now time.Time, advance BucketAdvance) []MissedBucket {
	if !lastBucketEnd.Before(now) {
		return nil
	}
	var out []MissedBucket
	prevEnd := lastBucketEnd
	for {
		nextEnd := advance(prevEnd)
		if !nextEnd.After(prevEnd) {
			// advance 不前进（实现错误 / 时区异常）→ 防死循环
			return out
		}
		if nextEnd.After(now) {
			return out
		}
		out = append(out, MissedBucket{Start: prevEnd, End: nextEnd})
		prevEnd = nextEnd
		if len(out) > 10000 {
			// 安全上限（停机时间超过 10000 个 bucket 一般属配置错误，留 log warn 让运维介入）
			return out
		}
	}
}
