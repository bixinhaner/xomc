package geofence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/outbox"
	"github.com/omcgo/omcgo/internal/core/storage"
)

func buildVisibleBatchDevicesQuery(
	deviceIDs []uuid.UUID,
	serialNumbers []string,
	visibleGroups []uuid.UUID,
) (string, []any, error) {
	filters := make(sq.Or, 0, 2)
	if len(deviceIDs) > 0 {
		filters = append(filters, sq.Eq{"d.id": deviceIDs})
	}
	if len(serialNumbers) > 0 {
		filters = append(filters, sq.Eq{"d.serial_number": serialNumbers})
	}
	if len(filters) == 0 {
		return "", nil, fmt.Errorf("batch device inputs are empty")
	}

	builder := storage.Psql.
		Select("d.id", "d.serial_number", "d.carrier").
		From("devices d").
		Where("d.deleted_at IS NULL").
		Where(filters).
		OrderBy("d.id")
	query, args, err := authz.ApplyDeviceVisibilityFilter(
		builder,
		"d.id",
		visibleGroups,
	).ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build visible batch devices: %w", err)
	}
	return query, args, nil
}

func buildActiveBindingFactsQuery(
	geofenceID uuid.UUID,
	deviceIDs []uuid.UUID,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select("b.id", "b.device_id", "b.geofence_id", "b.rule_type", "b.status").
		From("device_geofence_bindings b").
		Join(
			"geofence_definitions target ON target.id = ?",
			geofenceID,
		).
		Where(sq.Eq{"b.device_id": deviceIDs}).
		Where(sq.Or{
			sq.And{
				sq.Eq{"b.geofence_id": geofenceID},
				sq.Expr("b.status IN ('active', 'suspended')"),
			},
			sq.And{
				sq.Expr("b.rule_type = target.rule_type"),
				sq.Expr("b.status = 'active'"),
			},
		}).
		OrderBy("b.device_id", "b.id").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build active binding facts: %w", err)
	}
	return query, args, nil
}

func buildInsertManualBindJobQuery(
	jobID uuid.UUID,
	scheduledAt time.Time,
	payload []byte,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Insert("async_jobs").
		Columns(
			"id",
			"job_type",
			"status",
			"scheduled_at",
			"payload",
			"max_attempts",
		).
		Values(
			jobID,
			ManualBindJobType,
			asyncjob.StatusPending,
			scheduledAt,
			json.RawMessage(payload),
			ManualBindJobMaxAttempts,
		).
		Suffix("ON CONFLICT DO NOTHING RETURNING id").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build insert manual bind job: %w", err)
	}
	return query, args, nil
}

func buildListManualBindItemsQuery(
	filter BatchItemFilter,
) (string, []any, error) {
	visible := batchItemVisibleExpression(filter.VisibleGroups)
	builder := storage.Psql.
		Select(
			"bi.id",
			"bi.job_id",
			"bi.geofence_id",
		).
		Column(
			sq.Expr(
				"CASE WHEN ? THEN bi.input_key ELSE '' END AS input_key",
				visible,
			),
		).
		Column("bi.input_kind").
		Column(
			sq.Expr(
				"CASE WHEN ? THEN bi.input_value ELSE '' END AS input_value",
				visible,
			),
		).
		Column(
			sq.Expr(
				"CASE WHEN ? THEN bi.device_id ELSE NULL::uuid END AS device_id",
				visible,
			),
		).
		Column(
			sq.Expr(
				"CASE WHEN ? THEN COALESCE(bi.device_sn_snapshot, '') "+
					"ELSE '' END AS device_sn_snapshot",
				visible,
			),
		).
		Columns(
			"bi.status",
			"COALESCE(bi.reason_code, '')",
		).
		Column(
			sq.Expr(
				"CASE WHEN ? THEN COALESCE(bi.error_message, '') "+
					"ELSE '' END AS error_message",
				visible,
			),
		).
		Column(
			sq.Expr(
				"CASE WHEN ? THEN bi.binding_id ELSE NULL::uuid END AS binding_id",
				visible,
			),
		).
		Columns(
			"bi.attempt",
			"bi.started_at",
			"bi.finished_at",
			"bi.created_at",
			"bi.updated_at",
			"COUNT(*) OVER() AS total_count",
		).
		From("geofence_batch_items bi").
		Where(sq.Eq{"bi.job_id": filter.JobID}).
		OrderBy("bi.created_at", "bi.id").
		Limit(uint64(filter.PageSize)).
		Offset(uint64((filter.Page - 1) * filter.PageSize))
	if filter.Status != "" {
		builder = builder.Where(sq.Eq{"bi.status": filter.Status})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build list manual bind items: %w", err)
	}
	return query, args, nil
}

func batchItemVisibleExpression(visibleGroups []uuid.UUID) sq.Sqlizer {
	if visibleGroups == nil {
		return sq.Expr("TRUE")
	}
	deviceVisible := sq.Select("1").
		From("devices visible_device").
		Where("visible_device.id = bi.device_id")
	deviceVisible = authz.ApplyDeviceVisibilityFilter(
		deviceVisible,
		"visible_device.id",
		visibleGroups,
	)
	return sq.Or{
		sq.Expr("bi.device_id IS NULL"),
		sq.Expr("EXISTS (?)", deviceVisible),
	}
}

