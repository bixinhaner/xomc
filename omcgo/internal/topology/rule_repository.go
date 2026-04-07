package topology

import (
	"context"

	"github.com/google/uuid"
)

// DeviceRuleRepository 设备规则仓储接口
type DeviceRuleRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, rule *DeviceRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*DeviceRule, error)
	Update(ctx context.Context, rule *DeviceRule) error
	Delete(ctx context.Context, id uuid.UUID) error

	// 列表查询
	List(ctx context.Context, req *RuleListRequest) ([]DeviceRule, int64, error)
	GetAll(ctx context.Context) ([]DeviceRule, error)

	// 按优先级获取启用的规则
	GetEnabledByPriority(ctx context.Context) ([]DeviceRule, error)

	// 检查优先级是否冲突
	ExistsByPriority(ctx context.Context, priority int, excludeID *uuid.UUID) (bool, error)

	// 批量更新优先级
	BatchUpdatePriority(ctx context.Context, items []RuleSortItem) error

	// 获取下一个可用的优先级
	GetNextPriority(ctx context.Context) (int, error)
}

// RuleTaskRepository 规则任务仓储接口
type RuleTaskRepository interface {
	Create(ctx context.Context, task *RuleTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*RuleTask, error)
	Update(ctx context.Context, task *RuleTask) error
	ListByRule(ctx context.Context, ruleID uuid.UUID, limit int) ([]RuleTask, error)
	GetLatestByRule(ctx context.Context, ruleID uuid.UUID) (*RuleTask, error)
}
