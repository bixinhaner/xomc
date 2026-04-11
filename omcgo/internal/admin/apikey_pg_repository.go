package admin

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var apiKeyColumns = []string{
	"id", "user_id", "name", "key_prefix", "key_hash", "scopes",
	"expires_at", "last_used_at", "created_at", "updated_at", "revoked_at",
}

// PgAPIKeyRepository implements API key storage using PostgreSQL.
type PgAPIKeyRepository struct {
	pool *pgxpool.Pool
}

// NewPgAPIKeyRepository creates a new PgAPIKeyRepository.
func NewPgAPIKeyRepository(pool *pgxpool.Pool) *PgAPIKeyRepository {
	return &PgAPIKeyRepository{pool: pool}
}

// Create inserts a new API key.
func (r *PgAPIKeyRepository) Create(ctx context.Context, key *APIKey) error {
	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	now := time.Now()
	key.CreatedAt = now
	key.UpdatedAt = now

	query, args, err := storage.Psql.Insert("api_keys").
		Columns(apiKeyColumns...).
		Values(
			key.ID, key.UserID, key.Name, key.KeyPrefix, key.KeyHash,
			key.Scopes, nullableTime(key.ExpiresAt), nullableTime(key.LastUsedAt),
			key.CreatedAt, key.UpdatedAt, nullableTime(key.RevokedAt),
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert api_key SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert api_key: %w", err)
	}
	return nil
}

// GetByPrefix returns all non-revoked API keys matching the given prefix.
func (r *PgAPIKeyRepository) GetByPrefix(ctx context.Context, prefix string) ([]*APIKey, error) {
	query, args, err := storage.Psql.Select(apiKeyColumns...).
		From("api_keys").
		Where(sq.Eq{"key_prefix": prefix}).
		Where("revoked_at IS NULL").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get api_key by prefix SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query api_keys by prefix: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// ListByUser returns all API keys for a given user.
func (r *PgAPIKeyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	query, args, err := storage.Psql.Select(apiKeyColumns...).
		From("api_keys").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list api_keys SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list api_keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// Revoke soft-deletes an API key by setting revoked_at.
func (r *PgAPIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query, args, err := storage.Psql.Update("api_keys").
		Set("revoked_at", now).
		Set("updated_at", now).
		Where(sq.Eq{"id": id}).
		Where("revoked_at IS NULL").
		ToSql()
	if err != nil {
		return fmt.Errorf("build revoke api_key SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("revoke api_key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp for an API key.
func (r *PgAPIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Update("api_keys").
		Set("last_used_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update last_used SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func scanAPIKey(rows pgx.Rows) (*APIKey, error) {
	var k APIKey
	err := rows.Scan(
		&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.KeyHash,
		&k.Scopes, &k.ExpiresAt, &k.LastUsedAt,
		&k.CreatedAt, &k.UpdatedAt, &k.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan api_key: %w", err)
	}
	return &k, nil
}
