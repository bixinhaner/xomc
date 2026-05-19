package topology

import (
	"time"

	"github.com/google/uuid"
)

// DeviceRule 设备归属规则
type DeviceRule struct {
	ID               uuid.UUID    `json:"id"`
	Name             string       `json:"name"`
	Priority         int          `json:"priority"`
	TargetGroupID    *uuid.UUID   `json:"target_group_id,omitempty"`
	TargetGroupName  string       `json:"target_group_name,omitempty"`
	Enabled          bool         `json:"enabled"`
	MatchingMode     MatchingMode `json:"matching_mode,omitempty"`
	NameRuleList     []NameRule   `json:"name_rule_list,omitempty"`
	LACList          []int        `json:"lac_list,omitempty"`
	TACList          []int        `json:"tac_list,omitempty"`
	SerialNumberList []string     `json:"serial_number_list,omitempty"` // serialNumber 模式（migration 000124）
	Description      string       `json:"description,omitempty"`
	Operators        string       `json:"operators"` // 生成的规则描述
	CreatedBy        string       `json:"created_by,omitempty"`
	UpdatedBy        string       `json:"updated_by,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name             string     `json:"name" binding:"required"`
	Priority         int        `json:"priority"`
	TargetGroupID    string     `json:"target_group_id" binding:"required"`
	Enabled          bool       `json:"enabled"`
	MatchingMode     string     `json:"matching_mode" binding:"required"`
	NameRuleList     []NameRule `json:"name_rule_list"`
	LACList          []int      `json:"lac_list"`
	TACList          []int      `json:"tac_list"`
	SerialNumberList []string   `json:"serial_number_list"`
	Description      string     `json:"description"`
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name             *string    `json:"name"`
	Priority         *int       `json:"priority"`
	TargetGroupID    *string    `json:"target_group_id"`
	Enabled          *bool      `json:"enabled"`
	MatchingMode     *string    `json:"matching_mode"`
	NameRuleList     []NameRule `json:"name_rule_list"`
	LACList          []int      `json:"lac_list"`
	TACList          []int      `json:"tac_list"`
	SerialNumberList []string   `json:"serial_number_list"`
	Description      *string    `json:"description"`
}

// RuleListRequest 规则列表查询请求
type RuleListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Enabled  *bool  `form:"enabled"`
	Name     string `form:"name"`
}

// RuleListResponse 规则列表响应
type RuleListResponse struct {
	Items []DeviceRule `json:"items"`
	Total int64        `json:"total"`
}

// BatchSortRuleRequest 批量调整优先级请求
type BatchSortRuleRequest struct {
	Items []RuleSortItem `json:"items" binding:"required"`
}

// RuleSortItem 规则排序项
type RuleSortItem struct {
	ID       string `json:"id" binding:"required"`
	Priority int    `json:"priority"`
}

// RuleTask 规则应用任务
type RuleTask struct {
	ID            uuid.UUID  `json:"id"`
	RuleID        uuid.UUID  `json:"rule_id"`
	RuleName      string     `json:"rule_name,omitempty"`
	Status        string     `json:"status"` // pending, running, completed, failed
	TotalDevices  int        `json:"total_devices"`
	MatchedCount  int        `json:"matched_count"`
	FailedCount   int        `json:"failed_count"`
	Progress      int        `json:"progress"` // 百分比进度
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	CreatedBy     string     `json:"created_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ApplyRuleRequest 应用规则请求
type ApplyRuleRequest struct {
	SourceGroupIDs []string `json:"source_group_ids"` // 可选，指定从哪些分组中匹配设备
}

// RuleTaskListResponse 任务列表响应
type RuleTaskListResponse struct {
	Items []RuleTask `json:"items"`
	Total int64      `json:"total"`
}