func (r *PgRepository) CreateManualBindJob(
	ctx context.Context,
	params CreateManualBindJobParams,
) (BatchJobAccepted, error) {
	if params.GeofenceID == uuid.Nil ||
		params.ActorID == uuid.Nil ||
		strings.TrimSpace(params.PreviewFingerprint) == "" ||
		strings.TrimSpace(params.Reason) == "" ||
		len(params.Inputs) == 0 {
		return BatchJobAccepted{}, fmt.Errorf(
			"manual bind job idempotency inputs are required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if params.ScheduledAt.IsZero() {
		params.ScheduledAt = time.Now().UTC()
	}
	params.PreviewFingerprint = strings.TrimSpace(params.PreviewFingerprint)
	params.Reason = strings.TrimSpace(params.Reason)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return BatchJobAccepted{}, batchDatabaseError(
			"begin create manual bind job",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	confirmationLockSQL, confirmationLockArgs, err :=
		buildManualBindConfirmationLockQuery(params)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	if _, err := tx.Exec(
		ctx,
		confirmationLockSQL,
		confirmationLockArgs...,
	); err != nil {
		return BatchJobAccepted{}, batchDatabaseError(
			"lock manual bind confirmation",
			err,
		)
	}

	existingJobID, found, err := findReusableManualBindJob(
		ctx,
		tx,
		params,
	)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	if found {
		if err := tx.Commit(ctx); err != nil {
			return BatchJobAccepted{}, batchDatabaseError(
				"commit replayed manual bind job",
				err,
			)
		}
		return BatchJobAccepted{JobID: existingJobID}, nil
	}

	definition, err := lockBatchDefinition(ctx, tx, params.GeofenceID)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	if r.hooks != nil && r.hooks.afterManualBindDefinitionLock != nil {
		if err := r.hooks.afterManualBindDefinitionLock(ctx); err != nil {
			return BatchJobAccepted{}, fmt.Errorf(
				"after manual bind definition lock: %w",
				err,
			)
		}
	}
	devices, err := loadVisibleBatchDevices(
		ctx,
		tx,
		params.Inputs,
		definition.Carrier,
		params.VisibleGroups,
	)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	if err := lockBatchDevices(ctx, tx, devices); err != nil {
		return BatchJobAccepted{}, err
	}
	devices, err = loadVisibleBatchDevices(
		ctx,
		tx,
		params.Inputs,
		definition.Carrier,
		params.VisibleGroups,
	)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	facts, err := loadBatchBindingFacts(
		ctx,
		tx,
		definition.ID,
		devices,
	)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	snapshot, err := buildBatchSnapshot(
		*definition,
		params.Inputs,
		devices,
		facts,
		params.VisibleGroups,
	)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	preview, err := buildManualBindingPreview(snapshot)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	if preview.PreviewFingerprint != params.PreviewFingerprint {
		return BatchJobAccepted{}, ErrStaleBindingPreview
	}
	if preview.EligibleCount+preview.MoveCount == 0 {
		return BatchJobAccepted{}, ErrNoEligibleBindingInputs
	}

	payload, err := json.Marshal(ManualBindJobPayload{
		SchemaVersion:      ManualBindPayloadVersion,
		GeofenceID:         definition.ID,
		GeofenceVersionID:  preview.GeofenceVersionID,
		RequestedBy:        params.ActorID,
		PreviewFingerprint: params.PreviewFingerprint,
		Reason:             params.Reason,
	})
	if err != nil {
		return BatchJobAccepted{}, fmt.Errorf(
			"marshal manual bind job payload: %w",
			err,
		)
	}
	jobID := uuid.New()
	insertSQL, insertArgs, err := buildInsertManualBindJobQuery(
		jobID,
		params.ScheduledAt,
		payload,
	)
	if err != nil {
		return BatchJobAccepted{}, err
	}
	err = tx.QueryRow(ctx, insertSQL, insertArgs...).Scan(&jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		if r.hooks != nil && r.hooks.afterManualBindConflict != nil {
			if err := r.hooks.afterManualBindConflict(ctx); err != nil {
				return BatchJobAccepted{}, fmt.Errorf(
					"after manual bind conflict: %w",
					err,
				)
			}
		}
		jobID, err = findExistingManualBindJob(
			ctx,
			tx,
			params,
		)
		if err != nil {
			return BatchJobAccepted{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return BatchJobAccepted{}, batchDatabaseError(
				"commit duplicate manual bind job",
				err,
			)
		}
		return BatchJobAccepted{JobID: jobID}, nil
	}
	if err != nil {
		return BatchJobAccepted{}, batchDatabaseError(
			"insert manual bind job",
			err,
		)
	}
	if err := insertManualBindItems(
		ctx,
		tx,
		jobID,
		definition.ID,
		snapshot,
		preview,
	); err != nil {
		return BatchJobAccepted{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BatchJobAccepted{}, batchDatabaseError(
			"commit manual bind job",
			err,
		)
	}
	return BatchJobAccepted{JobID: jobID}, nil
}

func lockBatchDefinition(
	ctx context.Context,
	tx pgx.Tx,
	geofenceID uuid.UUID,
) (*Definition, error) {
	query, args, err := storage.Psql.
		Select(
			"id", "name", "carrier", "rule_type", "owner_device_id", "status",
			"current_version_id", "created_by", "updated_by", "created_at",
			"updated_at",
		).
		From("geofence_definitions").
		Where(sq.Eq{"id": geofenceID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock manual bind geofence: %w", err)
	}
	var definition Definition
	err = tx.QueryRow(ctx, query, args...).Scan(
		&definition.ID,
		&definition.Name,
		&definition.Carrier,
		&definition.RuleType,
		&definition.OwnerDeviceID,
		&definition.Status,
		&definition.CurrentVersionID,
		&definition.CreatedBy,
		&definition.UpdatedBy,
		&definition.CreatedAt,
		&definition.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, commonerrors.ErrNotFound
	}
	if err != nil {
		return nil, batchDatabaseError("lock manual bind geofence", err)
	}
	return &definition, nil
}

type batchQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadVisibleBatchDevices(
	ctx context.Context,
	querier batchQuerier,
	inputs []BindingInput,
	definitionCarrier string,
	visibleGroups []uuid.UUID,
) (manualBindingDevices, error) {
	deviceIDs, serialNumbers := bindingInputValues(inputs)
	devices := newManualBindingDevices()
	if len(deviceIDs) == 0 && len(serialNumbers) == 0 {
		return devices, nil
	}
	query, args, err := buildVisibleBatchDevicesQuery(
		deviceIDs,
		serialNumbers,
		visibleGroups,
	)
	if err != nil {
		return manualBindingDevices{}, err
	}
	rows, err := querier.Query(ctx, query, args...)
	if err != nil {
		return manualBindingDevices{}, batchDatabaseError(
			"load visible batch devices",
			err,
		)
	}
	defer rows.Close()
	for rows.Next() {
		identity := &DeviceIdentity{}
		if err := rows.Scan(
			&identity.ID,
			&identity.SerialNumber,
			&identity.Carrier,
		); err != nil {
			return manualBindingDevices{}, batchDatabaseError(
				"scan visible batch device",
				err,
			)
		}
		devices.add(identity, definitionCarrier)
	}
	if err := rows.Err(); err != nil {
		return manualBindingDevices{}, batchDatabaseError(
			"iterate visible batch devices",
			err,
		)
	}
	for _, input := range inputs {
		if input.Kind == BindingInputDeviceID &&
			input.DeviceID != nil &&
			devices.byID[*input.DeviceID] == nil {
			return manualBindingDevices{}, commonerrors.ErrForbidden
		}
	}
	return devices, nil
}

func lockBatchDevices(
	ctx context.Context,
	tx pgx.Tx,
	devices manualBindingDevices,
) error {
	deviceIDs := sortedBatchDeviceIDs(devices)
	if len(deviceIDs) == 0 {
		return nil
	}
	query, args, err := storage.Psql.
		Select("id").
		From("devices").
		Where(sq.Eq{"id": deviceIDs}).
		OrderBy("id").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("build lock batch devices: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return batchDatabaseError("lock batch devices", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return batchDatabaseError("scan locked batch device", err)
		}
	}
	if err := rows.Err(); err != nil {
		return batchDatabaseError("iterate locked batch devices", err)
	}
	return nil
}

func sortedBatchDeviceIDs(devices manualBindingDevices) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(devices.byID))
	for id := range devices.byID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].String() < ids[j].String()
	})
	return ids
}

