// pg_device_counter.go — F06 System License 重构 Step 5。
//
// 老 PgLicenseRepository.CountDevices 与 LicenseRepository 接口一起删除后，
// enforcer / monitor 需要一个独立的 DeviceCounter 实现。本文件提供一个最
// 简单的 pgxpool 实现，仅 SELECT COUNT(*) FROM devices —— 与 license 表
// 完全无关，放在 license 包只是为了 DI 收敛（caller 不必跨包 wire）。
package license

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgDeviceCounter — DeviceCounter 的 PostgreSQL 实现。
type PgDeviceCounter struct {
	pool *pgxpool.Pool
}

// NewPgDeviceCounter 构造一个 device 行数计数器。
func NewPgDeviceCounter(pool *pgxpool.Pool) *PgDeviceCounter {
	return &PgDeviceCounter{pool: pool}
}

// CountDevices 返回 devices 表当前行数（不区分状态 / 不过滤删除）。
//
// 用于 enforcer 容量裁决（used vs max）与 monitor 容量阈值告警；这两者都需要
// "已纳管设备总数"语义，回收站设备暂不剔除（与老 PgLicenseRepository.CountDevices
// 完全保持一致行为）。
func (c *PgDeviceCounter) CountDevices(ctx context.Context) (int, error) {
	var count int
	if err := c.pool.QueryRow(ctx, "SELECT COUNT(*) FROM devices").Scan(&count); err != nil {
		return 0, fmt.Errorf("count devices: %w", err)
	}
	return count, nil
}

// 编译期接口契约检查。
var _ DeviceCounter = (*PgDeviceCounter)(nil)
