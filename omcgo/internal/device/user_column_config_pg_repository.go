package device

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgColumnConfigRepository implements ColumnConfigRepository using PostgreSQL.
type PgColumnConfigRepository struct {
	pool *pgxpool.Pool
}

var _ ColumnConfigRepository = (*PgColumnConfigRepository)(nil)

// NewPgColumnConfigRepository creates a new PgColumnConfigRepository.
func NewPgColumnConfigRepository(pool *pgxpool.Pool) *PgColumnConfigRepository {
	return &PgColumnConfigRepository{pool: pool}
}

func (r *PgColumnConfigRepository) Get(ctx context.Context, userID uuid.UUID, pageKey string) (*ColumnConfig, error) {
	const rawSQL = `SELECT user_id, page_key, columns, created_at, updated_at FROM user_column_configs WHERE user_id = $1 AND page_key = $2`

	var cfg ColumnConfig
	var columnsJSON []byte
	err := r.pool.QueryRow(ctx, rawSQL, userID, pageKey).Scan(
		&cfg.UserID, &cfg.PageKey, &columnsJSON, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get column config: %w", err)
	}

	if err := json.Unmarshal(columnsJSON, &cfg.Columns); err != nil {
		return nil, fmt.Errorf("unmarshal columns: %w", err)
	}
	return &cfg, nil
}

func (r *PgColumnConfigRepository) Upsert(ctx context.Context, config *ColumnConfig) error {
	now := time.Now()
	config.UpdatedAt = now

	columnsJSON, err := json.Marshal(config.Columns)
	if err != nil {
		return fmt.Errorf("marshal columns: %w", err)
	}

	const rawSQL = `
		INSERT INTO user_column_configs (user_id, page_key, columns, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, page_key) DO UPDATE SET columns = EXCLUDED.columns, updated_at = EXCLUDED.updated_at`

	_, err = r.pool.Exec(ctx, rawSQL, config.UserID, config.PageKey, columnsJSON, now, now)
	if err != nil {
		return fmt.Errorf("upsert column config: %w", err)
	}
	return nil
}
