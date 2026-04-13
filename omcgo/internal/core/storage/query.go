package storage

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Psql is the unified PostgreSQL StatementBuilder for all repositories.
// All repository files should use this instead of declaring their own
// var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
var Psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// DB abstracts pgxpool.Pool for repository use, enabling mock testing.
// Repositories should accept this interface instead of *pgxpool.Pool directly.
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_db_test.go -package=storage . DB
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PoolDB wraps *pgxpool.Pool to implement the DB interface.
type PoolDB struct {
	pool *pgxpool.Pool
}

// NewPoolDB creates a new PoolDB wrapper.
func NewPoolDB(pool *pgxpool.Pool) *PoolDB {
	return &PoolDB{pool: pool}
}

// Query delegates to pgxpool.Pool.Query.
func (p *PoolDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// QueryRow delegates to pgxpool.Pool.QueryRow.
func (p *PoolDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// Exec delegates to pgxpool.Pool.Exec.
func (p *PoolDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

// Begin delegates to pgxpool.Pool.Begin.
func (p *PoolDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.pool.Begin(ctx)
}

// Pool returns the underlying *pgxpool.Pool (for CopyFrom and other pgx-specific operations).
func (p *PoolDB) Pool() *pgxpool.Pool {
	return p.pool
}
