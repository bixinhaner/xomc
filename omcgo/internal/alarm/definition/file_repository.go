package definition

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FileRepository 提供 XML 文件粒度的 DB 操作(对标 T-0180 indicator/file_repository.go;
// 告警 alarm_definitions 是单表,无级联子表,故只需按 loaded_from 计数 / 删除)。
type FileRepository interface {
	// CountByLoadedFrom 返回某 loaded_from 关联的 alarm_definitions 行数。
	CountByLoadedFrom(ctx context.Context, loadedFrom string) (int, error)
	// DeleteByLoadedFrom 删除某 loaded_from 关联的全部 alarm_definitions,返回删除行数。
	DeleteByLoadedFrom(ctx context.Context, loadedFrom string) (int, error)
}

// PgFileRepository 是 FileRepository 的 PostgreSQL 实现。
type PgFileRepository struct {
	pool *pgxpool.Pool
}

func NewPgFileRepository(pool *pgxpool.Pool) *PgFileRepository {
	return &PgFileRepository{pool: pool}
}

func (r *PgFileRepository) CountByLoadedFrom(ctx context.Context, loadedFrom string) (int, error) {
	sqlStr, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("COUNT(*)").From("alarm_definitions").
		Where(sq.Eq{"loaded_from": loadedFrom}).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count sql: %w", err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, sqlStr, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count alarm_definitions by loaded_from: %w", err)
	}
	return n, nil
}

func (r *PgFileRepository) DeleteByLoadedFrom(ctx context.Context, loadedFrom string) (int, error) {
	sqlStr, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Delete("alarm_definitions").
		Where(sq.Eq{"loaded_from": loadedFrom}).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete sql: %w", err)
	}
	tag, err := r.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("delete alarm_definitions by loaded_from: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

var _ FileRepository = (*PgFileRepository)(nil)
