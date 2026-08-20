package notification

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/core/storage"
)

const (
	EmailRunBusinessAlarm = "alarm"
	EmailRunBusinessKPI   = "kpi"
)

// EmailRunHistory is the common, read-only projection of an alarm aggregation
// run or a KPI regular-report run. The business tables remain the source of
// truth; this type only gives operators one place to inspect delivery results.
type EmailRunHistory struct {
	ID             uuid.UUID  `json:"id"`
	BusinessType   string     `json:"business_type"`
	TemplateName   string     `json:"template_name"`
	Period         string     `json:"period,omitempty"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	JobAttempt     int        `json:"job_attempt"`
	WindowStart    time.Time  `json:"window_start"`
	WindowEnd      time.Time  `json:"window_end"`
	Status         string     `json:"status"`
	Subject        string     `json:"subject,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	RecipientCount int        `json:"recipient_count"`
	SentCount      int        `json:"sent_count"`
	FailedCount    int        `json:"failed_count"`
	TotalAttempts  int        `json:"total_attempts"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type EmailRunDeliveryHistory struct {
	ID        uuid.UUID  `json:"id"`
	Recipient string     `json:"recipient"`
	Status    string     `json:"status"`
	Attempt   int        `json:"attempt"`
	LastError string     `json:"last_error,omitempty"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type EmailRunHistoryFilter struct {
	BusinessType string
	Status       string
	CallerID     uuid.UUID
	IsSuperAdmin bool
	model.ListRequest
}

type EmailRunHistoryReader interface {
	ListEmailRuns(context.Context, EmailRunHistoryFilter) (*model.ListResponse[EmailRunHistory], error)
	ListEmailRunDeliveries(context.Context, string, uuid.UUID, uuid.UUID, bool) ([]EmailRunDeliveryHistory, error)
}

type PgEmailRunHistoryReader struct {
	pool *pgxpool.Pool
}

func NewPgEmailRunHistoryReader(pool *pgxpool.Pool) *PgEmailRunHistoryReader {
	return &PgEmailRunHistoryReader{pool: pool}
}

func (r *PgEmailRunHistoryReader) ListEmailRuns(ctx context.Context, filter EmailRunHistoryFilter) (*model.ListResponse[EmailRunHistory], error) {
	wanted := filter.Offset() + filter.Limit()
	items := make([]EmailRunHistory, 0, wanted*2)
	var total int64

	if filter.BusinessType == "" || filter.BusinessType == EmailRunBusinessAlarm {
		alarmItems, alarmTotal, err := r.listAlarmEmailRuns(ctx, filter, wanted)
		if err != nil {
			return nil, err
		}
		items = append(items, alarmItems...)
		total += alarmTotal
	}
	if filter.BusinessType == "" || filter.BusinessType == EmailRunBusinessKPI {
		kpiItems, kpiTotal, err := r.listKPIEmailRuns(ctx, filter, wanted)
		if err != nil {
			return nil, err
		}
		items = append(items, kpiItems...)
		total += kpiTotal
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ScheduledAt.After(items[j].ScheduledAt)
	})
	start := filter.Offset()
	if start > len(items) {
		start = len(items)
	}
	end := start + filter.Limit()
	if end > len(items) {
		end = len(items)
	}
	return model.NewListResponse(items[start:end], total, filter.Page, filter.PageSize), nil
}

