package geofence

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/omcgo/omcgo/internal/core/storage"
)

func buildListMapDefinitionsQuery(
	filter MapDefinitionFilter,
) (string, []any, error) {
	builder := storage.Psql.Select(
		"definition.id",
		"definition.name",
		"definition.carrier",
		"definition.rule_type",
		"definition.owner_device_id",
		"definition.status",
		"definition.current_version_id",
		"definition.created_by",
		"definition.updated_by",
		"definition.created_at",
		"definition.updated_at",
		"version.id",
		"version.geofence_id",
		"version.version",
		"version.status",
		"version.geometry_json",
		"version.bbox_min_longitude",
		"version.bbox_min_latitude",
		"version.bbox_max_longitude",
		"version.bbox_max_latitude",
		"version.policy_json",
		"version.created_by",
		"version.published_by",
		"version.created_at",
		"version.published_at",
	).From("geofence_definitions definition").
		LeftJoin(
			"geofence_versions version " +
				"ON version.id = definition.current_version_id",
		)
	builder = applyCarrierVisibility(
		builder,
		"definition.carrier",
		filter.VisibleGroups,
	)

	if filter.Carrier != "" {
		builder = builder.Where(sq.Eq{"definition.carrier": filter.Carrier})
	}
	if filter.Status != "" {
		builder = builder.Where(sq.Eq{"definition.status": filter.Status})
	} else {
		builder = builder.Where(sq.NotEq{
			"definition.status": DefinitionStatusArchived,
		})
	}
	if filter.Name != "" {
		builder = builder.Where(sq.ILike{
			"definition.name": "%" + filter.Name + "%",
		})
	}
	if filter.Bounds != nil {
		builder = builder.
			Where("definition.current_version_id IS NOT NULL").
			Where(
				"version.bbox_min_longitude <= ?",
				filter.Bounds.MaxLongitude,
			).
			Where(
				"version.bbox_max_longitude >= ?",
				filter.Bounds.MinLongitude,
			).
			Where(
				"version.bbox_min_latitude <= ?",
				filter.Bounds.MaxLatitude,
			).
			Where(
				"version.bbox_max_latitude >= ?",
				filter.Bounds.MinLatitude,
			)
	}
	return builder.
		OrderBy("definition.updated_at DESC", "definition.id").
		ToSql()
}

func (r *PgRepository) ListMapDefinitions(
	ctx context.Context,
	filter MapDefinitionFilter,
) ([]MapDefinition, error) {
	query, args, err := buildListMapDefinitionsQuery(filter)
	if err != nil {
		return nil, fmt.Errorf("build list geofence map definitions: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list geofence map definitions: %w", err)
	}
	defer rows.Close()

	items := make([]MapDefinition, 0)
	for rows.Next() {
		item, err := scanMapDefinition(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofence map definitions: %w", err)
	}
	return items, nil
}

func scanMapDefinition(row interface {
	Scan(dest ...any) error
}) (MapDefinition, error) {
	var item MapDefinition
	var (
		versionID          pgtype.UUID
		versionGeofenceID  pgtype.UUID
		versionNumber      pgtype.Int8
		versionStatus      pgtype.Text
		geometryJSON       []byte
		minLongitude       pgtype.Float8
		minLatitude        pgtype.Float8
		maxLongitude       pgtype.Float8
		maxLatitude        pgtype.Float8
		policyJSON         []byte
		versionCreatedBy   pgtype.UUID
		versionPublishedBy pgtype.UUID
		versionCreatedAt   pgtype.Timestamptz
		versionPublishedAt pgtype.Timestamptz
	)
	err := row.Scan(
		&item.Definition.ID,
		&item.Definition.Name,
		&item.Definition.Carrier,
		&item.Definition.RuleType,
		&item.Definition.OwnerDeviceID,
		&item.Definition.Status,
		&item.Definition.CurrentVersionID,
		&item.Definition.CreatedBy,
		&item.Definition.UpdatedBy,
		&item.Definition.CreatedAt,
		&item.Definition.UpdatedAt,
		&versionID,
		&versionGeofenceID,
		&versionNumber,
		&versionStatus,
		&geometryJSON,
		&minLongitude,
		&minLatitude,
		&maxLongitude,
		&maxLatitude,
		&policyJSON,
		&versionCreatedBy,
		&versionPublishedBy,
		&versionCreatedAt,
		&versionPublishedAt,
	)
	if err != nil {
		return MapDefinition{}, fmt.Errorf(
			"scan geofence map definition: %w",
			err,
		)
	}
	if !versionID.Valid {
		return item, nil
	}
	if !versionGeofenceID.Valid ||
		!versionNumber.Valid ||
		!versionStatus.Valid ||
		!minLongitude.Valid ||
		!minLatitude.Valid ||
		!maxLongitude.Valid ||
		!maxLatitude.Valid ||
		!versionCreatedBy.Valid ||
		!versionCreatedAt.Valid {
		return MapDefinition{}, fmt.Errorf(
			"scan geofence map definition: current version is incomplete",
		)
	}
	version := &Version{
		ID:           uuid.UUID(versionID.Bytes),
		GeofenceID:   uuid.UUID(versionGeofenceID.Bytes),
		Version:      versionNumber.Int64,
		Status:       VersionStatus(versionStatus.String),
		GeometryJSON: json.RawMessage(append([]byte(nil), geometryJSON...)),
		BoundingBox: BoundingBox{
			MinLongitude: minLongitude.Float64,
			MinLatitude:  minLatitude.Float64,
			MaxLongitude: maxLongitude.Float64,
			MaxLatitude:  maxLatitude.Float64,
		},
		PolicyJSON: json.RawMessage(append([]byte(nil), policyJSON...)),
		CreatedBy:  uuid.UUID(versionCreatedBy.Bytes),
		CreatedAt:  versionCreatedAt.Time,
	}
	if versionPublishedBy.Valid {
		publishedBy := uuid.UUID(versionPublishedBy.Bytes)
		version.PublishedBy = &publishedBy
	}
	if versionPublishedAt.Valid {
		publishedAt := versionPublishedAt.Time
		version.PublishedAt = &publishedAt
	}
	item.CurrentVersion = version
	return item, nil
}
