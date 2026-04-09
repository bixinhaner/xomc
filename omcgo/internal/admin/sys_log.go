package admin

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/model"
)

// --- Models ---

// LoginLog represents a login event log.
type LoginLog struct {
	ID        int64       `json:"id"`
	UserID    *uuid.UUID  `json:"user_id,omitempty"`
	Username  string      `json:"username"`
	IPAddress string      `json:"ip_address"`
	Location  string      `json:"location"`
	Browser   string      `json:"browser"`
	OS        string      `json:"os"`
	Status    bool        `json:"status"`
	Message   string      `json:"message"`
	LoginAt   time.Time   `json:"login_at"`
}

// OperLog represents an operation log.
type OperLog struct {
	ID        int64       `json:"id"`
	UserID    *uuid.UUID  `json:"user_id,omitempty"`
	Username  string      `json:"username"`
	Action    string      `json:"action"`
	Module    string      `json:"module"`
	Target    string      `json:"target"`
	Detail    string      `json:"detail"`
	IPAddress string      `json:"ip_address"`
	UserAgent string      `json:"user_agent"`
	Status    bool        `json:"status"`
	ErrorMsg  string      `json:"error_msg"`
	CostMs    int         `json:"cost_ms"`
	CreatedAt time.Time   `json:"created_at"`
}

// TaskLog represents a task execution log.
type TaskLog struct {
	ID         int64       `json:"id"`
	TaskType   string      `json:"task_type"`
	TaskID     string      `json:"task_id"`
	Status     string      `json:"status"`
	OperatorID *uuid.UUID  `json:"operator_id,omitempty"`
	Operator   string      `json:"operator"`
	Target     string      `json:"target"`
	Detail     string      `json:"detail"`
	ErrorMsg   string      `json:"error_msg"`
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt *time.Time  `json:"finished_at,omitempty"`
	CostMs     int         `json:"cost_ms"`
}

// LoginLogFilter provides filtering for login logs.
type LoginLogFilter struct {
	Username  *string `form:"username"`
	IPAddress *string `form:"ip_address"`
	Status    *bool   `form:"status"`
	StartTime *string `form:"start_time"`
	EndTime   *string `form:"end_time"`
	model.ListRequest
}

// OperLogFilter provides filtering for operation logs.
type OperLogFilter struct {
	Username  *string `form:"username"`
	Action    *string `form:"action"`
	Module    *string `form:"module"`
	Status    *bool   `form:"status"`
	StartTime *string `form:"start_time"`
	EndTime   *string `form:"end_time"`
	model.ListRequest
}

// TaskLogFilter provides filtering for task logs.
type TaskLogFilter struct {
	TaskType  *string `form:"task_type"`
	Status    *string `form:"status"`
	Operator  *string `form:"operator"`
	StartTime *string `form:"start_time"`
	EndTime   *string `form:"end_time"`
	model.ListRequest
}

// CreateLoginLogRequest is used to record a login event.
type CreateLoginLogRequest struct {
	UserID    *uuid.UUID `json:"user_id"`
	Username  string     `json:"username"`
	IPAddress string     `json:"ip_address"`
	Browser   string     `json:"browser"`
	OS        string     `json:"os"`
	Status    bool       `json:"status"`
	Message   string     `json:"message"`
}

// CreateOperLogRequest is used to record an operation event.
type CreateOperLogRequest struct {
	UserID    *uuid.UUID `json:"user_id"`
	Username  string     `json:"username"`
	Action    string     `json:"action"`
	Module    string     `json:"module"`
	Target    string     `json:"target"`
	Detail    string     `json:"detail"`
	IPAddress string     `json:"ip_address"`
	UserAgent string     `json:"user_agent"`
	Status    bool       `json:"status"`
	ErrorMsg  string     `json:"error_msg"`
	CostMs    int        `json:"cost_ms"`
}

// CreateTaskLogRequest is used to record a task event.
type CreateTaskLogRequest struct {
	TaskType   string     `json:"task_type"`
	TaskID     string     `json:"task_id"`
	Status     string     `json:"status"`
	OperatorID *uuid.UUID `json:"operator_id"`
	Operator   string     `json:"operator"`
	Target     string     `json:"target"`
	Detail     string     `json:"detail"`
	ErrorMsg   string     `json:"error_msg"`
	CostMs     int        `json:"cost_ms"`
}

