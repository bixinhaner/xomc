package admin

import (
	"context"
	"fmt"
	"math"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

var menuColumns = []string{
	"id", "name", "type", "permission_key", "parent_id", "sort_order",
	"route_path", "component_path", "icon", "show_status", "status",
	"created_by", "created_at", "updated_by", "updated_at",
}

// PgMenuRepository implements MenuRepository using PostgreSQL.
type PgMenuRepository struct {
	pool *pgxpool.Pool
}

var _ MenuRepository = (*PgMenuRepository)(nil)

// NewPgMenuRepository creates a new PgMenuRepository.
func NewPgMenuRepository(pool *pgxpool.Pool) *PgMenuRepository {
	return &PgMenuRepository{pool: pool}
}

func (r *PgMenuRepository) Create(ctx context.Context, menu *Menu, operatorID uuid.UUID) error {
	menu.ID = uuid.New()
	menu.CreatedBy = &operatorID
	menu.CreatedAt = time.Now()
	menu.UpdatedBy = &operatorID
	menu.UpdatedAt = time.Now()

	query, args, err := psql.Insert("menus").
		Columns(menuColumns...).
		Values(
			menu.ID, menu.Name, menu.Type, menu.PermissionKey, nullableUUID(menu.ParentID), menu.SortOrder,
			nullableString(menu.RoutePath), nullableString(menu.ComponentPath), nullableString(menu.Icon),
			menu.ShowStatus, menu.Status,
			menu.CreatedBy, menu.CreatedAt, menu.UpdatedBy, menu.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert menu SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert menu: %w", err)
	}
	return nil
}

func (r *PgMenuRepository) GetByID(ctx context.Context, id uuid.UUID) (*Menu, error) {
	query, args, err := psql.Select(menuColumns...).
		From("menus").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get menu SQL: %w", err)
	}

	menu, err := scanMenu(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return menu, nil
}

func (r *PgMenuRepository) GetByPermissionKey(ctx context.Context, key string) (*Menu, error) {
	query, args, err := psql.Select(menuColumns...).
		From("menus").
		Where(sq.Eq{"permission_key": key}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get menu by permission key SQL: %w", err)
	}

	menu, err := scanMenu(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return menu, nil
}

func (r *PgMenuRepository) List(ctx context.Context, filter MenuFilter) (*model.ListResponse[Menu], error) {
	base := psql.Select(menuColumns...).From("menus")
	countBase := psql.Select("COUNT(*)").From("menus")

	// Apply filters
	if filter.Type != nil {
		base = base.Where(sq.Eq{"type": *filter.Type})
		countBase = countBase.Where(sq.Eq{"type": *filter.Type})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.ParentID != nil {
		if *filter.ParentID == uuid.Nil {
			base = base.Where("parent_id IS NULL")
			countBase = countBase.Where("parent_id IS NULL")
		} else {
			base = base.Where(sq.Eq{"parent_id": *filter.ParentID})
			countBase = countBase.Where(sq.Eq{"parent_id": *filter.ParentID})
		}
	}

	// Count
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count menus SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count menus: %w", err)
	}

	// Pagination
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
		OrderBy("sort_order ASC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list menus SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list menus: %w", err)
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		m, err := scanMenuFromRows(rows)
		if err != nil {
			return nil, err
		}
		menus = append(menus, *m)
	}

	return &model.ListResponse[Menu]{
		Items:      menus,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	}, nil
}

func (r *PgMenuRepository) Update(ctx context.Context, id uuid.UUID, req *UpdateMenuRequest, operatorID uuid.UUID) error {
	updates := map[string]interface{}{}

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.RoutePath != nil {
		updates["route_path"] = nullableString(*req.RoutePath)
	}
	if req.Icon != nil {
		updates["icon"] = nullableString(*req.Icon)
	}
	if req.ShowStatus != nil {
		updates["show_status"] = *req.ShowStatus
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	updates["updated_by"] = operatorID
	updates["updated_at"] = time.Now()

	query, args, err := psql.Update("menus").
		SetMap(updates).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update menu SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update menu: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgMenuRepository) Delete(ctx context.Context, ids []uuid.UUID) error {
	query, args, err := psql.Delete("menus").
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete menus SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete menus: %w", err)
	}
	return nil
}

func (r *PgMenuRepository) GetTree(ctx context.Context, status *MenuStatus) ([]Menu, error) {
	query := psql.Select(menuColumns...).From("menus")

	if status != nil {
		query = query.Where(sq.Eq{"status": *status})
	}

	query = query.OrderBy("parent_id NULLS FIRST, sort_order ASC")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get menu tree SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("get menu tree: %w", err)
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		m, err := scanMenuFromRows(rows)
		if err != nil {
			return nil, err
		}
		menus = append(menus, *m)
	}

	return r.buildTree(menus), nil
}

