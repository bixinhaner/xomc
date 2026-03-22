package report

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

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// ---- column lists ----

var definitionColumns = []string{
	"id", "report_name", "report_type", "description", "format", "period",
	"kpi_codes", "device_groups", "auto_generate", "cron_expression",
	"status", "creator", "last_gen_time", "created_at", "updated_at",
}

var recordColumns = []string{
	"id", "report_definition_id", "report_name", "period", "generate_time",
	"file_size", "download_url", "format", "status", "minio_path", "created_at",
}

// ======================================================================
// PgDefinitionRepository
// ======================================================================

var _ DefinitionRepository = (*PgDefinitionRepository)(nil)

// PgDefinitionRepository is a PostgreSQL implementation of DefinitionRepository.
type PgDefinitionRepository struct {
	pool *pgxpool.Pool
}

// NewPgDefinitionRepository creates a new PgDefinitionRepository.
func NewPgDefinitionRepository(pool *pgxpool.Pool) *PgDefinitionRepository {
	return &PgDefinitionRepository{pool: pool}
}

func (r *PgDefinitionRepository) Create(ctx context.Context, def *ReportDefinition) error {
	if def.Format == nil {
		def.Format = []string{"pdf"}
	}
	if def.KPICodes == nil {
		def.KPICodes = []string{}
	}
	if def.DeviceGroups == nil {
		def.DeviceGroups = []string{}
	}

	formatJSON, err := json.Marshal(def.Format)
	if err != nil {
		return fmt.Errorf("marshal format: %w", err)
	}
	kpiJSON, err := json.Marshal(def.KPICodes)
	if err != nil {
		return fmt.Errorf("marshal kpi_codes: %w", err)
	}
	groupsJSON, err := json.Marshal(def.DeviceGroups)
	if err != nil {
		return fmt.Errorf("marshal device_groups: %w", err)
	}

	query, args, err := psql.Insert("report_definitions").
		Columns("report_name", "report_type", "description", "format", "period",
			"kpi_codes", "device_groups", "auto_generate", "cron_expression",
			"status", "creator").
		Values(def.ReportName, def.ReportType, nullableString(def.Description),
			formatJSON, def.Period, kpiJSON, groupsJSON,
			def.AutoGenerate, nullableString(def.CronExpression),
			def.Status, nullableString(def.Creator)).
		Suffix("RETURNING " + joinColumns(definitionColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert report_definition SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanDefinition(row)
	if err != nil {
		return fmt.Errorf("create report_definition: %w", err)
	}
	*def = *created
	return nil
}

func (r *PgDefinitionRepository) GetByID(ctx context.Context, id uuid.UUID) (*ReportDefinition, error) {
	query, args, err := psql.Select(definitionColumns...).
		From("report_definitions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get report_definition SQL: %w", err)
	}

	def, err := scanDefinition(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get report_definition: %w", err)
	}
	return def, nil
}

func (r *PgDefinitionRepository) Update(ctx context.Context, def *ReportDefinition) error {
	def.UpdatedAt = time.Now()

	formatJSON, err := json.Marshal(def.Format)
	if err != nil {
		return fmt.Errorf("marshal format: %w", err)
	}
	kpiJSON, err := json.Marshal(def.KPICodes)
	if err != nil {
		return fmt.Errorf("marshal kpi_codes: %w", err)
	}
	groupsJSON, err := json.Marshal(def.DeviceGroups)
	if err != nil {
		return fmt.Errorf("marshal device_groups: %w", err)
	}

	query, args, err := psql.Update("report_definitions").
		Set("report_name", def.ReportName).
		Set("report_type", def.ReportType).
		Set("description", nullableString(def.Description)).
		Set("format", formatJSON).
		Set("period", def.Period).
		Set("kpi_codes", kpiJSON).
		Set("device_groups", groupsJSON).
		Set("auto_generate", def.AutoGenerate).
		Set("cron_expression", nullableString(def.CronExpression)).
		Set("status", def.Status).
		Set("creator", nullableString(def.Creator)).
		Set("last_gen_time", def.LastGenTime).
		Where(sq.Eq{"id": def.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update report_definition SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update report_definition: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgDefinitionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("report_definitions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete report_definition SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete report_definition: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgDefinitionRepository) List(ctx context.Context, filter DefinitionFilter) (*model.ListResponse[ReportDefinition], error) {
	base := psql.Select(definitionColumns...).From("report_definitions")
	countBase := psql.Select("COUNT(*)").From("report_definitions")

	if filter.ReportType != nil {
		base = base.Where(sq.Eq{"report_type": string(*filter.ReportType)})
		countBase = countBase.Where(sq.Eq{"report_type": string(*filter.ReportType)})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": string(*filter.Status)})
		countBase = countBase.Where(sq.Eq{"status": string(*filter.Status)})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count report_definition SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count report_definitions: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list report_definition SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list report_definitions: %w", err)
	}
	defer rows.Close()

	var items []ReportDefinition
	for rows.Next() {
		def, err := scanDefinitionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report_definition row: %w", err)
		}
		items = append(items, *def)
	}

	if items == nil {
		items = []ReportDefinition{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- Definition scanning helpers ----

func scanDefinition(row pgx.Row) (*ReportDefinition, error) {
	var d ReportDefinition
	var (
		description    sql.NullString
		formatJSON     []byte
		kpiJSON        []byte
		groupsJSON     []byte
		cronExpression sql.NullString
		creator        sql.NullString
		lastGenTime    sql.NullTime
	)

	err := row.Scan(
		&d.ID, &d.ReportName, &d.ReportType, &description, &formatJSON, &d.Period,
		&kpiJSON, &groupsJSON, &d.AutoGenerate, &cronExpression,
		&d.Status, &creator, &lastGenTime, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		d.Description = description.String
	}
	if cronExpression.Valid {
		d.CronExpression = cronExpression.String
	}
	if creator.Valid {
		d.Creator = creator.String
	}
	if lastGenTime.Valid {
		d.LastGenTime = &lastGenTime.Time
	}

	if formatJSON != nil {
		if unmarshalErr := json.Unmarshal(formatJSON, &d.Format); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshal format: %w", unmarshalErr)
		}
	}
	if d.Format == nil {
		d.Format = []string{}
	}

	if kpiJSON != nil {
		if unmarshalErr := json.Unmarshal(kpiJSON, &d.KPICodes); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshal kpi_codes: %w", unmarshalErr)
		}
	}
	if d.KPICodes == nil {
		d.KPICodes = []string{}
	}

	if groupsJSON != nil {
		if unmarshalErr := json.Unmarshal(groupsJSON, &d.DeviceGroups); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshal device_groups: %w", unmarshalErr)
		}
	}
	if d.DeviceGroups == nil {
		d.DeviceGroups = []string{}
	}

	return &d, nil
}

func scanDefinitionRow(rows pgx.Rows) (*ReportDefinition, error) {
	var d ReportDefinition
	var (
		description    sql.NullString
		formatJSON     []byte
		kpiJSON        []byte
		groupsJSON     []byte
		cronExpression sql.NullString
		creator        sql.NullString
		lastGenTime    sql.NullTime
	)

	err := rows.Scan(
		&d.ID, &d.ReportName, &d.ReportType, &description, &formatJSON, &d.Period,
		&kpiJSON, &groupsJSON, &d.AutoGenerate, &cronExpression,
		&d.Status, &creator, &lastGenTime, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		d.Description = description.String
	}
	if cronExpression.Valid {
		d.CronExpression = cronExpression.String
	}
	if creator.Valid {
		d.Creator = creator.String
	}
	if lastGenTime.Valid {
		d.LastGenTime = &lastGenTime.Time
	}

	if formatJSON != nil {
		if unmarshalErr := json.Unmarshal(formatJSON, &d.Format); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshal format: %w", unmarshalErr)
		}
	}
	if d.Format == nil {
		d.Format = []string{}
	}

	if kpiJSON != nil {
		if unmarshalErr := json.Unmarshal(kpiJSON, &d.KPICodes); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshal kpi_codes: %w", unmarshalErr)
		}
	}
	if d.KPICodes == nil {
		d.KPICodes = []string{}
	}

	if groupsJSON != nil {
		if unmarshalErr := json.Unmarshal(groupsJSON, &d.DeviceGroups); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshal device_groups: %w", unmarshalErr)
		}
	}
	if d.DeviceGroups == nil {
		d.DeviceGroups = []string{}
	}

	return &d, nil
}

