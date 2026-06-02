package rebootrecord

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository 统一重启记录的只读查询层（UNION event_logs + station_fault_logs）。
type Repository interface {
	List(ctx context.Context, f Filter) ([]*RebootRecord, int64, error)
	StatByDevice(ctx context.Context, f Filter) ([]*DeviceRebootStat, error)
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// buildUnion 组装两表 UNION ALL 的子查询。
//
// 两半共用同一组过滤值（占位符复用）：device_sn / device_type / start / end 在
// 两个 SELECT 里引用相同的 $N，只是时间列名不同（occurred_at vs collected_at）。
// 所有用户输入都走参数化占位符，无字符串拼接注入。
//
// 投影列在两半严格对齐（名称 + 类型），可空列一律 COALESCE 成零值，便于扫描。
func buildUnion(f Filter) (string, []interface{}) {
	var args []interface{}
	// 共享过滤：(event 半子句, fault 半子句) —— 仅时间列名不同
	type cond struct{ event, fault string }
	var conds []cond
	add := func(val interface{}, mk func(ph string) (string, string)) {
		args = append(args, val)
		ph := fmt.Sprintf("$%d", len(args))
		e, fa := mk(ph)
		conds = append(conds, cond{e, fa})
	}

	if f.DeviceSN != "" {
		add("%"+f.DeviceSN+"%", func(ph string) (string, string) {
			return "device_sn ILIKE " + ph, "device_sn ILIKE " + ph
		})
	}
	if f.DeviceType != "" {
		add(f.DeviceType, func(ph string) (string, string) {
			return "device_type = " + ph, "device_type = " + ph
		})
	}
	if f.StartTime != nil {
		add(*f.StartTime, func(ph string) (string, string) {
			return "occurred_at >= " + ph, "collected_at >= " + ph
		})
	}
	if f.EndTime != nil {
		add(*f.EndTime, func(ph string) (string, string) {
			return "occurred_at <= " + ph, "collected_at <= " + ph
		})
	}

	eventWhere := []string{"event_type = 'boot'"}
	faultWhere := []string{"is_deleted = false"}
	for _, c := range conds {
		eventWhere = append(eventWhere, c.event)
		faultWhere = append(faultWhere, c.fault)
	}

	eventSelect := `SELECT id::text AS id, 'event' AS source, false AS is_abnormal, device_sn,
		COALESCE(device_name, '') AS device_name, COALESCE(device_type, '') AS device_type,
		COALESCE(operate_ip, '') AS operate_ip, COALESCE(software_version, '') AS software_version,
		COALESCE(event_reason, '') AS reason, ''::text AS detail_reason,
		COALESCE((event_data->>'runtime_before_reboot')::bigint, 0) AS runtime_before_reboot,
		occurred_at AS reboot_time
		FROM event_logs WHERE ` + strings.Join(eventWhere, " AND ")

	faultSelect := `SELECT id::text AS id, 'fault' AS source, true AS is_abnormal, device_sn,
		COALESCE(device_name, '') AS device_name, COALESCE(device_type, '') AS device_type,
		COALESCE(operate_ip, '') AS operate_ip, COALESCE(software_version, '') AS software_version,
		COALESCE(fault_reason, '') AS reason, COALESCE(fault_detail, '') AS detail_reason,
		COALESCE(runtime_before_reboot, 0) AS runtime_before_reboot, collected_at AS reboot_time
		FROM station_fault_logs WHERE ` + strings.Join(faultWhere, " AND ")

	var parts []string
	switch f.RebootType {
	case RebootTypeNormal:
		parts = []string{eventSelect}
	case RebootTypeAbnormal:
		parts = []string{faultSelect}
	default:
		parts = []string{eventSelect, faultSelect}
	}
	return "(" + strings.Join(parts, " UNION ALL ") + ")", args
}

func (r *PgRepository) List(ctx context.Context, f Filter) ([]*RebootRecord, int64, error) {
	union, args := buildUnion(f)

	var total int64
	countSQL := "SELECT COUNT(*) FROM " + union + " AS t"
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reboot records: %w", err)
	}

	limitPH := fmt.Sprintf("$%d", len(args)+1)
	offsetPH := fmt.Sprintf("$%d", len(args)+2)
	pageArgs := append(append([]interface{}{}, args...), f.Limit(), f.Offset())
	listSQL := "SELECT id, source, is_abnormal, device_sn, device_name, device_type, operate_ip, " +
		"software_version, reason, detail_reason, runtime_before_reboot, reboot_time FROM " + union +
		" AS t ORDER BY reboot_time DESC LIMIT " + limitPH + " OFFSET " + offsetPH

	rows, err := r.pool.Query(ctx, listSQL, pageArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query reboot records: %w", err)
	}
	defer rows.Close()

	var items []*RebootRecord
	for rows.Next() {
		var rec RebootRecord
		if err := rows.Scan(
			&rec.ID, &rec.Source, &rec.IsAbnormal, &rec.DeviceSN, &rec.DeviceName,
			&rec.DeviceType, &rec.OperateIP, &rec.SoftwareVersion, &rec.Reason,
			&rec.DetailReason, &rec.RuntimeBeforeReboot, &rec.RebootTime,
		); err != nil {
			return nil, 0, fmt.Errorf("scan reboot record: %w", err)
		}
		items = append(items, &rec)
	}
	return items, total, rows.Err()
}

func (r *PgRepository) StatByDevice(ctx context.Context, f Filter) ([]*DeviceRebootStat, error) {
	union, args := buildUnion(f)

	statSQL := `SELECT device_sn,
		(ARRAY_AGG(device_name ORDER BY reboot_time DESC))[1] AS device_name,
		COUNT(*) AS total_count,
		COUNT(*) FILTER (WHERE is_abnormal) AS abnormal_count,
		MAX(reboot_time) AS latest_at
		FROM ` + union + ` AS t
		GROUP BY device_sn
		ORDER BY total_count DESC, device_sn ASC`

	rows, err := r.pool.Query(ctx, statSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query reboot record stat: %w", err)
	}
	defer rows.Close()

	var items []*DeviceRebootStat
	for rows.Next() {
		var (
			s          DeviceRebootStat
			deviceName *string
		)
		if err := rows.Scan(&s.DeviceSN, &deviceName, &s.TotalCount, &s.AbnormalCount, &s.LatestAt); err != nil {
			return nil, fmt.Errorf("scan reboot record stat: %w", err)
		}
		if deviceName != nil {
			s.DeviceName = *deviceName
		}
		items = append(items, &s)
	}
	return items, rows.Err()
}
