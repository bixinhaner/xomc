package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgAlarmLibraryRepository struct {
	pool *pgxpool.Pool
}

var _ AlarmLibraryRepository = (*PgAlarmLibraryRepository)(nil)

func NewPgAlarmLibraryRepository(pool *pgxpool.Pool) *PgAlarmLibraryRepository {
	return &PgAlarmLibraryRepository{
		pool: pool,
	}
}

func (r *PgAlarmLibraryRepository) Create(ctx context.Context, lib *AlarmLibrary) error {
	if lib.ID == uuid.Nil {
		lib.ID = uuid.New()
	}
	now := time.Now()
	lib.CreatedAt = now
	lib.UpdatedAt = now

	additionalInfoJSON, err := json.Marshal(lib.AdditionalInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal additional_info: %w", err)
	}

	sql, args, err := storage.Psql.Insert("alarm_libraries").
		Columns("id", "alarm_code", "alarm_source", "event_type", "severity",
			"enabled", "probable_cause", "explanation", "additional_info",
			"carrier", "technology", "created_at", "updated_at").
		Values(lib.ID, lib.AlarmCode, lib.AlarmSource, lib.EventType, lib.Severity,
			lib.Enabled, lib.ProbableCause, lib.Explanation, additionalInfoJSON,
			lib.Carrier, lib.Technology, lib.CreatedAt, lib.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert sql: %w", err)
	}

	_, err = r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to insert alarm library: %w", err)
	}

	return nil
}

func (r *PgAlarmLibraryRepository) GetByID(ctx context.Context, id uuid.UUID) (*AlarmLibrary, error) {
	sql, args, err := storage.Psql.Select(
		"id", "alarm_code", "alarm_source", "event_type", "severity",
		"enabled", "probable_cause", "explanation", "additional_info",
		"carrier", "technology", "created_at", "updated_at").
		From("alarm_libraries").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select sql: %w", err)
	}

	var lib AlarmLibrary
	var additionalInfoJSON []byte

	err = r.pool.QueryRow(ctx, sql, args...).Scan(
		&lib.ID, &lib.AlarmCode, &lib.AlarmSource, &lib.EventType, &lib.Severity,
		&lib.Enabled, &lib.ProbableCause, &lib.Explanation, &additionalInfoJSON,
		&lib.Carrier, &lib.Technology, &lib.CreatedAt, &lib.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query alarm library: %w", err)
	}

	if len(additionalInfoJSON) > 0 {
		if err := json.Unmarshal(additionalInfoJSON, &lib.AdditionalInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal additional_info: %w", err)
		}
	}

	return &lib, nil
}

func (r *PgAlarmLibraryRepository) GetByCode(ctx context.Context, code string) (*AlarmLibrary, error) {
	sql, args, err := storage.Psql.Select(
		"id", "alarm_code", "alarm_source", "event_type", "severity",
		"enabled", "probable_cause", "explanation", "additional_info",
		"carrier", "technology", "created_at", "updated_at").
		From("alarm_libraries").
		Where(squirrel.Eq{"alarm_code": code}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select sql: %w", err)
	}

	var lib AlarmLibrary
	var additionalInfoJSON []byte

	err = r.pool.QueryRow(ctx, sql, args...).Scan(
		&lib.ID, &lib.AlarmCode, &lib.AlarmSource, &lib.EventType, &lib.Severity,
		&lib.Enabled, &lib.ProbableCause, &lib.Explanation, &additionalInfoJSON,
		&lib.Carrier, &lib.Technology, &lib.CreatedAt, &lib.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query alarm library by code: %w", err)
	}

	if len(additionalInfoJSON) > 0 {
		if err := json.Unmarshal(additionalInfoJSON, &lib.AdditionalInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal additional_info: %w", err)
		}
	}

	return &lib, nil
}

