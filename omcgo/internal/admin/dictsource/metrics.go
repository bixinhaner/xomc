package dictsource

import "github.com/prometheus/client_golang/prometheus"

// Metrics 是数据字典数据源同步的 Prometheus 指标集合(T-0182)。
// 当前只有一个 CounterVec(每次同步结束打一行结果),覆盖手动 + 定时两条路径。
// 直方图与 gauge 留给后续基于实际数据规模评估。
type Metrics struct {
	syncTotal *prometheus.CounterVec
}

// NewMetrics 注册并返回 Metrics;reg nil 用匿名 Registry(测试)。
//
// dictionary_source_sync_total{result, trigger}
//   - result  = ok | failed | timeout
//   - trigger = manual | initial | switch | daily
//
// 单字典维度按需可加 dict_id label,但 dict_id 是 high-cardinality,
// v1 不引入(管理员可在 last_refresh_status / Grafana Loki 日志侧 drill-down)。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		syncTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dictionary_source_sync_total",
				Help: "Total dictionary source sync runs by result and trigger.",
			},
			[]string{"result", "trigger"},
		),
	}
	if reg != nil {
		reg.MustRegister(m.syncTotal)
	}
	return m
}

// Observe 记录一次同步结果。result + trigger 都被加进 Prometheus label 集。
func (m *Metrics) Observe(result, trigger string) {
	if m == nil || m.syncTotal == nil {
		return
	}
	m.syncTotal.WithLabelValues(result, trigger).Inc()
}

// SyncTrigger 枚举(label trigger 取值)。
const (
	TriggerManual  = "manual"  // 用户手动点"刷新"按钮
	TriggerInitial = "initial" // CreateDictionary 后首次同步
	TriggerSwitch  = "switch"  // UpdateDictionary 切换 source_table 后重建
	TriggerDaily   = "daily"   // worker daily cron
)

// SyncResultLabel 枚举(label result 取值)。
const (
	ResultOK      = "ok"
	ResultFailed  = "failed"
	ResultTimeout = "timeout"
)
