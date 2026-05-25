package topology

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DeviceLister 设备列举器（消费侧 narrow 接口）。
// 由 GroupMatchEngine 消费 —— 周期/事件触发 reEvaluate 时拉全量设备做匹配；
// 收到 device.attributes.changed 单设备事件时通过 GetByID 拿最新 LAC/TAC。
type DeviceLister interface {
	ListAllForRuleEval(ctx context.Context) ([]DeviceForMatch, error)
	GetByID(ctx context.Context, deviceID uuid.UUID) (*DeviceForMatch, error)
}

// DeviceForMatch 匹配阶段需要的设备最小信息集。
//
// LAC / TAC 来源于 device_info.lac / device_info.tac（migration 000182 补齐
// LAC 列；TAC 列从 Phase 2 起就存在）。CPE Inform 由 InfoSyncer.SyncFromParameters
// 通过 universalInformMapping 解析 Device.DeviceInfo.BTS.CurrentLac /
// Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC 写入。
//
// VARCHAR(16) → *int 转换在 SQL 层用 NULLIF + CAST 完成，无法解析为整数的
// 值（脏数据 / 厂商扩展格式）回落为 nil；matcher.go matchByCode 对 nil 安全
// 降级返 false（不命中也不 panic）。
type DeviceForMatch struct {
	ID           uuid.UUID
	Name         string
	SerialNumber string
	LAC          *int
	TAC          *int
}

// PgDeviceLister DeviceLister 的 PostgreSQL 实现。
//
// 从 devices LEFT JOIN device_info 取出供匹配的最小信息集：
//   - ID / SerialNumber 来自 devices（必有）
//   - Name = devices.site_name（"设备名称"匹配模式的字段，与设备列表 / 分组页一致）
//   - LAC / TAC 来自 device_info（可空 —— 设备未上报 / Inform 解析失败时为 nil）
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
// SQL 字段来源：
//   - Name = COALESCE(NULLIF(site_name,''), serial_number)：
//     site_name 非空 → 用 site_name；site_name 空/NULL → 回落 SN。与前端 mapper
//     friendlyName=device_name||serial_number 完全对齐，保证"UI 上看到什么、
//     '设备名称' 匹配模式就拿什么比对"——避免 site_name 空的设备被悄悄排除。
//   - SerialNumber 取 devices.serial_number（SN 匹配模式，migration 000124 补齐）。
//   - LAC / TAC 取 device_info.lac / device_info.tac，VARCHAR(16) 用 NULLIF +
//     CAST 转 INTEGER；解析失败的脏数据回落为 NULL → DeviceForMatch.LAC/TAC = nil
//     → matcher 安全降级不命中，不会 panic。
//
// 性能：当前一次性返全部设备；100 万规模下需切换流式 cursor 或分批 OFFSET
// （PRD §7 反例监控目标 cron CPU < 50% 隐含分批要求）— 后续优化项。
func (l *PgDeviceLister) ListAllForRuleEval(ctx context.Context) ([]DeviceForMatch, error) {
	// NULLIF 把空串视为 NULL，避免 CAST '' 抛错；regexp 进一步过滤非纯整数串
	// （如 "0x1234" / "100,200"），让 CAST 只接收能稳定解析的输入。
	const sqlText = `
		SELECT
			d.id,
			COALESCE(NULLIF(d.site_name, ''), d.serial_number) AS name,
			d.serial_number,
			CASE WHEN di.lac ~ '^-?[0-9]+$' THEN di.lac::int ELSE NULL END AS lac,
			CASE WHEN di.tac ~ '^-?[0-9]+$' THEN di.tac::int ELSE NULL END AS tac
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
		if err := rows.Scan(&d.ID, &d.Name, &d.SerialNumber, &d.LAC, &d.TAC); err != nil {
			return nil, fmt.Errorf("scan device row: %w", err)
		}
		devices = append(devices, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device rows: %w", err)
	}

	return devices, nil
}

// GetByID 返回单台设备的匹配信息集，供 device.attributes.changed 单设备路径使用。
// 字段语义、LAC/TAC 解析行为与 ListAllForRuleEval 保持一致。设备不存在返 (nil, nil)。
func (l *PgDeviceLister) GetByID(ctx context.Context, deviceID uuid.UUID) (*DeviceForMatch, error) {
	const sqlText = `
		SELECT
			d.id,
			COALESCE(NULLIF(d.site_name, ''), d.serial_number) AS name,
			d.serial_number,
			CASE WHEN di.lac ~ '^-?[0-9]+$' THEN di.lac::int ELSE NULL END AS lac,
			CASE WHEN di.tac ~ '^-?[0-9]+$' THEN di.tac::int ELSE NULL END AS tac
		FROM devices d
		LEFT JOIN device_info di ON di.device_id = d.id
		WHERE d.id = $1
	`
	var d DeviceForMatch
	if err := l.pool.QueryRow(ctx, sqlText, deviceID).Scan(&d.ID, &d.Name, &d.SerialNumber, &d.LAC, &d.TAC); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query device for match: %w", err)
	}
	return &d, nil
}
