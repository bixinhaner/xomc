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

// rawFallbackGranularity 是回退口径读取的原始明细粒度（pm_metrics raw 表本身就是 15min 粒度，
// 与性能仪表板默认模板 15min 直读 pm_metrics 原表的容错口径对齐，见 pm/handler.go:296-297）。
const rawFallbackGranularity = "15min"

// buildRawNetworkKPISeriesQuery 构建「直读原始明细 pm_metrics 现场汇总成全网一条线」的回退 SQL
// （纯函数，便于单测）。
//
// 背景（issue #359）：首页 KPI 折线图只读每小时预聚合表 pm_adhoc_aggregation_results。刚灌数
// 未到整点 / continuous scheduler 未起 / 聚合落后时该表为空，首页裸空白，但同期性能仪表板默认
// 模板能 15min 直读 pm_metrics 原表出图——这是两条取数链路容错能力的结构性差异。本回退让首页在
// 预聚合缺数据时也能 15min 直读原始明细现场汇总，与性能仪表板对齐容错。
//
// 汇总口径：与 aggregator.queryNetworkTable（全网一条总线）一致——按 metric_path + time GROUP BY
// （不带任何设备/小区实体键），算子按 statis_type 路由（sum/avg/max/min；未知按 sum），把全网所有
// 设备/小区的同指标同时间桶汇成一条全网线。只读 15min 原始明细（granularity=15min）。
//
// 成本：10 万级全网即时汇总有代价，故调用方对回退默认时窗设下限（见 service.go GetKPITimeSeries）。
func buildRawNetworkKPISeriesQuery(codes []string, startTimeArg, endTimeArg any) (string, []any, error) {
	// 算子按 statis_type 路由（与 aggregator.queryNetworkTable 同范式）。
	const aggValueExpr = `
		CASE MIN(statis_type)
			WHEN 'sum' THEN SUM(metric_value)
			WHEN 'avg' THEN AVG(metric_value)
			WHEN 'max' THEN MAX(metric_value)
			WHEN 'min' THEN MIN(metric_value)
			ELSE SUM(metric_value)
		END`
	return storage.Psql.Select("metric_path", "time", aggValueExpr+" AS metric_value").
		From("pm_metrics").
		Where("metric_path = ANY(?)", codes).
		Where(sq.Eq{"granularity": rawFallbackGranularity}).
		Where(sq.GtOrEq{"time": startTimeArg}).
		Where(sq.LtOrEq{"time": endTimeArg}).
		GroupBy("metric_path", "time").
		OrderBy("metric_path", "time ASC").
		ToSql()
}
