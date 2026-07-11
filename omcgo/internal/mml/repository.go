package mml

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ErrScriptVersionConflict indicates that an imported script was changed by
// another writer after the caller read its updated_at version.
var ErrScriptVersionConflict = errors.New("mml script version conflict")

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
	NameExistsForCreator(ctx context.Context, creator, name string, excludeID *uuid.UUID) (bool, error)
	Update(ctx context.Context, script *MMLScript) error
	UpdateLifecycle(ctx context.Context, script *MMLScript) error
	// UpdateLastRun 只写 last_run_status / last_run_at 两列，避免与 Update / UpdateLifecycle
	// 发生字段级覆盖（P1 新增，供 MMLAggregator 回写脚本最近一次执行态）。
	UpdateLastRun(ctx context.Context, id uuid.UUID, status string, at time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error)
}

// ImportedScriptRepository is the persistence boundary for TXT-imported
// scripts. Imported writes intentionally use separate methods so callers do
// not accidentally accept client-supplied content or plan fields.
type ImportedScriptRepository interface {
	ScriptRepository
	CreateImported(ctx context.Context, script *MMLScript) error
	GetByImportSessionID(ctx context.Context, sessionID uuid.UUID) (*MMLScript, error)
	ReplaceImported(ctx context.Context, script *MMLScript, expectedUpdatedAt time.Time) error
	UpdateMetadata(ctx context.Context, id uuid.UUID, name, description string, tags []string) error
}

// TaskRepository provides persistence for MML task execution records.
type TaskRepository interface {
	Create(ctx context.Context, task *MMLTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*MMLTask, error)
	GetByRequestID(ctx context.Context, creator, requestID string) (*MMLTask, error)
	Update(ctx context.Context, task *MMLTask) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error
	IncrementStats(ctx context.Context, id uuid.UUID, successDelta, failedDelta int) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error)
	// ListByScriptID 返回某 mml_script 相关的全部执行实例（模板 + 子实例），
	// 按 created_at 倒序、分页。供脚本详情页 "历史执行" tab 使用
	// （P4 docs/design/mml-task-flow-design-20260424.md §3.3 C11）。
	ListByScriptID(ctx context.Context, scriptID uuid.UUID, req model.ListRequest) (*model.ListResponse[MMLTask], error)
	// UpdateExportAggregate 记录全设备汇总 CSV 的 MinIO object key（mml_tasks.export_object）。
	UpdateExportAggregate(ctx context.Context, id uuid.UUID, objectKey string, t time.Time) error
	// UpdateExportDevice 把单设备 CSV 的 object key 合并进 mml_tasks.device_export_objects 映射。
	UpdateExportDevice(ctx context.Context, id uuid.UUID, deviceSN, objectKey string, t time.Time) error
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

	// NameExistsForPublic 检查公共命名空间内是否已存在该 command_name（全局，跨所有用户）。
	// excludeID 非空时排除自身（用于 Update 改名场景）。
	// 仅在 command_scope='public' 子集内匹配——公共命令对所有用户可见，故须全局唯一。
	// 纯查询防重，公共侧不加 DB 唯一约束。
	NameExistsForPublic(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error)
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

// ScriptValidationRepository loads every external fact needed to validate a
// parsed TXT script. Both methods deliberately accept batches: import files can
// contain 2,000 rows and validation must not perform one database round-trip per
// command or device.
type ScriptValidationRepository interface {
	LoadCommandsByCodes(ctx context.Context, codes []string, actor ValidationActor) (map[string]ValidationCommand, error)
	LoadDevicesBySNs(ctx context.Context, sns []string) (map[string]*model.Device, error)
}