func loadBatchBindingFacts(
	ctx context.Context,
	querier batchQuerier,
	geofenceID uuid.UUID,
	devices manualBindingDevices,
) (manualBindingFacts, error) {
	facts := manualBindingFacts{
		sameGeofenceBindingIDs: make(map[uuid.UUID]uuid.UUID),
		sameGeofenceStatus:     make(map[uuid.UUID]BindingStatus),
		activeRuleConflicts:    make(map[uuid.UUID]manualBindingConflict),
	}
	deviceIDs := sortedBatchDeviceIDs(devices)
	if len(deviceIDs) == 0 {
		return facts, nil
	}
	query, args, err := buildActiveBindingFactsQuery(geofenceID, deviceIDs)
	if err != nil {
		return manualBindingFacts{}, err
	}
	rows, err := querier.Query(ctx, query, args...)
	if err != nil {
		return manualBindingFacts{}, batchDatabaseError(
			"load active binding facts",
			err,
		)
	}
	defer rows.Close()
	for rows.Next() {
		var binding Binding
		if err := rows.Scan(
			&binding.ID,
			&binding.DeviceID,
			&binding.GeofenceID,
			&binding.RuleType,
			&binding.Status,
		); err != nil {
			return manualBindingFacts{}, batchDatabaseError(
				"scan active binding fact",
				err,
			)
		}
		if binding.GeofenceID == geofenceID {
			if binding.Status == BindingStatusActive ||
				facts.sameGeofenceStatus[binding.DeviceID] !=
					BindingStatusActive {
				facts.sameGeofenceBindingIDs[binding.DeviceID] = binding.ID
				facts.sameGeofenceStatus[binding.DeviceID] = binding.Status
			}
		} else if binding.Status == BindingStatusActive {
			facts.activeRuleConflicts[binding.DeviceID] = manualBindingConflict{
				bindingID:  binding.ID,
				geofenceID: binding.GeofenceID,
			}
		}
	}
	if err := rows.Err(); err != nil {
		return manualBindingFacts{}, batchDatabaseError(
			"iterate active binding facts",
			err,
		)
	}
	return facts, nil
}

func buildBatchSnapshot(
	definition Definition,
	inputs []BindingInput,
	devices manualBindingDevices,
	facts manualBindingFacts,
	visibleGroups []uuid.UUID,
) (ManualBindSnapshot, error) {
	snapshot := ManualBindSnapshot{
		Definition:       definition,
		Inputs:           make([]BindingCandidateFact, 0, len(inputs)),
		VisibilityDigest: manualBindingVisibilityDigest(visibleGroups),
	}
	seenDevices := make(map[uuid.UUID]struct{}, len(devices.byID))
	for _, input := range inputs {
		fact := BindingCandidateFact{Input: input}
		switch input.Kind {
		case BindingInputDeviceID:
			if input.DeviceID == nil || devices.byID[*input.DeviceID] == nil {
				return ManualBindSnapshot{}, commonerrors.ErrForbidden
			}
			fact.Device = devices.byID[*input.DeviceID]
		case BindingInputDeviceSN:
			fact.Device = devices.bySerialNumber[input.Value]
		}
		if fact.Device != nil {
			if _, duplicate := seenDevices[fact.Device.ID]; duplicate {
				continue
			}
			seenDevices[fact.Device.ID] = struct{}{}
			if status, ok := facts.sameGeofenceStatus[fact.Device.ID]; ok {
				bindingID := facts.sameGeofenceBindingIDs[fact.Device.ID]
				fact.SameGeofenceBindingID = &bindingID
				statusCopy := status
				fact.SameGeofenceStatus = &statusCopy
			}
			if conflict, ok := facts.activeRuleConflicts[fact.Device.ID]; ok {
				bindingID := conflict.bindingID
				conflictGeofenceID := conflict.geofenceID
				fact.ActiveRuleBindingID = &bindingID
				fact.ActiveRuleGeofenceID = &conflictGeofenceID
			}
		}
		snapshot.Inputs = append(snapshot.Inputs, fact)
	}
	return snapshot, nil
}

func findReusableManualBindJob(
	ctx context.Context,
	tx pgx.Tx,
	params CreateManualBindJobParams,
) (uuid.UUID, bool, error) {
	query, args, err := buildFindReusableManualBindJobQuery(params)
	if err != nil {
		return uuid.Nil, false, err
	}
	var jobID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&jobID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, batchDatabaseError(
			"find reusable manual bind job",
			err,
		)
	}
	return jobID, true, nil
}

