package geofence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/outbox"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type Repository interface {
	CreateDefinitionWithDraft(context.Context, *Definition, *Version) error
	CreateDraftVersion(context.Context, *Version) error
	GetDefinition(context.Context, uuid.UUID) (*Definition, error)
	UpdateDefinitionName(context.Context, uuid.UUID, string, uuid.UUID, time.Time) error
	GetVersion(context.Context, uuid.UUID) (*Version, error)
	ListVersions(context.Context, uuid.UUID) ([]Version, error)
	ListDefinitions(context.Context, DefinitionFilter) ([]Definition, error)
	ListMapDefinitions(context.Context, MapDefinitionFilter) ([]MapDefinition, error)
	IsCarrierVisible(context.Context, string, []uuid.UUID) (bool, error)
	IsDeviceVisible(context.Context, uuid.UUID, []uuid.UUID) (bool, error)
	PublishVersion(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) error
	CreateBinding(context.Context, *Binding) error
	GetBinding(context.Context, uuid.UUID) (*Binding, error)
	ListBindings(context.Context, uuid.UUID) ([]Binding, error)
	ListBindingDetails(
		context.Context,
		BindingDetailFilter,
	) (BindingDetailPage, error)
	GetLifecycleImpact(context.Context, uuid.UUID) (LifecycleImpact, error)
	TransitionDefinition(
		context.Context,
		uuid.UUID,
		DefinitionStatus,
		uuid.UUID,
		string,
		string,
		time.Time,
	) error
	TransitionBinding(
		context.Context,
		uuid.UUID,
		BindingStatus,
		uuid.UUID,
		string,
		time.Time,
	) (*Binding, error)
	LoadManualBindSnapshot(
		context.Context,
		uuid.UUID,
		[]BindingInput,
		[]uuid.UUID,
	) (ManualBindSnapshot, error)
	CreateManualBindJob(
		context.Context,
		CreateManualBindJobParams,
	) (BatchJobAccepted, error)
	GetManualBindJob(context.Context, uuid.UUID) (*BatchJob, error)
	ListManualBindItems(context.Context, BatchItemFilter) (BatchItemPage, error)
}

func buildCreateDraftVersionQueries(
	version *Version,
) ([]repositoryQuery, error) {
	lockSQL, lockArgs, err := storage.Psql.
		Select("status").
		From("geofence_definitions").
		Where(sq.Eq{"id": version.GeofenceID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock geofence for draft version: %w", err)
	}
	insertSQL, insertArgs, err := storage.Psql.
		Insert("geofence_versions").
		Columns(
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
			"created_at",
		).
		Values(
			version.ID,
			version.GeofenceID,
			sq.Expr(
				"(SELECT COALESCE(MAX(version), 0) + 1 "+
					"FROM geofence_versions WHERE geofence_id = ?)",
				version.GeofenceID,
			),
			VersionStatusDraft,
			version.GeometryJSON,
			version.BoundingBox.MinLongitude,
			version.BoundingBox.MinLatitude,
			version.BoundingBox.MaxLongitude,
			version.BoundingBox.MaxLatitude,
			version.PolicyJSON,
			version.CreatedBy,
			version.CreatedAt,
		).
		Suffix("RETURNING version").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create geofence draft version: %w", err)
	}
	return []repositoryQuery{
		{sql: lockSQL, args: lockArgs},
		{sql: insertSQL, args: insertArgs},
	}, nil
}

func (r *PgRepository) CreateDraftVersion(
	ctx context.Context,
	version *Version,
) error {
	queries, err := buildCreateDraftVersionQueries(version)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create geofence draft version: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status DefinitionStatus
	if err := tx.QueryRow(
		ctx,
		queries[0].sql,
		queries[0].args...,
	).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return commonerrors.ErrNotFound
		}
		return fmt.Errorf("lock geofence for draft version: %w", err)
	}
	if status == DefinitionStatusArchived {
		return fmt.Errorf(
			"archived geofence cannot create a draft: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if err := tx.QueryRow(
		ctx,
		queries[1].sql,
		queries[1].args...,
	).Scan(&version.Version); err != nil {
		return fmt.Errorf("create geofence draft version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit geofence draft version: %w", err)
	}
	return nil
}

func buildLifecycleImpactQuery(
	geofenceID uuid.UUID,
) (string, []any, error) {
	deactivationCountQuery := lifecycleDeactivationCandidateQuery(
		geofenceID,
		"COUNT(DISTINCT b.device_id)",
	)
	deactivationSignatureQuery := lifecycleDeactivationCandidateQuery(
		geofenceID,
		"COALESCE(string_agg("+
			"b.id::text || ':' || b.device_id::text || ':' || "+
			"COALESCE(s.last_observation_version::text, ''), "+
			"',' ORDER BY b.id), '')",
	)
	query, args, err := storage.Psql.
		Select(
			"d.id",
			"d.current_version_id",
			"d.status",
			"COUNT(b.id) FILTER (WHERE b.status IN ('active','suspended'))",
			"COUNT(DISTINCT b.device_id) FILTER "+
				"(WHERE b.status IN ('active','suspended'))",
			"COALESCE(string_agg("+
				"b.id::text || ':' || b.device_id::text || ':' || b.status::text, "+
				"',' ORDER BY b.id) FILTER "+
				"(WHERE b.status IN ('active','suspended')), '')",
		).
		Column(sq.Alias(
			deactivationCountQuery.Prefix("(").Suffix(")"),
			"deactivation_device_count",
		)).
		Column(sq.Alias(
			deactivationSignatureQuery.Prefix("(").Suffix(")"),
			"deactivation_signature",
		)).
		Column(sq.Alias(
			activeManualBindJobCountQuery().
				Where(sq.Expr("i.geofence_id = d.id")).
				Prefix("(").
				Suffix(")"),
			"active_batch_job_count",
		)).
		Column(sq.Alias(
			activeManualBindJobSignatureQuery().
				Where(sq.Expr("i.geofence_id = d.id")).
				Prefix("(").
				Suffix(")"),
			"active_batch_job_signature",
		)).
		From("geofence_definitions d").
		LeftJoin(
			"device_geofence_bindings b ON b.geofence_id = d.id",
		).
		Where(sq.Eq{"d.id": geofenceID}).
		GroupBy("d.id", "d.current_version_id", "d.status").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build geofence lifecycle impact: %w", err)
	}
	return query, args, nil
}

func activeManualBindJobCountQuery() sq.SelectBuilder {
	return sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Select("COUNT(DISTINCT i.job_id)").
		From("geofence_batch_items i").
		Join("async_jobs j ON j.id = i.job_id").
		Where(sq.Eq{"j.job_type": ManualBindJobType}).
		Where(sq.Eq{"j.status": []string{"pending", "running"}}).
		Where(sq.Expr("j.payload->>'geofence_id' = i.geofence_id::text"))
}

