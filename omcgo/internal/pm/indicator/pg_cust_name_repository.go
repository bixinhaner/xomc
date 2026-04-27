package indicator

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var custNameColumns = []string{
	"operator_code", "perf_id", "cust_name", "created_at", "updated_at",
}

var _ CustNameRepository = (*PgCustNameRepository)(nil)

type PgCustNameRepository struct {
	db storage.DB
}

func NewPgCustNameRepository(pool *pgxpool.Pool) *PgCustNameRepository {
	return &PgCustNameRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgCustNameRepository) Upsert(ctx context.Context, custName *CustName, tx pgx.Tx) error {
	now := time.Now()
	custName.CreatedAt = now
	custName.UpdatedAt = now

	query, args, err := storage.Psql.Insert("perf_cust_name").
		Columns("operator_code", "perf_id", "cust_name", "created_at", "updated_at").
		Values(custName.OperatorCode, custName.PerfID, custName.CustName, custName.CreatedAt, custName.UpdatedAt).
		Suffix("ON CONFLICT (operator_code, perf_id) DO UPDATE SET cust_name = EXCLUDED.cust_name, updated_at = EXCLUDED.updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert perf_cust_name SQL: %w", err)
	}

	q := querier(r.db, tx)
	_, err = q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("upsert perf_cust_name: %w", err)
	}
	return nil
}

func (r *PgCustNameRepository) Get(ctx context.Context, operatorCode string, perfID string) (*CustName, error) {
	query, args, err := storage.Psql.Select(custNameColumns...).
		From("perf_cust_name").
		Where(sq.Eq{"operator_code": operatorCode, "perf_id": perfID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get perf_cust_name SQL: %w", err)
	}

	var cn CustName
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&cn.OperatorCode, &cn.PerfID, &cn.CustName, &cn.CreatedAt, &cn.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get perf_cust_name: %w", err)
	}
	return &cn, nil
}

func (r *PgCustNameRepository) ListByOperator(ctx context.Context, operatorCode string) ([]*CustName, error) {
	query, args, err := storage.Psql.Select(custNameColumns...).
		From("perf_cust_name").
		Where(sq.Eq{"operator_code": operatorCode}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list perf_cust_name SQL: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list perf_cust_name: %w", err)
	}
	defer rows.Close()

	var items []*CustName
	for rows.Next() {
		var cn CustName
		if err := rows.Scan(
			&cn.OperatorCode, &cn.PerfID, &cn.CustName, &cn.CreatedAt, &cn.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan perf_cust_name row: %w", err)
		}
		items = append(items, &cn)
	}
	if err := rows.Err(); err != nil {
				return nil, fmt.Errorf("iterating cust name rows: %w", err)
		}
		return items, nil
}