func buildFindReusableManualBindJobQuery(
	params CreateManualBindJobParams,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select("id").
		From("async_jobs").
		Where(sq.Eq{
			"job_type": ManualBindJobType,
			"status": []asyncjob.Status{
				asyncjob.StatusPending,
				asyncjob.StatusRunning,
				asyncjob.StatusSucceeded,
			},
		}).
		Where(
			"payload->>'geofence_id' = ?",
			params.GeofenceID.String(),
		).
		Where(
			"payload->>'requested_by' = ?",
			params.ActorID.String(),
		).
		Where(
			"payload->>'preview_fingerprint' = ?",
			params.PreviewFingerprint,
		).
		OrderBy("created_at DESC", "id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf(
			"build find reusable manual bind job: %w",
			err,
		)
	}
	return query, args, nil
}

func findExistingManualBindJob(
	ctx context.Context,
	tx pgx.Tx,
	params CreateManualBindJobParams,
) (uuid.UUID, error) {
	query, args, err := buildFindExistingManualBindJobQuery(params)
	if err != nil {
		return uuid.Nil, err
	}
	var jobID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&jobID); err != nil {
		return uuid.Nil, batchDatabaseError(
			"find existing manual bind job",
			err,
		)
	}
	return jobID, nil
}

func buildFindExistingManualBindJobQuery(
	params CreateManualBindJobParams,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select("id").
		From("async_jobs").
		Where(sq.Eq{"job_type": ManualBindJobType}).
		Where(
			"payload->>'geofence_id' = ?",
			params.GeofenceID.String(),
		).
		Where(
			"payload->>'requested_by' = ?",
			params.ActorID.String(),
		).
		Where(
			"payload->>'preview_fingerprint' = ?",
			params.PreviewFingerprint,
		).
		OrderBy("created_at DESC", "id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build find existing manual bind job: %w", err)
	}
	return query, args, nil
}

func buildManualBindConfirmationLockQuery(
	params CreateManualBindJobParams,
) (string, []any, error) {
	businessKey := params.GeofenceID.String() + "|" +
		params.ActorID.String() + "|" + params.PreviewFingerprint
	query, args, err := storage.Psql.
		Select().
		Column(
			sq.Expr(
				"pg_advisory_xact_lock(hashtextextended(?, 0))",
				businessKey,
			),
		).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf(
			"build manual bind confirmation lock: %w",
			err,
		)
	}
	return query, args, nil
}

func insertManualBindItems(
	ctx context.Context,
	tx pgx.Tx,
	jobID uuid.UUID,
	geofenceID uuid.UUID,
	snapshot ManualBindSnapshot,
	preview ManualBindingPreview,
) error {
	builder := storage.Psql.
		Insert("geofence_batch_items").
		Columns(
			"id", "job_id", "geofence_id", "input_key", "input_kind",
			"input_value", "device_id", "device_sn_snapshot", "status",
			"reason_code", "expected_source_binding_id",
		)
	for index, item := range preview.Items {
		fact := snapshot.Inputs[index]
		status := BatchItemSkipped
		if item.Decision == BindingDecisionEligible ||
			item.Decision == BindingDecisionMove {
			status = BatchItemPending
		}
		var deviceID *uuid.UUID
		deviceSN := ""
		if fact.Input.Kind == BindingInputDeviceSN {
			deviceSN = fact.Input.Value
		}
		if fact.Device != nil {
			idCopy := fact.Device.ID
			deviceID = &idCopy
			deviceSN = fact.Device.SerialNumber
		}
		builder = builder.Values(
			uuid.New(),
			jobID,
			geofenceID,
			fact.Input.Key,
			fact.Input.Kind,
			fact.Input.Value,
			deviceID,
			deviceSN,
			status,
			nullableBatchString(item.ReasonCode),
			item.SourceBindingID,
		)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert manual bind items: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return batchDatabaseError("insert manual bind items", err)
	}
	return nil
}

func nullableBatchString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *PgRepository) GetManualBindJob(
	ctx context.Context,
	jobID uuid.UUID,
) (*BatchJob, error) {
	query, args, err := storage.Psql.
		Select(
			"aj.id",
			"aj.job_type",
			"aj.status",
			"(aj.payload->>'geofence_id')::uuid",
			"(aj.payload->>'requested_by')::uuid",
			"COALESCE(aj.payload->>'reason', '')",
			"aj.attempt",
			"aj.max_attempts",
			"COALESCE(aj.error_message, '')",
			"aj.created_at",
			"aj.started_at",
			"aj.finished_at",
			"COUNT(bi.id)",
			"COUNT(bi.id) FILTER (WHERE bi.status = 'pending')",
			"COUNT(bi.id) FILTER (WHERE bi.status = 'succeeded')",
			"COUNT(bi.id) FILTER (WHERE bi.status = 'skipped')",
			"COUNT(bi.id) FILTER (WHERE bi.status = 'failed')",
		).
		From("async_jobs aj").
		LeftJoin("geofence_batch_items bi ON bi.job_id = aj.id").
		Where(sq.Eq{
			"aj.id":       jobID,
			"aj.job_type": ManualBindJobType,
		}).
		GroupBy("aj.id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get manual bind job: %w", err)
	}
	var job BatchJob
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&job.ID,
		&job.JobType,
		&job.Status,
		&job.GeofenceID,
		&job.RequestedBy,
		&job.Reason,
		&job.Attempt,
		&job.MaxAttempts,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.StartedAt,
		&job.FinishedAt,
		&job.Progress.Total,
		&job.Progress.Pending,
		&job.Progress.Succeeded,
		&job.Progress.Skipped,
		&job.Progress.Failed,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, batchDatabaseError("get manual bind job", err)
	}
	return &job, nil
}

