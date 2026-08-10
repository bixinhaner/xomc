package outbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
)

const maxStoredErrorRunes = 2048

// PgRelayRepository implements multi-instance claims and lease-guarded
// finalization for the generic event outbox.
type PgRelayRepository struct {
	db storage.DB
}

func NewPgRelayRepository(db storage.DB) *PgRelayRepository {
	return &PgRelayRepository{db: db}
}

func (r *PgRelayRepository) ClaimDue(
	ctx context.Context,
	options ClaimOptions,
) ([]Entry, error) {
	if err := validateClaimOptions(options); err != nil {
		return nil, err
	}
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("claim event outbox: database is required")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin event outbox claim: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	expireSQL, expireArgs, err := buildExpireExhaustedClaims(options.Now, options.MaxAttempts)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, expireSQL, expireArgs...); err != nil {
		return nil, fmt.Errorf("expire exhausted event outbox claims: %w", err)
	}

	selectSQL, selectArgs, err := buildClaimSelect(options)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("select claimable event outbox rows: %w", err)
	}

	entries := make([]Entry, 0, options.Limit)
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(
			&entry.ID,
			&entry.AggregateType,
			&entry.AggregateID,
			&entry.Subject,
			&entry.Payload,
			&entry.DedupeKey,
			&entry.Status,
			&entry.Attempts,
			&entry.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan claimable event outbox row: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate claimable event outbox rows: %w", err)
	}
	rows.Close()

	if len(entries) > 0 {
		ids := make([]uuid.UUID, 0, len(entries))
		for _, entry := range entries {
			ids = append(ids, entry.ID)
		}
		claimToken := uuid.New()
		claimExpiresAt := options.Now.Add(options.Lease)
		updateSQL, updateArgs, err := buildClaimUpdate(ids, claimToken, claimExpiresAt)
		if err != nil {
			return nil, err
		}
		tag, err := tx.Exec(ctx, updateSQL, updateArgs...)
		if err != nil {
			return nil, fmt.Errorf("mark event outbox rows publishing: %w", err)
		}
		if tag.RowsAffected() != int64(len(entries)) {
			return nil, fmt.Errorf(
				"mark event outbox rows publishing: affected %d, expected %d",
				tag.RowsAffected(),
				len(entries),
			)
		}
		for i := range entries {
			entries[i].Status = StatusPublishing
			entries[i].Attempts++
			entries[i].ClaimToken = claimToken
			entries[i].ClaimExpiresAt = claimExpiresAt
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit event outbox claim: %w", err)
	}
	return entries, nil
}

func (r *PgRelayRepository) MarkPublished(
	ctx context.Context,
	id uuid.UUID,
	claimToken uuid.UUID,
	publishedAt time.Time,
) (bool, error) {
	query, args, err := buildMarkPublished(id, claimToken, publishedAt)
	if err != nil {
		return false, err
	}
	return r.execFinalizer(ctx, "mark event outbox published", query, args)
}

func (r *PgRelayRepository) MarkFailed(
	ctx context.Context,
	id uuid.UUID,
	claimToken uuid.UUID,
	errorMessage string,
	nextAttemptAt time.Time,
) (bool, error) {
	now := time.Now().UTC()
	query, args, err := buildMarkFailed(
		id,
		claimToken,
		truncateOutboxError(errorMessage),
		nextAttemptAt,
		now,
	)
	if err != nil {
		return false, err
	}
	return r.execFinalizer(ctx, "mark event outbox failed", query, args)
}

func (r *PgRelayRepository) MarkDead(
	ctx context.Context,
	id uuid.UUID,
	claimToken uuid.UUID,
	errorMessage string,
	deadAt time.Time,
) (bool, error) {
	query, args, err := buildMarkDead(
		id,
		claimToken,
		truncateOutboxError(errorMessage),
		deadAt,
	)
	if err != nil {
		return false, err
	}
	return r.execFinalizer(ctx, "mark event outbox dead", query, args)
}

func (r *PgRelayRepository) execFinalizer(
	ctx context.Context,
	operation string,
	query string,
	args []any,
) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("%s: database is required", operation)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("%s: %w", operation, err)
	}
	return tag.RowsAffected() == 1, nil
}

func validateClaimOptions(options ClaimOptions) error {
	switch {
	case options.Now.IsZero():
		return fmt.Errorf("claim event outbox: current time is required")
	case options.Lease <= 0:
		return fmt.Errorf("claim event outbox: lease must be positive")
	case options.Limit <= 0:
		return fmt.Errorf("claim event outbox: limit must be positive")
	case options.MaxAttempts <= 0:
		return fmt.Errorf("claim event outbox: maximum attempts must be positive")
	default:
		return nil
	}
}

func claimSelectColumns() []string {
	return []string{
		"id",
		"aggregate_type",
		"aggregate_id",
		"subject",
		"payload",
		"dedupe_key",
		"status",
		"attempts",
		"created_at",
	}
}

