package definition

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgRepository 是 Repository 的 PostgreSQL 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// ListAll 拉取全部 alarm_definitions 与 severity 反查（join 到 alarm_severity_levels）。
//
// 期望规模 ~442 行（设计 §3.4 给定），单次查询完整加载到内存即可，不做分页。
func (r *PgRepository) ListAll(ctx context.Context) ([]ResolvedDefinition, error) {
	sqlStr, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select(
			"d.id", "d.identifier", "d.ne_type", "d.cn_name", "d.en_name",
			"d.severity_id", "d.event_type", "d.cn_probable_cause", "d.en_probable_cause",
			"d.cn_suggestion", "d.en_suggestion",
			"d.is_show", "d.description",
			"l.code", "l.name",
		).
		From("alarm_definitions d").
		Join("alarm_severity_levels l ON d.severity_id = l.id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sql list alarm_definitions: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query alarm_definitions: %w", err)
	}
	defer rows.Close()

	var out []ResolvedDefinition
	for rows.Next() {
		var rd ResolvedDefinition
		var (
			cnName, enName, cnProbCause, enProbCause, cnSuggestion, enSuggestion, description *string
			eventType                                                                         *int
		)
		if err := rows.Scan(
			&rd.ID, &rd.Identifier, &rd.NeType, &cnName, &enName,
			&rd.SeverityID, &eventType, &cnProbCause, &enProbCause,
			&cnSuggestion, &enSuggestion,
			&rd.IsShow, &description,
			&rd.SeverityCode, &rd.SeverityName,
		); err != nil {
			return nil, fmt.Errorf("scan alarm_definition: %w", err)
		}
		rd.CnName = strDeref(cnName)
		rd.EnName = strDeref(enName)
		rd.CnProbableCause = strDeref(cnProbCause)
		rd.EnProbableCause = strDeref(enProbCause)
		rd.CnSuggestion = strDeref(cnSuggestion)
		rd.EnSuggestion = strDeref(enSuggestion)
		rd.Description = strDeref(description)
		rd.EventType = eventType
		out = append(out, rd)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alarm_definitions: %w", err)
	}
	return out, nil
}

// ListSeverityLevels 拉取 alarm_severity_levels 全部 4 行（启动期一次性）。
func (r *PgRepository) ListSeverityLevels(ctx context.Context) ([]SeverityLevel, error) {
	sqlStr, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "code", "name", "display_order").
		From("alarm_severity_levels").
		OrderBy("display_order").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sql list severity: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query alarm_severity_levels: %w", err)
	}
	defer rows.Close()

	var out []SeverityLevel
	for rows.Next() {
		var sl SeverityLevel
		if err := rows.Scan(&sl.ID, &sl.Code, &sl.Name, &sl.DisplayOrder); err != nil {
			return nil, fmt.Errorf("scan severity_level: %w", err)
		}
		out = append(out, sl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate severity_levels: %w", err)
	}
	return out, nil
}

func strDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ── P3-04 写路径与查询过滤实现 ──────────────────────────────────────

const baseAlarmDefSelect = `
SELECT d.id, d.identifier, d.ne_type, d.cn_name, d.en_name,
       d.severity_id, d.event_type, d.cn_probable_cause, d.en_probable_cause,
       d.cn_suggestion, d.en_suggestion,
       d.is_show, d.description,
       l.code, l.name
FROM alarm_definitions d
JOIN alarm_severity_levels l ON d.severity_id = l.id`

