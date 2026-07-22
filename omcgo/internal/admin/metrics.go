package admin

import "github.com/prometheus/client_golang/prometheus"

// AdminMetrics 汇集 admin 模块对外暴露的 Prometheus 指标。
// 当前仅含 PRD docs/prd/system/users.md §10 DoD 要求的失效失败计数；
// 未来可扩展用户/登录/操作类指标（参见 PRD §9）。
type AdminMetrics struct {
	// PermCacheInvalidateFailed PRD §10 DoD：用户写操作后 InvalidateUserCache 失败的累计次数。
	// 失败本身不阻塞 handler 返回，但需要可观测以便发现 Redis 异常。
	PermCacheInvalidateFailed prometheus.Counter
	// ConfigApplyAttempts 记录系统配置每次运行态应用尝试的结果。
	ConfigApplyAttempts *prometheus.CounterVec
}

// NewAdminMetrics 创建并注册 admin 指标到给定 Registerer。
func NewAdminMetrics(reg prometheus.Registerer) *AdminMetrics {
	m := &AdminMetrics{
		PermCacheInvalidateFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_perm_cache_invalidate_failed_total",
			Help: "Total times PermissionService.InvalidateUserCache failed after a user write op (does not block the response)",
		}),
		ConfigApplyAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_sys_config_apply_attempts_total",
			Help: "Total system configuration apply attempts by category, target and result.",
		}, []string{"category", "target", "result"}),
	}
	if reg != nil {
		reg.MustRegister(m.PermCacheInvalidateFailed, m.ConfigApplyAttempts)
	}
	return m
}

// ObserveConfigApply 把一次应用结果写入低基数指标。批次 ID 和错误详情由结构化日志承载。
func (m *AdminMetrics) ObserveConfigApply(observation ConfigApplyObservation) {
	if m == nil || m.ConfigApplyAttempts == nil {
		return
	}
	m.ConfigApplyAttempts.WithLabelValues(observation.Category, observation.Target, observation.Result).Inc()
}
