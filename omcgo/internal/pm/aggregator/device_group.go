package aggregator

import (
	"context"
	"fmt"
)

// AggregateDeviceGroup 把设备维度聚合表 deviceTarget 中桶内的 counter 行按
// device_group 维度再聚合一次写入 groupTarget。
//
// JOIN 链：deviceTarget m → devices d (oui+sn 双键) → device_group_members dgm (device_id)
//   → GROUP BY dgm.group_id, m.metric_path, m.statis_type
//
// 注意：
//   - 仅 counter 行参与（KPI 跨设备求和无业务意义；如需 group 级 KPI 应在 group 维度
//     的 counter 写完后再走 KPIRouter，但因 KPIRouter 依赖具体设备 productClass，
//     当前阶段不支持 group 级 KPI 反算 — 留 G7 / 早上讨论）
//   - 落 pm_group_metrics_hourly 等表的 conflict target 含 device_group_id
//   - 聚合方式与 deviceTarget 行的 statis_type 保持一致（sum→SUM、avg→AVG、max→MAX）
//
// 返回写入行数。
func (a *Aggregator) AggregateDeviceGroup(ctx context.Context, deviceTarget, groupTarget string, w WindowSpec) (int, error) {
	sql, args := buildDeviceGroupSQL(deviceTarget, groupTarget, w)
	tag, err := a.db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("aggregator.AggregateDeviceGroup %s→%s: %w", deviceTarget, groupTarget, err)
	}
	return int(tag.RowsAffected()), nil
}

// buildDeviceGroupSQL 构造 device 聚合表 → device_group 聚合表的 INSERT。
//
// 设计要点：
//   - JOIN devices d ON d.oui=m.device_oui AND d.serial_number=m.device_sn
//     （devices 是 partition by carrier 的分区表，oui+sn 不是 UNIQUE 但实际唯一；
//     如同 oui+sn 跨 carrier 出现重复，会按 group 维度算两次 — 业务上不应发生）
//   - JOIN device_group_members dgm ON dgm.device_id = d.id
//     （dgm 表 UNIQUE constraint uq_dgm_device 保证 1 设备 ≤ 1 group）
//   - 不在 SQL 里走 ParamModel/Translator —— metric_path 已是 standardPath
//   - 聚合方式同 buildCountersSQL：CASE WHEN m.statis_type → SUM/AVG/MAX
func buildDeviceGroupSQL(deviceTarget, groupTarget string, w WindowSpec) (string, []any) {
	conflictTarget := conflictTargetForTable(groupTarget)
	withID := targetHasIDColumn(groupTarget)

	// device_group 快表按「组 × 制式 × 指标」拆行（设备组制式治本 B 方案）：
	// SELECT 带出 d.technology、GROUP BY 加制式、insertCols 加 technology 列、冲突列尾部含 technology。
	insertCols := "device_group_id, technology, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time, ingest_time, extra"
	selectIDExpr := ""
	if withID {
		insertCols = "id, " + insertCols
		selectIDExpr = "gen_random_uuid(),\n    "
	}

	sql := fmt.Sprintf(`
INSERT INTO %s (%s)
SELECT
    %sdgm.group_id,
    d.technology,
    m.metric_path,
    'counter',
    CASE m.statis_type
        WHEN 'sum' THEN SUM(m.metric_value)
        WHEN 'avg' THEN AVG(m.metric_value)
        WHEN 'max' THEN MAX(m.metric_value)
        WHEN 'min' THEN MIN(m.metric_value)
    END,
    m.statis_type,
    $1,
    $2,
    $2,
    $3,
    NOW(),
    NULL::jsonb
FROM %s m
JOIN device_dim d
  ON d.oui = m.device_oui AND d.serial_number = m.device_sn
JOIN device_group_member_dim dgm
  ON dgm.device_id = d.id
WHERE m.metric_type = 'counter'
  AND m.end_time >= $4
  AND m.end_time <  $5
  AND m.statis_type IN ('sum','avg','max','min')
GROUP BY dgm.group_id, d.technology, m.metric_path, m.statis_type
ON CONFLICT %s DO UPDATE SET
    metric_value = EXCLUDED.metric_value,
    ingest_time  = NOW()`,
		groupTarget, insertCols,
		selectIDExpr,
		deviceTarget,
		conflictTarget,
	)
	return sql, []any{string(w.Granularity), w.Start, w.End, w.Start, w.End}
}
