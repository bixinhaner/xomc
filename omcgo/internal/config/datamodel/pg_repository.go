package datamodel

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// psql is the squirrel statement builder configured for PostgreSQL dollar placeholders.
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// dataModelColumns lists all columns for scanning data_model_definitions rows.
var dataModelColumns = []string{
	"id", "carrier", "technology", "version", "oui", "product_class",
	"firmware_version",
	"scope", "status", "is_active", "root_object", "parameter_tree",
	"source", "source_type", "imported_by", "spec_document_ref", "description",
	"last_accessed_at", "created_at", "updated_at",
}

// allowedSortColumns defines columns that can be used for ordering data model queries.
var allowedSortColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"carrier":    true,
	"technology": true,
	"scope":      true,
	"status":     true,
	"version":    true,
}

// --- PgDataModelRepository ---

// PgDataModelRepository implements DataModelRepository using PostgreSQL.
type PgDataModelRepository struct {
	pool *pgxpool.Pool
}

// NewPgDataModelRepository creates a new PgDataModelRepository.
func NewPgDataModelRepository(pool *pgxpool.Pool) *PgDataModelRepository {
	return &PgDataModelRepository{pool: pool}
}

// Create inserts a new data model definition into the database.
func (r *PgDataModelRepository) Create(ctx context.Context, dm *DataModel) error {
	if dm.ID == uuid.Nil {
		dm.ID = uuid.New()
	}
	now := time.Now()
	dm.LastAccessedAt = now
	dm.CreatedAt = now
	dm.UpdatedAt = now

	query, args, err := psql.Insert("data_model_definitions").
		Columns(dataModelColumns...).
		Values(
			dm.ID, dm.Carrier, dm.Technology, dm.Version,
			nullableString(dm.OUI), nullableString(dm.ProductClass),
			nullableString(dm.FirmwareVersion),
			dm.Scope, dm.Status, dm.IsActive, dm.RootObject, dm.ParameterTree,
			nullableString(dm.Source), dm.SourceType, nullableString(dm.ImportedBy),
			nullableString(dm.SpecDocumentRef), nullableString(dm.Description),
			dm.LastAccessedAt, dm.CreatedAt, dm.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert data model SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert data model: %w", err)
	}
	return nil
}

// GetByID retrieves a data model definition by its UUID.
func (r *PgDataModelRepository) GetByID(ctx context.Context, id uuid.UUID) (*DataModel, error) {
	query, args, err := psql.Select(dataModelColumns...).
		From("data_model_definitions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select data model SQL: %w", err)
	}

	dm, err := scanDataModel(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return dm, nil
}

