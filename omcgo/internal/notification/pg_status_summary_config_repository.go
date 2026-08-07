package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgStatusSummaryConfigRepository struct{ db storage.DB }

func NewPgStatusSummaryConfigRepository(pool *pgxpool.Pool) *PgStatusSummaryConfigRepository {
	return &PgStatusSummaryConfigRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgStatusSummaryConfigRepository) Get(ctx context.Context, id uuid.UUID) (*StatusSummaryRuntimeConfig, error) {
	query, args, err := storage.Psql.Select("id", "enabled", "to_char(send_time, 'HH24:MI')", "time_zone", "revision", "updated_at").
		From("notification_status_summary_configs").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build Zed status summary config lookup: %w", err)
	}
	var config StatusSummaryRuntimeConfig
	if err := r.db.QueryRow(ctx, query, args...).Scan(&config.ID, &config.Enabled, &config.SendTime, &config.TimeZone, &config.Revision, &config.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, commonerrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("scan Zed status summary config: %w", err)
	}
	recipients, err := r.listRecipients(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	config.Recipients = recipients
	return &config, nil
}

func (r *PgStatusSummaryConfigRepository) Update(ctx context.Context, id uuid.UUID, expectedRevision int64, input StatusSummaryConfigInput, recipients []ProtectedStatusSummaryRecipient, actor string) (*StatusSummaryRuntimeConfig, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin Zed status summary config update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	lookupQuery, lookupArgs, err := storage.Psql.Select("revision").
		From("notification_status_summary_configs").Where(sq.Eq{"id": id}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build Zed status summary config lookup for update: %w", err)
	}
	var currentRevision int64
	lookupErr := tx.QueryRow(ctx, lookupQuery, lookupArgs...).Scan(&currentRevision)
	if errors.Is(lookupErr, pgx.ErrNoRows) {
		if expectedRevision != 1 {
			return nil, ErrRevisionMismatch
		}
		query, args, buildErr := storage.Psql.Insert("notification_status_summary_configs").
			Columns("id", "enabled", "send_time", "time_zone", "revision", "updated_by").
			Values(id, input.Enabled, input.SendTime, input.TimeZone, 1, actor).ToSql()
		if buildErr != nil {
			return nil, fmt.Errorf("build Zed status summary config insert: %w", buildErr)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("insert Zed status summary config: %w", err)
		}
	} else if lookupErr != nil {
		return nil, fmt.Errorf("lookup Zed status summary config for update: %w", lookupErr)
	} else {
		if currentRevision != expectedRevision {
			return nil, ErrRevisionMismatch
		}
		query, args, buildErr := storage.Psql.Update("notification_status_summary_configs").
			Set("enabled", input.Enabled).Set("send_time", input.SendTime).Set("time_zone", input.TimeZone).
			Set("revision", expectedRevision+1).Set("updated_by", actor).Set("updated_at", time.Now().UTC()).
			Where(sq.Eq{"id": id, "revision": expectedRevision}).ToSql()
		if buildErr != nil {
			return nil, fmt.Errorf("build Zed status summary config update: %w", buildErr)
		}
		if tag, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("update Zed status summary config: %w", err)
		} else if tag.RowsAffected() != 1 {
			return nil, ErrRevisionMismatch
		}
	}
	deleteQuery, deleteArgs, err := storage.Psql.Delete("notification_status_summary_config_recipients").Where(sq.Eq{"config_id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build Zed status summary recipient replacement: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteQuery, deleteArgs...); err != nil {
		return nil, fmt.Errorf("delete Zed status summary recipients: %w", err)
	}
	for _, recipient := range recipients {
		insertQuery, insertArgs, err := storage.Psql.Insert("notification_status_summary_config_recipients").Columns("config_id", "address_ciphertext", "address_key_version", "recipient_fingerprint").Values(id, recipient.Ciphertext, recipient.KeyVersion, recipient.Fingerprint).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build Zed status summary recipient insert: %w", err)
		}
		if _, err := tx.Exec(ctx, insertQuery, insertArgs...); err != nil {
			return nil, fmt.Errorf("insert Zed status summary recipient: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit Zed status summary config: %w", err)
	}
	return r.Get(ctx, id)
}

type statusSummaryConfigQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (r *PgStatusSummaryConfigRepository) listRecipients(ctx context.Context, db statusSummaryConfigQueryer, configID uuid.UUID) ([]ProtectedStatusSummaryRecipient, error) {
	query, args, err := storage.Psql.Select("address_ciphertext", "address_key_version", "recipient_fingerprint").From("notification_status_summary_config_recipients").Where(sq.Eq{"config_id": configID}).OrderBy("created_at", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build Zed status summary recipient lookup: %w", err)
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list Zed status summary recipients: %w", err)
	}
	defer rows.Close()
	items := make([]ProtectedStatusSummaryRecipient, 0)
	for rows.Next() {
		var item ProtectedStatusSummaryRecipient
		if err := rows.Scan(&item.Ciphertext, &item.KeyVersion, &item.Fingerprint); err != nil {
			return nil, fmt.Errorf("scan Zed status summary recipient: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

var _ StatusSummaryConfigRepository = (*PgStatusSummaryConfigRepository)(nil)
