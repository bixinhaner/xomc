package license

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// LicenseRepository provides persistence for licenses.
type LicenseRepository interface {
	Create(ctx context.Context, lic *License) error
	GetByID(ctx context.Context, id uuid.UUID) (*License, error)
	GetByCode(ctx context.Context, code string) (*License, error)
	Update(ctx context.Context, lic *License) error
	List(ctx context.Context, filter LicenseFilter) (*model.ListResponse[License], error)
	Summary(ctx context.Context) (*LicenseSummary, error)
}
