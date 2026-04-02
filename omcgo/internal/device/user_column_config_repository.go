package device

import (
	"context"

	"github.com/google/uuid"
)

// ColumnConfigRepository defines the persistence interface for column configs.
type ColumnConfigRepository interface {
	Get(ctx context.Context, userID uuid.UUID, pageKey string) (*ColumnConfig, error)
	Upsert(ctx context.Context, config *ColumnConfig) error
}