// Update modifies an existing data model definition. Only draft or active models
// may be updated.
func (r *PgDataModelRepository) Update(ctx context.Context, dm *DataModel) error {
	dm.UpdatedAt = time.Now()

	query, args, err := psql.Update("data_model_definitions").
		Set("carrier", dm.Carrier).
		Set("technology", dm.Technology).
		Set("version", dm.Version).
		Set("oui", nullableString(dm.OUI)).
		Set("product_class", nullableString(dm.ProductClass)).
		Set("firmware_version", nullableString(dm.FirmwareVersion)).
		Set("scope", dm.Scope).
		Set("status", dm.Status).
		Set("is_active", dm.IsActive).
		Set("root_object", dm.RootObject).
		Set("parameter_tree", dm.ParameterTree).
		Set("source", nullableString(dm.Source)).
		Set("source_type", dm.SourceType).
		Set("imported_by", nullableString(dm.ImportedBy)).
		Set("spec_document_ref", nullableString(dm.SpecDocumentRef)).
		Set("description", nullableString(dm.Description)).
		Set("updated_at", dm.UpdatedAt).
		Where(sq.And{
			sq.Eq{"id": dm.ID},
			sq.Or{
				sq.Eq{"status": StatusDraft},
				sq.Eq{"status": StatusActive},
			},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update data model SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update data model: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update data model %s: %w (model may be deprecated or not found)", dm.ID, commonerrors.ErrNotFound)
	}
	return nil
}

// Delete removes a data model definition. Only draft models may be deleted.
func (r *PgDataModelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("data_model_definitions").
		Where(sq.And{
			sq.Eq{"id": id},
			sq.Eq{"status": StatusDraft},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete data model SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete data model: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete data model %s: %w (only draft models can be deleted)", id, commonerrors.ErrInvalidInput)
	}
	return nil
}

// List retrieves data model definitions matching the given filter with pagination.
func (r *PgDataModelRepository) List(ctx context.Context, filter DataModelFilter) (*model.ListResponse[DataModel], error) {
	// Build the base WHERE clause.
	pred := sq.And{}
	if filter.Carrier != "" {
		pred = append(pred, sq.Eq{"carrier": filter.Carrier})
	}
	if filter.Technology != "" {
		pred = append(pred, sq.Eq{"technology": filter.Technology})
	}
	if filter.OUI != "" {
		pred = append(pred, sq.Eq{"oui": filter.OUI})
	}
	if filter.ProductClass != "" {
		pred = append(pred, sq.Eq{"product_class": filter.ProductClass})
	}
	if filter.Scope != "" {
		pred = append(pred, sq.Eq{"scope": filter.Scope})
	}
	if filter.Status != "" {
		pred = append(pred, sq.Eq{"status": filter.Status})
	}

	// Count total matching rows.
	countBuilder := psql.Select("COUNT(*)").From("data_model_definitions")
	if len(pred) > 0 {
		countBuilder = countBuilder.Where(pred)
	}
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count data model SQL: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count data models: %w", err)
	}

	// Apply pagination defaults.
	limit := filter.ListRequest.Limit()
	offset := filter.ListRequest.Offset()
	page := filter.ListRequest.Page
	if page < 1 {
		page = 1
	}

	// Build the data query.
	queryBuilder := psql.Select(dataModelColumns...).
		From("data_model_definitions").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	if len(pred) > 0 {
		queryBuilder = queryBuilder.Where(pred)
	}

	// Apply sorting.
	sortCol := "created_at"
	if filter.SortBy != "" && allowedSortColumns[filter.SortBy] {
		sortCol = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	queryBuilder = queryBuilder.OrderBy(sortCol + " " + sortDir)

	querySQL, queryArgs, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list data model SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, querySQL, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list data models: %w", err)
	}
	defer rows.Close()

	items, err := scanDataModels(rows)
	if err != nil {
		return nil, err
	}

	return model.NewListResponse(items, total, page, limit), nil
}

// FindActive locates the currently active data model for a specific classification.
func (r *PgDataModelRepository) FindActive(ctx context.Context, carrier model.CarrierCode, tech model.Technology,
	oui, productClass string, scope model.DataModelScope) (*DataModel, error) {

	builder := psql.Select(dataModelColumns...).
		From("data_model_definitions").
		Where(sq.And{
			sq.Eq{"carrier": carrier},
			sq.Eq{"technology": tech},
			sq.Eq{"scope": scope},
			sq.Eq{"is_active": true},
		})

	// Handle nullable OUI/ProductClass matching.
	if oui == "" {
		builder = builder.Where("oui IS NULL")
	} else {
		builder = builder.Where(sq.Eq{"oui": oui})
	}
	if productClass == "" {
		builder = builder.Where("product_class IS NULL")
	} else {
		builder = builder.Where(sq.Eq{"product_class": productClass})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find active data model SQL: %w", err)
	}

	dm, err := scanDataModel(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return dm, nil
}

// FindActiveWithFirmware finds an active data model matching carrier/tech/oui/productClass/firmwareVersion.
func (r *PgDataModelRepository) FindActiveWithFirmware(ctx context.Context, carrier model.CarrierCode,
	tech model.Technology, oui, productClass, firmwareVersion string, scope model.DataModelScope) (*DataModel, error) {

	builder := psql.Select(dataModelColumns...).
		From("data_model_definitions").
		Where(sq.And{
			sq.Eq{"carrier": carrier},
			sq.Eq{"technology": tech},
			sq.Eq{"scope": scope},
			sq.Eq{"is_active": true},
		})

	if oui == "" {
		builder = builder.Where("oui IS NULL")
	} else {
		builder = builder.Where(sq.Eq{"oui": oui})
	}
	if productClass == "" {
		builder = builder.Where("product_class IS NULL")
	} else {
		builder = builder.Where(sq.Eq{"product_class": productClass})
	}
	if firmwareVersion == "" {
		builder = builder.Where("firmware_version IS NULL")
	} else {
		builder = builder.Where(sq.Eq{"firmware_version": firmwareVersion})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find active with firmware SQL: %w", err)
	}

	dm, err := scanDataModel(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return dm, nil
}

// Activate transitions a data model to active status within a transaction.
// Any previously active model with the same classification is deprecated.
func (r *PgDataModelRepository) Activate(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin activate transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Get the model being activated.
	getSQL, getArgs, err := psql.Select(dataModelColumns...).
		From("data_model_definitions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build select for activate SQL: %w", err)
	}

	dm, err := scanDataModel(tx.QueryRow(ctx, getSQL, getArgs...))
	if err != nil {
		return fmt.Errorf("get data model for activation: %w", err)
	}

	if dm.Status == StatusDeprecated {
		return fmt.Errorf("activate data model %s: %w (cannot activate deprecated model)", id, commonerrors.ErrInvalidInput)
	}

	// 2. Find any currently active model with the same classification and deprecate it.
	deprecateBuilder := psql.Update("data_model_definitions").
		Set("is_active", false).
		Set("status", StatusDeprecated).
		Set("updated_at", time.Now()).
		Where(sq.And{
			sq.Eq{"carrier": dm.Carrier},
			sq.Eq{"technology": dm.Technology},
			sq.Eq{"scope": dm.Scope},
			sq.Eq{"is_active": true},
			sq.NotEq{"id": id},
		})

	if dm.OUI == "" {
		deprecateBuilder = deprecateBuilder.Where("oui IS NULL")
	} else {
		deprecateBuilder = deprecateBuilder.Where(sq.Eq{"oui": dm.OUI})
	}
	if dm.ProductClass == "" {
		deprecateBuilder = deprecateBuilder.Where("product_class IS NULL")
	} else {
		deprecateBuilder = deprecateBuilder.Where(sq.Eq{"product_class": dm.ProductClass})
	}
	if dm.FirmwareVersion == "" {
		deprecateBuilder = deprecateBuilder.Where("firmware_version IS NULL")
	} else {
		deprecateBuilder = deprecateBuilder.Where(sq.Eq{"firmware_version": dm.FirmwareVersion})
	}

	deprecateSQL, deprecateArgs, err := deprecateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("build deprecate old model SQL: %w", err)
	}

	if _, err := tx.Exec(ctx, deprecateSQL, deprecateArgs...); err != nil {
		return fmt.Errorf("deprecate old active model: %w", err)
	}

	// 3. Activate the new model.
	activateSQL, activateArgs, err := psql.Update("data_model_definitions").
		Set("is_active", true).
		Set("status", StatusActive).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build activate model SQL: %w", err)
	}

	if _, err := tx.Exec(ctx, activateSQL, activateArgs...); err != nil {
		return fmt.Errorf("activate data model: %w", err)
	}

	// 4. Commit the transaction.
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit activate transaction: %w", err)
	}
	return nil
}

