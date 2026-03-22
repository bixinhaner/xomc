package nedirect

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// SessionRepository provides persistence for NE direct connection sessions.
type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*Session, error)
	GetActiveByDeviceAndUser(ctx context.Context, deviceSN, userID string) (*Session, error)
	Update(ctx context.Context, session *Session) error
	List(ctx context.Context, filter SessionFilter) (*model.ListResponse[Session], error)
	ListActiveByDevice(ctx context.Context, deviceSN string) ([]Session, error)
	CloseExpiredSessions(ctx context.Context, timeout int) (int64, error)
}

// CommandRepository provides persistence for NE direct command records.
type CommandRepository interface {
	Create(ctx context.Context, cmd *Command) error
	GetByID(ctx context.Context, id uuid.UUID) (*Command, error)
	Update(ctx context.Context, cmd *Command) error
	List(ctx context.Context, filter CommandFilter) (*model.ListResponse[Command], error)
}
