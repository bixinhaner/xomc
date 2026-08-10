// Package authz 是设备组数据权限（租户隔离）的**统一强制层**。
//
// 背景：v1.0 把租户边界从「users.carrier + RequireCarrier 中间件」迁到「设备分组
// 可见性（VisibleGroups）」，由 admin.PermissionService.GetUserVisibleGroupIDs 从
// 认证主体派生。迁移初期只有 device 主列表/单读/topology 三处消费，alarm/pm/GIS/
// 预注册及 ufte/bundle/stationlog/eventlog/rebootrecord/interop 等 HTTP 暴露面在
// service/repository 层完全没有消费点 → 跨设备组水平越权（#63/#64）+ 升级/回退可越权
// 写（#59）。
//
// 本包把「ctx → 可见分组解析」与「按设备 ID 的归属校验」收口为单一机制，所有模块统一
// 消费，避免各造一套导致语义漂移。可见分组三态契约（与 device_info_pg_repository.go
// 的 fail-closed 范式一致，超管旁路锚定 device_authz.go）：
//
//	nil            → 超管（PermissionService 对 source='builtIn' 返 nil）：不过滤，可见全部。
//	[]uuid.UUID{}  → 已认证但无任何分组权限：fail-closed（空集）。
//	[g1, g2, ...]  → 仅限这些分组下的设备。
package authz

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// VisibleGroupsResolver 从主体身份（userID + isSuperAdmin）解析可见设备组。
// 由 admin.PermissionService 实现。返回切片遵循本包顶部的三态契约。
type VisibleGroupsResolver interface {
	GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID, isSuperAdmin bool) ([]uuid.UUID, error)
}

// GroupReader 读取单个设备的设备组归属（device_group_members）。
// 由 device.PgDeviceGroupReader 实现——此处用结构化接口，不反向 import device。
type GroupReader interface {
	GetDeviceGroupIDs(ctx context.Context, deviceID uuid.UUID) ([]uuid.UUID, error)
}

// SplitVisibleGroups returns the real visible group IDs and whether legacy
// devices without any membership row should also be visible. The default L2
// group is a real group, but retains compatibility with legacy ungrouped data.
func SplitVisibleGroups(visibleGroups []uuid.UUID) (realGroups []uuid.UUID, includeUngrouped bool) {
	if visibleGroups == nil {
		return nil, false
	}
	realGroups = make([]uuid.UUID, 0, len(visibleGroups))
	for _, gid := range visibleGroups {
		if gid.String() == global.DefaultLevel2GroupID {
			includeUngrouped = true
		}
		realGroups = append(realGroups, gid)
	}
	if len(realGroups) == 0 {
		realGroups = nil
	}
	return realGroups, includeUngrouped
}

// Resolver 从 gin 请求上下文解析调用者的可见设备组，是所有 HTTP 模块的统一入口。
type Resolver struct {
	perm VisibleGroupsResolver
}

type permissionBackendError struct {
	cause error
}

func (e *permissionBackendError) Error() string {
	return fmt.Sprintf("resolve visible groups: %v", e.cause)
}

func (e *permissionBackendError) Unwrap() error {
	return commonerrors.ErrInternal
}

// NewResolver 构造 Resolver。perm 为 nil 时退化为不强制（dev/test）。
func NewResolver(perm VisibleGroupsResolver) *Resolver { return &Resolver{perm: perm} }

// Enabled 报告数据权限强制是否已装配（perm != nil）。
// perm == nil 对应 dev/test 退化（不强制），与 device 既有 nil-safe 语义一致；
// 生产路由始终注入 perm。
func (r *Resolver) Enabled() bool { return r != nil && r.perm != nil }

// ResolveFromContext resolves visible groups without writing an HTTP response.
// Handlers that need module-specific logging or public error normalization can
// use this method and render the returned error through their normal error path.
func (r *Resolver) ResolveFromContext(
	c *gin.Context,
) ([]uuid.UUID, error) {
	if !r.Enabled() {
		return nil, nil
	}
	userIDVal, _ := c.Get(admin.CtxKeyUserID)
	uid, isUUID := userIDVal.(uuid.UUID)
	if !isUUID {
		return nil, commonerrors.ErrForbidden
	}
	isSuperVal, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := isSuperVal.(bool)

	groups, err := r.perm.GetUserVisibleGroupIDs(
		c.Request.Context(),
		uid,
		isSuper,
	)
	if err != nil {
		return nil, &permissionBackendError{cause: err}
	}
	return groups, nil
}

