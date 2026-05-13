package alarm

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// DeviceGroupResolver resolves the current group membership for a device.
type DeviceGroupResolver interface {
	GetGroupID(ctx context.Context, deviceID uuid.UUID) (*uuid.UUID, error)
}

// PgDeviceGroupResolver resolves device group membership from device_group_members.
type PgDeviceGroupResolver struct {
	db *pgxpool.Pool
}

func NewPgDeviceGroupResolver(db *pgxpool.Pool) *PgDeviceGroupResolver {
	return &PgDeviceGroupResolver{db: db}
}

func (r *PgDeviceGroupResolver) GetGroupID(ctx context.Context, deviceID uuid.UUID) (*uuid.UUID, error) {
	query, args, err := storage.Psql.
		Select("group_id").
		From("device_group_members").
		Where(sq.Eq{"device_id": deviceID}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device group query: %w", err)
	}

	var groupID uuid.UUID
	if err := r.db.QueryRow(ctx, query, args...).Scan(&groupID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query device group: %w", err)
	}

	return &groupID, nil
}

var _ DeviceGroupResolver = (*PgDeviceGroupResolver)(nil)