// Deprecate sets a data model's status to deprecated and marks it inactive.
func (r *PgDataModelRepository) Deprecate(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Update("data_model_definitions").
		Set("status", StatusDeprecated).
		Set("is_active", false).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build deprecate data model SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deprecate data model: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("deprecate data model %s: %w", id, commonerrors.ErrNotFound)
	}
	return nil
}

// Statistics returns aggregate counts of data model definitions.
func (r *PgDataModelRepository) Statistics(ctx context.Context) (*DataModelStats, error) {
	stats := &DataModelStats{
		ByCarrier: make(map[string]int64),
		ByScope:   make(map[string]int64),
	}

	// Total and by-status counts in a single query.
	statusSQL, _, err := psql.Select(
		"COUNT(*) AS total",
		"COUNT(*) FILTER (WHERE status = 'active') AS active",
		"COUNT(*) FILTER (WHERE status = 'draft') AS draft",
		"COUNT(*) FILTER (WHERE status = 'deprecated') AS deprecated",
	).From("data_model_definitions").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build statistics status SQL: %w", err)
	}

	if err := r.pool.QueryRow(ctx, statusSQL).Scan(
		&stats.Total, &stats.Active, &stats.Draft, &stats.Deprecated,
	); err != nil {
		return nil, fmt.Errorf("query statistics status: %w", err)
	}

	// By carrier.
	carrierSQL, _, err := psql.Select("carrier", "COUNT(*)").
		From("data_model_definitions").
		GroupBy("carrier").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build statistics carrier SQL: %w", err)
	}

	carrierRows, err := r.pool.Query(ctx, carrierSQL)
	if err != nil {
		return nil, fmt.Errorf("query statistics by carrier: %w", err)
	}
	defer carrierRows.Close()

	for carrierRows.Next() {
		var carrier string
		var count int64
		if err := carrierRows.Scan(&carrier, &count); err != nil {
			return nil, fmt.Errorf("scan statistics carrier row: %w", err)
		}
		stats.ByCarrier[carrier] = count
	}
	if err := carrierRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate statistics carrier rows: %w", err)
	}

	// By scope.
	scopeSQL, _, err := psql.Select("scope", "COUNT(*)").
		From("data_model_definitions").
		GroupBy("scope").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build statistics scope SQL: %w", err)
	}

	scopeRows, err := r.pool.Query(ctx, scopeSQL)
	if err != nil {
		return nil, fmt.Errorf("query statistics by scope: %w", err)
	}
	defer scopeRows.Close()

	for scopeRows.Next() {
		var scope string
		var count int64
		if err := scopeRows.Scan(&scope, &count); err != nil {
			return nil, fmt.Errorf("scan statistics scope row: %w", err)
		}
		stats.ByScope[scope] = count
	}
	if err := scopeRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate statistics scope rows: %w", err)
	}

	return stats, nil
}