// ======================================================================
// PgRecordRepository
// ======================================================================

var _ RecordRepository = (*PgRecordRepository)(nil)

// PgRecordRepository is a PostgreSQL implementation of RecordRepository.
type PgRecordRepository struct {
	pool *pgxpool.Pool
}

// NewPgRecordRepository creates a new PgRecordRepository.
func NewPgRecordRepository(pool *pgxpool.Pool) *PgRecordRepository {
	return &PgRecordRepository{pool: pool}
}

func (r *PgRecordRepository) Create(ctx context.Context, record *ReportRecord) error {
	query, args, err := psql.Insert("report_records").
		Columns("report_definition_id", "report_name", "period", "generate_time",
			"file_size", "download_url", "format", "status", "minio_path").
		Values(record.ReportDefinitionID, record.ReportName, nullableString(record.Period),
			record.GenerateTime, record.FileSize, nullableString(record.DownloadURL),
			record.Format, record.Status, nullableString(record.MinioPath)).
		Suffix("RETURNING " + joinColumns(recordColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert report_record SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanRecord(row)
	if err != nil {
		return fmt.Errorf("create report_record: %w", err)
	}
	*record = *created
	return nil
}

func (r *PgRecordRepository) GetByID(ctx context.Context, id uuid.UUID) (*ReportRecord, error) {
	query, args, err := psql.Select(recordColumns...).
		From("report_records").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get report_record SQL: %w", err)
	}

	record, err := scanRecord(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get report_record: %w", err)
	}
	return record, nil
}

func (r *PgRecordRepository) Update(ctx context.Context, record *ReportRecord) error {
	query, args, err := psql.Update("report_records").
		Set("status", record.Status).
		Set("file_size", record.FileSize).
		Set("minio_path", nullableString(record.MinioPath)).
		Set("download_url", nullableString(record.DownloadURL)).
		Where(sq.Eq{"id": record.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update report_record SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update report_record: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgRecordRepository) List(ctx context.Context, filter RecordFilter) (*model.ListResponse[ReportRecord], error) {
	base := psql.Select(recordColumns...).From("report_records")
	countBase := psql.Select("COUNT(*)").From("report_records")

	if filter.DefinitionID != nil {
		base = base.Where(sq.Eq{"report_definition_id": *filter.DefinitionID})
		countBase = countBase.Where(sq.Eq{"report_definition_id": *filter.DefinitionID})
	}
	if filter.Format != nil {
		base = base.Where(sq.Eq{"format": *filter.Format})
		countBase = countBase.Where(sq.Eq{"format": *filter.Format})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count report_record SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count report_records: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "generate_time"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list report_record SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list report_records: %w", err)
	}
	defer rows.Close()

	var items []ReportRecord
	for rows.Next() {
		record, err := scanRecordRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report_record row: %w", err)
		}
		items = append(items, *record)
	}

	if items == nil {
		items = []ReportRecord{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- Record scanning helpers ----

func scanRecord(row pgx.Row) (*ReportRecord, error) {
	var rec ReportRecord
	var (
		period      sql.NullString
		downloadURL sql.NullString
		minioPath   sql.NullString
	)

	err := row.Scan(
		&rec.ID, &rec.ReportDefinitionID, &rec.ReportName, &period, &rec.GenerateTime,
		&rec.FileSize, &downloadURL, &rec.Format, &rec.Status, &minioPath, &rec.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if period.Valid {
		rec.Period = period.String
	}
	if downloadURL.Valid {
		rec.DownloadURL = downloadURL.String
	}
	if minioPath.Valid {
		rec.MinioPath = minioPath.String
	}

	return &rec, nil
}

func scanRecordRow(rows pgx.Rows) (*ReportRecord, error) {
	var rec ReportRecord
	var (
		period      sql.NullString
		downloadURL sql.NullString
		minioPath   sql.NullString
	)

	err := rows.Scan(
		&rec.ID, &rec.ReportDefinitionID, &rec.ReportName, &period, &rec.GenerateTime,
		&rec.FileSize, &downloadURL, &rec.Format, &rec.Status, &minioPath, &rec.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if period.Valid {
		rec.Period = period.String
	}
	if downloadURL.Valid {
		rec.DownloadURL = downloadURL.String
	}
	if minioPath.Valid {
		rec.MinioPath = minioPath.String
	}

	return &rec, nil
}

// ---- shared helpers ----

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func joinColumns(cols []string) string {
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
