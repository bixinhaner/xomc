package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgThirdPartyLocationBatchRepository struct {
	pool *pgxpool.Pool
}

func NewPgThirdPartyLocationBatchRepository(pool *pgxpool.Pool) *PgThirdPartyLocationBatchRepository {
	return &PgThirdPartyLocationBatchRepository{pool: pool}
}

func (r *PgThirdPartyLocationBatchRepository) Claim(
	ctx context.Context,
	key string,
	hash string,
) (*ThirdPartyLocationResult, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("third-party location batch database is required")
	}
	key = strings.TrimSpace(key)
	if key == "" || hash == "" {
		return nil, fmt.Errorf("third-party location batch key and hash are required")
	}
	claimQuery, claimArgs, err := storage.Psql.
		Insert("third_party_location_batches").
		Columns("idempotency_key", "request_hash", "status").
		Values(key, hash, "processing").
		Suffix("ON CONFLICT (idempotency_key) DO NOTHING RETURNING idempotency_key").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build claim third-party location batch: %w", err)
	}
	var claimedKey string
	err = r.pool.QueryRow(ctx, claimQuery, claimArgs...).Scan(&claimedKey)
	if err == nil {
		return nil, nil
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("claim third-party location batch: %w", err)
	}

	var storedHash, status string
	var resultJSON []byte
	readQuery, readArgs, err := storage.Psql.
		Select("request_hash", "status", "result").
		From("third_party_location_batches").
		Where(sq.Eq{"idempotency_key": key}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build read third-party location batch claim: %w", err)
	}
	err = r.pool.QueryRow(ctx, readQuery, readArgs...).Scan(
		&storedHash,
		&status,
		&resultJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("read third-party location batch claim: %w", err)
	}
	if storedHash != hash {
		return nil, ErrThirdPartyLocationBatchConflict
	}
	if status == "processing" {
		return nil, ErrThirdPartyLocationBatchInProgress
	}
	if status != "completed" || len(resultJSON) == 0 {
		return nil, fmt.Errorf("third-party location batch has unsupported status %q", status)
	}
	var result ThirdPartyLocationResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("decode third-party location batch result: %w", err)
	}
	return &result, nil
}

func (r *PgThirdPartyLocationBatchRepository) Complete(
	ctx context.Context,
	key string,
	result ThirdPartyLocationResult,
) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("third-party location batch database is required")
	}
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode third-party location batch result: %w", err)
	}
	completedAt := time.Now().UTC()
	query, args, err := storage.Psql.
		Update("third_party_location_batches").
		Set("status", "completed").
		Set("result", data).
		Set("completed_at", completedAt).
		Set("updated_at", completedAt).
		Where(sq.Eq{
			"idempotency_key": key,
			"status":          "processing",
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build complete third-party location batch: %w", err)
	}
	command, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("complete third-party location batch: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("third-party location batch is not claimable: %w", pgx.ErrNoRows)
	}
	return nil
}
