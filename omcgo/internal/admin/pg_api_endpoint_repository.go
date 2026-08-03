package admin

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgApiEndpointRepository implements ApiEndpointRepository using PostgreSQL.
type PgApiEndpointRepository struct {
	pool *pgxpool.Pool
}

// NewPgApiEndpointRepository creates a new PgApiEndpointRepository.
func NewPgApiEndpointRepository(pool *pgxpool.Pool) *PgApiEndpointRepository {
	return &PgApiEndpointRepository{pool: pool}
}

var _ ApiEndpointRepository = (*PgApiEndpointRepository)(nil)

// Create inserts a new API endpoint into the database.
func (r *PgApiEndpointRepository) Create(ctx context.Context, ep *ApiEndpointDB) error {
	if ep.ID == uuid.Nil {
		ep.ID = uuid.New()
	}
	now := time.Now()
	ep.CreatedAt = now
	ep.UpdatedAt = now

	query, args, err := storage.Psql.Insert("api_endpoints").
		Columns("id", "path", "method", "name", "description", "api_group", "is_auto", "is_user_modified", "created_at", "updated_at").
		Values(ep.ID, ep.Path, ep.Method, ep.Name, ep.Description, ep.ApiGroup, ep.IsAuto, ep.IsUserModified, ep.CreatedAt, ep.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert api_endpoint SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert api_endpoint: %w", err)
	}
	return nil
}

// GetByID retrieves an API endpoint by its ID.
func (r *PgApiEndpointRepository) GetByID(ctx context.Context, id uuid.UUID) (*ApiEndpointDB, error) {
	query, args, err := storage.Psql.
		Select("id", "path", "method", "name", "description", "api_group", "is_auto", "is_user_modified", "created_at", "updated_at").
		From("api_endpoints").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get api_endpoint SQL: %w", err)
	}

	var ep ApiEndpointDB
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&ep.ID, &ep.Path, &ep.Method, &ep.Name, &ep.Description,
		&ep.ApiGroup, &ep.IsAuto, &ep.IsUserModified, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get api_endpoint: %w", err)
	}
	return &ep, nil
}

// Update modifies an existing API endpoint.
func (r *PgApiEndpointRepository) Update(ctx context.Context, id uuid.UUID, req UpdateApiEndpointRequest) (*ApiEndpointDB, error) {
	setter := storage.Psql.Update("api_endpoints").
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id})

	if req.Path != nil {
		setter = setter.Set("path", *req.Path)
	}
	if req.Method != nil {
		setter = setter.Set("method", *req.Method)
	}
	if req.Name != nil {
		setter = setter.Set("name", *req.Name)
	}
	if req.Description != nil {
		setter = setter.Set("description", *req.Description)
	}
	if req.ApiGroup != nil {
		setter = setter.Set("api_group", *req.ApiGroup)
	}
	// 用户改了展示元数据时打标，后续 Sync 扫描不再覆盖这些字段。
	if req.Name != nil || req.Description != nil || req.ApiGroup != nil {
		setter = setter.Set("is_user_modified", true)
	}

	query, args, err := setter.Suffix("RETURNING id, path, method, name, description, api_group, is_auto, is_user_modified, created_at, updated_at").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update api_endpoint SQL: %w", err)
	}

	var ep ApiEndpointDB
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&ep.ID, &ep.Path, &ep.Method, &ep.Name, &ep.Description,
		&ep.ApiGroup, &ep.IsAuto, &ep.IsUserModified, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("update api_endpoint: %w", err)
	}
	return &ep, nil
}

// Delete removes a single API endpoint by ID.
func (r *PgApiEndpointRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("api_endpoints").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete api_endpoint SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete api_endpoint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// DeleteByIDs removes multiple API endpoints by their IDs.
func (r *PgApiEndpointRepository) DeleteByIDs(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := storage.Psql.Delete("api_endpoints").
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build batch delete api_endpoints SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch delete api_endpoints: %w", err)
	}
	return nil
}

