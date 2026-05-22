package mml

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ParamVersion represents a parameter library version (e.g. QB1.0, MLN1.0).
type ParamVersion struct {
	VersionCode     string    `json:"version_code" db:"version_code"`
	VersionName     string    `json:"version_name" db:"version_name"`
	Description     string    `json:"description" db:"description"`
	ReleaseDate     *time.Time `json:"release_date,omitempty" db:"release_date"`
	ProductModels   []string  `json:"product_models" db:"product_models"`
	SoftwareVersions []string  `json:"software_versions" db:"software_versions"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	IsDeprecated    bool      `json:"is_deprecated" db:"is_deprecated"`
	GroupCount      int       `json:"group_count" db:"group_count"`
	ParamCount      int       `json:"param_count" db:"param_count"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// CommandGroup represents a hierarchical group of parameters.
type CommandGroup struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	GroupCode             string     `json:"group_code" db:"group_code"`
	GroupNameZh           string     `json:"group_name_zh" db:"group_name_zh"`
	GroupNameEn           string     `json:"group_name_en" db:"group_name_en"`
	ParentID              *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	Level                 int        `json:"level" db:"level"`
	IsListable            bool       `json:"is_listable" db:"is_listable"`
	IsModifiable          bool       `json:"is_modifiable" db:"is_modifiable"`
	IsAddable             bool       `json:"is_addable" db:"is_addable"`
	IsRemovable           bool       `json:"is_removable" db:"is_removable"`
	AddObjectPath         *string    `json:"add_object_path,omitempty" db:"add_object_path"`
	DeleteObjectPath      *string    `json:"delete_object_path,omitempty" db:"delete_object_path"`
	ParamVersion          string     `json:"param_version" db:"param_version"`
	PlatformSupport       []string   `json:"platform_support" db:"platform_support"`
	MobileSupport         bool       `json:"mobile_support" db:"mobile_support"`
	BroadbandSupport      bool       `json:"broadband_support" db:"broadband_support"`
	CellNumber            int        `json:"cell_number" db:"cell_number"`
	CellIndexLocation     int        `json:"cell_index_location" db:"cell_index_location"`
	RequireSecondConfirm  bool       `json:"require_second_confirm" db:"require_second_confirm"`
	ConfirmMessageZh      *string    `json:"confirm_message_zh,omitempty" db:"confirm_message_zh"`
	ConfirmMessageEn      *string    `json:"confirm_message_en,omitempty" db:"confirm_message_en"`
	DisplayOrder          int        `json:"display_order" db:"display_order"`

	// T-0123-P0 catalog 管理元数据（migration 000095）
	Source            string `json:"source" db:"source"`                         // standard / admin
	CatalogProtected  bool   `json:"catalog_protected" db:"catalog_protected"`   // standard 行不可删除

	Children              []CommandGroup `json:"children,omitempty"`
}

