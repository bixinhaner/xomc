package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgDictionaryRepository implements DictionaryRepository using PostgreSQL.
type PgDictionaryRepository struct {
	pool *pgxpool.Pool
}

var _ DictionaryRepository = (*PgDictionaryRepository)(nil)

// NewPgDictionaryRepository creates a new PgDictionaryRepository.
func NewPgDictionaryRepository(pool *pgxpool.Pool) *PgDictionaryRepository {
	return &PgDictionaryRepository{pool: pool}
}

func (r *PgDictionaryRepository) Create(ctx context.Context, dict *Dictionary) error {
	now := time.Now()

	query, args, err := storage.Psql.Insert("sys_dictionaries").
		Columns(
			"name", "type", "status", "description",
			"source_table", "source_label_field", "source_value_field",
			"created_at", "updated_at",
		).
		Values(
			dict.Name, dict.Type, dict.Status, dict.Description,
			dict.SourceTable, dict.SourceLabelField, dict.SourceValueField,
			now, now,
		).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert dictionary SQL: %w", err)
	}

	err = r.pool.QueryRow(ctx, query, args...).Scan(&dict.ID, &dict.CreatedAt, &dict.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert dictionary: %w", err)
	}
	return nil
}

// dictColumns 是所有读路径的列清单 — 单点维护避免 SELECT/scan 错位。
var dictColumns = []string{
	"id", "name", "type", "status", "description", "created_at", "updated_at",
	"source_table", "source_label_field", "source_value_field",
	"last_refresh_at", "last_refresh_status", "last_refresh_error", "last_refresh_count",
	"name_i18n", "description_i18n", // migration 000003 i18n columns
}

// scanDict 把 dictColumns 对应的列读到 Dictionary。
func scanDict(dst *Dictionary, scan func(...any) error) error {
	var nameI18n, descI18n []byte
	if err := scan(
		&dst.ID, &dst.Name, &dst.Type, &dst.Status, &dst.Description, &dst.CreatedAt, &dst.UpdatedAt,
		&dst.SourceTable, &dst.SourceLabelField, &dst.SourceValueField,
		&dst.LastRefreshAt, &dst.LastRefreshStatus, &dst.LastRefreshError, &dst.LastRefreshCount,
		&nameI18n, &descI18n,
	); err != nil {
		return err
	}
	if len(nameI18n) > 0 {
		_ = json.Unmarshal(nameI18n, &dst.NameI18n)
	}
	if len(descI18n) > 0 {
		_ = json.Unmarshal(descI18n, &dst.DescriptionI18n)
	}
	return nil
}

func (r *PgDictionaryRepository) GetByID(ctx context.Context, id int64) (*Dictionary, error) {
	query, args, err := storage.Psql.Select(dictColumns...).
		From("sys_dictionaries").
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get dictionary SQL: %w", err)
	}

	var dict Dictionary
	row := r.pool.QueryRow(ctx, query, args...)
	if err := scanDict(&dict, row.Scan); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get dictionary: %w", err)
	}
	return &dict, nil
}

func (r *PgDictionaryRepository) GetByType(ctx context.Context, dictType string) (*Dictionary, error) {
	query, args, err := storage.Psql.Select(dictColumns...).
		From("sys_dictionaries").
		Where(sq.And{sq.Eq{"type": dictType}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get dictionary by type SQL: %w", err)
	}

	var dict Dictionary
	row := r.pool.QueryRow(ctx, query, args...)
	if err := scanDict(&dict, row.Scan); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get dictionary by type: %w", err)
	}

	// Load active details sorted by sort_order
	details, err := r.listActiveDetails(ctx, dict.ID)
	if err != nil {
		return nil, err
	}
	dict.Details = details
	return &dict, nil
}

func (r *PgDictionaryRepository) List(ctx context.Context) ([]Dictionary, error) {
	query, args, err := storage.Psql.Select(dictColumns...).
		From("sys_dictionaries").
		Where(sq.Eq{"deleted_at": nil}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list dictionaries SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list dictionaries: %w", err)
	}
	defer rows.Close()

	var dicts []Dictionary
	for rows.Next() {
		var d Dictionary
		if err := scanDict(&d, rows.Scan); err != nil {
			return nil, fmt.Errorf("scan dictionary: %w", err)
		}
		dicts = append(dicts, d)
	}
	return dicts, rows.Err()
}

