package slothealth

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type Repository struct {
	metaDB *pgxpool.Pool
	tsDB   *pgxpool.Pool
}

func NewRepository(metaDB, tsDB *pgxpool.Pool) *Repository {
	return &Repository{metaDB: metaDB, tsDB: tsDB}
}

func expectedGroupsQuery(slotStart time.Time) (string, []any, error) {
	return storage.Psql.
		Select(
			"LOWER(v.technology)",
			"LOWER(d.carrier)",
			"COUNT(DISTINCT m.device_id)",
			"STRING_AGG(DISTINCT v.id::text, ',' ORDER BY v.id::text)",
		).
		From("pm_aggregation_task_versions v").
		Join("pm_aggregation_version_members m ON m.task_version_id = v.id").
		Join("devices d ON d.id = m.device_id").
		Where(sq.Eq{"v.enabled": true}).
		Where(sq.LtOrEq{"d.created_at": slotStart}).
		Where(sq.Or{sq.Eq{"d.deleted_at": nil}, sq.Gt{"d.deleted_at": slotStart}}).
		Where(sq.NotEq{"v.technology": nil}).
		Where(sq.LtOrEq{"v.effective_from": slotStart}).
		Where(sq.Or{sq.Eq{"v.effective_to": nil}, sq.Gt{"v.effective_to": slotStart}}).
		GroupBy("LOWER(v.technology)", "LOWER(d.carrier)").
		OrderBy("LOWER(v.technology)", "LOWER(d.carrier)").
		ToSql()
}

func receivedGroupsQuery(slotEnd time.Time) (string, []any, error) {
	return storage.Psql.
		Select("LOWER(technology)", "LOWER(carrier)", "COUNT(DISTINCT device_id)").
		From("pm_files").
		Where(sq.Eq{"measurement_end": slotEnd, "parsed": true}).
		GroupBy("LOWER(technology)", "LOWER(carrier)").
		OrderBy("LOWER(technology)", "LOWER(carrier)").
		ToSql()
}

func upsertSnapshotsQuery(snapshots []Snapshot) (string, []any, error) {
	if len(snapshots) == 0 {
		return "", nil, nil
	}
	builder := storage.Psql.Insert("pm_slot_health").Columns(
		"slot_start", "slot_end", "technology", "carrier",
		"expected_devices", "received_devices", "coverage_ratio",
		"expected_snapshot_version", "evaluated_at", "status",
	)
	for _, snapshot := range snapshots {
		builder = builder.Values(
			snapshot.SlotStart, snapshot.SlotEnd, snapshot.Technology, snapshot.Carrier,
			snapshot.ExpectedDevices, snapshot.ReceivedDevices, snapshot.CoverageRatio,
			snapshot.ExpectedSnapshotVersion, snapshot.EvaluatedAt, string(snapshot.Status),
		)
	}
	return builder.Suffix(`
ON CONFLICT (slot_end, technology, carrier) DO UPDATE SET
  slot_start = CASE WHEN (pm_slot_health.status = 'bootstrap_ignored' AND EXCLUDED.status <> 'bootstrap_ignored') OR ((pm_slot_health.status = 'bootstrap_ignored') = (EXCLUDED.status = 'bootstrap_ignored') AND EXCLUDED.evaluated_at >= pm_slot_health.evaluated_at) THEN EXCLUDED.slot_start ELSE pm_slot_health.slot_start END,
  expected_devices = CASE WHEN (pm_slot_health.status = 'bootstrap_ignored' AND EXCLUDED.status <> 'bootstrap_ignored') OR ((pm_slot_health.status = 'bootstrap_ignored') = (EXCLUDED.status = 'bootstrap_ignored') AND EXCLUDED.evaluated_at >= pm_slot_health.evaluated_at) THEN EXCLUDED.expected_devices ELSE pm_slot_health.expected_devices END,
  received_devices = CASE WHEN (pm_slot_health.status = 'bootstrap_ignored' AND EXCLUDED.status <> 'bootstrap_ignored') OR ((pm_slot_health.status = 'bootstrap_ignored') = (EXCLUDED.status = 'bootstrap_ignored') AND EXCLUDED.evaluated_at >= pm_slot_health.evaluated_at) THEN EXCLUDED.received_devices ELSE pm_slot_health.received_devices END,
  coverage_ratio = CASE WHEN (pm_slot_health.status = 'bootstrap_ignored' AND EXCLUDED.status <> 'bootstrap_ignored') OR ((pm_slot_health.status = 'bootstrap_ignored') = (EXCLUDED.status = 'bootstrap_ignored') AND EXCLUDED.evaluated_at >= pm_slot_health.evaluated_at) THEN EXCLUDED.coverage_ratio ELSE pm_slot_health.coverage_ratio END,
  expected_snapshot_version = CASE WHEN (pm_slot_health.status = 'bootstrap_ignored' AND EXCLUDED.status <> 'bootstrap_ignored') OR ((pm_slot_health.status = 'bootstrap_ignored') = (EXCLUDED.status = 'bootstrap_ignored') AND EXCLUDED.evaluated_at >= pm_slot_health.evaluated_at) THEN EXCLUDED.expected_snapshot_version ELSE pm_slot_health.expected_snapshot_version END,
  evaluated_at = CASE WHEN (pm_slot_health.status = 'bootstrap_ignored' AND EXCLUDED.status <> 'bootstrap_ignored') OR ((pm_slot_health.status = 'bootstrap_ignored') = (EXCLUDED.status = 'bootstrap_ignored') AND EXCLUDED.evaluated_at >= pm_slot_health.evaluated_at) THEN EXCLUDED.evaluated_at ELSE pm_slot_health.evaluated_at END,
  status = CASE WHEN (pm_slot_health.status = 'bootstrap_ignored' AND EXCLUDED.status <> 'bootstrap_ignored') OR ((pm_slot_health.status = 'bootstrap_ignored') = (EXCLUDED.status = 'bootstrap_ignored') AND EXCLUDED.evaluated_at >= pm_slot_health.evaluated_at) THEN EXCLUDED.status ELSE pm_slot_health.status END
RETURNING slot_start, slot_end, technology, carrier,
  expected_devices, received_devices, coverage_ratio,
  expected_snapshot_version, evaluated_at, status`).ToSql()
}