// --- LogRepository ---

// LogRepository defines the persistence interface for system logs.
type LogRepository interface {
	CreateLoginLog(ctx context.Context, req CreateLoginLogRequest) error
	ListLoginLogs(ctx context.Context, filter LoginLogFilter) (*model.ListResponse[LoginLog], error)

	CreateOperLog(ctx context.Context, req CreateOperLogRequest) error
	ListOperLogs(ctx context.Context, filter OperLogFilter) (*model.ListResponse[OperLog], error)

	CreateTaskLog(ctx context.Context, req CreateTaskLogRequest) error
	ListTaskLogs(ctx context.Context, filter TaskLogFilter) (*model.ListResponse[TaskLog], error)
}

// PgLogRepository implements LogRepository using PostgreSQL.
type PgLogRepository struct {
	pool *pgxpool.Pool
}

var _ LogRepository = (*PgLogRepository)(nil)

// NewPgLogRepository creates a new PgLogRepository.
func NewPgLogRepository(pool *pgxpool.Pool) *PgLogRepository {
	return &PgLogRepository{pool: pool}
}

func (r *PgLogRepository) CreateLoginLog(ctx context.Context, req CreateLoginLogRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sys_login_logs (user_id, username, ip_address, browser, os, status, message, login_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		req.UserID, req.Username, req.IPAddress, req.Browser, req.OS, req.Status, req.Message,
	)
	return err
}

func (r *PgLogRepository) ListLoginLogs(ctx context.Context, filter LoginLogFilter) (*model.ListResponse[LoginLog], error) {
	where := sq.And{}
	if filter.Username != nil && *filter.Username != "" {
		where = append(where, sq.Expr("username ILIKE ?", "%"+*filter.Username+"%"))
	}
	if filter.IPAddress != nil && *filter.IPAddress != "" {
		where = append(where, sq.Eq{"ip_address": *filter.IPAddress})
	}
	if filter.Status != nil {
		where = append(where, sq.Eq{"status": *filter.Status})
	}
	if filter.StartTime != nil && *filter.StartTime != "" {
		where = append(where, sq.GtOrEq{"login_at": *filter.StartTime})
	}
	if filter.EndTime != nil && *filter.EndTime != "" {
		where = append(where, sq.LtOrEq{"login_at": *filter.EndTime})
	}

	return listPaginated[LoginLog](ctx, r.pool, "sys_login_logs",
		[]string{"id", "user_id", "username", "ip_address", "location", "browser", "os", "status", "message", "login_at"},
		where, "login_at DESC", &filter.ListRequest,
		func(scan func(...interface{}) error, item *LoginLog) error {
			return scan(&item.ID, &item.UserID, &item.Username, &item.IPAddress, &item.Location, &item.Browser, &item.OS, &item.Status, &item.Message, &item.LoginAt)
		},
	)
}

