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
	// NeTypeExists 报告是否已有同 ne_type 的 alarm_definitions 行。
	// 用于 Upload 内容主键(neType)唯一性校验(三库 XML 导入重构 §7.1):
	// 一个 ne_type 对应一个 XML 文件,故 DB 中已存在该 ne_type = 内容主键冲突。
	NeTypeExists(ctx context.Context, neType string) (bool, error)
	// LoadedFromsByNeType 返回拥有某 ne_type 的去重 loaded_from 列表。
	// 导入 XML 覆盖调整(2026-06-05):上传遇到重复 neType 时,用它定位"归属文件",
	// force 覆盖即替换该文件(常态恰好一个;空 loaded_from 行被忽略)。
	LoadedFromsByNeType(ctx context.Context, neType string) ([]string, error)
	// ListManualByNeType 返回某 ne_type 下全部手工新增定义(loaded_from 为空)，
	// JOIN severity_levels 带出名称。#268: file-content 生成模式的数据源。
	ListManualByNeType(ctx context.Context, neType string) ([]ResolvedDefinition, error)
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

func (r *PgFileRepository) NeTypeExists(ctx context.Context, neType string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM alarm_definitions WHERE ne_type = $1)`, neType,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check alarm_definitions ne_type exists %q: %w", neType, err)
	}
	return exists, nil
}

// LoadedFromsByNeType 实现 FileRepository。
func (r *PgFileRepository) LoadedFromsByNeType(ctx context.Context, neType string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT loaded_from FROM alarm_definitions WHERE ne_type = $1 AND loaded_from <> ''`, neType)
	if err != nil {
		return nil, fmt.Errorf("query alarm_definitions loaded_from by ne_type %q: %w", neType, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var lf string
		if err := rows.Scan(&lf); err != nil {
			return nil, fmt.Errorf("scan loaded_from: %w", err)
		}
		out = append(out, lf)
	}
	return out, rows.Err()
}

// ListManualByNeType 实现 FileRepository:按 ne_type 取 loaded_from 为空
// (NULL 或 '')的全量行,与一级表"手工新增"行的 drill-down 口径一致。
// 单 ne_type 手工定义远小于 pageSize 上限,不分页,按 identifier 排序保证导出稳定。
func (r *PgFileRepository) ListManualByNeType(ctx context.Context, neType string) ([]ResolvedDefinition, error) {
	sb := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select(
			"d.id", "d.identifier", "d.ne_type", "d.cn_name", "d.en_name",
			"d.severity_id", "d.event_type", "d.cn_probable_cause", "d.en_probable_cause",
			"d.is_show", "d.description",
			"l.code", "l.name",
		).
		From("alarm_definitions d").
		Join("alarm_severity_levels l ON d.severity_id = l.id").
		Where(sq.Eq{"d.ne_type": neType}).
		Where(sq.Expr("COALESCE(d.loaded_from, '') = ''")).
		OrderBy("d.identifier ASC")
	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build manual-by-netype sql: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query manual alarm_definitions by ne_type %q: %w", neType, err)
	}
	defer rows.Close()
	out, err := scanResolvedDefs(rows)
	if err != nil {
		return nil, fmt.Errorf("scan manual definitions by ne_type %q: %w", neType, err)
	}
	return out, nil
}

var _ FileRepository = (*PgFileRepository)(nil)
