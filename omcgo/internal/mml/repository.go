package mml

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
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
	List(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error)
}
