package dashboard

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// Repository 是 PM dashboard / panel / user preferences 持久化接口。
type Repository interface {
	// Dashboard CRUD
	CreateDashboard(ctx context.Context, ownerID uuid.UUID, req CreateDashboardRequest) (*Dashboard, error)
	GetDashboard(ctx context.Context, id uuid.UUID) (*Dashboard, error)
	ListByOwnerOrShared(ctx context.Context, userID uuid.UUID) ([]Dashboard, error)
	UpdateDashboard(ctx context.Context, id uuid.UUID, req UpdateDashboardRequest) error
	DeleteDashboard(ctx context.Context, id uuid.UUID) error

	// Fork：复制 dashboard + panels；parent_dashboard_id 指源
	Fork(ctx context.Context, srcID uuid.UUID, ownerID uuid.UUID, newName string) (*Dashboard, error)

	// Share / Unshare：维护 shared_with 数组
	AddShare(ctx context.Context, dashID uuid.UUID, userIDs []uuid.UUID) error
	RemoveShare(ctx context.Context, dashID uuid.UUID, userID uuid.UUID) error

	// Panel CRUD（按 dashboard_id 隔离）
	CreatePanel(ctx context.Context, req CreatePanelRequest) (*Panel, error)
	GetPanel(ctx context.Context, id uuid.UUID) (*Panel, error)
	ListPanels(ctx context.Context, dashboardID uuid.UUID) ([]Panel, error)
	UpdatePanel(ctx context.Context, id uuid.UUID, req CreatePanelRequest) error
	DeletePanel(ctx context.Context, id uuid.UUID) error

	// UserPreferences
	GetUserPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error)
	UpsertUserPreferences(ctx context.Context, userID uuid.UUID, kpiCardLayout []byte) error
}

// Errors

var (
	ErrNotFound = errors.New("dashboard: not found")
)

// PgRepository 是 Repository 的 pgxpool 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

var _ Repository = (*PgRepository)(nil)

// ── Dashboard ────────────────────────────────────────────────────────────

var dashboardCols = []string{
	"id", "name", "description", "owner_id", "shared_with", "parent_dashboard_id",
	"technology", "layout", "created_at", "updated_at",
}

func (r *PgRepository) CreateDashboard(ctx context.Context, ownerID uuid.UUID, req CreateDashboardRequest) (*Dashboard, error) {
	layout := req.Layout
	if len(layout) == 0 {
		layout = []byte(`{"panels":[]}`)
	}
	q, args, err := storage.Psql.Insert("pm_dashboards").
		Columns("name", "description", "owner_id", "technology", "layout").
		Values(req.Name, req.Description, ownerID, string(req.Technology), layout).
		Suffix(fmt.Sprintf("RETURNING %s", joinCols(dashboardCols, ""))).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("CreateDashboard build: %w", err)
	}
	row := r.pool.QueryRow(ctx, q, args...)
	return scanDashboard(row)
}

func (r *PgRepository) GetDashboard(ctx context.Context, id uuid.UUID) (*Dashboard, error) {
	q, args, err := storage.Psql.Select(dashboardCols...).
		From("pm_dashboards").Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("GetDashboard build: %w", err)
	}
	row := r.pool.QueryRow(ctx, q, args...)
	d, err := scanDashboard(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return d, nil
}

