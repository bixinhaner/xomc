package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// 本文件为 P3-01 product handler 提供 CRUD/查询/路由测试/孤儿设备 等
// 写入/复杂查询路径，挂在既有 PgRepository 上。

// ── 类型 ────────────────────────────────────────────────────────────

// CreateProductInput / UpdateProductInput 是 handler 用的入参。
type CreateProductInput struct {
	Name                string
	Vendor              string
	Tech                string
	RadioModes          string
	Description         string
	ParamModelID        *uuid.UUID
	IndicatorDeviceType string
	IndicatorPlatform   string
	AlarmNeType         string
	EnableFileType11    bool
	DeviceAttrsOverride map[string]any
	EnableUnknownAlarm  bool
}

// UpdateProductInput nil 字段保留原值。
type UpdateProductInput struct {
	Name                *string
	Vendor              *string
	Tech                *string
	RadioModes          *string
	Description         *string
	ParamModelID        *uuid.UUID
	IndicatorDeviceType *string
	IndicatorPlatform   *string
	AlarmNeType         *string
	EnableFileType11    *bool
	DeviceAttrsOverride map[string]any // 提供则替换；nil 表示不变
	EnableUnknownAlarm  *bool
	clearParamModelID   bool // true 时强制 set NULL
}

// PatternView 是带 sort_order/is_active 的 pattern 视图。
type PatternView struct {
	ID           uuid.UUID
	ProductID    uuid.UUID
	ProductClass string
	SortOrder    int
	IsActive     bool
}

// MatchOrderRow 是 GET /products/match-order 的单行返回。
type MatchOrderRow struct {
	PatternID    uuid.UUID `json:"pattern_id"`
	ProductID    uuid.UUID `json:"product_id"`
	ProductName  string    `json:"product_name"`
	ProductClass string    `json:"product_class"`
	SortOrder    int       `json:"sort_order"`
	IsActive     bool      `json:"is_active"`
}

// OrphanDevice 单条孤儿设备。
type OrphanDevice struct {
	ID            uuid.UUID `json:"id"`
	SerialNumber  string    `json:"serial_number"`
	OUI           string    `json:"oui"`
	ProductClass  string    `json:"product_class"`
	Carrier       string    `json:"carrier"`
	Manufacturer  string    `json:"manufacturer"`
	LastInformAt  *string   `json:"last_inform_at,omitempty"`
}

// ── ParamModel name → ID 反查（外部 Loader 已写入 param_models 表）─────

