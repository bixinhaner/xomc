package software

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

var upgradeColumns = []string{
	"id", "device_id", "firmware_id", "batch_id", "status",
	"error_message", "retry_count", "max_retries",
	"started_at", "completed_at", "created_at", "updated_at",
}

var _ UpgradeTaskRepository = (*PgUpgradeTaskRepository)(nil)

// PgUpgradeTaskRepository is a PostgreSQL implementation of UpgradeTaskRepository.
type PgUpgradeTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgUpgradeTaskRepository creates a new PgUpgradeTaskRepository.
func NewPgUpgradeTaskRepository(pool *pgxpool.Pool) *PgUpgradeTaskRepository {
	return &PgUpgradeTaskRepository{pool: pool}
}

func scanUpgradeTask(row pgx.Row) (*UpgradeTask, error) {
	var task UpgradeTask
	var batchID sql.NullString
	var errorMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := row.Scan(
		&task.ID, &task.DeviceID, &task.FirmwareID, &batchID, &task.Status,
		&errorMsg, &task.RetryCount, &task.MaxRetries,
		&startedAt, &completedAt, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if batchID.Valid {
		bid, _ := uuid.Parse(batchID.String)
		task.BatchID = &bid
	}
	if errorMsg.Valid {
		task.ErrorMessage = errorMsg.String
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	return &task, nil
}

func (r *PgUpgradeTaskRepository) Create(ctx context.Context, task *UpgradeTask) error {
	builder := psql.Insert("upgrade_tasks").
		Columns("device_id", "firmware_id", "batch_id", "status", "max_retries")

	var batchID interface{}
	if task.BatchID != nil {
		batchID = *task.BatchID
	}

	builder = builder.Values(task.DeviceID, task.FirmwareID, batchID, task.Status, task.MaxRetries).
		Suffix("RETURNING " + joinColumns(upgradeColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert upgrade task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanUpgradeTask(row)
	if err != nil {
		return fmt.Errorf("create upgrade task: %w", err)
	}
	*task = *created
	return nil
}

func (r *PgUpgradeTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error) {
	query, args, err := psql.Select(upgradeColumns...).
		From("upgrade_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get upgrade task SQL: %w", err)
	}

	task, err := scanUpgradeTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get upgrade task: %w", err)
	}
	return task, nil
}

func (r *PgUpgradeTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error {
	builder := psql.Update("upgrade_tasks").
		Set("status", status).
		Where(sq.Eq{"id": id})

	if errorMsg != "" {
		builder = builder.Set("error_message", errorMsg)
	}

	if status == UpgradeDownloading {
		builder = builder.Set("started_at", time.Now())
	}
	if status == UpgradeCompleted || status == UpgradeFailed || status == UpgradeTerminated {
		builder = builder.Set("completed_at", time.Now())
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update upgrade task SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update upgrade task status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgUpgradeTaskRepository) List(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	base := psql.Select(upgradeColumns...).From("upgrade_tasks")
	countBase := psql.Select("COUNT(*)").From("upgrade_tasks")

	if filter.DeviceID != nil {
		base = base.Where(sq.Eq{"device_id": *filter.DeviceID})
		countBase = countBase.Where(sq.Eq{"device_id": *filter.DeviceID})
	}
	if filter.FirmwareID != nil {
		base = base.Where(sq.Eq{"firmware_id": *filter.FirmwareID})
		countBase = countBase.Where(sq.Eq{"firmware_id": *filter.FirmwareID})
	}
	if filter.BatchID != nil {
		base = base.Where(sq.Eq{"batch_id": *filter.BatchID})
		countBase = countBase.Where(sq.Eq{"batch_id": *filter.BatchID})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count upgrade tasks SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count upgrade tasks: %w", err)
	}

	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query, args, err := base.
		OrderBy("created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list upgrade tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list upgrade tasks: %w", err)
	}
	defer rows.Close()

	var items []UpgradeTask
	for rows.Next() {
		var task UpgradeTask
		var batchID sql.NullString
		var errorMsgN sql.NullString
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&task.ID, &task.DeviceID, &task.FirmwareID, &batchID, &task.Status,
			&errorMsgN, &task.RetryCount, &task.MaxRetries,
			&startedAt, &completedAt, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan upgrade task row: %w", err)
		}

		if batchID.Valid {
			bid, _ := uuid.Parse(batchID.String)
			task.BatchID = &bid
		}
		if errorMsgN.Valid {
			task.ErrorMessage = errorMsgN.String
		}
		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		items = append(items, task)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &model.ListResponse[UpgradeTask]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *PgUpgradeTaskRepository) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeTask, error) {
	query, args, err := psql.Select(upgradeColumns...).
		From("upgrade_tasks").
		Where(sq.And{
			sq.Eq{"device_id": deviceID},
			sq.NotEq{"status": []UpgradeState{UpgradeCompleted, UpgradeFailed, UpgradeTerminated}},
		}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get active upgrade SQL: %w", err)
	}

	task, err := scanUpgradeTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get active upgrade task: %w", err)
	}
	return task, nil
}

func (r *PgUpgradeTaskRepository) CountByBatchStatus(ctx context.Context, batchID uuid.UUID) (map[UpgradeState]int64, error) {
	query, args, err := psql.Select("status", "COUNT(*)").
		From("upgrade_tasks").
		Where(sq.Eq{"batch_id": batchID}).
		GroupBy("status").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count by batch status SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("count by batch status: %w", err)
	}
	defer rows.Close()

	result := make(map[UpgradeState]int64)
	for rows.Next() {
		var status UpgradeState
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan batch status count: %w", err)
		}
		result[status] = count
	}
	return result, nil
}
