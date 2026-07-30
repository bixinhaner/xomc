package indicator

import (
	"context"
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var indicatorColumns = []string{
	"i.id", "i.en_name", "i.cn_name", "i.en_description", "i.cn_description",
	"i.group_id", "i.operator_code", "i.data_type", "i.unit_id", "i.updator",
	"i.is_build_in", "i.is_counter", "i.arithmetic", "i.statis_type",
	"i.calculating_status", "i.product_types", "i.indicator_level",
	"i.created_at", "i.updated_at",
}

var _ IndicatorRepository = (*PgIndicatorRepository)(nil)

type PgIndicatorRepository struct {
	db storage.DB
}

func NewPgIndicatorRepository(pool *pgxpool.Pool) *PgIndicatorRepository {
	return &PgIndicatorRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgIndicatorRepository) List(ctx context.Context, filter IndicatorListFilter) (*model.ListResponse[IndicatorListItem], error) {
	dt, err := ParseDeviceType(filter.DeviceType)
	if err != nil {
		return nil, fmt.Errorf("parse device type: %w", err)
	}

	table := dt.IndicatorTable()
	enabledTable := dt.EnabledTable()
	groupTable := dt.GroupTable()

	cols := indicatorColumnsForDevice(dt)
	selectCols := append(cols,
		"CASE WHEN e.indicator_id IS NOT NULL THEN true ELSE false END AS is_enabled",
		"COALESCE(cn.cust_name, '') AS cust_name",
		"COALESCE(g.cn_name, g.en_name, '') AS group_name",
	)

	builder := storage.Psql.Select(selectCols...).
		From(table+" AS i").
		LeftJoin(enabledTable+" e ON e.indicator_id = i.id AND e.operator_code = ?", filter.OperatorCode).
		LeftJoin("perf_cust_name cn ON cn.perf_id = i.id AND cn.operator_code = ?", filter.OperatorCode).
		LeftJoin(groupTable + " g ON g.id = i.group_id")

	builder = applyIndicatorFilters(builder, filter, dt)

	countBuilder := storage.Psql.Select("COUNT(*)").
		From(table+" AS i").
		LeftJoin(enabledTable+" e ON e.indicator_id = i.id AND e.operator_code = ?", filter.OperatorCode)
	countBuilder = applyIndicatorFilters(countBuilder, filter, dt)

	var total int64
	countSQL, countArgs, _ := countBuilder.ToSql()
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count %s: %w", table, err)
	}

	sortBy := sanitizeSortBy(filter.SortBy)
	sortDir := sanitizeSortDir(filter.SortDir)
	builder = builder.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list %s SQL: %w", table, err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", table, err)
	}
	defer rows.Close()

	var items []IndicatorListItem
	for rows.Next() {
		item, err := scanIndicatorListItem(rows, dt)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating indicator rows: %w", err)
	}
	if items == nil {
		items = []IndicatorListItem{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgIndicatorRepository) GetByID(ctx context.Context, dt DeviceType, id string) (*PerfIndicator, error) {
	table := dt.IndicatorTable()
	cols := indicatorColumnsWithoutAliasForDevice(dt)

	query, args, err := storage.Psql.Select(cols...).
		From(table).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get %s SQL: %w", table, err)
	}

	ind, err := scanIndicatorFromRow(r.db.QueryRow(ctx, query, args...), dt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get %s: %w", table, err)
	}
	return ind, nil
}

