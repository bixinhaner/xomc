package geofence

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/storage"
)

func applyCarrierVisibility(
	builder sq.SelectBuilder,
	carrierColumn string,
	visibleGroups []uuid.UUID,
) sq.SelectBuilder {
	if visibleGroups == nil {
		return builder
	}
	if len(visibleGroups) == 0 {
		return builder.Where(sq.Expr("FALSE"))
	}
	visibleCarriers := sq.
		Select("LOWER(scope_device.carrier)").
		Distinct().
		From("devices scope_device").
		Where("scope_device.carrier IS NOT NULL")
	visibleCarriers = authz.ApplyDeviceVisibilityFilter(
		visibleCarriers,
		"scope_device.id",
		visibleGroups,
	)
	return builder.Where(
		sq.Expr("LOWER("+carrierColumn+") IN (?)", visibleCarriers),
	)
}

func buildCarrierVisibilityQuery(
	carrier string,
	visibleGroups []uuid.UUID,
) (string, []any, error) {
	if visibleGroups == nil {
		return "", nil, nil
	}
	builder := storage.Psql.
		Select("1").
		From("devices scope_device").
		Where(sq.Expr("LOWER(scope_device.carrier) = LOWER(?)", carrier)).
		Limit(1)
	builder = authz.ApplyDeviceVisibilityFilter(
		builder,
		"scope_device.id",
		visibleGroups,
	)
	query, args, err := builder.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build carrier visibility query: %w", err)
	}
	return query, args, nil
}

func buildDeviceVisibilityQuery(
	deviceID uuid.UUID,
	visibleGroups []uuid.UUID,
) (string, []any, error) {
	builder := storage.Psql.
		Select("1").
		From("devices scope_device").
		Where(sq.Eq{"scope_device.id": deviceID}).
		Limit(1)
	builder = authz.ApplyDeviceVisibilityFilter(
		builder,
		"scope_device.id",
		visibleGroups,
	)
	query, args, err := builder.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build device visibility query: %w", err)
	}
	return query, args, nil
}
