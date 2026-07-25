package device

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	defer func() { _ = tx.Rollback(ctx) }()

	updatedAt := time.Now()
	for start := 0; start < len(rows); start += deviceParameterUpsertBatchSize {
		end := min(start+deviceParameterUpsertBatchSize, len(rows))
		query, args, err := buildDeviceParameterUpsert(rows[start:end], updatedAt)
		if err != nil {
			return result, fmt.Errorf("build device parameter upsert batch: %w", err)
		}
		returned, err := tx.Query(ctx, query, args...)
		if err != nil {
			return result, fmt.Errorf("execute device parameter upsert batch: %w", err)
		}
		for returned.Next() {
			var deviceID uuid.UUID
			if err := returned.Scan(&deviceID); err != nil {
				returned.Close()
				return result, fmt.Errorf("scan changed device parameter: %w", err)
			}
			result.changed++
			result.changedDevices[deviceID] = struct{}{}
		}
		if err := returned.Err(); err != nil {
			returned.Close()
			return result, fmt.Errorf("iterate changed device parameters: %w", err)
		}
		returned.Close()
	}
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit device parameter upsert: %w", err)
	}
	return result, nil
}
