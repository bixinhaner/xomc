package kpi

import (
	"errors"
	"fmt"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// AggregateByStatisType 按 counter 元数据驱动的聚合方式（sum/avg/max/min/pct）合并多条原始 counter 值。
//
// 设计依据：docs/design/pm-kpi-pipeline-improvements.md §4.5 G5 自然桶预聚合。
// T-0098 的 perf_indicators_* 表已落 statis_type 字段，本函数是 G5 cron 消费它的核心算子。
// 对齐老 OMC 系统 perf_indicators.statis_type 五个业务枚举（sum/avg/max/min/pct），
// DB CHECK 约束（pm_metrics*.statis_type）亦同。
//
// G3 阶段本函数尚不在实时计算链路上调用（KPIEngine.Calculate 现走 SUM(counter_value) GROUP BY
// 在 SQL 层做聚合，过渡期保留）；G5 cron 启用后，counter 按粒度落 pm_metrics 后由此函数
// 进一步聚合到 hourly/daily/weekly/monthly。本阶段先打通字段消费链路 + 工具函数 ready。
//
// pct 是百分比/比率类指标，需要 arithmetic 上下文（分子分母 counter 一起算）才能正确聚合，
// 单纯传一组数值无意义 — 返回 error 让调用方走 G5 公式路径。
func AggregateByStatisType(values []float64, stype metrics.StatisType) (float64, error) {
	if len(values) == 0 {
		return 0, errors.New("aggregate: empty values")
	}
	switch stype {
	case metrics.StatisSum:
		var s float64
		for _, v := range values {
			s += v
		}
		return s, nil
	case metrics.StatisAvg:
		var s float64
		for _, v := range values {
			s += v
		}
		return s / float64(len(values)), nil
	case metrics.StatisMax:
		m := values[0]
		for _, v := range values[1:] {
			if v > m {
				m = v
			}
		}
		return m, nil
	case metrics.StatisMin:
		m := values[0]
		for _, v := range values[1:] {
			if v < m {
				m = v
			}
		}
		return m, nil
	case metrics.StatisPct:
		return 0, errors.New("aggregate: pct requires arithmetic context (numerator/denominator), not raw values")
	default:
		return 0, fmt.Errorf("aggregate: unknown statis_type %q", stype)
	}
}
