// Package topology — T-0027 D8 rule engine 可观测性埋点。
//
// 6 metric (PRD §12.5)：
//   - omc_topology_rule_evaluations_total{result, source}
//     result ∈ matched | skipped | failed   source ∈ manual | cron | inform
//   - omc_topology_rule_evaluation_duration_seconds{rule_id}  Histogram
//   - omc_topology_rule_apply_failures_total{reason}
//     reason ∈ db_error | target_group_missing | other
//   - omc_topology_devices_in_rule_groups            Gauge — 留 Step 11 优化
//     （需独立 SQL count + 周期更新；Day 8 先暴露字段，由 cron tick 抽样设置）
//   - omc_topology_active_rules                      Gauge — 同上抽样设置
//   - omc_topology_default_group_migrations_total{result}
//     result ∈ migrated | skipped_manual | failed
//
// 与 internal/backup/policy_metrics.go 同模式：所有方法 nil-safe，让生产
// 装配（DI 传 registry）和测试（不传）共享同一调用面。
package topology

import "github.com/prometheus/client_golang/prometheus"

// RuleMetrics 拓扑规则引擎 Prometheus 收集器。
type RuleMetrics struct {
	evaluationsTotal       *prometheus.CounterVec
	evaluationDuration     *prometheus.HistogramVec
	applyFailuresTotal     *prometheus.CounterVec
	devicesInRuleGroups    prometheus.Gauge
	activeRules            prometheus.Gauge
	defaultGroupMigrations *prometheus.CounterVec
}

// NewRuleMetrics 构造 + 可选注册到 registry。reg=nil 不注册（test 友好）。
func NewRuleMetrics(reg prometheus.Registerer) *RuleMetrics {
	m := &RuleMetrics{
		evaluationsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_topology_rule_evaluations_total",
			Help: "Topology rule evaluation outcomes by result × source (T-0027). " +
				"result ∈ matched|skipped|failed; source ∈ manual|cron|inform.",
		}, []string{"result", "source"}),
		evaluationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "omc_topology_rule_evaluation_duration_seconds",
			Help:    "Single-rule evaluation latency (T-0027), labeled by rule_id.",
			Buckets: prometheus.DefBuckets,
		}, []string{"rule_id"}),
		applyFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_topology_rule_apply_failures_total",
			Help: "Rule apply task failure tally by reason (T-0027). " +
				"reason ∈ db_error|target_group_missing|other.",
		}, []string{"reason"}),
		devicesInRuleGroups: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_topology_devices_in_rule_groups",
			Help: "Current device count whose group_member.source_type='rule' (T-0027 D5.B).",
		}),
		activeRules: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_topology_active_rules",
			Help: "Current count of enabled DeviceRule rows (T-0027).",
		}),
		defaultGroupMigrations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_topology_default_group_migrations_total",
			Help: "Devices migrated from default group by rule cron (T-0027 D5.B). " +
				"result ∈ migrated|skipped_manual|failed.",
		}, []string{"result"}),
	}
	if reg != nil {
		reg.MustRegister(
			m.evaluationsTotal, m.evaluationDuration, m.applyFailuresTotal,
			m.devicesInRuleGroups, m.activeRules, m.defaultGroupMigrations,
		)
	}
	return m
}

// RecordEvaluation 记录一次规则评估结果。
// result ∈ {"matched","skipped","failed"}；source ∈ {"manual","cron","inform"}.
func (m *RuleMetrics) RecordEvaluation(result, source string) {
	if m == nil {
		return
	}
	m.evaluationsTotal.WithLabelValues(result, source).Inc()
}

// ObserveEvaluationDuration 记录单规则评估耗时（秒）。
func (m *RuleMetrics) ObserveEvaluationDuration(ruleID string, seconds float64) {
	if m == nil {
		return
	}
	m.evaluationDuration.WithLabelValues(ruleID).Observe(seconds)
}

// RecordApplyFailure 记录规则 apply 任务失败。
// reason ∈ {"db_error","target_group_missing","other"}.
func (m *RuleMetrics) RecordApplyFailure(reason string) {
	if m == nil {
		return
	}
	m.applyFailuresTotal.WithLabelValues(reason).Inc()
}

// SetDevicesInRuleGroups 周期更新 source_type='rule' 的设备数（cron 抽样）。
func (m *RuleMetrics) SetDevicesInRuleGroups(n float64) {
	if m == nil {
		return
	}
	m.devicesInRuleGroups.Set(n)
}

// SetActiveRules 周期更新启用规则数（cron tick 抽样）。
func (m *RuleMetrics) SetActiveRules(n float64) {
	if m == nil {
		return
	}
	m.activeRules.Set(n)
}

// RecordDefaultGroupMigration 记录从 default 组迁出事件（D5.B 专用）。
// result ∈ {"migrated","skipped_manual","failed"}.
func (m *RuleMetrics) RecordDefaultGroupMigration(result string) {
	if m == nil {
		return
	}
	m.defaultGroupMigrations.WithLabelValues(result).Inc()
}