func (r *PgIndicatorRepository) Create(ctx context.Context, dt DeviceType, indicator *PerfIndicator, tx pgx.Tx) error {
	table := dt.IndicatorTable()
	now := time.Now()
	indicator.CreatedAt = now
	indicator.UpdatedAt = now

	cols := []string{"id", "en_name", "cn_name", "en_description", "cn_description",
		"group_id", "operator_code", "data_type", "unit_id", "updator",
		"is_build_in", "is_counter", "arithmetic", "statis_type", "calculating_status"}
	vals := []interface{}{
		indicator.ID, indicator.EnName, indicator.CnName, indicator.EnDescription, indicator.CnDescription,
		indicator.GroupID, indicator.OperatorCode, indicator.DataType, indicator.UnitID, indicator.Updator,
		indicator.IsBuildIn, indicator.IsCounter, indicator.Arithmetic, indicator.StatisType, indicator.CalculatingStatus,
	}

	if dt.HasProductTypes() {
		cols = append(cols, "product_types", "indicator_level")
		vals = append(vals, indicator.ProductTypes, indicator.IndicatorLevel)
	}

	query, args, err := storage.Psql.Insert(table).Columns(cols...).Values(vals...).ToSql()
	if err != nil {
		return fmt.Errorf("build insert %s SQL: %w", table, err)
	}

	q := querier(r.db, tx)
	_, err = q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert %s: %w", table, err)
	}
	return nil
}

func (r *PgIndicatorRepository) Update(ctx context.Context, dt DeviceType, id string, req *UpdateIndicatorRequest, tx pgx.Tx) error {
	table := dt.IndicatorTable()
	builder := storage.Psql.Update(table)

	if req.EnName != nil {
		builder = builder.Set("en_name", *req.EnName)
	}
	if req.CnName != nil {
		builder = builder.Set("cn_name", *req.CnName)
	}
	if req.EnDescription != nil {
		builder = builder.Set("en_description", *req.EnDescription)
	}
	if req.CnDescription != nil {
		builder = builder.Set("cn_description", *req.CnDescription)
	}
	if req.GroupID != nil {
		builder = builder.Set("group_id", *req.GroupID)
	}
	if req.DataType != nil {
		builder = builder.Set("data_type", *req.DataType)
	}
	if req.UnitID != nil {
		builder = builder.Set("unit_id", *req.UnitID)
	}
	if req.Updator != nil {
		builder = builder.Set("updator", *req.Updator)
	}
	if req.Arithmetic != nil {
		builder = builder.Set("arithmetic", *req.Arithmetic)
	}
	if req.StatisType != nil {
		builder = builder.Set("statis_type", *req.StatisType)
	}
	if req.CalculatingStatus != nil {
		builder = builder.Set("calculating_status", *req.CalculatingStatus)
	}
	if dt.HasProductTypes() {
		if req.ProductTypes != nil {
			builder = builder.Set("product_types", *req.ProductTypes)
		}
		if req.IndicatorLevel != nil {
			builder = builder.Set("indicator_level", *req.IndicatorLevel)
		}
	}

	query, args, err := builder.Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build update %s SQL: %w", table, err)
	}

	if !strings.Contains(query, "SET ") {
		return nil
	}

	q := querier(r.db, tx)
	result, err := q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update %s: %w", table, err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgIndicatorRepository) Delete(ctx context.Context, dt DeviceType, id string, tx pgx.Tx) error {
	table := dt.IndicatorTable()
	query, args, err := storage.Psql.Delete(table).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete %s SQL: %w", table, err)
	}

	q := querier(r.db, tx)
	result, err := q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete %s: %w", table, err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgIndicatorRepository) DeleteByGroupID(ctx context.Context, dt DeviceType, groupID string, tx pgx.Tx) error {
	table := dt.IndicatorTable()
	query, args, err := storage.Psql.Delete(table).
		Where(sq.Eq{"group_id": groupID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete %s by group SQL: %w", table, err)
	}

	q := querier(r.db, tx)
	_, err = q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete %s by group: %w", table, err)
	}
	return nil
}

func (r *PgIndicatorRepository) GetIDsByGroupID(ctx context.Context, dt DeviceType, groupID string) ([]string, error) {
	table := dt.IndicatorTable()
	query, args, err := storage.Psql.Select("id").
		From(table).
		Where(sq.Eq{"group_id": groupID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get IDs by group SQL: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get IDs by group from %s: %w", table, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan indicator ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating indicator ID rows: %w", err)
	}
	return ids, nil
}