func (r *PgRepository) ListManualBindItems(
	ctx context.Context,
	filter BatchItemFilter,
) (BatchItemPage, error) {
	if filter.JobID == uuid.Nil ||
		filter.PageSize > MaxBatchItemPageSize ||
		(filter.Status != "" && !validBatchItemStatus(filter.Status)) {
		return BatchItemPage{}, fmt.Errorf(
			"invalid manual bind item filter: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = DefaultBatchItemPageSize
	}
	countBuilder := storage.Psql.
		Select("COUNT(*)").
		From("geofence_batch_items").
		Where(sq.Eq{"job_id": filter.JobID})
	if filter.Status != "" {
		countBuilder = countBuilder.Where(sq.Eq{"status": filter.Status})
	}
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return BatchItemPage{}, fmt.Errorf(
			"build count manual bind items: %w",
			err,
		)
	}
	page := BatchItemPage{
		Items:    make([]BatchItem, 0, filter.PageSize),
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if err := r.pool.QueryRow(
		ctx,
		countSQL,
		countArgs...,
	).Scan(&page.Total); err != nil {
		return BatchItemPage{}, batchDatabaseError(
			"count manual bind items",
			err,
		)
	}
	query, args, err := buildListManualBindItemsQuery(filter)
	if err != nil {
		return BatchItemPage{}, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return BatchItemPage{}, batchDatabaseError(
			"list manual bind items",
			err,
		)
	}
	defer rows.Close()
	for rows.Next() {
		var item BatchItem
		var total int64
		if err := rows.Scan(
			&item.ID,
			&item.JobID,
			&item.GeofenceID,
			&item.InputKey,
			&item.InputKind,
			&item.InputValue,
			&item.DeviceID,
			&item.DeviceSNSnapshot,
			&item.Status,
			&item.ReasonCode,
			&item.ErrorMessage,
			&item.BindingID,
			&item.Attempt,
			&item.StartedAt,
			&item.FinishedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&total,
		); err != nil {
			return BatchItemPage{}, batchDatabaseError(
				"scan manual bind item",
				err,
			)
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return BatchItemPage{}, batchDatabaseError(
			"iterate manual bind items",
			err,
		)
	}
	return page, nil
}

func buildListPendingManualBindItemIDsQuery(
	jobID uuid.UUID,
	jobAttempt int,
	limit int,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select("id").
		From("geofence_batch_items").
		Where(sq.Eq{
			"job_id": jobID,
			"status": BatchItemPending,
		}).
		Where(sq.Expr("attempt < ?", jobAttempt)).
		OrderBy("device_id", "id").
		Suffix("LIMIT ?", limit).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf(
			"build list pending manual bind item IDs: %w",
			err,
		)
	}
	return query, args, nil
}

func (r *PgRepository) ListPendingManualBindItemIDs(
	ctx context.Context,
	jobID uuid.UUID,
	jobAttempt int,
	limit int,
) ([]uuid.UUID, error) {
	if jobID == uuid.Nil || jobAttempt <= 0 || limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf(
			"invalid pending manual bind item query: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	query, args, err := buildListPendingManualBindItemIDsQuery(
		jobID,
		jobAttempt,
		limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, batchDatabaseError(
			"list pending manual bind item IDs",
			err,
		)
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, batchDatabaseError(
				"scan pending manual bind item ID",
				err,
			)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, batchDatabaseError(
			"iterate pending manual bind item IDs",
			err,
		)
	}
	return ids, nil
}

type lockedManualBindItem struct {
	DeviceID                *uuid.UUID
	Status                  BatchItemStatus
	ReasonCode              string
	ExpectedSourceBindingID *uuid.UUID
	BindingID               *uuid.UUID
	Attempt                 int
}

func (r *PgRepository) ProcessManualBindItem(
	ctx context.Context,
	req ProcessManualBindItemRequest,
) (BatchItemResult, error) {
	if req.JobID == uuid.Nil ||
		req.ItemID == uuid.Nil ||
		req.GeofenceID == uuid.Nil ||
		req.ExpectedGeofenceVersionID == uuid.Nil ||
		req.ActorID == uuid.Nil ||
		req.JobAttempt <= 0 ||
		req.MaxAttempts <= 0 ||
		req.JobAttempt > req.MaxAttempts {
		return BatchItemResult{}, fmt.Errorf(
			"invalid manual bind item request: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if req.ProcessedAt.IsZero() {
		req.ProcessedAt = time.Now().UTC()
	}

	result, err := r.processManualBindItemTx(ctx, req)
	if err == nil {
		return result, nil
	}
	failureResult, markErr := r.markManualBindItemFailure(ctx, req)
	if markErr != nil {
		return BatchItemResult{}, errors.Join(
			fmt.Errorf("process manual bind item transaction: %w", err),
			fmt.Errorf("record manual bind item attempt: %w", markErr),
		)
	}
	return failureResult, fmt.Errorf(
		"process manual bind item transaction: %w",
		err,
	)
}

func (r *PgRepository) processManualBindItemTx(
	ctx context.Context,
	req ProcessManualBindItemRequest,
) (BatchItemResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return BatchItemResult{}, batchDatabaseError(
			"begin process manual bind item",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	definition, err := lockManualBindDefinition(ctx, tx, req.GeofenceID)
	if err != nil {
		return BatchItemResult{}, err
	}
	item, err := lockManualBindItem(ctx, tx, req)
	if err != nil {
		return BatchItemResult{}, err
	}
	if item.Status != BatchItemPending {
		return BatchItemResult{
			Status:     item.Status,
			ReasonCode: item.ReasonCode,
			BindingID:  item.BindingID,
		}, nil
	}
	if item.Attempt >= req.JobAttempt {
		return BatchItemResult{Status: BatchItemPending}, nil
	}
	if r.hooks != nil && r.hooks.afterManualBindItemLock != nil {
		if err := r.hooks.afterManualBindItemLock(ctx); err != nil {
			return BatchItemResult{}, fmt.Errorf(
				"after manual bind item lock: %w",
				err,
			)
		}
	}

	if definition.Status != DefinitionStatusEnabled ||
		definition.CurrentVersionID == nil {
		return finishSkippedManualBindItem(
			ctx,
			tx,
			req,
			ReasonGeofenceNotEnabled,
		)
	}
	if *definition.CurrentVersionID != req.ExpectedGeofenceVersionID {
		return finishSkippedManualBindItem(
			ctx,
			tx,
			req,
			ReasonGeofenceVersionChanged,
		)
	}
	if item.DeviceID == nil {
		return finishSkippedManualBindItem(
			ctx,
			tx,
			req,
			ReasonDeviceUnavailable,
		)
	}

	device, ok, err := lockManualBindDevice(ctx, tx, *item.DeviceID)
	if err != nil {
		return BatchItemResult{}, err
	}
	if !ok {
		return finishSkippedManualBindItem(
			ctx,
			tx,
			req,
			ReasonDeviceUnavailable,
		)
	}
	bindings, err := lockManualBindDeviceBindings(ctx, tx, device.ID)
	if err != nil {
		return BatchItemResult{}, err
	}
	reasonCode, sourceBinding := revalidateManualBinding(
		*definition,
		device,
		bindings,
	)
	if reasonCode != "" {
		return finishSkippedManualBindItem(ctx, tx, req, reasonCode)
	}
	if item.ExpectedSourceBindingID == nil && sourceBinding != nil {
		return finishSkippedManualBindItem(
			ctx,
			tx,
			req,
			ReasonActiveRuleConflict,
		)
	}
	if item.ExpectedSourceBindingID != nil &&
		(sourceBinding == nil || sourceBinding.ID != *item.ExpectedSourceBindingID) {
		return finishSkippedManualBindItem(
			ctx,
			tx,
			req,
			ReasonBindingChanged,
		)
	}
	initialConfirmedState := ConfirmedStateUnknown
	if sourceBinding != nil {
		if _, found, lockErr := lockEffectiveState(
			ctx,
			tx,
			device.ID,
		); lockErr != nil {
			return BatchItemResult{}, fmt.Errorf(
				"lock effective state before geofence reassignment: %w",
				lockErr,
			)
		} else if !found {
			return BatchItemResult{}, fmt.Errorf(
				"effective state for reassigned geofence binding is missing: %w",
				commonerrors.ErrInvalidInput,
			)
		}
		initialConfirmedState, err = lockManualBindSourceConfirmedState(
			ctx,
			tx,
			sourceBinding.ID,
		)
		if err != nil {
			return BatchItemResult{}, err
		}
		if err := removeManualBindSourceBinding(
			ctx,
			tx,
			*sourceBinding,
			req.ActorID,
			req.ProcessedAt,
		); err != nil {
			return BatchItemResult{}, err
		}
	}

	bindingID := uuid.New()
	binding := &Binding{
		ID:         bindingID,
		DeviceID:   device.ID,
		GeofenceID: definition.ID,
		RuleType:   definition.RuleType,
		Status:     BindingStatusActive,
		BindSource: "manual",
		BoundBy:    req.ActorID,
		BoundAt:    req.ProcessedAt,
	}
	if err := createBindingTxWithConfirmedState(
		ctx,
		tx,
		binding,
		initialConfirmedState,
	); err != nil {
		return BatchItemResult{}, err
	}
	if err := enqueueManualBindLatestLocationReplay(
		ctx,
		tx,
		bindingID,
		device.ID,
	); err != nil {
		return BatchItemResult{}, err
	}
	if err := finishSucceededManualBindItem(
		ctx,
		tx,
		req,
		bindingID,
	); err != nil {
		return BatchItemResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BatchItemResult{}, batchDatabaseError(
			"commit process manual bind item",
			err,
		)
	}
	return BatchItemResult{
		Status:    BatchItemSucceeded,
		BindingID: &bindingID,
	}, nil
}

func lockManualBindDefinition(
	ctx context.Context,
	tx pgx.Tx,
	geofenceID uuid.UUID,
) (*Definition, error) {
	query, args, err := storage.Psql.
		Select(
			"id", "carrier", "rule_type", "owner_device_id", "status",
			"current_version_id",
		).
		From("geofence_definitions").
		Where(sq.Eq{"id": geofenceID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock manual bind definition: %w", err)
	}
	var definition Definition
	if err := tx.QueryRow(ctx, query, args...).Scan(
		&definition.ID,
		&definition.Carrier,
		&definition.RuleType,
		&definition.OwnerDeviceID,
		&definition.Status,
		&definition.CurrentVersionID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, batchDatabaseError("lock manual bind definition", err)
	}
	return &definition, nil
}

func lockManualBindItem(
	ctx context.Context,
	tx pgx.Tx,
	req ProcessManualBindItemRequest,
) (lockedManualBindItem, error) {
	query, args, err := storage.Psql.
		Select(
			"device_id", "status", "COALESCE(reason_code, '')",
			"expected_source_binding_id", "binding_id", "attempt",
		).
		From("geofence_batch_items").
		Where(sq.Eq{
			"id":          req.ItemID,
			"job_id":      req.JobID,
			"geofence_id": req.GeofenceID,
		}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return lockedManualBindItem{}, fmt.Errorf(
			"build lock manual bind item: %w",
			err,
		)
	}
	var item lockedManualBindItem
	if err := tx.QueryRow(ctx, query, args...).Scan(
		&item.DeviceID,
		&item.Status,
		&item.ReasonCode,
		&item.ExpectedSourceBindingID,
		&item.BindingID,
		&item.Attempt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return lockedManualBindItem{}, commonerrors.ErrNotFound
		}
		return lockedManualBindItem{}, batchDatabaseError(
			"lock manual bind item",
			err,
		)
	}
	return item, nil
}

func lockManualBindDevice(
	ctx context.Context,
	tx pgx.Tx,
	deviceID uuid.UUID,
) (DeviceIdentity, bool, error) {
	query, args, err := storage.Psql.
		Select("id", "serial_number", "carrier", "deleted_at").
		From("devices").
		Where(sq.Eq{"id": deviceID}).
		OrderBy("id").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return DeviceIdentity{}, false, fmt.Errorf(
			"build lock manual bind device: %w",
			err,
		)
	}
	var (
		device    DeviceIdentity
		deletedAt *time.Time
	)
	if err := tx.QueryRow(ctx, query, args...).Scan(
		&device.ID,
		&device.SerialNumber,
		&device.Carrier,
		&deletedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DeviceIdentity{}, false, nil
		}
		return DeviceIdentity{}, false, batchDatabaseError(
			"lock manual bind device",
			err,
		)
	}
	return device, deletedAt == nil, nil
}

func lockManualBindDeviceBindings(
	ctx context.Context,
	tx pgx.Tx,
	deviceID uuid.UUID,
) ([]Binding, error) {
	query, args, err := storage.Psql.
		Select("id", "geofence_id", "rule_type", "status").
		From("device_geofence_bindings").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.Eq{"status": []BindingStatus{
			BindingStatusActive,
			BindingStatusSuspended,
		}}).
		OrderBy("id").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf(
			"build lock manual bind device bindings: %w",
			err,
		)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, batchDatabaseError(
			"lock manual bind device bindings",
			err,
		)
	}
	defer rows.Close()
	bindings := make([]Binding, 0)
	for rows.Next() {
		var binding Binding
		binding.DeviceID = deviceID
		if err := rows.Scan(
			&binding.ID,
			&binding.GeofenceID,
			&binding.RuleType,
			&binding.Status,
		); err != nil {
			return nil, batchDatabaseError(
				"scan manual bind device binding",
				err,
			)
		}
		bindings = append(bindings, binding)
	}
	if err := rows.Err(); err != nil {
		return nil, batchDatabaseError(
			"iterate manual bind device bindings",
			err,
		)
	}
	return bindings, nil
}

