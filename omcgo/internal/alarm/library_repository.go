package alarm

import (
	"context"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

type AlarmLibraryRepository interface {
	Create(ctx context.Context, lib *AlarmLibrary) error
	GetByID(ctx context.Context, id uuid.UUID) (*AlarmLibrary, error)
	GetByCode(ctx context.Context, code string) (*AlarmLibrary, error)
	Update(ctx context.Context, lib *AlarmLibrary) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter AlarmLibraryFilter) (*model.ListResponse[AlarmLibrary], error)
	CreateI18n(ctx context.Context, i18n *AlarmLibraryI18n) error
	UpdateI18n(ctx context.Context, i18n *AlarmLibraryI18n) error
	DeleteI18n(ctx context.Context, id uuid.UUID) error
	ListI18n(ctx context.Context, libraryID uuid.UUID) ([]AlarmLibraryI18n, error)
}