// TouchLastAccessed updates last_accessed_at to the current time for a data model.
func (r *PgDataModelRepository) TouchLastAccessed(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Update("data_model_definitions").
		Set("last_accessed_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build touch last_accessed_at SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("touch last_accessed_at for %s: %w", id, err)
	}
	return nil
}

// DeleteExpired removes data models that have exceeded their idle period.
// Auto-discovered templates expire after autoMaxAge days, manual templates after manualMaxAge days.
func (r *PgDataModelRepository) DeleteExpired(ctx context.Context, autoMaxAge, manualMaxAge int) (int64, error) {
	query := `DELETE FROM data_model_definitions
		WHERE (source_type = 'auto_discovered' AND last_accessed_at < NOW() - make_interval(days => $1))
		   OR (source_type = 'manual' AND last_accessed_at < NOW() - make_interval(days => $2))`

	tag, err := r.pool.Exec(ctx, query, autoMaxAge, manualMaxAge)
	if err != nil {
		return 0, fmt.Errorf("delete expired data models: %w", err)
	}
	return tag.RowsAffected(), nil
}

// FindActiveForMatch performs two-level precise matching for parameter templates.
//
// Level 1: OUI + ProductClass + FirmwareVersion (exact firmware match)
// Level 2: OUI + ProductClass with firmware_version IS NULL or empty (manual > auto_discovered)
//
// Returns nil, nil if no matching template is found.
func (r *PgDataModelRepository) FindActiveForMatch(ctx context.Context,
	carrier model.CarrierCode, tech model.Technology,
	oui, productClass, firmwareVersion string) (*DataModel, error) {

	if oui == "" || productClass == "" {
		return nil, nil
	}

	query := `SELECT ` + joinColumns(dataModelColumns) + ` FROM data_model_definitions
		WHERE is_active = true
		  AND scope = 'product'
		  AND carrier = $1
		  AND technology = $2
		  AND oui = $3
		  AND product_class = $4
		  AND (
		      (firmware_version = $5 AND $5 != '')
		      OR
		      (firmware_version IS NULL OR firmware_version = '')
		  )
		ORDER BY
		  CASE WHEN firmware_version = $5 AND $5 != '' THEN 1 ELSE 2 END,
		  CASE source_type WHEN 'manual' THEN 0 ELSE 1 END
		LIMIT 1`

	dm, err := scanDataModel(r.pool.QueryRow(ctx, query, carrier, tech, oui, productClass, firmwareVersion))
	if err != nil {
		if err == commonerrors.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("find active for match: %w", err)
	}
	return dm, nil
}

