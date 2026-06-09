# RBAC 用户-角色-菜单系统修改方案

> 基于简化设计：菜单即权限，删除冗余的 permissions 表
> 前后端数据传递统一使用驼峰命名（camelCase）
> **后端 JSON 序列化直接输出 camelCase，无需前端转换**

---

## 重要：现有模型已更新

`internal/admin/model.go` 中的现有模型已全部更新为 camelCase JSON 标签：

| 原标签 | 新标签 |
|-------|-------|
| `json:"display_name"` | `json:"displayName"` |
| `json:"device_group_ids"` | `json:"deviceGroupIds"` |
| `json:"is_system"` | `json:"isSystem"` |
| `json:"role_id"` | `json:"roleId"` |
| `json:"role_ids"` | `json:"roleIds"` |
| `json:"failed_login_attempts"` | `json:"failedLoginAttempts"` |
| `json:"locked_until"` | `json:"lockedUntil"` |
| `json:"last_failed_login_at"` | `json:"lastFailedLoginAt"` |
| `json:"last_login_at"` | `json:"lastLoginAt"` |
| `json:"created_at"` | `json:"createdAt"` |
| `json:"updated_at"` | `json:"updatedAt"` |
| `json:"access_token"` | `json:"accessToken"` |
| `json:"refresh_token"` | `json:"refreshToken"` |
| `json:"expires_at"` | `json:"expiresAt"` |
| `json:"token_type"` | `json:"tokenType"` |
| `json:"user_id"` | `json:"userId"` |
| `json:"resource_id"` | `json:"resourceId"` |
| `json:"ip_address"` | `json:"ipAddress"` |
| `json:"user_agent"` | `json:"userAgent"` |
| `json:"captcha_id"` | `json:"captchaId"` |
| `json:"captcha_answer"` | `json:"captchaAnswer"` |
| `json:"refresh_token"` | `json:"refreshToken"` |
| `json:"new_password"` | `json:"newPassword"` |

**注意**：`form` 标签保持 snake_case（如 `form:"user_id"`），因为 URL 查询参数通常使用 snake_case。

---

## 一、后端修改方案

### 1.1 目录结构

```
internal/admin/
├── model.go              # 添加 Menu 模型
├── menu_repository.go    # 新增：菜单仓储
├── menu_service.go       # 新增：菜单服务
├── menu_handler.go       # 新增：菜单处理器
└── router/
    └── admin_router.go   # 注册菜单路由
```

### 1.2 模型定义 (model.go)

```go
// ==================== 菜单相关 ====================

// MenuType 菜单类型
type MenuType string

const (
    MenuTypeDirectory MenuType = "directory"
    MenuTypeMenu      MenuType = "menu"
    MenuTypeButton    MenuType = "button"
)

// MenuStatus 菜单状态
type MenuStatus string

const (
    MenuStatusNormal   MenuStatus = "normal"
    MenuStatusDisabled MenuStatus = "disabled"
)

// MenuShowStatus 显示状态
type MenuShowStatus string

const (
    MenuShowStatusShow MenuShowStatus = "show"
    MenuShowStatusHide MenuShowStatus = "hide"
)

// Menu 菜单实体
type Menu struct {
    ID            uuid.UUID      `json:"id"`
    Name          string         `json:"name"`
    Type          MenuType       `json:"type"`
    PermissionKey string         `json:"permissionKey"`
    ParentID      *uuid.UUID     `json:"parentId,omitempty"`
    SortOrder     int            `json:"sortOrder"`
    
    // 路由
    RoutePath     string         `json:"routePath,omitempty"`
    ComponentPath string         `json:"componentPath,omitempty"`
    
    // 显示
    Icon          string         `json:"icon,omitempty"`
    ShowStatus    MenuShowStatus `json:"showStatus"`
    
    // 状态
    Status        MenuStatus     `json:"status"`
    
    // 审计字段
    CreatedBy     *uuid.UUID     `json:"createdBy,omitempty"`
    CreatedAt     time.Time      `json:"createdAt"`
    UpdatedBy     *uuid.UUID     `json:"updatedBy,omitempty"`
    UpdatedAt     time.Time      `json:"updatedAt"`
    
    // 关联（不序列化）
    Children      []Menu         `json:"children,omitempty"`
    CreatedByUser *UserSummary  `json:"createdByUser,omitempty"`
    UpdatedByUser *UserSummary  `json:"updatedByUser,omitempty"`
}

// UserSummary 用户摘要（用于审计字段）
type UserSummary struct {
    ID       uuid.UUID `json:"id"`
    Username string     `json:"username"`
}

// MenuTreeNode 菜单树节点（用于前端渲染）
type MenuTreeNode struct {
    ID            string         `json:"id"`
    Name          string         `json:"name"`
    Type          MenuType       `json:"type"`
    PermissionKey string         `json:"permissionKey"`
    RoutePath     string         `json:"routePath,omitempty"`
    Icon          string         `json:"icon,omitempty"`
    Children      []MenuTreeNode `json:"children,omitempty"`
}

// CreateMenuRequest 创建菜单请求
type CreateMenuRequest struct {
    Name          string         `json:"name" binding:"required,max=64"`
    Type          MenuType       `json:"type" binding:"required"`
    PermissionKey string         `json:"permissionKey" binding:"required,max=128"`
    ParentID      *uuid.UUID     `json:"parentId"`
    SortOrder     int            `json:"sortOrder"`
    RoutePath     string         `json:"routePath"`
    Icon          string         `json:"icon"`
    ShowStatus    MenuShowStatus `json:"showStatus"`
    ComponentPath string         `json:"componentPath"`
}

// UpdateMenuRequest 更新菜单请求
type UpdateMenuRequest struct {
    Name       *string         `json:"name"`
    SortOrder  *int            `json:"sortOrder"`
    RoutePath  *string         `json:"routePath"`
    Icon       *string         `json:"icon"`
    ShowStatus *MenuShowStatus `json:"showStatus"`
    Status     *MenuStatus     `json:"status"`
}

// MenuFilter 菜单查询条件
type MenuFilter struct {
    Type     *MenuType   `form:"type"`
    Status   *MenuStatus `form:"status"`
    ParentID *uuid.UUID  `form:"parentId"`
    model.ListRequest
}

// ==================== 角色扩展 ====================

// 更新 Role 模型，添加菜单字段
type Role struct {
    ID             uuid.UUID    `json:"id"`
    Name           string       `json:"name"`
    Description    string       `json:"description"`
    IsSystem       bool         `json:"isSystem"`
    Status         RoleStatus   `json:"status"`
    
    // 数据权限
    DeviceGroupIDs []uuid.UUID  `json:"deviceGroupIds,omitempty"`
    
    // 菜单权限（新增）
    MenuIDs        []uuid.UUID  `json:"menuIds,omitempty"`
    Menus          []Menu       `json:"menus,omitempty"`
    
    // 统计
    UserCount      int          `json:"userCount,omitempty"`
    
    // 审计字段
    CreatedBy      *uuid.UUID   `json:"createdBy,omitempty"`
    CreatedAt      time.Time    `json:"createdAt"`
    UpdatedBy      *uuid.UUID   `json:"updatedBy,omitempty"`
    UpdatedAt      time.Time    `json:"updatedAt"`
}

// UpdateRoleRequest 添加菜单字段
type UpdateRoleRequest struct {
    Name            *string           `json:"name"`
    Description     *string           `json:"description"`
    Status          *RoleStatus       `json:"status"`
    DeviceGroupIDs  []uuid.UUID       `json:"deviceGroupIds"`
    MenuIDs         []uuid.UUID       `json:"menuIds"` // 新增
}

// SetRoleMenusRequest 设置角色菜单
type SetRoleMenusRequest struct {
    MenuIDs []uuid.UUID `json:"menuIds" binding:"required"`
}
```

### 1.3 菜单仓储 (menu_repository.go)