func (r *Repository) ExpectedGroups(ctx context.Context, slotStart time.Time) ([]ExpectedGroup, error) {
	query, args, err := expectedGroupsQuery(slotStart)
	if err != nil {
		return nil, fmt.Errorf("build PM slot expected groups query: %w", err)
	}
	rows, err := r.metaDB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM slot expected groups: %w", err)
	}
	defer rows.Close()
	var groups []ExpectedGroup
	for rows.Next() {
		var group ExpectedGroup
		if err := rows.Scan(&group.Technology, &group.Carrier, &group.Devices, &group.SnapshotVersion); err != nil {
			return nil, fmt.Errorf("scan PM slot expected group: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM slot expected groups: %w", err)
	}
	return groups, nil
}

func (r *Repository) ReceivedGroups(ctx context.Context, slotEnd time.Time) ([]ReceivedGroup, error) {
	query, args, err := receivedGroupsQuery(slotEnd)
	if err != nil {
		return nil, fmt.Errorf("build PM slot received groups query: %w", err)
	}
	rows, err := r.tsDB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM slot received groups: %w", err)
	}
	defer rows.Close()
	var groups []ReceivedGroup
	for rows.Next() {
		var group ReceivedGroup
		if err := rows.Scan(&group.Technology, &group.Carrier, &group.Devices); err != nil {
			return nil, fmt.Errorf("scan PM slot received group: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM slot received groups: %w", err)
	}
	return groups, nil
}

func (r *Repository) UpsertSnapshots(ctx context.Context, snapshots []Snapshot) ([]Snapshot, error) {
	query, args, err := upsertSnapshotsQuery(snapshots)
	if err != nil {
		return nil, fmt.Errorf("build PM slot health upsert: %w", err)
	}
	if query == "" {
		return nil, nil
	}
	rows, err := r.tsDB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("upsert PM slot health: %w", err)
	}
	defer rows.Close()
	persisted := make([]Snapshot, 0, len(snapshots))
	for rows.Next() {
		var snapshot Snapshot
		if err := rows.Scan(
			&snapshot.SlotStart, &snapshot.SlotEnd, &snapshot.Technology, &snapshot.Carrier,
			&snapshot.ExpectedDevices, &snapshot.ReceivedDevices, &snapshot.CoverageRatio,
			&snapshot.ExpectedSnapshotVersion, &snapshot.EvaluatedAt, &snapshot.Status,
		); err != nil {
			return nil, fmt.Errorf("scan persisted PM slot health: %w", err)
		}
		persisted = append(persisted, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate persisted PM slot health: %w", err)
	}
	return persisted, nil
}
