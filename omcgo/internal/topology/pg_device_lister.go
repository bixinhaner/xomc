package topology

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PgDeviceLister DeviceLister 的 PostgreSQL 实现。
//
// 用 LEFT JOIN devices + device_info 取出供 rule 匹配的最小信息集（ID + Name）。
// LAC / TAC 字段返 nil — 见 PRD §12.8 W3 待定点（pre-existing 缺口：
// devices/device_info/sites 三表均无 lac/tac 列），后续 carve out T-0098 处理。
//
// matcher.go matchByCode 对 nil 安全降级返 false，因此 LAC/TAC 匹配 mode
// 在生产无数据源情况下天然不命中（不会 panic、不会误匹配）。
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
// SQL：LEFT JOIN device_info 取 device_name；缺失时 fallback 到
// devices.serial_number 作为 Name（确保 Name 字段非空可参与 NameRule 匹配）。
//
// 性能：当前一次性返全部设备；100 万规模下需切换流式 cursor 或分批 OFFSET
// （PRD §7 反例监控目标 cron CPU < 50% 隐含分批要求）— 后续优化项。
func (l *PgDeviceLister) ListAllForRuleEval(ctx context.Context) ([]DeviceForMatch, error) {
	const sqlText = `
		SELECT d.id, COALESCE(NULLIF(di.device_name, ''), d.serial_number) AS name
		FROM devices d
		LEFT JOIN device_info di ON di.device_id = d.id
	`

	rows, err := l.pool.Query(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("query devices for rule eval: %w", err)
	}
	defer rows.Close()

	devices := make([]DeviceForMatch, 0)
	for rows.Next() {
		var d DeviceForMatch
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, fmt.Errorf("scan device row: %w", err)
		}
		devices = append(devices, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device rows: %w", err)
	}

	return devices, nil
}