func (r *PgLogRepository) CreateOperLog(ctx context.Context, req CreateOperLogRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sys_oper_logs (user_id, username, action, module, target, detail, ip_address, user_agent, status, error_msg, cost_ms, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())`,
		req.UserID, req.Username, req.Action, req.Module, req.Target, req.Detail, req.IPAddress, req.UserAgent, req.Status, req.ErrorMsg, req.CostMs,
	)
	return err
}

func (r *PgLogRepository) ListOperLogs(ctx context.Context, filter OperLogFilter) (*model.ListResponse[OperLog], error) {
	where := sq.And{}
	if filter.Username != nil && *filter.Username != "" {
		where = append(where, sq.Expr("username ILIKE ?", "%"+*filter.Username+"%"))
	}
	if filter.Action != nil && *filter.Action != "" {
		where = append(where, sq.Eq{"action": *filter.Action})
	}
	if filter.Module != nil && *filter.Module != "" {
		where = append(where, sq.Eq{"module": *filter.Module})
	}
	if filter.Status != nil {
		where = append(where, sq.Eq{"status": *filter.Status})
	}
	if filter.StartTime != nil && *filter.StartTime != "" {
		where = append(where, sq.GtOrEq{"created_at": *filter.StartTime})
	}
	if filter.EndTime != nil && *filter.EndTime != "" {
		where = append(where, sq.LtOrEq{"created_at": *filter.EndTime})
	}

	return listPaginated[OperLog](ctx, r.pool, "sys_oper_logs",
		[]string{"id", "user_id", "username", "action", "module", "target", "detail", "ip_address", "user_agent", "status", "error_msg", "cost_ms", "created_at"},
		where, "created_at DESC", &filter.ListRequest,
		func(scan func(...interface{}) error, item *OperLog) error {
			return scan(&item.ID, &item.UserID, &item.Username, &item.Action, &item.Module, &item.Target, &item.Detail, &item.IPAddress, &item.UserAgent, &item.Status, &item.ErrorMsg, &item.CostMs, &item.CreatedAt)
		},
	)
}

func (r *PgLogRepository) CreateTaskLog(ctx context.Context, req CreateTaskLogRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sys_task_logs (task_type, task_id, status, operator_id, operator, target, detail, error_msg, started_at, finished_at, cost_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NULL, $9)`,
		req.TaskType, req.TaskID, req.Status, req.OperatorID, req.Operator, req.Target, req.Detail, req.ErrorMsg, req.CostMs,
	)
	return err
}

func (r *PgLogRepository) ListTaskLogs(ctx context.Context, filter TaskLogFilter) (*model.ListResponse[TaskLog], error) {
	where := sq.And{}
	if filter.TaskType != nil && *filter.TaskType != "" {
		where = append(where, sq.Eq{"task_type": *filter.TaskType})
	}
	if filter.Status != nil && *filter.Status != "" {
		where = append(where, sq.Eq{"status": *filter.Status})
	}
	if filter.Operator != nil && *filter.Operator != "" {
		where = append(where, sq.Expr("operator ILIKE ?", "%"+*filter.Operator+"%"))
	}
	if filter.StartTime != nil && *filter.StartTime != "" {
		where = append(where, sq.GtOrEq{"started_at": *filter.StartTime})
	}
	if filter.EndTime != nil && *filter.EndTime != "" {
		where = append(where, sq.LtOrEq{"started_at": *filter.EndTime})
	}

	return listPaginated[TaskLog](ctx, r.pool, "sys_task_logs",
		[]string{"id", "task_type", "task_id", "status", "operator_id", "operator", "target", "detail", "error_msg", "started_at", "finished_at", "cost_ms"},
		where, "started_at DESC", &filter.ListRequest,
		func(scan func(...interface{}) error, item *TaskLog) error {
			return scan(&item.ID, &item.TaskType, &item.TaskID, &item.Status, &item.OperatorID, &item.Operator, &item.Target, &item.Detail, &item.ErrorMsg, &item.StartedAt, &item.FinishedAt, &item.CostMs)
		},
	)
}

// listPaginated is a generic helper for paginated list queries.
func listPaginated[T any](ctx context.Context, pool *pgxpool.Pool, table string, columns []string, where sq.And, orderBy string, lr *model.ListRequest, scanRow func(func(...interface{}) error, *T) error) (*model.ListResponse[T], error) {
	// Count
	countQ, countA, err := psql.Select("COUNT(*)").From(table).Where(where).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count %s SQL: %w", table, err)
	}
	var total int64
	if err := pool.QueryRow(ctx, countQ, countA...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count %s: %w", table, err)
	}

	offset := lr.Offset()
	limit := lr.Limit()

	cols := make([]string, len(columns))
	copy(cols, columns)
	selQ, selA, err := psql.Select(cols...).From(table).Where(where).OrderBy(orderBy).Limit(uint64(limit)).Offset(uint64(offset)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list %s SQL: %w", table, err)
	}

	rows, err := pool.Query(ctx, selQ, selA...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", table, err)
	}
	defer rows.Close()

	var items []T
	for rows.Next() {
		var item T
		if err := scanRow(rows.Scan, &item); err != nil {
			return nil, fmt.Errorf("scan %s: %w", table, err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []T{}
	}
	return model.NewListResponse(items, total, lr.Page, lr.PageSize), rows.Err()
}
