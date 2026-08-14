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

// CountDevices 返回当前**在线**设备数（is_online=true 且未进回收站）。
//
// 容量口径（issue #316）：license 容量限制的是"在线/接入"设备数，离线和回收站
// 设备不占容量——这样降容时把超容设备置离线即可腾出容量，无需删除。用于 enforcer
// 容量裁决与 monitor 容量阈值告警。
func (c *PgDeviceCounter) CountDevices(ctx context.Context) (int, error) {
	var count int
	if err := c.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM devices WHERE is_online = true AND deleted_at IS NULL").Scan(&count); err != nil {
		return 0, fmt.Errorf("count devices: %w", err)
	}
	return count, nil
}

// CountDevicesByType 按网元类型（products.alarm_ne_type）分组返回当前**在线**
// 设备计数，供 EnforceCapacity 的 per-type 限额使用。
//
// key 为 UPPER 归一化后的 ne_type（与 enforcer.upperKey 对齐：license key
// `eNB`/`gNB` ↔ alarm_ne_type `ENB`/`GNB` 大小写不敏感匹配）。product_id 为
// NULL 的设备（孤儿）计入空串 key ""。只数在线设备（is_online=true 且未进回收站）。
func (c *PgDeviceCounter) CountDevicesByType(ctx context.Context) (map[string]int, error) {
	const q = `SELECT UPPER(COALESCE(p.alarm_ne_type, '')) AS ne_type, COUNT(*) AS cnt
		FROM devices d
		LEFT JOIN products p ON d.product_id = p.id
		WHERE d.is_online = true AND d.deleted_at IS NULL
		GROUP BY p.alarm_ne_type`
	rows, err := c.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("count devices by type: %w", err)
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var neType string
		var cnt int
		if err := rows.Scan(&neType, &cnt); err != nil {
			return nil, fmt.Errorf("scan devices by type: %w", err)
		}
		result[neType] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate devices by type: %w", err)
	}
	return result, nil
}

// 编译期接口契约检查。
var _ DeviceCounter = (*PgDeviceCounter)(nil)