// ListWithFilter 实现 WriteRepository。
func (r *PgRepository) ListWithFilter(ctx context.Context, f ListFilter) ([]ResolvedDefinition, int, error) {
	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 50
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	conds := sq.And{}
	if f.NeType != nil && *f.NeType != "" {
		conds = append(conds, sq.Eq{"d.ne_type": *f.NeType})
	}
	if f.LoadedFrom != nil {
		if *f.LoadedFrom == "" {
			conds = append(conds, sq.Expr("COALESCE(d.loaded_from, '') = ''"))
		} else {
			conds = append(conds, sq.Eq{"d.loaded_from": *f.LoadedFrom})
		}
	}
	if f.SeverityCode != nil {
		conds = append(conds, sq.Eq{"l.code": *f.SeverityCode})
	}
	if f.Keyword != nil && strings.TrimSpace(*f.Keyword) != "" {
		kw := "%" + strings.TrimSpace(*f.Keyword) + "%"
		conds = append(conds, sq.Or{
			sq.ILike{"d.identifier": kw},
			sq.ILike{"d.cn_name": kw},
			sq.ILike{"d.en_name": kw},
			sq.ILike{"d.description": kw},
		})
	}

	// COUNT
	cb := psql.Select("COUNT(*)").
		From("alarm_definitions d").
		Join("alarm_severity_levels l ON d.severity_id = l.id")
	if len(conds) > 0 {
		cb = cb.Where(conds)
	}
	cSQL, cArgs, err := cb.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count sql: %w", err)
	}
	var total int
	if err := r.pool.QueryRow(ctx, cSQL, cArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count alarm_definitions: %w", err)
	}

	// PAGE
	lb := psql.Select(
		"d.id", "d.identifier", "d.ne_type", "d.cn_name", "d.en_name",
		"d.severity_id", "d.event_type", "d.cn_probable_cause", "d.en_probable_cause",
		"d.cn_suggestion", "d.en_suggestion",
		"d.is_show", "d.description",
		"l.code", "l.name",
	).
		From("alarm_definitions d").
		Join("alarm_severity_levels l ON d.severity_id = l.id").
		OrderBy("d.identifier ASC").
		Limit(uint64(pageSize)).
		Offset(uint64((page - 1) * pageSize))
	if len(conds) > 0 {
		lb = lb.Where(conds)
	}
	lSQL, lArgs, err := lb.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list sql: %w", err)
	}
	rows, err := r.pool.Query(ctx, lSQL, lArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query alarm_definitions: %w", err)
	}
	defer rows.Close()

	out, err := scanResolvedDefs(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// GetByIdentifier 实现 WriteRepository。
func (r *PgRepository) GetByIdentifier(ctx context.Context, identifier string) (*ResolvedDefinition, error) {
	row := r.pool.QueryRow(ctx, baseAlarmDefSelect+" WHERE d.identifier = $1", identifier)
	rd, err := scanOneResolved(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUnknownIdentifier
	}
	if err != nil {
		return nil, fmt.Errorf("get alarm_definition %q: %w", identifier, err)
	}
	return rd, nil
}

// Create 实现 WriteRepository。
func (r *PgRepository) Create(ctx context.Context, in CreateInput) (*ResolvedDefinition, error) {
	if strings.TrimSpace(in.Identifier) == "" {
		return nil, fmt.Errorf("identifier is required")
	}
	if strings.TrimSpace(in.NeType) == "" {
		return nil, fmt.Errorf("ne_type is required")
	}

	// 反查 severity_id
	var severityID uuid.UUID
	if err := r.pool.QueryRow(ctx,
		`SELECT id FROM alarm_severity_levels WHERE code = $1`, in.SeverityCode,
	).Scan(&severityID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("severity_code %d not in alarm_severity_levels", in.SeverityCode)
		}
		return nil, fmt.Errorf("lookup severity_code: %w", err)
	}

	const insertSQL = `
INSERT INTO alarm_definitions (
    identifier, ne_type, cn_name, en_name, severity_id, event_type,
    cn_probable_cause, en_probable_cause, cn_suggestion, en_suggestion, is_show, description
) VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), $5, $6,
          NULLIF($7,''), NULLIF($8,''), NULLIF($9,''), NULLIF($10,''), $11, NULLIF($12,''))
RETURNING id`
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, insertSQL,
		in.Identifier, in.NeType, in.CnName, in.EnName, severityID, in.EventType,
		in.CnProbableCause, in.EnProbableCause, in.CnSuggestion, in.EnSuggestion, in.IsShow, in.Description,
	).Scan(&id); err != nil {
		return nil, fmt.Errorf("insert alarm_definition: %w", err)
	}

	return r.GetByIdentifier(ctx, in.Identifier)
}

