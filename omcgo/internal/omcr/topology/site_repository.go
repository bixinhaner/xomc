package topology

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// SiteRepository provides persistence for sites.
type SiteRepository interface {
	Create(ctx context.Context, site *Site) error
	GetByID(ctx context.Context, id uuid.UUID) (*Site, error)
	List(ctx context.Context, filter SiteFilter) (*model.ListResponse[Site], error)
	ListWithCoordinates(ctx context.Context) ([]Site, error)
}

// TopoNodeRepository provides persistence for topology nodes.
type TopoNodeRepository interface {
	List(ctx context.Context, filter TopoNodeFilter) (*model.ListResponse[TopoNode], error)
	ListAll(ctx context.Context, domainID *uuid.UUID) ([]TopoNode, error)
}

// TopoEdgeRepository provides persistence for topology edges.
type TopoEdgeRepository interface {
	List(ctx context.Context, filter TopoEdgeFilter) (*model.ListResponse[TopoEdge], error)
	ListAll(ctx context.Context) ([]TopoEdge, error)
}
