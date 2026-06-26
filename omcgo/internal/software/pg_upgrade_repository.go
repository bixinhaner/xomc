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
	"github.com/omcgo/omcgo/internal/core/storage"
)

var subTaskColumns = []string{
	"id", "task_id", "device_id", "firmware_id", "status",
	"error_message", "retry_count", "max_retries",
	"device_sn", "ori_version", "dest_version",
	"command_key", "failure_reason", "pre_suspend_status",
	"started_at", "completed_at", "created_at", "updated_at",
}

var _ SubTaskRepository = (*PgSubTaskRepository)(nil)

// PgSubTaskRepository is a PostgreSQL implementation of SubTaskRepository.
type PgSubTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgSubTaskRepository creates a new PgSubTaskRepository.
func NewPgSubTaskRepository(pool *pgxpool.Pool) *PgSubTaskRepository {
	return &PgSubTaskRepository{pool: pool}
}

func scanSubTask(row pgx.Row) (*UpgradeSubTask, error) {
	var t UpgradeSubTask
	var firmwareID sql.NullString
	var errorMsg, deviceSN, oriVer, destVer, cmdKey, failReason, preSuspend sql.NullString
	var startedAt, completedAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&t.ID, &t.TaskID, &t.DeviceID, &firmwareID, &t.Status,
		&errorMsg, &t.RetryCount, &t.MaxRetries,
		&deviceSN, &oriVer, &destVer,
		&cmdKey, &failReason, &preSuspend,
		&startedAt, &completedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if firmwareID.Valid {
		fid, _ := uuid.Parse(firmwareID.String)
		t.FirmwareID = &fid
	}
	if errorMsg.Valid {
		t.ErrorMessage = errorMsg.String
	}
	if deviceSN.Valid {
		t.DeviceSN = deviceSN.String
	}
	if oriVer.Valid {
		t.OriVersion = oriVer.String
	}
	if destVer.Valid {
		t.DestVersion = destVer.String
	}
	if cmdKey.Valid {
		t.CommandKey = cmdKey.String
	}
	if failReason.Valid {
		t.FailureReason = failReason.String
	}
	if preSuspend.Valid {
		t.PreSuspendStatus = preSuspend.String
	}
	if startedAt.Valid {
		jt := JSONTime(startedAt.Time)
		t.StartedAt = &jt
	}
	if completedAt.Valid {
		jt := JSONTime(completedAt.Time)
		t.CompletedAt = &jt
	}
	t.CreatedAt = JSONTime(createdAt)
	t.UpdatedAt = JSONTime(updatedAt)
	return &t, nil
}

func scanSubTaskRow(rows pgx.Rows) (*UpgradeSubTask, error) {
	var t UpgradeSubTask
	var firmwareID sql.NullString
	var errorMsg, deviceSN, oriVer, destVer, cmdKey, failReason, preSuspend sql.NullString
	var startedAt, completedAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&t.ID, &t.TaskID, &t.DeviceID, &firmwareID, &t.Status,
		&errorMsg, &t.RetryCount, &t.MaxRetries,
		&deviceSN, &oriVer, &destVer,
		&cmdKey, &failReason, &preSuspend,
		&startedAt, &completedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if firmwareID.Valid {
		fid, _ := uuid.Parse(firmwareID.String)
		t.FirmwareID = &fid
	}
	if errorMsg.Valid {
		t.ErrorMessage = errorMsg.String
	}
	if deviceSN.Valid {
		t.DeviceSN = deviceSN.String
	}
	if oriVer.Valid {
		t.OriVersion = oriVer.String
	}
	if destVer.Valid {
		t.DestVersion = destVer.String
	}
	if cmdKey.Valid {
		t.CommandKey = cmdKey.String
	}
	if failReason.Valid {
		t.FailureReason = failReason.String
	}
	if preSuspend.Valid {
		t.PreSuspendStatus = preSuspend.String
	}
	if startedAt.Valid {
		jt := JSONTime(startedAt.Time)
		t.StartedAt = &jt
	}
	if completedAt.Valid {
		jt := JSONTime(completedAt.Time)
		t.CompletedAt = &jt
	}
	t.CreatedAt = JSONTime(createdAt)
	t.UpdatedAt = JSONTime(updatedAt)
	return &t, nil
}

