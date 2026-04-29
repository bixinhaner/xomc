package dlq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgRepository 是 dead_letters 表的 pgxpool 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

var _ Repository = (*PgRepository)(nil)

// NewPgRepository 构造 PostgreSQL 实现。pool 不能为空。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// Insert 写入一条死信记录。
// nil entry 直接返回错误，零 ID 自动生成，零时间戳填 now。
// Error 字段超出 MaxErrorLength 自动截断（避免 PG 行越界）。
func (r *PgRepository) Insert(ctx context.Context, entry *DeadLetter) error {
	if entry == nil {
		return fmt.Errorf("dead letter entry is nil")
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	now := time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if entry.LastAttemptAt.IsZero() {
		entry.LastAttemptAt = now
	}
	entry.Error = TruncateError(entry.Error)

	q := storage.Psql.
		Insert("dead_letters").
		Columns("id", "source_module", "source_subject", "payload", "error", "retry_count", "created_at", "last_attempt_at").
		Values(entry.ID, entry.SourceModule, entry.SourceSubject, entry.Payload, entry.Error, entry.RetryCount, entry.CreatedAt, entry.LastAttemptAt)

	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build dead-letter insert: %w", err)
	}
	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("insert dead-letter: %w", err)
	}
	return nil
}

// List 按 created_at DESC 分页查询死信记录。
// Filter 中的 Module / Subject 为 nil 时不过滤。
func (r *PgRepository) List(ctx context.Context, filter Filter) (*model.ListResponse[DeadLetter], error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	base := storage.Psql.Select().From("dead_letters")
	if filter.Module != nil && *filter.Module != "" {
		base = base.Where("source_module = ?", *filter.Module)
	}
	if filter.Subject != nil && *filter.Subject != "" {
		base = base.Where("source_subject = ?", *filter.Subject)
	}

	// total
	countSQL, countArgs, err := base.Columns("COUNT(*)").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build dead-letter count: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count dead-letters: %w", err)
	}

	// items
	itemsSQL, itemsArgs, err := base.
		Columns("id", "source_module", "source_subject", "payload", "error", "retry_count", "created_at", "last_attempt_at").
		OrderBy("created_at DESC").
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset())).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build dead-letter list: %w", err)
	}
	rows, err := r.pool.Query(ctx, itemsSQL, itemsArgs...)
	if err != nil {
		return nil, fmt.Errorf("query dead-letters: %w", err)
	}
	defer rows.Close()

	items := make([]DeadLetter, 0, filter.PageSize)
	for rows.Next() {
		var d DeadLetter
		if err := rows.Scan(&d.ID, &d.SourceModule, &d.SourceSubject, &d.Payload, &d.Error, &d.RetryCount, &d.CreatedAt, &d.LastAttemptAt); err != nil {
			return nil, fmt.Errorf("scan dead-letter row: %w", err)
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dead-letters: %w", err)
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// Get 按主键查询死信记录。未找到返回 (nil, nil)，DB 错误返回 (nil, err)。
func (r *PgRepository) Get(ctx context.Context, id uuid.UUID) (*DeadLetter, error) {
	q := storage.Psql.
		Select("id", "source_module", "source_subject", "payload", "error", "retry_count", "created_at", "last_attempt_at").
		From("dead_letters").
		Where("id = ?", id)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build dead-letter get: %w", err)
	}
	var d DeadLetter
	row := r.pool.QueryRow(ctx, sql, args...)
	if err := row.Scan(&d.ID, &d.SourceModule, &d.SourceSubject, &d.Payload, &d.Error, &d.RetryCount, &d.CreatedAt, &d.LastAttemptAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get dead-letter: %w", err)
	}
	return &d, nil
}

// Delete 按主键删除。删除不存在的记录不报错（pg DELETE 0 行不视为错误）。
func (r *PgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := storage.Psql.Delete("dead_letters").Where("id = ?", id)
	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build dead-letter delete: %w", err)
	}
	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("delete dead-letter: %w", err)
	}
	return nil
}

// Count 返回某 module 下的死信总数（worker_dlq_size 拉取使用）。
// sourceModule 为空字符串时返回全表总数。
func (r *PgRepository) Count(ctx context.Context, sourceModule string) (int64, error) {
	q := storage.Psql.Select("COUNT(*)").From("dead_letters")
	if sourceModule != "" {
		q = q.Where("source_module = ?", sourceModule)
	}
	sql, args, err := q.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build dead-letter count: %w", err)
	}
	var n int64
	if err := r.pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count dead-letters: %w", err)
	}
	return n, nil
}