func (r *PgIndicatorRepository) GetNextKPIID(ctx context.Context, dt DeviceType, operatorCode string) (string, error) {
	table := dt.IndicatorTable()
	// 2026-06-25:KPI 自定义指标 ID 去掉运营商前缀,统一为 K90000xxxx,与 counter
	// D000xxxx 视觉对齐（用户反馈 #3）。当前实际部署仅 default 一个运营商,旧的
	// defaultK90000xxxx 通过 formula_validator 的 K90000 子串匹配仍能识别,序号
	// 空间共享避免新旧 ID 撞车。
	_ = operatorCode
	const newPrefix = "K90000"

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx for KPI ID generation: %w", err)
	}
	defer tx.Rollback(ctx)

	// Advisory lock to serialize ID generation for this table
	lockKey := int64(crc32.ChecksumIEEE([]byte("kpi_id_" + string(dt))))
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey); err != nil {
		return "", fmt.Errorf("acquire advisory lock for KPI ID: %w", err)
	}

	// 扫所有包含 K90000 的 ID（含旧 defaultK90000xxxx + 新 K90000xxxx 两种格式）,
	// 取末 4 位作为 sequence,共享同一序号空间。
	query, args, err := storage.Psql.Select("id").
		From(table).
		Where(sq.Like{"id": "%K90000%"}).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("build KPI ID scan SQL: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("scan KPI IDs: %w", err)
	}
	defer rows.Close()

	maxSeq := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", fmt.Errorf("scan KPI ID row: %w", err)
		}
		if len(id) < 4 {
			continue
		}
		tail := id[len(id)-4:]
		var n int
		if _, err := fmt.Sscanf(tail, "%d", &n); err == nil && n > maxSeq {
			maxSeq = n
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate KPI IDs: %w", err)
	}

	seq := maxSeq + 1
	if seq > 9999 {
		return "", fmt.Errorf("KPI ID sequence overflow for prefix %s", newPrefix)
	}
	return fmt.Sprintf("%s%04d", newPrefix, seq), tx.Commit(ctx)
}

func (r *PgIndicatorRepository) GetNextCounterID(ctx context.Context) (string, error) {
	prefix := "D00000"

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx for counter ID generation: %w", err)
	}
	defer tx.Rollback(ctx)

	// Advisory lock to serialize counter ID generation
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(0x44303030)); err != nil {
		return "", fmt.Errorf("acquire advisory lock for counter ID: %w", err)
	}

	// Check ENB, GSM, GNB tables for the max counter ID.
	var maxSeq int
	for _, dt := range []DeviceType{DeviceTypeENB, DeviceTypeGSM, DeviceTypeGNB} {
		table := dt.IndicatorTable()
		var maxID *string
		query, args, err := storage.Psql.Select("MAX(id)").
			From(table).
			Where(sq.Like{"id": prefix + "%"}).
			ToSql()
		if err != nil {
			return "", fmt.Errorf("build max counter ID SQL for %s: %w", table, err)
		}

		if err := tx.QueryRow(ctx, query, args...).Scan(&maxID); err != nil {
			return "", fmt.Errorf("query max counter ID from %s: %w", table, err)
		}

		if maxID != nil && len(*maxID) > len(prefix) {
			numStr := (*maxID)[len(prefix):]
			var n int
			if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil && n > maxSeq {
				maxSeq = n
			}
		}
	}

	next := maxSeq + 1
	if next > 9999 {
		return "", fmt.Errorf("counter ID sequence overflow")
	}
	return fmt.Sprintf("%s%04d", prefix, next), tx.Commit(ctx)
}

func (r *PgIndicatorRepository) ListByIDs(ctx context.Context, dt DeviceType, ids []string) ([]*PerfIndicator, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	table := dt.IndicatorTable()
	// PM-P2: ListByIDs 额外取 report_key（路由→白名单据此把上报名翻成编号）。
	// 用专属列集 + 专属 scan，避免触动 List/GetByID 的既有列序。
	cols := indicatorColumnsByIDsForDevice(dt)
	query, args, err := storage.Psql.Select(cols...).
		From(table).
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list by IDs SQL: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list by IDs from %s: %w", table, err)
	}
	defer rows.Close()

	var result []*PerfIndicator
	for rows.Next() {
		ind, err := scanIndicatorByIDs(rows, dt)
		if err != nil {
			return nil, err
		}
		result = append(result, ind)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating indicator rows by IDs: %w", err)
	}
	return result, nil
}

