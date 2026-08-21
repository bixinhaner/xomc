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
// 两个 SELECT 里引用相同的 $N，只是时间列名不同。
// 异常重启用 station_fault_logs.created_at 作为检测时间；collected_at 是日志采集
// 领域的历史字段，不能拿来表示 1 BOOT 发生时间。
// 所有用户输入都走参数化占位符，无字符串拼接注入。
//
// 投影列在两半严格对齐（名称 + 类型），可空列一律 COALESCE 成零值，便于扫描。
func buildUnion(f Filter) (string, []interface{}) {
	var args []interface{}
	eventDeviceSN := "COALESCE(NULLIF(el.device_sn, ''), d.serial_number, '')"
	faultDeviceSN := "COALESCE(NULLIF(fl.device_sn, ''), fd.serial_number, '')"
	eventDeviceName := "COALESCE(NULLIF(el.device_name, ''), NULLIF(di.device_name, ''), d.site_name, '')"
	faultDeviceName := "COALESCE(NULLIF(fl.device_name, ''), NULLIF(fdi.device_name, ''), fd.site_name, '')"
	eventDeviceType := "COALESCE(NULLIF(CASE WHEN UPPER(COALESCE(d.product_class, '')) LIKE 'UPS%' THEN 'UPS' WHEN d.technology = 'nr' THEN 'gNB' WHEN d.technology = 'lte' THEN 'eNB' WHEN d.technology = 'gsm' THEN 'GSM' ELSE '' END, ''), NULLIF(el.device_type, ''), '')"
	faultDeviceType := "COALESCE(NULLIF(CASE WHEN UPPER(COALESCE(fd.product_class, '')) LIKE 'UPS%' THEN 'UPS' WHEN fd.technology = 'nr' THEN 'gNB' WHEN fd.technology = 'lte' THEN 'eNB' WHEN fd.technology = 'gsm' THEN 'GSM' ELSE '' END, ''), NULLIF(fl.device_type, ''), '')"
	eventOperateIP := "COALESCE(NULLIF(NULLIF(el.operate_ip, ''), '0.0.0.0'), NULLIF(host(d.ip_address), '0.0.0.0'), '')"
	faultOperateIP := "COALESCE(NULLIF(NULLIF(fl.operate_ip, ''), '0.0.0.0'), NULLIF(host(fd.ip_address), '0.0.0.0'), '')"
	eventSoftwareVersion := "COALESCE(NULLIF(el.software_version, ''), '')"
	faultSoftwareVersion := "COALESCE(NULLIF(fl.software_version, ''), '')"

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
			return eventDeviceSN + " ILIKE " + ph, faultDeviceSN + " ILIKE " + ph
		})
	}
	if f.DeviceType != "" {
		add(f.DeviceType, func(ph string) (string, string) {
			return eventDeviceType + " = " + ph, faultDeviceType + " = " + ph
		})
	}
	if f.StartTime != nil {
		add(*f.StartTime, func(ph string) (string, string) {
			return "el.occurred_at >= " + ph, "fl.created_at >= " + ph
		})
	}
	if f.EndTime != nil {
		add(*f.EndTime, func(ph string) (string, string) {
			return "el.occurred_at <= " + ph, "fl.created_at <= " + ph
		})
	}

	eventWhere := []string{"el.event_type = 'boot'"}
	faultWhere := []string{
		"fl.is_deleted = false",
		faultDeviceSN + " <> ''",
		"NULLIF(fl.fault_reason, '') IS NOT NULL",
	}
	for _, c := range conds {
		eventWhere = append(eventWhere, c.event)
		faultWhere = append(faultWhere, c.fault)
	}

	// #63 设备组可见性 fail-closed 过滤（三态契约见 authz 包）：
	//   nil       → 超管：不加条件。
	//   []        → 无权限：两半各加 FALSE（空集）。
	//   [g1,...]  → 两半各按 device_id 关联 device_group_members 子查询收窄；
	//               device_id 为 NULL 的记录不在子查询结果内，自然被排除（fail-closed）。
	if f.VisibleGroups != nil {
		if len(f.VisibleGroups) == 0 {
			eventWhere = append(eventWhere, "FALSE")
			faultWhere = append(faultWhere, "FALSE")
		} else {
			args = append(args, f.VisibleGroups)
			ph := fmt.Sprintf("$%d", len(args))
			eventWhere = append(eventWhere, "el.device_id IN (SELECT device_id FROM device_group_members WHERE group_id = ANY("+ph+"))")
			faultWhere = append(faultWhere, "fl.device_id IN (SELECT device_id FROM device_group_members WHERE group_id = ANY("+ph+"))")
		}
	}

	eventSelect := `SELECT el.id::text AS id, 'event' AS source, false AS is_abnormal, ` + eventDeviceSN + ` AS device_sn,
		` + eventDeviceName + ` AS device_name, ` + eventDeviceType + ` AS device_type,
		` + eventOperateIP + ` AS operate_ip, ` + eventSoftwareVersion + ` AS software_version,
		COALESCE(el.event_reason, '') AS reason, ''::text AS detail_reason,
		COALESCE((el.event_data->>'runtime_before_reboot')::bigint, 0) AS runtime_before_reboot,
		el.occurred_at AS reboot_time
		FROM event_logs el
		LEFT JOIN devices d ON d.id = el.device_id
		LEFT JOIN device_info di ON di.device_id = d.id
		WHERE ` + strings.Join(eventWhere, " AND ")

	faultSelect := `SELECT fl.id::text AS id, 'fault' AS source, true AS is_abnormal, ` + faultDeviceSN + ` AS device_sn,
		` + faultDeviceName + ` AS device_name, ` + faultDeviceType + ` AS device_type,
		` + faultOperateIP + ` AS operate_ip, ` + faultSoftwareVersion + ` AS software_version,
		COALESCE(fl.fault_reason, '') AS reason, COALESCE(fl.fault_detail, '') AS detail_reason,
		COALESCE(fl.runtime_before_reboot, 0) AS runtime_before_reboot, fl.created_at AS reboot_time
		FROM station_fault_logs fl
		LEFT JOIN devices fd ON fd.id = fl.device_id
		LEFT JOIN device_info fdi ON fdi.device_id = fd.id
		WHERE ` + strings.Join(faultWhere, " AND ")

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
		(ARRAY_AGG(device_type ORDER BY reboot_time DESC) FILTER (WHERE device_type <> ''))[1] AS device_type,
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
			deviceType *string
		)
		if err := rows.Scan(&s.DeviceSN, &deviceName, &deviceType, &s.TotalCount, &s.AbnormalCount, &s.LatestAt); err != nil {
			return nil, fmt.Errorf("scan reboot record stat: %w", err)
		}
		if deviceName != nil {
			s.DeviceName = *deviceName
		}
		if deviceType != nil {
			s.DeviceType = *deviceType
		}
		items = append(items, &s)
	}
	return items, rows.Err()
}
