package aggregator

import (
	"context"
	"fmt"
)

// AggregateDeviceGroup 把设备维度聚合表 deviceTarget 中桶内的 counter 行按
// device_group 维度再聚合一次写入 groupTarget。
//
// JOIN 链：deviceTarget m → devices d (oui+sn 双键) → device_group_members dgm (device_id)
//
//	→ GROUP BY dgm.group_id, m.metric_path, m.statis_type
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
	numberProcess, err := a.numberProcess(ctx)
	if err != nil {
		return 0, err
	}
	sql, args := buildDeviceGroupSQLWithNumberProcess(deviceTarget, groupTarget, w, numberProcess)
	tag, err := a.execAggregateSQL(ctx, sql, args...)
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
//
// 时刻语义（issue #395 修复）：
//   - deviceTarget（pm_metrics_*）是**已按自然桶分行**的设备维度聚合表，每行自带
//     正确的 time/start_time/end_time。组聚合必须按源行自身的桶时刻分桶，
//     GROUP BY 增加 m.time/m.start_time/m.end_time，写入也取源行这三列，
//     而非窗口起点 w.Start。
//   - 修复前：GROUP BY 不含 time → 整窗内所有源小时被压成一个聚合点；写入 time
//     硬编码为 w.Start；cron 用 [end-1h,end) 窗口 + WHERE 按 end_time 命中，
//     使 device 行（time=T, end_time=T+1h）落进窗口被标成 T+1h → 恒后移 1 格。
//   - 修复后：每个源小时各成一行，写入 time 与设备单维度表逐档对齐、无偏移；
//     宽窗补算自然产出多行。
//
// 源筛选窗口（#516 分区裁剪）：与 buildCountersSQL 同步，按**分区列** time 落入
// 半开窗口 [w.Start, w.End)（time >= w.Start AND time < w.End）。deviceTarget 为
// TimescaleDB 超表、按 time 列分区，改用分区列过滤后只命中目标分片、走索引（分区裁剪），
// 消除原按非分区列 start_time 全表扫。等价性：#479 已统一 time == start_time，查的是
// 同一批源行、聚合行为完全不变；半开区间语义不变；args 顺序不变（$2=w.Start, $3=w.End）。
func buildDeviceGroupSQL(deviceTarget, groupTarget string, w WindowSpec) (string, []any) {
	return buildDeviceGroupSQLWithNumberProcess(deviceTarget, groupTarget, w, "")
}

func buildDeviceGroupSQLWithNumberProcess(deviceTarget, groupTarget string, w WindowSpec, numberProcess string) (string, []any) {
	conflictTarget := conflictTargetForTable(groupTarget)
	withID := targetHasIDColumn(groupTarget)

	// device_group 快表按「组 × 制式 × 指标 × 源桶时刻」拆行：
	// SELECT 带出 d.technology + 源行三时刻列、GROUP BY 加制式与桶时刻、
	// insertCols 加 technology 列、冲突列尾部含 technology。
	insertCols := "device_group_id, technology, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time, ingest_time, extra"
	selectIDExpr := ""
	if withID {
		insertCols = "id, " + insertCols
		selectIDExpr = "gen_random_uuid(),\n    "
	}

	aggregateValue := `CASE m.statis_type
        WHEN 'sum' THEN SUM(m.metric_value)
        WHEN 'avg' THEN AVG(m.metric_value)
        WHEN 'max' THEN MAX(m.metric_value)
        WHEN 'min' THEN MIN(m.metric_value)
    END`
	normalizedValue := normalizeSQLValue(aggregateValue, "im.unit_id", "im.statis_type", "$4", "m.metric_path")

	sql := fmt.Sprintf(`
WITH %s
INSERT INTO %s (%s)
SELECT
    %sdgm.group_id,
    d.technology,
    m.metric_path,
    'counter',
    %s,
    m.statis_type,
    $1,
    m.time,
    m.start_time,
    m.end_time,
    NOW(),
    NULL::jsonb
FROM %s m
LEFT JOIN indicator_meta im ON im.id = m.metric_path
JOIN device_dim d
  ON d.oui = m.device_oui AND d.serial_number = m.device_sn
JOIN device_group_member_dim dgm
  ON dgm.device_id = d.id
WHERE m.metric_type = 'counter'
  AND m.time >= $2
  AND m.time <  $3
  AND m.statis_type IN ('sum','avg','max','min')
GROUP BY dgm.group_id, d.technology, m.metric_path, m.statis_type, m.time, m.start_time, m.end_time, im.unit_id, im.statis_type
ON CONFLICT %s DO UPDATE SET
    metric_value = EXCLUDED.metric_value,
    ingest_time  = NOW()`,
		indicatorMetaSQL(),
		groupTarget, insertCols,
		selectIDExpr,
		normalizedValue,
		deviceTarget,
		conflictTarget,
	)
	return sql, []any{string(w.Granularity), w.Start, w.End, numberProcess}
}