func (r *PgIndicatorRepository) ListAll(ctx context.Context, filter IndicatorListFilter) ([]IndicatorListItem, error) {
	dt, err := ParseDeviceType(filter.DeviceType)
	if err != nil {
		return nil, fmt.Errorf("parse device type: %w", err)
	}

	table := dt.IndicatorTable()
	enabledTable := dt.EnabledTable()
	groupTable := dt.GroupTable()

	cols := indicatorColumnsForDevice(dt)
	selectCols := append(cols,
		"CASE WHEN e.indicator_id IS NOT NULL THEN true ELSE false END AS is_enabled",
		"COALESCE(cn.cust_name, '') AS cust_name",
		"COALESCE(g.cn_name, g.en_name, '') AS group_name",
	)

	builder := storage.Psql.Select(selectCols...).
		From(table+" AS i").
		LeftJoin(enabledTable+" e ON e.indicator_id = i.id AND e.operator_code = ?", filter.OperatorCode).
		LeftJoin("perf_cust_name cn ON cn.perf_id = i.id AND cn.operator_code = ?", filter.OperatorCode).
		LeftJoin(groupTable + " g ON g.id = i.group_id")

	builder = applyIndicatorFilters(builder, filter, dt)
	builder = builder.OrderBy("i.en_name")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list all %s SQL: %w", table, err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all %s: %w", table, err)
	}
	defer rows.Close()

	var items []IndicatorListItem
	for rows.Next() {
		item, err := scanIndicatorListItem(rows, dt)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating indicator rows for export: %w", err)
	}
	return items, nil
}

// applyIndicatorFilters applies common filter conditions to the query builder.
func applyIndicatorFilters(builder sq.SelectBuilder, f IndicatorListFilter, dt DeviceType) sq.SelectBuilder {
	if f.GroupID != nil && *f.GroupID != "" {
		builder = builder.Where(sq.Eq{"i.group_id": *f.GroupID})
	}
	if f.IsBuildIn != nil && *f.IsBuildIn != "" {
		builder = builder.Where(sq.Eq{"i.is_build_in": *f.IsBuildIn})
	}
	if f.IsCounter != nil && *f.IsCounter != "" {
		builder = builder.Where(sq.Eq{"i.is_counter": *f.IsCounter})
	}
	if f.Keyword != nil && *f.Keyword != "" {
		pattern := "%" + *f.Keyword + "%"
		builder = builder.Where(sq.Or{
			sq.ILike{"i.en_name": pattern},
			sq.ILike{"i.id": pattern},
			sq.ILike{"i.cn_name": pattern},
		})
	}
	if f.ProductType != nil && *f.ProductType != "" && dt.HasProductTypes() {
		builder = builder.Where(sq.ILike{"i.product_types": "%" + *f.ProductType + "%"})
	}
	if f.IndicatorLevel != nil && *f.IndicatorLevel != "" && dt.HasProductTypes() {
		level := *f.IndicatorLevel
		if level == "device" || level == "plmn" {
			builder = builder.Where(sq.Eq{"i.indicator_level": []string{level, "both"}})
		} else {
			builder = builder.Where(sq.Eq{"i.indicator_level": level})
		}
	}
	if f.IsEnabled != nil && *f.IsEnabled != "" {
		if *f.IsEnabled == "1" {
			builder = builder.Where("e.indicator_id IS NOT NULL")
		} else if *f.IsEnabled == "0" {
			builder = builder.Where("e.indicator_id IS NULL")
		}
	}
	if f.PlatformName != nil && strings.TrimSpace(*f.PlatformName) != "" {
		platform := strings.TrimSpace(*f.PlatformName)
		builder = builder.Where(sq.Expr(
			fmt.Sprintf("EXISTS (SELECT 1 FROM %s f WHERE f.indicator_id = i.id AND f.platform_name = ANY(?))", dt.FormulaTable()),
			[]string{platform},
		))
	}
	return builder
}

