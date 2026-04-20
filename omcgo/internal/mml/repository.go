package mml

import (
	"context"

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
}

// TemplateRepository provides CRUD operations for MML command templates.
type TemplateRepository interface {
	Create(ctx context.Context, tmpl *MMLTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*MMLTemplate, error)
	Update(ctx context.Context, tmpl *MMLTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter TemplateFilter) (*model.ListResponse[MMLTemplate], error)
}

// AuditRepository writes MML command execution audit records.
type AuditRepository interface {
	Create(ctx context.Context, entry *MMLAuditLog) error
	CreateBatch(ctx context.Context, entries []*MMLAuditLog) error
}