// LookupParamModelIDByName 反查 param_models.id by name；handler create/update 用。
func (r *PgRepository) LookupParamModelIDByName(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM param_models WHERE name = $1`, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("param_model %q not found", name)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("lookup param_model %q: %w", name, err)
	}
	return id, nil
}

// ── Product CRUD ────────────────────────────────────────────────────

// CreateProduct 新建 products 行；data_attrs_override.data_type=true 拒绝。
func (r *PgRepository) CreateProduct(ctx context.Context, in CreateProductInput) (*Product, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("product_name required")
	}
	if v, ok := in.DeviceAttrsOverride["data_type"]; ok {
		if b, _ := v.(bool); b {
			return nil, fmt.Errorf("device_attrs_override.data_type=true is not allowed")
		}
	}
	overrideJSON, err := json.Marshal(in.DeviceAttrsOverride)
	if err != nil {
		return nil, fmt.Errorf("marshal device_attrs_override: %w", err)
	}

	const insertSQL = `
INSERT INTO products (
    product_name, vendor, tech, radio_modes, description,
    param_model_id, indicator_device_type, indicator_platform, alarm_ne_type,
    enable_filetype11, device_attrs_override, enable_unknown_alarm
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id`
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, insertSQL,
		in.Name, in.Vendor, in.Tech, in.RadioModes, in.Description,
		in.ParamModelID, in.IndicatorDeviceType, in.IndicatorPlatform, in.AlarmNeType,
		in.EnableFileType11, overrideJSON, in.EnableUnknownAlarm,
	).Scan(&id); err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}
	return r.GetProductByID(ctx, id)
}

// UpdateProduct 局部更新 products 行。
func (r *PgRepository) UpdateProduct(ctx context.Context, id uuid.UUID, in UpdateProductInput) (*Product, error) {
	if v, ok := in.DeviceAttrsOverride["data_type"]; ok {
		if b, _ := v.(bool); b {
			return nil, fmt.Errorf("device_attrs_override.data_type=true is not allowed")
		}
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("products").Where(sq.Eq{"id": id})
	dirty := false
	if in.Name != nil {
		ub = ub.Set("product_name", *in.Name)
		dirty = true
	}
	if in.Vendor != nil {
		ub = ub.Set("vendor", *in.Vendor)
		dirty = true
	}
	if in.Tech != nil {
		ub = ub.Set("tech", *in.Tech)
		dirty = true
	}
	if in.RadioModes != nil {
		ub = ub.Set("radio_modes", *in.RadioModes)
		dirty = true
	}
	if in.Description != nil {
		ub = ub.Set("description", *in.Description)
		dirty = true
	}
	if in.ParamModelID != nil {
		ub = ub.Set("param_model_id", *in.ParamModelID)
		dirty = true
	} else if in.clearParamModelID {
		ub = ub.Set("param_model_id", nil)
		dirty = true
	}
	if in.IndicatorDeviceType != nil {
		ub = ub.Set("indicator_device_type", *in.IndicatorDeviceType)
		dirty = true
	}
	if in.IndicatorPlatform != nil {
		ub = ub.Set("indicator_platform", *in.IndicatorPlatform)
		dirty = true
	}
	if in.AlarmNeType != nil {
		ub = ub.Set("alarm_ne_type", *in.AlarmNeType)
		dirty = true
	}
	if in.EnableFileType11 != nil {
		ub = ub.Set("enable_filetype11", *in.EnableFileType11)
		dirty = true
	}
	if in.DeviceAttrsOverride != nil {
		j, err := json.Marshal(in.DeviceAttrsOverride)
		if err != nil {
			return nil, fmt.Errorf("marshal device_attrs_override: %w", err)
		}
		ub = ub.Set("device_attrs_override", j)
		dirty = true
	}
	if in.EnableUnknownAlarm != nil {
		ub = ub.Set("enable_unknown_alarm", *in.EnableUnknownAlarm)
		dirty = true
	}
	if !dirty {
		return r.GetProductByID(ctx, id)
	}
	uSQL, uArgs, err := ub.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update product sql: %w", err)
	}
	tag, err := r.pool.Exec(ctx, uSQL, uArgs...)
	if err != nil {
		return nil, fmt.Errorf("update product %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("product %s not found", id)
	}
	return r.GetProductByID(ctx, id)
}

// DeleteProduct 删除 products 行；被 device.product_id 引用时返回 409 错误。
// product_class_patterns / discovered_param_mappings 通过 FK CASCADE 自动清。
func (r *PgRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	// 检查 device 引用（计数 + 列出前 10 个）
	const cntQ = `SELECT COUNT(*) FROM devices WHERE product_id = $1 AND deleted_at IS NULL`
	var n int
	if err := r.pool.QueryRow(ctx, cntQ, id).Scan(&n); err != nil {
		return fmt.Errorf("count device refs: %w", err)
	}
	if n > 0 {
		return fmt.Errorf("product %s referenced by %d devices; rebind/delete devices first", id, n)
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product %s not found", id)
	}
	return nil
}

// CountDevicesByProduct 返回每个 product_id 的设备数（含 NULL 单独一项 product_id=zero）。
func (r *PgRepository) CountDevicesByProduct(ctx context.Context) (map[uuid.UUID]int, error) {
	const q = `SELECT COALESCE(product_id, '00000000-0000-0000-0000-000000000000'::uuid), COUNT(*)
	          FROM devices WHERE deleted_at IS NULL GROUP BY product_id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("count devices by product: %w", err)
	}
	defer rows.Close()
	out := make(map[uuid.UUID]int, 16)
	for rows.Next() {
		var id uuid.UUID
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("scan device count: %w", err)
		}
		out[id] = n
	}
	return out, rows.Err()
}

// ── Pattern CRUD ────────────────────────────────────────────────────

