package pm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// #18 分层收敛：把 PM Handler 原先直连 SQL 池（h.pool）的两处查询收敛到
// Service 层（handler → service → repository）。Handler 不再持有
// *pgxpool.Pool；跨模块读 devices 表、读 pm_metrics 明细都经此 Service，
// 便于后续在单点挂权限检查 / 缓存策略。行为保持不变（SQL 与原实现逐字一致）。

// MetricObject 是设备在 pm_metrics 里出现过的一个 distinct 小区/PLMN 项。
type MetricObject struct {
	ObjectLDN string // 原始字符串
	CellID    string // 解析出的小区号（友好名用）
	PLMN      string // 解析出的 PLMN（友好名用）
}

// DeviceQueryService 封装 PM 模块对 devices / pm_metrics 的只读查询。
//
// 接口优先：Handler 依赖此接口而非具体 *pgxpool.Pool，便于测试替身与未来
// 切换数据源（例如改走 Device 模块的公开接口）时不动 Handler。
type DeviceQueryService interface {
	// LookupDeviceOUISN 反查 devices 表的 (oui, serial_number) 双键。
	// 给 KPIEngine.CalculateAndStore 提供设备主键（原 T-0164-P3 fix）。
	LookupDeviceOUISN(ctx context.Context, deviceID uuid.UUID) (oui, sn string, err error)

	// ListMetricObjects 列出一批设备在最细原始表 pm_metrics 里实际出现过的
	// distinct object_ldn（按 technology 可选过滤），每项解析出 cell_id / plmn。
	ListMetricObjects(ctx context.Context, deviceSNs, technologies []string, startTime, endTime time.Time) ([]MetricObject, error)

	// DeviceGroupIDs 读取单个设备所属的设备组 id 列表（device_group_members）。
	// #64：PM handler 在请求显式带 device_id 时预检其是否落在调用者可见分组内，
	// 复用本服务的连接池，避免反向 import device 模块。
	DeviceGroupIDs(ctx context.Context, deviceID uuid.UUID) ([]uuid.UUID, error)
}

// pgDeviceQueryService 是 DeviceQueryService 的 PostgreSQL/TimescaleDB 实现。
type pgDeviceQueryService struct {
	pool *pgxpool.Pool
}

// NewDeviceQueryService 用时序库连接池构造 PM 设备/指标只读查询服务。
func NewDeviceQueryService(pool *pgxpool.Pool) DeviceQueryService {
	return &pgDeviceQueryService{pool: pool}
}

// LookupDeviceOUISN 反查 devices 表的 (oui, serial_number) 双键。
func (s *pgDeviceQueryService) LookupDeviceOUISN(ctx context.Context, deviceID uuid.UUID) (string, string, error) {
	var oui, sn string
	// s.pool 是 TsPool；devices 反查改读本库影子表 device_dim（跨库分离）。
	err := s.pool.QueryRow(ctx,
		`SELECT oui, serial_number FROM device_dim WHERE id = $1`, deviceID,
	).Scan(&oui, &sn)
	if err != nil {
		return "", "", fmt.Errorf("lookup device oui+sn: %w", err)
	}
	return oui, sn, nil
}

// DeviceGroupIDs 读取设备所属分组 id 列表（device_group_members）。设备未分组返回空切片。
func (s *pgDeviceQueryService) DeviceGroupIDs(ctx context.Context, deviceID uuid.UUID) ([]uuid.UUID, error) {
	// s.pool 是 TsPool；device_group_members 改读本库影子表 device_group_member_dim。
	rows, err := s.pool.Query(ctx,
		`SELECT group_id FROM device_group_member_dim WHERE device_id = $1`, deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("query device group ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan group_id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device group ids: %w", err)
	}
	return ids, nil
}

// ListMetricObjects 查询设备小区/PLMN 清单。
func (s *pgDeviceQueryService) ListMetricObjects(ctx context.Context, deviceSNs, technologies []string, startTime, endTime time.Time) ([]MetricObject, error) {
	q, args := buildObjectsQuery(deviceSNs, technologies, startTime, endTime)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query metric objects: %w", err)
	}
	defer rows.Close()

	items := make([]MetricObject, 0)
	for rows.Next() {
		var ldn string
		if err := rows.Scan(&ldn); err != nil {
			return nil, fmt.Errorf("scan object_ldn: %w", err)
		}
		cellID, plmn := parseObjectLDN(ldn)
		items = append(items, MetricObject{ObjectLDN: ldn, CellID: cellID, PLMN: plmn})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric objects: %w", err)
	}
	return items, nil
}

// buildObjectsQuery 纯函数：拼"列设备小区/PLMN"查询 SQL + 占位参数。
//
//   - $1 = device_sns（TEXT[]）
//   - 制式过滤（technologies 非空时）照 applyCommonFilters 范式：
//     d.technology = ANY($n)
//     （查询跑在 TsPool，devices 用本库影子表 device_dim）
//
// 抽出便于单测断言（device 过滤 + 制式过滤 + distinct）。
func buildObjectsQuery(deviceSNs, technologies []string, startTime, endTime time.Time) (string, []any) {
	q := `WITH target_devices AS MATERIALIZED (
  SELECT id
  FROM device_dim
  WHERE serial_number = ANY($1)`
	args := []any{deviceSNs}
	if len(technologies) > 0 {
		args = append(args, technologies)
		q += fmt.Sprintf(`
    AND technology = ANY($%d)`, len(args))
	}
	q += `
)
SELECT DISTINCT object_ldn
FROM target_devices d
JOIN pm_measurement_anchors a ON a.device_dim_id=d.id
WHERE a.granularity = '15min'
  AND a.object_ldn <> ''`
	if !startTime.IsZero() {
		args = append(args, startTime)
		q += fmt.Sprintf(`
  AND a."time" >= $%d`, len(args))
	}
	if !endTime.IsZero() {
		args = append(args, endTime)
		q += fmt.Sprintf(`
  AND a."time" < $%d`, len(args))
	}
	q += `
ORDER BY object_ldn`
	return q, args
}

// parseObjectLDN 从原始 object_ldn 拆出 cell_id / plmn（缺段则留空）。
// 委托给 metrics.ParseObjectLDN 单一真值源（与 aggregator KPI 跨层级配对同口径），不另造解析。
// 对 5G/GSM 行：cellID/plmn 留空（设计上 metrics 包不把 NrCGI/Uid 塞这两个字段），
// MetricObject 仍能用 BaseCellID 取天然小区标识，但本服务暴露的 (cellID, plmn) 维持 4G 语义。
func parseObjectLDN(ldn string) (cellID, plmn string) {
	r := metrics.ParseObjectLDN(ldn)
	return r.CellID, r.Plmn
}