// FromContext 解析 gin ctx 中主体的可见设备组。
//
// 成功返回 (groups, true)；任一失败（拿不到 user_id / perm 报错）会 abort c
// （403/500）并返回 (nil, false)，调用方在 ok==false 时必须立即 return。
//
// 未装配强制（perm == nil）时返回 (nil, true)，即「超管等价、不限制」，保持既有
// dev/test 退化行为；生产装配始终注入 perm。
func (r *Resolver) FromContext(c *gin.Context) (groups []uuid.UUID, ok bool) {
	groups, err := r.ResolveFromContext(c)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		responseErr := err
		if errors.Is(err, commonerrors.ErrInternal) {
			responseErr = commonerrors.ErrInternal
		}
		commonerrors.AbortWithError(c, status, responseErr)
		return nil, false
	}
	return groups, true
}

// AuthorizeDeviceAccess 是「按设备 ID」校验的**唯一**规范实现，被设备按 ID 直读以及
// 各类破坏性按设备写（固件升级/回退、ufte、interop、基站/事件日志按 ID 操作）统一复用。
//
//	visibleGroups == nil   → 超管：放行。
//	reader == nil          → dev/test 退化：放行（nil-safe）。
//	len(visibleGroups)==0  → 无任何分组权限：ErrForbidden。
//	否则                   → 设备分组与 visibleGroups 有交集才放行；默认 L2 组还兼容无归属记录的历史设备。
//
// 设备是否存在由调用方上层另行判定（本函数不查 devices 表）。
func AuthorizeDeviceAccess(ctx context.Context, reader GroupReader, deviceID uuid.UUID, visibleGroups []uuid.UUID) error {
	if visibleGroups == nil {
		return nil
	}
	if reader == nil {
		return nil
	}
	realGroups, includeUngrouped := SplitVisibleGroups(visibleGroups)
	if len(realGroups) == 0 && !includeUngrouped {
		return commonerrors.ErrForbidden
	}
	deviceGroups, err := reader.GetDeviceGroupIDs(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("authorize device group access: %w", err)
	}
	if len(deviceGroups) == 0 {
		if includeUngrouped {
			return nil
		}
		return commonerrors.ErrForbidden
	}
	visible := make(map[uuid.UUID]struct{}, len(realGroups))
	for _, gid := range realGroups {
		visible[gid] = struct{}{}
	}
	for _, gid := range deviceGroups {
		if _, ok := visible[gid]; ok {
			return nil
		}
	}
	// 设备所有分组都不在可见集合 → 越权。
	return commonerrors.ErrForbidden
}

// AuthorizeDevicesAccess 对一批 deviceID 逐个做归属校验，任一越权即返回该错误（fail-fast）。
// 用于批量升级/回退/ufte 等：只要请求体里混入域外设备即整批拒绝。
func AuthorizeDevicesAccess(ctx context.Context, reader GroupReader, visibleGroups []uuid.UUID, deviceIDs ...uuid.UUID) error {
	if visibleGroups == nil || reader == nil {
		return nil
	}
	if len(visibleGroups) == 0 {
		return commonerrors.ErrForbidden
	}
	for _, id := range deviceIDs {
		if err := AuthorizeDeviceAccess(ctx, reader, id, visibleGroups); err != nil {
			return err
		}
	}
	return nil
}

// ApplyDeviceVisibilityFilter 给一个 squirrel SelectBuilder 施加规范的三态 fail-closed
// 设备组过滤，把结果行限定到 visibleGroups 下的设备。
//
// deviceIDColumn 是驱动表上持有设备 UUID 的限定列名（如 "alarms_active.device_id"）。
// 用相关子查询而非 JOIN device_group_members，避免设备属多组时行翻倍：
//
//	nil       → 不过滤（超管）
//	[]        → WHERE FALSE（fail-closed）
//	[g1,...]  → 按 device_group_members 归属过滤；包含默认 L2 组时额外兼容无归属记录的历史设备
func ApplyDeviceVisibilityFilter(b sq.SelectBuilder, deviceIDColumn string, visibleGroups []uuid.UUID) sq.SelectBuilder {
	if visibleGroups == nil {
		return b
	}
	if len(visibleGroups) == 0 {
		return b.Where(sq.Expr("FALSE"))
	}
	realGroups, includeUngrouped := SplitVisibleGroups(visibleGroups)
	clauses := make(sq.Or, 0, 2)
	if len(realGroups) > 0 {
		// 子查询用默认 Question(?) 占位符（非 storage.Psql 的 Dollar）：被外层 Sqlizer
		// 内嵌时占位符随顶层 builder 统一重排为 $N，避免与外层已有 WHERE 的 $N 撞号。
		sub := sq.Select("device_id").
			From("device_group_members").
			Where(sq.Eq{"group_id": realGroups})
		clauses = append(clauses, sq.Expr(deviceIDColumn+" IN (?)", sub))
	}
	if includeUngrouped {
		clauses = append(clauses, sq.Expr("NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = "+deviceIDColumn+")"))
	}
	if len(clauses) == 0 {
		return b.Where(sq.Expr("FALSE"))
	}
	if len(clauses) == 1 {
		return b.Where(clauses[0])
	}
	return b.Where(clauses)
}

