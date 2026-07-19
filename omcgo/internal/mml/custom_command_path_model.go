package mml

import (
	"time"

	"github.com/google/uuid"
)

// custom_command_path_model.go — issue #115 调整3（A1）
//
// 自定义命令(mml_custom_command) ↔ 标准路径(standard_params) 的精瘦关联模型。
// 与内置命令的 mml_command_sub_fields 不同：本表不复制 path 元数据（类型/access/
// 描述/是否支持），那些读时 JOIN standard_params / param_mappings 取。

// MMLCustomCommandPath 是关联表 mml_custom_command_paths 的一行。
type MMLCustomCommandPath struct {
	ID              uuid.UUID `json:"id"`
	CommandID       uuid.UUID `json:"command_id"`
	StandardPathID  uuid.UUID `json:"standard_path_id"`
	DefaultSelected bool      `json:"default_selected"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MMLCustomCommandPathView 是 List 返回的富化视图（JOIN standard_params）。
// 是否支持(is_supported，按 param_mappings 聚合) 留待切片 A3/A4 接入执行支持判定时补，
// A1 仅返字典静态元数据 + 关联属性。
type MMLCustomCommandPathView struct {
	ID              uuid.UUID `json:"id"`
	CommandID       uuid.UUID `json:"command_id"`
	StandardPathID  uuid.UUID `json:"standard_path_id"`
	StandardPath    string    `json:"standard_path"`
	EntryType       string    `json:"entry_type"`
	Access          string    `json:"access"`
	DataType        string    `json:"data_type"`
	Description     string    `json:"description"`
	MinValue        *int64    `json:"min_value,omitempty"`
	MaxValue        *int64    `json:"max_value,omitempty"`
	DefaultSelected bool      `json:"default_selected"`
	SortOrder       int       `json:"sort_order"`
	// Mutable=false 表示仅从历史 param_paths JSON 补出的兼容行，没有真实关联 ID。
	Mutable bool `json:"mutable"`
}
