package topology

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DeviceLister 设备列举器（消费侧 narrow 接口，单方法）。
// 由 GroupMatchEngine 消费 —— 周期/事件触发 reEvaluate 时拉全量设备做匹配。
type DeviceLister interface {
	ListAllForRuleEval(ctx context.Context) ([]DeviceForMatch, error)
}

// DeviceForMatch 匹配阶段需要的设备最小信息集。LAC/TAC 字段当前未填充
// （devices/device_info/sites 三表均无 lac/tac 列），matcher 对 nil 安全降级。
type DeviceForMatch struct {
	ID           uuid.UUID
	Name         string
	SerialNumber string
	LAC          *int
	TAC          *int
}

// PgDeviceLister DeviceLister 的 PostgreSQL 实现。
//
// 从 devices 表取出供匹配的最小信息集（ID + Name + SerialNumber）。
// Name = devices.site_name（站点名称），即"名称匹配"模式的匹配字段。
// LAC / TAC 字段返 nil —— matcher.go matchByCode 对 nil 安全降级返 false，
// 因此 LAC/TAC 匹配 mode 在生产无数据源情况下天然不命中（不会 panic、不会误匹配）。
type PgDeviceLister struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPgDeviceLister 构造器。
func NewPgDeviceLister(pool *pgxpool.Pool, logger *zap.Logger) *PgDeviceLister {
	return &PgDeviceLister{
		pool:   pool,
		logger: logger,
	}
}

// ListAllForRuleEval 返回所有可被规则匹配的设备的最小信息集。
//
// SQL：Name 取 devices.site_name（"名称匹配"模式的匹配字段，与设备列表"名称"列
// 一致）。site_name 为空时 Name 为空串 —— matchByDeviceName 对空串安全降级
// 不命中（见 matcher.go），未配站点名的设备天然不参与名称匹配。SN 匹配走
// 独立的 serial_number 字段（MatchingModeSerialNumber）。
//
// 性能：当前一次性返全部设备；100 万规模下需切换流式 cursor 或分批 OFFSET
// （PRD §7 反例监控目标 cron CPU < 50% 隐含分批要求）— 后续优化项。
func (l *PgDeviceLister) ListAllForRuleEval(ctx context.Context) ([]DeviceForMatch, error) {
	const sqlText = `
		SELECT d.id, COALESCE(d.site_name, '') AS name, d.serial_number
		FROM devices d
	`

	rows, err := l.pool.Query(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("query devices for rule eval: %w", err)
	}
	defer rows.Close()

	devices := make([]DeviceForMatch, 0)
	for rows.Next() {
		var d DeviceForMatch
		// migration 000124：补 serial_number 字段供 SN 模式匹配。
		if err := rows.Scan(&d.ID, &d.Name, &d.SerialNumber); err != nil {
			return nil, fmt.Errorf("scan device row: %w", err)
		}
		devices = append(devices, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device rows: %w", err)
	}

	return devices, nil
}
