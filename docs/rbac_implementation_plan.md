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