// Update 实现 WriteRepository。
func (r *PgRepository) Update(ctx context.Context, identifier string, in UpdateInput) (*ResolvedDefinition, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("alarm_definitions").Where(sq.Eq{"identifier": identifier})

	dirty := false
	if in.NeType != nil {
		ub = ub.Set("ne_type", *in.NeType)
		dirty = true
	}
	if in.CnName != nil {
		ub = ub.Set("cn_name", nullIfEmptyAny(*in.CnName))
		dirty = true
	}
	if in.EnName != nil {
		ub = ub.Set("en_name", nullIfEmptyAny(*in.EnName))
		dirty = true
	}
	if in.SeverityCode != nil {
		var sid uuid.UUID
		if err := r.pool.QueryRow(ctx,
			`SELECT id FROM alarm_severity_levels WHERE code = $1`, *in.SeverityCode,
		).Scan(&sid); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("severity_code %d not in alarm_severity_levels", *in.SeverityCode)
			}
			return nil, fmt.Errorf("lookup severity_code: %w", err)
		}
		ub = ub.Set("severity_id", sid)
		dirty = true
	}
	if in.EventType != nil {
		ub = ub.Set("event_type", *in.EventType)
		dirty = true
	}
	if in.CnProbableCause != nil {
		ub = ub.Set("cn_probable_cause", nullIfEmptyAny(*in.CnProbableCause))
		dirty = true
	}
	if in.EnProbableCause != nil {
		ub = ub.Set("en_probable_cause", nullIfEmptyAny(*in.EnProbableCause))
		dirty = true
	}
	if in.CnSuggestion != nil {
		ub = ub.Set("cn_suggestion", nullIfEmptyAny(*in.CnSuggestion))
		dirty = true
	}
	if in.EnSuggestion != nil {
		ub = ub.Set("en_suggestion", nullIfEmptyAny(*in.EnSuggestion))
		dirty = true
	}
	if in.IsShow != nil {
		ub = ub.Set("is_show", *in.IsShow)
		dirty = true
	}
	if in.Description != nil {
		ub = ub.Set("description", nullIfEmptyAny(*in.Description))
		dirty = true
	}

	if !dirty {
		return r.GetByIdentifier(ctx, identifier)
	}

	uSQL, uArgs, err := ub.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update sql: %w", err)
	}
	tag, err := r.pool.Exec(ctx, uSQL, uArgs...)
	if err != nil {
		return nil, fmt.Errorf("update alarm_definition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrUnknownIdentifier
	}
	return r.GetByIdentifier(ctx, identifier)
}