```go
package admin

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "github.com/Masterminds/squirrel"
)

type menuRepository struct {
    db  *sqlx.DB
    sb  squirrel.StatementBuilderType
}

func NewMenuRepository(db *sqlx.DB) MenuRepository {
    return &menuRepository{
        db: db,
        sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
    }
}

type MenuRepository interface {
    Create(ctx context.Context, menu *Menu, operatorID uuid.UUID) error
    GetByID(ctx context.Context, id uuid.UUID) (*Menu, error)
    GetByPermissionKey(ctx context.Context, key string) (*Menu, error)
    List(ctx context.Context, filter MenuFilter) ([]*Menu, int64, error)
    Update(ctx context.Context, id uuid.UUID, req *UpdateMenuRequest, operatorID uuid.UUID) error
    Delete(ctx context.Context, ids []uuid.UUID) error
    
    // 树形结构
    GetTree(ctx context.Context, status *MenuStatus) ([]*Menu, error)
    GetChildren(ctx context.Context, parentID *uuid.UUID) ([]*Menu, error)
    
    // 角色关联
    GetByRole(ctx context.Context, roleID uuid.UUID) ([]*Menu, error)
    GetByUser(ctx context.Context, userID uuid.UUID) ([]*Menu, error)
    
    // 设置角色菜单
    SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error
    GetRoleMenus(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
}

func (r *menuRepository) Create(ctx context.Context, menu *Menu, operatorID uuid.UUID) error {
    menu.ID = uuid.New()
    menu.CreatedBy = &operatorID
    menu.CreatedAt = time.Now()
    menu.UpdatedBy = &operatorID
    menu.UpdatedAt = time.Now()
    
    query := r.sb.Insert("menus").
        SetMap(map[string]interface{}{
            "id":             menu.ID,
            "name":           menu.Name,
            "type":           menu.Type,
            "permission_key": menu.PermissionKey,
            "parent_id":       menu.ParentID,
            "sort_order":     menu.SortOrder,
            "route_path":     menu.RoutePath,
            "component_path": menu.ComponentPath,
            "icon":           menu.Icon,
            "show_status":    menu.ShowStatus,
            "status":         menu.Status,
            "created_by":     menu.CreatedBy,
            "created_at":     menu.CreatedAt,
            "updated_by":     menu.UpdatedBy,
            "updated_at":     menu.UpdatedAt,
        })
    
    sql, args, _ := query.ToSql()
    _, err := r.db.ExecContext(ctx, sql, args...)
    return err
}

func (r *menuRepository) GetByID(ctx context.Context, id uuid.UUID) (*Menu, error) {
    query := r.sb.Select("*").
        From("menus").
        Where(squirrel.Eq{"id": id})
    
    return r.findOne(ctx, query)
}

func (r *menuRepository) List(ctx context.Context, filter MenuFilter) ([]*Menu, int64, error) {
    query := r.sb.Select("*").From("menus")
    
    // 筛选条件
    if filter.Type != nil {
        query = query.Where(squirrel.Eq{"type": *filter.Type})
    }
    if filter.Status != nil {
        query = query.Where(squirrel.Eq{"status": *filter.Status})
    }
    if filter.ParentID != nil {
        if *filter.ParentID == uuid.Nil {
            query = query.Where(squirrel.IsNull{"parent_id"})
        } else {
            query = query.Where(squirrel.Eq{"parent_id": *filter.ParentID})
        }
    }
    
    // 总数
    countQuery := r.sb.Select("COUNT(*)").From("menus")
    if filter.Type != nil {
        countQuery = countQuery.Where(squirrel.Eq{"type": *filter.Type})
    }
    if filter.Status != nil {
        countQuery = countQuery.Where(squirrel.Eq{"status": *filter.Status})
    }
    if filter.ParentID != nil {
        if *filter.ParentID == uuid.Nil {
            countQuery = countQuery.Where(squirrel.IsNull{"parent_id"})
        } else {
            countQuery = countQuery.Where(squirrel.Eq{"parent_id": *filter.ParentID})
        }
    }
    
    // 分页
    if filter.PageSize > 0 {
        query = query.Limit(filter.PageSize).Offset((filter.Page - 1) * filter.PageSize)
    }
    
    query = query.OrderBy("sort_order ASC")
    
    // 执行
    menus := []*Menu{}
    sql, args, _ := query.ToSql()
    if err := r.db.SelectContext(ctx, &menus, sql, args...); err != nil {
        return nil, 0, err
    }
    
    // 计数
    var total int64
    countSql, countArgs, _ := countQuery.ToSql()
    if err := r.db.GetContext(ctx, &total, countSql, countArgs...); err != nil {
        return nil, 0, err
    }
    
    // 填充审计字段
    for _, m := range menus {
        r.fillAuditFields(ctx, m)
    }
    
    return menus, total, nil
}

func (r *menuRepository) Update(ctx context.Context, id uuid.UUID, req *UpdateMenuRequest, operatorID uuid.UUID) error {
    updates := make(map[string]interface{})
    
    if req.Name != nil {
        updates["name"] = *req.Name
    }
    if req.SortOrder != nil {
        updates["sort_order"] = *req.SortOrder
    }
    if req.RoutePath != nil {
        updates["route_path"] = *req.RoutePath
    }
    if req.Icon != nil {
        updates["icon"] = *req.Icon
    }
    if req.ShowStatus != nil {
        updates["show_status"] = *req.ShowStatus
    }
    if req.Status != nil {
        updates["status"] = *req.Status
    }
    
    updates["updated_by"] = operatorID
    updates["updated_at"] = time.Now()
    
    query := r.sb.Update("menus").
        SetMap(updates).
        Where(squirrel.Eq{"id": id})
    
    sql, args, _ := query.ToSql()
    _, err := r.db.ExecContext(ctx, sql, args...)
    return err
}

func (r *menuRepository) Delete(ctx context.Context, ids []uuid.UUID) error {
    query := r.sb.Delete("menus").
        Where(squirrel.Eq{"id": ids})
    
    sql, args, _ := query.ToSql()
    _, err := r.db.ExecContext(ctx, sql, args...)
    return err
}

func (r *menuRepository) GetTree(ctx context.Context, status *MenuStatus) ([]*Menu, error) {
    query := r.sb.Select("*").From("menus")
    
    if status != nil {
        query = query.Where(squirrel.Eq{"status": *status})
    }
    
    query = query.OrderBy("parent_id NULLS FIRST, sort_order ASC")
    
    sql, args, _ := query.ToSql()
    menus := []*Menu{}
    if err := r.db.SelectContext(ctx, &menus, sql, args...); err != nil {
        return nil, err
    }
    
    // 构建树形结构
    return r.buildTree(menus), nil
}

func (r *menuRepository) buildTree(flat []*Menu) []*Menu {
    menuMap := make(map[uuid.UUID]*Menu)
    var roots []*Menu
    
    for _, m := range flat {
        menuMap[m.ID] = m
        m.Children = nil // 清空
    }
    
    for _, m := range flat {
        if m.ParentID == nil {
            roots = append(roots, m)
        } else {
            if parent, ok := menuMap[*m.ParentID]; ok {
                parent.Children = append(parent.Children, m)
            }
        }
    }
    
    return roots
}

func (r *menuRepository) GetByRole(ctx context.Context, roleID uuid.UUID) ([]*Menu, error) {
    query := r.sb.Select("m.*").
        From("menus m").
        Join("role_menus rm ON m.id = rm.menu_id").
        Where(squirrel.Eq{"rm.role_id": roleID}).
        OrderBy("m.sort_order ASC")
    
    sql, args, _ := query.ToSql()
    menus := []*Menu{}
    if err := r.db.SelectContext(ctx, &menus, sql, args...); err != nil {
        return nil, err
    }
    
    return menus, nil
}

func (r *menuRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*Menu, error) {
    query := r.sb.Select("DISTINCT m.*").
        From("menus m").
        Join("role_menus rm ON m.id = rm.menu_id").
        Join("user_roles ur ON ur.role_id = rm.role_id").
        Join("roles r ON r.id = ur.role_id").
        Where(squirrel.And{
            squirrel.Eq{"ur.user_id": userID},
            squirrel.Eq{"r.status": "active"},
            squirrel.Eq{"m.status": "normal"},
        }).
        OrderBy("m.sort_order ASC")
    
    sql, args, _ := query.ToSql()
    menus := []*Menu{}
    if err := r.db.SelectContext(ctx, &menus, sql, args...); err != nil {
        return nil, err
    }
    
    return menus, nil
}

func (r *menuRepository) SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 删除旧的关联
    _, err = tx.ExecContext(ctx, "DELETE FROM role_menus WHERE role_id = $1", roleID)
    if err != nil {
        return err
    }
    
    // 插入新的关联
    if len(menuIDs) > 0 {
        insert := r.sb.Insert("role_menus").
            Columns("role_id", "menu_id", "created_by", "created_at")
        
        for _, menuID := range menuIDs {
            insert = insert.Values(roleID, menuID, operatorID, time.Now())
        }
        
        sql, args, _ := insert.ToSql()
        _, err = tx.ExecContext(ctx, sql, args...)
        if err != nil {
            return err
        }
    }
    
    return tx.Commit()
}

func (r *menuRepository) GetRoleMenus(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
    query := r.sb.Select("menu_id").
        From("role_menus").
        Where(squirrel.Eq{"role_id": roleID})
    
    sql, args, _ := query.ToSql()
    ids := []uuid.UUID{}
    if err := r.db.SelectContext(ctx, &ids, sql, args...); err != nil {
        return nil, err
    }
    
    return ids, nil
}

// 辅助方法
func (r *menuRepository) findOne(ctx context.Context, query squirrel.SelectBuilder) (*Menu, error) {
    sql, args, _ := query.ToSql()
    menu := &Menu{}
    if err := r.db.GetContext(ctx, menu, sql, args...); err != nil {
        return nil, err
    }
    return menu, nil
}

func (r *menuRepository) fillAuditFields(ctx context.Context, m *Menu) {
    if m.CreatedBy != nil {
        user, _ := r.getUserSummary(ctx, *m.CreatedBy)
        m.CreatedByUser = user
    }
    if m.UpdatedBy != nil {
        user, _ := r.getUserSummary(ctx, *m.UpdatedBy)
        m.UpdatedByUser = user
    }
}

func (r *menuRepository) getUserSummary(ctx context.Context, userID uuid.UUID) (*UserSummary, error) {
    query := r.sb.Select("id", "username").
        From("users").
        Where(squirrel.Eq{"id": userID})
    
    sql, args, _ := query.ToSql()
    user := &UserSummary{}
    err := r.db.GetContext(ctx, user, sql, args...)
    return user, err
}
```

### 1.4 菜单服务 (menu_service.go)

