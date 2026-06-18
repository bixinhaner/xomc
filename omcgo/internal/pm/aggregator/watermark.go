package aggregator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// WatermarkLevel 标识完成水位的层级（#528 P1）。
//
// 上游卷数据分两级链式产出：先设备级（device），设备级该桶成功后再链式产出设备组级（group）。
// 两级各记一条水位，group 级完成时刻总晚于 device 级。
type WatermarkLevel string

const (
	// WatermarkLevelDevice 设备级完成水位（15min→小时→… 设备维度聚合）。
	WatermarkLevelDevice WatermarkLevel = "device"
	// WatermarkLevelGroup 设备组级完成水位（设备级之后链式产出的组维度聚合）。
	WatermarkLevelGroup WatermarkLevel = "group"
)

// ErrWatermarkNotFound 表示该 (粒度, 层级) 尚无水位记录（首次卷数据前）。
var ErrWatermarkNotFound = errors.New("pm completion watermark not found")

// Watermark 是 pm_completion_watermarks 表一行的领域视图（#528 P1）。
type Watermark struct {
	Granularity          metrics.Granularity
	Level                WatermarkLevel
	CompletedBucketStart time.Time // 已处理完成到的格起点（含），桶头时间戳
	UpdatedAt            time.Time
}

// WatermarkExecer 是写水位所需的最小执行能力；*pgxpool.Pool 与 pgx.Tx 均满足。
//
// 抽成接口的目的：让水位 UPSERT 既能用连接池单独跑，也能在调用方持有的事务里跑——
// 上游卷数据每成功处理完一格后，在其成功收尾处复用同一执行体写水位，保证「卷完即记」原子贴合。
type WatermarkExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// WatermarkQuerier 是读水位所需的最小查询能力；*pgxpool.Pool 与 pgx.Tx 均满足。
type WatermarkQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WatermarkRepository 读写 PM 完成水位（#528 P1）。
//
// 写入语义：UPSERT 取 max(现值, 本格起点)——水位只进不退，并发/重复触发幂等。
// 之所以取 max 而非直接覆盖：上游可能乱序补跑历史格（catchup / 重试），
// 用 max 保证「补一个旧格」不会把已推进的水位往回拽。
//
// 读取语义：按 (粒度, 层级) 取当前水位；无记录返回 ErrWatermarkNotFound。
type WatermarkRepository struct {
	db watermarkDB
}

// watermarkDB 是 WatermarkRepository 读写所需的最小 pgx 子集（*pgxpool.Pool 满足）。
type watermarkDB interface {
	WatermarkExecer
	WatermarkQuerier
}

// NewWatermarkRepository 构造水位仓库。db 真实实现为 *pgxpool.Pool。
func NewWatermarkRepository(db watermarkDB) *WatermarkRepository {
	return &WatermarkRepository{db: db}
}

// upsertWatermarkSQL 是「取 max 不回退」的幂等 UPSERT。
//
// ON CONFLICT 命中时用 GREATEST 保证并发/乱序补跑下取 max；同时刷新 updated_at。
const upsertWatermarkSQL = `
INSERT INTO pm_completion_watermarks (granularity, level, completed_bucket_start, created_at, updated_at)
VALUES ($1, $2, $3, now(), now())
ON CONFLICT (granularity, level) DO UPDATE
SET completed_bucket_start = GREATEST(pm_completion_watermarks.completed_bucket_start, EXCLUDED.completed_bucket_start),
    updated_at = now()
`

// Upsert 把 (粒度, 层级) 的完成水位推进到 bucketStart（取 max，不回退；幂等）。
//
// 用 db（连接池）执行。需在调用方事务里写时改用包级 UpsertWatermark 传入事务执行体。
func (r *WatermarkRepository) Upsert(ctx context.Context, gran metrics.Granularity, level WatermarkLevel, bucketStart time.Time) error {
	return UpsertWatermark(ctx, r.db, gran, level, bucketStart)
}

// UpsertWatermark 与 Upsert 同语义，但用调用方传入的执行体（事务或池）跑，
// 便于上游卷数据在「卷完一格」的同一执行上下文里推进水位。
func UpsertWatermark(ctx context.Context, exec WatermarkExecer, gran metrics.Granularity, level WatermarkLevel, bucketStart time.Time) error {
	if _, err := exec.Exec(ctx, upsertWatermarkSQL, string(gran), string(level), bucketStart); err != nil {
		return fmt.Errorf("watermark upsert (%s/%s): %w", gran, level, err)
	}
	return nil
}

// Get 取 (粒度, 层级) 当前完成水位；无记录返回 ErrWatermarkNotFound。
func (r *WatermarkRepository) Get(ctx context.Context, gran metrics.Granularity, level WatermarkLevel) (*Watermark, error) {
	const q = `
SELECT granularity, level, completed_bucket_start, updated_at
FROM pm_completion_watermarks
WHERE granularity = $1 AND level = $2`
	var wm Watermark
	var g, lv string
	err := r.db.QueryRow(ctx, q, string(gran), string(level)).Scan(&g, &lv, &wm.CompletedBucketStart, &wm.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWatermarkNotFound
		}
		return nil, fmt.Errorf("watermark get (%s/%s): %w", gran, level, err)
	}
	wm.Granularity = metrics.Granularity(g)
	wm.Level = WatermarkLevel(lv)
	return &wm, nil
}
