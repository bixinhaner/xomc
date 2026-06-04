package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgRepository 是 Repository 的 PostgreSQL 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 构造一个绑定到给定连接池的 Repository。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

const listActivePatternsSQL = `
SELECT id, product_id, product_class, sort_order, is_active
FROM product_class_patterns
WHERE is_active = TRUE
ORDER BY sort_order ASC`

// ListActivePatterns 实现 Repository。
func (r *PgRepository) ListActivePatterns(ctx context.Context) ([]ProductClassPattern, error) {
	rows, err := r.pool.Query(ctx, listActivePatternsSQL)
	if err != nil {
		return nil, fmt.Errorf("query active patterns: %w", err)
	}
	defer rows.Close()

	var out []ProductClassPattern
	for rows.Next() {
		var p ProductClassPattern
		if err := rows.Scan(&p.ID, &p.ProductID, &p.ProductClass, &p.SortOrder, &p.IsActive); err != nil {
			return nil, fmt.Errorf("scan pattern row: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

const productSelectColumns = `id, product_name, vendor, tech, radio_modes, description,
	param_model_id, indicator_device_type, indicator_platform, alarm_ne_type,
	enable_filetype11, device_attrs_override, enable_unknown_alarm, is_builtin`

// GetProductByID 实现 Repository。
func (r *PgRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	const q = `SELECT ` + productSelectColumns + ` FROM products WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	p, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get product %s: %w", id, err)
	}
	return p, nil
}

// ListProducts 实现 Repository。
func (r *PgRepository) ListProducts(ctx context.Context) ([]*Product, error) {
	const q = `SELECT ` + productSelectColumns + ` FROM products ORDER BY product_name ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()
	var out []*Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// FetchIndicatorPlatformsByDeviceType 实现 Repository。
//
// 表名按 deviceType 静态路由到 rela_platform_indicator_formula_{enb,gsm,gnb}（设计 §2 KPI 三平台）。
// 平台名落在公式表 platform_name 列；perf_indicators_* 本身不存平台维度。
// 未识别 deviceType 返回空集（与上层 WARN-only 校验语义一致）。
func (r *PgRepository) FetchIndicatorPlatformsByDeviceType(ctx context.Context, deviceType string) (map[string]struct{}, error) {
	table, ok := formulaTableByDeviceType(deviceType)
	if !ok {
		return map[string]struct{}{}, nil
	}
	q := fmt.Sprintf(`SELECT DISTINCT platform_name FROM %s WHERE platform_name IS NOT NULL`, table)
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query %s platforms: %w", table, err)
	}
	defer rows.Close()
	out := make(map[string]struct{}, 8)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan platform: %w", err)
		}
		out[s] = struct{}{}
	}
	return out, rows.Err()
}

// ListIndicatorPlatforms 返回某 indicator_device_type 在 KPI 公式表中已存在的
// platform_name 集合（distinct + 排序）。供产品创建/编辑表单的下拉框使用。
// deviceType 不区分大小写；未识别 deviceType 返回空切片。
func (r *PgRepository) ListIndicatorPlatforms(ctx context.Context, deviceType string) ([]string, error) {
	table, ok := formulaTableByDeviceType(strings.ToLower(strings.TrimSpace(deviceType)))
	if !ok {
		return []string{}, nil
	}
	q := fmt.Sprintf(`SELECT DISTINCT platform_name FROM %s
		WHERE platform_name IS NOT NULL AND platform_name <> ''
		ORDER BY platform_name`, table)
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query %s platforms: %w", table, err)
	}
	defer rows.Close()
	out := make([]string, 0, 8)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan platform: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ListAlarmNeTypes 返回 alarm_definitions 中已存在的 ne_type（distinct + 排序）。
// 供产品创建/编辑表单的下拉框使用。
func (r *PgRepository) ListAlarmNeTypes(ctx context.Context) ([]string, error) {
	const q = `SELECT DISTINCT ne_type FROM alarm_definitions
		WHERE ne_type IS NOT NULL AND ne_type <> ''
		ORDER BY ne_type`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query alarm ne_types: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0, 8)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan ne_type: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchAlarmNeTypes 实现 Repository。
func (r *PgRepository) FetchAlarmNeTypes(ctx context.Context) (map[string]struct{}, error) {
	const q = `SELECT DISTINCT ne_type FROM alarm_definitions`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query alarm ne_types: %w", err)
	}
	defer rows.Close()
	out := make(map[string]struct{}, 8)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan ne_type: %w", err)
		}
		out[s] = struct{}{}
	}
	return out, rows.Err()
}

// indicatorTableByDeviceType 把 product.indicator_device_type 映射到物理表名。
// 与 internal/pm/indicator/loader.go 的命名约定保持一致。
func indicatorTableByDeviceType(deviceType string) (string, bool) {
	switch deviceType {
	case "enb":
		return "perf_indicators_enb", true
	case "gsm":
		return "perf_indicators_gsm", true
	case "gnb":
		return "perf_indicators_gnb", true
	default:
		return "", false
	}
}

// formulaTableByDeviceType 把 product.indicator_device_type 映射到平台公式物理表名。
// platform_name 列只存在于公式表，不在 perf_indicators_*。
func formulaTableByDeviceType(deviceType string) (string, bool) {
	switch deviceType {
	case "enb":
		return "rela_platform_indicator_formula_enb", true
	case "gsm":
		return "rela_platform_indicator_formula_gsm", true
	case "gnb":
		return "rela_platform_indicator_formula_gnb", true
	default:
		return "", false
	}
}

// rowScanner 抽象 pgx Row / Rows 的 Scan 接口，便于 GetProductByID 与 ListProducts 共享 scanner。
type rowScanner interface {
	Scan(dest ...any) error
}

func scanProduct(row rowScanner) (*Product, error) {
	var (
		p             Product
		paramModelID  *uuid.UUID
		overrideJSONB []byte
	)
	if err := row.Scan(
		&p.ID,
		&p.Name,
		&p.Vendor,
		&p.Tech,
		&p.RadioModes,
		&p.Description,
		&paramModelID,
		&p.IndicatorDeviceType,
		&p.IndicatorPlatform,
		&p.AlarmNeType,
		&p.EnableFileType11,
		&overrideJSONB,
		&p.EnableUnknownAlarm,
		&p.IsBuiltin,
	); err != nil {
		return nil, err
	}
	p.ParamModelID = paramModelID
	if len(overrideJSONB) > 0 {
		p.DeviceAttrsOverride = make(map[string]any, 8)
		if err := json.Unmarshal(overrideJSONB, &p.DeviceAttrsOverride); err != nil {
			return nil, fmt.Errorf("unmarshal device_attrs_override for %q: %w", p.Name, err)
		}
	}
	return &p, nil
}
