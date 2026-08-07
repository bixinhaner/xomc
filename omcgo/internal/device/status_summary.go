package device

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// TechnologyStatusSummary is the non-CPE device status snapshot used by
// scheduled operational summaries.
type TechnologyStatusSummary struct {
	Total       int64
	Online      int64
	Activated   int64
	ExcludedCPE int64
}

// TechnologyStatusSummarySet contains one row for each supported radio technology.
type TechnologyStatusSummarySet struct {
	ByTechnology map[model.Technology]TechnologyStatusSummary
}

// BuildTechnologyStatusSummaryQuery keeps the summary's data source and
// filtering semantics explicit so callers and tests can audit the SQL shape.
func BuildTechnologyStatusSummaryQuery() (string, []any, error) {
	cpePredicate := "COALESCE(" + cpeProductClassPredicate + ", FALSE)"
	query, args, err := storage.Psql.Select(
		"d.technology",
		"COUNT(DISTINCT d.id) FILTER (WHERE NOT "+cpePredicate+") AS total_count",
		"COUNT(DISTINCT d.id) FILTER (WHERE NOT "+cpePredicate+" AND d.is_online = TRUE) AS online_count",
		"COUNT(DISTINCT d.id) FILTER (WHERE NOT "+cpePredicate+" AND COALESCE(di.op_state, '0') = '1') AS activated_count",
		"COUNT(DISTINCT d.id) FILTER (WHERE "+cpePredicate+") AS excluded_cpe_count",
	).
		From("devices d").
		LeftJoin("device_info di ON di.device_id = d.id").
		Where("d.deleted_at IS NULL").
		Where(sq.Eq{"d.technology": []model.Technology{model.TechGSM, model.TechLTE, model.TechNR}}).
		GroupBy("d.technology").
		OrderBy("d.technology").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build technology status summary query: %w", err)
	}
	return query, args, nil
}

// ComputeTechnologyStatusSummary returns a single database snapshot. The
// LEFT JOIN keeps devices without device_info visible as not activated.
func (r *PgDeviceInfoRepository) ComputeTechnologyStatusSummary(ctx context.Context) (*TechnologyStatusSummarySet, error) {
	query, args, err := BuildTechnologyStatusSummaryQuery()
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query technology status summary: %w", err)
	}
	defer rows.Close()

	result := &TechnologyStatusSummarySet{ByTechnology: make(map[model.Technology]TechnologyStatusSummary, 3)}
	for rows.Next() {
		var technology model.Technology
		var summary TechnologyStatusSummary
		if err := rows.Scan(&technology, &summary.Total, &summary.Online, &summary.Activated, &summary.ExcludedCPE); err != nil {
			return nil, fmt.Errorf("scan technology status summary: %w", err)
		}
		result.ByTechnology[technology] = summary
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate technology status summary: %w", err)
	}
	return result, nil
}
