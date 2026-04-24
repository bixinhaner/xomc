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
