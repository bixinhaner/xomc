package indicator

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

var thresholdIndicatorColumns = []string{
	"id", "indicator_id", "threshold_period", "threshold_color",
	"threshold_low", "threshold_high", "threshold_level", "created_at", "updated_at",
}

var _ IndicatorThresholdRepository = (*PgIndicatorThresholdRepository)(nil)

type PgIndicatorThresholdRepository struct {
	db storage.DB
}

func NewPgIndicatorThresholdRepository(pool *pgxpool.Pool) *PgIndicatorThresholdRepository {
	return &PgIndicatorThresholdRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgIndicatorThresholdRepository) ListByIndicatorID(ctx context.Context, indicatorID string) ([]*IndicatorThreshold, error) {
	query, args, err := storage.Psql.Select(thresholdIndicatorColumns...).
		From("indicator_threshold").
		Where(sq.Eq{"indicator_id": indicatorID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list indicator_threshold SQL: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list indicator_threshold: %w", err)
	}
	defer rows.Close()

	var items []*IndicatorThreshold
	for rows.Next() {
		t, err := scanIndicatorThreshold(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating threshold rows: %w", err)
	}
	return items, nil
}

func (r *PgIndicatorThresholdRepository) ExistsByIndicatorID(ctx context.Context, indicatorID string) (bool, error) {
	query, args, err := storage.Psql.Select("1").
		From("indicator_threshold").
		Where(sq.Eq{"indicator_id": indicatorID}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build indicator_threshold exists SQL: %w", err)
	}

	var exists int
	err = r.db.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check indicator_threshold exists: %w", err)
	}
	return true, nil
}

func scanIndicatorThreshold(rows pgx.Rows) (*IndicatorThreshold, error) {
	var t IndicatorThreshold
	err := rows.Scan(
		&t.ID, &t.IndicatorID, &t.ThresholdPeriod, &t.ThresholdColor,
		&t.ThresholdLow, &t.ThresholdHigh, &t.ThresholdLevel,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan indicator_threshold row: %w", err)
	}
	return &t, nil
}
