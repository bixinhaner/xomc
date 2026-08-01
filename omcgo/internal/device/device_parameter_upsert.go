package device

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

const deviceParameterUpsertBatchSize = 500

const deviceParameterUnchangedPredicate = `(device_parameters.parameter_value IS DISTINCT FROM EXCLUDED.parameter_value
	OR device_parameters.parameter_type IS DISTINCT FROM EXCLUDED.parameter_type
	OR device_parameters.writable IS DISTINCT FROM EXCLUDED.writable
	OR device_parameters.fap_instance IS DISTINCT FROM EXCLUDED.fap_instance
	OR device_parameters.param_group IS DISTINCT FROM EXCLUDED.param_group)`

type deviceParameterUpsertRow struct {
	deviceID  uuid.UUID
	parameter model.DeviceParameter
}

type deviceParameterUpsertResult struct {
	attempted      int
	changed        int
	changedDevices map[uuid.UUID]struct{}
}

type deviceParameterKey struct {
	deviceID uuid.UUID
	path     string
}

func orderedDeviceParameterWriteIDs(rows []deviceParameterUpsertRow) []uuid.UUID {
	unique := make(map[uuid.UUID]struct{}, len(rows))
	for _, row := range rows {
		if row.deviceID != uuid.Nil {
			unique[row.deviceID] = struct{}{}
		}
	}
	ids := make([]uuid.UUID, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].String() < ids[j].String()
	})
	return ids
}

// AcquireParameterWriteLocks serializes every official device_parameters
// mutation for the same device. A full-sync finalize both upserts and deletes
// rows, while Inform projection concurrently upserts them; without one shared
// lock the two statements can lock parameter rows in opposite order and
// deadlock. Multiple devices are always acquired in UUID order so batch Inform
// transactions cannot create a lock-order cycle with each other.
func AcquireParameterWriteLocks(ctx context.Context, tx pgx.Tx, deviceIDs ...uuid.UUID) error {
	uniqueRows := make([]deviceParameterUpsertRow, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		uniqueRows = append(uniqueRows, deviceParameterUpsertRow{deviceID: deviceID})
	}
	for _, deviceID := range orderedDeviceParameterWriteIDs(uniqueRows) {
		if _, err := tx.Exec(ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			"device_parameters:"+deviceID.String(),
		); err != nil {
			return fmt.Errorf("acquire device parameter write lock for %s: %w", deviceID, err)
		}
	}
	return nil
}

