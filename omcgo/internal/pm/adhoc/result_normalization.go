package adhoc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/omcgo/omcgo/internal/pm/resultnorm"
)

type indicatorMetadataQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// NewIndicatorMetadataLookup returns a batch lookup for PM result normalization metadata.
func NewIndicatorMetadataLookup(db indicatorMetadataQuerier) IndicatorMetadataLookup {
	return func(ctx context.Context, metricPaths []string) (map[string]resultnorm.Metadata, error) {
		out := make(map[string]resultnorm.Metadata, len(metricPaths))
		if len(metricPaths) == 0 {
			return out, nil
		}
		if db == nil {
			return out, fmt.Errorf("nil indicator metadata db")
		}
		const q = `
SELECT id, unit_id, statis_type FROM perf_indicators_enb WHERE id = ANY($1)
UNION ALL
SELECT id, unit_id, statis_type FROM perf_indicators_gnb WHERE id = ANY($1)
UNION ALL
SELECT id, unit_id, statis_type FROM perf_indicators_gsm WHERE id = ANY($1)`
		rows, err := db.Query(ctx, q, metricPaths)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			var unit, statisType *string
			if err := rows.Scan(&id, &unit, &statisType); err != nil {
				return nil, err
			}
			if _, exists := out[id]; exists {
				continue
			}
			meta := resultnorm.Metadata{}
			if unit != nil {
				meta.Unit = *unit
			}
			if statisType != nil {
				meta.StatisType = *statisType
			}
			out[id] = meta
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return out, nil
	}
}
