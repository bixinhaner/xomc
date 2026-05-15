package ufte

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

type TaskTypeRepository interface {
	List(ctx context.Context) ([]TaskType, error)
	GetByCode(ctx context.Context, typeCode string) (*TaskType, error)
	Upsert(ctx context.Context, item *TaskType) error
	Delete(ctx context.Context, typeCode string) error
}

type PgTaskTypeRepository struct {
	pool *pgxpool.Pool
}

func NewPgTaskTypeRepository(pool *pgxpool.Pool) *PgTaskTypeRepository {
	return &PgTaskTypeRepository{pool: pool}
}

var taskTypeColumns = []string{
	"type_code",
	"category",
	"category_label",
	"display_name",
	"description",
	"rpc_type",
	"built_in",
	"enabled",
	"step_chain",
	"post_tc_event_code",
	"permission_code",
	"platform_scope",
	"file_type",
	"file_type_label",
	"file_type_editable",
	"url_template",
	"target_file_name_template",
	"file_name_template",
	"file_size_field",
	"checksum_field",
	"raw_mode",
	"delay_seconds",
	"transport_path",
	"last_editor",
	"created_at",
	"updated_at",
}

func (r *PgTaskTypeRepository) List(ctx context.Context) ([]TaskType, error) {
	query, args, err := storage.Psql.Select(taskTypeColumns...).
		From("ufte_task_types").
		OrderBy("built_in DESC", "category_label ASC", "display_name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list UFTE task types SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list UFTE task types: %w", err)
	}
	defer rows.Close()

	items := make([]TaskType, 0)
	for rows.Next() {
		item, err := scanTaskType(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate UFTE task types: %w", err)
	}
	return items, nil
}

func (r *PgTaskTypeRepository) GetByCode(ctx context.Context, typeCode string) (*TaskType, error) {
	query, args, err := storage.Psql.Select(taskTypeColumns...).
		From("ufte_task_types").
		Where(sq.Eq{"type_code": typeCode}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get UFTE task type SQL: %w", err)
	}
	item, err := scanTaskType(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get UFTE task type: %w", err)
	}
	return item, nil
}

func (r *PgTaskTypeRepository) Upsert(ctx context.Context, item *TaskType) error {
	stepChain, err := json.Marshal(normalizeStringSlice(item.StepChain))
	if err != nil {
		return fmt.Errorf("marshal UFTE step chain: %w", err)
	}
	platformScope, err := json.Marshal(normalizeStringSlice(item.PlatformScope))
	if err != nil {
		return fmt.Errorf("marshal UFTE platform scope: %w", err)
	}
	now := time.Now()
	if item.LastEditor == "" {
		item.LastEditor = "system"
	}
	query, args, err := storage.Psql.Insert("ufte_task_types").
		Columns(
			"type_code",
			"category",
			"category_label",
			"display_name",
			"description",
			"rpc_type",
			"built_in",
			"enabled",
			"step_chain",
			"post_tc_event_code",
			"permission_code",
			"platform_scope",
			"file_type",
			"file_type_label",
			"file_type_editable",
			"url_template",
			"target_file_name_template",
			"file_name_template",
			"file_size_field",
			"checksum_field",
			"raw_mode",
			"delay_seconds",
			"transport_path",
			"last_editor",
			"created_at",
			"updated_at",
		).
		Values(
			item.TypeCode,
			item.Category,
			item.CategoryLabel,
			item.DisplayName,
			item.Description,
			item.RPCType,
			item.BuiltIn,
			item.Enabled,
			stepChain,
			item.PostTCEventCode,
			item.PermissionCode,
			platformScope,
			item.FileType,
			item.FileTypeLabel,
			item.FileTypeEditable,
			item.URLTemplate,
			item.TargetFileNameTemplate,
			item.FileNameTemplate,
			item.FileSizeField,
			item.ChecksumField,
			item.RawMode,
			item.DelaySeconds,
			item.TransportPath,
			item.LastEditor,
			now,
			now,
		).
		Suffix(`ON CONFLICT (type_code) DO UPDATE SET
			category = EXCLUDED.category,
			category_label = EXCLUDED.category_label,
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			rpc_type = EXCLUDED.rpc_type,
			built_in = EXCLUDED.built_in,
			enabled = EXCLUDED.enabled,
			step_chain = EXCLUDED.step_chain,
			post_tc_event_code = EXCLUDED.post_tc_event_code,
			permission_code = EXCLUDED.permission_code,
			platform_scope = EXCLUDED.platform_scope,
			file_type = EXCLUDED.file_type,
			file_type_label = EXCLUDED.file_type_label,
			file_type_editable = EXCLUDED.file_type_editable,
			url_template = EXCLUDED.url_template,
			target_file_name_template = EXCLUDED.target_file_name_template,
			file_name_template = EXCLUDED.file_name_template,
			file_size_field = EXCLUDED.file_size_field,
			checksum_field = EXCLUDED.checksum_field,
			raw_mode = EXCLUDED.raw_mode,
			delay_seconds = EXCLUDED.delay_seconds,
			transport_path = EXCLUDED.transport_path,
			last_editor = EXCLUDED.last_editor,
			updated_at = EXCLUDED.updated_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert UFTE task type SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert UFTE task type: %w", err)
	}
	item.UpdatedAt = formatTime(now)
	return nil
}

func (r *PgTaskTypeRepository) Delete(ctx context.Context, typeCode string) error {
	query, args, err := storage.Psql.Delete("ufte_task_types").
		Where(sq.Eq{"type_code": typeCode}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete UFTE task type SQL: %w", err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete UFTE task type: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

type taskTypeScanner interface {
	Scan(dest ...any) error
}

func scanTaskType(scanner taskTypeScanner) (*TaskType, error) {
	var item TaskType
	var stepChainRaw []byte
	var platformScopeRaw []byte
	var createdAt time.Time
	var updatedAt time.Time
	if err := scanner.Scan(
		&item.TypeCode,
		&item.Category,
		&item.CategoryLabel,
		&item.DisplayName,
		&item.Description,
		&item.RPCType,
		&item.BuiltIn,
		&item.Enabled,
		&stepChainRaw,
		&item.PostTCEventCode,
		&item.PermissionCode,
		&platformScopeRaw,
		&item.FileType,
		&item.FileTypeLabel,
		&item.FileTypeEditable,
		&item.URLTemplate,
		&item.TargetFileNameTemplate,
		&item.FileNameTemplate,
		&item.FileSizeField,
		&item.ChecksumField,
		&item.RawMode,
		&item.DelaySeconds,
		&item.TransportPath,
		&item.LastEditor,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan UFTE task type: %w", err)
	}
	if err := json.Unmarshal(stepChainRaw, &item.StepChain); err != nil {
		return nil, fmt.Errorf("unmarshal UFTE step chain: %w", err)
	}
	if err := json.Unmarshal(platformScopeRaw, &item.PlatformScope); err != nil {
		return nil, fmt.Errorf("unmarshal UFTE platform scope: %w", err)
	}
	item.StepChain = normalizeStringSlice(item.StepChain)
	item.PlatformScope = normalizeStringSlice(item.PlatformScope)
	item.UpdatedAt = formatTime(updatedAt)
	return &item, nil
}