func tryAcquireParameterWriteLocks(
	ctx context.Context,
	tx pgx.Tx,
	deviceIDs []uuid.UUID,
) (map[uuid.UUID]struct{}, error) {
	acquired := make(map[uuid.UUID]struct{}, len(deviceIDs))
	if len(deviceIDs) == 0 {
		return acquired, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT device_id,
		       pg_try_advisory_xact_lock(hashtextextended('device_parameters:' || device_id::text, 0))
		FROM unnest($1::uuid[]) AS locked_devices(device_id)
	`, deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("try device parameter write locks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var deviceID uuid.UUID
		var locked bool
		if err := rows.Scan(&deviceID, &locked); err != nil {
			return nil, fmt.Errorf("scan device parameter try-lock: %w", err)
		}
		if locked {
			acquired[deviceID] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device parameter try-locks: %w", err)
	}
	return acquired, nil
}

func partitionDeviceParameterRows(
	rows []deviceParameterUpsertRow,
	acquired map[uuid.UUID]struct{},
) ([]deviceParameterUpsertRow, map[uuid.UUID][]deviceParameterUpsertRow) {
	ready := make([]deviceParameterUpsertRow, 0, len(rows))
	deferred := make(map[uuid.UUID][]deviceParameterUpsertRow)
	for _, row := range rows {
		if _, ok := acquired[row.deviceID]; ok {
			ready = append(ready, row)
			continue
		}
		deferred[row.deviceID] = append(deferred[row.deviceID], row)
	}
	return ready, deferred
}

func dedupeDeviceParameterRows(rows []deviceParameterUpsertRow) []deviceParameterUpsertRow {
	if len(rows) < 2 {
		return rows
	}
	indexByKey := make(map[deviceParameterKey]int, len(rows))
	deduped := make([]deviceParameterUpsertRow, 0, len(rows))
	for _, row := range rows {
		key := deviceParameterKey{deviceID: row.deviceID, path: row.parameter.ParameterPath}
		if index, ok := indexByKey[key]; ok {
			deduped[index] = row
			continue
		}
		indexByKey[key] = len(deduped)
		deduped = append(deduped, row)
	}
	return deduped
}

func buildDeviceParameterUpsert(
	rows []deviceParameterUpsertRow,
	updatedAt time.Time,
) (string, []any, error) {
	builder := storage.Psql.Insert("device_parameters").
		Columns(
			"device_id", "parameter_path", "parameter_value", "parameter_type",
			"writable", "last_updated_at", "fap_instance", "param_group",
		)
	for _, row := range rows {
		parameter := row.parameter
		fapInstance := parameter.FAPInstance
		if fapInstance == 0 {
			fapInstance = ExtractFAPInstance(parameter.ParameterPath)
		}
		paramGroup := parameter.ParamGroup
		if paramGroup == "" {
			paramGroup = ClassifyParamGroup(parameter.ParameterPath)
		}
		builder = builder.Values(
			row.deviceID, parameter.ParameterPath, parameter.ParameterValue,
			parameter.ParameterType, parameter.Writable, updatedAt,
			fapInstance, paramGroup,
		)
	}
	return builder.Suffix(
		`ON CONFLICT (device_id, parameter_path) DO UPDATE SET
			parameter_value = EXCLUDED.parameter_value,
			parameter_type = EXCLUDED.parameter_type,
			writable = EXCLUDED.writable,
			last_updated_at = EXCLUDED.last_updated_at,
			fap_instance = EXCLUDED.fap_instance,
			param_group = EXCLUDED.param_group
		WHERE ` + deviceParameterUnchangedPredicate + `
		RETURNING device_id`,
	).ToSql()
}

func upsertDeviceParameterRows(
	ctx context.Context,
	tx pgx.Tx,
	rows []deviceParameterUpsertRow,
	updatedAt time.Time,
	result *deviceParameterUpsertResult,
) error {
	for start := 0; start < len(rows); start += deviceParameterUpsertBatchSize {
		end := min(start+deviceParameterUpsertBatchSize, len(rows))
		query, args, err := buildDeviceParameterUpsert(rows[start:end], updatedAt)
		if err != nil {
			return fmt.Errorf("build device parameter upsert batch: %w", err)
		}
		returned, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("execute device parameter upsert batch: %w", err)
		}
		for returned.Next() {
			var deviceID uuid.UUID
			if err := returned.Scan(&deviceID); err != nil {
				returned.Close()
				return fmt.Errorf("scan changed device parameter: %w", err)
			}
			result.changed++
			result.changedDevices[deviceID] = struct{}{}
		}
		if err := returned.Err(); err != nil {
			returned.Close()
			return fmt.Errorf("iterate changed device parameters: %w", err)
		}
		returned.Close()
	}
	return nil
}

func upsertContendedDeviceParameterRows(
	ctx context.Context,
	pool *pgxpool.Pool,
	deviceID uuid.UUID,
	rows []deviceParameterUpsertRow,
	updatedAt time.Time,
	result *deviceParameterUpsertResult,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin contended device parameter upsert for %s: %w", deviceID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := AcquireParameterWriteLocks(ctx, tx, deviceID); err != nil {
		return err
	}
	if err := upsertDeviceParameterRows(ctx, tx, rows, updatedAt, result); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit contended device parameter upsert for %s: %w", deviceID, err)
	}
	return nil
}

func bulkUpsertDeviceParameters(
	ctx context.Context,
	pool *pgxpool.Pool,
	input []deviceParameterUpsertRow,
) (deviceParameterUpsertResult, error) {
	rows := dedupeDeviceParameterRows(input)
	result := deviceParameterUpsertResult{
		attempted:      len(rows),
		changedDevices: make(map[uuid.UUID]struct{}),
	}
	if len(rows) == 0 {
		return result, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin device parameter upsert: %w", err)
	}
	updatedAt := time.Now()
	defer func() { _ = tx.Rollback(ctx) }()
	acquired, err := tryAcquireParameterWriteLocks(ctx, tx, orderedDeviceParameterWriteIDs(rows))
	if err != nil {
		return result, err
	}
	ready, deferred := partitionDeviceParameterRows(rows, acquired)
	if err := upsertDeviceParameterRows(ctx, tx, ready, updatedAt, &result); err != nil {
		return result, err
	}
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit device parameter upsert: %w", err)
	}

	// A busy device must not hold back unrelated devices from the same Inform
	// batch. The uncontended rows are committed above; each contended device is
	// then serialized and written in its own short transaction.
	deferredIDs := make([]uuid.UUID, 0, len(deferred))
	for deviceID := range deferred {
		deferredIDs = append(deferredIDs, deviceID)
	}
	sort.Slice(deferredIDs, func(i, j int) bool {
		return deferredIDs[i].String() < deferredIDs[j].String()
	})
	for _, deviceID := range deferredIDs {
		if err := upsertContendedDeviceParameterRows(
			ctx, pool, deviceID, deferred[deviceID], updatedAt, &result,
		); err != nil {
			return result, err
		}
	}
	return result, nil
}
