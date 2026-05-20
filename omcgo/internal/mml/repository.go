package mml

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// CommandRepository provides read-only access to predefined MML commands.
type CommandRepository interface {
	List(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error)
	GetByID(ctx context.Context, id uuid.UUID) (*MMLCommand, error)
	GetByCode(ctx context.Context, code string) (*MMLCommand, error)
	// ListByGroupID 返回某个 mml_param_group 下的全部命令（按 sort_order 排序）。
	// Sprint B Q-V3-1 决议：group 作为"批量执行单元"，本方法为 POST
	// /mml/groups/:id/execute 提供命令展开能力。
	ListByGroupID(ctx context.Context, groupID uuid.UUID) ([]MMLCommand, error)
}

// ScriptRepository provides CRUD operations for user-defined MML scripts.
type ScriptRepository interface {
	Create(ctx context.Context, script *MMLScript) error
	GetByID(ctx context.Context, id uuid.UUID) (*MMLScript, error)
	Update(ctx context.Context, script *MMLScript) error
	UpdateLifecycle(ctx context.Context, script *MMLScript) error
	// UpdateLastRun 只写 last_run_status / last_run_at 两列，避免与 Update / UpdateLifecycle
	// 发生字段级覆盖（P1 新增，供 MMLAggregator 回写脚本最近一次执行态）。
	UpdateLastRun(ctx context.Context, id uuid.UUID, status string, at time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error)
}

// TaskRepository provides persistence for MML task execution records.
type TaskRepository interface {
	Create(ctx context.Context, task *MMLTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*MMLTask, error)
	Update(ctx context.Context, task *MMLTask) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error
	IncrementStats(ctx context.Context, id uuid.UUID, successDelta, failedDelta int) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error)
	// ListByScriptID 返回某 mml_script 相关的全部执行实例（模板 + 子实例），
	// 按 created_at 倒序、分页。供脚本详情页 "历史执行" tab 使用
	// （P4 docs/design/mml-task-flow-design-20260424.md §3.3 C11）。
	ListByScriptID(ctx context.Context, scriptID uuid.UUID, req model.ListRequest) (*model.ListResponse[MMLTask], error)
}

// CustomCommandRepository provides CRUD operations for user-defined custom commands.
type CustomCommandRepository interface {
	Create(ctx context.Context, cmd *MMLCustomCommand) error
	GetByID(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error)
	Update(ctx context.Context, cmd *MMLCustomCommand) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error)

	// NameExistsForPrivate 检查同 owner 私有命名空间内是否已存在该 command_name。
	// excludeID 非空时排除自身（用于 Update 改名场景）。
	// 仅在 (owner_user_id, command_name) WHERE command_scope='private' 子集内匹配。
	// 关联：docs/design/mml-user-private-template-crud-20260520.md §4.2
	NameExistsForPrivate(ctx context.Context, ownerID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)
}

// AuditRepository writes MML command execution audit records.
type AuditRepository interface {
	Create(ctx context.Context, entry *MMLAuditLog) error
	CreateBatch(ctx context.Context, entries []*MMLAuditLog) error
}

// CommandParamRepository provides access to MML command-parameter relationships.
// Replaces the old SubCommandRepository — commands now directly reference mml_params.
type CommandParamRepository interface {
	ListByCommandID(ctx context.Context, commandID uuid.UUID) ([]MMLParamRef, error)
	ListByCommandIDs(ctx context.Context, commandIDs []uuid.UUID) (map[uuid.UUID][]MMLParamRef, error)
}
