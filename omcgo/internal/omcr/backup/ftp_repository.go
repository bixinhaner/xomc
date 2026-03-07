package backup

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// FTPConfigRepository provides persistence for FTP configurations.
type FTPConfigRepository interface {
	Create(ctx context.Context, config *FTPConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*FTPConfig, error)
	Update(ctx context.Context, config *FTPConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter FTPConfigFilter) (*model.ListResponse[FTPConfig], error)
}
