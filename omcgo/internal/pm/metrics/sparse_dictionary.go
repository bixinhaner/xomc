package metrics

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// sparseMetadataQuerier is the transaction-scoped database surface needed to
// resolve immutable sparse PM metadata.
type sparseMetadataQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type metricDefinition struct {
	path       string
	metricType MetricType
	statisType string
	unit       string
}

type resolvedMetricDefinition struct {
	metricDefinition
	id int64
}

func resolveMetricDictionary(
	ctx context.Context,
	tx sparseMetadataQuerier,
	defs []metricDefinition,
) (map[string]int64, error) {
	defs, err := canonicalMetricDefinitions(defs)
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		return map[string]int64{}, nil
	}

	paths := make([]string, len(defs))
	byPath := make(map[string]metricDefinition, len(defs))
	for i, def := range defs {
		paths[i] = def.path
		byPath[def.path] = def
	}

	existing, err := queryMetricDictionary(ctx, tx, paths)
	if err != nil {
		return nil, err
	}
	missing := make([]metricDefinition, 0, len(defs))
	for _, def := range defs {
		current, ok := existing[def.path]
		if !ok {
			missing = append(missing, def)
			continue
		}
		if err := validateMetricDefinition(def, current); err != nil {
			return nil, err
		}
	}
	if len(missing) > 0 {
		if err := insertMissingMetricDefinitions(ctx, tx, missing); err != nil {
			return nil, err
		}
	}

	resolved, err := queryMetricDictionary(ctx, tx, paths)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]int64, len(defs))
	for _, def := range defs {
		current, ok := resolved[def.path]
		if !ok {
			return nil, fmt.Errorf("resolve metric dictionary: metric path %q was not resolved", def.path)
		}
		if err := validateMetricDefinition(byPath[def.path], current); err != nil {
			return nil, err
		}
		ids[def.path] = current.id
	}
	return ids, nil
}

func canonicalMetricDefinitions(defs []metricDefinition) ([]metricDefinition, error) {
	byPath := make(map[string]metricDefinition, len(defs))
	for _, def := range defs {
		if def.path == "" {
			return nil, fmt.Errorf("canonical metric definitions: empty metric path")
		}
		if def.metricType != MetricTypeCounter && def.metricType != MetricTypeKPI {
			return nil, fmt.Errorf("canonical metric definitions: metric path %q has invalid metric type %q", def.path, def.metricType)
		}
		if existing, ok := byPath[def.path]; ok {
			if existing.metricType != def.metricType ||
				(existing.statisType != "" && def.statisType != "" && existing.statisType != def.statisType) ||
				(existing.unit != "" && def.unit != "" && existing.unit != def.unit) {
				return nil, fmt.Errorf("canonical metric definitions: duplicate metric definition for path %q is inconsistent", def.path)
			}
			if existing.statisType == "" {
				existing.statisType = def.statisType
			}
			if existing.unit == "" {
				existing.unit = def.unit
			}
			byPath[def.path] = existing
			continue
		}
		byPath[def.path] = def
	}
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	out := make([]metricDefinition, 0, len(paths))
	for _, path := range paths {
		out = append(out, byPath[path])
	}
	return out, nil
}

func queryMetricDictionary(ctx context.Context, tx sparseMetadataQuerier, paths []string) (map[string]resolvedMetricDefinition, error) {
	rows, err := tx.Query(ctx, `
SELECT metric_path, metric_id, metric_type,
       COALESCE(statis_type, ''), COALESCE(unit, '')
FROM pm_metric_dictionary
WHERE metric_path = ANY($1::text[])`, paths)
	if err != nil {
		return nil, fmt.Errorf("query metric dictionary: %w", err)
	}
	defer rows.Close()

	resolved := make(map[string]resolvedMetricDefinition, len(paths))
	for rows.Next() {
		var path string
		var id int64
		var metricType MetricType
		var statisType, unit string
		if err := rows.Scan(&path, &id, &metricType, &statisType, &unit); err != nil {
			return nil, fmt.Errorf("scan metric dictionary: %w", err)
		}
		if _, exists := resolved[path]; exists {
			return nil, fmt.Errorf("query metric dictionary: duplicate metadata row for path %q", path)
		}
		resolved[path] = resolvedMetricDefinition{metricDefinition: metricDefinition{
			path: path, metricType: metricType, statisType: statisType, unit: unit,
		}, id: id}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric dictionary: %w", err)
	}
	return resolved, nil
}