// ListSourceBound 返所有 source_table IS NOT NULL AND status=true 的字典(T-0182)。
// 由 worker daily cron 通过 SyncEngine.SyncAll 间接调用。
func (r *PgDictionaryRepository) ListSourceBound(ctx context.Context) ([]Dictionary, error) {
	query, args, err := storage.Psql.Select(dictColumns...).
		From("sys_dictionaries").
		Where(sq.And{
			sq.Eq{"deleted_at": nil},
			sq.Eq{"status": true},
			sq.NotEq{"source_table": nil},
		}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list source-bound dicts SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list source-bound dicts: %w", err)
	}
	defer rows.Close()

	var dicts []Dictionary
	for rows.Next() {
		var d Dictionary
		if err := scanDict(&d, rows.Scan); err != nil {
			return nil, fmt.Errorf("scan source-bound dict: %w", err)
		}
		dicts = append(dicts, d)
	}
	return dicts, rows.Err()
}

// UpdateRefreshMetadata 单次同步收尾时写 last_refresh_* 列(T-0182)。
// status: ok / failed / timeout(SyncEngine 决定);count: ok 时为 auto 项总数,
// failed 时为 0。errMsg 已被 SyncEngine 截到 500 字符。
func (r *PgDictionaryRepository) UpdateRefreshMetadata(ctx context.Context, dictID int64, status, errMsg string, count int) error {
	now := time.Now()
	var errMsgArg any
	if errMsg == "" {
		errMsgArg = nil
	} else {
		errMsgArg = errMsg
	}
	query, args, err := storage.Psql.Update("sys_dictionaries").
		Set("last_refresh_at", now).
		Set("last_refresh_status", status).
		Set("last_refresh_error", errMsgArg).
		Set("last_refresh_count", count).
		Set("updated_at", now).
		Where(sq.Eq{"id": dictID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update refresh metadata SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update refresh metadata: %w", err)
	}
	return nil
}

func (r *PgDictionaryRepository) Update(ctx context.Context, dict *Dictionary) error {
	now := time.Now()

	builder := storage.Psql.Update("sys_dictionaries").
		Set("updated_at", now)

	if dict.Name != "" {
		builder = builder.Set("name", dict.Name)
	}
	if dict.Type != "" {
		builder = builder.Set("type", dict.Type)
	}
	builder = builder.Set("status", dict.Status)
	builder = builder.Set("description", dict.Description)
	// T-0182:source_* 直接覆盖入库(nil 也写,让"解绑"语义生效)。
	// 校验三字段一致性已在 service 层完成。
	builder = builder.Set("source_table", dict.SourceTable)
	builder = builder.Set("source_label_field", dict.SourceLabelField)
	builder = builder.Set("source_value_field", dict.SourceValueField)

	query, args, err := builder.
		Where(sq.And{sq.Eq{"id": dict.ID}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update dictionary SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update dictionary: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgDictionaryRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete dictionary tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Soft-delete details first
	_, err = tx.Exec(ctx,
		`UPDATE sys_dictionary_details SET deleted_at = $1, updated_at = $1 WHERE sys_dictionary_id = $2 AND deleted_at IS NULL`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete dictionary details: %w", err)
	}

	// Soft-delete dictionary
	tag, err := tx.Exec(ctx,
		`UPDATE sys_dictionaries SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete dictionary: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return tx.Commit(ctx)
}

// listActiveDetails returns enabled details for a dictionary, sorted by sort_order.
func (r *PgDictionaryRepository) listActiveDetails(ctx context.Context, dictID int64) ([]DictionaryDetail, error) {
	query, args, err := storage.Psql.Select("id", "label", "value", "extend", "status", "sort", "sys_dictionary_id", "parent_id", "level", "origin", "created_at", "updated_at", "label_i18n").
		From("sys_dictionary_details").
		Where(sq.And{sq.Eq{"sys_dictionary_id": dictID}, sq.Eq{"deleted_at": nil}, sq.Eq{"status": true}}).
		OrderBy("sort ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list active details SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active details: %w", err)
	}
	defer rows.Close()

	var details []DictionaryDetail
	for rows.Next() {
		var d DictionaryDetail
		var labelI18n []byte
		if err := rows.Scan(&d.ID, &d.Label, &d.Value, &d.Extend, &d.Status, &d.Sort, &d.SysDictionaryID, &d.ParentID, &d.Level, &d.Origin, &d.CreatedAt, &d.UpdatedAt, &labelI18n); err != nil {
			return nil, fmt.Errorf("scan detail: %w", err)
		}
		if len(labelI18n) > 0 {
			_ = json.Unmarshal(labelI18n, &d.LabelI18n)
		}
		details = append(details, d)
	}
	return details, rows.Err()
}

// --- DictionaryDetailRepository implementation ---

// PgDictionaryDetailRepository implements DictionaryDetailRepository using PostgreSQL.
type PgDictionaryDetailRepository struct {
	pool *pgxpool.Pool
}

var _ DictionaryDetailRepository = (*PgDictionaryDetailRepository)(nil)

// NewPgDictionaryDetailRepository creates a new PgDictionaryDetailRepository.
func NewPgDictionaryDetailRepository(pool *pgxpool.Pool) *PgDictionaryDetailRepository {
	return &PgDictionaryDetailRepository{pool: pool}
}

func (r *PgDictionaryDetailRepository) Create(ctx context.Context, detail *DictionaryDetail) error {
	now := time.Now()
	origin := detail.Origin
	if origin == "" {
		origin = OriginManual
	}

	query, args, err := storage.Psql.Insert("sys_dictionary_details").
		Columns("label", "value", "extend", "status", "sort", "sys_dictionary_id", "parent_id", "level", "origin", "created_at", "updated_at").
		Values(detail.Label, detail.Value, detail.Extend, detail.Status, detail.Sort, detail.SysDictionaryID, detail.ParentID, detail.Level, origin, now, now).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert detail SQL: %w", err)
	}

	err = r.pool.QueryRow(ctx, query, args...).Scan(&detail.ID, &detail.CreatedAt, &detail.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert detail: %w", err)
	}
	return nil
}

func (r *PgDictionaryDetailRepository) GetByID(ctx context.Context, id int64) (*DictionaryDetail, error) {
	query, args, err := storage.Psql.Select("id", "label", "value", "extend", "status", "sort", "sys_dictionary_id", "parent_id", "level", "origin", "created_at", "updated_at", "label_i18n").
		From("sys_dictionary_details").
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get detail SQL: %w", err)
	}

	var d DictionaryDetail
	var labelI18n []byte
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&d.ID, &d.Label, &d.Value, &d.Extend, &d.Status, &d.Sort, &d.SysDictionaryID, &d.ParentID, &d.Level, &d.Origin, &d.CreatedAt, &d.UpdatedAt, &labelI18n,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get detail: %w", err)
	}
	if len(labelI18n) > 0 {
		_ = json.Unmarshal(labelI18n, &d.LabelI18n)
	}
	return &d, nil
}

func (r *PgDictionaryDetailRepository) List(ctx context.Context, req DictionaryDetailListRequest) ([]DictionaryDetail, int64, error) {
	where := sq.And{sq.Eq{"deleted_at": nil}}
	if req.SysDictionaryID != nil {
		where = append(where, sq.Eq{"sys_dictionary_id": *req.SysDictionaryID})
	}
	if req.Label != nil && *req.Label != "" {
		where = append(where, sq.Expr("label ILIKE ?", ilikePattern(*req.Label)))
	}
	if req.Value != nil && *req.Value != "" {
		where = append(where, sq.Expr("value ILIKE ?", ilikePattern(*req.Value)))
	}
	if req.Status != nil {
		where = append(where, sq.Eq{"status": *req.Status})
	}

	// Count
	countQuery, countArgs, err := storage.Psql.Select("COUNT(*)").
		From("sys_dictionary_details").
		Where(where).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count details SQL: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count details: %w", err)
	}

	offset := req.Offset()
	limit := req.Limit()

	query, args, err := storage.Psql.Select("id", "label", "value", "extend", "status", "sort", "sys_dictionary_id", "parent_id", "level", "origin", "created_at", "updated_at", "label_i18n").
		From("sys_dictionary_details").
		Where(where).
		OrderBy("sort ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list details SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list details: %w", err)
	}
	defer rows.Close()

	var items []DictionaryDetail
	for rows.Next() {
		var d DictionaryDetail
		var labelI18n []byte
		if err := rows.Scan(&d.ID, &d.Label, &d.Value, &d.Extend, &d.Status, &d.Sort, &d.SysDictionaryID, &d.ParentID, &d.Level, &d.Origin, &d.CreatedAt, &d.UpdatedAt, &labelI18n); err != nil {
			return nil, 0, fmt.Errorf("scan detail: %w", err)
		}
		if len(labelI18n) > 0 {
			_ = json.Unmarshal(labelI18n, &d.LabelI18n)
		}
		items = append(items, d)
	}
	return items, total, rows.Err()
}

func (r *PgDictionaryDetailRepository) Update(ctx context.Context, detail *DictionaryDetail) error {
	now := time.Now()

	builder := storage.Psql.Update("sys_dictionary_details").
		Set("updated_at", now)

	if detail.Label != "" {
		builder = builder.Set("label", detail.Label)
	}
	if detail.Value != "" {
		builder = builder.Set("value", detail.Value)
	}
	builder = builder.Set("extend", detail.Extend)
	builder = builder.Set("status", detail.Status)
	builder = builder.Set("sort", detail.Sort)
	if detail.SysDictionaryID != 0 {
		builder = builder.Set("sys_dictionary_id", detail.SysDictionaryID)
	}
	// PRD §10：parent_id / level 由 service 层在调用 Update 前显式置入 detail；
	// 包含切顶层（ParentID=nil） + 转移到不同父（ParentID 非 nil）两路。
	builder = builder.Set("parent_id", detail.ParentID)
	builder = builder.Set("level", detail.Level)

	query, args, err := builder.
		Where(sq.And{sq.Eq{"id": detail.ID}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update detail SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update detail: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgDictionaryDetailRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	tag, err := r.pool.Exec(ctx,
		`UPDATE sys_dictionary_details SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("soft-delete detail: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgDictionaryDetailRepository) DeleteByDictionaryID(ctx context.Context, dictID int64) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE sys_dictionary_details SET deleted_at = $1, updated_at = $1 WHERE sys_dictionary_id = $2 AND deleted_at IS NULL`,
		now, dictID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete details by dictionary ID: %w", err)
	}
	return nil
}

// ListSubtreeIDs 用 PG recursive CTE 一次查全 rootID 自身 + 所有后代（深度优先）。
// 仅返回未软删除的行。
func (r *PgDictionaryDetailRepository) ListSubtreeIDs(ctx context.Context, rootID int64) ([]int64, error) {
	const sql = `
WITH RECURSIVE subtree AS (
    SELECT id FROM sys_dictionary_details WHERE id = $1 AND deleted_at IS NULL
    UNION ALL
    SELECT d.id
    FROM sys_dictionary_details d
    INNER JOIN subtree s ON d.parent_id = s.id
    WHERE d.deleted_at IS NULL
)
SELECT id FROM subtree
`
	rows, err := r.pool.Query(ctx, sql, rootID)
	if err != nil {
		return nil, fmt.Errorf("query subtree ids: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan subtree id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ShiftLevelDelta 把 ids 集合中所有行的 level 各加 delta。delta 可为负。
// 用于「换父」级联调整子树深度。
func (r *PgDictionaryDetailRepository) ShiftLevelDelta(ctx context.Context, ids []int64, delta int) error {
	if len(ids) == 0 || delta == 0 {
		return nil
	}
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE sys_dictionary_details SET level = level + $1, updated_at = $2 WHERE id = ANY($3) AND deleted_at IS NULL`,
		delta, now, ids,
	)
	if err != nil {
		return fmt.Errorf("shift level delta: %w", err)
	}
	return nil
}

// === T-0182 数据源同步专用方法 ===

// UpsertAutoBatch 对一批 (label, value) 做 upsert(origin='auto'):
//   - 自然键 (sys_dictionary_id, value, parent_id=NULL, deleted_at IS NULL) 单行
//     —— 跟 uniq_dict_detail_top_value 部分唯一约束对齐(不分 origin)
//   - 命中已有 manual 行(同 value 顶层) → **跳过插入**,manual 优先(PRD §3.2.4
//     方案 A:托管字典里 manual 项保留;同 value 时 manual 占位赢,auto 不挤兑)
//   - 命中已有 auto 行 → label 不同时 UPDATE,否则不动
//   - 未命中 → INSERT 新行 (status=true, sort=0, parent_id=NULL, level=0, origin='auto')
//
// 返回 (inserted, updated)。空 rows 直接返 0,0。
//
// 实现:不走 ON CONFLICT(uniq_dict_detail_top_value 部分唯一约束不在
// (sys_dictionary_id, value, origin) 上,无法借力),而是按 dict+value 分桶
// SELECT(查全部 origin)→ 逐条决策 → 写。N <= 5000 行内可接受。
//
// 历史 bug(2026-05-31 T-0182 上线后真机抓):原实现只查 origin='auto' 现存行,
// manual 同 value 不在 existing 集合 → 直接 INSERT → 撞 uniq_dict_detail_top_value
// 报 23505。修复改为查全部 origin + manual 跳过策略。
func (r *PgDictionaryDetailRepository) UpsertAutoBatch(ctx context.Context, dictID int64, rows []AutoDetailRow) (int, int, error) {
	if len(rows) == 0 {
		return 0, 0, nil
	}

	// 1. 拉当前所有 origin 的顶层未删除行 (value → id, origin, label) 映射,O(n) 比对。
	// 必须查全部 origin(不只 'auto')才能跟 uniq_dict_detail_top_value 唯一约束
	// 对齐 —— 否则 manual 同 value 会被 INSERT 撞约束。
	type existingRow struct {
		ID     int64
		Origin string
		Label  string
	}
	existing := make(map[string]existingRow, len(rows))
	const existsSQL = `
SELECT id, label, value, origin
FROM sys_dictionary_details
WHERE sys_dictionary_id = $1
  AND parent_id IS NULL
  AND deleted_at IS NULL`
	qRows, err := r.pool.Query(ctx, existsSQL, dictID)
	if err != nil {
		return 0, 0, fmt.Errorf("load existing top-level rows: %w", err)
	}
	for qRows.Next() {
		var id int64
		var label, value, origin string
		if err := qRows.Scan(&id, &label, &value, &origin); err != nil {
			qRows.Close()
			return 0, 0, fmt.Errorf("scan existing row: %w", err)
		}
		existing[value] = existingRow{ID: id, Origin: origin, Label: label}
	}
	qRows.Close()
	if err := qRows.Err(); err != nil {
		return 0, 0, fmt.Errorf("iter existing rows: %w", err)
	}

	// 2. 分桶 + 批量执行 INSERT / UPDATE。事务包裹,任意一步失败回滚。
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin upsert tx: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	inserted, updated := 0, 0
	for _, row := range rows {
		cur, ok := existing[row.Value]
		switch {
		case !ok:
			// 无现存行 → 插入新 auto 行
			_, err := tx.Exec(ctx,
				`INSERT INTO sys_dictionary_details
				 (label, value, extend, status, sort, sys_dictionary_id, parent_id, level, origin, created_at, updated_at)
				 VALUES ($1, $2, '', true, 0, $3, NULL, 0, 'auto', $4, $4)`,
				row.Label, row.Value, dictID, now,
			)
			if err != nil {
				return 0, 0, fmt.Errorf("insert auto detail value=%s: %w", row.Value, err)
			}
			inserted++
		case cur.Origin == OriginManual:
			// 已有 manual 行 → 跳过(manual 优先,PRD §3.2.4 方案 A)。
			// 不计入 inserted/updated/deleted。注意:DeleteAutoNotIn 不会
			// 删 manual 行(WHERE origin='auto'),manual 永远在。
		case cur.Label != row.Label:
			// 已有 auto 行且 label 变化 → UPDATE
			_, err := tx.Exec(ctx,
				`UPDATE sys_dictionary_details SET label = $1, updated_at = $2 WHERE id = $3`,
				row.Label, now, cur.ID,
			)
			if err != nil {
				return 0, 0, fmt.Errorf("update auto detail id=%d: %w", cur.ID, err)
			}
			updated++
		}
		// 已有 auto 行 + label 相同 → no-op
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit upsert tx: %w", err)
	}
	return inserted, updated, nil
}

// DeleteAutoNotIn 软删 sys_dictionary_id 下所有 origin='auto' 且 value 不在
// keepValues 集合的行。keepValues 为空 → 软删全部 auto 行(表示用户解绑数据源)。
// 返回被软删的行数。
func (r *PgDictionaryDetailRepository) DeleteAutoNotIn(ctx context.Context, dictID int64, keepValues []string) (int, error) {
	now := time.Now()
	if len(keepValues) == 0 {
		tag, err := r.pool.Exec(ctx,
			`UPDATE sys_dictionary_details
			 SET deleted_at = $1, updated_at = $1
			 WHERE sys_dictionary_id = $2 AND origin = 'auto' AND deleted_at IS NULL`,
			now, dictID,
		)
		if err != nil {
			return 0, fmt.Errorf("delete all auto details: %w", err)
		}
		return int(tag.RowsAffected()), nil
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE sys_dictionary_details
		 SET deleted_at = $1, updated_at = $1
		 WHERE sys_dictionary_id = $2
		   AND origin = 'auto'
		   AND deleted_at IS NULL
		   AND NOT (value = ANY($3))`,
		now, dictID, keepValues,
	)
	if err != nil {
		return 0, fmt.Errorf("delete orphan auto details: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// CountAutoActive 回读当前 auto 行数(未软删)用于写 last_refresh_count。
func (r *PgDictionaryDetailRepository) CountAutoActive(ctx context.Context, dictID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sys_dictionary_details
		 WHERE sys_dictionary_id = $1 AND origin = 'auto' AND deleted_at IS NULL`,
		dictID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count auto details: %w", err)
	}
	return n, nil
}