func activeManualBindJobSignatureQuery() sq.SelectBuilder {
	return sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Select(
			"COALESCE(string_agg(DISTINCT j.id::text, ',' ORDER BY j.id::text), '')",
		).
		From("geofence_batch_items i").
		Join("async_jobs j ON j.id = i.job_id").
		Where(sq.Eq{"j.job_type": ManualBindJobType}).
		Where(sq.Eq{"j.status": []string{"pending", "running"}}).
		Where(sq.Expr("j.payload->>'geofence_id' = i.geofence_id::text"))
}

func buildActiveManualBindJobCountQuery(
	geofenceID uuid.UUID,
) (string, []any, error) {
	query, args, err := activeManualBindJobCountQuery().
		Where(sq.Eq{"i.geofence_id": geofenceID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build active manual bind job count: %w", err)
	}
	return query, args, nil
}

func buildLockLifecycleDefinitionQuery(
	geofenceID uuid.UUID,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select("current_version_id", "status").
		From("geofence_definitions").
		Where(sq.Eq{"id": geofenceID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build lock geofence lifecycle: %w", err)
	}
	return query, args, nil
}

func buildLockLifecycleBindingsQuery(
	geofenceID uuid.UUID,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select("id", "device_id", "status").
		From("device_geofence_bindings").
		Where(sq.Eq{
			"geofence_id": geofenceID,
			"status": []BindingStatus{
				BindingStatusActive,
				BindingStatusSuspended,
			},
		}).
		OrderBy("id").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build lock geofence lifecycle bindings: %w", err)
	}
	return query, args, nil
}

func buildListLifecycleBindingDeviceIDsQuery(
	geofenceID uuid.UUID,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select("DISTINCT device_id").
		From("device_geofence_bindings").
		Where(sq.Eq{
			"geofence_id": geofenceID,
			"status": []BindingStatus{
				BindingStatusActive,
				BindingStatusSuspended,
			},
		}).
		OrderBy("device_id").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf(
			"build list lifecycle binding devices: %w",
			err,
		)
	}
	return query, args, nil
}

func buildLifecycleDeactivationCandidatesQuery(
	geofenceID uuid.UUID,
) (string, []any, error) {
	query, args, err := lifecycleDeactivationCandidateQuery(
		geofenceID,
		"b.id",
		"b.device_id",
		"d.serial_number",
		"s.last_observation_version",
	).
		OrderBy("b.id").
		Suffix("FOR UPDATE OF s").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf(
			"build lifecycle deactivation candidates: %w",
			err,
		)
	}
	return query, args, nil
}

func lifecycleDeactivationCandidateQuery(
	geofenceID uuid.UUID,
	columns ...string,
) sq.SelectBuilder {
	return sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Select(columns...).
		From("device_geofence_bindings b").
		Join("device_geofence_states s ON s.binding_id = b.id").
		Join("devices d ON d.id = b.device_id").
		Join("geofence_definitions gd ON gd.id = b.geofence_id").
		Join("geofence_versions gv ON gv.id = gd.current_version_id").
		Where(sq.Eq{
			"b.geofence_id":     geofenceID,
			"b.status":          BindingStatusActive,
			"s.confirmed_state": ConfirmedStateInside,
			"d.deleted_at":      nil,
		}).
		Where(sq.Expr(
			"gv.policy_json ->> 'exit_action' = ?",
			string(ActionLevelDeactivate),
		))
}

func buildUpdateDefinitionStatusQuery(
	geofenceID uuid.UUID,
	status DefinitionStatus,
	actorID uuid.UUID,
	updatedAt time.Time,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Update("geofence_definitions").
		Set("status", status).
		Set("updated_by", actorID).
		Set("updated_at", updatedAt).
		Where(sq.Eq{"id": geofenceID}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build update geofence lifecycle: %w", err)
	}
	return query, args, nil
}

func buildUpdateDefinitionNameQuery(
	geofenceID uuid.UUID,
	name string,
	actorID uuid.UUID,
	updatedAt time.Time,
) (string, []any, error) {
	query, args, err := storage.Psql.Update("geofence_definitions").
		Set("name", name).
		Set("updated_by", actorID).
		Set("updated_at", updatedAt).
		Where(sq.Eq{"id": geofenceID}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build update geofence name: %w", err)
	}
	return query, args, nil
}

func buildArchiveBindingsQuery(
	geofenceID uuid.UUID,
	actorID uuid.UUID,
	reason string,
	removedAt time.Time,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Update("device_geofence_bindings").
		Set("status", BindingStatusRemoved).
		Set("removed_by", actorID).
		Set("removed_at", removedAt).
		Set("remove_reason", reason).
		Where(sq.Eq{
			"geofence_id": geofenceID,
			"status": []BindingStatus{
				BindingStatusActive,
				BindingStatusSuspended,
			},
		}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build archive geofence bindings: %w", err)
	}
	return query, args, nil
}

func (r *PgRepository) GetLifecycleImpact(
	ctx context.Context,
	geofenceID uuid.UUID,
) (LifecycleImpact, error) {
	return getLifecycleImpact(ctx, r.pool, geofenceID)
}

type lifecycleImpactQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getLifecycleImpact(
	ctx context.Context,
	querier lifecycleImpactQuerier,
	geofenceID uuid.UUID,
) (LifecycleImpact, error) {
	query, args, err := buildLifecycleImpactQuery(geofenceID)
	if err != nil {
		return LifecycleImpact{}, err
	}
	var impact LifecycleImpact
	err = querier.QueryRow(ctx, query, args...).Scan(
		&impact.GeofenceID,
		&impact.CurrentVersionID,
		&impact.CurrentStatus,
		&impact.BindingCount,
		&impact.DeviceCount,
		&impact.bindingSignature,
		&impact.DeactivationDeviceCount,
		&impact.deactivationSignature,
		&impact.ActiveBatchJobCount,
		&impact.activeBatchJobSignature,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleImpact{}, commonerrors.ErrNotFound
	}
	if err != nil {
		return LifecycleImpact{}, lifecycleDatabaseError(
			"get geofence lifecycle impact",
			err,
		)
	}
	return impact, nil
}