// ListByOwnerOrShared 返当前用户拥有 + 被分享的 dashboard。shared_with 用 GIN 索引快速查。
func (r *PgRepository) ListByOwnerOrShared(ctx context.Context, userID uuid.UUID) ([]Dashboard, error) {
	const q = `
SELECT id, name, description, owner_id, shared_with, parent_dashboard_id,
       technology, layout, created_at, updated_at
FROM pm_dashboards
WHERE owner_id = $1 OR $1 = ANY(shared_with)
ORDER BY updated_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("ListByOwnerOrShared: %w", err)
	}
	defer rows.Close()
	var out []Dashboard
	for rows.Next() {
		d, err := scanDashboard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *PgRepository) UpdateDashboard(ctx context.Context, id uuid.UUID, req UpdateDashboardRequest) error {
	qb := storage.Psql.Update("pm_dashboards").Where(sq.Eq{"id": id})
	if req.Name != nil {
		qb = qb.Set("name", *req.Name)
	}
	if req.Description != nil {
		qb = qb.Set("description", *req.Description)
	}
	if req.Technology != nil {
		qb = qb.Set("technology", string(*req.Technology))
	}
	if len(req.Layout) > 0 {
		qb = qb.Set("layout", req.Layout)
	}
	q, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("UpdateDashboard build: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("UpdateDashboard exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) DeleteDashboard(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM pm_dashboards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeleteDashboard: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Fork 单事务 INSERT 新 dashboard + 复制全部 panels。
func (r *PgRepository) Fork(ctx context.Context, srcID uuid.UUID, ownerID uuid.UUID, newName string) (*Dashboard, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("Fork begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1) 拉源
	src, err := r.getDashboardTx(ctx, tx, srcID)
	if err != nil {
		return nil, err
	}

	// 2) INSERT new dashboard
	var newDash Dashboard
	err = tx.QueryRow(ctx, `
INSERT INTO pm_dashboards (name, description, owner_id, parent_dashboard_id, technology, layout)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, description, owner_id, shared_with, parent_dashboard_id, technology, layout, created_at, updated_at`,
		newName, src.Description, ownerID, srcID, string(src.Technology), src.Layout).
		Scan(&newDash.ID, &newDash.Name, &newDash.Description, &newDash.OwnerID,
			&newDash.SharedWith, &newDash.ParentDashboardID, &newDash.Technology,
			&newDash.Layout, &newDash.CreatedAt, &newDash.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("Fork insert: %w", err)
	}

	// 3) 复制 panels（INSERT ... SELECT 替换 dashboard_id）
	_, err = tx.Exec(ctx, `
INSERT INTO pm_panels (
    dashboard_id, panel_type, title, metric_paths, granularity, dimension,
    device_sns, device_group_ids, time_range, compare_mode, adhoc_task_id, config
)
SELECT
    $1, panel_type, title, metric_paths, granularity, dimension,
    device_sns, device_group_ids, time_range, compare_mode, adhoc_task_id, config
FROM pm_panels WHERE dashboard_id = $2`, newDash.ID, srcID)
	if err != nil {
		return nil, fmt.Errorf("Fork copy panels: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("Fork commit: %w", err)
	}
	return &newDash, nil
}

// getDashboardTx 事务内查 dashboard（Fork 用）。
func (r *PgRepository) getDashboardTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*Dashboard, error) {
	q, args, err := storage.Psql.Select(dashboardCols...).
		From("pm_dashboards").Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}
	row := tx.QueryRow(ctx, q, args...)
	d, err := scanDashboard(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return d, nil
}

// AddShare 把多个用户加到 shared_with（去重 + 跳过 owner）。
func (r *PgRepository) AddShare(ctx context.Context, dashID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	// 用 array_append + DISTINCT 去重
	const q = `
UPDATE pm_dashboards
SET shared_with = (
    SELECT ARRAY(SELECT DISTINCT unnest(shared_with || $2::uuid[]))
)
WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, dashID, userIDs)
	if err != nil {
		return fmt.Errorf("AddShare: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) RemoveShare(ctx context.Context, dashID uuid.UUID, userID uuid.UUID) error {
	const q = `
UPDATE pm_dashboards
SET shared_with = array_remove(shared_with, $2::uuid)
WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, dashID, userID)
	if err != nil {
		return fmt.Errorf("RemoveShare: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Panel ────────────────────────────────────────────────────────────────

var panelCols = []string{
	"id", "dashboard_id", "panel_type", "title", "metric_paths", "granularity",
	"dimension", "device_sns", "device_group_ids", "time_range", "compare_mode",
	"adhoc_task_id", "config", "created_at", "updated_at",
}

func (r *PgRepository) CreatePanel(ctx context.Context, req CreatePanelRequest) (*Panel, error) {
	timeRange := req.TimeRange
	if len(timeRange) == 0 {
		timeRange = []byte(`{}`)
	}
	config := req.Config
	if len(config) == 0 {
		config = []byte(`{}`)
	}
	var compareMode any
	if req.CompareMode != nil {
		compareMode = string(*req.CompareMode)
	}
	var adhocTaskID any
	if req.AdhocTaskID != nil {
		adhocTaskID = *req.AdhocTaskID
	}

	q, args, err := storage.Psql.Insert("pm_panels").
		Columns("dashboard_id", "panel_type", "title", "metric_paths", "granularity",
			"dimension", "device_sns", "device_group_ids", "time_range",
			"compare_mode", "adhoc_task_id", "config").
		Values(req.DashboardID, string(req.PanelType), req.Title, req.MetricPaths, req.Granularity,
			string(req.Dimension), req.DeviceSNs, req.DeviceGroupIDs, timeRange,
			compareMode, adhocTaskID, config).
		Suffix(fmt.Sprintf("RETURNING %s", joinCols(panelCols, ""))).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("CreatePanel build: %w", err)
	}
	row := r.pool.QueryRow(ctx, q, args...)
	return scanPanel(row)
}

func (r *PgRepository) GetPanel(ctx context.Context, id uuid.UUID) (*Panel, error) {
	q, args, err := storage.Psql.Select(panelCols...).
		From("pm_panels").Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, q, args...)
	p, err := scanPanel(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PgRepository) ListPanels(ctx context.Context, dashboardID uuid.UUID) ([]Panel, error) {
	q, args, err := storage.Psql.Select(panelCols...).
		From("pm_panels").Where(sq.Eq{"dashboard_id": dashboardID}).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ListPanels: %w", err)
	}
	defer rows.Close()
	var out []Panel
	for rows.Next() {
		p, err := scanPanel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PgRepository) UpdatePanel(ctx context.Context, id uuid.UUID, req CreatePanelRequest) error {
	timeRange := req.TimeRange
	if len(timeRange) == 0 {
		timeRange = []byte(`{}`)
	}
	config := req.Config
	if len(config) == 0 {
		config = []byte(`{}`)
	}
	var compareMode any
	if req.CompareMode != nil {
		compareMode = string(*req.CompareMode)
	}
	var adhocTaskID any
	if req.AdhocTaskID != nil {
		adhocTaskID = *req.AdhocTaskID
	}
	tag, err := r.pool.Exec(ctx, `
UPDATE pm_panels SET
    panel_type=$2, title=$3, metric_paths=$4, granularity=$5, dimension=$6,
    device_sns=$7, device_group_ids=$8, time_range=$9,
    compare_mode=$10, adhoc_task_id=$11, config=$12
WHERE id=$1`,
		id, string(req.PanelType), req.Title, req.MetricPaths, req.Granularity,
		string(req.Dimension), req.DeviceSNs, req.DeviceGroupIDs, timeRange,
		compareMode, adhocTaskID, config,
	)
	if err != nil {
		return fmt.Errorf("UpdatePanel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) DeletePanel(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM pm_panels WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeletePanel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── UserPreferences ──────────────────────────────────────────────────────

func (r *PgRepository) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	var p UserPreferences
	var layout []byte
	err := r.pool.QueryRow(ctx, `
SELECT user_id, kpi_card_layout, created_at, updated_at
FROM pm_user_dashboard_preferences WHERE user_id = $1`, userID).
		Scan(&p.UserID, &layout, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetUserPreferences: %w", err)
	}
	p.KPICardLayout = layout
	return &p, nil
}

func (r *PgRepository) UpsertUserPreferences(ctx context.Context, userID uuid.UUID, kpiCardLayout []byte) error {
	if len(kpiCardLayout) == 0 {
		kpiCardLayout = []byte(`{}`)
	}
	_, err := r.pool.Exec(ctx, `
INSERT INTO pm_user_dashboard_preferences (user_id, kpi_card_layout)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET kpi_card_layout = EXCLUDED.kpi_card_layout, updated_at = NOW()`,
		userID, kpiCardLayout)
	if err != nil {
		return fmt.Errorf("UpsertUserPreferences: %w", err)
	}
	return nil
}

// ── scan helper ──────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDashboard(row rowScanner) (*Dashboard, error) {
	var d Dashboard
	var description *string
	var layout []byte
	if err := row.Scan(
		&d.ID, &d.Name, &description, &d.OwnerID, &d.SharedWith, &d.ParentDashboardID,
		&d.Technology, &layout, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if description != nil {
		d.Description = *description
	}
	d.Layout = layout
	return &d, nil
}

func scanPanel(row rowScanner) (*Panel, error) {
	var p Panel
	var panelType, dimension string
	var compareMode *string
	var timeRange, config []byte
	if err := row.Scan(
		&p.ID, &p.DashboardID, &panelType, &p.Title, &p.MetricPaths, &p.Granularity,
		&dimension, &p.DeviceSNs, &p.DeviceGroupIDs, &timeRange,
		&compareMode, &p.AdhocTaskID, &config, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	p.PanelType = PanelType(panelType)
	p.Dimension = Dimension(dimension)
	if compareMode != nil {
		m := CompareMode(*compareMode)
		p.CompareMode = &m
	}
	p.TimeRange = timeRange
	p.Config = config
	return &p, nil
}

func joinCols(cols []string, alias string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		if alias != "" {
			out += alias + "."
		}
		out += c
	}
	return out
}
