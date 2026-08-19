package device

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// paramColumns defines the standard column set for device_parameters queries.
var paramColumns = []string{
	"device_id", "parameter_path", "parameter_value",
	"parameter_type", "writable", "last_updated_at",
	"fap_instance", "param_group",
}

// scanParam scans a row into a DeviceParameter struct.
func scanParam(s interface{ Scan(dest ...any) error }) (model.DeviceParameter, error) {
	var p model.DeviceParameter
	err := s.Scan(
		&p.DeviceID, &p.ParameterPath, &p.ParameterValue,
		&p.ParameterType, &p.Writable, &p.LastUpdatedAt,
		&p.FAPInstance, &p.ParamGroup,
	)
	return p, err
}

// PgDeviceParameterRepository implements DeviceParameterRepository using PostgreSQL.
type PgDeviceParameterRepository struct {
	pool *pgxpool.Pool
}

// NewPgDeviceParameterRepository creates a new PostgreSQL device parameter repository.
func NewPgDeviceParameterRepository(pool *pgxpool.Pool) *PgDeviceParameterRepository {
	return &PgDeviceParameterRepository{pool: pool}
}

func (r *PgDeviceParameterRepository) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
	if len(params) == 0 {
		return nil
	}
	rows := make([]deviceParameterUpsertRow, 0, len(params))
	for _, p := range params {
		rows = append(rows, deviceParameterUpsertRow{deviceID: deviceID, parameter: p})
	}
	if _, err := bulkUpsertDeviceParameters(ctx, r.pool, rows); err != nil {
		return fmt.Errorf("batch upsert device parameters: %w", err)
	}
	return nil
}

func (r *PgDeviceParameterRepository) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		OrderBy("parameter_path ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query device parameters: %w", err)
	}
	defer rows.Close()

	var params []model.DeviceParameter
	for rows.Next() {
		p, err := scanParam(rows)
		if err != nil {
			return nil, fmt.Errorf("scan parameter: %w", err)
		}
		params = append(params, p)
	}

	if params == nil {
		params = []model.DeviceParameter{}
	}
	return params, nil
}

func (r *PgDeviceParameterRepository) GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID, "parameter_path": path}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	p, err := scanParam(r.pool.QueryRow(ctx, query, args...))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query parameter: %w", err)
	}
	return &p, nil
}

func (r *PgDeviceParameterRepository) DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error {
	query, args, _ := storage.Psql.Delete("device_parameters").Where(sq.Eq{"device_id": deviceID}).ToSql()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete device parameters: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := AcquireParameterWriteLocks(ctx, tx, deviceID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete device parameters: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete device parameters: %w", err)
	}
	return nil
}

func (r *PgDeviceParameterRepository) DeleteByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int64, error) {
	if prefix == "" {
		return 0, fmt.Errorf("empty prefix not allowed (would delete all device parameters)")
	}
	query, args, _ := storage.Psql.Delete("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.Like{"parameter_path": prefix + "%"}).
		ToSql()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin delete device parameters by prefix: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := AcquireParameterWriteLocks(ctx, tx, deviceID); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete device parameters by prefix: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit delete device parameters by prefix: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *PgDeviceParameterRepository) GetByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error) {
	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.Like{"parameter_path": prefix + "%"}).
		OrderBy("parameter_path ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build prefix query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query by prefix: %w", err)
	}
	defer rows.Close()

	var params []model.DeviceParameter
	for rows.Next() {
		p, err := scanParam(rows)
		if err != nil {
			return nil, fmt.Errorf("scan parameter: %w", err)
		}
		params = append(params, p)
	}
	if params == nil {
		params = []model.DeviceParameter{}
	}
	return params, nil
}

func (r *PgDeviceParameterRepository) CountByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int, error) {
	query, args, err := storage.Psql.Select("COUNT(*)").
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.Like{"parameter_path": prefix + "%"}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count by prefix: %w", err)
	}
	return count, nil
}

// MaxInstanceNumberByPrefix 查询 device_parameters 中以 prefix 开头的 path,
// 提取 prefix 之后的第一段(如果是数字)取最大值。供 Path B instance 展开估算用。
//
// 例 prefix="DeviceGSM.Bts.":
//
//	DB path = "DeviceGSM.Bts.254.CellId"  → 提取出 "254"
//	DB path = "DeviceGSM.Bts.256.Trx.1.Rf" → 提取出 "256"
//	返回 256
//
// 查询返回去重后的"第一段"集合(BSC 场景 ≤ 256 行),Go 侧扫描取 max,避免拉全部
// 17791 行 path。无匹配返回 (0, nil)。仅 expand 路径调用,不在主接口。
func (r *PgDeviceParameterRepository) MaxInstanceNumberByPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int, error) {
	if prefix == "" {
		return 0, nil
	}
	// substring(path FROM length(prefix)+1) 去掉前缀,
	// regexp_replace 去掉第一个 "." 及之后,得到 "第一段"。
	// DISTINCT 后行数 = 实例数量(BTS 256 → 256 行)。
	const q = `SELECT DISTINCT regexp_replace(
		substring(parameter_path FROM length($2) + 1),
		'\..*$', ''
	) AS inst
	FROM device_parameters
	WHERE device_id = $1
	  AND parameter_path LIKE $2 || '%'`
	rows, err := r.pool.Query(ctx, q, deviceID, prefix)
	if err != nil {
		return 0, fmt.Errorf("query max instance number: %w", err)
	}
	defer rows.Close()
	maxInst := 0
	for rows.Next() {
		var inst string
		if err := rows.Scan(&inst); err != nil {
			return 0, fmt.Errorf("scan inst: %w", err)
		}
		if inst == "" {
			continue
		}
		n := 0
		for _, c := range inst {
			if c < '0' || c > '9' {
				n = 0
				break
			}
			n = n*10 + int(c-'0')
		}
		if n > maxInst {
			maxInst = n
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate max instance numbers: %w", err)
	}
	return maxInst, nil
}

func (r *PgDeviceParameterRepository) SearchByKeyword(ctx context.Context, deviceID uuid.UUID, keyword string, limit int) ([]model.DeviceParameter, error) {
	if limit <= 0 {
		limit = 100
	}
	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.ILike{"parameter_path": "%" + keyword + "%"}).
		OrderBy("parameter_path ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build search query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search parameters: %w", err)
	}
	defer rows.Close()

	var params []model.DeviceParameter
	for rows.Next() {
		p, err := scanParam(rows)
		if err != nil {
			return nil, fmt.Errorf("scan parameter: %w", err)
		}
		params = append(params, p)
	}
	if params == nil {
		params = []model.DeviceParameter{}
	}
	return params, nil
}