func (r *PgRepository) TransitionDefinition(
	ctx context.Context,
	geofenceID uuid.UUID,
	target DefinitionStatus,
	actorID uuid.UUID,
	reason string,
	expectedFingerprint string,
	updatedAt time.Time,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin geofence lifecycle transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lockSQL, lockArgs, err := buildLockLifecycleDefinitionQuery(geofenceID)
	if err != nil {
		return err
	}
	var currentVersionID *uuid.UUID
	var currentStatus DefinitionStatus
	if err := tx.QueryRow(ctx, lockSQL, lockArgs...).Scan(
		&currentVersionID,
		&currentStatus,
	); err != nil {
		if err == pgx.ErrNoRows {
			return commonerrors.ErrNotFound
		}
		return fmt.Errorf("lock geofence lifecycle: %w", err)
	}
	if err := validateDefinitionTransition(currentStatus, target); err != nil {
		return err
	}
	if target == DefinitionStatusEnabled && currentVersionID == nil {
		return fmt.Errorf(
			"geofence has no published version: %w",
			commonerrors.ErrInvalidInput,
		)
	}

	deviceListSQL, deviceListArgs, err :=
		buildListLifecycleBindingDeviceIDsQuery(geofenceID)
	if err != nil {
		return err
	}
	deviceRows, err := tx.Query(ctx, deviceListSQL, deviceListArgs...)
	if err != nil {
		return fmt.Errorf("list geofence lifecycle binding devices: %w", err)
	}
	deviceIDs := make([]uuid.UUID, 0)
	for deviceRows.Next() {
		var deviceID uuid.UUID
		if err := deviceRows.Scan(&deviceID); err != nil {
			deviceRows.Close()
			return fmt.Errorf(
				"scan geofence lifecycle binding device: %w",
				err,
			)
		}
		deviceIDs = append(deviceIDs, deviceID)
	}
	if err := deviceRows.Err(); err != nil {
		deviceRows.Close()
		return fmt.Errorf(
			"iterate geofence lifecycle binding devices: %w",
			err,
		)
	}
	deviceRows.Close()
	lockPlan, err := buildBindingFactMutationLockQueries(
		geofenceID,
		deviceIDs,
		nil,
	)
	if err != nil {
		return err
	}
	lockedDeviceRows, err := tx.Query(
		ctx,
		lockPlan[1].sql,
		lockPlan[1].args...,
	)
	if err != nil {
		return fmt.Errorf("lock geofence lifecycle devices: %w", err)
	}
	for lockedDeviceRows.Next() {
		var (
			deviceID  uuid.UUID
			carrier   string
			deletedAt *time.Time
		)
		if err := lockedDeviceRows.Scan(
			&deviceID,
			&carrier,
			&deletedAt,
		); err != nil {
			lockedDeviceRows.Close()
			return fmt.Errorf(
				"scan locked geofence lifecycle device: %w",
				err,
			)
		}
	}
	if err := lockedDeviceRows.Err(); err != nil {
		lockedDeviceRows.Close()
		return fmt.Errorf(
			"iterate locked geofence lifecycle devices: %w",
			err,
		)
	}
	lockedDeviceRows.Close()

	bindingSQL, bindingArgs, err := buildLockLifecycleBindingsQuery(geofenceID)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, bindingSQL, bindingArgs...)
	if err != nil {
		return fmt.Errorf("lock geofence lifecycle bindings: %w", err)
	}
	var bindingCount int64
	devices := make(map[uuid.UUID]struct{})
	bindingSignatureParts := make([]string, 0)
	for rows.Next() {
		var bindingID, deviceID uuid.UUID
		var status BindingStatus
		if err := rows.Scan(&bindingID, &deviceID, &status); err != nil {
			rows.Close()
			return fmt.Errorf("scan geofence lifecycle binding: %w", err)
		}
		bindingCount++
		devices[deviceID] = struct{}{}
		bindingSignatureParts = append(
			bindingSignatureParts,
			fmt.Sprintf("%s:%s:%s", bindingID, deviceID, status),
		)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate geofence lifecycle bindings: %w", err)
	}
	rows.Close()
	type lifecycleDeactivationCandidate struct {
		bindingID    uuid.UUID
		deviceID     uuid.UUID
		serialNumber string
		observation  *int64
	}
	deactivationCandidates := make([]lifecycleDeactivationCandidate, 0)
	deactivationDevices := make(map[uuid.UUID]struct{})
	deactivationSignatureParts := make([]string, 0)
	if target == DefinitionStatusDisabled || target == DefinitionStatusArchived {
		candidateSQL, candidateArgs, err :=
			buildLifecycleDeactivationCandidatesQuery(geofenceID)
		if err != nil {
			return err
		}
		candidateRows, err := tx.Query(ctx, candidateSQL, candidateArgs...)
		if err != nil {
			return fmt.Errorf("list lifecycle deactivation candidates: %w", err)
		}
		for candidateRows.Next() {
			var candidate lifecycleDeactivationCandidate
			if err := candidateRows.Scan(
				&candidate.bindingID,
				&candidate.deviceID,
				&candidate.serialNumber,
				&candidate.observation,
			); err != nil {
				candidateRows.Close()
				return fmt.Errorf("scan lifecycle deactivation candidate: %w", err)
			}
			deactivationCandidates = append(deactivationCandidates, candidate)
			deactivationDevices[candidate.deviceID] = struct{}{}
			observation := ""
			if candidate.observation != nil {
				observation = fmt.Sprintf("%d", *candidate.observation)
			}
			deactivationSignatureParts = append(
				deactivationSignatureParts,
				fmt.Sprintf(
					"%s:%s:%s",
					candidate.bindingID,
					candidate.deviceID,
					observation,
				),
			)
		}
		if err := candidateRows.Err(); err != nil {
			candidateRows.Close()
			return fmt.Errorf("iterate lifecycle deactivation candidates: %w", err)
		}
		candidateRows.Close()
	}
	activeBatchJobSQL, activeBatchJobArgs, err :=
		buildActiveManualBindJobCountQuery(geofenceID)
	if err != nil {
		return err
	}
	var activeBatchJobCount int64
	if err := tx.QueryRow(
		ctx,
		activeBatchJobSQL,
		activeBatchJobArgs...,
	).Scan(&activeBatchJobCount); err != nil {
		return lifecycleDatabaseError("count active manual bind jobs", err)
	}
	activeBatchJobSignatureSQL, activeBatchJobSignatureArgs, err :=
		activeManualBindJobSignatureQuery().
			Where(sq.Eq{"i.geofence_id": geofenceID}).
			PlaceholderFormat(sq.Dollar).
			ToSql()
	if err != nil {
		return fmt.Errorf("build active manual bind job signature: %w", err)
	}
	var activeBatchJobSignature string
	if err := tx.QueryRow(
		ctx,
		activeBatchJobSignatureSQL,
		activeBatchJobSignatureArgs...,
	).Scan(&activeBatchJobSignature); err != nil {
		return lifecycleDatabaseError("get active manual bind job signature", err)
	}
	if target == DefinitionStatusArchived && activeBatchJobCount != 0 {
		return ErrActiveBatchJobs
	}
	impact := LifecycleImpact{
		GeofenceID:              geofenceID,
		CurrentVersionID:        currentVersionID,
		CurrentStatus:           currentStatus,
		TargetStatus:            target,
		BindingCount:            bindingCount,
		DeviceCount:             int64(len(devices)),
		DeactivationDeviceCount: int64(len(deactivationDevices)),
		ActiveBatchJobCount:     activeBatchJobCount,
		PreviewFingerprint:      expectedFingerprint,
		bindingSignature:        canonicalLifecycleSignature(bindingSignatureParts),
		deactivationSignature:   canonicalLifecycleSignature(deactivationSignatureParts),
		activeBatchJobSignature: activeBatchJobSignature,
	}
	if lifecyclePreviewFingerprint(impact) != expectedFingerprint {
		return ErrStaleLifecyclePreview
	}
	updateSQL, updateArgs, err := buildUpdateDefinitionStatusQuery(
		geofenceID,
		target,
		actorID,
		updatedAt,
	)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, updateSQL, updateArgs...); err != nil {
		return fmt.Errorf("update geofence lifecycle: %w", err)
	}
	if target == DefinitionStatusArchived {
		removeSQL, removeArgs, err := buildArchiveBindingsQuery(
			geofenceID,
			actorID,
			reason,
			updatedAt,
		)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, removeSQL, removeArgs...); err != nil {
			return fmt.Errorf("archive geofence bindings: %w", err)
		}
	}
	for _, candidate := range deactivationCandidates {
		deactivationEvent, err := event.NewEvent(
			event.SubjectGeofenceLifecycleDeactivationRequired,
			event.GeofenceLifecycleDeactivationPayload{
				DeviceID:     candidate.deviceID,
				SerialNumber: candidate.serialNumber,
				GeofenceID:   geofenceID,
				TargetStatus: string(target),
				Reason:       reason,
			},
		)
		if err != nil {
			return fmt.Errorf("build geofence lifecycle deactivation event: %w", err)
		}
		if err := r.outbox.InsertTx(ctx, tx, outbox.Record{
			AggregateType: "device",
			AggregateID:   candidate.deviceID.String(),
			Subject:       event.SubjectGeofenceLifecycleDeactivationRequired,
			Payload:       deactivationEvent.Payload,
			DedupeKey: fmt.Sprintf(
				"geofence:lifecycle-deactivate:%s:%s:%s:%d",
				geofenceID,
				candidate.deviceID,
				target,
				updatedAt.UnixNano(),
			),
		}); err != nil {
			return fmt.Errorf("insert geofence lifecycle deactivation event: %w", err)
		}
	}
	for deviceID := range devices {
		reevaluateEvent, err := event.NewEvent(
			event.SubjectGeofenceLifecycleReevaluate,
			event.GeofenceLifecycleReevaluatePayload{
				DeviceID:   deviceID,
				GeofenceID: geofenceID,
				Reason:     fmt.Sprintf("lifecycle:%s", target),
			},
		)
		if err != nil {
			return fmt.Errorf("build geofence lifecycle reevaluation event: %w", err)
		}
		if err := r.outbox.InsertTx(ctx, tx, outbox.Record{
			AggregateType: "geofence_definition",
			AggregateID:   geofenceID.String(),
			Subject:       event.SubjectGeofenceLifecycleReevaluate,
			Payload:       reevaluateEvent.Payload,
			DedupeKey:     fmt.Sprintf("geofence:lifecycle:%s:%s:%d", geofenceID, deviceID, updatedAt.UnixNano()),
		}); err != nil {
			return fmt.Errorf("insert geofence lifecycle reevaluation event: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit geofence lifecycle transition: %w", err)
	}
	return nil
}

