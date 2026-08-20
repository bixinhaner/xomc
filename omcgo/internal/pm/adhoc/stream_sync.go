package adhoc

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
)

func (r *PgRepository) syncStreamingTask(ctx context.Context, task *Task, enabled bool) error {
	if r.streamRepo == nil || task == nil {
		return nil
	}
	outputMetricPaths := streamingTaskOutputMetricPaths(task, nil)
	if task.IsBuiltin {
		enabledPaths, err := r.resolveEnabledStreamingMetricPaths(ctx, task.Technology)
		if err != nil {
			return err
		}
		outputMetricPaths = streamingTaskOutputMetricPaths(task, enabledPaths)
	}
	rules, err := r.resolveStreamingRules(
		ctx,
		task.Technology,
		outputMetricPaths,
	)
	if err != nil {
		return err
	}
	counters, err := r.resolveStreamingCounters(ctx, task.Technology, rules)
	if err != nil {
		return err
	}
	members, err := r.resolveStreamingMembers(ctx, task)
	if err != nil {
		return err
	}
	if shouldRejectEmptyStreamingMembers(task, members) {
		return fmt.Errorf("PM aggregation task resolved no devices")
	}
	granularities := streamingRollupGranularities()
	_, err = r.streamRepo.Save(ctx, pmstream.SaveTaskRequest{
		TaskID: task.ID, Name: task.Name, Enabled: enabled,
		Visibility: string(normalizeVisibility(task.Visibility)), Creator: streamingTaskCreator(task),
		Technology: task.Technology, Dimension: pmstream.Dimension(task.Dimension),
		Granularities: granularities, ObjectLDNs: task.ObjectLDNs,
		Metrics: rules, Counters: counters, Members: members, PlannedEndAt: task.PlannedEndAt,
		SourceUpdatedAt: task.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("save PM streaming task version: %w", err)
	}
	return nil
}

func shouldRejectEmptyStreamingMembers(task *Task, members []pmstream.TaskMember) bool {
	return task != nil && !task.IsBuiltin && len(members) == 0
}

func streamingTaskCreator(task *Task) string {
	if task == nil || task.Creator == "" {
		return "system"
	}
	return task.Creator
}

func streamingTaskOutputMetricPaths(task *Task, enabledPaths []string) []string {
	if task == nil {
		return nil
	}
	if !task.IsBuiltin {
		return streamingOutputMetricPaths(task.MetricPaths, nil)
	}
	return streamingOutputMetricPaths(task.MetricPaths, enabledPaths)
}

func streamingRollupGranularities() []pmstream.Granularity {
	return []pmstream.Granularity{
		pmstream.GranularityHourly,
		pmstream.GranularityDaily,
		pmstream.GranularityWeekly,
		pmstream.GranularityMonthly,
	}
}

func (r *PgRepository) resolveStreamingRules(
	ctx context.Context,
	technology string,
	paths []string,
) ([]pmstream.MetricRule, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("PM aggregation task has no metrics")
	}
	const query = `
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, ''), COALESCE(arithmetic, ''), 'ENB'
  FROM perf_indicators_enb WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, ''), COALESCE(arithmetic, ''), 'GNB'
  FROM perf_indicators_gnb WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, ''), COALESCE(arithmetic, ''), 'GSM'
  FROM perf_indicators_gsm WHERE id = ANY($1)`
	rows, err := r.pool.Query(ctx, query, paths)
	if err != nil {
		return nil, fmt.Errorf("resolve PM aggregation metric rules: %w", err)
	}
	defer rows.Close()
	resolved := make(map[string]pmstream.MetricRule, len(paths))
	for rows.Next() {
		var path, isCounter, statisType, formula, deviceType string
		if err := rows.Scan(&path, &isCounter, &statisType, &formula, &deviceType); err != nil {
			return nil, fmt.Errorf("scan PM aggregation metric rule: %w", err)
		}
		if technology != "" {
			expectedType, typeErr := indicatorDeviceTypeForTechnology(technology)
			if typeErr != nil {
				return nil, typeErr
			}
			if string(expectedType) != deviceType {
				continue
			}
		}
		if _, exists := resolved[path]; exists {
			continue
		}
		metricType := "kpi"
		if isCounter == "1" {
			metricType = "counter"
		}
		rule := pmstream.MetricRule{
			MetricID: path, MetricPath: path, MetricType: metricType,
			Aggregation: streamAggregationOp(statisType),
		}
		if metricType == "counter" {
			rule.Dependencies = []string{path}
		} else {
			formula = indicator.CompileRuntimeArithmetic(indicator.DeviceType(deviceType), formula)
			parsed, parseErr := expr.Parse(formula)
			if parseErr != nil {
				return nil, fmt.Errorf("parse PM aggregation KPI %s formula: %w", path, parseErr)
			}
			rule.Formula = formula
			rule.Dependencies = parsed.Identifiers()
			rule.Aggregation = pmstream.AggregationFormula
			if len(rule.Dependencies) == 0 {
				return nil, fmt.Errorf("PM aggregation KPI %s formula has no counter dependencies", path)
			}
		}
		resolved[path] = rule
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM aggregation metric rules: %w", err)
	}
	rules := make([]pmstream.MetricRule, 0, len(paths))
	for _, path := range paths {
		rule, ok := resolved[path]
		if !ok {
			return nil, fmt.Errorf("PM aggregation metric %s has no registered metadata", path)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *PgRepository) resolveEnabledStreamingMetricPaths(
	ctx context.Context,
	technology string,
) ([]string, error) {
	deviceTypes, err := streamingEnabledDeviceTypes(technology)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var paths []string
	for _, deviceType := range deviceTypes {
		table := deviceType.EnabledTable()
		query, args, buildErr := storage.Psql.Select("DISTINCT indicator_id").
			From(table).
			Where(sq.Eq{"operator_code": "default"}).
			OrderBy("indicator_id").
			ToSql()
		if buildErr != nil {
			return nil, fmt.Errorf("build PM aggregation enabled metric SQL: %w", buildErr)
		}
		rows, queryErr := r.pool.Query(ctx, query, args...)
		if queryErr != nil {
			return nil, fmt.Errorf("resolve PM aggregation enabled metrics from %s: %w", table, queryErr)
		}
		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan PM aggregation enabled metric from %s: %w", table, err)
			}
			if path == "" {
				continue
			}
			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate PM aggregation enabled metrics from %s: %w", table, err)
		}
		rows.Close()
	}
	return paths, nil
}