func revalidateManualBinding(
	definition Definition,
	device DeviceIdentity,
	bindings []Binding,
) (string, *Binding) {
	if device.Carrier != definition.Carrier {
		return ReasonCarrierMismatch, nil
	}
	if definition.RuleType == RuleTypeBaselineRadius &&
		(definition.OwnerDeviceID == nil ||
			*definition.OwnerDeviceID != device.ID) {
		return ReasonBaselineOwnerMismatch, nil
	}
	for _, binding := range bindings {
		if binding.GeofenceID == definition.ID {
			if binding.Status == BindingStatusSuspended {
				return ReasonBindingSuspended, nil
			}
			if binding.Status == BindingStatusActive {
				return ReasonAlreadyBound, nil
			}
		}
		if binding.RuleType == definition.RuleType &&
			binding.Status == BindingStatusActive {
			bindingCopy := binding
			return "", &bindingCopy
		}
	}
	return "", nil
}

func removeManualBindSourceBinding(
	ctx context.Context,
	tx pgx.Tx,
	binding Binding,
	actorID uuid.UUID,
	removedAt time.Time,
) error {
	query, args, err := storage.Psql.
		Update("device_geofence_bindings").
		Set("status", BindingStatusRemoved).
		Set("removed_by", actorID).
		Set("removed_at", removedAt).
		Set("remove_reason", ReasonReassigned).
		Where(sq.Eq{
			"id":     binding.ID,
			"status": BindingStatusActive,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build remove reassigned geofence binding: %w", err)
	}
	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return batchDatabaseError("remove reassigned geofence binding", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("remove reassigned geofence binding: %w", commonerrors.ErrAlreadyExists)
	}
	return nil
}

func lockManualBindSourceConfirmedState(
	ctx context.Context,
	tx pgx.Tx,
	bindingID uuid.UUID,
) (ConfirmedState, error) {
	query, args, err := storage.Psql.
		Select("confirmed_state").
		From("device_geofence_states").
		Where(sq.Eq{"binding_id": bindingID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return ConfirmedStateUnknown, fmt.Errorf(
			"build lock reassigned geofence binding state: %w",
			err,
		)
	}
	var confirmedState ConfirmedState
	if err := tx.QueryRow(ctx, query, args...).Scan(&confirmedState); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ConfirmedStateUnknown, fmt.Errorf(
				"reassigned geofence binding state is missing: %w",
				commonerrors.ErrInvalidInput,
			)
		}
		return ConfirmedStateUnknown, batchDatabaseError(
			"lock reassigned geofence binding state",
			err,
		)
	}
	switch confirmedState {
	case ConfirmedStateUnknown, ConfirmedStateInside, ConfirmedStateOutside:
		return confirmedState, nil
	default:
		return ConfirmedStateUnknown, fmt.Errorf(
			"reassigned geofence binding state %q is invalid: %w",
			confirmedState,
			commonerrors.ErrInvalidInput,
		)
	}
}

