package device

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
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

// AuthorizeDeviceGroupAccess 校验调用者（由其可见设备组 ID 表征）
// 是否有权按 ID 直读指定设备。该方法保留给 northbound / 旧调用链使用，只看组归属。
func (s *DeviceService) AuthorizeDeviceGroupAccess(ctx context.Context, deviceID uuid.UUID, visibleGroups []uuid.UUID) error {
	if visibleGroups == nil {
		return nil
	}
	if s.groupReader == nil || s.deviceRepo == nil {
		return nil
	}
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("authorize device access: %w", err)
	}
	if device == nil {
		return commonerrors.ErrForbidden
	}
	deviceGroups, err := s.groupReader.GetDeviceGroupIDs(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("authorize device group access: %w", err)
	}
	if len(deviceGroups) == 0 {
		for _, groupID := range visibleGroups {
			if groupID.String() == global.DefaultLevel2GroupID {
				return nil
			}
		}
		return commonerrors.ErrForbidden
	}
	for _, visibleGroup := range visibleGroups {
		for _, deviceGroup := range deviceGroups {
			if visibleGroup == deviceGroup {
				return nil
			}
		}
	}
	return commonerrors.ErrForbidden
}

// AuthorizeDeviceGroupAccessByGrants 校验调用者（由其可见设备数据 grants 表征）
// 是否有权按 ID 直读指定设备。语义与 ListDevicesWithInfo 的 VisibleDeviceGrants
// 过滤完全对齐（device_info_pg_repository.go §Data permission filter）：
//
//	grants == nil                → 超管（PermissionService 上游 source='builtIn'），放行
//	len(grants) == 0             → 无任何数据权限，拒绝（ErrForbidden）
//	[grant1, grant2, ...]        → 设备分组与制式同时命中任一 grant 才放行
//
// 未注入 DeviceGroupReader（dev/test 退化）→ 放行（nil-safe，不降低生产安全：
// 生产路由始终注入 reader + permService）。返回 nil 表示放行；ErrForbidden 表示
// 越权（handler 映射为 403）。设备是否存在由调用方上层另行判定（本方法不查 devices 表）。
func (s *DeviceService) AuthorizeDeviceGroupAccessByGrants(ctx context.Context, deviceID uuid.UUID, visibleGrants []model.DeviceVisibilityGrant) error {
	if visibleGrants == nil {
		return nil
	}
	if s.groupReader == nil || s.deviceRepo == nil {
		return nil
	}
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("authorize device access: %w", err)
	}
	if device == nil {
		return commonerrors.ErrForbidden
	}
	deviceGroups, err := s.groupReader.GetDeviceGroupIDs(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("authorize device group access: %w", err)
	}
	return authz.AuthorizeDeviceAccessByGrants(deviceGroups, device.Technology, visibleGrants)
}