func (r *PgAlarmLibraryRepository) Update(ctx context.Context, lib *AlarmLibrary) error {
	lib.UpdatedAt = time.Now()

	additionalInfoJSON, err := json.Marshal(lib.AdditionalInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal additional_info: %w", err)
	}

	query, args, err := storage.Psql.Update("alarm_libraries").
		Set("alarm_code", lib.AlarmCode).
		Set("alarm_source", lib.AlarmSource).
		Set("event_type", lib.EventType).
		Set("severity", lib.Severity).
		Set("enabled", lib.Enabled).
		Set("probable_cause", lib.ProbableCause).
		Set("explanation", lib.Explanation).
		Set("additional_info", additionalInfoJSON).
		Set("carrier", lib.Carrier).
		Set("technology", lib.Technology).
		Set("updated_at", lib.UpdatedAt).
		Where(squirrel.Eq{"id": lib.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update sql: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update alarm library: %w", err)
	}

	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return nil
}

func (r *PgAlarmLibraryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	sql, args, err := storage.Psql.Delete("alarm_libraries").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete sql: %w", err)
	}

	result, err := r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to delete alarm library: %w", err)
	}

	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return nil
}

func (r *PgAlarmLibraryRepository) List(ctx context.Context, filter AlarmLibraryFilter) (*model.ListResponse[AlarmLibrary], error) {
	builder := storage.Psql.Select(
		"id", "alarm_code", "alarm_source", "event_type", "severity",
		"enabled", "probable_cause", "explanation", "additional_info",
		"carrier", "technology", "created_at", "updated_at").
		From("alarm_libraries")
	countBuilder := storage.Psql.Select("COUNT(*)").From("alarm_libraries")

	builder = applyLibraryFilters(builder, filter)
	countBuilder = applyLibraryFilters(countBuilder, filter)

	// Count total
	countSQL, countArgs, _ := countBuilder.ToSql()
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count alarm libraries: %w", err)
	}

	if total == 0 {
		var emptyItems []AlarmLibrary
	return model.NewListResponse(emptyItems, total, filter.Page, filter.PageSize), nil
	}

	// Apply sorting and pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	builder = builder.
		OrderBy(fmt.Sprintf("%s %s", sortBy, sortDir)).
		Limit(uint64(filter.PageSize)).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list alarm_libraries SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list alarm_libraries: %w", err)
	}
	defer rows.Close()

	var libraries []AlarmLibrary
	for rows.Next() {
		lib, err := scanAlarmLibraryRow(rows)
		if err != nil {
			return nil, err
		}
		libraries = append(libraries, *lib)
	}

	if libraries == nil {
		libraries = []AlarmLibrary{}
	}

	return model.NewListResponse(libraries, total, filter.Page, filter.PageSize), nil
}

func (r *PgAlarmLibraryRepository) CreateI18n(ctx context.Context, i18n *AlarmLibraryI18n) error {
	if i18n.ID == uuid.Nil {
		i18n.ID = uuid.New()
	}
	now := time.Now()
	i18n.CreatedAt = now
	i18n.UpdatedAt = now

	sql, args, err := storage.Psql.Insert("alarm_library_i18n").
		Columns("id", "library_id", "locale", "probable_cause", "explanation", "created_at", "updated_at").
		Values(i18n.ID, i18n.LibraryID, i18n.Locale, i18n.ProbableCause, i18n.Explanation, i18n.CreatedAt, i18n.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert i18n sql: %w", err)
	}

	_, err = r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to insert alarm library i18n: %w", err)
	}

	return nil
}

