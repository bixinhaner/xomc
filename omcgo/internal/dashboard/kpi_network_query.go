package dashboard

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// kpi_network_query.go —— 首页 KPI 折线图取数改读「全网预聚合结果表」（阶段2）。
//
// 背景：首页 KPI 折线图原来直接读「每设备每小区」明细表（pm_metrics，metric_type='kpi'），
// 没有真正的全网汇总，10 万级规模下口径不成立。系统已有 3 条内置「全网」(network 维度)
// 连续聚合任务（LTE/NR/GSM 各一条），每小时把全库指标（counter 求和、KPI 重算）汇成
// 全网总线写入 pm_adhoc_aggregation_results（device_sn='AGGREGATED'）。本层把首页取数
// 改读这张预聚合表，直接拿口径正确的全网线。
//
// 查询键：从「精选 symbolic 别名 key」改成「指标编号(K/C 编号)」。前端可直接传编号；
// 老的精选 symbolic 别名经 kpi_alias.go 的 resolveKPIAliases 映射成编号（存量兼容）；
// none 项（库内无对应编号，如 LTE_CELL_AVAILABLE）沿用「返回空序列」语义。
//
// 制式定位：3 条 network 任务各只聚自己制式的指标，故同一指标编号只会出现在其对应制式
// 的那条任务结果里。按「编号 ∈ ANY(codes)」+「task_id ∈ 三条 network 任务」查询即可
// 唯一定位，无需面板额外存制式、也无需按前缀解析制式——满足设计「能唯一定位指标库条目」。

// networkAggregationTaskIDs 是 3 条内置「全网」(network 维度) 连续聚合任务的固定 ID
// （seed/000001 预置，LTE/NR/GSM 各一条）。首页全网线只读这三条任务的聚合结果。
var networkAggregationTaskIDs = []string{
	"0184dddd-0001-4000-8000-000000000001", // 内置-全网-LTE
	"0184dddd-0001-4000-8000-000000000002", // 内置-全网-NR
	"0184dddd-0001-4000-8000-000000000003", // 内置-全网-GSM
}

// networkResultGranularity 是首页折线图读取的预聚合粒度（内置 network 任务按小时聚合）。
const networkResultGranularity = "hourly"

// buildNetworkKPISeriesQuery 构建「按指标编号读全网预聚合结果表」的 SQL（纯函数，便于单测）。
//
// 过滤：task_id ∈ 三条 network 任务（覆盖全制式）+ metric_path ∈ codes + granularity=hourly
// + time ∈ [startTime,endTime]。按 metric_path,time 升序，便于回填时序。
// 不限制 metric_type：counter 与 KPI 都从同一张表按编号读（一面板一时刻只画一条线，无量纲问题）。
func buildNetworkKPISeriesQuery(codes []string, startTimeArg, endTimeArg any) (string, []any, error) {
	return storage.Psql.Select("metric_path", "time", "metric_value").
		From("pm_adhoc_aggregation_results").
		Where("task_id = ANY(?)", networkAggregationTaskIDs).
		Where("metric_path = ANY(?)", codes).
		Where(sq.Eq{"granularity": networkResultGranularity}).
		Where(sq.GtOrEq{"time": startTimeArg}).
		Where(sq.LtOrEq{"time": endTimeArg}).
		OrderBy("metric_path", "time ASC").
		ToSql()
}