func validateMetricDefinition(want metricDefinition, got resolvedMetricDefinition) error {
	if want.metricType != got.metricType ||
		(want.statisType != "" && got.statisType != "" && want.statisType != got.statisType) ||
		(want.unit != "" && got.unit != "" && want.unit != got.unit) {
		return fmt.Errorf("resolve metric dictionary: incompatible immutable metadata for path %q: got type=%q statis_type=%q unit=%q, want type=%q statis_type=%q unit=%q", want.path, got.metricType, got.statisType, got.unit, want.metricType, want.statisType, want.unit)
	}
	return nil
}

func insertMissingMetricDefinitions(ctx context.Context, tx sparseMetadataQuerier, defs []metricDefinition) error {
	paths := make([]string, len(defs))
	types := make([]string, len(defs))
	statis := make([]string, len(defs))
	units := make([]string, len(defs))
	for i, def := range defs {
		paths[i] = def.path
		types[i] = string(def.metricType)
		statis[i] = def.statisType
		units[i] = def.unit
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO pm_metric_dictionary (metric_path, report_key, metric_type, statis_type, unit)
SELECT path, path, metric_type, NULLIF(statis_type,''), NULLIF(unit,'')
FROM unnest($1::text[], $2::text[], $3::text[], $4::text[])
  AS input(path, metric_type, statis_type, unit)
ON CONFLICT (metric_path) DO NOTHING`, paths, types, statis, units); err != nil {
		return fmt.Errorf("insert missing metric dictionary definitions: %w", err)
	}
	return nil
}

func resolveMetricSet(
	ctx context.Context,
	tx sparseMetadataQuerier,
	productKey, counterGroup, contentHash string,
	metricIDs []int64,
) (int64, error) {
	metricIDs, err := canonicalMetricIDs(metricIDs)
	if err != nil {
		return 0, err
	}

	sets, err := queryMetricSets(ctx, tx, productKey, counterGroup, contentHash)
	if err != nil {
		return 0, err
	}
	if len(sets) == 1 {
		if err := validateMetricSet(metricIDs, sets[0]); err != nil {
			return 0, err
		}
		return sets[0].id, nil
	}
	if len(sets) > 1 {
		return 0, fmt.Errorf("resolve metric set: duplicate metadata rows for product %q group %q", productKey, counterGroup)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO pm_metric_sets (product_key, counter_group, content_hash, metric_ids)
VALUES ($1, $2, decode($3, 'hex'), $4)
ON CONFLICT (product_key, counter_group, content_hash) DO NOTHING`, productKey, counterGroup, contentHash, metricIDs); err != nil {
		return 0, fmt.Errorf("insert missing metric set: %w", err)
	}

	sets, err = queryMetricSets(ctx, tx, productKey, counterGroup, contentHash)
	if err != nil {
		return 0, err
	}
	if len(sets) == 0 {
		return 0, fmt.Errorf("resolve metric set: product %q group %q was not resolved", productKey, counterGroup)
	}
	if len(sets) > 1 {
		return 0, fmt.Errorf("resolve metric set: duplicate metadata rows for product %q group %q", productKey, counterGroup)
	}
	if err := validateMetricSet(metricIDs, sets[0]); err != nil {
		return 0, err
	}
	return sets[0].id, nil
}

type resolvedMetricSet struct {
	id        int64
	metricIDs []int64
}

func queryMetricSets(ctx context.Context, tx sparseMetadataQuerier, productKey, counterGroup, contentHash string) ([]resolvedMetricSet, error) {
	rows, err := tx.Query(ctx, `
SELECT metric_set_id, metric_ids
FROM pm_metric_sets
WHERE product_key = $1
  AND counter_group = $2
  AND content_hash = decode($3, 'hex')`, productKey, counterGroup, contentHash)
	if err != nil {
		return nil, fmt.Errorf("query metric set: %w", err)
	}
	defer rows.Close()

	sets := make([]resolvedMetricSet, 0, 1)
	for rows.Next() {
		var set resolvedMetricSet
		if err := rows.Scan(&set.id, &set.metricIDs); err != nil {
			return nil, fmt.Errorf("scan metric set: %w", err)
		}
		sets = append(sets, set)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric sets: %w", err)
	}
	return sets, nil
}

func canonicalMetricIDs(ids []int64) ([]int64, error) {
	sorted := append([]int64(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			return nil, fmt.Errorf("canonical metric set: duplicate metric ID %d", sorted[i])
		}
	}
	return sorted, nil
}

func validateMetricSet(want []int64, got resolvedMetricSet) error {
	if len(want) != len(got.metricIDs) {
		return fmt.Errorf("resolve metric set: inconsistent metric set %d: stored metric IDs do not match content hash input", got.id)
	}
	for i := range want {
		if want[i] != got.metricIDs[i] {
			return fmt.Errorf("resolve metric set: inconsistent metric set %d: stored metric IDs do not match content hash input", got.id)
		}
	}
	return nil
}