func canonicalLifecycleSignature(parts []string) string {
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func (r *PgRepository) UpdateDefinitionName(
	ctx context.Context,
	geofenceID uuid.UUID,
	name string,
	actorID uuid.UUID,
	updatedAt time.Time,
) error {
	query, args, err := buildUpdateDefinitionNameQuery(geofenceID, name, actorID, updatedAt)
	if err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return definitionNameDatabaseError("update geofence name", err)
	}
	if result.RowsAffected() != 1 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func definitionNameDatabaseError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "uq_geofence_definitions_carrier_name" {
		return fmt.Errorf("%s: %w", operation, duplicateDefinitionNameError(err))
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func lifecycleDatabaseError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("%s: %w", operation, commonerrors.ErrInternal)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func buildBindingFactMutationLockQueries(
	geofenceID uuid.UUID,
	deviceIDs []uuid.UUID,
	bindingID *uuid.UUID,
) ([]repositoryQuery, error) {
	definitionSQL, definitionArgs, err := storage.Psql.
		Select(
			"id", "carrier", "rule_type", "owner_device_id", "status",
			"current_version_id",
		).
		From("geofence_definitions").
		Where(sq.Eq{"id": geofenceID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock binding fact definition: %w", err)
	}

	sortedDeviceIDs := append([]uuid.UUID(nil), deviceIDs...)
	sort.Slice(sortedDeviceIDs, func(i, j int) bool {
		return sortedDeviceIDs[i].String() < sortedDeviceIDs[j].String()
	})
	deviceSQL, deviceArgs, err := storage.Psql.
		Select("id", "carrier", "deleted_at").
		From("devices").
		Where(sq.Eq{"id": sortedDeviceIDs}).
		OrderBy("id").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock binding fact devices: %w", err)
	}

	bindingScope := sq.Or{
		sq.Eq{"status": []BindingStatus{
			BindingStatusActive,
			BindingStatusSuspended,
		}},
	}
	if bindingID != nil {
		bindingScope = append(bindingScope, sq.Eq{"id": *bindingID})
	}
	bindingSQL, bindingArgs, err := storage.Psql.
		Select(
			"id", "device_id", "geofence_id", "rule_type", "status",
			"bind_source", "bound_by", "bound_at", "removed_by",
			"removed_at", "remove_reason",
		).
		From("device_geofence_bindings").
		Where(sq.Eq{"device_id": sortedDeviceIDs}).
		Where(bindingScope).
		OrderBy("id").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock binding facts: %w", err)
	}
	return []repositoryQuery{
		{sql: definitionSQL, args: definitionArgs},
		{sql: deviceSQL, args: deviceArgs},
		{sql: bindingSQL, args: bindingArgs},
	}, nil
}

type bindingFactMutationSnapshot struct {
	definition Definition
	devices    map[uuid.UUID]DeviceIdentity
	bindings   []Binding
}

func (r *PgRepository) lockBindingFactMutation(
	ctx context.Context,
	tx pgx.Tx,
	geofenceID uuid.UUID,
	deviceIDs []uuid.UUID,
	bindingID *uuid.UUID,
) (bindingFactMutationSnapshot, error) {
	queries, err := buildBindingFactMutationLockQueries(
		geofenceID,
		deviceIDs,
		bindingID,
	)
	if err != nil {
		return bindingFactMutationSnapshot{}, err
	}
	snapshot := bindingFactMutationSnapshot{
		devices:  make(map[uuid.UUID]DeviceIdentity, len(deviceIDs)),
		bindings: make([]Binding, 0),
	}
	if err := tx.QueryRow(
		ctx,
		queries[0].sql,
		queries[0].args...,
	).Scan(
		&snapshot.definition.ID,
		&snapshot.definition.Carrier,
		&snapshot.definition.RuleType,
		&snapshot.definition.OwnerDeviceID,
		&snapshot.definition.Status,
		&snapshot.definition.CurrentVersionID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return bindingFactMutationSnapshot{}, commonerrors.ErrNotFound
		}
		return bindingFactMutationSnapshot{}, fmt.Errorf(
			"lock binding fact definition: %w",
			err,
		)
	}
	if r.hooks != nil && r.hooks.afterBindingDefinitionLock != nil {
		if err := r.hooks.afterBindingDefinitionLock(ctx); err != nil {
			return bindingFactMutationSnapshot{}, fmt.Errorf(
				"after binding definition lock: %w",
				err,
			)
		}
	}

	deviceRows, err := tx.Query(ctx, queries[1].sql, queries[1].args...)
	if err != nil {
		return bindingFactMutationSnapshot{}, fmt.Errorf(
			"lock binding fact devices: %w",
			err,
		)
	}
	for deviceRows.Next() {
		var (
			device    DeviceIdentity
			deletedAt *time.Time
		)
		if err := deviceRows.Scan(
			&device.ID,
			&device.Carrier,
			&deletedAt,
		); err != nil {
			deviceRows.Close()
			return bindingFactMutationSnapshot{}, fmt.Errorf(
				"scan binding fact device: %w",
				err,
			)
		}
		if deletedAt == nil {
			snapshot.devices[device.ID] = device
		}
	}
	if err := deviceRows.Err(); err != nil {
		deviceRows.Close()
		return bindingFactMutationSnapshot{}, fmt.Errorf(
			"iterate binding fact devices: %w",
			err,
		)
	}
	deviceRows.Close()

	bindingRows, err := tx.Query(ctx, queries[2].sql, queries[2].args...)
	if err != nil {
		return bindingFactMutationSnapshot{}, fmt.Errorf(
			"lock binding facts: %w",
			err,
		)
	}
	for bindingRows.Next() {
		var (
			binding      Binding
			removeReason *string
		)
		if err := bindingRows.Scan(
			&binding.ID,
			&binding.DeviceID,
			&binding.GeofenceID,
			&binding.RuleType,
			&binding.Status,
			&binding.BindSource,
			&binding.BoundBy,
			&binding.BoundAt,
			&binding.RemovedBy,
			&binding.RemovedAt,
			&removeReason,
		); err != nil {
			bindingRows.Close()
			return bindingFactMutationSnapshot{}, fmt.Errorf(
				"scan locked binding fact: %w",
				err,
			)
		}
		if removeReason != nil {
			binding.RemoveReason = *removeReason
		}
		snapshot.bindings = append(snapshot.bindings, binding)
	}
	if err := bindingRows.Err(); err != nil {
		bindingRows.Close()
		return bindingFactMutationSnapshot{}, fmt.Errorf(
			"iterate locked binding facts: %w",
			err,
		)
	}
	bindingRows.Close()
	return snapshot, nil
}