func buildClaimSelect(options ClaimOptions) (string, []any, error) {
	if err := validateClaimOptions(options); err != nil {
		return "", nil, err
	}
	return storage.Psql.
		Select(claimSelectColumns()...).
		From("event_outbox").
		Where(sq.Or{
			sq.And{
				sq.Eq{"status": []Status{StatusPending, StatusFailed}},
				sq.LtOrEq{"next_attempt_at": options.Now},
			},
			sq.And{
				sq.Eq{"status": StatusPublishing},
				sq.Or{
					sq.Eq{"claim_expires_at": nil},
					sq.LtOrEq{"claim_expires_at": options.Now},
				},
			},
		}).
		Where(sq.Lt{"attempts": options.MaxAttempts}).
		OrderBy("CASE WHEN status = 'publishing' THEN 0 ELSE 1 END").
		OrderBy("next_attempt_at ASC", "created_at ASC", "id ASC").
		Limit(uint64(options.Limit)).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
}

func buildClaimUpdate(
	ids []uuid.UUID,
	claimToken uuid.UUID,
	claimExpiresAt time.Time,
) (string, []any, error) {
	switch {
	case len(ids) == 0:
		return "", nil, fmt.Errorf("build event outbox claim update: ids are required")
	case claimToken == uuid.Nil:
		return "", nil, fmt.Errorf("build event outbox claim update: claim token is required")
	case claimExpiresAt.IsZero():
		return "", nil, fmt.Errorf("build event outbox claim update: claim expiry is required")
	}
	return storage.Psql.
		Update("event_outbox").
		Set("status", StatusPublishing).
		Set("attempts", sq.Expr("attempts + 1")).
		Set("claim_token", claimToken).
		Set("claim_expires_at", claimExpiresAt).
		Set("last_error", nil).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"id": ids}).
		ToSql()
}

func buildExpireExhaustedClaims(
	now time.Time,
	maxAttempts int,
) (string, []any, error) {
	if now.IsZero() || maxAttempts <= 0 {
		return "", nil, fmt.Errorf("build exhausted event outbox claims: invalid options")
	}
	return storage.Psql.
		Update("event_outbox").
		Set("status", StatusDead).
		Set("claim_token", nil).
		Set("claim_expires_at", nil).
		Set("last_error", "claim lease expired after maximum attempts").
		Set("updated_at", now).
		Where(sq.Eq{"status": StatusPublishing}).
		Where(sq.LtOrEq{"claim_expires_at": now}).
		Where(sq.GtOrEq{"attempts": maxAttempts}).
		ToSql()
}

func buildMarkPublished(
	id uuid.UUID,
	claimToken uuid.UUID,
	publishedAt time.Time,
) (string, []any, error) {
	if err := validateFinalizerIdentity(id, claimToken, publishedAt); err != nil {
		return "", nil, fmt.Errorf("build mark event outbox published: %w", err)
	}
	return buildFinalizer(id, claimToken, StatusPublished, publishedAt).
		Set("published_at", publishedAt).
		Set("last_error", nil).
		ToSql()
}

func buildMarkFailed(
	id uuid.UUID,
	claimToken uuid.UUID,
	errorMessage string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) (string, []any, error) {
	if err := validateFinalizerIdentity(id, claimToken, failedAt); err != nil {
		return "", nil, fmt.Errorf("build mark event outbox failed: %w", err)
	}
	if nextAttemptAt.IsZero() {
		return "", nil, fmt.Errorf("build mark event outbox failed: next attempt time is required")
	}
	return buildFinalizer(id, claimToken, StatusFailed, failedAt).
		Set("last_error", errorMessage).
		Set("next_attempt_at", nextAttemptAt).
		ToSql()
}

func buildMarkDead(
	id uuid.UUID,
	claimToken uuid.UUID,
	errorMessage string,
	deadAt time.Time,
) (string, []any, error) {
	if err := validateFinalizerIdentity(id, claimToken, deadAt); err != nil {
		return "", nil, fmt.Errorf("build mark event outbox dead: %w", err)
	}
	return buildFinalizer(id, claimToken, StatusDead, deadAt).
		Set("last_error", errorMessage).
		ToSql()
}

func buildFinalizer(
	id uuid.UUID,
	claimToken uuid.UUID,
	status Status,
	updatedAt time.Time,
) sq.UpdateBuilder {
	return storage.Psql.
		Update("event_outbox").
		Set("status", status).
		Set("claim_token", nil).
		Set("claim_expires_at", nil).
		Set("updated_at", updatedAt).
		Where(sq.Eq{
			"id":          id,
			"status":      StatusPublishing,
			"claim_token": claimToken,
		})
}

func validateFinalizerIdentity(
	id uuid.UUID,
	claimToken uuid.UUID,
	updatedAt time.Time,
) error {
	switch {
	case id == uuid.Nil:
		return fmt.Errorf("id is required")
	case claimToken == uuid.Nil:
		return fmt.Errorf("claim token is required")
	case updatedAt.IsZero():
		return fmt.Errorf("updated time is required")
	default:
		return nil
	}
}

func truncateOutboxError(message string) string {
	message = strings.TrimSpace(message)
	runes := []rune(message)
	if len(runes) <= maxStoredErrorRunes {
		return message
	}
	return string(runes[:maxStoredErrorRunes])
}

var _ DeliveryRepository = (*PgRelayRepository)(nil)
