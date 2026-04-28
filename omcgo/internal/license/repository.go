package license

import (
	"context"
	"time"

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

	// Enforcement helpers (T-0015 / R-103).
	//
	// GetActiveLicenseWithMaxDevices returns the single active license with
	// the largest MaxDevices, the canonical "enforcement" license when
	// multiple actives exist. Returns (nil, nil) if no active license exists
	// (caller treats this as default-allow + warning).
	GetActiveLicenseWithMaxDevices(ctx context.Context) (*License, error)

	// ListActiveLicenses returns every license currently in active status,
	// used by the cron monitor to scan expiry windows and capacity alerts.
	ListActiveLicenses(ctx context.Context) ([]*License, error)

	// CountDevices returns the current count of registered devices,
	// used as the "used_devices" measurement for capacity enforcement.
	CountDevices(ctx context.Context) (int, error)

	// MarkExpired transitions a license from active → expired.
	// Idempotent: returns nil if already expired.
	MarkExpired(ctx context.Context, id uuid.UUID) error

	// UpdateCapacityAlert records the most recent capacity alert (threshold
	// and timestamp) so the hourly checker can dedupe within a 6h window.
	UpdateCapacityAlert(ctx context.Context, id uuid.UUID, threshold int, at time.Time) error
}
