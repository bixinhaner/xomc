package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// RegistrationRepository defines the persistence interface for device registrations.
type RegistrationRepository interface {
	Create(ctx context.Context, reg *DeviceRegistration) error
	BatchCreate(ctx context.Context, regs []*DeviceRegistration) (int, error)
	GetBySerialNumber(ctx context.Context, sn string) (*DeviceRegistration, error)
	List(ctx context.Context, filter RegistrationFilter) (*model.ListResponse[DeviceRegistration], error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