// indicatorColumnsWithoutAlias returns column names without the "i." prefix for single-table queries.
func indicatorColumnsWithoutAlias() []string {
	return indicatorColumnsWithoutAliasForDevice(DeviceTypeENB)
}

func indicatorColumnsWithoutAliasForDevice(dt DeviceType) []string {
	cols := indicatorColumnsForDevice(dt)
	result := make([]string, len(cols))
	for i, c := range cols {
		result[i] = strings.TrimPrefix(c, "i.")
	}
	return result
}

func indicatorColumnsForDevice(dt DeviceType) []string {
	if dt.HasProductTypes() {
		return indicatorColumns
	}
	// GNB: exclude product_types and indicator_level
	return []string{
		"i.id", "i.en_name", "i.cn_name", "i.en_description", "i.cn_description",
		"i.group_id", "i.operator_code", "i.data_type", "i.unit_id", "i.updator",
		"i.is_build_in", "i.is_counter", "i.arithmetic", "i.statis_type",
		"i.calculating_status",
		"i.created_at", "i.updated_at",
	}
}

// indicatorColumnsByIDsForDevice 是 ListByIDs 专属列集：在标准列集尾部追加
// report_key（PM-P2）。追加在末尾，与 scanIndicatorByIDs 一一对位，不影响
// List/GetByID 的列序。
func indicatorColumnsByIDsForDevice(dt DeviceType) []string {
	return append(indicatorColumnsWithoutAliasForDevice(dt), "report_key")
}