func (r *PgMenuRepository) GetByRole(ctx context.Context, roleID uuid.UUID) ([]Menu, error) {
	query, args, err := psql.Select("m.id", "m.name", "m.type", "m.permission_key", "m.parent_id",
		"m.sort_order", "m.route_path", "m.component_path", "m.icon", "m.show_status", "m.status",
		"m.created_by", "m.created_at", "m.updated_by", "m.updated_at").
		From("menus m").
		Join("role_menus rm ON m.id = rm.menu_id").
		Where(sq.Eq{"rm.role_id": roleID}).
		OrderBy("m.sort_order ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get menus by role SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get menus by role: %w", err)
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		m, err := scanMenuFromRows(rows)
		if err != nil {
			return nil, err
		}
		menus = append(menus, *m)
	}

	return menus, nil
}

func (r *PgMenuRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]Menu, error) {
	query, args, err := psql.Select("DISTINCT m.id", "m.name", "m.type", "m.permission_key", "m.parent_id",
		"m.sort_order", "m.route_path", "m.component_path", "m.icon", "m.show_status", "m.status",
		"m.created_by", "m.created_at", "m.updated_by", "m.updated_at").
		From("menus m").
		Join("role_menus rm ON m.id = rm.menu_id").
		Join("user_roles ur ON ur.role_id = rm.role_id").
		Join("roles r ON r.id = ur.role_id").
		Where(sq.And{
			sq.Eq{"ur.user_id": userID},
			sq.Eq{"r.status": "active"},
			sq.Eq{"m.status": "normal"},
		}).
		OrderBy("m.sort_order ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get menus by user SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get menus by user: %w", err)
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		m, err := scanMenuFromRows(rows)
		if err != nil {
			return nil, err
		}
		menus = append(menus, *m)
	}

	return menus, nil
}

func (r *PgMenuRepository) SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete old associations
	_, err = tx.Exec(ctx, "DELETE FROM role_menus WHERE role_id = $1", roleID)
	if err != nil {
		return fmt.Errorf("delete old role menus: %w", err)
	}

	// Insert new associations
	if len(menuIDs) > 0 {
		now := time.Now()
		for _, menuID := range menuIDs {
			_, err = tx.Exec(ctx,
				"INSERT INTO role_menus (role_id, menu_id, created_by, created_at) VALUES ($1, $2, $3, $4)",
				roleID, menuID, operatorID, now)
			if err != nil {
				return fmt.Errorf("insert role menu: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (r *PgMenuRepository) GetRoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := psql.Select("menu_id").
		From("role_menus").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get role menu IDs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get role menu IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan menu ID: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// scanMenu scans a single menu from a pgx.Row.
func scanMenu(row pgx.Row) (*Menu, error) {
	var m Menu
	var parentID, createdBy, updatedBy *uuid.UUID
	var routePath, componentPath, icon *string

	err := row.Scan(
		&m.ID, &m.Name, &m.Type, &m.PermissionKey, &parentID, &m.SortOrder,
		&routePath, &componentPath, &icon, &m.ShowStatus, &m.Status,
		&createdBy, &m.CreatedAt, &updatedBy, &m.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan menu: %w", err)
	}

	m.ParentID = parentID
	m.CreatedBy = createdBy
	m.UpdatedBy = updatedBy
	if routePath != nil {
		m.RoutePath = *routePath
	}
	if componentPath != nil {
		m.ComponentPath = *componentPath
	}
	if icon != nil {
		m.Icon = *icon
	}

	return &m, nil
}

// scanMenuFromRows scans a single menu from a pgx.Rows.
func scanMenuFromRows(rows pgx.Rows) (*Menu, error) {
	var m Menu
	var parentID, createdBy, updatedBy *uuid.UUID
	var routePath, componentPath, icon *string

	err := rows.Scan(
		&m.ID, &m.Name, &m.Type, &m.PermissionKey, &parentID, &m.SortOrder,
		&routePath, &componentPath, &icon, &m.ShowStatus, &m.Status,
		&createdBy, &m.CreatedAt, &updatedBy, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan menu row: %w", err)
	}

	m.ParentID = parentID
	m.CreatedBy = createdBy
	m.UpdatedBy = updatedBy
	if routePath != nil {
		m.RoutePath = *routePath
	}
	if componentPath != nil {
		m.ComponentPath = *componentPath
	}
	if icon != nil {
		m.Icon = *icon
	}

	return &m, nil
}

// buildTree builds a tree structure from a flat list of menus.
func (r *PgMenuRepository) buildTree(flat []Menu) []Menu {
	menuMap := make(map[uuid.UUID]*Menu)
	var roots []Menu

	for i := range flat {
		menuMap[flat[i].ID] = &flat[i]
		flat[i].Children = nil
	}

	for _, m := range flat {
		if m.ParentID == nil {
			roots = append(roots, m)
		} else if parent, ok := menuMap[*m.ParentID]; ok {
			parent.Children = append(parent.Children, m)
		}
	}

	return roots
}

func nullableUUID(u *uuid.UUID) interface{} {
	if u == nil {
		return nil
	}
	return *u
}
