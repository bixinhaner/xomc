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
	Tr069Path          string            `json:"tr069_path"`
	ValueType          string            `json:"value_type"`
	AccessType         string            `json:"access_type"`
	IsObject           bool              `json:"is_object"`
	SupportsAdd        bool              `json:"supports_add"`
	SupportsDelete     bool              `json:"supports_delete"`
	ChangeApplies      string            `json:"change_applies"`
	ConstraintTextI18n map[string]string `json:"constraint_text_i18n"`
	DefaultValue       *string           `json:"default_value,omitempty"`
	JsRegex            *string           `json:"js_regex,omitempty"`
	// ParamNameI18n 是 mml_params.name_i18n（全局默认 label，LabelI18n 为空时兜底）
	ParamNameI18n map[string]string `json:"param_name_i18n,omitempty"`
	// Description 是 standard_params.description（TR-181 path 中文含义说明，
	// 由 cmcc_tdlte_v23.json 等 spec seed 回填；MML 控制台 path 行 tooltip / 行内提示用）。
	// 可为空（非 cmcc-td-lte 来源的 standard_params 行 description 未维护）。
	Description string `json:"description,omitempty"`

	// IsSupported 是 mml_command_sub_fields.is_supported 列原值。
	// console 端 ListEnrichedByCommand 已硬过滤 is_supported=true，不暴露此字段；
	// 但 admin 端 ListAdminByCommand 必须看见已被 auto-learn 关掉的 path，让维护人员
	// 评估"是否手工恢复" / "是否真要永久隐藏"。json tag 与列名对齐。
	IsSupported bool `json:"is_supported"`
}