func streamingEnabledDeviceTypes(technology string) ([]indicator.DeviceType, error) {
	if technology != "" {
		deviceType, err := indicatorDeviceTypeForTechnology(technology)
		if err != nil {
			return nil, err
		}
		return []indicator.DeviceType{deviceType}, nil
	}
	return []indicator.DeviceType{
		indicator.DeviceTypeENB,
		indicator.DeviceTypeGNB,
		indicator.DeviceTypeGSM,
	}, nil
}

func streamingOutputMetricPaths(taskPaths, enabledPaths []string) []string {
	seen := make(map[string]struct{}, len(taskPaths)+len(enabledPaths))
	out := make([]string, 0, len(taskPaths)+len(enabledPaths))
	for _, paths := range [][]string{taskPaths, enabledPaths} {
		for _, path := range paths {
			if path == "" {
				continue
			}
			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			out = append(out, path)
		}
	}
	return out
}

func (r *PgRepository) resolveStreamingCounters(
	ctx context.Context,
	technology string,
	metrics []pmstream.MetricRule,
) ([]pmstream.CounterRule, error) {
	seen := make(map[string]struct{})
	var paths []string
	for _, metric := range metrics {
		for _, dependency := range metric.Dependencies {
			if _, ok := seen[dependency]; ok {
				continue
			}
			seen[dependency] = struct{}{}
			paths = append(paths, dependency)
		}
	}
	const query = `
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, ''), 'ENB'
  FROM perf_indicators_enb WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, ''), 'GNB'
  FROM perf_indicators_gnb WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, ''), 'GSM'
  FROM perf_indicators_gsm WHERE id = ANY($1)`
	rows, err := r.pool.Query(ctx, query, paths)
	if err != nil {
		return nil, fmt.Errorf("resolve PM aggregation counter dependencies: %w", err)
	}
	defer rows.Close()
	resolved := make(map[string]pmstream.CounterRule, len(paths))
	for rows.Next() {
		var path, isCounter, statisType, deviceType string
		if err := rows.Scan(&path, &isCounter, &statisType, &deviceType); err != nil {
			return nil, fmt.Errorf("scan PM aggregation counter dependency: %w", err)
		}
		if technology != "" {
			expectedType, typeErr := indicatorDeviceTypeForTechnology(technology)
			if typeErr != nil {
				return nil, typeErr
			}
			if string(expectedType) != deviceType {
				continue
			}
		}
		if isCounter != "1" {
			continue
		}
		resolved[path] = pmstream.CounterRule{
			MetricPath: path, Aggregation: streamAggregationOp(statisType),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM aggregation counter dependencies: %w", err)
	}
	out := make([]pmstream.CounterRule, 0, len(paths))
	for _, path := range paths {
		rule, ok := resolved[path]
		if !ok {
			return nil, fmt.Errorf("PM aggregation counter dependency %s has no registered counter metadata", path)
		}
		out = append(out, rule)
	}
	return out, nil
}

func streamAggregationOp(statisType string) pmstream.AggregationOp {
	switch strings.ToLower(statisType) {
	case "avg", "pct":
		return pmstream.AggregationAvg
	case "min":
		return pmstream.AggregationMin
	case "max":
		return pmstream.AggregationMax
	default:
		return pmstream.AggregationSum
	}
}

func (r *PgRepository) resolveStreamingMembers(
	ctx context.Context,
	task *Task,
) ([]pmstream.TaskMember, error) {
	if task.Dimension == DimensionBand {
		return r.resolveBandMembers(ctx, task)
	}
	if task.Dimension == DimensionDeviceGroup {
		return r.resolveDeviceGroupMembers(ctx, task)
	}
	builder := storage.Psql.Select(
		"d.id", "d.serial_number", "d.product_id", "COALESCE(p.product_name, '')",
	).From("devices d").
		LeftJoin("products p ON p.id = d.product_id").
		Where("d.deleted_at IS NULL")
	if task.Technology != "" {
		builder = builder.Where(sq.Eq{"d.technology": task.Technology})
	}
	if len(task.DeviceSNs) > 0 {
		builder = builder.Where(sq.Eq{"d.serial_number": task.DeviceSNs})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build PM aggregation members SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM aggregation members: %w", err)
	}
	defer rows.Close()
	var members []pmstream.TaskMember
	for rows.Next() {
		var deviceID uuid.UUID
		var deviceSN, productName string
		var productID *uuid.UUID
		if err := rows.Scan(&deviceID, &deviceSN, &productID, &productName); err != nil {
			return nil, fmt.Errorf("scan PM aggregation member: %w", err)
		}
		key, name := streamingDimensionKey(task, deviceID, deviceSN, productID, productName)
		if key == "" {
			continue
		}
		members = append(members, pmstream.TaskMember{
			DeviceID: deviceID, DeviceSN: deviceSN, DimensionKey: key, DimensionName: name,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM aggregation members: %w", err)
	}
	return members, nil
}

// resolveStreamingDeviceBaseline captures the eligible device population once
// per technology. Built-in dimensions are reconciled sequentially, so using
// this shared baseline prevents devices registering mid-reconcile from making
// downstream dimensions expect events that the upstream network snapshot can
// never publish.
func (r *PgRepository) resolveStreamingDeviceBaseline(
	ctx context.Context,
	technology string,
) ([]uuid.UUID, error) {
	builder := storage.Psql.Select("id").From("devices").
		Where("deleted_at IS NULL")
	if technology != "" {
		builder = builder.Where(sq.Eq{"technology": technology})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build PM aggregation device baseline SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM aggregation device baseline: %w", err)
	}
	defer rows.Close()
	deviceIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var deviceID uuid.UUID
		if err := rows.Scan(&deviceID); err != nil {
			return nil, fmt.Errorf("scan PM aggregation device baseline: %w", err)
		}
		deviceIDs = append(deviceIDs, deviceID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM aggregation device baseline: %w", err)
	}
	return deviceIDs, nil
}

func streamingDimensionKey(
	task *Task,
	deviceID uuid.UUID,
	deviceSN string,
	productID *uuid.UUID,
	productName string,
) (string, string) {
	switch task.Dimension {
	case DimensionDevice:
		return deviceID.String(), deviceSN
	case DimensionAggregateGroup:
		return "AggregateGroup=" + task.ID.String(), task.Name
	case DimensionProduct:
		if productID == nil {
			return "", ""
		}
		return productID.String(), productName
	case DimensionNetwork:
		return "network", "Network"
	default:
		return deviceID.String(), deviceSN
	}
}

func (r *PgRepository) resolveDeviceGroupMembers(
	ctx context.Context,
	task *Task,
) ([]pmstream.TaskMember, error) {
	builder := storage.Psql.Select(
		"d.id", "d.serial_number",
		"COALESCE(g.id, '00000000-0000-4000-8000-000000000001'::uuid)",
		"COALESCE(g.name, '未分组设备')",
	).From("devices d").
		LeftJoin("device_group_members gm ON gm.device_id = d.id").
		LeftJoin("device_groups g ON g.id = gm.group_id").
		Where("d.deleted_at IS NULL")
	if task.Technology != "" {
		builder = builder.Where(sq.Eq{"d.technology": task.Technology})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build PM device-group members SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM device-group members: %w", err)
	}
	defer rows.Close()
	var members []pmstream.TaskMember
	for rows.Next() {
		var deviceID, groupID uuid.UUID
		var deviceSN, groupName string
		if err := rows.Scan(&deviceID, &deviceSN, &groupID, &groupName); err != nil {
			return nil, fmt.Errorf("scan PM device-group member: %w", err)
		}
		members = append(members, pmstream.TaskMember{
			DeviceID: deviceID, DeviceSN: deviceSN,
			DimensionKey: "DeviceGroup=" + groupID.String(), DimensionName: groupName,
		})
	}
	return members, rows.Err()
}

func (r *PgRepository) resolveBandMembers(
	ctx context.Context,
	task *Task,
) ([]pmstream.TaskMember, error) {
	query := `
SELECT d.id, d.serial_number, cb.cell_id, cb.band
  FROM device_dim d
  JOIN cell_band_dim cb ON cb.device_id = d.id
 WHERE ($1 = '' OR d.technology = $1)`
	rows, err := r.tsPool.Query(ctx, query, task.Technology)
	if err != nil {
		return nil, fmt.Errorf("query PM band members: %w", err)
	}
	defer rows.Close()
	var members []pmstream.TaskMember
	for rows.Next() {
		var deviceID uuid.UUID
		var deviceSN, cellID, band string
		if err := rows.Scan(&deviceID, &deviceSN, &cellID, &band); err != nil {
			return nil, fmt.Errorf("scan PM band member: %w", err)
		}
		members = append(members, pmstream.TaskMember{
			DeviceID: deviceID, DeviceSN: deviceSN,
			DimensionKey: "Band=" + band, DimensionName: "Band=" + band,
			ObjectLDN: cellID,
		})
	}
	return members, rows.Err()
}
