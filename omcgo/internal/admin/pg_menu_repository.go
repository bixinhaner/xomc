package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// menuColumns 的列顺序必须与 scanMenu / scanMenuFromRows 的 Scan 顺序严格一致。
// name_i18n 由 migration 000083 引入；i18n_key 由 000085 删除。
var menuColumns = []string{
	"id", "name", "name_i18n", "type", "permission_key", "parent_id", "sort_order",
	"route_path", "component_path", "icon", "show_status", "status",
	"created_by", "created_at", "updated_by", "updated_at", "feature_code",
}

// marshalNameI18n 把 map 序列化为 JSONB；nil/空 map 返 nil（写 SQL NULL）。
// 之所以 empty map 也归一为 NULL：方案 C 语义上「未配置多语言」与「配置了空字典」无区别，
// 一律 NULL 让 SELECT 时 fallback 链路（i18n_key → NameI18n → Name）走默认分支。
func marshalNameI18n(m map[string]string) (interface{}, error) {
	if len(m) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal name_i18n: %w", err)
	}
	return b, nil
}

// unmarshalNameI18n 反序列化 JSONB；NULL / 空 bytes 返 nil map。
func unmarshalNameI18n(raw []byte) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("unmarshal name_i18n: %w", err)
	}
	return m, nil
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

	nameI18n, err := marshalNameI18n(menu.NameI18n)
	if err != nil {
		return err
	}

	query, args, err := storage.Psql.Insert("menus").
		Columns(menuColumns...).
		Values(
			menu.ID, menu.Name, nameI18n,
			menu.Type, menu.PermissionKey, nullableUUID(menu.ParentID), menu.SortOrder,
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
	query, args, err := storage.Psql.Select(menuColumns...).
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
	query, args, err := storage.Psql.Select(menuColumns...).
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
	base := storage.Psql.Select(menuColumns...).From("menus")
	countBase := storage.Psql.Select("COUNT(*)").From("menus")

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
	if req.NameI18n != nil {
		// *req.NameI18n 可能是空 map（表示「清空译文」），marshalNameI18n 会归一为 NULL。
		nameI18n, err := marshalNameI18n(*req.NameI18n)
		if err != nil {
			return err
		}
		updates["name_i18n"] = nameI18n
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

	query, args, err := storage.Psql.Update("menus").
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
	query, args, err := storage.Psql.Delete("menus").
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
	query := storage.Psql.Select(menuColumns...).From("menus")

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

// GetAllActive 返回全部 status=MenuStatusNormal 且 show_status='show' 的菜单（含 directory/menu/button），
// 仅供超管旁路使用（user.source='builtIn'）。
// 参 docs/prd/system/menu-dynamic-loading.md §4.2.2 / §设计原则 #4。
func (r *PgMenuRepository) GetAllActive(ctx context.Context) ([]Menu, error) {
	sql, args, err := storage.Psql.Select(menuColumns...).
		From("menus").
		Where(sq.And{
			sq.Eq{"status": MenuStatusNormal},
			sq.Eq{"show_status": MenuShow},
		}).
		OrderBy("parent_id NULLS FIRST, sort_order ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get all active menus SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("get all active menus: %w", err)
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

func (r *PgMenuRepository) GetByRole(ctx context.Context, roleID uuid.UUID) ([]Menu, error) {
	query, args, err := storage.Psql.Select("m.id", "m.name", "m.name_i18n",
		"m.type", "m.permission_key", "m.parent_id",
		"m.sort_order", "m.route_path", "m.component_path", "m.icon", "m.show_status", "m.status",
		"m.created_by", "m.created_at", "m.updated_by", "m.updated_at", "m.feature_code").
		From("menus m").
		Join("role_menus rm ON m.id = rm.menu_id").
		Where(sq.And{
			sq.Eq{"rm.role_id": roleID},
			sq.Eq{"m.status": MenuStatusNormal},
			sq.Eq{"m.show_status": MenuShow},
		}).
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
	// 注：曾有 r.status='active' 过滤，但 roles 表从来没有 status 列（migrations
	// 000002 + 000056 都没加），是 dead-code filter；已删除并连带去掉只为它服务
	// 的 roles JOIN。如未来需要"禁用角色"概念，请先加 ALTER TABLE roles ADD COLUMN status。
	query, args, err := storage.Psql.Select("DISTINCT m.id", "m.name", "m.name_i18n",
		"m.type", "m.permission_key", "m.parent_id",
		"m.sort_order", "m.route_path", "m.component_path", "m.icon", "m.show_status", "m.status",
		"m.created_by", "m.created_at", "m.updated_by", "m.updated_at", "m.feature_code").
		From("menus m").
		Join("role_menus rm ON m.id = rm.menu_id").
		Join("user_roles ur ON ur.role_id = rm.role_id").
		Where(sq.And{
			sq.Eq{"ur.user_id": userID},
			sq.Eq{"m.status": MenuStatusNormal},
			sq.Eq{"m.show_status": MenuShow},
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
	query, args, err := storage.Psql.Select("menu_id").
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
//
// Scan 顺序与 menuColumns 严格对齐；name_i18n 用 *[]byte 接 JSONB 原始字节，
// 再走 unmarshalNameI18n 解码。NULL → nil bytes → nil map（前端兜底到 Name）。
func scanMenu(row pgx.Row) (*Menu, error) {
	var m Menu
	var parentID, createdBy, updatedBy *uuid.UUID
	var routePath, componentPath, icon *string
	var nameI18nRaw []byte

	err := row.Scan(
		&m.ID, &m.Name, &nameI18nRaw,
		&m.Type, &m.PermissionKey, &parentID, &m.SortOrder,
		&routePath, &componentPath, &icon, &m.ShowStatus, &m.Status,
		&createdBy, &m.CreatedAt, &updatedBy, &m.UpdatedAt, &m.FeatureCodes,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan menu: %w", err)
	}

	if err := applyMenuOptionalCols(&m, parentID, createdBy, updatedBy, routePath, componentPath, icon, nameI18nRaw); err != nil {
		return nil, err
	}
	return &m, nil
}

// scanMenuFromRows scans a single menu from a pgx.Rows.
func scanMenuFromRows(rows pgx.Rows) (*Menu, error) {
	var m Menu
	var parentID, createdBy, updatedBy *uuid.UUID
	var routePath, componentPath, icon *string
	var nameI18nRaw []byte

	err := rows.Scan(
		&m.ID, &m.Name, &nameI18nRaw,
		&m.Type, &m.PermissionKey, &parentID, &m.SortOrder,
		&routePath, &componentPath, &icon, &m.ShowStatus, &m.Status,
		&createdBy, &m.CreatedAt, &updatedBy, &m.UpdatedAt, &m.FeatureCodes,
	)
	if err != nil {
		return nil, fmt.Errorf("scan menu row: %w", err)
	}

	if err := applyMenuOptionalCols(&m, parentID, createdBy, updatedBy, routePath, componentPath, icon, nameI18nRaw); err != nil {
		return nil, err
	}
	return &m, nil
}

// applyMenuOptionalCols 把 scan 出的可空列回填到 Menu，
// 避免 scanMenu / scanMenuFromRows 两份重复代码。
func applyMenuOptionalCols(
	m *Menu,
	parentID, createdBy, updatedBy *uuid.UUID,
	routePath, componentPath, icon *string,
	nameI18nRaw []byte,
) error {
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
	nameI18n, err := unmarshalNameI18n(nameI18nRaw)
	if err != nil {
		return err
	}
	m.NameI18n = nameI18n
	return nil
}

// buildTree builds a tree structure from a flat list of menus.
//
// 历史 bug：早期实现按 parent.Children = append(..., m) 一遍循环装配，因 m 是值拷贝
// 且 append 当时父子双方 Children 都还未填好，只能装出 2 层，第 3 层（按钮）整体丢失。
// 现改为「按 parent_id 收集 → 自顶向下递归 build」，保证每层 Children 在被装入父节点
// 之前已经填充完毕。
func (r *PgMenuRepository) buildTree(flat []Menu) []Menu {
	return assembleMenuTree(flat)
}

// assembleMenuTree 是包级 helper，方便 service 层共用。
func assembleMenuTree(flat []Menu) []Menu {
	if len(flat) == 0 {
		return nil
	}
	byID := make(map[uuid.UUID]Menu, len(flat))
	childIDs := make(map[uuid.UUID][]uuid.UUID)
	var rootIDs []uuid.UUID

	for _, m := range flat {
		m.Children = nil
		byID[m.ID] = m
		if m.ParentID == nil {
			rootIDs = append(rootIDs, m.ID)
		} else {
			childIDs[*m.ParentID] = append(childIDs[*m.ParentID], m.ID)
		}
	}

	var build func(id uuid.UUID) Menu
	build = func(id uuid.UUID) Menu {
		node := byID[id]
		for _, cid := range childIDs[id] {
			node.Children = append(node.Children, build(cid))
		}
		return node
	}

	roots := make([]Menu, 0, len(rootIDs))
	for _, id := range rootIDs {
		roots = append(roots, build(id))
	}
	return roots
}

func nullableUUID(u *uuid.UUID) interface{} {
	if u == nil {
		return nil
	}
	return *u
}
