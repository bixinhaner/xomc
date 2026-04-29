package software

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var taskColumns = []string{
	"id", "task_name", "task_type", "firmware_id", "file_name", "file_md5",
	"status", "result", "product_class", "is_keep_config",
	"create_status", "create_user", "total_count", "success_count", "fail_count",
	"max_concurrent", "started_at", "ended_at", "created_at", "updated_at",
}

var _ TaskRepository = (*PgTaskRepository)(nil)

// PgTaskRepository is a PostgreSQL implementation of TaskRepository for main upgrade tasks.
type PgTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgTaskRepository creates a new PgTaskRepository.
func NewPgTaskRepository(pool *pgxpool.Pool) *PgTaskRepository {
	return &PgTaskRepository{pool: pool}
}

func scanUpgradeTask(row pgx.Row) (*UpgradeTask, error) {
	var task UpgradeTask
	var firmwareID sql.NullString
	var fileName, fileMD5, result sql.NullString
	var startedAt, endedAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&task.ID, &task.TaskName, &task.TaskType, &firmwareID, &fileName, &fileMD5,
		&task.Status, &result, &task.ProductClass, &task.IsKeepConfig,
		&task.CreateStatus, &task.CreateUser, &task.TotalCount, &task.SuccessCount, &task.FailCount,
		&task.MaxConcurrent, &startedAt, &endedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if firmwareID.Valid {
		fid, _ := uuid.Parse(firmwareID.String)
		task.FirmwareID = &fid
	}
	if fileName.Valid {
		task.FileName = fileName.String
	}
	if fileMD5.Valid {
		task.FileMD5 = fileMD5.String
	}
	if result.Valid {
		task.Result = TaskResult(result.String)
	}
	if startedAt.Valid {
		t := JSONTime(startedAt.Time)
		task.StartedAt = &t
	}
	if endedAt.Valid {
		t := JSONTime(endedAt.Time)
		task.EndedAt = &t
	}
	task.CreatedAt = JSONTime(createdAt)
	task.UpdatedAt = JSONTime(updatedAt)
	return &task, nil
}