func (r *PgEmailRunHistoryReader) listAlarmEmailRuns(ctx context.Context, filter EmailRunHistoryFilter, limit int) ([]EmailRunHistory, int64, error) {
	base := storage.Psql.Select(
		"r.id", "s.name", "j.scheduled_at", "j.started_at", "j.finished_at", "j.attempt", "r.window_start", "r.window_end",
		"r.status", "COALESCE(r.subject, '')", "COALESCE(r.last_error, '')",
		"(SELECT COUNT(*) FROM alarm_email_deliveries d WHERE d.run_id = r.id)",
		"(SELECT COUNT(*) FROM alarm_email_deliveries d WHERE d.run_id = r.id AND d.status = 'sent')",
		"(SELECT COUNT(*) FROM alarm_email_deliveries d WHERE d.run_id = r.id AND d.status = 'failed')",
		"(SELECT COALESCE(SUM(d.attempt), 0) FROM alarm_email_deliveries d WHERE d.run_id = r.id)",
		"r.created_at", "r.updated_at",
	).From("alarm_email_runs r").
		Join("alarm_email_subscriptions s ON s.id = r.subscription_id").
		Join("async_jobs j ON j.id = r.async_job_id")
	countBase := storage.Psql.Select("COUNT(*)").From("alarm_email_runs r").Join("alarm_email_subscriptions s ON s.id = r.subscription_id")
	base, countBase = addEmailRunFilter(base, countBase, filter, "s.created_by")
	total, err := r.countEmailRuns(ctx, countBase)
	if err != nil {
		return nil, 0, fmt.Errorf("count alarm email runs: %w", err)
	}
	query, args, err := base.OrderBy("j.scheduled_at DESC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build alarm email runs query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list alarm email runs: %w", err)
	}
	defer rows.Close()
	items := make([]EmailRunHistory, 0)
	for rows.Next() {
		item := EmailRunHistory{BusinessType: EmailRunBusinessAlarm}
		if err := rows.Scan(&item.ID, &item.TemplateName, &item.ScheduledAt, &item.StartedAt, &item.FinishedAt, &item.JobAttempt, &item.WindowStart, &item.WindowEnd,
			&item.Status, &item.Subject, &item.LastError, &item.RecipientCount, &item.SentCount,
			&item.FailedCount, &item.TotalAttempts, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan alarm email run: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate alarm email runs: %w", err)
	}
	return items, total, nil
}

func (r *PgEmailRunHistoryReader) listKPIEmailRuns(ctx context.Context, filter EmailRunHistoryFilter, limit int) ([]EmailRunHistory, int64, error) {
	base := storage.Psql.Select(
		"r.id", "r.template_name", "r.period", "j.scheduled_at", "j.started_at", "j.finished_at", "j.attempt", "r.window_start", "r.window_end",
		"r.status", "COALESCE(r.subject, '')", "COALESCE(r.last_error, '')",
		"(SELECT COUNT(*) FROM pm_kpi_report_deliveries d WHERE d.run_id = r.id)",
		"(SELECT COUNT(*) FROM pm_kpi_report_deliveries d WHERE d.run_id = r.id AND d.status = 'sent')",
		"(SELECT COUNT(*) FROM pm_kpi_report_deliveries d WHERE d.run_id = r.id AND d.status = 'failed')",
		"(SELECT COALESCE(SUM(d.attempt), 0) FROM pm_kpi_report_deliveries d WHERE d.run_id = r.id)",
		"r.created_at", "r.updated_at",
	).From("pm_kpi_report_runs r").Join("async_jobs j ON j.id = r.async_job_id")
	countBase := storage.Psql.Select("COUNT(*)").From("pm_kpi_report_runs r")
	base, countBase = addEmailRunFilter(base, countBase, filter, "r.creator_id")
	total, err := r.countEmailRuns(ctx, countBase)
	if err != nil {
		return nil, 0, fmt.Errorf("count KPI email runs: %w", err)
	}
	query, args, err := base.OrderBy("j.scheduled_at DESC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build KPI email runs query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list KPI email runs: %w", err)
	}
	defer rows.Close()
	items := make([]EmailRunHistory, 0)
	for rows.Next() {
		item := EmailRunHistory{BusinessType: EmailRunBusinessKPI}
		if err := rows.Scan(&item.ID, &item.TemplateName, &item.Period, &item.ScheduledAt, &item.StartedAt, &item.FinishedAt, &item.JobAttempt, &item.WindowStart, &item.WindowEnd,
			&item.Status, &item.Subject, &item.LastError, &item.RecipientCount, &item.SentCount,
			&item.FailedCount, &item.TotalAttempts, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan KPI email run: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate KPI email runs: %w", err)
	}
	return items, total, nil
}

func addEmailRunFilter(base, count sq.SelectBuilder, filter EmailRunHistoryFilter, ownerColumn string) (sq.SelectBuilder, sq.SelectBuilder) {
	if !filter.IsSuperAdmin {
		base = base.Where(sq.Eq{ownerColumn: filter.CallerID})
		count = count.Where(sq.Eq{ownerColumn: filter.CallerID})
	}
	if filter.Status != "" {
		base = base.Where(sq.Eq{"r.status": filter.Status})
		count = count.Where(sq.Eq{"r.status": filter.Status})
	}
	return base, count
}