func buildBindingMutationRouteQuery(
	bindingID uuid.UUID,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select("device_id", "geofence_id").
		From("device_geofence_bindings").
		Where(sq.Eq{"id": bindingID}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build binding mutation route: %w", err)
	}
	return query, args, nil
}

func buildUpdateBindingLifecycleQuery(
	bindingID uuid.UUID,
	target BindingStatus,
	actorID uuid.UUID,
	reason string,
	updatedAt time.Time,
) (string, []any, error) {
	builder := storage.Psql.
		Update("device_geofence_bindings").
		Set("status", target).
		Where(sq.Eq{"id": bindingID})
	if target == BindingStatusRemoved {
		builder = builder.
			Set("removed_by", actorID).
			Set("removed_at", updatedAt).
			Set("remove_reason", reason)
	}
	query, args, err := builder.
		Suffix(
			"RETURNING id, device_id, geofence_id, rule_type, status, " +
				"bind_source, bound_by, bound_at, removed_by, removed_at, " +
				"remove_reason",
		).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build update geofence binding lifecycle: %w", err)
	}
	return query, args, nil
}

func (r *PgRepository) TransitionBinding(
	ctx context.Context,
	bindingID uuid.UUID,
	target BindingStatus,
	actorID uuid.UUID,
	reason string,
	updatedAt time.Time,
) (*Binding, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin geofence binding transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	routeSQL, routeArgs, err := buildBindingMutationRouteQuery(bindingID)
	if err != nil {
		return nil, err
	}
	var routeDeviceID, routeGeofenceID uuid.UUID
	if err := tx.QueryRow(ctx, routeSQL, routeArgs...).Scan(
		&routeDeviceID,
		&routeGeofenceID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("route geofence binding transition: %w", err)
	}

	locked, err := r.lockBindingFactMutation(
		ctx,
		tx,
		routeGeofenceID,
		[]uuid.UUID{routeDeviceID},
		&bindingID,
	)
	if err != nil {
		return nil, err
	}
	var binding *Binding
	for index := range locked.bindings {
		if locked.bindings[index].ID == bindingID {
			binding = &locked.bindings[index]
			break
		}
	}
	if binding == nil ||
		binding.DeviceID != routeDeviceID ||
		binding.GeofenceID != routeGeofenceID {
		return nil, commonerrors.ErrNotFound
	}
	if err := validateBindingTransition(binding.Status, target); err != nil {
		return nil, err
	}
	if target == BindingStatusActive {
		if err := validateBindingResumeParent(
			locked.definition.Status,
			locked.definition.CurrentVersionID,
		); err != nil {
			return nil, err
		}
		for _, existing := range locked.bindings {
			if existing.ID != binding.ID &&
				existing.DeviceID == binding.DeviceID &&
				existing.RuleType == binding.RuleType &&
				existing.Status == BindingStatusActive {
				return nil, fmt.Errorf(
					"active geofence binding already exists: %w",
					commonerrors.ErrAlreadyExists,
				)
			}
		}
	}

	updateSQL, updateArgs, err := buildUpdateBindingLifecycleQuery(
		bindingID,
		target,
		actorID,
		reason,
		updatedAt,
	)
	if err != nil {
		return nil, err
	}
	var removeReason *string
	if err := tx.QueryRow(ctx, updateSQL, updateArgs...).Scan(
		&binding.ID,
		&binding.DeviceID,
		&binding.GeofenceID,
		&binding.RuleType,
		&binding.Status,
		&binding.BindSource,
		&binding.BoundBy,
		&binding.BoundAt,
		&binding.RemovedBy,
		&binding.RemovedAt,
		&removeReason,
	); err != nil {
		if isActiveBindingConflict(err) {
			return nil, fmt.Errorf(
				"active geofence binding already exists: %w",
				commonerrors.ErrAlreadyExists,
			)
		}
		return nil, fmt.Errorf("update geofence binding lifecycle: %w", err)
	}
	if removeReason != nil {
		binding.RemoveReason = *removeReason
	} else {
		binding.RemoveReason = ""
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit geofence binding transition: %w", err)
	}
	return binding, nil
}

type PgRepository struct {
	pool   *pgxpool.Pool
	hooks  *pgRepositoryHooks
	outbox *outbox.Repository
}

type pgRepositoryHooks struct {
	afterManualBindDefinitionLock func(context.Context) error
	afterManualBindConflict       func(context.Context) error
	afterManualBindItemLock       func(context.Context) error
	afterBindingDefinitionLock    func(context.Context) error
}

type repositoryQuery struct {
	sql  string
	args []any
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool, outbox: outbox.NewRepository()}
}