func (r *PgTaskRepository) Create(ctx context.Context, task *UpgradeTask) error {
	builder := storage.Psql.Insert("upgrade_tasks").
		Columns("task_name", "task_type", "firmware_id", "file_name", "file_md5",
			"status", "product_class", "is_keep_config",
			"create_status", "create_user", "total_count", "max_concurrent").
		Values(task.TaskName, task.TaskType, task.FirmwareID, task.FileName, task.FileMD5,
			task.Status, task.ProductClass, task.IsKeepConfig,
			task.CreateStatus, task.CreateUser, task.TotalCount, task.MaxConcurrent).
		Suffix("RETURNING " + joinColumns(taskColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanUpgradeTask(row)
	if err != nil {
		return fmt.Errorf("create upgrade task: %w", err)
	}
	*task = *created
	return nil
}

func (r *PgTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error) {
	query, args, err := storage.Psql.Select(taskColumns...).
		From("upgrade_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get task SQL: %w", err)
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

func (r *PgTaskRepository) Update(ctx context.Context, task *UpgradeTask) error {
	builder := storage.Psql.Update("upgrade_tasks").
		Set("task_name", task.TaskName).
		Set("task_type", task.TaskType).
		Set("firmware_id", task.FirmwareID).
		Set("file_name", task.FileName).
		Set("file_md5", task.FileMD5).
		Set("status", task.Status).
		Set("result", task.Result).
		Set("product_class", task.ProductClass).
		Set("is_keep_config", task.IsKeepConfig).
		Set("create_status", task.CreateStatus).
		Set("create_user", task.CreateUser).
		Set("total_count", task.TotalCount).
		Set("success_count", task.SuccessCount).
		Set("fail_count", task.FailCount).
		Set("max_concurrent", task.MaxConcurrent).
		Set("ended_at", task.EndedAt).
		Where(sq.Eq{"id": task.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update task SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update upgrade task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error {
	builder := storage.Psql.Update("upgrade_tasks").
		Set("status", status).
		Where(sq.Eq{"id": id})

	if result != "" {
		builder = builder.Set("result", result)
	}
	if status == TaskInProgress {
		builder = builder.Set("started_at", time.Now())
	}
	if status == TaskEnded {
		builder = builder.Set("ended_at", time.Now())
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update task status SQL: %w", err)
	}

	result_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update upgrade task status: %w", err)
	}
	if result_.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) List(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	base := storage.Psql.Select(taskColumns...).From("upgrade_tasks")
	countBase := storage.Psql.Select("COUNT(*)").From("upgrade_tasks")

	if filter.TaskType != nil {
		base = base.Where(sq.Eq{"task_type": *filter.TaskType})
		countBase = countBase.Where(sq.Eq{"task_type": *filter.TaskType})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.ProductClass != nil {
		base = base.Where(sq.Eq{"product_class": *filter.ProductClass})
		countBase = countBase.Where(sq.Eq{"product_class": *filter.ProductClass})
	}
	if filter.CreateUser != nil {
		base = base.Where(sq.Eq{"create_user": *filter.CreateUser})
		countBase = countBase.Where(sq.Eq{"create_user": *filter.CreateUser})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count tasks SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
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
		return nil, fmt.Errorf("build list tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var items []UpgradeTask
	for rows.Next() {
		task, err := scanUpgradeTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		items = append(items, *task)
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

func (r *PgTaskRepository) IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error {
	query := `UPDATE upgrade_tasks
	          SET success_count = success_count + $1,
	              fail_count = fail_count + $2,
	              updated_at = NOW()
	          WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, successDelta, failDelta, taskID)
	if err != nil {
		return fmt.Errorf("increment task counts: %w", err)
	}
	return nil
}

func (r *PgTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("upgrade_tasks").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build delete task SQL: %w", err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete upgrade task: %w", err)
	}
	return nil
}

func scanUpgradeTaskRow(rows pgx.Rows) (*UpgradeTask, error) {
	var task UpgradeTask
	var firmwareID sql.NullString
	var fileName, fileMD5, result sql.NullString
	var startedAt, endedAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&task.ID, &task.TaskName, &task.TaskType, &firmwareID, &fileName, &fileMD5,
		&task.Status, &result, &task.ProductClass, &task.IsKeepConfig,
		&task.CreateStatus, &task.CreateUser, &task.TotalCount, &task.SuccessCount, &task.FailCount,
		&task.MaxConcurrent, &startedAt, &endedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if firmwareID.Valid {
		fid, _ := uuid.Parse(firmwareID.String)
		task.FirmwareID = &fid
	}
	if fileName.Valid {
		task.FileName = fileName.String
	}
	if fileMD5.Valid {
		task.FileMD5 = fileMD5.String
	}
	if result.Valid {
		task.Result = TaskResult(result.String)
	}
	if startedAt.Valid {
		t := JSONTime(startedAt.Time)
		task.StartedAt = &t
	}
	if endedAt.Valid {
		t := JSONTime(endedAt.Time)
		task.EndedAt = &t
	}
	task.CreatedAt = JSONTime(createdAt)
	task.UpdatedAt = JSONTime(updatedAt)
	return &task, nil
}

// =============================================================================
// Canary fields (T-0018 / R-101) — separate read/write paths so the legacy
// scanUpgradeTask + Create remain byte-identical.
// =============================================================================

// GetCanaryFields reads the canary-related columns for one task. total_count
// is included so the cron monitor can compute per-stage device counts without
// a second round trip.
func (r *PgTaskRepository) GetCanaryFields(ctx context.Context, id uuid.UUID) (*CanaryFields, error) {
	const q = `
		SELECT strategy, canary_stages, current_stage, stage_status, stage_history,
		       auto_advance, auto_advance_minutes, total_count
		FROM upgrade_tasks
		WHERE id = $1`
	var (
		strategy           string
		stagesRaw          []byte
		currentStage       int
		stageStatus        string
		historyRaw         []byte
		autoAdvance        bool
		autoAdvanceMinutes int
		totalCount         int
	)
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&strategy, &stagesRaw, &currentStage, &stageStatus, &historyRaw,
		&autoAdvance, &autoAdvanceMinutes, &totalCount,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get canary fields: %w", err)
	}

	stages, err := UnmarshalStages(stagesRaw)
	if err != nil {
		return nil, fmt.Errorf("decode canary_stages: %w", err)
	}

	var history []StageHistoryEntry
	if len(historyRaw) > 0 {
		if err := json.Unmarshal(historyRaw, &history); err != nil {
			return nil, fmt.Errorf("decode stage_history: %w", err)
		}
	}

	return &CanaryFields{
		TaskID:             id,
		Strategy:           strategy,
		Stages:             stages,
		CurrentStage:       currentStage,
		StageStatus:        stageStatus,
		StageHistory:       history,
		AutoAdvance:        autoAdvance,
		AutoAdvanceMinutes: autoAdvanceMinutes,
		TotalCount:         totalCount,
	}, nil
}

// UpdateCanaryFields persists every canary column. Used both at Canary start
// (after Create) and on stage transitions (advance/pause/resume/abort).
func (r *PgTaskRepository) UpdateCanaryFields(ctx context.Context, id uuid.UUID, fields *CanaryFields) error {
	if fields == nil {
		return fmt.Errorf("UpdateCanaryFields: nil fields: %w", commonerrors.ErrInvalidInput)
	}
	stagesRaw, err := MarshalStages(fields.Stages)
	if err != nil {
		return fmt.Errorf("encode canary_stages: %w", err)
	}
	historyRaw, err := json.Marshal(fields.StageHistory)
	if err != nil {
		return fmt.Errorf("encode stage_history: %w", err)
	}

	const q = `
		UPDATE upgrade_tasks
		SET strategy = $1,
		    canary_stages = $2,
		    current_stage = $3,
		    stage_status = $4,
		    stage_history = $5,
		    auto_advance = $6,
		    auto_advance_minutes = $7,
		    updated_at = NOW()
		WHERE id = $8`
	tag, err := r.pool.Exec(ctx, q,
		fields.Strategy, stagesRaw, fields.CurrentStage, fields.StageStatus,
		historyRaw, fields.AutoAdvance, fields.AutoAdvanceMinutes, id,
	)
	if err != nil {
		return fmt.Errorf("update canary fields: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// ListActiveCanaryTaskIDs returns ids of canary tasks that are still in a
// non-terminal stage_status (running or paused). Used by the cron monitor.
func (r *PgTaskRepository) ListActiveCanaryTaskIDs(ctx context.Context) ([]uuid.UUID, error) {
	const q = `
		SELECT id FROM upgrade_tasks
		WHERE strategy = 'canary' AND stage_status IN ('running', 'paused')
		ORDER BY updated_at ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list active canary tasks: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan canary task id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