func (r *PgDeviceParameterRepository) GetDirectChildLeaves(ctx context.Context, deviceID uuid.UUID, prefix string, limit, offset int) ([]model.DeviceParameter, int, error) {
	if limit <= 0 {
		limit = 50
	}

	// 直接叶子参数：匹配前缀，但去掉前缀后不再包含 "."
	// 即 parameter_path LIKE 'prefix%' AND parameter_path NOT LIKE 'prefix%.%'
	likePrefix := prefix + "%"
	notLikeDeeper := prefix + "%.%"

	// 先查总数
	countQuery, countArgs, err := storage.Psql.Select("COUNT(*)").
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.Like{"parameter_path": likePrefix}).
		Where(sq.NotLike{"parameter_path": notLikeDeeper}).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count direct child leaves: %w", err)
	}

	// 查分页数据
	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID}).
		Where(sq.Like{"parameter_path": likePrefix}).
		Where(sq.NotLike{"parameter_path": notLikeDeeper}).
		OrderBy("parameter_path ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build direct children query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query direct child leaves: %w", err)
	}
	defer rows.Close()

	var params []model.DeviceParameter
	for rows.Next() {
		p, err := scanParam(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan parameter: %w", err)
		}
		params = append(params, p)
	}
	if params == nil {
		params = []model.DeviceParameter{}
	}
	return params, total, nil
}

func (r *PgDeviceParameterRepository) GetByGroup(ctx context.Context, deviceID uuid.UUID, group string) ([]model.DeviceParameter, error) {
	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID, "param_group": group}).
		OrderBy("parameter_path ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build group query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query by group: %w", err)
	}
	defer rows.Close()

	var params []model.DeviceParameter
	for rows.Next() {
		p, err := scanParam(rows)
		if err != nil {
			return nil, fmt.Errorf("scan parameter: %w", err)
		}
		params = append(params, p)
	}
	if params == nil {
		params = []model.DeviceParameter{}
	}
	return params, nil
}

// GetByDeviceIDsAndGroup loads one functional parameter group for a page of
// devices in a single query. Device-list decorations use this to avoid N+1 reads.
func (r *PgDeviceParameterRepository) GetByDeviceIDsAndGroup(
	ctx context.Context,
	deviceIDs []uuid.UUID,
	group string,
) (map[uuid.UUID][]model.DeviceParameter, error) {
	result := make(map[uuid.UUID][]model.DeviceParameter, len(deviceIDs))
	if len(deviceIDs) == 0 {
		return result, nil
	}

	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceIDs, "param_group": group}).
		OrderBy("device_id ASC", "parameter_path ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build batch group query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query batch group: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		param, scanErr := scanParam(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan batch group parameter: %w", scanErr)
		}
		result[param.DeviceID] = append(result[param.DeviceID], param)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate batch group parameters: %w", err)
	}
	return result, nil
}

func (r *PgDeviceParameterRepository) GetByFAPInstance(ctx context.Context, deviceID uuid.UUID, instance int) ([]model.DeviceParameter, error) {
	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID, "fap_instance": instance}).
		OrderBy("parameter_path ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build fap instance query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query by fap instance: %w", err)
	}
	defer rows.Close()

	var params []model.DeviceParameter
	for rows.Next() {
		p, err := scanParam(rows)
		if err != nil {
			return nil, fmt.Errorf("scan parameter: %w", err)
		}
		params = append(params, p)
	}
	if params == nil {
		params = []model.DeviceParameter{}
	}
	return params, nil
}

func (r *PgDeviceParameterRepository) GetByFAPInstanceAndGroup(ctx context.Context, deviceID uuid.UUID, instance int, group string) ([]model.DeviceParameter, error) {
	query, args, err := storage.Psql.Select(paramColumns...).
		From("device_parameters").
		Where(sq.Eq{"device_id": deviceID, "fap_instance": instance, "param_group": group}).
		OrderBy("parameter_path ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build fap instance+group query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query by fap instance and group: %w", err)
	}
	defer rows.Close()

	var params []model.DeviceParameter
	for rows.Next() {
		p, err := scanParam(rows)
		if err != nil {
			return nil, fmt.Errorf("scan parameter: %w", err)
		}
		params = append(params, p)
	}
	if params == nil {
		params = []model.DeviceParameter{}
	}
	return params, nil
}
