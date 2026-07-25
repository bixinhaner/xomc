package adhoc

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
)

func (r *PgRepository) syncStreamingTask(ctx context.Context, task *Task, enabled bool) error {
	if r.streamRepo == nil || task == nil {
		return nil
	}
	rules, err := r.resolveStreamingRules(ctx, task.MetricPaths)
	if err != nil {
		return err
	}
	members, err := r.resolveStreamingMembers(ctx, task)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		return fmt.Errorf("PM aggregation task resolved no devices")
	}
	granularities := make([]pmstream.Granularity, 0, len(task.Granularities))
	for _, value := range task.Granularities {
		granularities = append(granularities, pmstream.Granularity(value))
	}
	_, err = r.streamRepo.Save(ctx, pmstream.SaveTaskRequest{
		TaskID: task.ID, Name: task.Name, Enabled: enabled,
		Visibility: string(normalizeVisibility(task.Visibility)), Creator: task.Creator,
		Technology: task.Technology, Dimension: pmstream.Dimension(task.Dimension),
		Granularities: granularities, ObjectLDNs: task.ObjectLDNs,
		Metrics: rules, Members: members,
	})
	if err != nil {
		return fmt.Errorf("save PM streaming task version: %w", err)
	}
	return nil
}

func (r *PgRepository) resolveStreamingRules(
	ctx context.Context,
	paths []string,
) ([]pmstream.MetricRule, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("PM aggregation task has no metrics")
	}
	const query = `
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, '')
  FROM perf_indicators_enb WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, '')
  FROM perf_indicators_gnb WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(is_counter, '0'), COALESCE(statis_type, '')
  FROM perf_indicators_gsm WHERE id = ANY($1)`
	rows, err := r.pool.Query(ctx, query, paths)
	if err != nil {
		return nil, fmt.Errorf("resolve PM aggregation metric rules: %w", err)
	}
	defer rows.Close()
	resolved := make(map[string]pmstream.MetricRule, len(paths))
	for rows.Next() {
		var path, isCounter, statisType string
		if err := rows.Scan(&path, &isCounter, &statisType); err != nil {
			return nil, fmt.Errorf("scan PM aggregation metric rule: %w", err)
		}
		if _, exists := resolved[path]; exists {
			continue
		}
		metricType := "kpi"
		if isCounter == "1" {
			metricType = "counter"
		}
		resolved[path] = pmstream.MetricRule{
			MetricID: path, MetricPath: path, MetricType: metricType,
			Aggregation: streamAggregationOp(statisType),
		}
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
		"d.id", "d.serial_number", "g.id", "g.name",
	).From("device_group_members gm").
		Join("devices d ON d.id = gm.device_id").
		Join("device_groups g ON g.id = gm.group_id").
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