func finishSkippedManualBindItem(
	ctx context.Context,
	tx pgx.Tx,
	req ProcessManualBindItemRequest,
	reasonCode string,
) (BatchItemResult, error) {
	query, args, err := storage.Psql.
		Update("geofence_batch_items").
		Set("status", BatchItemSkipped).
		Set("reason_code", reasonCode).
		Set("error_message", nil).
		Set("attempt", req.JobAttempt).
		Set("started_at", sq.Expr("COALESCE(started_at, ?)", req.ProcessedAt)).
		Set("finished_at", req.ProcessedAt).
		Set("updated_at", req.ProcessedAt).
		Where(sq.Eq{"id": req.ItemID}).
		ToSql()
	if err != nil {
		return BatchItemResult{}, fmt.Errorf(
			"build skip manual bind item: %w",
			err,
		)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return BatchItemResult{}, batchDatabaseError(
			"skip manual bind item",
			err,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		return BatchItemResult{}, batchDatabaseError(
			"commit skipped manual bind item",
			err,
		)
	}
	return BatchItemResult{
		Status:     BatchItemSkipped,
		ReasonCode: reasonCode,
	}, nil
}

func finishSucceededManualBindItem(
	ctx context.Context,
	tx pgx.Tx,
	req ProcessManualBindItemRequest,
	bindingID uuid.UUID,
) error {
	query, args, err := storage.Psql.
		Update("geofence_batch_items").
		Set("status", BatchItemSucceeded).
		Set("reason_code", nil).
		Set("error_message", nil).
		Set("binding_id", bindingID).
		Set("attempt", req.JobAttempt).
		Set("started_at", sq.Expr("COALESCE(started_at, ?)", req.ProcessedAt)).
		Set("finished_at", req.ProcessedAt).
		Set("updated_at", req.ProcessedAt).
		Where(sq.Eq{"id": req.ItemID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build succeed manual bind item: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return batchDatabaseError("succeed manual bind item", err)
	}
	return nil
}

func enqueueManualBindLatestLocationReplay(
	ctx context.Context,
	tx pgx.Tx,
	bindingID uuid.UUID,
	deviceID uuid.UUID,
) error {
	versionQuery, versionArgs, err := storage.Psql.
		Select("version").
		From("device_location_observations").
		Where(sq.Eq{"device_id": deviceID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build load latest location version: %w", err)
	}
	var observationVersion int64
	if err := tx.QueryRow(ctx, versionQuery, versionArgs...).Scan(
		&observationVersion,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return batchDatabaseError("load latest location version", err)
	}

	originalKey := fmt.Sprintf(
		"%s:%s:%d",
		event.SubjectDeviceLocationObserved,
		deviceID,
		observationVersion,
	)
	payloadQuery, payloadArgs, err := storage.Psql.
		Select("payload").
		From("event_outbox").
		Where(sq.Eq{
			"dedupe_key": originalKey,
			"subject":    event.SubjectDeviceLocationObserved,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf(
			"build load original location observed payload: %w",
			err,
		)
	}
	var rawPayload json.RawMessage
	if err := tx.QueryRow(ctx, payloadQuery, payloadArgs...).Scan(
		&rawPayload,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return batchDatabaseError(
			"load original location observed payload",
			err,
		)
	}
	var payload event.DeviceLocationObservedPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return nil
	}
	if payload.DeviceID != deviceID ||
		payload.ObservationVersion != observationVersion {
		return nil
	}
	replayKey := fmt.Sprintf(
		"geofence.binding.evaluate:%s:%d",
		bindingID,
		observationVersion,
	)
	if err := outbox.NewRepository().InsertTx(ctx, tx, outbox.Record{
		AggregateType: "geofence_binding",
		AggregateID:   bindingID.String(),
		Subject:       event.SubjectDeviceLocationObserved,
		Payload:       rawPayload,
		DedupeKey:     replayKey,
	}); err != nil {
		return fmt.Errorf("enqueue manual bind location replay: %w", err)
	}
	return nil
}

func (r *PgRepository) markManualBindItemFailure(
	ctx context.Context,
	req ProcessManualBindItemRequest,
) (BatchItemResult, error) {
	status := BatchItemPending
	var finishedAt any
	if req.JobAttempt >= req.MaxAttempts {
		status = BatchItemFailed
		finishedAt = req.ProcessedAt
	}
	query, args, err := storage.Psql.
		Update("geofence_batch_items").
		Set("status", status).
		Set("reason_code", ReasonProcessingError).
		Set("error_message", manualBindPublicErrorMessage()).
		Set("attempt", req.JobAttempt).
		Set("started_at", sq.Expr("COALESCE(started_at, ?)", req.ProcessedAt)).
		Set("finished_at", finishedAt).
		Set("updated_at", req.ProcessedAt).
		Where(sq.Eq{
			"id":          req.ItemID,
			"job_id":      req.JobID,
			"geofence_id": req.GeofenceID,
			"status":      BatchItemPending,
		}).
		Where(sq.Expr("attempt < ?", req.JobAttempt)).
		Suffix("RETURNING status").
		ToSql()
	if err != nil {
		return BatchItemResult{}, fmt.Errorf(
			"build record manual bind item failure: %w",
			err,
		)
	}
	var savedStatus BatchItemStatus
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&savedStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.loadManualBindItemResult(ctx, req)
		}
		return BatchItemResult{}, batchDatabaseError(
			"record manual bind item failure",
			err,
		)
	}
	return BatchItemResult{
		Status:     savedStatus,
		ReasonCode: ReasonProcessingError,
	}, nil
}

func (r *PgRepository) loadManualBindItemResult(
	ctx context.Context,
	req ProcessManualBindItemRequest,
) (BatchItemResult, error) {
	query, args, err := storage.Psql.
		Select("status", "COALESCE(reason_code, '')", "binding_id").
		From("geofence_batch_items").
		Where(sq.Eq{
			"id":     req.ItemID,
			"job_id": req.JobID,
		}).
		ToSql()
	if err != nil {
		return BatchItemResult{}, fmt.Errorf(
			"build load manual bind item result: %w",
			err,
		)
	}
	var result BatchItemResult
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&result.Status,
		&result.ReasonCode,
		&result.BindingID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BatchItemResult{}, commonerrors.ErrNotFound
		}
		return BatchItemResult{}, batchDatabaseError(
			"load manual bind item result",
			err,
		)
	}
	return result, nil
}