// joinColumns joins column names with commas for use in raw SQL queries.
func joinColumns(cols []string) string {
	result := ""
	for i, col := range cols {
		if i > 0 {
			result += ", "
		}
		result += col
	}
	return result
}

// --- PgImportLogRepository ---

// PgImportLogRepository implements ImportLogRepository using PostgreSQL.
type PgImportLogRepository struct {
	pool *pgxpool.Pool
}

// NewPgImportLogRepository creates a new PgImportLogRepository.
func NewPgImportLogRepository(pool *pgxpool.Pool) *PgImportLogRepository {
	return &PgImportLogRepository{pool: pool}
}

// Create inserts a new import log entry.
func (r *PgImportLogRepository) Create(ctx context.Context, entry *ImportLogEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	entry.CreatedAt = time.Now()

	query, args, err := psql.Insert("data_model_import_log").
		Columns("id", "data_model_id", "action", "performed_by", "changes_summary", "created_at").
		Values(entry.ID, entry.DataModelID, entry.Action, entry.PerformedBy, nullableJSON(entry.ChangesSummary), entry.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert import log SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert import log: %w", err)
	}
	return nil
}

// ListByModel retrieves all import log entries for a given data model, ordered by creation time descending.
func (r *PgImportLogRepository) ListByModel(ctx context.Context, modelID uuid.UUID) ([]ImportLogEntry, error) {
	query, args, err := psql.Select("id", "data_model_id", "action", "performed_by", "changes_summary", "created_at").
		From("data_model_import_log").
		Where(sq.Eq{"data_model_id": modelID}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list import log SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list import logs: %w", err)
	}
	defer rows.Close()

	var entries []ImportLogEntry
	for rows.Next() {
		var entry ImportLogEntry
		var changesSummary []byte
		if err := rows.Scan(
			&entry.ID, &entry.DataModelID, &entry.Action, &entry.PerformedBy,
			&changesSummary, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan import log row: %w", err)
		}
		if changesSummary != nil {
			entry.ChangesSummary = json.RawMessage(changesSummary)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import log rows: %w", err)
	}

	return entries, nil
}

// --- PgOUIRepository ---

// PgOUIRepository implements OUIRepository using PostgreSQL.
type PgOUIRepository struct {
	pool *pgxpool.Pool
}

// NewPgOUIRepository creates a new PgOUIRepository.
func NewPgOUIRepository(pool *pgxpool.Pool) *PgOUIRepository {
	return &PgOUIRepository{pool: pool}
}

// GetByOUI retrieves an OUI entry by its code.
func (r *PgOUIRepository) GetByOUI(ctx context.Context, oui string) (*OUIEntry, error) {
	query, args, err := psql.Select("oui", "manufacturer", "short_name", "country", "created_at").
		From("oui_registry").
		Where(sq.Eq{"oui": oui}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select OUI SQL: %w", err)
	}

	var entry OUIEntry
	var country sql.NullString
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&entry.OUI, &entry.Manufacturer, &entry.ShortName, &country, &entry.CreatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("OUI %s: %w", oui, commonerrors.ErrNotFound)
		}
		return nil, fmt.Errorf("get OUI %s: %w", oui, err)
	}
	if country.Valid {
		entry.Country = country.String
	}

	return &entry, nil
}

// List retrieves all OUI entries ordered by manufacturer short name.
func (r *PgOUIRepository) List(ctx context.Context) ([]OUIEntry, error) {
	query, args, err := psql.Select("oui", "manufacturer", "short_name", "country", "created_at").
		From("oui_registry").
		OrderBy("short_name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list OUI SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list OUI entries: %w", err)
	}
	defer rows.Close()

	var entries []OUIEntry
	for rows.Next() {
		var entry OUIEntry
		var country sql.NullString
		if err := rows.Scan(
			&entry.OUI, &entry.Manufacturer, &entry.ShortName, &country, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan OUI row: %w", err)
		}
		if country.Valid {
			entry.Country = country.String
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate OUI rows: %w", err)
	}

	return entries, nil
}