// CreatePattern 追加正则到全局尾部（sort_order = max+1）。
func (r *PgRepository) CreatePattern(ctx context.Context, productID uuid.UUID, productClass string) (*PatternView, error) {
	productClass = strings.TrimSpace(productClass)
	if productClass == "" {
		return nil, fmt.Errorf("product_class required")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var nextOrder int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM product_class_patterns WHERE is_active`,
	).Scan(&nextOrder); err != nil {
		return nil, fmt.Errorf("compute next sort_order: %w", err)
	}

	var newID uuid.UUID
	if err := tx.QueryRow(ctx,
		`INSERT INTO product_class_patterns (product_id, product_class, sort_order, is_active)
		 VALUES ($1, $2, $3, TRUE) RETURNING id`,
		productID, productClass, nextOrder,
	).Scan(&newID); err != nil {
		return nil, fmt.Errorf("insert pattern: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit insert pattern: %w", err)
	}
	return &PatternView{
		ID: newID, ProductID: productID, ProductClass: productClass,
		SortOrder: nextOrder, IsActive: true,
	}, nil
}

// UpdatePattern 修改 product_class 文本（不改 sort_order/is_active；后者走 Move）。
func (r *PgRepository) UpdatePattern(ctx context.Context, patternID uuid.UUID, productClass string) (*PatternView, error) {
	productClass = strings.TrimSpace(productClass)
	if productClass == "" {
		return nil, fmt.Errorf("product_class required")
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE product_class_patterns SET product_class = $1 WHERE id = $2`,
		productClass, patternID,
	)
	if err != nil {
		return nil, fmt.Errorf("update pattern %s: %w", patternID, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("pattern %s not found", patternID)
	}
	return r.GetPatternByID(ctx, patternID)
}

// DeletePattern 物理删除单条。
func (r *PgRepository) DeletePattern(ctx context.Context, patternID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM product_class_patterns WHERE id = $1`, patternID)
	if err != nil {
		return fmt.Errorf("delete pattern %s: %w", patternID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pattern %s not found", patternID)
	}
	return nil
}

// MovePattern 与相邻活跃 pattern 交换 sort_order。
//
// direction: "up" / "down"。失败：已在端点 / 不存在 / 无相邻活跃项。
//
// 实现：sort_order 上有 partial unique index，直接交换两行会因临时冲突报错。
// 借助"临时大值"绕过约束 — 将目标行先 set 为负值，再回填。
func (r *PgRepository) MovePattern(ctx context.Context, patternID uuid.UUID, direction string) (*PatternView, error) {
	if direction != "up" && direction != "down" {
		return nil, fmt.Errorf("direction must be up|down")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 加锁读当前行
	var curID uuid.UUID
	var curOrder int
	if err := tx.QueryRow(ctx,
		`SELECT id, sort_order FROM product_class_patterns
		 WHERE id = $1 AND is_active FOR UPDATE`,
		patternID,
	).Scan(&curID, &curOrder); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("pattern %s not found or inactive", patternID)
		}
		return nil, fmt.Errorf("lock current pattern: %w", err)
	}

	var neighborQ string
	if direction == "up" {
		neighborQ = `SELECT id, sort_order FROM product_class_patterns
		             WHERE is_active AND sort_order < $1
		             ORDER BY sort_order DESC LIMIT 1 FOR UPDATE`
	} else {
		neighborQ = `SELECT id, sort_order FROM product_class_patterns
		             WHERE is_active AND sort_order > $1
		             ORDER BY sort_order ASC LIMIT 1 FOR UPDATE`
	}
	var neighborID uuid.UUID
	var neighborOrder int
	if err := tx.QueryRow(ctx, neighborQ, curOrder).Scan(&neighborID, &neighborOrder); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("pattern already at the %s end", direction)
		}
		return nil, fmt.Errorf("lock neighbor pattern: %w", err)
	}

	// 走临时负值绕开 unique index
	if _, err := tx.Exec(ctx,
		`UPDATE product_class_patterns SET sort_order = -1 WHERE id = $1`, curID,
	); err != nil {
		return nil, fmt.Errorf("temp set current to -1: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE product_class_patterns SET sort_order = $1 WHERE id = $2`, curOrder, neighborID,
	); err != nil {
		return nil, fmt.Errorf("set neighbor sort_order: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE product_class_patterns SET sort_order = $1 WHERE id = $2`, neighborOrder, curID,
	); err != nil {
		return nil, fmt.Errorf("set current sort_order: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit move: %w", err)
	}
	return r.GetPatternByID(ctx, patternID)
}

// ListPatternsByProduct 按 product_id 列出 patterns。
func (r *PgRepository) ListPatternsByProduct(ctx context.Context, productID uuid.UUID) ([]PatternView, error) {
	const q = `SELECT id, product_id, product_class, sort_order, is_active
	          FROM product_class_patterns WHERE product_id = $1
	          ORDER BY sort_order ASC`
	rows, err := r.pool.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("query patterns by product: %w", err)
	}
	defer rows.Close()
	var out []PatternView
	for rows.Next() {
		var p PatternView
		if err := rows.Scan(&p.ID, &p.ProductID, &p.ProductClass, &p.SortOrder, &p.IsActive); err != nil {
			return nil, fmt.Errorf("scan pattern: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPatternByID 单条详情。
func (r *PgRepository) GetPatternByID(ctx context.Context, patternID uuid.UUID) (*PatternView, error) {
	const q = `SELECT id, product_id, product_class, sort_order, is_active
	          FROM product_class_patterns WHERE id = $1`
	var p PatternView
	if err := r.pool.QueryRow(ctx, q, patternID).Scan(
		&p.ID, &p.ProductID, &p.ProductClass, &p.SortOrder, &p.IsActive,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("pattern %s not found", patternID)
		}
		return nil, fmt.Errorf("get pattern: %w", err)
	}
	return &p, nil
}

// ListMatchOrder 全局 sort_order 升序的 pattern + 所属产品名（设计 §4.4 match-order）。
func (r *PgRepository) ListMatchOrder(ctx context.Context) ([]MatchOrderRow, error) {
	const q = `SELECT pcp.id, pcp.product_id, p.product_name, pcp.product_class,
	                  pcp.sort_order, pcp.is_active
	          FROM product_class_patterns pcp
	          JOIN products p ON p.id = pcp.product_id
	          WHERE pcp.is_active
	          ORDER BY pcp.sort_order ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query match-order: %w", err)
	}
	defer rows.Close()
	var out []MatchOrderRow
	for rows.Next() {
		var m MatchOrderRow
		if err := rows.Scan(&m.PatternID, &m.ProductID, &m.ProductName, &m.ProductClass,
			&m.SortOrder, &m.IsActive); err != nil {
			return nil, fmt.Errorf("scan match-order: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ── Orphan device ───────────────────────────────────────────────────

// ListOrphanDevices 返回 product_id IS NULL 且 product_class 非空的活跃设备。
func (r *PgRepository) ListOrphanDevices(ctx context.Context, limit int) ([]OrphanDevice, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	const q = `SELECT id, serial_number, oui, COALESCE(product_class,''),
	                  carrier, COALESCE(manufacturer,''), last_inform_at
	          FROM devices
	          WHERE product_id IS NULL AND deleted_at IS NULL
	          ORDER BY last_inform_at DESC NULLS LAST
	          LIMIT $1`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("query orphan devices: %w", err)
	}
	defer rows.Close()
	var out []OrphanDevice
	for rows.Next() {
		var d OrphanDevice
		var lastInform *string
		if err := rows.Scan(&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass,
			&d.Carrier, &d.Manufacturer, &lastInform); err != nil {
			return nil, fmt.Errorf("scan orphan device: %w", err)
		}
		d.LastInformAt = lastInform
		out = append(out, d)
	}
	return out, rows.Err()
}

// BindOrphanDevice 手动把孤儿设备绑定到指定 product。
func (r *PgRepository) BindOrphanDevice(ctx context.Context, deviceID, productID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE devices SET product_id = $1 WHERE id = $2 AND deleted_at IS NULL`,
		productID, deviceID,
	)
	if err != nil {
		return fmt.Errorf("bind orphan device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("device %s not found or already deleted", deviceID)
	}
	return nil
}

// 引入 context 占位（避免 import 未使用 — handler 路径会调用本文件函数）。
var _ = context.Background
