package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceParameterRepository defines the interface for device parameter persistence.
type DeviceParameterRepository interface {
	BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error
	GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
	GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)
	DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error
}