func (r *PgSubTaskRepository) Create(ctx context.Context, task *UpgradeSubTask) error {
	builder := storage.Psql.Insert("upgrade_sub_tasks").
		Columns("task_id", "device_id", "firmware_id", "status", "max_retries",
			"device_sn", "ori_version", "dest_version").
		Values(task.TaskID, task.DeviceID, task.FirmwareID, task.Status, task.MaxRetries,
			task.DeviceSN, task.OriVersion, task.DestVersion).
		Suffix("RETURNING " + joinColumns(subTaskColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert sub-task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanSubTask(row)
	if err != nil {
		return fmt.Errorf("create sub-task: %w", err)
	}
	*task = *created
	return nil
}

func (r *PgSubTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From("upgrade_sub_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get sub-task SQL: %w", err)
	}

	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get sub-task: %w", err)
	}
	return task, nil
}

func (r *PgSubTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error {
	return r.UpdateStatusWithCode(ctx, id, status, errorMsg, "")
}

// shouldSetSubTaskStartedAt 决定 status 跃迁是否触发 started_at 写入。语义：
// 「调度器已实际开始处理这个 sub_task」就写。设计原则：
//   - 包含 suspended —— issue #667：executor 把 sub_task 从 pending 挑出来发现设备
//     离线、转 suspended 时，就是用户视角「轮到该设备级别开始升级」的精确时刻；不写
//     started_at 会导致任务里 N 个离线设备的「开始时间」全部用 createdAt 兜底显示
//     同一时刻（任务下发时刻），与并发调度（max_concurrent）实际行为不符；
//   - 包含 4 种执行态 downloading/uploading/rebooting/verifying —— qa-614 #371
//     的原有逻辑（5G 手动回退 RollbackNeedsEnableCheck=false 从 pending 直跳 rebooting）；
//   - 包含 failed —— pre-flight 失败（FIRMWARE_NOT_FOUND / DEVICE_NOT_FOUND /
//     COMMAND_PUSH_FAILED）从 pending 直跳 failed，但 executor 已经"处理过"该 sub_task，
//     started_at 应记录为「调度器处理时刻」（CompletedAt 几乎同一刻，但语义清晰）；
//   - 排除 pending —— sub_task 排队中，调度器尚未挑出来，没有「开始」概念；
//   - 排除 terminated —— 操作员主动叫停，不是调度器自然推进；若停在 pending 即
//     未被调度过，started_at 留空由 mapDeviceItem 兜底显示为 endedAt（issue #655）；
//     若停在已开始的状态，started_at 已被先前转换写入，COALESCE 守卫不会被覆盖。
//   - 排除 completed —— 终态必然先经过 verifying/uploading 等执行态，started_at
//     已写；为防御诡异路径（直接 pending→completed），不主动补写避免覆盖语义。
func shouldSetSubTaskStartedAt(status UpgradeState) bool {
	switch status {
	case UpgradeSuspended, UpgradeDownloading, UpgradeUploading,
		UpgradeRebooting, UpgradeVerifying, UpgradeFailed:
		return true
	}
	return false
}

func (r *PgSubTaskRepository) UpdateStatusWithCode(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string, code FailureCode) error {
	builder := storage.Psql.Update("upgrade_sub_tasks").
		Set("status", status).
		Where(sq.Eq{"id": id})

	if errorMsg != "" {
		builder = builder.Set("error_message", errorMsg)
	}
	if code != "" && status == UpgradeFailed {
		builder = builder.Set("failure_reason", string(code))
	}
	// 用 COALESCE(started_at, now()) 守卫：仅当 started_at 仍为 NULL 时写入，避免多次
	// 状态翻转把先前已记录的开始时间覆盖。触发状态集见 shouldSetSubTaskStartedAt 注释。
	if shouldSetSubTaskStartedAt(status) {
		builder = builder.Set("started_at", sq.Expr("COALESCE(started_at, now())"))
	}
	if status == UpgradeCompleted || status == UpgradeFailed || status == UpgradeTerminated {
		builder = builder.Set("completed_at", time.Now())
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update sub-task status SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update sub-task status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgSubTaskRepository) Update(ctx context.Context, task *UpgradeSubTask) error {
	builder := storage.Psql.Update("upgrade_sub_tasks").
		Set("status", task.Status).
		Set("error_message", task.ErrorMessage).
		Set("retry_count", task.RetryCount).
		Set("command_key", task.CommandKey).
		Set("failure_reason", task.FailureReason).
		Set("pre_suspend_status", task.PreSuspendStatus).
		Set("device_sn", task.DeviceSN).
		Set("ori_version", task.OriVersion).
		Set("dest_version", task.DestVersion).
		Where(sq.Eq{"id": task.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update sub-task SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update sub-task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgSubTaskRepository) List(ctx context.Context, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return r.ListByTaskID(ctx, filter.TaskID, filter)
}

func (r *PgSubTaskRepository) ListByTaskID(ctx context.Context, taskID uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	cols := make([]string, len(subTaskColumns))
	cols[0] = "ust." + subTaskColumns[0]
	for i := 1; i < len(subTaskColumns); i++ {
		cols[i] = "ust." + subTaskColumns[i]
	}
	cols = append(cols, "ut.task_name")

	base := storage.Psql.Select(cols...).
		From("upgrade_sub_tasks ust").
		Join("upgrade_tasks ut ON ust.task_id = ut.id")
	countBase := storage.Psql.Select("COUNT(*)").
		From("upgrade_sub_tasks ust").
		Where(sq.Eq{"ust.task_id": taskID})

	base = base.Where(sq.Eq{"ust.task_id": taskID})

	if len(filter.Statuses) > 0 {
		base = base.Where(sq.Eq{"ust.status": filter.Statuses})
		countBase = countBase.Where(sq.Eq{"ust.status": filter.Statuses})
	} else if filter.Status != nil {
		base = base.Where(sq.Eq{"ust.status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"ust.status": *filter.Status})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count sub-tasks SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count sub-tasks: %w", err)
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
		OrderBy("ust.created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list sub-tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sub-tasks: %w", err)
	}
	defer rows.Close()

	var items []UpgradeSubTaskWithTaskName
	for rows.Next() {
		var t UpgradeSubTaskWithTaskName
		var firmwareID sql.NullString
		var errorMsg, deviceSN, oriVer, destVer, cmdKey, failReason, preSuspend, taskName sql.NullString
		var startedAt, completedAt sql.NullTime
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&t.ID, &t.TaskID, &t.DeviceID, &firmwareID, &t.Status,
			&errorMsg, &t.RetryCount, &t.MaxRetries,
			&deviceSN, &oriVer, &destVer,
			&cmdKey, &failReason, &preSuspend,
			&startedAt, &completedAt, &createdAt, &updatedAt,
			&taskName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sub-task row: %w", err)
		}

		if firmwareID.Valid {
			fid, _ := uuid.Parse(firmwareID.String)
			t.FirmwareID = &fid
		}
		if errorMsg.Valid {
			t.ErrorMessage = errorMsg.String
		}
		if deviceSN.Valid {
			t.DeviceSN = deviceSN.String
		}
		if oriVer.Valid {
			t.OriVersion = oriVer.String
		}
		if destVer.Valid {
			t.DestVersion = destVer.String
		}
		if cmdKey.Valid {
			t.CommandKey = cmdKey.String
		}
		if failReason.Valid {
			t.FailureReason = failReason.String
		}
		if preSuspend.Valid {
			t.PreSuspendStatus = preSuspend.String
		}
		if startedAt.Valid {
			jt := JSONTime(startedAt.Time)
			t.StartedAt = &jt
		}
		if completedAt.Valid {
			jt := JSONTime(completedAt.Time)
			t.CompletedAt = &jt
		}
		if taskName.Valid {
			t.TaskName = taskName.String
		}
		t.CreatedAt = JSONTime(createdAt)
		t.UpdatedAt = JSONTime(updatedAt)
		items = append(items, t)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &model.ListResponse[UpgradeSubTaskWithTaskName]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *PgSubTaskRepository) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From("upgrade_sub_tasks").
		Where(sq.And{
			sq.Eq{"device_id": deviceID},
			sq.NotEq{"status": []UpgradeState{UpgradeCompleted, UpgradeFailed, UpgradeTerminated}},
		}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get active sub-task SQL: %w", err)
	}

	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get active sub-task: %w", err)
	}
	return task, nil
}

func (r *PgSubTaskRepository) GetByCommandKey(ctx context.Context, commandKey string) (*UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From("upgrade_sub_tasks").
		Where(sq.Eq{"command_key": commandKey}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get by command key SQL: %w", err)
	}

	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get sub-task by command key: %w", err)
	}
	return task, nil
}

func (r *PgSubTaskRepository) BatchCreate(ctx context.Context, tasks []*UpgradeSubTask) error {
	if len(tasks) == 0 {
		return nil
	}

	builder := storage.Psql.Insert("upgrade_sub_tasks").
		Columns("task_id", "device_id", "firmware_id", "status", "max_retries",
			"device_sn", "ori_version", "dest_version", "command_key")

	for _, t := range tasks {
		builder = builder.Values(
			t.TaskID, t.DeviceID, t.FirmwareID, t.Status, t.MaxRetries,
			t.DeviceSN, t.OriVersion, t.DestVersion, t.CommandKey,
		)
	}

	builder = builder.Suffix("RETURNING " + joinColumns(subTaskColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build batch insert sub-tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch create sub-tasks: %w", err)
	}
	defer rows.Close()

	idx := 0
	for rows.Next() {
		created, err := scanSubTaskRow(rows)
		if err != nil {
			return fmt.Errorf("scan batch created sub-task: %w", err)
		}
		if idx < len(tasks) {
			*tasks[idx] = *created
		}
		idx++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("batch create sub-tasks rows: %w", err)
	}
	return nil
}

func (r *PgSubTaskRepository) DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error {
	query, args, err := storage.Psql.Delete("upgrade_sub_tasks").Where(sq.Eq{"task_id": taskID}).ToSql()
	if err != nil {
		return fmt.Errorf("build delete sub-tasks SQL: %w", err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete sub-tasks by task_id: %w", err)
	}
	return nil
}


func (r *PgSubTaskRepository) FailStale(ctx context.Context, cutoffs StaleTimeouts) (map[uuid.UUID]int64, error) {
	rpcCutoff := time.Now().Add(-cutoffs.RPCResponse)
	onlineCutoff := time.Now().Add(-cutoffs.DeviceOnline)
	tcCutoff := time.Now().Add(-cutoffs.TransferComplete)

	// 三段超时口径：
	//   · 'downloading'（Download RPC，固件升级 / 回滚）→ RPCResponse 短超时（默认 5min）
	//     —— 设备拿到 Download SOAP 后应该很快回 DownloadResponse；超时多半是 ACS 没派发或 CPE 没回。
	//   · 'suspended' → DeviceOnline 长超时（默认 60min）—— 等设备 inform 上线。
	//   · 其它（'uploading' / 'rebooting' / 'verifying'）→ TransferComplete 超时（默认 30min）
	//     —— 文件上传 + CPE 内部安装 / 回写 TC 都属于"已派发 RPC，等事务完成"阶段，时间窗较长。
	// 这里把 'uploading' 归入第三段而不是 'downloading' 同款 RPCResponse 短超时，是 fix 顺手修的 BUG：
	// 旧实现把 Upload 也算成 'downloading'，5min 内卡死的大文件备份 / 日志采集会被误判超时。
	//
	// issue #667：started_at = COALESCE(started_at, NOW()) 兜底——理论上 sub_task 走到
	// downloading/suspended/uploading/... 这几个状态时 UpdateStatusWithCode 已写过 started_at
	// （shouldSetSubTaskStartedAt 覆盖），但 reaper 是直接 SQL 不走 ORM；万一未来有路径
	// 绕过 UpdateStatusWithCode 直接 UPDATE 状态，这里兜底保证「被 reaper 处理过的 sub_task
	// 一定有 started_at」，避免前端「开始时间」空列。COALESCE 守卫不会覆盖已有值。
	query := `WITH failed AS (
		UPDATE upgrade_sub_tasks ust
		SET status = 'failed', error_message = CASE
		    WHEN ust.status = 'downloading' THEN 'Timed out waiting for RPC response from device.'
		    WHEN ust.status = 'uploading'   THEN 'Timed out waiting for upload / TransferComplete from device.'
		    WHEN ust.status = 'suspended'   THEN 'Timed out waiting for device to come online.'
		    ELSE                                 'Timed out waiting for TransferComplete from device.'
		END, started_at = COALESCE(ust.started_at, NOW()), completed_at = NOW(), updated_at = NOW()
		FROM upgrade_tasks ut
		WHERE ust.task_id = ut.id
		  AND ut.status NOT IN ('pending', 'suspended')
		  AND (
		    (ust.status = 'downloading' AND ust.updated_at < $1)
		    OR (ust.status = 'suspended'  AND ust.updated_at < $2)
		    OR (ust.status NOT IN ('completed', 'failed', 'terminated', 'downloading', 'suspended') AND ust.updated_at < $3)
		  )
		RETURNING ust.task_id
	)
	SELECT task_id, COUNT(*)::bigint AS cnt
	FROM failed
	GROUP BY task_id`

	rows, err := r.pool.Query(ctx, query, rpcCutoff, onlineCutoff, tcCutoff)
	if err != nil {
		return nil, fmt.Errorf("fail stale sub-tasks: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]int64)
	for rows.Next() {
		var taskID uuid.UUID
		var cnt int64
		if err := rows.Scan(&taskID, &cnt); err != nil {
			return nil, fmt.Errorf("scan stale task counts: %w", err)
		}
		result[taskID] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fail stale sub-tasks rows: %w", err)
	}
	return result, nil
}
// ListAll returns sub-tasks across all main tasks, JOINing upgrade_tasks for task_name.
func (r *PgSubTaskRepository) ListAll(ctx context.Context, filter AllSubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	cols := append([]string{"ust." + subTaskColumns[0]}, subTaskColumns[1:]...)
	for i := range cols {
		if i > 0 {
			cols[i] = "ust." + cols[i]
		}
	}
	cols = append(cols, "ut.task_name")

	base := storage.Psql.Select(cols...).
		From("upgrade_sub_tasks ust").
		Join("upgrade_tasks ut ON ust.task_id = ut.id")
	countBase := storage.Psql.Select("COUNT(*)").
		From("upgrade_sub_tasks ust").
		Join("upgrade_tasks ut ON ust.task_id = ut.id")

	if filter.TaskName != nil && *filter.TaskName != "" {
		pred := sq.ILike{"ut.task_name": "%" + *filter.TaskName + "%"}
		base = base.Where(pred)
		countBase = countBase.Where(pred)
	}
	if filter.DeviceSN != nil && *filter.DeviceSN != "" {
		pred := sq.ILike{"ust.device_sn": "%" + *filter.DeviceSN + "%"}
		base = base.Where(pred)
		countBase = countBase.Where(pred)
	}
	if filter.Status != nil {
		pred := sq.Eq{"ust.status": *filter.Status}
		base = base.Where(pred)
		countBase = countBase.Where(pred)
	}

	if filter.TaskType != nil {
		pred := sq.Eq{"ut.task_type": *filter.TaskType}
		base = base.Where(pred)
		countBase = countBase.Where(pred)
	}
	if filter.TaskID != nil {
		pred := sq.Eq{"ust.task_id": *filter.TaskID}
		base = base.Where(pred)
		countBase = countBase.Where(pred)
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count all sub-tasks SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count all sub-tasks: %w", err)
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
		OrderBy("ust.created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list all sub-tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all sub-tasks: %w", err)
	}
	defer rows.Close()

	var items []UpgradeSubTaskWithTaskName
	for rows.Next() {
		var t UpgradeSubTaskWithTaskName
		var firmwareID sql.NullString
		var errorMsg, deviceSN, oriVer, destVer, cmdKey, failReason, preSuspend, taskName sql.NullString
		var startedAt, completedAt sql.NullTime
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&t.ID, &t.TaskID, &t.DeviceID, &firmwareID, &t.Status,
			&errorMsg, &t.RetryCount, &t.MaxRetries,
			&deviceSN, &oriVer, &destVer,
			&cmdKey, &failReason, &preSuspend,
			&startedAt, &completedAt, &createdAt, &updatedAt,
			&taskName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan all sub-task row: %w", err)
		}

		if firmwareID.Valid {
			fid, _ := uuid.Parse(firmwareID.String)
			t.FirmwareID = &fid
		}
		if errorMsg.Valid {
			t.ErrorMessage = errorMsg.String
		}
		if deviceSN.Valid {
			t.DeviceSN = deviceSN.String
		}
		if oriVer.Valid {
			t.OriVersion = oriVer.String
		}
		if destVer.Valid {
			t.DestVersion = destVer.String
		}
		if cmdKey.Valid {
			t.CommandKey = cmdKey.String
		}
		if failReason.Valid {
			t.FailureReason = failReason.String
		}
		if preSuspend.Valid {
			t.PreSuspendStatus = preSuspend.String
		}
		if startedAt.Valid {
			jt := JSONTime(startedAt.Time)
			t.StartedAt = &jt
		}
		if completedAt.Valid {
			jt := JSONTime(completedAt.Time)
			t.CompletedAt = &jt
		}
		if taskName.Valid {
			t.TaskName = taskName.String
		}
		t.CreatedAt = JSONTime(createdAt)
		t.UpdatedAt = JSONTime(updatedAt)
		items = append(items, t)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &model.ListResponse[UpgradeSubTaskWithTaskName]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateDestVersionByCommandKey 按 command_key 把 dest_version 写成 destVersion。
// CONFIG_RESTORE / LICENSE_UPGRADE 派发后回填实际下发文件名。RowsAffected=0 不报错——
// command_key 跨表只在某一张表落库，本表未命中是预期。
func (r *PgSubTaskRepository) UpdateDestVersionByCommandKey(ctx context.Context, commandKey, destVersion string) error {
	query, args, err := storage.Psql.Update("upgrade_sub_tasks").
		Set("dest_version", destVersion).
		Where(sq.Eq{"command_key": commandKey}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update dest_version SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update dest_version by command_key: %w", err)
	}
	return nil
}

// UpdateDestVersionByID sets dest_version for a single sub-task by its ID (qa-614 #371).
// RowsAffected=0 不视为错误（子任务可能已被清理）。
func (r *PgSubTaskRepository) UpdateDestVersionByID(ctx context.Context, id uuid.UUID, destVersion string) error {
	query, args, err := storage.Psql.Update("upgrade_sub_tasks").
		Set("dest_version", destVersion).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update dest_version by id SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update dest_version by id: %w", err)
	}
	return nil
}

// UpdateFailureReasonByTask sets failure_reason for all failed sub-tasks under a main task.
func (r *PgSubTaskRepository) UpdateFailureReasonByTask(ctx context.Context, taskID uuid.UUID, code FailureCode) error {
	query, args, err := storage.Psql.Update("upgrade_sub_tasks").
		Set("failure_reason", string(code)).
		Where(sq.And{
			sq.Eq{"task_id": taskID},
			sq.Eq{"status": UpgradeFailed},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update failure_reason SQL: %w", err)
	}
	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update failure_reason: %w", err)
	}
	return nil
}