func buildCreateDefinitionWithDraftQueries(
	definition *Definition,
	version *Version,
) (string, []any, string, []any, error) {
	definitionSQL, definitionArgs, err := storage.Psql.
		Insert("geofence_definitions").
		Columns(
			"id", "name", "carrier", "rule_type", "owner_device_id", "status",
			"created_by", "updated_by", "created_at", "updated_at",
		).
		Values(
			definition.ID, definition.Name, definition.Carrier,
			definition.RuleType, definition.OwnerDeviceID, definition.Status,
			definition.CreatedBy, definition.UpdatedBy,
			definition.CreatedAt, definition.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return "", nil, "", nil, fmt.Errorf("build create geofence definition: %w", err)
	}
	versionSQL, versionArgs, err := storage.Psql.
		Insert("geofence_versions").
		Columns(
			"id", "geofence_id", "version", "status", "geometry_json",
			"bbox_min_longitude", "bbox_min_latitude",
			"bbox_max_longitude", "bbox_max_latitude",
			"policy_json", "created_by", "created_at",
		).
		Values(
			version.ID, version.GeofenceID, version.Version, version.Status,
			version.GeometryJSON,
			version.BoundingBox.MinLongitude, version.BoundingBox.MinLatitude,
			version.BoundingBox.MaxLongitude, version.BoundingBox.MaxLatitude,
			version.PolicyJSON, version.CreatedBy, version.CreatedAt,
		).
		ToSql()
	if err != nil {
		return "", nil, "", nil, fmt.Errorf("build create geofence draft version: %w", err)
	}
	return definitionSQL, definitionArgs, versionSQL, versionArgs, nil
}

func (r *PgRepository) CreateDefinitionWithDraft(
	ctx context.Context,
	definition *Definition,
	version *Version,
) error {
	definitionSQL, definitionArgs, versionSQL, versionArgs, err :=
		buildCreateDefinitionWithDraftQueries(definition, version)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create geofence: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, definitionSQL, definitionArgs...); err != nil {
		return definitionNameDatabaseError("create geofence definition", err)
	}
	if _, err := tx.Exec(ctx, versionSQL, versionArgs...); err != nil {
		return fmt.Errorf("create geofence draft version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create geofence: %w", err)
	}
	return nil
}

func (r *PgRepository) GetDefinition(ctx context.Context, id uuid.UUID) (*Definition, error) {
	query, args, err := storage.Psql.Select(
		"id", "name", "carrier", "rule_type", "owner_device_id", "status",
		"current_version_id", "created_by", "updated_by", "created_at", "updated_at",
	).From("geofence_definitions").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get geofence definition: %w", err)
	}
	var definition Definition
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&definition.ID, &definition.Name, &definition.Carrier, &definition.RuleType,
		&definition.OwnerDeviceID, &definition.Status, &definition.CurrentVersionID,
		&definition.CreatedBy, &definition.UpdatedBy,
		&definition.CreatedAt, &definition.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get geofence definition: %w", err)
	}
	return &definition, nil
}

func (r *PgRepository) GetVersion(ctx context.Context, id uuid.UUID) (*Version, error) {
	query, args, err := storage.Psql.Select(
		"id", "geofence_id", "version", "status", "geometry_json",
		"bbox_min_longitude", "bbox_min_latitude",
		"bbox_max_longitude", "bbox_max_latitude",
		"policy_json", "created_by", "published_by", "created_at", "published_at",
	).From("geofence_versions").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get geofence version: %w", err)
	}
	var version Version
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&version.ID, &version.GeofenceID, &version.Version, &version.Status,
		&version.GeometryJSON,
		&version.BoundingBox.MinLongitude, &version.BoundingBox.MinLatitude,
		&version.BoundingBox.MaxLongitude, &version.BoundingBox.MaxLatitude,
		&version.PolicyJSON, &version.CreatedBy, &version.PublishedBy,
		&version.CreatedAt, &version.PublishedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get geofence version: %w", err)
	}
	return &version, nil
}

func (r *PgRepository) ListDefinitions(
	ctx context.Context,
	filter DefinitionFilter,
) ([]Definition, error) {
	query, args, err := buildListDefinitionsQuery(filter)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list geofence definitions: %w", err)
	}
	defer rows.Close()
	definitions := make([]Definition, 0)
	for rows.Next() {
		var definition Definition
		if err := rows.Scan(
			&definition.ID, &definition.Name, &definition.Carrier, &definition.RuleType,
			&definition.OwnerDeviceID, &definition.Status, &definition.CurrentVersionID,
			&definition.CreatedBy, &definition.UpdatedBy,
			&definition.CreatedAt, &definition.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan geofence definition: %w", err)
		}
		definitions = append(definitions, definition)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofence definitions: %w", err)
	}
	return definitions, nil
}

func buildListDefinitionsQuery(
	filter DefinitionFilter,
) (string, []any, error) {
	builder := storage.Psql.Select(
		"id", "name", "carrier", "rule_type", "owner_device_id", "status",
		"current_version_id", "created_by", "updated_by", "created_at", "updated_at",
	).From("geofence_definitions").OrderBy("updated_at DESC")
	builder = applyCarrierVisibility(builder, "carrier", filter.VisibleGroups)
	if filter.Carrier != "" {
		builder = builder.Where(sq.Eq{"carrier": filter.Carrier})
	}
	if filter.Status != "" {
		builder = builder.Where(sq.Eq{"status": filter.Status})
	}
	if filter.Name != "" {
		builder = builder.Where(sq.ILike{"name": "%" + filter.Name + "%"})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build list geofence definitions: %w", err)
	}
	return query, args, nil
}

func (r *PgRepository) IsCarrierVisible(
	ctx context.Context,
	carrier string,
	visibleGroups []uuid.UUID,
) (bool, error) {
	if visibleGroups == nil {
		return true, nil
	}
	query, args, err := buildCarrierVisibilityQuery(carrier, visibleGroups)
	if err != nil {
		return false, err
	}
	var marker int
	err = r.pool.QueryRow(ctx, query, args...).Scan(&marker)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check geofence carrier visibility: %w", err)
	}
	return true, nil
}

func (r *PgRepository) IsDeviceVisible(
	ctx context.Context,
	deviceID uuid.UUID,
	visibleGroups []uuid.UUID,
) (bool, error) {
	if visibleGroups == nil {
		return true, nil
	}
	query, args, err := buildDeviceVisibilityQuery(
		deviceID,
		visibleGroups,
	)
	if err != nil {
		return false, err
	}
	var marker int
	err = r.pool.QueryRow(ctx, query, args...).Scan(&marker)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check geofence device visibility: %w", err)
	}
	return true, nil
}

func isActiveBindingConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "uq_device_geofence_active_rule_type"
}