func manualBindPublicErrorMessage() string {
	return commonerrors.ErrInternal.Error()
}

func (r *PgRepository) GetBatchProgress(
	ctx context.Context,
	jobID uuid.UUID,
) (BatchProgress, error) {
	if jobID == uuid.Nil {
		return BatchProgress{}, fmt.Errorf(
			"manual bind job is required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	query, args, err := storage.Psql.
		Select(
			"COUNT(*)",
			"COUNT(*) FILTER (WHERE status = 'pending')",
			"COUNT(*) FILTER (WHERE status = 'succeeded')",
			"COUNT(*) FILTER (WHERE status = 'skipped')",
			"COUNT(*) FILTER (WHERE status = 'failed')",
		).
		From("geofence_batch_items").
		Where(sq.Eq{"job_id": jobID}).
		ToSql()
	if err != nil {
		return BatchProgress{}, fmt.Errorf(
			"build get manual bind progress: %w",
			err,
		)
	}
	var progress BatchProgress
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&progress.Total,
		&progress.Pending,
		&progress.Succeeded,
		&progress.Skipped,
		&progress.Failed,
	); err != nil {
		return BatchProgress{}, batchDatabaseError(
			"get manual bind progress",
			err,
		)
	}
	return progress, nil
}

func batchDatabaseError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("%s: %w", operation, commonerrors.ErrInternal)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