// List queries API endpoints with optional filtering and pagination.
func (r *PgApiEndpointRepository) List(ctx context.Context, filter ApiEndpointFilter) (*model.ListResponse[ApiEndpointDB], error) {
	// Build WHERE conditions
	where := sq.And{}
	if filter.Path != "" {
		where = append(where, sq.Expr("path ILIKE ?", ilikePattern(filter.Path)))
	}
	if filter.Method != "" {
		where = append(where, sq.Eq{"method": filter.Method})
	}
	if filter.ApiGroup != "" {
		where = append(where, sq.Eq{"api_group": filter.ApiGroup})
	}
	if filter.Name != "" {
		where = append(where, sq.Expr("name ILIKE ?", ilikePattern(filter.Name)))
	}

	// Count
	countQuery, countArgs, err := storage.Psql.Select("COUNT(*)").
		From("api_endpoints").
		Where(where).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count api_endpoints SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count api_endpoints: %w", err)
	}

	offset := filter.Offset()
	limit := filter.Limit()

	dataQuery, dataArgs, err := storage.Psql.
		Select("id", "path", "method", "name", "description", "api_group", "is_auto", "is_user_modified", "created_at", "updated_at").
		From("api_endpoints").
		Where(where).
		OrderBy("api_group ASC", "path ASC", "method ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list api_endpoints SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("list api_endpoints: %w", err)
	}
	defer rows.Close()

	var items []ApiEndpointDB
	for rows.Next() {
		var ep ApiEndpointDB
		if err := rows.Scan(
			&ep.ID, &ep.Path, &ep.Method, &ep.Name, &ep.Description,
			&ep.ApiGroup, &ep.IsAuto, &ep.IsUserModified, &ep.CreatedAt, &ep.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan api_endpoint: %w", err)
		}
		items = append(items, ep)
	}
	if items == nil {
		items = []ApiEndpointDB{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// Upsert inserts or updates an API endpoint (path+method as unique key).
// 当扫描到已存在的路由时：
//   - is_user_modified=true 的行（name/description/api_group 被用户手工改过）保留原值，不被自动推断值覆盖；
//   - 否则用扫描值刷新 name/description/api_group。
//
// is_auto=false（UI 手工新建）的行同样不被覆盖（WHERE 限定）。
func (r *PgApiEndpointRepository) Upsert(ctx context.Context, path, method, name, description, apiGroup string) (created bool, err error) {
	upsertSQL := `
INSERT INTO api_endpoints (path, method, name, description, api_group, is_auto)
VALUES ($1, $2, $3, $4, $5, TRUE)
ON CONFLICT (path, method)
DO UPDATE SET
    name = CASE WHEN api_endpoints.is_user_modified THEN api_endpoints.name ELSE EXCLUDED.name END,
	description = CASE
		WHEN api_endpoints.is_user_modified OR COALESCE(api_endpoints.description, '') <> '' THEN api_endpoints.description
		ELSE EXCLUDED.description
	END,
    api_group = CASE WHEN api_endpoints.is_user_modified THEN api_endpoints.api_group ELSE EXCLUDED.api_group END,
    updated_at = NOW()
WHERE api_endpoints.is_auto = TRUE
RETURNING xmax = 0`

	if err := r.pool.QueryRow(ctx, upsertSQL, path, method, name, description, apiGroup).Scan(&created); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("upsert api_endpoint: %w", err)
	}
	return created, nil
}

func (r *PgApiEndpointRepository) UpsertBatch(ctx context.Context, inputs []ApiEndpointUpsertInput) (created int, err error) {
	if len(inputs) == 0 {
		return 0, nil
	}

	query := sq.Insert("api_endpoints").
		Columns("path", "method", "name", "description", "api_group", "is_auto")
	for _, input := range inputs {
		query = query.Values(input.Path, input.Method, input.Name, input.Description, input.ApiGroup, true)
	}
	query = query.Suffix(`
ON CONFLICT (path, method)
DO UPDATE SET
    name = CASE WHEN api_endpoints.is_user_modified THEN api_endpoints.name ELSE EXCLUDED.name END,
    description = CASE
        WHEN api_endpoints.is_user_modified OR COALESCE(api_endpoints.description, '') <> '' THEN api_endpoints.description
        ELSE EXCLUDED.description
    END,
    api_group = CASE WHEN api_endpoints.is_user_modified THEN api_endpoints.api_group ELSE EXCLUDED.api_group END,
    updated_at = NOW()
WHERE api_endpoints.is_auto = TRUE
RETURNING xmax = 0`)

	sql, args, err := query.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build batch upsert api endpoints: %w", err)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("batch upsert api endpoints: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var inserted bool
		if err := rows.Scan(&inserted); err != nil {
			return 0, fmt.Errorf("scan batch upsert api endpoints: %w", err)
		}
		if inserted {
			created++
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read batch upsert api endpoints: %w", err)
	}
	return created, nil
}

// GetGroups returns a distinct sorted list of api_group values.
func (r *PgApiEndpointRepository) GetGroups(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT api_group FROM api_endpoints WHERE api_group != '' ORDER BY api_group`)
	if err != nil {
		return nil, fmt.Errorf("get api groups: %w", err)
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, fmt.Errorf("scan api group: %w", err)
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []string{}
	}
	return groups, nil
}