// scanIndicatorByIDs 扫描 ListByIDs 行（标准列 + 末尾 report_key）。
func scanIndicatorByIDs(rows pgx.Rows, dt DeviceType) (*PerfIndicator, error) {
	var ind PerfIndicator
	var err error
	if dt.HasProductTypes() {
		err = rows.Scan(
			&ind.ID, &ind.EnName, &ind.CnName, &ind.EnDescription, &ind.CnDescription,
			&ind.GroupID, &ind.OperatorCode, &ind.DataType, &ind.UnitID, &ind.Updator,
			&ind.IsBuildIn, &ind.IsCounter, &ind.Arithmetic, &ind.StatisType,
			&ind.CalculatingStatus, &ind.ProductTypes, &ind.IndicatorLevel,
			&ind.CreatedAt, &ind.UpdatedAt, &ind.ReportKey,
		)
	} else {
		err = rows.Scan(
			&ind.ID, &ind.EnName, &ind.CnName, &ind.EnDescription, &ind.CnDescription,
			&ind.GroupID, &ind.OperatorCode, &ind.DataType, &ind.UnitID, &ind.Updator,
			&ind.IsBuildIn, &ind.IsCounter, &ind.Arithmetic, &ind.StatisType,
			&ind.CalculatingStatus,
			&ind.CreatedAt, &ind.UpdatedAt, &ind.ReportKey,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("scan perf_indicator row (by IDs): %w", err)
	}
	return &ind, nil
}

func scanIndicator(row pgx.Rows, dt DeviceType) (*PerfIndicator, error) {
	var ind PerfIndicator
	var err error
	if dt.HasProductTypes() {
		err = row.Scan(
			&ind.ID, &ind.EnName, &ind.CnName, &ind.EnDescription, &ind.CnDescription,
			&ind.GroupID, &ind.OperatorCode, &ind.DataType, &ind.UnitID, &ind.Updator,
			&ind.IsBuildIn, &ind.IsCounter, &ind.Arithmetic, &ind.StatisType,
			&ind.CalculatingStatus, &ind.ProductTypes, &ind.IndicatorLevel,
			&ind.CreatedAt, &ind.UpdatedAt,
		)
	} else {
		err = row.Scan(
			&ind.ID, &ind.EnName, &ind.CnName, &ind.EnDescription, &ind.CnDescription,
			&ind.GroupID, &ind.OperatorCode, &ind.DataType, &ind.UnitID, &ind.Updator,
			&ind.IsBuildIn, &ind.IsCounter, &ind.Arithmetic, &ind.StatisType,
			&ind.CalculatingStatus,
			&ind.CreatedAt, &ind.UpdatedAt,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("scan perf_indicator row: %w", err)
	}
	return &ind, nil
}

func scanIndicatorFromRow(row pgx.Row, dt DeviceType) (*PerfIndicator, error) {
	var ind PerfIndicator
	var err error
	if dt.HasProductTypes() {
		err = row.Scan(
			&ind.ID, &ind.EnName, &ind.CnName, &ind.EnDescription, &ind.CnDescription,
			&ind.GroupID, &ind.OperatorCode, &ind.DataType, &ind.UnitID, &ind.Updator,
			&ind.IsBuildIn, &ind.IsCounter, &ind.Arithmetic, &ind.StatisType,
			&ind.CalculatingStatus, &ind.ProductTypes, &ind.IndicatorLevel,
			&ind.CreatedAt, &ind.UpdatedAt,
		)
	} else {
		err = row.Scan(
			&ind.ID, &ind.EnName, &ind.CnName, &ind.EnDescription, &ind.CnDescription,
			&ind.GroupID, &ind.OperatorCode, &ind.DataType, &ind.UnitID, &ind.Updator,
			&ind.IsBuildIn, &ind.IsCounter, &ind.Arithmetic, &ind.StatisType,
			&ind.CalculatingStatus,
			&ind.CreatedAt, &ind.UpdatedAt,
		)
	}
	if err != nil {
		return nil, err
	}
	return &ind, nil
}

func scanIndicatorListItem(rows pgx.Rows, dt DeviceType) (*IndicatorListItem, error) {
	var item IndicatorListItem
	var err error

	if dt.HasProductTypes() {
		err = rows.Scan(
			&item.ID, &item.EnName, &item.CnName, &item.EnDescription, &item.CnDescription,
			&item.GroupID, &item.OperatorCode, &item.DataType, &item.UnitID, &item.Updator,
			&item.IsBuildIn, &item.IsCounter, &item.Arithmetic, &item.StatisType,
			&item.CalculatingStatus, &item.ProductTypes, &item.IndicatorLevel,
			&item.CreatedAt, &item.UpdatedAt,
			&item.IsEnabled, &item.CustName, &item.GroupName,
		)
	} else {
		err = rows.Scan(
			&item.ID, &item.EnName, &item.CnName, &item.EnDescription, &item.CnDescription,
			&item.GroupID, &item.OperatorCode, &item.DataType, &item.UnitID, &item.Updator,
			&item.IsBuildIn, &item.IsCounter, &item.Arithmetic, &item.StatisType,
			&item.CalculatingStatus,
			&item.CreatedAt, &item.UpdatedAt,
			&item.IsEnabled, &item.CustName, &item.GroupName,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("scan indicator list item: %w", err)
	}
	return &item, nil
}

var allowedSortColumns = map[string]string{
	"created_at":         "i.created_at",
	"id":                 "i.id",
	"en_name":            "i.en_name",
	"cn_name":            "i.cn_name",
	"group_id":           "i.group_id",
	"data_type":          "i.data_type",
	"is_build_in":        "i.is_build_in",
	"is_counter":         "i.is_counter",
	"calculating_status": "i.calculating_status",
	"product_types":      "i.product_types",
	"indicator_level":    "i.indicator_level",
	"updated_at":         "i.updated_at",
	"updator":            "i.updator",
}

func sanitizeSortBy(input string) string {
	if col, ok := allowedSortColumns[input]; ok {
		return col
	}
	return "i.created_at"
}

func sanitizeSortDir(input string) string {
	if input == "asc" {
		return "asc"
	}
	return "desc"
}
