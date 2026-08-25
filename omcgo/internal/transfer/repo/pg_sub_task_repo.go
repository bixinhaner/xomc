package repo

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
	"github.com/omcgo/omcgo/internal/software"
)

// subTaskColumns 跟 software.subTaskColumns 严格保持一致 — 4 张新 sub_task 表
// 由 CREATE TABLE LIKE upgrade_sub_tasks 复制而来，列顺序一致才能复用 scan。
var subTaskColumns = []string{
	"id", "task_id", "device_id", "firmware_id", "status",
	"error_message", "retry_count", "max_retries",
	"device_sn", "ori_version", "dest_version",
	"command_key", "failure_reason", "pre_suspend_status",
	"started_at", "completed_at", "created_at", "updated_at",
}

var _ SubTaskRepo = (*PgSubTaskRepo)(nil)

// nullableStr 把空字符串映射为 SQL NULL，避免占位 sub_task 把空 command_key 当
// 字面值写入。command_key 列允许 NULL（migration 没加 NOT NULL），保持 NULL 语义
// 让"行没装 CommandKey"和"装了空串"可区分。
func nullableStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// PgSubTaskRepo 是 SubTaskRepo 接口的 PostgreSQL 实现。
// 持有两张物理表名：subTaskTable（本表）+ mainTable（List 的 JOIN 目标 / FailStale 关联用）。
type PgSubTaskRepo struct {
	pool         *pgxpool.Pool
	subTaskTable string
	mainTable    string
}

// NewPgSubTaskRepo 创建 sub_task 仓储。
//
//	subTaskTable: config_backup_sub_tasks / config_restore_sub_tasks / ...
//	mainTable:    对应的 main 表（List/ListByTaskID JOIN 取 task_name 用）
func NewPgSubTaskRepo(pool *pgxpool.Pool, subTaskTable, mainTable string) *PgSubTaskRepo {
	return &PgSubTaskRepo{pool: pool, subTaskTable: subTaskTable, mainTable: mainTable}
}

func scanSubTask(row pgx.Row) (*software.UpgradeSubTask, error) {
	var t software.UpgradeSubTask
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
		jt := software.JSONTime(startedAt.Time)
		t.StartedAt = &jt
	}
	if completedAt.Valid {
		jt := software.JSONTime(completedAt.Time)
		t.CompletedAt = &jt
	}
	t.CreatedAt = software.JSONTime(createdAt)
	t.UpdatedAt = software.JSONTime(updatedAt)
	return &t, nil
}

func (r *PgSubTaskRepo) Create(ctx context.Context, task *software.UpgradeSubTask) error {
	// command_key 必须落库：占位 sub_task 在 CreatePlaceholderTrackingTask 阶段就把
	// CommandKey 算好（CONFIG_RESTORE_<id8>_<sn> / LICENSE_UPGRADE_*），TC 回流时
	// handleTCBody → GetByCommandKey 全表 fan-out 反查 sub_task。早期实现漏了这列，
	// DB 行 command_key=NULL → 永远 NotFound → 任务超时（实测 17:54 那次 CONFIG_RESTORE）。
	builder := storage.Psql.Insert(r.subTaskTable).
		Columns("task_id", "device_id", "firmware_id", "status", "max_retries",
			"device_sn", "ori_version", "dest_version", "command_key").
		Values(task.TaskID, task.DeviceID, task.FirmwareID, task.Status, task.MaxRetries,
			task.DeviceSN, task.OriVersion, task.DestVersion, nullableStr(task.CommandKey)).
		Suffix("RETURNING " + joinColumns(subTaskColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert %s SQL: %w", r.subTaskTable, err)
	}
	created, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return fmt.Errorf("create %s: %w", r.subTaskTable, err)
	}
	*task = *created
	return nil
}

func (r *PgSubTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*software.UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From(r.subTaskTable).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get %s SQL: %w", r.subTaskTable, err)
	}
	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get %s: %w", r.subTaskTable, err)
	}
	return task, nil
}

func (r *PgSubTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status software.UpgradeState, errorMsg string) error {
	return r.UpdateStatusWithCode(ctx, id, status, errorMsg, "")
}

