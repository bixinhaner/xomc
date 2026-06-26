package dashboard

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// kpi_summary_query.go —— 首页 KPI 卡 kpi_overview 全网最新值取数（issue HD01 修复）。
//
// 背景：原实现走 kpiRepo.Query(KPIFilter{StartTime: now-24h, PageSize: 10, SortBy: time desc})，
// 拿 24h 内 ORDER BY time DESC 的前 10 行 KPI，再按 KPIName 去重写入 KPIOverview。
// 这是「top-10 行」上限而非「每个指标最新一行」：当全网 KPI 指标数 ≫ 10（真实环境必然），
// 同一采集窗口写入的同时刻 ≥10 条 KPI 行里只有恰好排进 top-10 的那批 metric_path 会出现，
// 像 UE_ACTIVE 这种 indicator 是否上榜完全看 ORDER BY time DESC 同 time 行的 tiebreak 运气。
// 表现：首页「活跃 UE」卡常驻显示 0；前端用 `?? 0` 兜底，把「未上榜」与「真 0」混在一起。
//
// 修复：改走 `SELECT DISTINCT ON (metric_path) ... ORDER BY metric_path, time DESC`——以指标
// 编号为去重键，每个指标取时间窗内最新一行；LIMIT 是「不同 metric_path 上限」，给一个保守上界
// (latestKPISummaryLimit) 防 OOM，同时远大于系统总指标数。

// latestKPISummaryLimit 是首页 KPIOverview 一次性返回的「不同指标」上限。
// DISTINCT ON 已把行集压缩到 |distinct metric_path|，500 足以覆盖全网所有出现过的 KPI 编号，
// 同时给个明确上界防 OOM（pm_metrics 是时序大表）。
const latestKPISummaryLimit = 500

// buildLatestKPIPerNameQuery 构建「时间窗内按指标编号各取最新一行 KPI」的 SQL（纯函数，便于单测）。
//
// 用于 GetSummary 填充 KPIOverview：每个 metric_path 在 [startTime, endTime] 内的最近一次值。
// 全网视图——不限设备/小区/制式；不限 granularity（同一指标可能 15min/hourly 同时存在，取最新）。
//   - 去重键：metric_path（= K/C 指标编号）。
//   - 排序：metric_path, time DESC —— PostgreSQL DISTINCT ON 保留每组 ORDER 首行，即时间窗内最新。
//   - LIMIT：latestKPISummaryLimit，按"不同指标数"上界（非"行数"上界）。
func buildLatestKPIPerNameQuery(startTimeArg, endTimeArg any) (string, []any, error) {
	// DISTINCT ON 必须与 ORDER BY 首键一致（metric_path），由 PG 强校验。
	// squirrel 没有 DISTINCT ON 一等支持，把它嵌入第一列字符串原样输出即可。
	return storage.Psql.Select("DISTINCT ON (metric_path) metric_path", "metric_value").
		From("pm_metrics").
		Where(sq.Eq{"metric_type": "kpi"}).
		Where(sq.GtOrEq{"time": startTimeArg}).
		Where(sq.LtOrEq{"time": endTimeArg}).
		OrderBy("metric_path", "time DESC").
		Limit(latestKPISummaryLimit).
		ToSql()
}
