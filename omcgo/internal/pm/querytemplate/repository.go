package querytemplate

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// Repository 是 pm_query_templates 持久化接口。
type Repository interface {
	Create(ctx context.Context, req CreateRequest) (uuid.UUID, error)
	Get(ctx context.Context, id uuid.UUID) (*Template, error)
	List(ctx context.Context, filter ListFilter) ([]Template, int, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PgRepository 是 Repository 的 pgxpool 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 构造仓库。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

var _ Repository = (*PgRepository)(nil)

var cols = []string{
	"id", "name", "visibility", "creator_id", "description",
	"payload", "created_at", "updated_at",
}

func (r *PgRepository) Create(ctx context.Context, req CreateRequest) (uuid.UUID, error) {
	payload := req.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	q, args, err := storage.Psql.Insert("pm_query_templates").
		Columns("name", "visibility", "creator_id", "description", "payload").
		Values(req.Name, string(req.Visibility), req.CreatorID, req.Description, payload).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build insert: %w", err)
	}

	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return uuid.Nil, ErrDuplicate
		}
		return uuid.Nil, fmt.Errorf("insert: %w", err)
	}
	return id, nil
}

func (r *PgRepository) Get(ctx context.Context, id uuid.UUID) (*Template, error) {
	q, args, err := storage.Psql.Select(cols...).
		From("pm_query_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select: %w", err)
	}
	row := r.pool.QueryRow(ctx, q, args...)
	var t Template
	var visibility string
	if err := row.Scan(&t.ID, &t.Name, &visibility, &t.CreatorID, &t.Description,
		&t.Payload, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	t.Visibility = Visibility(visibility)
	return &t, nil
}

func (r *PgRepository) List(ctx context.Context, filter ListFilter) ([]Template, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	// 可见性过滤：
	//   非 super_admin：visibility='public' OR creator_id=caller
	//   super_admin：全部可见（不加 visibility 过滤，除非显式传入 filter.Visibility）
	base := storage.Psql.Select(cols...).From("pm_query_templates")
	countBase := storage.Psql.Select("COUNT(*)").From("pm_query_templates")

	addAuthAndFilter := func(b sq.SelectBuilder) sq.SelectBuilder {
		if !filter.CallerIsSuperAdmin {
			b = b.Where(sq.Or{
				sq.Eq{"visibility": "public"},
				sq.Eq{"creator_id": filter.CallerID},
			})
		}
		if filter.Visibility != nil {
			b = b.Where(sq.Eq{"visibility": string(*filter.Visibility)})
		}
		if filter.Search != "" {
			b = b.Where(sq.ILike{"name": "%" + filter.Search + "%"})
		}
		return b
	}

	base = addAuthAndFilter(base)
	countBase = addAuthAndFilter(countBase)

	// 总数
	cq, cargs, err := countBase.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count: %w", err)
	}
	var total int
	if err := r.pool.QueryRow(ctx, cq, cargs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}

	// 列表
	q, args, err := base.OrderBy("created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64((page - 1) * pageSize)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build select: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	out := make([]Template, 0)
	for rows.Next() {
		var t Template
		var visibility string
		if err := rows.Scan(&t.ID, &t.Name, &visibility, &t.CreatorID, &t.Description,
			&t.Payload, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan: %w", err)
		}
		t.Visibility = Visibility(visibility)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}
	return out, total, nil
}

func (r *PgRepository) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) error {
	b := storage.Psql.Update("pm_query_templates").Where(sq.Eq{"id": id})
	hasChange := false
	if req.Name != nil {
		b = b.Set("name", *req.Name)
		hasChange = true
	}
	if req.Description != nil {
		b = b.Set("description", *req.Description)
		hasChange = true
	}
	if req.Payload != nil {
		b = b.Set("payload", req.Payload)
		hasChange = true
	}
	if req.Visibility != nil {
		b = b.Set("visibility", string(*req.Visibility))
		hasChange = true
	}
	if !hasChange {
		return nil
	}
	b = b.Set("updated_at", sq.Expr("NOW()"))

	q, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("build update: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicate
		}
		return fmt.Errorf("update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q, args, err := storage.Psql.Delete("pm_query_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