```go
package admin

import (
    "context"
    "fmt"

    "github.com/google/uuid"
)

type menuService struct {
    repo   MenuRepository
    roleRepo RoleRepository
}

func NewMenuService(repo MenuRepository, roleRepo RoleRepository) MenuService {
    return &menuService{
        repo:   repo,
        roleRepo: roleRepo,
    }
}

type MenuService interface {
    CreateMenu(ctx context.Context, req *CreateMenuRequest, operatorID uuid.UUID) (*Menu, error)
    GetMenu(ctx context.Context, id uuid.UUID) (*Menu, error)
    ListMenus(ctx context.Context, filter MenuFilter) ([]*Menu, int64, error)
    UpdateMenu(ctx context.Context, id uuid.UUID, req *UpdateMenuRequest, operatorID uuid.UUID) error
    DeleteMenus(ctx context.Context, ids []uuid.UUID, operatorID uuid.UUID) error
    
    GetMenuTree(ctx context.Context, status *MenuStatus) ([]*Menu, error)
    GetUserMenuTree(ctx context.Context, userID uuid.UUID) ([]*Menu, error)
    
    // 角色菜单
    SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error
    GetRoleMenus(ctx context.Context, roleID uuid.UUID) ([]*Menu, error)
}

func (s *menuService) CreateMenu(ctx context.Context, req *CreateMenuRequest, operatorID uuid.UUID) (*Menu, error) {
    // 检查权限键是否已存在
    existing, _ := s.repo.GetByPermissionKey(ctx, req.PermissionKey)
    if existing != nil {
        return nil, fmt.Errorf("权限标识 %s 已存在", req.PermissionKey)
    }
    
    // 检查父菜单是否存在
    if req.ParentID != nil {
        parent, err := s.repo.GetByID(ctx, *req.ParentID)
        if err != nil {
            return nil, fmt.Errorf("父菜单不存在")
        }
        if parent.Type != MenuTypeDirectory && parent.Type != MenuTypeMenu {
            return nil, fmt.Errorf("父菜单必须是目录或菜单类型")
        }
    }
    
    menu := &Menu{
        Name:          req.Name,
        Type:          req.Type,
        PermissionKey: req.PermissionKey,
        ParentID:      req.ParentID,
        SortOrder:     req.SortOrder,
        RoutePath:     req.RoutePath,
        ComponentPath: req.ComponentPath,
        Icon:          req.Icon,
        ShowStatus:    req.ShowStatus,
        Status:        MenuStatusNormal,
    }
    
    if err := s.repo.Create(ctx, menu, operatorID); err != nil {
        return nil, err
    }
    
    return s.repo.GetByID(ctx, menu.ID)
}

func (s *menuService) GetMenu(ctx context.Context, id uuid.UUID) (*Menu, error) {
    return s.repo.GetByID(ctx, id)
}

func (s *menuService) ListMenus(ctx context.Context, filter MenuFilter) ([]*Menu, int64, error) {
    return s.repo.List(ctx, filter)
}

func (s *menuService) UpdateMenu(ctx context.Context, id uuid.UUID, req *UpdateMenuRequest, operatorID uuid.UUID) error {
    // 检查菜单是否存在
    menu, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return fmt.Errorf("菜单不存在")
    }
    
    // 内置菜单只能修改部分字段
    // TODO: 添加内置菜单的检查逻辑
    
    return s.repo.Update(ctx, id, req, operatorID)
}

func (s *menuService) DeleteMenus(ctx context.Context, ids []uuid.UUID, operatorID uuid.UUID) error {
    // TODO: 检查是否有关联的角色
    return s.repo.Delete(ctx, ids)
}

func (s *menuService) GetMenuTree(ctx context.Context, status *MenuStatus) ([]*Menu, error) {
    return s.repo.GetTree(ctx, status)
}

func (s *menuService) GetUserMenuTree(ctx context.Context, userID uuid.UUID) ([]*Menu, error) {
    menus, err := s.repo.GetByUser(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // 构建树形结构并过滤按钮
    tree := s.repo.buildTree(menus)
    return s.filterTreeForDisplay(tree), nil
}

func (s *menuService) SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error {
    return s.repo.SetRoleMenus(ctx, roleID, menuIDs, operatorID)
}

func (s *menuService) GetRoleMenus(ctx context.Context, roleID uuid.UUID) ([]*Menu, error) {
    menuIDs, err := s.repo.GetRoleMenus(ctx, roleID)
    if err != nil {
        return nil, err
    }
    
    var menus []*Menu
    for _, id := range menuIDs {
        menu, err := s.repo.GetByID(ctx, id)
        if err != nil {
            continue
        }
        menus = append(menus, menu)
    }
    
    return menus, nil
}

// 辅助方法：过滤树，只显示目录和菜单（不显示按钮）
func (s *menuService) filterTreeForDisplay(nodes []*Menu) []*Menu {
    var result []*Menu
    for _, node := range nodes {
        if node.Type == MenuTypeButton {
            continue
        }
        
        filtered := &Menu{
            ID:            node.ID,
            Name:          node.Name,
            Type:          node.Type,
            PermissionKey: node.PermissionKey,
            SortOrder:     node.SortOrder,
            RoutePath:     node.RoutePath,
            Icon:          node.Icon,
            ShowStatus:    node.ShowStatus,
            Status:        node.Status,
        }
        
        if len(node.Children) > 0 {
            filtered.Children = s.filterTreeForDisplay(node.Children)
        }
        
        result = append(result, filtered)
    }
    return result
}
```

### 1.5 菜单处理器 (menu_handler.go)

```go
package admin

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type menuHandler struct {
    service MenuService
}

func NewMenuHandler(service MenuService) *menuHandler {
    return &menuHandler{service: service}
}

func (h *menuHandler) RegisterRoutes(r *gin.RouterGroup) {
    r.GET("/menus", h.ListMenus)
    r.GET("/menus/tree", h.GetMenuTree)
    r.GET("/menus/user", h.GetUserMenuTree)
    r.GET("/menus/:id", h.GetMenu)
    r.POST("/menus", h.CreateMenu)
    r.PUT("/menus/:id", h.UpdateMenu)
    r.DELETE("/menus", h.DeleteMenus)
}

// CreateMenu 创建菜单
// @Summary 创建菜单
// @Tags 菜单管理
func (h *menuHandler) CreateMenu(c *gin.Context) {
    var req CreateMenuRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    operatorID := GetOperatorID(c) // 从 JWT 获取
    
    menu, err := h.service.CreateMenu(c.Request.Context(), &req, operatorID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, gin.H{"data": menu})
}

// ListMenus 菜单列表
// @Summary 菜单列表
// @Tags 菜单管理
func (h *menuHandler) ListMenus(c *gin.Context) {
    var filter MenuFilter
    if err := c.ShouldBindQuery(&filter); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    menus, total, err := h.service.ListMenus(c.Request.Context(), filter)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "data": gin.H{
            "items":     menus,
            "total":     total,
            "page":      filter.Page,
            "pageSize":  filter.PageSize,
        },
    })
}

// GetMenuTree 获取菜单树
// @Summary 菜单树
// @Tags 菜单管理
func (h *menuHandler) GetMenuTree(c *gin.Context) {
    status := c.Query("status")
    
    var menuStatus *MenuStatus
    if status != "" {
        s := MenuStatus(status)
        menuStatus = &s
    }
    
    menus, err := h.service.GetMenuTree(c.Request.Context(), menuStatus)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"data": menus})
}

// GetUserMenuTree 获取当前用户菜单树
// @Summary 用户菜单树
// @Tags 菜单管理
func (h *menuHandler) GetUserMenuTree(c *gin.Context) {
    userID := GetUserID(c) // 从 JWT 获取
    
    menus, err := h.service.GetUserMenuTree(c.Request.Context(), userID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"data": menus})
}

// GetMenu 获取菜单详情
// @Summary 菜单详情
// @Tags 菜单管理
func (h *menuHandler) GetMenu(c *gin.Context) {
    id := c.Param("id")
    menuID, err := uuid.Parse(id)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的菜单ID"})
        return
    }
    
    menu, err := h.service.GetMenu(c.Request.Context(), menuID)
    if err != nil {
        c.JSON(404, gin.H{"error": "菜单不存在"})
        return
    }
    
    c.JSON(200, gin.H{"data": menu})
}

// UpdateMenu 更新菜单
// @Summary 更新菜单
// @Tags 菜单管理
func (h *menuHandler) UpdateMenu(c *gin.Context) {
    id := c.Param("id")
    menuID, err := uuid.Parse(id)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的菜单ID"})
        return
    }
    
    var req UpdateMenuRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    operatorID := GetOperatorID(c)
    
    if err := h.service.UpdateMenu(c.Request.Context(), menuID, &req, operatorID); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"data": gin.H{"success": true}})
}

// DeleteMenus 删除菜单
// @Summary 删除菜单
// @Tags 菜单管理
func (h *menuHandler) DeleteMenus(c *gin.Context) {
    var req struct {
        IDs []string `json:"ids" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    ids := make([]uuid.UUID, len(req.IDs))
    for i, idStr := range req.IDs {
        id, err := uuid.Parse(idStr)
        if err != nil {
            c.JSON(400, gin.H{"error": "无效的菜单ID"})
            return
        }
        ids[i] = id
    }
    
    operatorID := GetOperatorID(c)
    
    if err := h.service.DeleteMenus(c.Request.Context(), ids, operatorID); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"data": gin.H{"success": true}})
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) uuid.UUID {
    if userID, exists := c.Get("user_id"); exists {
        return userID.(uuid.UUID)
    }
    return uuid.Nil
}

// GetOperatorID 从上下文获取操作人ID
func GetOperatorID(c *gin.Context) uuid.UUID {
    return GetUserID(c)
}
```

### 1.6 路由注册 (router/admin_router.go)

```go
// 注册菜单路由
menuHandler := NewMenuHandler(menuService)
menuGroup := v1.Group("/menus")
menuHandler.RegisterRoutes(menuGroup)