func (r *PgEmailRunHistoryReader) countEmailRuns(ctx context.Context, builder sq.SelectBuilder) (int64, error) {
	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build email run count query: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("query email run count: %w", err)
	}
	return total, nil
}

func (r *PgEmailRunHistoryReader) ListEmailRunDeliveries(ctx context.Context, business string, runID, callerID uuid.UUID, isSuperAdmin bool) ([]EmailRunDeliveryHistory, error) {
	var builder sq.SelectBuilder
	switch business {
	case EmailRunBusinessAlarm:
		builder = storage.Psql.Select("d.id", "d.recipient", "d.status", "d.attempt", "COALESCE(d.last_error, '')", "d.sent_at", "d.updated_at").
			From("alarm_email_deliveries d").
			Join("alarm_email_runs r ON r.id = d.run_id").
			Join("alarm_email_subscriptions s ON s.id = r.subscription_id").
			Where(sq.Eq{"d.run_id": runID})
		if !isSuperAdmin {
			builder = builder.Where(sq.Eq{"s.created_by": callerID})
		}
	case EmailRunBusinessKPI:
		builder = storage.Psql.Select("d.id", "d.recipient", "d.status", "d.attempt", "COALESCE(d.last_error, '')", "d.sent_at", "d.updated_at").
			From("pm_kpi_report_deliveries d").
			Join("pm_kpi_report_runs r ON r.id = d.run_id").
			Where(sq.Eq{"d.run_id": runID})
		if !isSuperAdmin {
			builder = builder.Where(sq.Eq{"r.creator_id": callerID})
		}
	default:
		return nil, commonerrors.ErrInvalidInput
	}
	query, args, err := builder.OrderBy("lower(d.recipient)").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build email run deliveries query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list email run deliveries: %w", err)
	}
	defer rows.Close()
	items := make([]EmailRunDeliveryHistory, 0)
	for rows.Next() {
		var item EmailRunDeliveryHistory
		if err := rows.Scan(&item.ID, &item.Recipient, &item.Status, &item.Attempt, &item.LastError, &item.SentAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan email run delivery: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate email run deliveries: %w", err)
	}
	return items, nil
}

type EmailRunHistoryHandler struct {
	reader EmailRunHistoryReader
}

func NewEmailRunHistoryHandler(reader EmailRunHistoryReader) *EmailRunHistoryHandler {
	return &EmailRunHistoryHandler{reader: reader}
}

func (h *EmailRunHistoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	runs := rg.Group("/email-runs")
	runs.GET("", h.List)
	runs.GET("/:business/:id/deliveries", h.ListDeliveries)
}

type emailRunListQuery struct {
	BusinessType string `form:"business_type"`
	Status       string `form:"status"`
	model.ListRequest
}

func (h *EmailRunHistoryHandler) List(c *gin.Context) {
	query := emailRunListQuery{ListRequest: model.DefaultListRequest()}
	if err := c.ShouldBindQuery(&query); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if query.BusinessType != "" && query.BusinessType != EmailRunBusinessAlarm && query.BusinessType != EmailRunBusinessKPI {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if query.Status != "" && !validEmailRunStatus(query.Status) {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	callerID, ok := admin.UserIDFromCtx(c)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return
	}
	result, err := h.reader.ListEmailRuns(c.Request.Context(), EmailRunHistoryFilter{
		BusinessType: query.BusinessType,
		Status:       query.Status,
		CallerID:     callerID,
		IsSuperAdmin: emailRunCallerIsSuperAdmin(c),
		ListRequest:  query.ListRequest,
	})
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *EmailRunHistoryHandler) ListDeliveries(c *gin.Context) {
	business := strings.ToLower(c.Param("business"))
	if business != EmailRunBusinessAlarm && business != EmailRunBusinessKPI {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	callerID, ok := admin.UserIDFromCtx(c)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return
	}
	items, err := h.reader.ListEmailRunDeliveries(c.Request.Context(), business, runID, callerID, emailRunCallerIsSuperAdmin(c))
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func emailRunCallerIsSuperAdmin(c *gin.Context) bool {
	value, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuperAdmin, _ := value.(bool)
	return isSuperAdmin
}

func validEmailRunStatus(status string) bool {
	switch status {
	case "pending", "processing", "sent", "partial_failed", "failed":
		return true
	default:
		return false
	}
}
