package indicator

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

var _ TemplateRelRepository = (*PgTemplateRelRepository)(nil)

type PgTemplateRelRepository struct {
	db storage.DB
}

func NewPgTemplateRelRepository(pool *pgxpool.Pool) *PgTemplateRelRepository {
	return &PgTemplateRelRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgTemplateRelRepository) ExistsByIndicatorID(ctx context.Context, indicatorID string) (bool, error) {
	query, args, err := storage.Psql.Select("1").
		From("perf_template_rel_arithmetic").
		Where(sq.Eq{"indicator_id": indicatorID}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build template rel exists SQL: %w", err)
	}

	var exists int
	err = r.db.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check template rel exists: %w", err)
	}
	return true, nil
}