func (r *PgAlarmLibraryRepository) UpdateI18n(ctx context.Context, i18n *AlarmLibraryI18n) error {
	i18n.UpdatedAt = time.Now()

	query, args, err := storage.Psql.Update("alarm_library_i18n").
		Set("locale", i18n.Locale).
		Set("probable_cause", i18n.ProbableCause).
		Set("explanation", i18n.Explanation).
		Set("updated_at", i18n.UpdatedAt).
		Where(squirrel.Eq{"id": i18n.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update i18n sql: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update alarm library i18n: %w", err)
	}

	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return nil
}

func (r *PgAlarmLibraryRepository) DeleteI18n(ctx context.Context, id uuid.UUID) error {
	sql, args, err := storage.Psql.Delete("alarm_library_i18n").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete i18n sql: %w", err)
	}

	result, err := r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to delete alarm library i18n: %w", err)
	}

	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	return nil
}

func (r *PgAlarmLibraryRepository) ListI18n(ctx context.Context, libraryID uuid.UUID) ([]AlarmLibraryI18n, error) {
	sql, args, err := storage.Psql.Select(
		"id", "library_id", "locale", "probable_cause", "explanation", "created_at", "updated_at").
		From("alarm_library_i18n").
		Where(squirrel.Eq{"library_id": libraryID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select i18n sql: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alarm library i18n: %w", err)
	}
	defer rows.Close()

	var i18nList []AlarmLibraryI18n
	for rows.Next() {
		i18n, err := scanAlarmLibraryI18nRow(rows)
		if err != nil {
			return nil, err
		}
		i18nList = append(i18nList, *i18n)
	}

	if i18nList == nil {
		i18nList = []AlarmLibraryI18n{}
	}

	return i18nList, nil
}

func applyLibraryFilters(qb squirrel.SelectBuilder, f AlarmLibraryFilter) squirrel.SelectBuilder {
	if f.AlarmCode != nil && *f.AlarmCode != "" {
		qb = qb.Where(squirrel.Eq{"alarm_code": *f.AlarmCode})
	}
	if f.AlarmSource != nil && *f.AlarmSource != "" {
		qb = qb.Where(squirrel.Like{"alarm_source": "%" + *f.AlarmSource + "%"})
	}
	if f.Severity != nil {
		qb = qb.Where(squirrel.Eq{"severity": *f.Severity})
	}
	if f.Enabled != nil {
		qb = qb.Where(squirrel.Eq{"enabled": *f.Enabled})
	}
	if f.Carrier != nil && *f.Carrier != "" {
		qb = qb.Where(squirrel.Like{"carrier": "%" + *f.Carrier + "%"})
	}
	if f.EventType != nil && *f.EventType != "" {
		qb = qb.Where(squirrel.Like{"event_type": "%" + *f.EventType + "%"})
	}
	if f.Keyword != nil && *f.Keyword != "" {
		kw := "%" + *f.Keyword + "%"
		qb = qb.Where(squirrel.Or{
			squirrel.Like{"alarm_code": kw},
			squirrel.Like{"alarm_source": kw},
			squirrel.Like{"probable_cause": kw},
		})
	}
	return qb
}

func scanAlarmLibraryRow(rows pgx.Rows) (*AlarmLibrary, error) {
	var lib AlarmLibrary
	var additionalInfoJSON []byte
	err := rows.Scan(
		&lib.ID, &lib.AlarmCode, &lib.AlarmSource, &lib.EventType, &lib.Severity,
		&lib.Enabled, &lib.ProbableCause, &lib.Explanation, &additionalInfoJSON,
		&lib.Carrier, &lib.Technology, &lib.CreatedAt, &lib.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan alarm_libraries row: %w", err)
	}

	if len(additionalInfoJSON) > 0 {
		if err := json.Unmarshal(additionalInfoJSON, &lib.AdditionalInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal additional_info: %w", err)
		}
	}

	return &lib, nil
}

func scanAlarmLibraryI18nRow(rows pgx.Rows) (*AlarmLibraryI18n, error) {
	var i18n AlarmLibraryI18n
	err := rows.Scan(
		&i18n.ID, &i18n.LibraryID, &i18n.Locale, &i18n.ProbableCause, &i18n.Explanation, &i18n.CreatedAt, &i18n.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan alarm_library_i18n row: %w", err)
	}

	return &i18n, nil
}