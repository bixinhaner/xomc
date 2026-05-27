// Package devsweep 实现 T-0179 — MML param_mappings 单设备 sweep 工具。
//
// 操作员针对指定设备（按 SN）触发 GPV 探测，把 paramModel 的
// param_mappings 里设备不支持的 path 标 is_supported=false，作为
// T-0176/T-0177 之后的"主动探测"工具（PR-E auto-learn 是被动 SPV 9005 触发，
// 互补）。
//
// 设计要点：
//   - 写真值源：param_mappings（paramModel 级，全 paramModel 设备受影响）
//   - 覆盖范围：is_active=true 全集（不论当前 is_supported），可选 prefix 过滤
//   - 安全门：unsupported>50% abort（--force 跳过）/ paramModel 关联设备>10
//     必须 --confirm-paramodel-wide
//   - {i} 占位符：GPV 发送前替换为 ".0."（CPE 协议要求实例号；不做实例发现）
//   - dry-run 默认；--apply 才写 DB
//   - 不写 migration / audit 表 / discovered_param_mappings（用户决策）
package devsweep

import (
	"time"

	"github.com/google/uuid"
)

// SafetyDefaults 与 PRD/spec 对齐的安全门默认值，单元测试和生产用同一组。
//
// MaxUnsupportedFraction：unsupported_count / candidate_count 超过此值
// 视为高风险 abort（用户需 --force 才能继续）。
// MaxParamModelDevices：paramModel 关联活跃设备数超过此值视为 wide-impact，
// 用户需 --confirm-paramodel-wide 才能继续。
//
// 故意做成常量而非 viper config — 这是 CLI 工具的"语义边界"，
// 不应在生产环境通过配置文件偷偷放宽。需要覆盖时显式传 SafetyConfig。
const (
	SafetyDefaultMaxUnsupportedFraction = 0.50
	SafetyDefaultMaxParamModelDevices   = 10
)

// SafetyConfig 控制 Service.Run 的安全门阈值。
// 零值不可用 — 调用方应通过 DefaultSafetyConfig() 初始化后再按需调整。
type SafetyConfig struct {
	MaxUnsupportedFraction float64
	MaxParamModelDevices   int
}

// DefaultSafetyConfig 返回 PRD 默认阈值（0.50 / 10）。
func DefaultSafetyConfig() SafetyConfig {
	return SafetyConfig{
		MaxUnsupportedFraction: SafetyDefaultMaxUnsupportedFraction,
		MaxParamModelDevices:   SafetyDefaultMaxParamModelDevices,
	}
}

// Options 是 Service.Run 的运行参数集合。零值除 DeviceSN/Operator 外可用。
type Options struct {
	DeviceSN string // 必填
	Prefix   string // 可选 path 前缀过滤（如 "Device.FaultMgmt."）

	BatchSize  int           // GPV 每批 path 数；默认 1（见 prober.go 说明）
	RPCTimeout time.Duration // 单 task 等待 terminal 状态的超时；默认 30s
	RPCRate    float64       // 全局每秒最多入队几个 GPV task；默认 5/s

	Apply                bool // false=dry-run（默认）；true=写 DB
	Force                bool // 绕过 MaxUnsupportedFraction
	ConfirmParamModelWide bool // 显式确认 paramModel 影响 >MaxParamModelDevices

	Operator string // 审计字段，写入日志 + JSON 输出
	Safety   SafetyConfig
}

// ProbeOutcome 标记单条 path 的探测结论。
type ProbeOutcome string

const (
	// OutcomeSupported — GPV 成功；该 path 在设备上存在。
	OutcomeSupported ProbeOutcome = "supported"
	// OutcomeUnsupported — GPV 失败且 fault code == 9005（Invalid parameter name）。
	OutcomeUnsupported ProbeOutcome = "unsupported"
	// OutcomeUnknown — GPV 失败但 code != 9005 / timeout / 解析失败。
	// 不标记 is_supported；下次再跑可能拿到明确结论。
	OutcomeUnknown ProbeOutcome = "unknown"
)

// ProbeRecord 是单条 path 的探测明细，--verbose / --json per_path 输出用。
type ProbeRecord struct {
	StandardPath string        `json:"standard_path"`
	ProbePath    string        `json:"probe_path"`     // 实际发给 CPE 的 path（{i} → .0.）
	Outcome      ProbeOutcome  `json:"outcome"`
	FaultCode    int           `json:"fault_code,omitempty"`
	FaultMessage string        `json:"fault_message,omitempty"`
	Batch        int           `json:"batch"`
	DurationMS   int64         `json:"duration_ms"`
	TaskID       string        `json:"task_id,omitempty"`
}

// Result 是 Service.Run 的返回，承载文本/JSON 渲染所需全部字段。
type Result struct {
	DeviceSN        string    `json:"device_sn"`
	ProductClass    string    `json:"product_class"`
	ProductID       uuid.UUID `json:"product_id"`
	ProductName     string    `json:"product_name"`
	ParamModelID    uuid.UUID `json:"param_model_id"`
	ParamModelName  string    `json:"param_model_name"`
	FirmwareVersion string    `json:"firmware_version"`
	IsOnline        bool      `json:"is_online"`

	CandidateCount   int `json:"candidate_count"`
	SupportedCount   int `json:"supported_count"`
	UnsupportedCount int `json:"unsupported_count"`
	UnknownCount     int `json:"unknown_count"`
	MarkedCount      int `json:"marked_count"` // 实际 UPDATE 影响行数；dry-run=0

	UnsupportedPaths []string `json:"unsupported_paths"`
	UnknownPaths     []string `json:"unknown_paths,omitempty"`
	PerPath          []ProbeRecord `json:"per_path,omitempty"` // 仅 verbose

	DryRun     bool      `json:"dry_run"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	DurationMS int64     `json:"duration_ms"`

	// ParamModelDevicesAffected 是该 paramModel 关联的活跃设备数。
	// 用于操作员理解 sweep 影响面（写 param_mappings 会影响所有关联设备）。
	ParamModelDevicesAffected int `json:"param_model_devices_affected"`

	// Aborted 在安全门阻止时为 true，对应 ErrorCode 给出原因。
	Aborted   bool   `json:"aborted"`
	ErrorCode string `json:"error_code,omitempty"`
}