// Param represents a single TR-069 parameter definition.
//
// 字段历史：
//   - 000022 初创（mml_param_library 整套）
//   - 000090 Sprint A 重构：DROP 13 列（is_listable/is_modifiable/is_addable/is_removable/
//     is_dynamic/platform_support/mobile_support/broadband_support/software_version/
//     memo/require_second_confirm/confirm_message_zh/confirm_message_en）；
//     Go struct 保留字段作 SELECT 占位，但 DB 已无对应列（pre-existing 偏差，T-0119 followup 处理）
//   - 000095 T-0123-P0：扩展 8 元数据列 + is_writable 转 GENERATED STORED 派生列
type Param struct {
	ID                   uuid.UUID       `json:"id" db:"id"`
	ParamCode            string          `json:"param_code" db:"param_code"`
	ParamNameZh          string          `json:"param_name_zh" db:"param_name_zh"`
	ParamNameEn          string          `json:"param_name_en" db:"param_name_en"`
	Tr069Path            string          `json:"tr069_path" db:"tr069_path"`
	ValueType            string          `json:"value_type" db:"value_type"`
	ValueConstraint      json.RawMessage `json:"value_constraint" db:"value_constraint"`
	DefaultValue         *string         `json:"default_value,omitempty" db:"default_value"`
	JsRegex              *string         `json:"js_regex,omitempty" db:"js_regex"`

	// IsWritable 自 000095 起为 GENERATED ALWAYS AS STORED 派生列；
	// 派生公式：access_type IN ('READ_WRITE','WRITE_ONLY')；
	// **不可直接 INSERT/UPDATE**（PG 自动拒绝），仅可读。
	IsWritable           bool            `json:"is_writable" db:"is_writable"`

	IsListable           bool            `json:"is_listable" db:"is_listable"`     // 000090 DROP — Go struct 占位
	IsModifiable         bool            `json:"is_modifiable" db:"is_modifiable"` // 同上
	IsAddable            bool            `json:"is_addable" db:"is_addable"`       // 同上
	IsRemovable          bool            `json:"is_removable" db:"is_removable"`   // 同上
	IsLeaf               bool            `json:"is_leaf" db:"is_leaf"`
	IsDynamic            bool            `json:"is_dynamic" db:"is_dynamic"`       // 000090 DROP — Go struct 占位
	DisplayOrder         int             `json:"display_order" db:"display_order"`
	ParamVersion         string          `json:"param_version" db:"param_version"`
	SoftwareVersion      *string         `json:"software_version,omitempty" db:"software_version"` // 000090 DROP
	PlatformSupport      []string        `json:"platform_support" db:"platform_support"`           // 000090 DROP
	MobileSupport        bool            `json:"mobile_support" db:"mobile_support"`               // 000090 DROP
	BroadbandSupport     bool            `json:"broadband_support" db:"broadband_support"`         // 000090 DROP
	Memo                 *string         `json:"memo,omitempty" db:"memo"`                         // 000090 DROP
	ExplanationZh        *string         `json:"explanation_zh,omitempty" db:"explanation_zh"`
	ExplanationEn        *string         `json:"explanation_en,omitempty" db:"explanation_en"`
	TitleZh              *string         `json:"title_zh,omitempty" db:"title_zh"`
	TitleEn              *string         `json:"title_en,omitempty" db:"title_en"`
	RequireSecondConfirm bool            `json:"require_second_confirm" db:"require_second_confirm"` // 000090 DROP
	ConfirmMessageZh     *string         `json:"confirm_message_zh,omitempty" db:"confirm_message_zh"` // 000090 DROP
	ConfirmMessageEn     *string         `json:"confirm_message_en,omitempty" db:"confirm_message_en"` // 000090 DROP

	// T-0123-P0 catalog 元数据（migration 000095 新增）
	AccessType         string            `json:"access_type" db:"access_type"`                 // READ_ONLY / READ_WRITE / WRITE_ONLY
	IsObject           bool              `json:"is_object" db:"is_object"`                     // TR-069 object 容器（path 以 . 结尾）
	SupportsAdd        bool              `json:"supports_add" db:"supports_add"`               // 仅 is_object=true 有效
	SupportsDelete     bool              `json:"supports_delete" db:"supports_delete"`         // 仅 is_object=true 有效
	ChangeApplies      string            `json:"change_applies" db:"change_applies"`           // Immediate / OnReboot
	ConstraintTextI18n map[string]string `json:"constraint_text_i18n" db:"constraint_text_i18n"` // {"en-US":"...","zh-CN":"..."}
	CatalogProtected   bool              `json:"catalog_protected" db:"catalog_protected"`     // standard 行不可删除/锁定关键字段
	Source             string            `json:"source" db:"source"`                           // standard / admin
}

// ParamFilter specifies criteria for searching parameters.
type ParamFilter struct {
	VersionCode string
	GroupID     *uuid.UUID
	Search      *string
	Tr069Path   *string
}
