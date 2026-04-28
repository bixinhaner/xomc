package alarm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgDeadLetterRepository 用 pgxpool 实现 webhook 死信记录持久化。
type PgDeadLetterRepository struct {
	pool *pgxpool.Pool
}

var _ DeadLetterRepository = (*PgDeadLetterRepository)(nil)

// NewPgDeadLetterRepository 构造 pg 死信仓储。
func NewPgDeadLetterRepository(pool *pgxpool.Pool) *PgDeadLetterRepository {
	return &PgDeadLetterRepository{pool: pool}
}

// Insert 写入一条死信记录。
// payload 字段是 JSONB，必须传可被 PostgreSQL 当作 JSON 解析的 []byte（即 json.Marshal 输出）。
func (r *PgDeadLetterRepository) Insert(ctx context.Context, rec *DeadLetterRecord) error {
	if rec == nil {
		return fmt.Errorf("dead letter record is nil")
	}
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	if rec.FailedAt.IsZero() {
		rec.FailedAt = time.Now()
	}

	query := storage.Psql.
		Insert("alarm_webhook_dead_letters").
		Columns("id", "filter_id", "alarm_id", "payload", "last_error", "retry_count", "failed_at").
		Values(rec.ID, rec.FilterID, rec.AlarmID, rec.Payload, rec.LastError, rec.RetryCount, rec.FailedAt)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build dead-letter insert: %w", err)
	}

	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("insert dead-letter: %w", err)
	}
	return nil
}
