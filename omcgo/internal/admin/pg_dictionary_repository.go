package admin

import (
	"context"
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
		Columns("name", "type", "status", "description", "created_at", "updated_at").
		Values(dict.Name, dict.Type, dict.Status, dict.Description, now, now).
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

func (r *PgDictionaryRepository) GetByID(ctx context.Context, id int64) (*Dictionary, error) {
	query, args, err := storage.Psql.Select("id", "name", "type", "status", "description", "created_at", "updated_at").
		From("sys_dictionaries").
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get dictionary SQL: %w", err)
	}

	var dict Dictionary
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&dict.ID, &dict.Name, &dict.Type, &dict.Status, &dict.Description, &dict.CreatedAt, &dict.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get dictionary: %w", err)
	}
	return &dict, nil
}

func (r *PgDictionaryRepository) GetByType(ctx context.Context, dictType string) (*Dictionary, error) {
	query, args, err := storage.Psql.Select("id", "name", "type", "status", "description", "created_at", "updated_at").
		From("sys_dictionaries").
		Where(sq.And{sq.Eq{"type": dictType}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get dictionary by type SQL: %w", err)
	}

	var dict Dictionary
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&dict.ID, &dict.Name, &dict.Type, &dict.Status, &dict.Description, &dict.CreatedAt, &dict.UpdatedAt,
	)
	if err != nil {
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
	query, args, err := storage.Psql.Select("id", "name", "type", "status", "description", "created_at", "updated_at").
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
		if err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Status, &d.Description, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan dictionary: %w", err)
		}
		dicts = append(dicts, d)
	}
	return dicts, rows.Err()
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
	query, args, err := storage.Psql.Select("id", "label", "value", "extend", "status", "sort", "sys_dictionary_id", "created_at", "updated_at").
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
		if err := rows.Scan(&d.ID, &d.Label, &d.Value, &d.Extend, &d.Status, &d.Sort, &d.SysDictionaryID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan detail: %w", err)
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

	query, args, err := storage.Psql.Insert("sys_dictionary_details").
		Columns("label", "value", "extend", "status", "sort", "sys_dictionary_id", "created_at", "updated_at").
		Values(detail.Label, detail.Value, detail.Extend, detail.Status, detail.Sort, detail.SysDictionaryID, now, now).
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
	query, args, err := storage.Psql.Select("id", "label", "value", "extend", "status", "sort", "sys_dictionary_id", "created_at", "updated_at").
		From("sys_dictionary_details").
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get detail SQL: %w", err)
	}

	var d DictionaryDetail
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&d.ID, &d.Label, &d.Value, &d.Extend, &d.Status, &d.Sort, &d.SysDictionaryID, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get detail: %w", err)
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

	query, args, err := storage.Psql.Select("id", "label", "value", "extend", "status", "sort", "sys_dictionary_id", "created_at", "updated_at").
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
		if err := rows.Scan(&d.ID, &d.Label, &d.Value, &d.Extend, &d.Status, &d.Sort, &d.SysDictionaryID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan detail: %w", err)
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
