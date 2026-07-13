package topology

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// MatchingMode 匹配模式
type MatchingMode string

const (
	MatchingModeDeviceName   MatchingMode = "deviceName"   // 设备名称匹配
	MatchingModeLAC          MatchingMode = "lac"          // LAC 位置区码匹配
	MatchingModeTAC          MatchingMode = "tac"          // TAC 跟踪区码匹配
	MatchingModeSerialNumber MatchingMode = "serialNumber" // 序列号精确成员匹配（migration 000124）
)

// NameRule 设备名称匹配规则
type NameRule struct {
	Condition string `json:"condition"` // contain(包含), startWith(开头是), endWith(结尾是), equal(等于)
	Value     string `json:"value"`     // 匹配值
	AndOr     string `json:"andOr"`     // and, or (第一条不需要此字段)
}

// UnmarshalJSON 自定义 JSON 解析，支持新旧字段名的兼容
// 旧字段名: type, operator (前端之前使用)
// 新字段名: condition, andOr (后端定义)
func (nr *NameRule) UnmarshalJSON(data []byte) error {
	// 临时结构体，包含新旧两种字段名
	type Alias NameRule
	aux := &struct {
		// 新字段名
		Condition string `json:"condition"`
		AndOr     string `json:"andOr"`
		// 旧字段名（兼容旧数据）
		Type     string `json:"type"`
		Operator string `json:"operator"`
		// 共用字段
		Value string `json:"value"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 优先使用新字段名，如果为空则使用旧字段名
	nr.Condition = aux.Condition
	if nr.Condition == "" {
		nr.Condition = aux.Type
	}

	nr.AndOr = aux.AndOr
	if nr.AndOr == "" {
		nr.AndOr = aux.Operator
	}

	nr.Value = aux.Value

	return nil
}

// DeviceGroup represents a hierarchical group for organizing devices.
type DeviceGroup struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	// i18n JSONB 列 (migration 000003)。形态 {"zh-CN": "...", "en-US": "..."},
	// 前端按 appLocale 取;不存在时 fallback 到单语言列 Name/Description/Remark。
	// 目前 GetTreeWithCounts 路径填充;其它路径暂不填,前端兜底中文。
	NameI18n        map[string]string `json:"name_i18n,omitempty"`
	DescriptionI18n map[string]string `json:"description_i18n,omitempty"`
	RemarkI18n      map[string]string `json:"remark_i18n,omitempty"`
	ParentID        *uuid.UUID        `json:"parent_id,omitempty"`
	Carrier         model.CarrierCode `json:"carrier,omitempty"`
	Description     string            `json:"description,omitempty"`
	SortOrder       int               `json:"sort_order"`
	Level           int               `json:"level"`
	Status          string            `json:"status"`
	IsDefault       bool              `json:"is_default"`
	Remark          string            `json:"remark,omitempty"`
	CreatedBy       string            `json:"created_by,omitempty"`
	UpdatedBy       string            `json:"updated_by,omitempty"`
	DeviceCount     int               `json:"device_count"`
	Children        []DeviceGroup     `json:"children,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	// 匹配规则（仅 L2 分组使用）
	SourceGroupID    *uuid.UUID   `json:"source_group_id,omitempty"`
	MatchingMode     MatchingMode `json:"matching_mode,omitempty"`
	NameRuleList     []NameRule   `json:"name_rule_list,omitempty"`
	LACList          []int        `json:"lac_list,omitempty"`
	TACList          []int        `json:"tac_list,omitempty"`
	SerialNumberList []string     `json:"serial_number_list,omitempty"` // serialNumber 模式专用（migration 000124）
}

// DeviceGroupMember represents a device's membership in a group.
type DeviceGroupMember struct {
	GroupID  uuid.UUID `json:"group_id"`
	DeviceID uuid.UUID `json:"device_id"`
	AddedAt  time.Time `json:"added_at"`
}

// --- Request/Response DTOs ---

// CreateGroupRequest is the payload for creating a device group.
type CreateGroupRequest struct {
	Name string `json:"name" binding:"required"`
	// i18n 三件套(可选)。后端 INSERT 时写入 {name,description,remark}_i18n JSONB 列。
	// 形态 {"zh-CN":"...","en-US":"..."}。空 map 落 '{}' 表示无 i18n,前端 fallback 单语言列。
	NameI18n        map[string]string `json:"name_i18n,omitempty"`
	DescriptionI18n map[string]string `json:"description_i18n,omitempty"`
	RemarkI18n      map[string]string `json:"remark_i18n,omitempty"`
	ParentID        string            `json:"parent_id"`
	Carrier         string            `json:"carrier"`
	Remark          string            `json:"remark"`
	SortOrder       int               `json:"sort_order"`
	SubGroups       []SubGroupInput   `json:"sub_groups,omitempty"`
	// 匹配规则（仅 L2 分组使用）
	SourceGroupID    string     `json:"source_group_id"`
	MatchingMode     string     `json:"matching_mode"`  // deviceName, lac, tac
	NameRuleList     []NameRule `json:"name_rule_list"` // 设备名称匹配规则
	LACList          []int      `json:"lac_list"`       // LAC 列表
	TACList          []int      `json:"tac_list"`       // TAC 列表
	SerialNumberList []string   `json:"serial_number_list"`
}

// SubGroupInput defines a sub-group to create in batch.
type SubGroupInput struct {
	Name      string   `json:"name" binding:"required"`
	Remark    string   `json:"remark"`
	DeviceIDs []string `json:"device_ids,omitempty"`
}

// UpdateGroupRequest is the payload for updating a device group.
type UpdateGroupRequest struct {
	Name *string `json:"name"`
	// i18n 三件套(可选)。nil = 不修改;非空 map = 覆盖整张 JSONB。
	NameI18n        map[string]string `json:"name_i18n,omitempty"`
	DescriptionI18n map[string]string `json:"description_i18n,omitempty"`
	RemarkI18n      map[string]string `json:"remark_i18n,omitempty"`
	ParentID        *string           `json:"parent_id"` // 修改父级分组（L1 转 L2 或 L2 转 L1）
	Remark          *string           `json:"remark"`
	SortOrder       *int              `json:"sort_order"`
	// 匹配规则（仅 L2 分组使用）
	SourceGroupID    *string    `json:"source_group_id"`
	MatchingMode     *string    `json:"matching_mode"`
	NameRuleList     []NameRule `json:"name_rule_list"`
	LACList          []int      `json:"lac_list"`
	TACList          []int      `json:"tac_list"`
	SerialNumberList []string   `json:"serial_number_list"`
}

// CheckDeleteResponse describes the impact of deleting a group.
type CheckDeleteResponse struct {
	CanDelete   bool   `json:"can_delete"`
	HasDevices  bool   `json:"has_devices"`
	DeviceCount int    `json:"device_count"`
	ChildCount  int    `json:"child_count"`
	Message     string `json:"message"`
}

// GroupStats provides aggregate statistics about device groups.
type GroupStats struct {
	TotalGroups      int `json:"total_groups"`
	GroupedDevices   int `json:"grouped_devices"`
	UngroupedDevices int `json:"ungrouped_devices"`
}

// MoveDevicesRequest is the payload for moving devices between groups.
type MoveDevicesRequest struct {
	DeviceIDs     []string `json:"device_ids" binding:"required"`
	TargetGroupID string   `json:"target_group_id" binding:"required"`
}

// BatchSortRequest is the payload for batch-updating sort orders.
type BatchSortRequest struct {
	Items []SortItem `json:"items" binding:"required"`
}

// SortItem specifies a sort order for a group.
type SortItem struct {
	ID        string `json:"id" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

// BatchDevicesRequest is the payload for batch adding/removing devices.
type BatchDevicesRequest struct {
	DeviceIDs []string `json:"device_ids" binding:"required"`
}

// TreeResponse is the response for the device group tree endpoint.
type TreeResponse struct {
	Items []DeviceGroup `json:"items"`
	Stats *GroupStats   `json:"stats"`
}
