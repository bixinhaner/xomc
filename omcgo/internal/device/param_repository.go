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
	GetByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error)
	CountByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int, error)
	SearchByKeyword(ctx context.Context, deviceID uuid.UUID, keyword string, limit int) ([]model.DeviceParameter, error)
	// GetDirectChildLeaves 获取指定前缀下的直接叶子参数（不含更深层级），支持分页。
	GetDirectChildLeaves(ctx context.Context, deviceID uuid.UUID, prefix string, limit, offset int) ([]model.DeviceParameter, int, error)
}
