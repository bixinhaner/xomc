package geofence

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type VersionList struct {
	Items []Version `json:"items"`
}

func buildListVersionsQuery(
	geofenceID uuid.UUID,
) (string, []any, error) {
	return storage.Psql.Select(
		"id",
		"geofence_id",
		"version",
		"status",
		"geometry_json",
		"bbox_min_longitude",
		"bbox_min_latitude",
		"bbox_max_longitude",
		"bbox_max_latitude",
		"policy_json",
		"created_by",
		"published_by",
		"created_at",
		"published_at",
	).From("geofence_versions").
		Where(sq.Eq{"geofence_id": geofenceID}).
		OrderBy("version DESC").
		ToSql()
}

func (r *PgRepository) ListVersions(
	ctx context.Context,
	geofenceID uuid.UUID,
) ([]Version, error) {
	query, args, err := buildListVersionsQuery(geofenceID)
	if err != nil {
		return nil, fmt.Errorf("build list geofence versions: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list geofence versions: %w", err)
	}
	defer rows.Close()

	versions := make([]Version, 0)
	for rows.Next() {
		var version Version
		if err := rows.Scan(
			&version.ID,
			&version.GeofenceID,
			&version.Version,
			&version.Status,
			&version.GeometryJSON,
			&version.BoundingBox.MinLongitude,
			&version.BoundingBox.MinLatitude,
			&version.BoundingBox.MaxLongitude,
			&version.BoundingBox.MaxLatitude,
			&version.PolicyJSON,
			&version.CreatedBy,
			&version.PublishedBy,
			&version.CreatedAt,
			&version.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan geofence version: %w", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofence versions: %w", err)
	}
	return versions, nil
}