// 角色菜单路由
v1.POST("/roles/:id/menus", roleHandler.SetMenus)
v1.GET("/roles/:id/menus", roleHandler.GetMenus)
```

### 1.7 权限中间件更新

```go
// MenuPermissionMiddleware 菜单权限检查中间件
func MenuPermissionMiddleware(permissionKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := GetUserID(c)
        if userID == uuid.Nil {
            c.JSON(401, gin.H{"error": "未授权"})
            c.Abort()
            return
        }
        
        // 检查是否为超级管理员
        if isAdmin(c) {
            c.Next()
            return
        }
        
        // 检查菜单权限
        if !hasMenuPermission(c.Request.Context(), userID, permissionKey) {
            c.JSON(403, gin.H{"error": "无权限"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}

func hasMenuPermission(ctx context.Context, userID uuid.UUID, permissionKey string) bool {
    // 获取用户的所有菜单
    menus, err := menuService.GetUserMenuTree(ctx, userID)
    if err != nil {
        return false
    }
    
    // 检查权限
    return checkPermission(menus, permissionKey)
}

func checkPermission(menus []*Menu, permissionKey string) bool {
    for _, menu := range menus {
        // 精确匹配
        if menu.PermissionKey == permissionKey {
            return true
        }
        
        // 前缀匹配（如 device:list 匹配 device:list:query）
        if strings.HasPrefix(permissionKey, menu.PermissionKey+":") {
            return true
        }
        
        // 递归检查子菜单
        if len(menu.Children) > 0 {
            if checkPermission(menu.Children, permissionKey) {
                return true
            }
        }
    }
    
    return false
}
```

---

## 二、前端修改方案

### 2.1 类型定义 (types/rbac.ts)

```typescript
// ==================== 用户 ====================

export type UserStatus = 'active' | 'disabled';

export interface User {
  id: string;
  username: string;
  displayName: string;
  email?: string;
  carrier?: 'cmcc' | 'ctcc' | 'cucc';
  status: UserStatus;
  lastLoginAt?: string;
  
  // 审计
  createdBy?: string;
  createdAt: string;
  updatedBy?: string;
  updatedAt?: string;
  
  // 关联
  roles?: Role[];
  roleIds?: string[];
  
  // 扩展
  builtIn: number;  // 从 roles 推断
}

// ==================== 角色 ====================

export type RoleStatus = 'active' | 'disabled';

export interface Role {
  id: string;
  name: string;
  description?: string;
  isSystem: boolean;
  status: RoleStatus;
  
  // 数据权限
  deviceGroupIds?: string[];
  
  // 菜单权限
  menuIds?: string[];
  menus?: MenuItem[];
  
  // 统计
  userCount?: number;
  
  // 审计
  createdBy?: string;
  createdAt: string;
  updatedBy?: string;
  updatedAt?: string;
  
  // 扩展
  builtIn: number;  // is_system 的数字形式
}

// ==================== 菜单 ====================

export type MenuType = 'directory' | 'menu' | 'button';
export type MenuStatus = 'normal' | 'disabled';
export type MenuShowStatus = 'show' | 'hide';

export interface MenuItem {
  id: string;
  name: string;
  type: MenuType;
  permissionKey: string;
  parentId?: string | null;
  sort_order: number;
  
  // 路由
  routePath?: string;
  icon?: string;
  showStatus: MenuShowStatus;
  componentPath?: string;
  
  // 状态
  status: MenuStatus;
  
  // 审计
  createdBy?: string;
  createdAt: string;
  updatedBy?: string;
  updatedAt?: string;
  
  // 树形
  children?: MenuItem[];
}

// 菜单树节点（用于前端渲染）
export interface MenuTreeNode {
  id: string;
  name: string;
  type: MenuType;
  permissionKey: string;
  routePath?: string;
  icon?: string;
  children?: MenuTreeNode[];
}

// ==================== 请求/响应 ====================

export interface CreateMenuRequest {
  name: string;
  type: MenuType;
  permissionKey: string;
  parentId?: string | null;
  sortOrder: number;
  routePath?: string;
  icon?: string;
  showStatus?: MenuShowStatus;
  componentPath?: string;
}

export interface UpdateMenuRequest {
  name?: string;
  sortOrder?: number;
  routePath?: string;
  icon?: string;
  showStatus?: MenuShowStatus;
  status?: MenuStatus;
}

export interface MenuListResponse {
  items: MenuItem[];
  total: number;
  page: number;
  pageSize: number;
}

// ==================== 过滤条件 ====================

export interface UserFilter {
  page?: number;
  pageSize?: number;
  username?: string;
  status?: UserStatus;
  carrier?: string;
  roleId?: string;
}

export interface RoleFilter {
  page?: number;
  pageSize?: number;
  name?: string;
  status?: RoleStatus;
}

export interface MenuFilter {
  page?: number;
  pageSize?: number;
  type?: MenuType;
  status?: MenuStatus;
  parentId?: string;
}

// ==================== 权限检查 ====================

// 检查用户是否有某个权限
export function hasPermission(user: User, permissionKey: string): boolean {
  // 从菜单中检查
  if (user.menus) {
    for (const menu of user.menus) {
      if (menu.permissionKey === permissionKey) {
        return true;
      }
      // 前缀匹配
      if (permissionKey.startsWith(menu.permissionKey + ':')) {
        return true;
      }
    }
  }
  return false;
}
```

### 2.2 API 服务 (services/api/rbacApi.ts)

```typescript
import { http } from '../http';
import type {
  User, CreateUserRequest, UpdateUserRequest, UserFilter,
  Role, CreateRoleRequest, UpdateRoleRequest, RoleFilter,
  MenuItem, CreateMenuRequest, UpdateMenuRequest, MenuFilter,
  MenuListResponse, MenuTreeNode,
} from '@/types/rbac';

// ==================== 菜单 API ====================

// 注意：后端 JSON 序列化已直接输出 camelCase，无需字段映射转换
// 后端 Go struct JSON 标签: `json:"permissionKey"`, `json:"parentId"` 等

export const menuApi = {
  // 列表
  list: (filter?: MenuFilter) => {
    return http.get<MenuListResponse>('/rbac/menus', {
      params: filter,
    }).then(res => res.data.data);
  },

  // 树形
  tree: (status?: 'normal' | 'disabled') => {
    return http.get<{ data: MenuItem[] }>('/rbac/menus/tree', {
      params: status ? { status } : undefined,
    }).then(res => ({
      items: res.data.data,
      tree: buildMenuTree(res.data.data),
    }));
  },

  // 用户菜单树
  userTree: () => {
    return http.get<{ data: MenuItem[] }>('/rbac/menus/user')
      .then(res => ({
        items: res.data.data,
        tree: buildMenuTree(res.data.data),
      }));
  },

  // 详情
  get: (id: string) => {
    return http.get<{ data: MenuItem }>(`/rbac/menus/${id}`)
      .then(res => res.data.data);
  },

  // 创建
  create: (data: CreateMenuRequest) => {
    return http.post<{ data: MenuItem }>('/rbac/menus', data)
      .then(res => res.data.data);
  },

  // 更新
  update: (id: string, data: UpdateMenuRequest) => {
    return http.put<{ data: MenuItem }>(`/rbac/menus/${id}`, data)
      .then(res => res.data.data);
  },

  // 删除
  delete: (ids: string[]) => {
    return http.delete('/rbac/menus', { data: { ids } });
  },
};

// ==================== 用户 API ====================

export const userApi = {
  list: (filter?: UserFilter) => {
    return http.get<{ data: { items: User[]; total: number; page: number; pageSize: number } }>
      ('/rbac/users', { params: filter })
      .then(res => res.data.data);
  },

  get: (id: string) => {
    return http.get<{ data: User }>(`/rbac/users/${id}`)
      .then(res => res.data.data);
  },

  create: (data: CreateUserRequest) => {
    return http.post<{ data: User }>('/rbac/users', data)
      .then(res => res.data.data);
  },

  update: (id: string, data: UpdateUserRequest) => {
    return http.put<{ data: User }>(`/rbac/users/${id}`, data)
      .then(res => res.data.data);
  },

  delete: (ids: string[]) => {
    return http.delete('/rbac/users', { data: { ids } });
  },
};

// ==================== 角色 API ====================

export const roleApi = {
  list: (filter?: RoleFilter) => {
    return http.get<{ data: { items: Role[]; total: number; page: number; pageSize: number } }>
      ('/rbac/roles', { params: filter })
      .then(res => res.data.data);
  },

  get: (id: string) => {
    return http.get<{ data: Role }>(`/rbac/roles/${id}`)
      .then(res => res.data.data);
  },

  create: (data: CreateRoleRequest) => {
    return http.post<{ data: Role }>('/rbac/roles', data)
      .then(res => res.data.data);
  },

  update: (id: string, data: UpdateRoleRequest) => {
    return http.put<{ data: Role }>(`/rbac/roles/${id}`, data)
      .then(res => res.data.data);
  },

  delete: (ids: string[]) => {
    return http.delete('/rbac/roles', { data: { ids } });
  },

  // 设置角色菜单
  setMenus: (roleId: string, menuIds: string[]) => {
    return http.post(`/rbac/roles/${roleId}/menus`, { menuIds });
  },

  // 获取角色菜单
  getMenus: (roleId: string) => {
    return http.get<{ data: MenuItem[] }>(`/rbac/roles/${roleId}/menus`)
      .then(res => res.data.data);
  },
};

// ==================== 辅助函数 ====================

// 构建菜单树
function buildMenuTree(items: MenuItem[]): MenuItem[] {
  const map = new Map<string, MenuItem>();
  const roots: MenuItem[] = [];

  // 先创建所有节点
  items.forEach(item => {
    map.set(item.id, { ...item, children: [] });
  });

  // 建立父子关系
  items.forEach(item => {
    const node = map.get(item.id)!;
    if (item.parentId) {
      const parent = map.get(item.parentId);
      if (parent) {
        parent.children = parent.children || [];
        parent.children.push(node);
      }
    } else {
      roots.push(node);
    }
  });

  return roots;
}
```

### 2.3 React Hooks (hooks/api/useRbac.ts)

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { userApi, roleApi, menuApi } from '@/services/api/rbacApi';
import type {
  UserFilter, RoleFilter, MenuFilter,
  CreateUserRequest, UpdateUserRequest,
  CreateRoleRequest, UpdateRoleRequest,
  CreateMenuRequest, UpdateMenuRequest,
} from '@/types/rbac';

// ==================== 用户 ====================

export function useUsers(filter: UserFilter) {
  return useQuery({
    queryKey: ['users', filter],
    queryFn: () => userApi.list(filter),
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreateUserRequest) => userApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries(['users']);
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateUserRequest }) =>
      userApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries(['users']);
    },
  });
}

export function useDeleteUsers() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (ids: string[]) => userApi.delete(ids),
    onSuccess: () => {
      queryClient.invalidateQueries(['users']);
    },
  });
}

// ==================== 角色 ====================

export function useRoles(filter?: RoleFilter) {
  return useQuery({
    queryKey: ['roles', filter],
    queryFn: () => roleApi.list(filter || {}),
  });
}

export function useCreateRole() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreateRoleRequest) => roleApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries(['roles']);
    },
  });
}

export function useUpdateRole() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateRoleRequest }) =>
      roleApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries(['roles']);
    },
  });
}

export function useDeleteRoles() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (ids: string[]) => roleApi.delete(ids),
    onSuccess: () => {
      queryClient.invalidateQueries(['roles']);
    },
  });
}

// ==================== 菜单 ====================

export function useMenus(filter?: MenuFilter) {
  return useQuery({
    queryKey: ['menus', filter],
    queryFn: () => menuApi.list(filter),
  });
}

export function useMenuTree(status?: 'normal' | 'disabled') {
  return useQuery({
    queryKey: ['menus', 'tree', status],
    queryFn: () => menuApi.tree(status),
  });
}

export function useUserMenuTree() {
  return useQuery({
    queryKey: ['menus', 'user'],
    queryFn: () => menuApi.userTree(),
    staleTime: 5 * 60 * 1000, // 5分钟
  });
}

export function useCreateMenu() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreateMenuRequest) => menuApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries(['menus']);
    },
  });
}

export function useUpdateMenu() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateMenuRequest }) =>
      menuApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries(['menus']);
    },
  });
}

export function useDeleteMenus() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (ids: string[]) => menuApi.delete(ids),
    onSuccess: () => {
      queryClient.invalidateQueries(['menus']);
    },
  });
}
```

---

## 三、命名约定

### 3.1 统一使用 camelCase

**后端 Go struct JSON 标签直接使用 camelCase**，前后端数据传递无需转换：

| 数据库字段 (snake_case) | 后端 JSON 标签 (camelCase) | 前端 TypeScript (camelCase) |
|------------------------|--------------------------|----------------------------|
| `permission_key` | `json:"permissionKey"` | `permissionKey: string` |
| `parent_id` | `json:"parentId"` | `parentId: string \| null` |
| `sort_order` | `json:"sortOrder"` | `sortOrder: number` |
| `route_path` | `json:"routePath"` | `routePath?: string` |
| `component_path` | `json:"componentPath"` | `componentPath?: string` |
| `show_status` | `json:"showStatus"` | `showStatus: 'show' \| 'hide'` |
| `created_by` | `json:"createdBy"` | `createdBy?: string` |
| `created_at` | `json:"createdAt"` | `createdAt: string` |
| `updated_by` | `json:"updatedBy"` | `updatedBy?: string` |
| `updated_at` | `json:"updatedAt"` | `updatedAt?: string` |
| `device_group_ids` | `json:"deviceGroupIds"` | `deviceGroupIds?: string[]` |
| `menu_ids` | `json:"menuIds"` | `menuIds?: string[]` |
| `is_system` | `json:"isSystem"` | `isSystem: boolean` |

**重要**：
- 数据库表字段保持 `snake_case`（PostgreSQL 约定）
- Go struct JSON 标签使用 `camelCase`（API 响应）
- TypeScript 接口使用 `camelCase`（前端）
- **无需任何字段映射转换**

### 3.2 URL 查询参数

URL 查询参数保持 `snake_case`（`form` 标签）：

```go
type MenuFilter struct {
    Type     *MenuType   `form:"type"`
    Status   *MenuStatus `form:"status"`
    ParentID *uuid.UUID  `form:"parent_id"`  // URL 参数使用 snake_case
}
```

前端请求时：
```typescript
http.get('/rbac/menus', { params: { parent_id: null } })
```

---

## 四、API 响应格式

### 4.1 列表响应

```json
{
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```

### 4.2 单对象响应

```json
{
  "data": {
    "id": "...",
    "name": "设备管理",
    ...
  }
}
```

### 4.3 操作响应

```json
{
  "data": {
    "success": true
  }
}
```

### 4.4 错误响应

```json
{
  "error": "错误信息"
}
```

---

## 五、增强功能需求

### 5.1 切换角色功能

#### 5.1.1 需求描述

管理员可以拥有多个角色，登录成功后可以切换当前使用的角色。每个管理员可以设置一个默认角色（登录成功后自动激活的角色）。

#### 5.1.2 数据模型扩展

**用户-角色关联表扩展**

```sql
-- 在 user_roles 表中添加 is_default 字段
ALTER TABLE user_roles 
ADD COLUMN is_default BOOLEAN DEFAULT FALSE;

-- 每个用户只能有一个默认角色
CREATE UNIQUE INDEX idx_user_roles_user_default 
ON user_roles(user_id) 
WHERE is_default = TRUE;
```

**Go 模型扩展**

```go
// UserRole 用户角色关联
type UserRole struct {
    ID        uuid.UUID `json:"id"`
    UserID    uuid.UUID `json:"userId"`
    RoleID    uuid.UUID `json:"roleId"`
    IsDefault bool      `json:"isDefault"` // 新增：是否为默认角色
    CreatedAt time.Time `json:"createdAt"`
}

// SwitchRoleRequest 切换角色请求
type SwitchRoleRequest struct {
    RoleID uuid.UUID `json:"roleId" binding:"required"`
}

// SetDefaultRoleRequest 设置默认角色请求
type SetDefaultRoleRequest struct {
    RoleID uuid.UUID `json:"roleId" binding:"required"`
}

// UserRolesResponse 用户角色列表响应
type UserRolesResponse struct {
    Roles       []*Role   `json:"roles"`
    CurrentRole *Role     `json:"currentRole"`      // 当前激活的角色
    DefaultRole *Role     `json:"defaultRole"`      // 默认角色
}
```

#### 5.1.3 API 接口设计

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 获取用户角色列表 | GET | `/api/v1/users/me/roles` | 获取当前用户的所有角色 |
| 切换角色 | POST | `/api/v1/users/me/roles/switch` | 切换当前使用的角色 |
| 设置默认角色 | POST | `/api/v1/users/me/roles/default` | 设置默认角色 |

#### 5.1.4 业务流程

1. **登录流程增强**
   - 用户登录成功后，查找用户的默认角色
   - 如果存在默认角色，将其作为当前角色写入 JWT Token
   - 如果不存在默认角色，选择第一个角色作为当前角色
   - 返回用户信息时包含当前角色和可选角色列表

2. **切换角色流程**
   - 用户发起切换角色请求
   - 验证用户是否拥有该角色
   - 更新 JWT Token 中的角色信息
   - 返回新的 Token 和菜单树
   - 前端存储新 Token，刷新页面菜单

3. **设置默认角色流程**
   - 用户设置某个角色为默认角色
   - 清除该用户其他角色的默认标记
   - 设置目标角色为默认角色
   - 下次登录时自动使用该角色

#### 5.1.5 前端交互设计

**用户下拉菜单结构**

```
[用户头像] 用户名 ▼
├── 切换角色
│   ├── ○ 系统管理员 (当前)
│   ├── ○ 运维管理员
│   └── ○ 监控管理员
├── 设置默认角色
│   └── [选择框] 选择默认角色
├── 修改密码
└── 退出登录
```

**切换角色交互**

```typescript
// 切换角色
const handleSwitchRole = async (roleId: string) => {
  const response = await userApi.switchRole(roleId);
  // 更新 Token
  localStorage.setItem('token', response.data.token);
  // 重新获取菜单
  const menus = await menuApi.userTree();
  // 更新路由和菜单
  store.setMenus(menus.tree);
  // 刷新页面或跳转到首页
  window.location.reload();
};
```

---

### 5.2 系统菜单初始化

#### 5.2.1 菜单结构设计

基于 OMC 系统现有功能，设计完整的菜单树结构：

```
OMC 系统
├── 首页 (Dashboard)
├── 设备管理
│   ├── 设备列表
│   ├── 设备分组
│   ├── 设备注册
│   ├── 设备回收站
│   └── 设备规则
├── 配置管理
│   ├── 配置模板
│   ├── 参数模型
│   ├── 配置基线
│   └── MML 脚本
├── 性能管理
│   ├── KPI 指标管理
│   ├── KPI 数据查询
│   ├── PM 任务管理
│   └── 计数器管理
├── 告警管理
│   ├── 活动告警
│   ├── 告警历史
│   ├── 告警规则
│   ├── 告警库
│   └── 告警统计
├── 软件管理
│   ├── 固件文件
│   ├── 升级任务
│   └── 升级历史
├── MR 管理
│   ├── MR 指标管理
│   ├── MR 数据查询
│   └── 设备 MR 映射
├── 自动开站
│   ├── 开站任务
│   ├── 模板匹配
│   └── 开站日志
├── 系统管理
│   ├── 用户管理
│   ├── 角色管理
│   ├── 菜单管理
│   ├── 字典管理
│   ├── 系统日志
│   └── 系统设置
└── 运维管理
    ├── 设备日志
    ├── 异常日志
    ├── 事件日志
    └── 检查列表
```

#### 5.2.2 菜单初始化 SQL

```sql
-- 插入顶级菜单
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, show_status, status, created_at, updated_at) VALUES
-- 一级菜单
(gen_random_uuid(), '首页', 'menu', 'dashboard', NULL, 1, '/dashboard', 'HomeOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), '设备管理', 'directory', 'device', NULL, 2, '/device', 'DeviceOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), '配置管理', 'directory', 'config', NULL, 3, '/config', 'SettingOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), '性能管理', 'directory', 'pm', NULL, 4, '/pm', 'LineChartOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), '告警管理', 'directory', 'alarm', NULL, 5, '/alarm', 'AlertOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), '软件管理', 'directory', 'software', NULL, 6, '/software', 'CloudUploadOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), 'MR管理', 'directory', 'mr', NULL, 7, '/mr', 'BarChartOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), '自动开站', 'directory', 'autoprov', NULL, 8, '/autoprov', 'RocketOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), '系统管理', 'directory', 'system', NULL, 9, '/system', 'ToolOutlined', 'show', 'normal', NOW(), NOW()),
(gen_random_uuid(), '运维管理', 'directory', 'operation', NULL, 10, '/operation', 'MonitorOutlined', 'show', 'normal', NOW(), NOW());

-- 设备管理子菜单
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, show_status, status, created_at, updated_at) 
SELECT 
    gen_random_uuid(), '设备列表', 'menu', 'device:list', m.id, 1, '/device/list', 'UnorderedListOutlined', 'show', 'normal', NOW(), NOW()
FROM menus m WHERE m.permission_key = 'device';

-- 更多子菜单... (根据完整菜单结构插入)
```

#### 5.2.3 菜单初始化 Go 代码

```go
package admin

import (
    "context"
    "log"
    
    "github.com/google/uuid"
)

// SeedMenus 初始化系统菜单
func (s *menuService) SeedMenus(ctx context.Context) error {
    // 检查是否已初始化
    count, err := s.countMenus(ctx)
    if err != nil {
        return err
    }
    if count > 0 {
        log.Println("菜单已初始化，跳过")
        return nil
    }
    
    log.Println("开始初始化系统菜单...")
    
    // 创建一级菜单
    menus := []CreateMenuRequest{
        // 首页
        {
            Name:          "首页",
            Type:          MenuTypeMenu,
            PermissionKey: "dashboard",
            RoutePath:     "/dashboard",
            Icon:          "HomeOutlined",
            ShowStatus:    MenuShowStatusShow,
            SortOrder:     1,
        },
        // 设备管理
        {
            Name:          "设备管理",
            Type:          MenuTypeDirectory,
            PermissionKey: "device",
            RoutePath:     "/device",
            Icon:          "DeviceOutlined",
            ShowStatus:    MenuShowStatusShow,
            SortOrder:     2,
        },
        // ... 其他一级菜单
    }
    
    operatorID := uuid.Nil // 系统初始化
    for _, req := range menus {
        menu, err := s.CreateMenu(ctx, &req, operatorID)
        if err != nil {
            return err
        }
        
        // 创建子菜单
        if err := s.seedChildMenus(ctx, menu.ID, menu.PermissionKey, operatorID); err != nil {
            return err
        }
    }
    
    log.Println("系统菜单初始化完成")
    return nil
}

// seedChildMenus 创建子菜单
func (s *menuService) seedChildMenus(ctx context.Context, parentID uuid.UUID, parentKey string, operatorID uuid.UUID) error {
    childMenus := map[string][]CreateMenuRequest{
        "device": {
            {Name: "设备列表", Type: MenuTypeMenu, PermissionKey: "device:list", RoutePath: "/device/list", Icon: "UnorderedListOutlined", SortOrder: 1},
            {Name: "设备分组", Type: MenuTypeMenu, PermissionKey: "device:group", RoutePath: "/device/groups", Icon: "ApartmentOutlined", SortOrder: 2},
            {Name: "设备注册", Type: MenuTypeMenu, PermissionKey: "device:register", RoutePath: "/device/register", Icon: "PlusOutlined", SortOrder: 3},
            {Name: "设备回收站", Type: MenuTypeMenu, PermissionKey: "device:recycle", RoutePath: "/device/recycle", Icon: "DeleteOutlined", SortOrder: 4},
            {Name: "设备规则", Type: MenuTypeMenu, PermissionKey: "device:rule", RoutePath: "/device/rules", Icon: "ControlOutlined", SortOrder: 5},
        },
        "system": {
            {Name: "用户管理", Type: MenuTypeMenu, PermissionKey: "system:user", RoutePath: "/system/users", Icon: "UserOutlined", SortOrder: 1},
            {Name: "角色管理", Type: MenuTypeMenu, PermissionKey: "system:role", RoutePath: "/system/roles", Icon: "TeamOutlined", SortOrder: 2},
            {Name: "菜单管理", Type: MenuTypeMenu, PermissionKey: "system:menu", RoutePath: "/system/menus", Icon: "MenuOutlined", SortOrder: 3},
            {Name: "字典管理", Type: MenuTypeMenu, PermissionKey: "system:dict", RoutePath: "/system/dicts", Icon: "BookOutlined", SortOrder: 4},
            {Name: "系统日志", Type: MenuTypeMenu, PermissionKey: "system:log", RoutePath: "/system/logs", Icon: "FileTextOutlined", SortOrder: 5},
            {Name: "系统设置", Type: MenuTypeMenu, PermissionKey: "system:setting", RoutePath: "/system/settings", Icon: "SettingOutlined", SortOrder: 6},
        },
        // ... 其他菜单的子菜单
    }
    
    if children, ok := childMenus[parentKey]; ok {
        pid := parentID
        for _, req := range children {
            req.ParentID = &pid
            _, err := s.CreateMenu(ctx, &req, operatorID)
            if err != nil {
                return err
            }
        }
    }
    
    return nil
}

func (s *menuService) countMenus(ctx context.Context) (int64, error) {
    // 实现计数逻辑
    return 0, nil
}
```

---

### 5.3 动态菜单显示

#### 5.3.1 业务流程

1. **用户登录**
   - 验证用户名密码
   - 获取用户的默认角色（或第一个角色）
   - 将角色 ID 写入 JWT Token
   - 返回用户信息和 Token

2. **前端初始化**
   - 存储 Token
   - 调用 `/api/v1/menus/user` 获取当前角色的菜单树
   - 根据菜单树动态生成侧边栏菜单
   - 根据菜单树动态注册路由

3. **切换角色**
   - 用户选择新角色
   - 调用 `/api/v1/users/me/roles/switch` 切换角色
   - 获取新 Token
   - 重新获取菜单树
   - 刷新页面菜单和路由

#### 5.3.2 后端实现

**JWT Token 扩展**

```go
// Claims JWT Claims
type Claims struct {
    UserID   uuid.UUID `json:"userId"`
    Username string     `json:"username"`
    RoleID   uuid.UUID `json:"roleId"`      // 新增：当前角色 ID
    RoleName string     `json:"roleName"`    // 新增：当前角色名称
    jwt.RegisteredClaims
}

// GenerateToken 生成 Token
func (s *authService) GenerateToken(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) (string, error) {
    user, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return "", err
    }
    
    role, err := s.roleRepo.GetByID(ctx, roleID)
    if err != nil {
        return "", err
    }
    
    claims := &Claims{
        UserID:   userID,
        Username: user.Username,
        RoleID:   roleID,
        RoleName: role.Name,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.jwtSecret))
}
```

**获取用户菜单树**

```go
// GetUserMenuTree 获取当前用户的菜单树（基于当前角色）
func (s *menuService) GetUserMenuTree(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) ([]*Menu, error) {
    // 获取当前角色的菜单
    menus, err := s.repo.GetByRole(ctx, roleID)
    if err != nil {
        return nil, err
    }
    
    // 构建树形结构并过滤
    tree := s.repo.buildTree(menus)
    return s.filterTreeForDisplay(tree), nil
}
```

#### 5.3.3 前端实现

**动态菜单生成**

```typescript
// 侧边栏菜单组件
import { Menu } from 'antd';
import { useUserMenuTree } from '@/hooks/api/useRbac';
import { useNavigate } from 'react-router-dom';

const SidebarMenu: React.FC = () => {
  const { data: menuData, isLoading } = useUserMenuTree();
  const navigate = useNavigate();
  
  // 将菜单树转换为 Ant Design Menu 格式
  const menuItems = menuData?.tree?.map(item => ({
    key: item.id,
    icon: React.createElement(icons[item.icon]),
    label: item.name,
    children: item.children?.map(child => ({
      key: child.id,
      icon: React.createElement(icons[child.icon]),
      label: child.name,
    })),
  }));
  
  const handleMenuClick = ({ key }: { key: string }) => {
    // 查找菜单对应的路由
    const menu = findMenuById(menuData?.items, key);
    if (menu?.routePath) {
      navigate(menu.routePath);
    }
  };
  
  if (isLoading) return <Spin />;
  
  return (
    <Menu
      mode="inline"
      items={menuItems}
      onClick={handleMenuClick}
    />
  );
};
```

**路由动态注册**

```typescript
// 路由配置
import { lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';

// 动态导入页面组件
const DeviceList = lazy(() => import('@/pages/device/list'));
const DeviceGroups = lazy(() => import('@/pages/device/groups'));
// ... 其他页面

const AppRoutes: React.FC<{ menus: MenuItem[] }> = ({ menus }) => {
  // 根据菜单动态生成路由
  const renderRoutes = (menuItems: MenuItem[]) => {
    return menuItems.map(menu => {
      if (menu.type === 'menu' && menu.routePath) {
        const Component = getComponentByRoute(menu.routePath);
        return (
          <Route
            key={menu.id}
            path={menu.routePath}
            element={
              <Suspense fallback={<Spin />}>
                <Component />
              </Suspense>
            }
          />
        );
      }
      if (menu.children) {
        return renderRoutes(menu.children);
      }
      return null;
    });
  };
  
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/dashboard" />} />
      {renderRoutes(menus)}
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
};
```

---

### 5.4 菜单与 API 关联关系

#### 5.4.1 需求描述

建立菜单、页面功能（添加、删除、批量删除等）与 API 的关联关系，通过数据库存储。当用户所属角色拥有某个"菜单"或"功能"时，自动拥有对应的 API 权限。

#### 5.4.2 数据模型设计

**API 权限表**

```sql
-- API 权限表
CREATE TABLE api_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_id UUID NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    name VARCHAR(64) NOT NULL,              -- API 名称，如"创建设备"
    permission_key VARCHAR(128) NOT NULL UNIQUE, -- 权限标识，如"device:create"
    path VARCHAR(256) NOT NULL,             -- API 路径，如"/api/v1/devices"
    method VARCHAR(16) NOT NULL,            -- HTTP 方法：GET, POST, PUT, DELETE
    description TEXT,                       -- 描述
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_permissions_menu ON api_permissions(menu_id);
CREATE INDEX idx_api_permissions_path_method ON api_permissions(path, method);

-- 角色 API 权限关联表
CREATE TABLE role_api_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    api_permission_id UUID NOT NULL REFERENCES api_permissions(id) ON DELETE CASCADE,
    created_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, api_permission_id)
);

CREATE INDEX idx_role_api_permissions_role ON role_api_permissions(role_id);
```

**Go 模型**

```go
// APIPermission API 权限
type APIPermission struct {
    ID            uuid.UUID `json:"id"`
    MenuID        uuid.UUID `json:"menuId"`
    Name          string    `json:"name"`
    PermissionKey string    `json:"permissionKey"`
    Path          string    `json:"path"`
    Method        string    `json:"method"`
    Description   string    `json:"description,omitempty"`
    CreatedAt     time.Time `json:"createdAt"`
    UpdatedAt     time.Time `json:"updatedAt"`
    
    // 关联
    Menu          *Menu     `json:"menu,omitempty"`
}

// CreateAPIPermissionRequest 创建 API 权限请求
type CreateAPIPermissionRequest struct {
    MenuID        uuid.UUID `json:"menuId" binding:"required"`
    Name          string    `json:"name" binding:"required,max=64"`
    PermissionKey string    `json:"permissionKey" binding:"required,max=128"`
    Path          string    `json:"path" binding:"required"`
    Method        string    `json:"method" binding:"required,oneof=GET POST PUT DELETE PATCH"`
    Description   string    `json:"description"`
}

// RoleAPIPermission 角色 API 权限关联
type RoleAPIPermission struct {
    ID              uuid.UUID `json:"id"`
    RoleID          uuid.UUID `json:"roleId"`
    APIPermissionID uuid.UUID `json:"apiPermissionId"`
    CreatedBy       *uuid.UUID `json:"createdBy,omitempty"`
    CreatedAt       time.Time `json:"createdAt"`
}
```

#### 5.4.3 菜单与 API 关联示例

| 菜单 | 功能 | API 路径 | 方法 | 权限标识 |
|------|------|----------|------|----------|
| 设备列表 | 查询设备 | `/api/v1/devices` | GET | `device:list:query` |
| 设备列表 | 创建设备 | `/api/v1/devices` | POST | `device:list:create` |
| 设备列表 | 更新设备 | `/api/v1/devices/:id` | PUT | `device:list:update` |
| 设备列表 | 删除设备 | `/api/v1/devices` | DELETE | `device:list:delete` |
| 设备列表 | 批量删除 | `/api/v1/devices/batch` | DELETE | `device:list:batchDelete` |
| 设备列表 | 导出设备 | `/api/v1/devices/export` | GET | `device:list:export` |
| 设备分组 | 查询分组 | `/api/v1/device-groups` | GET | `device:group:query` |
| 设备分组 | 创建分组 | `/api/v1/device-groups` | POST | `device:group:create` |
| 设备分组 | 更新分组 | `/api/v1/device-groups/:id` | PUT | `device:group:update` |
| 设备分组 | 删除分组 | `/api/v1/device-groups` | DELETE | `device:group:delete` |
| 用户管理 | 查询用户 | `/api/v1/users` | GET | `system:user:query` |
| 用户管理 | 创建用户 | `/api/v1/users` | POST | `system:user:create` |
| 用户管理 | 更新用户 | `/api/v1/users/:id` | PUT | `system:user:update` |
| 用户管理 | 删除用户 | `/api/v1/users` | DELETE | `system:user:delete` |
| 角色管理 | 查询角色 | `/api/v1/roles` | GET | `system:role:query` |
| 角色管理 | 创建角色 | `/api/v1/roles` | POST | `system:role:create` |
| 角色管理 | 更新角色 | `/api/v1/roles/:id` | PUT | `system:role:update` |
| 角色管理 | 删除角色 | `/api/v1/roles` | DELETE | `system:role:delete` |
| 角色管理 | 设置角色菜单 | `/api/v1/roles/:id/menus` | POST | `system:role:setMenus` |
| 角色管理 | 设置角色 API 权限 | `/api/v1/roles/:id/api-permissions` | POST | `system:role:setApiPermissions` |

#### 5.4.4 API 权限初始化 SQL

```sql
-- 设备管理 API 权限
INSERT INTO api_permissions (id, menu_id, name, permission_key, path, method, description) 
SELECT gen_random_uuid(), m.id, '查询设备', 'device:list:query', '/api/v1/devices', 'GET', '查询设备列表'
FROM menus m WHERE m.permission_key = 'device:list';

INSERT INTO api_permissions (id, menu_id, name, permission_key, path, method, description) 
SELECT gen_random_uuid(), m.id, '创建设备', 'device:list:create', '/api/v1/devices', 'POST', '创建新设备'
FROM menus m WHERE m.permission_key = 'device:list';

INSERT INTO api_permissions (id, menu_id, name, permission_key, path, method, description) 
SELECT gen_random_uuid(), m.id, '更新设备', 'device:list:update', '/api/v1/devices/:id', 'PUT', '更新设备信息'
FROM menus m WHERE m.permission_key = 'device:list';

INSERT INTO api_permissions (id, menu_id, name, permission_key, path, method, description) 
SELECT gen_random_uuid(), m.id, '删除设备', 'device:list:delete', '/api/v1/devices', 'DELETE', '删除设备'
FROM menus m WHERE m.permission_key = 'device:list';

-- 更多 API 权限...
```

#### 5.4.5 API 权限管理接口

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 获取菜单 API 列表 | GET | `/api/v1/menus/:id/api-permissions` | 获取菜单关联的 API 权限 |
| 创建 API 权限 | POST | `/api/v1/api-permissions` | 创建新的 API 权限 |
| 更新 API 权限 | PUT | `/api/v1/api-permissions/:id` | 更新 API 权限信息 |
| 删除 API 权限 | DELETE | `/api/v1/api-permissions/:id` | 删除 API 权限 |
| 设置角色 API 权限 | POST | `/api/v1/roles/:id/api-permissions` | 为角色分配 API 权限 |
| 获取角色 API 权限 | GET | `/api/v1/roles/:id/api-permissions` | 获取角色的 API 权限列表 |

---

### 5.5 API 鉴权增强

#### 5.5.1 鉴权流程

**三级鉴权链：角色 > 菜单 > API**

```
请求到达
  ↓
1. 验证 JWT Token（是否有效、是否过期）
  ↓
2. 提取当前角色 ID（从 JWT Token）
  ↓
3. 检查角色是否有菜单权限
  ↓
4. 检查角色是否有 API 权限
  ↓
5. 允许/拒绝请求
```

#### 5.5.2 中间件实现

```go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

// APIPermissionMiddleware API 权限检查中间件
func APIPermissionMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 获取用户信息
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(401, gin.H{"error": "未授权"})
            c.Abort()
            return
        }
        
        // 2. 获取当前角色 ID
        roleID, exists := c.Get("role_id")
        if !exists {
            c.JSON(403, gin.H{"error": "未选择角色"})
            c.Abort()
            return
        }
        
        // 3. 获取请求路径和方法
        requestPath := c.Request.URL.Path
        requestMethod := c.Request.Method
        
        // 4. 检查是否有 API 权限
        hasPermission, err := checkAPIPermission(c.Request.Context(), roleID.(uuid.UUID), requestPath, requestMethod)
        if err != nil {
            c.JSON(500, gin.H{"error": "权限检查失败"})
            c.Abort()
            return
        }
        
        if !hasPermission {
            c.JSON(403, gin.H{
                "error": "无 API 调用权限",
                "path": requestPath,
                "method": requestMethod,
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// checkAPIPermission 检查 API 权限
func checkAPIPermission(ctx context.Context, roleID uuid.UUID, path string, method string) (bool, error) {
    // 1. 标准化路径（将 /api/v1/devices/123 转为 /api/v1/devices/:id）
    normalizedPath := normalizePath(path)
    
    // 2. 查询角色是否有该 API 权限
    query := `
        SELECT COUNT(*) 
        FROM role_api_permissions rap
        JOIN api_permissions ap ON rap.api_permission_id = ap.id
        WHERE rap.role_id = $1 
          AND ap.path = $2 
          AND ap.method = $3
    `
    
    var count int
    err := db.GetContext(ctx, &count, query, roleID, normalizedPath, method)
    if err != nil {
        return false, err
    }
    
    return count > 0, nil
}

// normalizePath 标准化路径
func normalizePath(path string) string {
    // 将 /api/v1/devices/550e8400-e29b-41d4-a716-446655440000
    // 转为 /api/v1/devices/:id
    
    parts := strings.Split(path, "/")
    for i, part := range parts {
        // 如果是 UUID，则替换为 :id
        if _, err := uuid.Parse(part); err == nil {
            parts[i] = ":id"
        }
    }
    return strings.Join(parts, "/")
}
```

#### 5.5.3 路由注册增强

```go
// 注册路由时添加权限中间件
func RegisterRoutes(r *gin.Engine) {
    v1 := r.Group("/api/v1")
    
    // 公开路由（无需认证）
    public := v1.Group("/public")
    public.POST("/login", authHandler.Login)
    
    // 需要认证的路由
    authenticated := v1.Group("")
    authenticated.Use(AuthMiddleware()) // JWT 验证
    
    // 需要 API 权限的路由
    api := authenticated.Group("")
    api.Use(APIPermissionMiddleware()) // API 权限检查
    
    // 设备管理
    devices := api.Group("/devices")
    devices.GET("", deviceHandler.List)         // 需要 device:list:query 权限
    devices.POST("", deviceHandler.Create)      // 需要 device:list:create 权限
    devices.PUT("/:id", deviceHandler.Update)   // 需要 device:list:update 权限
    devices.DELETE("", deviceHandler.Delete)    // 需要 device:list:delete 权限
    
    // 系统管理
    users := api.Group("/users")
    users.GET("", userHandler.List)             // 需要 system:user:query 权限
    users.POST("", userHandler.Create)          // 需要 system:user:create 权限
    users.PUT("/:id", userHandler.Update)       // 需要 system:user:update 权限
    users.DELETE("", userHandler.Delete)        // 需要 system:user:delete 权限
}
```

#### 5.5.4 超级管理员特殊处理

```go
// checkAPIPermission 增强版（包含超级管理员检查）
func checkAPIPermission(ctx context.Context, userID uuid.UUID, roleID uuid.UUID, path string, method string) (bool, error) {
    // 1. 检查是否为超级管理员
    isAdmin, err := isSuperAdmin(ctx, userID)
    if err != nil {
        return false, err
    }
    if isAdmin {
        return true, nil // 超级管理员拥有所有权限
    }
    
    // 2. 检查角色是否有 API 权限
    normalizedPath := normalizePath(path)
    
    query := `
        SELECT COUNT(*) 
        FROM role_api_permissions rap
        JOIN api_permissions ap ON rap.api_permission_id = ap.id
        WHERE rap.role_id = $1 
          AND ap.path = $2 
          AND ap.method = $3
    `
    
    var count int
    err = db.GetContext(ctx, &count, query, roleID, normalizedPath, method)
    if err != nil {
        return false, err
    }
    
    return count > 0, nil
}

func isSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
    query := `
        SELECT COUNT(*) 
        FROM user_roles ur
        JOIN roles r ON ur.role_id = r.id
        WHERE ur.user_id = $1 AND r.name = '超级管理员' AND r.status = 'active'
    `
    
    var count int
    err := db.GetContext(ctx, &count, query, userID)
    return count > 0, err
}
```

#### 5.5.5 前端权限检查

**按钮级权限控制**

```typescript
// 权限检查 Hook
import { useQuery } from '@tanstack/react-query';
import { userApi } from '@/services/api/rbacApi';

export function usePermission(permissionKey: string): boolean {
  const { data: apiPermissions } = useQuery({
    queryKey: ['user', 'api-permissions'],
    queryFn: () => userApi.getApiPermissions(),
    staleTime: 5 * 60 * 1000,
  });
  
  if (!apiPermissions) return false;
  
  return apiPermissions.some(
    (perm: any) => perm.permissionKey === permissionKey
  );
}

// 权限组件
interface PermissionProps {
  permission: string;
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

const Permission: React.FC<PermissionProps> = ({ 
  permission, 
  children, 
  fallback = null 
}) => {
  const hasPermission = usePermission(permission);
  
  if (!hasPermission) {
    return <>{fallback}</>;
  }
  
  return <>{children}</>;
};

// 使用示例
const DeviceListPage: React.FC = () => {
  return (
    <div>
      <h1>设备列表</h1>
      
      <Permission permission="device:list:create">
        <Button onClick={handleCreate}>创建设备</Button>
      </Permission>
      
      <Permission permission="device:list:batchDelete">
        <Button onClick={handleBatchDelete}>批量删除</Button>
      </Permission>
      
      <Table 
        columns={[
          {
            title: '操作',
            render: (_, record) => (
              <>
                <Permission permission="device:list:update">
                  <Button onClick={() => handleEdit(record)}>编辑</Button>
                </Permission>
                
                <Permission permission="device:list:delete">
                  <Button onClick={() => handleDelete(record)}>删除</Button>
                </Permission>
              </>
            ),
          },
        ]}
      />
    </div>
  );
};
```

---

### 5.6 数据库完整 Schema

```sql
-- 用户表（已存在，需要扩展）
ALTER TABLE users 
ADD COLUMN default_role_id UUID REFERENCES roles(id);

-- 角色表（已存在）
-- roles 表已包含基本字段

-- 菜单表（已存在）
-- menus 表已包含基本字段

-- 用户角色关联表（已存在，需要扩展）
ALTER TABLE user_roles 
ADD COLUMN is_default BOOLEAN DEFAULT FALSE;

-- API 权限表（新增）
CREATE TABLE api_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_id UUID NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    name VARCHAR(64) NOT NULL,
    permission_key VARCHAR(128) NOT NULL UNIQUE,
    path VARCHAR(256) NOT NULL,
    method VARCHAR(16) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_permissions_menu ON api_permissions(menu_id);
CREATE INDEX idx_api_permissions_path_method ON api_permissions(path, method);

-- 角色菜单关联表（已存在）
-- role_menus 表已存在

-- 角色 API 权限关联表（新增）
CREATE TABLE role_api_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    api_permission_id UUID NOT NULL REFERENCES api_permissions(id) ON DELETE CASCADE,
    created_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, api_permission_id)
);

CREATE INDEX idx_role_api_permissions_role ON role_api_permissions(role_id);
```

---

## 六、实施计划

### 6.1 Phase 1: 基础功能（1-2 周）

- [ ] 扩展数据库 Schema（用户角色关联表添加 is_default）
- [ ] 实现切换角色 API
- [ ] 实现设置默认角色 API
- [ ] 扩展 JWT Token 包含角色信息
- [ ] 前端实现角色切换 UI

### 6.2 Phase 2: 菜单初始化（1 周）

- [ ] 编写菜单初始化 SQL
- [ ] 实现菜单初始化 Go 代码
- [ ] 编写种子数据脚本
- [ ] 测试菜单初始化流程

### 6.3 Phase 3: 动态菜单显示（1-2 周）

- [ ] 修改登录流程，返回当前角色
- [ ] 实现基于角色的菜单树 API
- [ ] 前端实现动态菜单生成
- [ ] 前端实现动态路由注册
- [ ] 测试切换角色后的菜单刷新

### 6.4 Phase 4: API 权限管理（2-3 周）

- [ ] 创建 API 权限表
- [ ] 实现 API 权限 CRUD API
- [ ] 编写 API 权限初始化数据
- [ ] 实现角色 API 权限分配 API
- [ ] 前端实现 API 权限管理界面

### 6.5 Phase 5: API 鉴权增强（1-2 周）

- [ ] 实现 API 权限中间件
- [ ] 实现路径标准化
- [ ] 实现超级管理员特殊处理
- [ ] 前端实现按钮级权限控制
- [ ] 测试完整的鉴权流程

### 6.6 Phase 6: 测试和优化（1-2 周）

- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能优化
- [ ] 文档编写
- [ ] 用户培训材料

**总工期：7-12 周**

---

## 七、风险与注意事项

### 7.1 数据迁移

- 现有用户需要分配默认角色
- 现有角色需要分配菜单权限
- 建议在迁移脚本中自动处理

### 7.2 性能考虑

- API 权限检查每次请求都会执行，需要缓存优化
- 建议使用 Redis 缓存角色权限信息
- 菜单树可以使用前端缓存（5 分钟）

### 7.3 安全性

- 超级管理员权限需要严格控制
- 敏感操作需要二次验证
- 所有权限变更需要记录审计日志

### 7.4 兼容性

- 需要兼容现有的 JWT Token（平滑升级）
- 前端需要支持新旧两种 Token 格式
- API 权限检查需要支持降级（未配置权限时默认允许/拒绝）

---

## 八、总结

本方案在原有 RBAC 系统基础上，增强了以下功能：

1. **多角色支持**：用户可以拥有多个角色，支持切换角色和设置默认角色
2. **动态菜单**：基于当前角色动态生成菜单和路由
3. **API 权限管理**：建立菜单、功能与 API 的关联关系
4. **三级鉴权链**：角色 > 菜单 > API 的完整鉴权体系
5. **按钮级权限控制**：前端实现细粒度的权限控制

这些增强功能将使 OMC 系统的权限管理更加灵活、安全和易用。
