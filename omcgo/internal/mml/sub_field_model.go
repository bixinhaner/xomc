package mml

import (
	"time"

	"github.com/google/uuid"
)

// MMLCommandSubField represents a single sub-field of an MML command.
//
// 表来源：migration 000095 mml_command_sub_fields（T-0123-P0）
// 替代关系：替代 000090 DROP 的 mml_command_params_rel junction 表；
// 新增字段：(mml_code, label_i18n, default_selected, is_required, sort_order)
//
// migration 000113：列 param_id（FK mml_params）改名为 standard_path_id（FK
// standard_params 系统级标准 path 字典）。结构体字段名保留 ParamID 做 soft
// alias（语义已从 mml_params.id 切换为 standard_params.id），repository 层在
// INSERT 列名用 standard_path_id，SELECT 用 `standard_path_id AS param_id` 别名。
//
// 设计要点：
//   - 一条 sub-field 行 = (command, standard_path) 组合 + 命令上下文的 mml_code / label 覆盖
//   - MMLCode 命令内唯一（UNIQUE (command_id, mml_code)）
//   - ParamID 命令内唯一（UNIQUE (command_id, standard_path_id)，migration 000113 替代 (command_id, param_id)）
//   - LabelI18n 覆盖全局默认 label（例：同一 path 在不同命令显示名可不同）
//   - DefaultSelected 控制 LST 命令的勾选 UI 默认值
//   - IsRequired 控制 MOD/ADD 命令的输入框必填红 *
//   - 写入 sub_fields 后由触发器自动重算 mml_commands.target_paths
type MMLCommandSubField struct {
	ID        uuid.UUID `json:"id"        db:"id"`
	CommandID uuid.UUID `json:"command_id" db:"command_id"`
	// ParamID 字段名为 soft alias，自 migration 000113 起 DB 列名为
	// standard_path_id（FK standard_params.id）。保留字段名避免上层调用面爆改。
	ParamID uuid.UUID `json:"param_id"   db:"param_id"`

	// 老系统 MML 字符串内部使用的 code（命令上下文相关）
	// 例：path=Device.DeviceInfo.X_COM_MODULE_TYPE 在 DEVICE_INFO 命令叫 LTE_GSM_MODEL_NAME
	// 渲染到 `LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,...}` MML 字符串
	MMLCode string `json:"mml_code" db:"mml_code"`

	// 命令上下文的 sub-field 显示标签（覆盖 mml_params.name_i18n 全局默认）
	// 例：{"en-US":"Model Name","zh-CN":"型号"}
	LabelI18n map[string]string `json:"label_i18n" db:"label_i18n"`

	// LST 命令：UI 是否默认勾选（老系统：14 sub-field 全 true）
	DefaultSelected bool `json:"default_selected" db:"default_selected"`

	// MOD/ADD 命令：sub-field 是否必填（前端校验红 *）
	IsRequired bool `json:"is_required" db:"is_required"`

	// 显示与渲染顺序（命令面板 UI + MML 字符串 lstId={} 内顺序）
	SortOrder int `json:"sort_order" db:"sort_order"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// SubFieldFilter 列查询子字段时的过滤器。
type SubFieldFilter struct {
	CommandID *uuid.UUID
	ParamID   *uuid.UUID
}

// MMLCommandSubFieldEnriched 在 SubField 基础上 join mml_params 取通用元数据，
// 供 Console GET /mml/commands/:id/sub-fields 端点直接 marshal 给前端使用。
//
// 字段优先级：LabelI18n（命令上下文覆盖） > ParamNameI18n（全局默认）。
type MMLCommandSubFieldEnriched struct {
	MMLCommandSubField

	// 从 join mml_params 拿到的通用元数据（前端 SubFieldChecklist / SubFieldInputList 渲染依据）
	Tr069Path          string               `json:"tr069_path"`
	ValueType          string               `json:"value_type"`
	AccessType         string               `json:"access_type"`
	IsObject           bool                 `json:"is_object"`
	SupportsAdd        bool                 `json:"supports_add"`
	SupportsDelete     bool                 `json:"supports_delete"`
	ChangeApplies      string               `json:"change_applies"`
	ConstraintTextI18n map[string]string    `json:"constraint_text_i18n"`
	DefaultValue       *string              `json:"default_value,omitempty"`
	JsRegex            *string              `json:"js_regex,omitempty"`
	ValidationPattern  *string              `json:"validation_pattern,omitempty"`
	EnumOptions        []MMLParamEnumOption `json:"enum_options,omitempty"`
	// MinValue/MaxValue 优先来自当前 paramModel 的 param_mappings，缺失时回退
	// standard_params。MML 控制台「命令参数」在 MOD/ADD 填值时用作范围校验和兼容默认值。
	MinValue *int64 `json:"min_value,omitempty"`
	MaxValue *int64 `json:"max_value,omitempty"`
	// ParamNameI18n 是 mml_params.name_i18n（全局默认 label，LabelI18n 为空时兜底）
	ParamNameI18n map[string]string `json:"param_name_i18n,omitempty"`
	// Description 是 standard_params.description（TR-181 path 中文含义说明，
	// 由 cmcc_tdlte_v23.json 等 spec seed 回填；MML 控制台 path 行 tooltip / 行内提示用）。
	// 可为空（非 cmcc-td-lte 来源的 standard_params 行 description 未维护）。
	Description string `json:"description,omitempty"`

	// IsSupported 是 admin 视角的"支持状态"汇总位。
	//
	// 2026-05-27 用户决策：真值源从 mml_command_sub_fields.is_supported 改为
	// param_mappings.is_supported —— 后者按 (param_model, standard_path) 维度
	// 记录"该 paramModel 是否支持此 path"，是 T-0176-PR-A 之后 catalog 唯一权威。
	//
	// 聚合规则（admin 端不绑特定 paramModel，呈现"跨 model 总体支持状况"）：
	//   - 若该 path 未在任何 active param_mapping 出现 → 视为 "支持"（默认 true）
	//   - 若该 path 在至少一个 active param_mapping 中是 supported=true → true
	//   - 否则（全 active 映射都标 false） → false
	// 即 BOOL_OR(pm.is_supported) FILTER (WHERE pm.is_active)，无行兜底 true。
	IsSupported bool `json:"is_supported"`

	// SupportedModelCount / TotalModelCount 给前端做"X / Y 个 paramModel 支持"的
	// 细粒度展示用。仅 admin ListAdminByCommand 填充；console 路径不写。
	SupportedModelCount int `json:"supported_model_count"`
	TotalModelCount     int `json:"total_model_count"`
}

// MMLParamEnumOption 是参数模型枚举值及其展示标签。
type MMLParamEnumOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
