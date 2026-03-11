package filemanager

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// FileRepository provides persistence for managed files.
type FileRepository interface {
	Create(ctx context.Context, file *ManagedFile) error
	GetByID(ctx context.Context, id uuid.UUID) (*ManagedFile, error)
	Update(ctx context.Context, file *ManagedFile) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter FileFilter) (*model.ListResponse[ManagedFile], error)
}