func (r *PgSubTaskRepo) UpdateStatusWithCode(ctx context.Context, id uuid.UUID, status software.UpgradeState, errorMsg string, code software.FailureCode) error {
	builder := storage.Psql.Update(r.subTaskTable).
		Set("status", status).
		Where(sq.Eq{"id": id})

	if errorMsg != "" {
		builder = builder.Set("error_message", errorMsg)
	}
	if code != "" && status == software.UpgradeFailed {
		builder = builder.Set("failure_reason", string(code))
	}
	if status == software.UpgradeCompleted || status == software.UpgradeFailed || status == software.UpgradeTerminated {
		builder = builder.Where(sq.NotEq{"status": []software.UpgradeState{
			software.UpgradeCompleted, software.UpgradeFailed, software.UpgradeTerminated,
		}})
	}
	if status == software.UpgradeDownloading || status == software.UpgradeUploading {
		builder = builder.Set("started_at", time.Now())
	}
	if status == software.UpgradeCompleted || status == software.UpgradeFailed || status == software.UpgradeTerminated {
		builder = builder.Set("completed_at", time.Now())
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update %s status SQL: %w", r.subTaskTable, err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update %s status: %w", r.subTaskTable, err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// UpdateStatusByOperator 与 UpdateStatusWithCode 一致，唯一区别是 **不写 started_at**。
// transfer 类业务（备份 / 配置恢复 / 日志采集）目前没有 operator suspend 路径，
// 但为了与 BasicSubTaskRepo 接口对齐（RoutingSubTaskRepository 按 ID 路由到该实现时需一致
// 签名）补齐。语义：若未来 transfer 业务引入 operator suspend，默认不干扰 started_at。
func (r *PgSubTaskRepo) UpdateStatusByOperator(ctx context.Context, id uuid.UUID, status software.UpgradeState, errorMsg string) error {
	builder := storage.Psql.Update(r.subTaskTable).
		Set("status", status).
		Where(sq.Eq{"id": id})

	if errorMsg != "" {
		builder = builder.Set("error_message", errorMsg)
	}
	if status == software.UpgradeCompleted || status == software.UpgradeFailed || status == software.UpgradeTerminated {
		builder = builder.Set("completed_at", time.Now())
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update %s status SQL: %w", r.subTaskTable, err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update %s status: %w", r.subTaskTable, err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgSubTaskRepo) Update(ctx context.Context, task *software.UpgradeSubTask) error {
	builder := storage.Psql.Update(r.subTaskTable).
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
		return fmt.Errorf("build update %s SQL: %w", r.subTaskTable, err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update %s: %w", r.subTaskTable, err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgSubTaskRepo) List(ctx context.Context, filter software.SubTaskFilter) (*model.ListResponse[software.UpgradeSubTaskWithTaskName], error) {
	return r.ListByTaskID(ctx, filter.TaskID, filter)
}

func (r *PgSubTaskRepo) ListByTaskID(ctx context.Context, taskID uuid.UUID, filter software.SubTaskFilter) (*model.ListResponse[software.UpgradeSubTaskWithTaskName], error) {
	cols := make([]string, len(subTaskColumns))
	for i := range subTaskColumns {
		cols[i] = "ust." + subTaskColumns[i]
	}
	cols = append(cols, "ut.task_name")

	subAlias := r.subTaskTable + " ust"
	// Squirrel 的 Join() 接受 "table_expression ON cond"；alias 只能写一次。
	joinExpr := r.mainTable + " ut ON ust.task_id = ut.id"

	base := storage.Psql.Select(cols...).
		From(subAlias).
		Join(joinExpr)
	countBase := storage.Psql.Select("COUNT(*)").From(subAlias)

	base = base.Where(sq.Eq{"ust.task_id": taskID})
	countBase = countBase.Where(sq.Eq{"ust.task_id": taskID})

	if len(filter.Statuses) > 0 {
		base = base.Where(sq.Eq{"ust.status": filter.Statuses})
		countBase = countBase.Where(sq.Eq{"ust.status": filter.Statuses})
	} else if filter.Status != nil {
		base = base.Where(sq.Eq{"ust.status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"ust.status": *filter.Status})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count %s SQL: %w", r.subTaskTable, err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count %s: %w", r.subTaskTable, err)
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
		return nil, fmt.Errorf("build list %s SQL: %w", r.subTaskTable, err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", r.subTaskTable, err)
	}
	defer rows.Close()

	items, err := scanSubTaskWithName(rows)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &model.ListResponse[software.UpgradeSubTaskWithTaskName]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *PgSubTaskRepo) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*software.UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From(r.subTaskTable).
		Where(sq.And{
			sq.Eq{"device_id": deviceID},
			sq.NotEq{"status": []software.UpgradeState{
				software.UpgradeCompleted, software.UpgradeFailed, software.UpgradeTerminated,
			}},
		}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get active %s SQL: %w", r.subTaskTable, err)
	}

	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get active %s: %w", r.subTaskTable, err)
	}
	return task, nil
}

func (r *PgSubTaskRepo) GetByCommandKey(ctx context.Context, commandKey string) (*software.UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From(r.subTaskTable).
		Where(sq.Eq{"command_key": commandKey}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get %s by command_key SQL: %w", r.subTaskTable, err)
	}
	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get %s by command_key: %w", r.subTaskTable, err)
	}
	return task, nil
}

func (r *PgSubTaskRepo) BatchCreate(ctx context.Context, tasks []*software.UpgradeSubTask) error {
	if len(tasks) == 0 {
		return nil
	}
	// 同 Create 注释：command_key 必须包含进 INSERT，否则 TC 回流命中失败。
	builder := storage.Psql.Insert(r.subTaskTable).
		Columns("task_id", "device_id", "firmware_id", "status", "max_retries",
			"device_sn", "ori_version", "dest_version", "command_key")
	for _, t := range tasks {
		builder = builder.Values(
			t.TaskID, t.DeviceID, t.FirmwareID, t.Status, t.MaxRetries,
			t.DeviceSN, t.OriVersion, t.DestVersion, nullableStr(t.CommandKey),
		)
	}
	builder = builder.Suffix("RETURNING " + joinColumns(subTaskColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build batch insert %s SQL: %w", r.subTaskTable, err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch create %s: %w", r.subTaskTable, err)
	}
	defer rows.Close()

	idx := 0
	for rows.Next() {
		created, err := scanSubTask(rows)
		if err != nil {
			return fmt.Errorf("scan batch created %s: %w", r.subTaskTable, err)
		}
		if idx < len(tasks) {
			*tasks[idx] = *created
		}
		idx++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("batch create %s rows: %w", r.subTaskTable, err)
	}
	return nil
}

func (r *PgSubTaskRepo) DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error {
	query, args, err := storage.Psql.Delete(r.subTaskTable).Where(sq.Eq{"task_id": taskID}).ToSql()
	if err != nil {
		return fmt.Errorf("build delete %s SQL: %w", r.subTaskTable, err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete %s by task_id: %w", r.subTaskTable, err)
	}
	return nil
}

// FailStale 把所有超时的 sub_task 置 failed。三段超时口径与 software.PgSubTaskRepository.FailStale
// 完全一致；这里把表名参数化以适配 4 张新业务表。
//
// 备注：新业务（备份 / 下发 / 日志）的"downloading"在实际语义上多为 RPC 派发但等待 TC 的阶段，
// 跟 firmware 升级的 downloading 不完全对等；但既然 schema 同源、状态机也复用了 software 的
// UpgradeState 枚举，这里保留同样的三段超时分类，由 StaleTimeouts.RPCResponse 单调参数控制。
func (r *PgSubTaskRepo) FailStale(ctx context.Context, cutoffs software.StaleTimeouts) (software.StaleFailures, error) {
	rpcCutoff := time.Now().Add(-cutoffs.RPCResponse)
	onlineCutoff := time.Now().Add(-cutoffs.DeviceOnline)
	tcCutoff := time.Now().Add(-cutoffs.TransferComplete)

	query := fmt.Sprintf(`WITH failed AS (
		UPDATE %[1]s ust
		SET status = 'failed', error_message = CASE
		    WHEN ust.status = 'downloading' THEN 'Timed out waiting for RPC response from device.'
		    WHEN ust.status = 'uploading'   THEN 'Timed out waiting for upload / TransferComplete from device.'
		    WHEN ust.status = 'suspended'   THEN 'Timed out waiting for device to come online.'
		    ELSE                                 'Timed out waiting for TransferComplete from device.'
		END, completed_at = NOW(), updated_at = NOW()
		FROM %[2]s ut
		WHERE ust.task_id = ut.id
		  AND ut.status NOT IN ('pending', 'suspended')
		  AND (
		    (ust.status = 'downloading' AND ust.updated_at < $1)
		    OR (ust.status = 'suspended'  AND ust.updated_at < $2)
		    OR (ust.status NOT IN ('completed', 'failed', 'terminated', 'downloading', 'suspended') AND ust.updated_at < $3)
		  )
			RETURNING ust.task_id, ust.id, ust.device_sn
		)
		SELECT task_id, id, device_sn
		FROM failed`, r.subTaskTable, r.mainTable)

	rows, err := r.pool.Query(ctx, query, rpcCutoff, onlineCutoff, tcCutoff)
	if err != nil {
		return software.StaleFailures{}, fmt.Errorf("fail stale %s: %w", r.subTaskTable, err)
	}
	defer rows.Close()

	var result software.StaleFailures
	for rows.Next() {
		var taskID, subTaskID uuid.UUID
		var deviceSN sql.NullString
		if err := rows.Scan(&taskID, &subTaskID, &deviceSN); err != nil {
			return software.StaleFailures{}, fmt.Errorf("scan stale sub-task: %w", err)
		}
		result.Add(taskID, subTaskID, deviceSN.String)
	}
	if err := rows.Err(); err != nil {
		return software.StaleFailures{}, fmt.Errorf("fail stale %s rows: %w", r.subTaskTable, err)
	}
	return result, nil
}

// UpdateDestVersionByCommandKey 同 software.PgSubTaskRepository 同名方法，作用于本 repo 绑定的物理表。
func (r *PgSubTaskRepo) UpdateDestVersionByCommandKey(ctx context.Context, commandKey, destVersion string) error {
	query, args, err := storage.Psql.Update(r.subTaskTable).
		Set("dest_version", destVersion).
		Where(sq.Eq{"command_key": commandKey}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update %s dest_version SQL: %w", r.subTaskTable, err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update %s dest_version: %w", r.subTaskTable, err)
	}
	return nil
}

// UpdateDestVersionByID 同 software.PgSubTaskRepository 同名方法（qa-614 #371），按子任务 ID 更新本表 dest_version。
func (r *PgSubTaskRepo) UpdateDestVersionByID(ctx context.Context, id uuid.UUID, destVersion string) error {
	query, args, err := storage.Psql.Update(r.subTaskTable).
		Set("dest_version", destVersion).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update %s dest_version by id SQL: %w", r.subTaskTable, err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update %s dest_version by id: %w", r.subTaskTable, err)
	}
	return nil
}

// UpdateFailureReasonByTask fills blank failure_reason values without overwriting
// more specific codes already written by a repo-level timeout branch.
func (r *PgSubTaskRepo) UpdateFailureReasonByTask(ctx context.Context, taskID uuid.UUID, code software.FailureCode) error {
	query, args, err := storage.Psql.Update(r.subTaskTable).
		Set("failure_reason", sq.Expr("COALESCE(NULLIF(failure_reason, ''), ?)", string(code))).
		Where(sq.And{
			sq.Eq{"task_id": taskID},
			sq.Eq{"status": software.UpgradeFailed},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update %s failure_reason SQL: %w", r.subTaskTable, err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update %s failure_reason: %w", r.subTaskTable, err)
	}
	return nil
}

// ListAll 跨主任务列出本业务 sub_task（不分 task_id），JOIN main 表取 task_name。
// 跟 software.PgSubTaskRepository.ListAll 同款 SQL，只是表名参数化。
// 注意：filter.TaskType 对新业务无意义（4 表都属于"日志收集类" 任务类型 10），但 SQL 仍按此字段过滤
// 以保持接口签名一致；新业务行的 task_type 字段值由 BatchCollect 写为 TaskTypeLogCollect=10。
func (r *PgSubTaskRepo) ListAll(ctx context.Context, filter software.AllSubTaskFilter) (*model.ListResponse[software.UpgradeSubTaskWithTaskName], error) {
	cols := make([]string, 0, len(subTaskColumns)+1)
	for _, c := range subTaskColumns {
		cols = append(cols, "ust."+c)
	}
	cols = append(cols, "ut.task_name")

	subAlias := r.subTaskTable + " ust"
	joinExpr := r.mainTable + " ut ON ust.task_id = ut.id"

	base := storage.Psql.Select(cols...).From(subAlias).Join(joinExpr)
	countBase := storage.Psql.Select("COUNT(*)").From(subAlias).Join(joinExpr)

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
		return nil, fmt.Errorf("build count all %s SQL: %w", r.subTaskTable, err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count all %s: %w", r.subTaskTable, err)
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
		return nil, fmt.Errorf("build list all %s SQL: %w", r.subTaskTable, err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all %s: %w", r.subTaskTable, err)
	}
	defer rows.Close()

	items, err := scanSubTaskWithName(rows)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return &model.ListResponse[software.UpgradeSubTaskWithTaskName]{
		Items: items, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages,
	}, nil
}

// scanSubTaskWithName 跟 software 的同名局部逻辑一致，独立提取出来给 List / ListByTaskID 复用。
func scanSubTaskWithName(rows pgx.Rows) ([]software.UpgradeSubTaskWithTaskName, error) {
	var items []software.UpgradeSubTaskWithTaskName
	for rows.Next() {
		var t software.UpgradeSubTaskWithTaskName
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
			jt := software.JSONTime(startedAt.Time)
			t.StartedAt = &jt
		}
		if completedAt.Valid {
			jt := software.JSONTime(completedAt.Time)
			t.CompletedAt = &jt
		}
		if taskName.Valid {
			t.TaskName = taskName.String
		}
		t.CreatedAt = software.JSONTime(createdAt)
		t.UpdatedAt = software.JSONTime(updatedAt)
		items = append(items, t)
	}
	return items, nil
}
