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

var formulaColumns = []string{
	"id", "platform_name", "indicator_id", "formula", "created_at", "updated_at",
}

var _ PlatformFormulaRepository = (*PgPlatformFormulaRepository)(nil)

type PgPlatformFormulaRepository struct {
	db storage.DB
}

func NewPgPlatformFormulaRepository(pool *pgxpool.Pool) *PgPlatformFormulaRepository {
	return &PgPlatformFormulaRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgPlatformFormulaRepository) ListByIndicatorID(ctx context.Context, dt DeviceType, indicatorID string) ([]*PlatformFormula, error) {
	table := dt.FormulaTable()
	query, args, err := storage.Psql.Select(formulaColumns...).
		From(table).
		Where(sq.Eq{"indicator_id": indicatorID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list %s by indicator SQL: %w", table, err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s by indicator: %w", table, err)
	}
	defer rows.Close()

	var formulas []*PlatformFormula
	for rows.Next() {
		f, err := scanFormula(rows)
		if err != nil {
			return nil, err
		}
		formulas = append(formulas, f)
	}
	if err := rows.Err(); err != nil {
				return nil, fmt.Errorf("iterating formula rows: %w", err)
		}
		return formulas, nil
}

func (r *PgPlatformFormulaRepository) BatchCreate(ctx context.Context, dt DeviceType, formulas []*PlatformFormula, tx pgx.Tx) error {
	if len(formulas) == 0 {
		return nil
	}
	table := dt.FormulaTable()
	now := time.Now()

	builder := storage.Psql.Insert(table).
		Columns("platform_name", "indicator_id", "formula", "created_at", "updated_at")

	for _, f := range formulas {
		builder = builder.Values(f.PlatformName, f.IndicatorID, f.Formula, now, now)
	}

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

func (r *PgPlatformFormulaRepository) DeleteByIndicatorID(ctx context.Context, dt DeviceType, indicatorID string, tx pgx.Tx) error {
	table := dt.FormulaTable()
	query, args, err := storage.Psql.Delete(table).
		Where(sq.Eq{"indicator_id": indicatorID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete %s by indicator SQL: %w", table, err)
	}

	q := querier(r.db, tx)
	_, err = q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete %s by indicator: %w", table, err)
	}
	return nil
}

func (r *PgPlatformFormulaRepository) DeleteByIndicatorIDs(ctx context.Context, dt DeviceType, indicatorIDs []string, tx pgx.Tx) error {
	if len(indicatorIDs) == 0 {
		return nil
	}
	table := dt.FormulaTable()
	query, args, err := storage.Psql.Delete(table).
		Where(sq.Eq{"indicator_id": indicatorIDs}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete %s by IDs SQL: %w", table, err)
	}

	q := querier(r.db, tx)
	_, err = q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete %s by IDs: %w", table, err)
	}
	return nil
}

func (r *PgPlatformFormulaRepository) ListPlatformNames(ctx context.Context, dt DeviceType) ([]string, error) {
	table := dt.FormulaTable()
	query, args, err := storage.Psql.Select("DISTINCT platform_name").
		From(table).
		OrderBy("platform_name").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list platform names SQL: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list platform names from %s: %w", table, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan platform name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating platform name rows: %w", err)
	}
	return names, nil
}

func scanFormula(rows pgx.Rows) (*PlatformFormula, error) {
	var f PlatformFormula
	err := rows.Scan(&f.ID, &f.PlatformName, &f.IndicatorID, &f.Formula, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan platform_formula row: %w", err)
	}
	return &f, nil
}
