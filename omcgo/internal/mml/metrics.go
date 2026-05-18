package mml

import "github.com/prometheus/client_golang/prometheus"

// FanoutMetrics 聚合 MML fanout 阶段的 Prometheus 指标。
//
// 当前唯一字段是 PathTranslationMissTotal — Stage 1 (T-0123 v5) 整改方案落地的
// 路径翻译未命中计数器。Fanouter 在 standardPath → privatePath 翻译任一步骤
// 失败时累计本计数器，便于 Prometheus alert（详见
// `deployments/monitoring/alerts/omc-rules.yml MMLPathTranslationMissSustained`）。
//
// 设计参考 `internal/product/metrics.go` 的 nil-safe 模式：传 nil registerer 时
// 自动建匿名 registry，单元测试与 main 启动均可用。
type FanoutMetrics struct {
	PathTranslationMissTotal *prometheus.CounterVec
}

// PathTranslationMissReason 是 Translator 兜底原因的枚举常量，与 metrics
// label 一一对应；新增 reason 时同步 alert 注释。
const (
	PathMissReasonDeviceLookup      = "device_lookup"        // DeviceLookup.GetBySerialNumber 失败
	PathMissReasonProductUnresolved = "product_unresolved"   // productClass 未匹配或无 ParamModelID
	PathMissReasonTranslatorUnavail = "translator_unavail"   // Translator 构造失败（非 ErrNoMapping）
	PathMissReasonPathUnmapped      = "path_unmapped"        // 单条 path ToPrivate 未命中
)

// NewFanoutMetrics 注册并返回 fanout 用的指标集合。
//
// 传入 nil 时返回一个内部已注册到匿名 Registry 的实例（零依赖、可丢弃），
// 测试场景与单元 main 启动均可用。
func NewFanoutMetrics(reg prometheus.Registerer) *FanoutMetrics {
	m := &FanoutMetrics{
		PathTranslationMissTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mml_path_translation_miss_total",
				Help: "Total MML standardPath→privatePath translation misses by reason (device_lookup/product_unresolved/translator_unavail/path_unmapped). Non-zero rate signals stale ParamModel / new firmware / missing discovered_param_mappings.",
			},
			[]string{"reason"},
		),
	}

	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	reg.MustRegister(m.PathTranslationMissTotal)
	return m
}

// missDeviceLookup / missProductUnresolved / missTranslatorUnavail / missPathUnmapped
// 是热路径捷径，避免 fanout.go 处处写字符串字面量并防笔误。
func (m *FanoutMetrics) missDeviceLookup()      { m.PathTranslationMissTotal.WithLabelValues(PathMissReasonDeviceLookup).Inc() }
func (m *FanoutMetrics) missProductUnresolved() { m.PathTranslationMissTotal.WithLabelValues(PathMissReasonProductUnresolved).Inc() }
func (m *FanoutMetrics) missTranslatorUnavail() { m.PathTranslationMissTotal.WithLabelValues(PathMissReasonTranslatorUnavail).Inc() }
func (m *FanoutMetrics) missPathUnmapped(n int) {
	if n <= 0 {
		return
	}
	m.PathTranslationMissTotal.WithLabelValues(PathMissReasonPathUnmapped).Add(float64(n))
}
