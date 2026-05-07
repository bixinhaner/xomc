package definition

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
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
			"d.cn_suggestion", "d.en_suggestion", "d.is_show",
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
			cnName, enName, cnProbCause, enProbCause, cnSugg, enSugg *string
			eventType                                                 *int
		)
		if err := rows.Scan(
			&rd.ID, &rd.Identifier, &rd.NeType, &cnName, &enName,
			&rd.SeverityID, &eventType, &cnProbCause, &enProbCause,
			&cnSugg, &enSugg, &rd.IsShow,
			&rd.SeverityCode, &rd.SeverityName,
		); err != nil {
			return nil, fmt.Errorf("scan alarm_definition: %w", err)
		}
		rd.CnName = strDeref(cnName)
		rd.EnName = strDeref(enName)
		rd.CnProbableCause = strDeref(cnProbCause)
		rd.EnProbableCause = strDeref(enProbCause)
		rd.CnSuggestion = strDeref(cnSugg)
		rd.EnSuggestion = strDeref(enSugg)
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