// AuthorizeDeviceAccessByGrants checks whether a device's group memberships and
// technology satisfy at least one visibility grant.
func AuthorizeDeviceAccessByGrants(deviceGroups []uuid.UUID, deviceTechnology model.Technology, grants []model.DeviceVisibilityGrant) error {
	if grants == nil {
		return nil
	}
	if len(grants) == 0 {
		return commonerrors.ErrForbidden
	}
	if len(deviceGroups) == 0 {
		for _, grant := range grants {
			if !grantAllowsTechnology(grant, deviceTechnology) {
				continue
			}
			if grantIncludesUngrouped(grant.GroupIDs) {
				return nil
			}
		}
		return commonerrors.ErrForbidden
	}
	for _, grant := range grants {
		if !grantAllowsTechnology(grant, deviceTechnology) {
			continue
		}
		if intersectsGrantGroups(deviceGroups, grant.GroupIDs) {
			return nil
		}
	}
	return commonerrors.ErrForbidden
}

// ApplyDeviceVisibilityGrantsFilter constrains a SELECT to devices visible under
// a set of group+technology grants. nil means superadmin (no filtering);
// empty slice means authenticated but no visible device scope.
func ApplyDeviceVisibilityGrantsFilter(b sq.SelectBuilder, deviceIDColumn, technologyColumn string, grants []model.DeviceVisibilityGrant) sq.SelectBuilder {
	if grants == nil {
		return b
	}
	if len(grants) == 0 {
		return b.Where(sq.Expr("FALSE"))
	}
	clauses := make(sq.Or, 0, len(grants))
	for _, grant := range grants {
		if len(grant.GroupIDs) == 0 {
			continue
		}
		groupIDs, includeUngrouped := SplitVisibleGroups(grant.GroupIDs)
		predicates := make(sq.Or, 0, 2)
		if len(groupIDs) > 0 {
			sub := sq.Select("device_id").
				From("device_group_members").
				Where(sq.Eq{"group_id": groupIDs})
			predicates = append(predicates, sq.Expr(deviceIDColumn+" IN (?)", sub))
		}
		if includeUngrouped {
			predicates = append(predicates, sq.Expr("NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = "+deviceIDColumn+")"))
		}
		if len(predicates) == 0 {
			continue
		}
		expr := predicates[0]
		if len(predicates) > 1 {
			expr = predicates
		}
		if len(grant.Technologies) > 0 {
			expr = sq.And{expr, sq.Eq{technologyColumn: grant.Technologies}}
		}
		clauses = append(clauses, expr)
	}
	if len(clauses) == 0 {
		return b.Where(sq.Expr("FALSE"))
	}
	if len(clauses) == 1 {
		return b.Where(clauses[0])
	}
	return b.Where(clauses)
}

func grantAllowsTechnology(grant model.DeviceVisibilityGrant, deviceTechnology model.Technology) bool {
	if len(grant.Technologies) == 0 {
		return true
	}
	for _, tech := range grant.Technologies {
		if tech == deviceTechnology {
			return true
		}
	}
	return false
}

func intersectsGrantGroups(deviceGroups []uuid.UUID, grantGroups []uuid.UUID) bool {
	if len(grantGroups) == 0 {
		return false
	}
	if len(deviceGroups) == 0 {
		return grantIncludesUngrouped(grantGroups)
	}
	visible := make(map[uuid.UUID]struct{}, len(grantGroups))
	for _, gid := range grantGroups {
		visible[gid] = struct{}{}
	}
	for _, gid := range deviceGroups {
		if _, ok := visible[gid]; ok {
			return true
		}
	}
	return false
}

func grantIncludesUngrouped(grantGroups []uuid.UUID) bool {
	for _, gid := range grantGroups {
		if gid.String() == global.DefaultLevel2GroupID {
			return true
		}
	}
	return false
}

