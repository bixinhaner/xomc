package provision

import "github.com/prometheus/client_golang/prometheus"

// Metrics 聚合 provisioning 引擎的 Prometheus 指标。
//
// 字段：
//   - DiscoveryStatusUpdateErrorsTotal: HIGH-27 — model_upload.go 中 discovery_log
//     状态写库失败的累计（按 operation 分桶）。此前这些 UpdateStatus 调用返回值被
//     `_ =` 吞掉，写库失败（网络分割/磁盘满）无法观测，discovery_log 状态与实际
//     处理结果不同步。非零率说明 discovery_log 落库异常，前端参数同步页面可能误判
//     设备状态，运维应排查 PG 可用性。
//   - RedisThrottleFailuresTotal: MEDIUM-19 — engine.go device.online / firmware.changed
//     节流用的 Redis SetNX 失败累计（按 operation 分桶）。fail-open 语义不变（失败后
//     继续执行），但此计数让运维可见"节流保护已失效，重复同步可能发生"，Redis 故障
//     持续 >5min 时应告警（重复 device.online 会放大 GPV 流量 3-5x）。
//
// 设计参考 `internal/mml/metrics.go` 的 nil-safe 模式：传 nil registerer 时
// 自动建匿名 registry，单元测试与 main 启动均可用。
type Metrics struct {
	DiscoveryStatusUpdateErrorsTotal *prometheus.CounterVec
	RedisThrottleFailuresTotal       *prometheus.CounterVec
}

// NewMetrics 注册并返回 provisioning 用的指标集合。
//
// 传入 nil 时返回一个内部已注册到匿名 Registry 的实例（零依赖、可丢弃），
// 测试场景与单元 main 启动均可用。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		DiscoveryStatusUpdateErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "discovery_status_update_errors_total",
				Help: "Total failures writing parameter discovery_log status (model_upload.go UpdateStatus) by operation. Non-zero rate signals PG write failures leaving discovery_log out of sync with real processing — front-end param-sync page may misjudge device state.",
			},
			[]string{"operation"},
		),
		RedisThrottleFailuresTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "provision_redis_throttle_failures_total",
				Help: "Total Redis SetNX failures on provisioning throttle paths (device.online / firmware.changed) by operation. Fail-open: handling continues, but throttling is disabled — sustained failures (>5min) risk 3-5x duplicate GPV traffic. Alert ops on Redis availability.",
			},
			[]string{"operation"},
		),
	}

	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	reg.MustRegister(m.DiscoveryStatusUpdateErrorsTotal)
	reg.MustRegister(m.RedisThrottleFailuresTotal)
	return m
}

// discoveryStatusUpdateErr 累加 discovery_log 状态写库失败计数（HIGH-27）。
// operation 标识失败的业务点（如 skipped_fileupload / minio_get_failed）。
func (m *Metrics) discoveryStatusUpdateErr(operation string) {
	if m == nil {
		return
	}
	m.DiscoveryStatusUpdateErrorsTotal.WithLabelValues(operation).Inc()
}

// redisThrottleFailure 累加 Redis 节流 SetNX 失败计数（MEDIUM-19）。
func (m *Metrics) redisThrottleFailure(operation string) {
	if m == nil {
		return
	}
	m.RedisThrottleFailuresTotal.WithLabelValues(operation).Inc()
}
