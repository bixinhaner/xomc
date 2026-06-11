package device

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// DeviceGroupReader 读取单个设备的设备组（device_group_members）归属。
//
// 拆成独立的窄接口（而非塞进 DeviceRepository）是为了：① 不必为越权校验改动
// 庞大的 DeviceRepository 接口 / 重新生成 mock；② 越权校验是一个聚焦关注点，
// 单独的读路径更易测试与复用。nil 实现允许（dev/test 退化为不校验，见
// DeviceService.AuthorizeDeviceGroupAccess）。
type DeviceGroupReader interface {
	// GetDeviceGroupIDs 返回设备所属的全部 L2 设备组 ID。
	// 设备未分组返回空切片（非 nil）；设备不存在同样返回空切片（调用方据上层
	// GetByID 的 nil 结果区分"不存在"与"未分组"）。
	GetDeviceGroupIDs(ctx context.Context, deviceID uuid.UUID) ([]uuid.UUID, error)
}

// PgDeviceGroupReader 是 DeviceGroupReader 的 PostgreSQL 实现。
type PgDeviceGroupReader struct {
	pool *pgxpool.Pool
}

// NewPgDeviceGroupReader 构造一个 PgDeviceGroupReader。
func NewPgDeviceGroupReader(pool *pgxpool.Pool) *PgDeviceGroupReader {
	return &PgDeviceGroupReader{pool: pool}
}

// GetDeviceGroupIDs 查询 device_group_members 取设备的全部分组 ID。
func (r *PgDeviceGroupReader) GetDeviceGroupIDs(ctx context.Context, deviceID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := storage.Psql.
		Select("group_id").
		From("device_group_members").
		Where(sq.Eq{"device_id": deviceID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device group ids query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query device group ids: %w", err)
	}
	defer rows.Close()

	groupIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var gid uuid.UUID
		if scanErr := rows.Scan(&gid); scanErr != nil {
			return nil, fmt.Errorf("scan device group id: %w", scanErr)
		}
		groupIDs = append(groupIDs, gid)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate device group ids: %w", rows.Err())
	}
	return groupIDs, nil
}

// AuthorizeDeviceGroupAccess 校验调用者（由其可见设备组集合 visibleGroups 表征）
// 是否有权按 ID 直读指定设备。语义与 ListDevicesWithInfo 的 VisibleGroups 过滤
// 完全对齐（device_info_pg_repository.go §Data permission filter）：
//
//	visibleGroups == nil          → 超管（PermissionService 上游 source='builtIn'），放行
//	len(visibleGroups) == 0       → 无任何分组权限，拒绝（ErrForbidden）
//	[gid1, gid2, ...]             → 设备分组与 visibleGroups 有交集才放行，否则拒绝
//
// 未注入 DeviceGroupReader（dev/test 退化）→ 放行（nil-safe，不降低生产安全：
// 生产路由始终注入 reader + permService）。返回 nil 表示放行；ErrForbidden 表示
// 越权（handler 映射为 403）。设备是否存在由调用方上层另行判定（本方法不查 devices 表）。
func (s *DeviceService) AuthorizeDeviceGroupAccess(ctx context.Context, deviceID uuid.UUID, visibleGroups []uuid.UUID) error {
	// #64 统一强制层：委派给 authz 包的唯一实现，避免按设备归属校验逻辑在 device/
	// alarm/pm/ufte/interop/software 等多处各自复刻导致语义漂移。
	return authz.AuthorizeDeviceAccess(ctx, s.groupReader, deviceID, visibleGroups)
}