// ApplyDeviceSNVisibilityFilter 是 ApplyDeviceVisibilityFilter 的「按设备序列号」变体，
// 用于 PM 时序表（pm_metrics / pm_metrics_hourly / …）这类**不持有 device_id UUID 列、
// 而以 device_sn 为设备键**的表（TR-069 标准 (oui, sn) 双键）。
//
// snColumn 是驱动表上持有设备序列号的限定列名（如 "device_sn"）。两层相关子查询把可见分组
// → device_id → devices.serial_number 映射，避免在 PM 大表上 JOIN device_group_members ×
// devices 造成行翻倍。三态契约与 ApplyDeviceVisibilityFilter 一致：
//
//	nil       → 不过滤（超管）
//	[]        → WHERE FALSE（fail-closed）
//	[g1,...]  → 按 device_group_members 归属映射到设备序列号；包含默认 L2 组时
//	             额外兼容无归属记录的历史设备
func ApplyDeviceSNVisibilityFilter(b sq.SelectBuilder, snColumn string, visibleGroups []uuid.UUID) sq.SelectBuilder {
	if visibleGroups == nil {
		return b
	}
	if len(visibleGroups) == 0 {
		return b.Where(sq.Expr("FALSE"))
	}
	realGroups, includeUngrouped := SplitVisibleGroups(visibleGroups)
	clauses := make(sq.Or, 0, 2)
	if len(realGroups) > 0 {
		sub := sq.Select("serial_number").
			From("devices").
			Where(sq.Expr("id IN (?)",
				sq.Select("device_id").
					From("device_group_members").
					Where(sq.Eq{"group_id": realGroups})))
		clauses = append(clauses, sq.Expr(snColumn+" IN (?)", sub))
	}
	if includeUngrouped {
		clauses = append(clauses, sq.Expr(snColumn+" IN (SELECT serial_number FROM devices d WHERE NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id))"))
	}
	if len(clauses) == 0 {
		return b.Where(sq.Expr("FALSE"))
	}
	if len(clauses) == 1 {
		return b.Where(clauses[0])
	}
	return b.Where(clauses)
}

// ApplyGroupVisibilityFilter 给设备组维度表（pm_group_metrics_* 等以 device_group_id 为键的表）
// 施加三态 fail-closed 过滤：把结果行限定到 visibleGroups 内的设备组。
//
// groupIDColumn 是驱动表上持有设备组 UUID 的限定列名（如 "device_group_id"）。
//
//	nil       → 不过滤（超管）
//	[]        → WHERE FALSE（fail-closed）
//	[g1,...]  → WHERE groupIDColumn IN (g1, ...)
//
// 与按设备过滤不同，这里直接对组 id 取交（请求侧若已带 device_group_id 过滤，二者叠加即交集）。
func ApplyGroupVisibilityFilter(b sq.SelectBuilder, groupIDColumn string, visibleGroups []uuid.UUID) sq.SelectBuilder {
	if visibleGroups == nil {
		return b
	}
	if len(visibleGroups) == 0 {
		return b.Where(sq.Expr("FALSE"))
	}
	return b.Where(sq.Eq{groupIDColumn: visibleGroups})
}

// VisibleSNSubquerySQL 返回「可见分组 → 设备序列号」子查询的裸 SQL 片段与参数，
// 供 product / band / network 维度等**手拼 SQL（带 CTE/JOIN，无 squirrel builder）**的聚合路径
// 复用同一收口逻辑。返回的 SQL 形如：
//
//	device_sn IN (SELECT serial_number FROM devices WHERE id IN (
//	    SELECT device_id FROM device_group_members WHERE group_id = ANY($N)))
//
// 调用方负责把 placeholder（用传入的 paramRef，如 "$3"）与 args 拼进自己的 WHERE。
// nil → 返回 ("", nil)（不过滤）；[] → 返回 ("FALSE", nil)（fail-closed）。
// visibleGroups 包含默认 L2 组时，SQL 额外 OR 无归属记录的历史设备分支。
func VisibleSNSubquerySQL(snColumn, paramRef string, visibleGroups []uuid.UUID) (string, []uuid.UUID) {
	if visibleGroups == nil {
		return "", nil
	}
	if len(visibleGroups) == 0 {
		return "FALSE", nil
	}
	realGroups, includeUngrouped := SplitVisibleGroups(visibleGroups)
	if len(realGroups) == 0 && !includeUngrouped {
		return "FALSE", nil
	}
	parts := make([]string, 0, 2)
	if len(realGroups) > 0 {
		parts = append(parts, snColumn+" IN (SELECT serial_number FROM devices WHERE id IN ("+
			"SELECT device_id FROM device_group_members WHERE group_id = ANY("+paramRef+")))")
	}
	if includeUngrouped {
		parts = append(parts, snColumn+" IN (SELECT serial_number FROM devices d WHERE NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id))")
	}
	if len(parts) == 1 {
		if len(realGroups) > 0 {
			return parts[0], realGroups
		}
		return parts[0], nil
	}
	return "(" + parts[0] + " OR " + parts[1] + ")", realGroups
}
