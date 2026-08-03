package indicator

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

var _ EnabledIndicatorRepository = (*PgEnabledRepository)(nil)

type PgEnabledRepository struct {
	db storage.DB
}

func NewPgEnabledRepository(pool *pgxpool.Pool) *PgEnabledRepository {
	return &PgEnabledRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgEnabledRepository) List(ctx context.Context, dt DeviceType, operatorCode string) ([]string, error) {
	return r.list(ctx, dt, operatorCode, nil)
}

func (r *PgEnabledRepository) ListTx(
	ctx context.Context,
	dt DeviceType,
	operatorCode string,
	tx pgx.Tx,
) ([]string, error) {
	return r.list(ctx, dt, operatorCode, tx)
}

func (r *PgEnabledRepository) list(
	ctx context.Context,
	dt DeviceType,
	operatorCode string,
	tx pgx.Tx,
) ([]string, error) {
	table := dt.EnabledTable()
	query, args, err := storage.Psql.Select("indicator_id").
		From(table).
		Where(sq.Eq{"operator_code": operatorCode}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list %s SQL: %w", table, err)
	}

	rows, err := querier(r.db, tx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", table, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan enabled indicator ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating enabled indicator rows: %w", err)
	}
	return ids, nil
}

func (r *PgEnabledRepository) ListAll(ctx context.Context, dt DeviceType) ([]string, error) {
	table := dt.EnabledTable()
	query, args, err := storage.Psql.Select("DISTINCT indicator_id").
		From(table).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list all %s SQL: %w", table, err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all %s: %w", table, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan enabled indicator ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating enabled indicator rows: %w", err)
	}
	return ids, nil
}

// ListOperatorCodes returns every persisted enabled-indicator scope. Startup
// reconciliation uses the scopes instead of assuming that only "default"
// exists, so operator-specific KPI selections receive the same closure repair.
func (r *PgEnabledRepository) ListOperatorCodes(ctx context.Context, dt DeviceType) ([]string, error) {
	table := dt.EnabledTable()
	query, args, err := storage.Psql.Select("operator_code").
		Distinct().
		From(table).
		OrderBy("operator_code").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list operator codes from %s: %w", table, err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list operator codes from %s: %w", table, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var operatorCode string
		if err := rows.Scan(&operatorCode); err != nil {
			return nil, fmt.Errorf("scan enabled indicator operator code: %w", err)
		}
		out = append(out, operatorCode)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled indicator operator codes: %w", err)
	}
	return out, nil
}

func (r *PgEnabledRepository) BatchCreate(ctx context.Context, dt DeviceType, operatorCode string, indicatorIDs []string, tx pgx.Tx) error {
	if len(indicatorIDs) == 0 {
		return nil
	}
	table := dt.EnabledTable()
	now := time.Now()

	builder := storage.Psql.Insert(table).
		Columns("operator_code", "indicator_id", "created_at", "updated_at")

	for _, id := range indicatorIDs {
		builder = builder.Values(operatorCode, id, now, now)
	}

	// Use ON CONFLICT DO NOTHING for idempotency.
	builder = builder.Suffix("ON CONFLICT (operator_code, indicator_id) DO NOTHING")

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build batch insert %s SQL: %w", table, err)
	}

	q := querier(r.db, tx)
	_, err = q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch insert %s: %w", table, err)
	}
	return nil
}

func (r *PgEnabledRepository) BatchDelete(ctx context.Context, dt DeviceType, operatorCode string, indicatorIDs []string, tx pgx.Tx) error {
	if len(indicatorIDs) == 0 {
		return nil
	}
	table := dt.EnabledTable()
	query, args, err := storage.Psql.Delete(table).
		Where(sq.Eq{"operator_code": operatorCode, "indicator_id": indicatorIDs}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build batch delete %s SQL: %w", table, err)
	}

	q := querier(r.db, tx)
	_, err = q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch delete %s: %w", table, err)
	}
	return nil
}

func (r *PgEnabledRepository) Exists(ctx context.Context, dt DeviceType, operatorCode string, indicatorID string) (bool, error) {
	table := dt.EnabledTable()
	query, args, err := storage.Psql.Select("1").
		From(table).
		Where(sq.Eq{"operator_code": operatorCode, "indicator_id": indicatorID}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build exists %s SQL: %w", table, err)
	}

	var exists int
	err = r.db.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check exists in %s: %w", table, err)
	}
	return true, nil
}