// Delete 实现 WriteRepository。
func (r *PgRepository) Delete(ctx context.Context, identifier string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM alarm_definitions WHERE identifier = $1`, identifier)
	if err != nil {
		return false, fmt.Errorf("delete alarm_definition %q: %w", identifier, err)
	}
	return tag.RowsAffected() > 0, nil
}

// UnknownStats 实现 WriteRepository。
//
// 聚合 alarms_active 中 is_unknown=true 的 alarm_identifier 频次；
// productID 非空时按 devices.product_id 关联过滤。
func (r *PgRepository) UnknownStats(ctx context.Context, productID *uuid.UUID, days int) ([]UnknownAlarmStat, error) {
	if days <= 0 {
		days = 7
	}
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	conds := sq.And{
		sq.Eq{"a.is_unknown": true},
		sq.Expr("a.raised_at >= NOW() - make_interval(days => ?)", days),
	}

	from := "alarms_active a"
	if productID != nil {
		from = "alarms_active a JOIN devices dv ON a.device_id = dv.id"
		conds = append(conds, sq.Eq{"dv.product_id": *productID})
	}

	qb := psql.Select(
		"a.alarm_identifier",
		"COUNT(*)",
		"MAX(a.raised_at)",
		"COALESCE(MAX(a.alarm_type), '')",
	).
		From(from).
		Where(conds).
		GroupBy("a.alarm_identifier").
		OrderBy("COUNT(*) DESC").
		Limit(200)

	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build unknown-stats sql: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query unknown-stats: %w", err)
	}
	defer rows.Close()

	var out []UnknownAlarmStat
	for rows.Next() {
		var (
			s        UnknownAlarmStat
			lastSeen time.Time
		)
		if err := rows.Scan(&s.Identifier, &s.Count, &lastSeen, &s.NeType); err != nil {
			return nil, fmt.Errorf("scan unknown-stat row: %w", err)
		}
		s.LastSeenAt = lastSeen.UTC().Format(time.RFC3339)
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListNeTypes 实现 WriteRepository(T-0179 drill-down 一级视图)。
//
// 按 (ne_type, loaded_from) 双键聚合;COUNT FILTER 一次扫表算 4 个严重级计数。
// loaded_from NULL 通过 COALESCE 为 ” 归到"未知 XML 来源"行。
//
// 期望规模 ~442 行 8 个 ne_type → 8-16 行返回,无需分页。
func (r *PgRepository) ListNeTypes(ctx context.Context) ([]NeTypeStat, error) {
	const aggSQL = `
SELECT d.ne_type,
       COALESCE(d.loaded_from, '')                                 AS loaded_from,
       COUNT(*)                                                    AS total,
       COUNT(*) FILTER (WHERE l.code = 31001)                      AS critical_cnt,
       COUNT(*) FILTER (WHERE l.code = 31002)                      AS major_cnt,
       COUNT(*) FILTER (WHERE l.code = 31003)                      AS minor_cnt,
       COUNT(*) FILTER (WHERE l.code = 31004)                      AS warning_cnt
  FROM alarm_definitions d
  JOIN alarm_severity_levels l ON d.severity_id = l.id
 GROUP BY d.ne_type, COALESCE(d.loaded_from, '')
 ORDER BY d.ne_type ASC, loaded_from ASC`

	rows, err := r.pool.Query(ctx, aggSQL)
	if err != nil {
		return nil, fmt.Errorf("query ne-types agg: %w", err)
	}
	defer rows.Close()

	var out []NeTypeStat
	for rows.Next() {
		var s NeTypeStat
		if err := rows.Scan(
			&s.NeType, &s.LoadedFrom, &s.Total,
			&s.CriticalCnt, &s.MajorCnt, &s.MinorCnt, &s.WarningCnt,
		); err != nil {
			return nil, fmt.Errorf("scan ne-type-stat row: %w", err)
		}
		// source/deletable 由 NeTypes handler 据 sidecar 回填(repo 不做文件 IO)。
		out = append(out, s)
	}
	return out, rows.Err()
}

// ── helpers ─────────────────────────────────────────────────────────

// nullIfEmptyAny 是 P3-04 的辅助：空字符串 → nil（用于 SQL NULL）；
// loader.go 已有同名 nullIfEmpty 但语义不同（返回 sql.NullString），故避开重名。
func nullIfEmptyAny(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func scanOneResolved(row pgx.Row) (*ResolvedDefinition, error) {
	var rd ResolvedDefinition
	var (
		cnName, enName, cnProbCause, enProbCause, cnSuggestion, enSuggestion, description *string
		eventType                                                                         *int
	)
	if err := row.Scan(
		&rd.ID, &rd.Identifier, &rd.NeType, &cnName, &enName,
		&rd.SeverityID, &eventType, &cnProbCause, &enProbCause,
		&cnSuggestion, &enSuggestion,
		&rd.IsShow, &description,
		&rd.SeverityCode, &rd.SeverityName,
	); err != nil {
		return nil, err
	}
	rd.CnName = strDeref(cnName)
	rd.EnName = strDeref(enName)
	rd.CnProbableCause = strDeref(cnProbCause)
	rd.EnProbableCause = strDeref(enProbCause)
	rd.CnSuggestion = strDeref(cnSuggestion)
	rd.EnSuggestion = strDeref(enSuggestion)
	rd.Description = strDeref(description)
	rd.EventType = eventType
	return &rd, nil
}

func scanResolvedDefs(rows pgx.Rows) ([]ResolvedDefinition, error) {
	var out []ResolvedDefinition
	for rows.Next() {
		var rd ResolvedDefinition
		var (
			cnName, enName, cnProbCause, enProbCause, cnSuggestion, enSuggestion, description *string
			eventType                                                                         *int
		)
		if err := rows.Scan(
			&rd.ID, &rd.Identifier, &rd.NeType, &cnName, &enName,
			&rd.SeverityID, &eventType, &cnProbCause, &enProbCause,
			&cnSuggestion, &enSuggestion,
			&rd.IsShow, &description,
			&rd.SeverityCode, &rd.SeverityName,
		); err != nil {
			return nil, fmt.Errorf("scan alarm_definition: %w", err)
		}
		rd.CnName = strDeref(cnName)
		rd.EnName = strDeref(enName)
		rd.CnProbableCause = strDeref(cnProbCause)
		rd.EnProbableCause = strDeref(enProbCause)
		rd.CnSuggestion = strDeref(cnSuggestion)
		rd.EnSuggestion = strDeref(enSuggestion)
		rd.Description = strDeref(description)
		rd.EventType = eventType
		out = append(out, rd)
	}
	return out, rows.Err()
}

// DeleteOrphansSince 删除 updated_at < since 的 alarm_definitions 行 — reload
// destructive 语义对齐 parammodel:Loader UPSERT 触发 BEFORE UPDATE 触发器把
// updated_at 改 NOW(),所以本次未被触达的旧定义 updated_at 会保留旧值 < since。
//
// 副作用:
//   - alarm_definitions 行物理删除
//   - 若 alarm_history / alarms_active 有外键 ON DELETE SET NULL/CASCADE 由 FK
//     自身保证;repo 不做手工 cascade
//   - Loader 下次扫描发现 XML 仍在会重建对应行(self-healing,与 parammodel 一致)
func (r *PgRepository) DeleteOrphansSince(ctx context.Context, since time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM alarm_definitions WHERE updated_at < $1`,
		since,
	)
	if err != nil {
		return 0, fmt.Errorf("delete orphan alarm_definitions: %w", err)
	}
	return tag.RowsAffected(), nil
}