// Create inserts a new OUI entry.
func (r *PgOUIRepository) Create(ctx context.Context, entry *OUIEntry) error {
	entry.CreatedAt = time.Now()

	query, args, err := psql.Insert("oui_registry").
		Columns("oui", "manufacturer", "short_name", "country", "created_at").
		Values(entry.OUI, entry.Manufacturer, entry.ShortName, nullableString(entry.Country), entry.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert OUI SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert OUI entry: %w", err)
	}
	return nil
}

// --- Helper functions ---

// nullableString converts an empty string to nil for SQL NULL handling.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nullableJSON converts a nil or empty json.RawMessage to nil for SQL NULL handling.
func nullableJSON(data json.RawMessage) interface{} {
	if len(data) == 0 {
		return nil
	}
	return []byte(data)
}

// scanDataModel scans a single data_model_definitions row into a DataModel struct.
func scanDataModel(row pgx.Row) (*DataModel, error) {
	var dm DataModel
	var (
		oui             sql.NullString
		productClass    sql.NullString
		firmwareVersion sql.NullString
		source          sql.NullString
		importedBy      sql.NullString
		specDocumentRef sql.NullString
		description     sql.NullString
		parameterTree   []byte
	)

	err := row.Scan(
		&dm.ID, &dm.Carrier, &dm.Technology, &dm.Version,
		&oui, &productClass, &firmwareVersion,
		&dm.Scope, &dm.Status, &dm.IsActive, &dm.RootObject, &parameterTree,
		&source, &dm.SourceType, &importedBy, &specDocumentRef, &description,
		&dm.LastAccessedAt, &dm.CreatedAt, &dm.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan data model row: %w", err)
	}

	dm.ParameterTree = json.RawMessage(parameterTree)
	if oui.Valid {
		dm.OUI = oui.String
	}
	if productClass.Valid {
		dm.ProductClass = productClass.String
	}
	if firmwareVersion.Valid {
		dm.FirmwareVersion = firmwareVersion.String
	}
	if source.Valid {
		dm.Source = source.String
	}
	if importedBy.Valid {
		dm.ImportedBy = importedBy.String
	}
	if specDocumentRef.Valid {
		dm.SpecDocumentRef = specDocumentRef.String
	}
	if description.Valid {
		dm.Description = description.String
	}

	return &dm, nil
}

// scanDataModels scans multiple data_model_definitions rows into a DataModel slice.
func scanDataModels(rows pgx.Rows) ([]DataModel, error) {
	var items []DataModel
	for rows.Next() {
		var dm DataModel
		var (
			oui             sql.NullString
			productClass    sql.NullString
			firmwareVersion sql.NullString
			source          sql.NullString
			importedBy      sql.NullString
			specDocumentRef sql.NullString
			description     sql.NullString
			parameterTree   []byte
		)

		err := rows.Scan(
			&dm.ID, &dm.Carrier, &dm.Technology, &dm.Version,
			&oui, &productClass, &firmwareVersion,
			&dm.Scope, &dm.Status, &dm.IsActive, &dm.RootObject, &parameterTree,
			&source, &dm.SourceType, &importedBy, &specDocumentRef, &description,
			&dm.LastAccessedAt, &dm.CreatedAt, &dm.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan data model row: %w", err)
		}

		dm.ParameterTree = json.RawMessage(parameterTree)
		if oui.Valid {
			dm.OUI = oui.String
		}
		if productClass.Valid {
			dm.ProductClass = productClass.String
		}
		if firmwareVersion.Valid {
			dm.FirmwareVersion = firmwareVersion.String
		}
		if source.Valid {
			dm.Source = source.String
		}
		if importedBy.Valid {
			dm.ImportedBy = importedBy.String
		}
		if specDocumentRef.Valid {
			dm.SpecDocumentRef = specDocumentRef.String
		}
		if description.Valid {
			dm.Description = description.String
		}

		items = append(items, dm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate data model rows: %w", err)
	}
	return items, nil
}

// Compile-time interface compliance checks.
var (
	_ DataModelRepository = (*PgDataModelRepository)(nil)
	_ ImportLogRepository = (*PgImportLogRepository)(nil)
	_ OUIRepository       = (*PgOUIRepository)(nil)
)