func buildPublishLockQueries(
	geofenceID uuid.UUID,
	versionID uuid.UUID,
) ([]repositoryQuery, error) {
	definitionSQL, definitionArgs, err := storage.Psql.
		Select("current_version_id").
		From("geofence_definitions").
		Where(sq.Eq{"id": geofenceID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock geofence definition: %w", err)
	}
	versionSQL, versionArgs, err := storage.Psql.
		Select("geofence_id", "status").
		From("geofence_versions").
		Where(sq.Eq{"id": versionID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock geofence version: %w", err)
	}
	return []repositoryQuery{
		{sql: definitionSQL, args: definitionArgs},
		{sql: versionSQL, args: versionArgs},
	}, nil
}

func buildPublishMutationQueries(
	geofenceID uuid.UUID,
	versionID uuid.UUID,
	currentVersionID *uuid.UUID,
	actorID uuid.UUID,
	publishedAt time.Time,
) ([]repositoryQuery, error) {
	queries := make([]repositoryQuery, 0, 3)
	if currentVersionID != nil {
		query, args, err := storage.Psql.
			Update("geofence_versions").
			Set("status", VersionStatusSuperseded).
			Where(sq.Eq{
				"id":     *currentVersionID,
				"status": VersionStatusPublished,
			}).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build supersede previous geofence version: %w", err)
		}
		queries = append(queries, repositoryQuery{sql: query, args: args})
	}

	publishSQL, publishArgs, err := storage.Psql.
		Update("geofence_versions").
		Set("status", VersionStatusPublished).
		Set("published_by", actorID).
		Set("published_at", publishedAt).
		Where(sq.Eq{"id": versionID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build publish geofence version: %w", err)
	}
	queries = append(queries, repositoryQuery{sql: publishSQL, args: publishArgs})

	activateSQL, activateArgs, err := storage.Psql.
		Update("geofence_definitions").
		Set("current_version_id", versionID).
		Set("status", DefinitionStatusEnabled).
		Set("updated_by", actorID).
		Set("updated_at", publishedAt).
		Where(sq.Eq{"id": geofenceID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build activate geofence version: %w", err)
	}
	queries = append(queries, repositoryQuery{sql: activateSQL, args: activateArgs})
	return queries, nil
}

func buildPublishVersionQueries(
	geofenceID uuid.UUID,
	versionID uuid.UUID,
	currentVersionID *uuid.UUID,
	actorID uuid.UUID,
	publishedAt time.Time,
) ([]repositoryQuery, error) {
	locks, err := buildPublishLockQueries(geofenceID, versionID)
	if err != nil {
		return nil, err
	}
	mutations, err := buildPublishMutationQueries(
		geofenceID,
		versionID,
		currentVersionID,
		actorID,
		publishedAt,
	)
	if err != nil {
		return nil, err
	}
	return append(locks, mutations...), nil
}

func (r *PgRepository) PublishVersion(
	ctx context.Context,
	geofenceID, versionID, actorID uuid.UUID,
	publishedAt time.Time,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin publish geofence version: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lockQueries, err := buildPublishLockQueries(geofenceID, versionID)
	if err != nil {
		return err
	}
	var currentVersionID *uuid.UUID
	if err := tx.QueryRow(
		ctx,
		lockQueries[0].sql,
		lockQueries[0].args...,
	).Scan(&currentVersionID); err != nil {
		if err == pgx.ErrNoRows {
			return commonerrors.ErrNotFound
		}
		return fmt.Errorf("lock geofence definition: %w", err)
	}

	var ownerID uuid.UUID
	var status VersionStatus
	if err := tx.QueryRow(
		ctx,
		lockQueries[1].sql,
		lockQueries[1].args...,
	).Scan(&ownerID, &status); err != nil {
		if err == pgx.ErrNoRows {
			return commonerrors.ErrNotFound
		}
		return fmt.Errorf("lock geofence version: %w", err)
	}
	if ownerID != geofenceID || status != VersionStatusDraft {
		return fmt.Errorf("version is not a draft of the requested geofence: %w", commonerrors.ErrInvalidInput)
	}

	mutations, err := buildPublishMutationQueries(
		geofenceID,
		versionID,
		currentVersionID,
		actorID,
		publishedAt,
	)
	if err != nil {
		return err
	}
	for index, mutation := range mutations {
		if _, err := tx.Exec(ctx, mutation.sql, mutation.args...); err != nil {
			if currentVersionID != nil && index == 0 {
				return fmt.Errorf("supersede previous geofence version: %w", err)
			}
			if index == len(mutations)-1 {
				return fmt.Errorf("activate geofence version: %w", err)
			}
			return fmt.Errorf("publish geofence version: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit publish geofence version: %w", err)
	}
	return nil
}

func buildBindingStateQueries(binding *Binding) (repositoryQuery, repositoryQuery, error) {
	return buildBindingStateQueriesWithConfirmedState(
		binding,
		ConfirmedStateUnknown,
	)
}

func buildBindingStateQueriesWithConfirmedState(
	binding *Binding,
	confirmedState ConfirmedState,
) (repositoryQuery, repositoryQuery, error) {
	bindingStateSQL, bindingStateArgs, err := storage.Psql.
		Insert("device_geofence_states").
		Columns(
			"binding_id",
			"device_id",
			"confirmed_state",
			"candidate_count",
			"state_version",
		).
		Values(binding.ID, binding.DeviceID, confirmedState, 0, 1).
		ToSql()
	if err != nil {
		return repositoryQuery{}, repositoryQuery{},
			fmt.Errorf("build initialize geofence binding state: %w", err)
	}

	effectiveStateSQL, effectiveStateArgs, err := storage.Psql.
		Insert("device_geofence_effective_states").
		Columns(
			"device_id",
			"effective_state",
			"required_action_level",
			"state_version",
			"evaluation_health",
		).
		Values(binding.DeviceID, "unknown", "none", 1, "healthy").
		Suffix(`
ON CONFLICT (device_id) DO UPDATE
SET effective_state = CASE
      WHEN device_geofence_effective_states.effective_state = 'unmanaged'
      THEN 'unknown'
      ELSE device_geofence_effective_states.effective_state
    END,
    updated_at = now()`).
		ToSql()
	if err != nil {
		return repositoryQuery{}, repositoryQuery{},
			fmt.Errorf("build initialize effective geofence state: %w", err)
	}
	return repositoryQuery{sql: bindingStateSQL, args: bindingStateArgs},
		repositoryQuery{sql: effectiveStateSQL, args: effectiveStateArgs},
		nil
}

func (r *PgRepository) CreateBinding(ctx context.Context, binding *Binding) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create geofence binding: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	locked, err := r.lockBindingFactMutation(
		ctx,
		tx,
		binding.GeofenceID,
		[]uuid.UUID{binding.DeviceID},
		nil,
	)
	if err != nil {
		return err
	}
	device, ok := locked.devices[binding.DeviceID]
	if !ok {
		return commonerrors.ErrNotFound
	}
	if binding.Status != BindingStatusActive ||
		locked.definition.Status != DefinitionStatusEnabled ||
		locked.definition.CurrentVersionID == nil ||
		binding.RuleType != locked.definition.RuleType ||
		device.Carrier != locked.definition.Carrier {
		return fmt.Errorf(
			"binding facts changed before create: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if locked.definition.RuleType == RuleTypeBaselineRadius &&
		(locked.definition.OwnerDeviceID == nil ||
			*locked.definition.OwnerDeviceID != binding.DeviceID) {
		return fmt.Errorf(
			"binding baseline owner changed before create: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	for _, existing := range locked.bindings {
		if existing.DeviceID != binding.DeviceID {
			continue
		}
		if existing.GeofenceID == binding.GeofenceID ||
			(existing.RuleType == binding.RuleType &&
				existing.Status == BindingStatusActive) {
			return commonerrors.ErrAlreadyExists
		}
	}

	if err := createBindingTx(ctx, tx, binding); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create geofence binding: %w", err)
	}
	return nil
}

func createBindingTx(
	ctx context.Context,
	tx pgx.Tx,
	binding *Binding,
) error {
	return createBindingTxWithConfirmedState(
		ctx,
		tx,
		binding,
		ConfirmedStateUnknown,
	)
}

func createBindingTxWithConfirmedState(
	ctx context.Context,
	tx pgx.Tx,
	binding *Binding,
	confirmedState ConfirmedState,
) error {
	query, args, err := storage.Psql.Insert("device_geofence_bindings").
		Columns(
			"id", "device_id", "geofence_id", "rule_type", "status",
			"bind_source", "bound_by", "bound_at",
		).
		Values(
			binding.ID, binding.DeviceID, binding.GeofenceID, binding.RuleType,
			binding.Status, binding.BindSource, binding.BoundBy, binding.BoundAt,
		).ToSql()
	if err != nil {
		return fmt.Errorf("build create geofence binding: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		if isActiveBindingConflict(err) {
			return commonerrors.ErrAlreadyExists
		}
		return fmt.Errorf("create geofence binding: %w", err)
	}
	bindingState, effectiveState, err := buildBindingStateQueriesWithConfirmedState(
		binding,
		confirmedState,
	)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, bindingState.sql, bindingState.args...); err != nil {
		return fmt.Errorf("initialize geofence binding state: %w", err)
	}
	if _, err := tx.Exec(ctx, effectiveState.sql, effectiveState.args...); err != nil {
		return fmt.Errorf("initialize effective geofence state: %w", err)
	}
	return nil
}

func (r *PgRepository) GetBinding(ctx context.Context, id uuid.UUID) (*Binding, error) {
	query, args, err := storage.Psql.Select(
		"id", "device_id", "geofence_id", "rule_type", "status", "bind_source",
		"bound_by", "bound_at", "removed_by", "removed_at",
		"COALESCE(remove_reason, '')",
	).From("device_geofence_bindings").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get geofence binding: %w", err)
	}
	var binding Binding
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&binding.ID, &binding.DeviceID, &binding.GeofenceID, &binding.RuleType,
		&binding.Status, &binding.BindSource, &binding.BoundBy, &binding.BoundAt,
		&binding.RemovedBy, &binding.RemovedAt, &binding.RemoveReason,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get geofence binding: %w", err)
	}
	return &binding, nil
}

func (r *PgRepository) ListBindings(
	ctx context.Context,
	geofenceID uuid.UUID,
) ([]Binding, error) {
	query, args, err := storage.Psql.Select(
		"id", "device_id", "geofence_id", "rule_type", "status", "bind_source",
		"bound_by", "bound_at", "removed_by", "removed_at",
		"COALESCE(remove_reason, '')",
	).From("device_geofence_bindings").
		Where(sq.Eq{"geofence_id": geofenceID}).
		OrderBy("bound_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list geofence bindings: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list geofence bindings: %w", err)
	}
	defer rows.Close()
	bindings := make([]Binding, 0)
	for rows.Next() {
		var binding Binding
		if err := rows.Scan(
			&binding.ID, &binding.DeviceID, &binding.GeofenceID, &binding.RuleType,
			&binding.Status, &binding.BindSource, &binding.BoundBy, &binding.BoundAt,
			&binding.RemovedBy, &binding.RemovedAt, &binding.RemoveReason,
		); err != nil {
			return nil, fmt.Errorf("scan geofence binding: %w", err)
		}
		bindings = append(bindings, binding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofence bindings: %w", err)
	}
	return bindings, nil
}

func (r *PgRepository) LoadManualBindSnapshot(
	ctx context.Context,
	geofenceID uuid.UUID,
	inputs []BindingInput,
	visibleGroups []uuid.UUID,
) (ManualBindSnapshot, error) {
	definition, err := r.GetDefinition(ctx, geofenceID)
	if err != nil {
		return ManualBindSnapshot{}, fmt.Errorf("get geofence definition: %w", err)
	}
	if definition == nil {
		return ManualBindSnapshot{}, commonerrors.ErrNotFound
	}
	devices, err := loadVisibleBatchDevices(
		ctx,
		r.pool,
		inputs,
		definition.Carrier,
		visibleGroups,
	)
	if err != nil {
		return ManualBindSnapshot{}, err
	}
	deviceFacts, err := loadBatchBindingFacts(
		ctx,
		r.pool,
		definition.ID,
		devices,
	)
	if err != nil {
		return ManualBindSnapshot{}, err
	}
	return buildBatchSnapshot(
		*definition,
		inputs,
		devices,
		deviceFacts,
		visibleGroups,
	)
}

type manualBindingDevices struct {
	byID           map[uuid.UUID]*DeviceIdentity
	bySerialNumber map[string]*DeviceIdentity
}

func newManualBindingDevices() manualBindingDevices {
	return manualBindingDevices{
		byID:           make(map[uuid.UUID]*DeviceIdentity),
		bySerialNumber: make(map[string]*DeviceIdentity),
	}
}

func (devices manualBindingDevices) add(
	identity *DeviceIdentity,
	definitionCarrier string,
) {
	devices.byID[identity.ID] = identity
	if identity.Carrier == definitionCarrier {
		devices.bySerialNumber[identity.SerialNumber] = identity
	}
}

func bindingInputValues(inputs []BindingInput) ([]uuid.UUID, []string) {
	ids := make([]uuid.UUID, 0, len(inputs))
	serialNumbers := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if input.Kind == BindingInputDeviceID && input.DeviceID != nil {
			ids = append(ids, *input.DeviceID)
		}
		if input.Kind == BindingInputDeviceSN {
			serialNumbers = append(serialNumbers, input.Value)
		}
	}
	return ids, serialNumbers
}

type manualBindingFacts struct {
	sameGeofenceBindingIDs map[uuid.UUID]uuid.UUID
	sameGeofenceStatus     map[uuid.UUID]BindingStatus
	activeRuleConflicts    map[uuid.UUID]manualBindingConflict
}

type manualBindingConflict struct {
	bindingID  uuid.UUID
	geofenceID uuid.UUID
}

func manualBindingVisibilityDigest(visibleGroups []uuid.UUID) string {
	if visibleGroups == nil {
		return "scope:all"
	}
	values := make([]string, 0, len(visibleGroups))
	seen := make(map[uuid.UUID]struct{}, len(visibleGroups))
	for _, groupID := range visibleGroups {
		if groupID == uuid.Nil {
			continue
		}
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		values = append(values, groupID.String())
	}
	sort.Strings(values)
	if len(values) == 0 {
		return "scope:groups:"
	}
	sum := sha256.Sum256([]byte(strings.Join(values, ",")))
	return "scope:groups:sha256:" + hex.EncodeToString(sum[:])
}

var _ Repository = (*PgRepository)(nil)
