package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// custom_command_path_repository.go — issue #115 调整3（A1）
//
// 自定义命令↔标准路径关联表 mml_custom_command_paths 的持久层。
// 写方法均以 (commandID, pathID) 双键限定，防跨命令越改/越删（service 已先做
// owner 鉴权，这里 SQL 层再加一道 command_id 约束作纵深防御）。

// CustomCommandPathRepository 关联表访问接口。
type CustomCommandPathRepository interface {
	ListByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCustomCommandPathView, error)
	BatchCreate(ctx context.Context, commandID uuid.UUID, standardPathIDs []uuid.UUID) ([]MMLCustomCommandPath, error)
	Update(ctx context.Context, commandID, pathID uuid.UUID, defaultSelected *bool, sortOrder *int) (*MMLCustomCommandPath, error)
	Delete(ctx context.Context, commandID, pathID uuid.UUID) error
}

// PgCustomCommandPathRepository PostgreSQL 实现。
type PgCustomCommandPathRepository struct {
	pool *pgxpool.Pool
}

// NewPgCustomCommandPathRepository 构造。
func NewPgCustomCommandPathRepository(pool *pgxpool.Pool) *PgCustomCommandPathRepository {
	return &PgCustomCommandPathRepository{pool: pool}
}

var _ CustomCommandPathRepository = (*PgCustomCommandPathRepository)(nil)

// ListByCommand 返回命令的全部关联 path（JOIN standard_params 富化），按 sort_order 排序。
//
// 正常创建/更新链路会同步 mml_custom_command_paths；为兼容同步逻辑上线前仅有
// mml_custom_command.param_paths JSON 的历史命令，查询还会补入未关联的 JSON Path。
// 兼容行 mutable=false，id 仅作为稳定展示标识，不可传给关联写接口。
func (r *PgCustomCommandPathRepository) ListByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCustomCommandPathView, error) {
	const q = `
		WITH legacy_paths AS (
			SELECT command.id AS command_id,
			       normalized.standard_path,
			       min(raw.sort_order) AS sort_order
			FROM mml_custom_command command
			CROSS JOIN LATERAL jsonb_array_elements_text(
				COALESCE(command.param_paths, '[]'::jsonb)
			) WITH ORDINALITY AS raw(standard_path, sort_order)
			CROSS JOIN LATERAL (
				VALUES (
					regexp_replace(
						raw.standard_path,
						'^[[:space:]]+|[[:space:]]+$',
						'',
						'g'
					)
				)
			) AS normalized(standard_path)
			WHERE command.id = $1
			  AND normalized.standard_path <> ''
			GROUP BY command.id, normalized.standard_path
		),
		effective_paths AS (
			SELECT ccp.id, ccp.command_id, ccp.standard_path_id,
			       sp.standard_path, sp.entry_type,
			       COALESCE(sp.access, '') AS access,
			       COALESCE(sp.data_type, '') AS data_type,
			       COALESCE(sp.description, '') AS description,
			       sp.min_value, sp.max_value,
			       ccp.default_selected, ccp.sort_order,
			       true AS mutable
			FROM mml_custom_command_paths ccp
			JOIN standard_params sp ON sp.id = ccp.standard_path_id
			WHERE ccp.command_id = $1

			UNION ALL

			SELECT sp.id AS id, legacy.command_id, sp.id AS standard_path_id,
			       sp.standard_path, sp.entry_type,
			       COALESCE(sp.access, '') AS access,
			       COALESCE(sp.data_type, '') AS data_type,
			       COALESCE(sp.description, '') AS description,
			       sp.min_value, sp.max_value,
			       false AS default_selected,
			       (legacy.sort_order - 1)::integer AS sort_order,
			       false AS mutable
			FROM legacy_paths legacy
			JOIN standard_params sp ON sp.standard_path = legacy.standard_path
			WHERE NOT EXISTS (
				SELECT 1
				FROM mml_custom_command_paths linked
				WHERE linked.command_id = legacy.command_id
				  AND linked.standard_path_id = sp.id
			  )
		)
		SELECT id, command_id, standard_path_id,
		       standard_path, entry_type, access, data_type, description,
		       min_value, max_value, default_selected, sort_order, mutable
		FROM effective_paths
		ORDER BY sort_order ASC, standard_path ASC`
	rows, err := r.pool.Query(ctx, q, commandID)
	if err != nil {
		return nil, fmt.Errorf("list custom command paths: %w", err)
	}
	defer rows.Close()

	items := make([]MMLCustomCommandPathView, 0)
	for rows.Next() {
		var v MMLCustomCommandPathView
		if err := rows.Scan(&v.ID, &v.CommandID, &v.StandardPathID,
			&v.StandardPath, &v.EntryType, &v.Access, &v.DataType, &v.Description,
			&v.MinValue, &v.MaxValue,
			&v.DefaultSelected, &v.SortOrder, &v.Mutable); err != nil {
			return nil, fmt.Errorf("scan custom command path: %w", err)
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom command paths: %w", err)
	}
	return items, nil
}

func lockCustomCommandParamPaths(
	ctx context.Context,
	tx pgx.Tx,
	commandID uuid.UUID,
) ([]string, error) {
	var raw []byte
	err := tx.QueryRow(ctx,
		`SELECT param_paths FROM mml_custom_command WHERE id = $1 FOR UPDATE`,
		commandID,
	).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("custom command %s not found: %w", commandID, commonerrors.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("lock custom command paths: %w", err)
	}

	var paths []string
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &paths); err != nil {
			return nil, fmt.Errorf("unmarshal custom command paths: %w", err)
		}
	}
	return normalizeCustomCommandParamPaths(paths), nil
}

func updateCustomCommandParamPaths(
	ctx context.Context,
	tx pgx.Tx,
	commandID uuid.UUID,
	paths []string,
) error {
	normalized := normalizeCustomCommandParamPaths(paths)
	raw, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal custom command paths: %w", err)
	}
	ct, err := tx.Exec(ctx, `
		UPDATE mml_custom_command
		SET param_paths = $2, updated_at = now()
		WHERE id = $1`,
		commandID,
		raw,
	)
	if err != nil {
		return fmt.Errorf("update custom command paths: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("custom command %s not found: %w", commandID, commonerrors.ErrNotFound)
	}
	return nil
}

func listLinkedCustomCommandParamPaths(
	ctx context.Context,
	tx pgx.Tx,
	commandID uuid.UUID,
) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT sp.standard_path
		FROM mml_custom_command_paths ccp
		JOIN standard_params sp ON sp.id = ccp.standard_path_id
		WHERE ccp.command_id = $1
		ORDER BY ccp.sort_order ASC, sp.standard_path ASC`,
		commandID,
	)
	if err != nil {
		return nil, fmt.Errorf("list linked custom command paths: %w", err)
	}
	defer rows.Close()

	paths := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan linked custom command path: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate linked custom command paths: %w", err)
	}
	return paths, nil
}

// BatchCreate 把 standardPathIDs 追加到命令的 path 集合（sort_order 接现有末尾）。
// 已在 param_paths 中的 path 静默跳过，返回实际新增为有效命令 Path 的关联行。
func (r *PgCustomCommandPathRepository) BatchCreate(ctx context.Context, commandID uuid.UUID, standardPathIDs []uuid.UUID) ([]MMLCustomCommandPath, error) {
	if len(standardPathIDs) == 0 {
		return nil, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existingPaths, err := lockCustomCommandParamPaths(ctx, tx, commandID)
	if err != nil {
		return nil, err
	}

	uniqueIDs := make([]uuid.UUID, 0, len(standardPathIDs))
	seenIDs := make(map[uuid.UUID]struct{}, len(standardPathIDs))
	for _, pathID := range standardPathIDs {
		if _, exists := seenIDs[pathID]; exists {
			continue
		}
		seenIDs[pathID] = struct{}{}
		uniqueIDs = append(uniqueIDs, pathID)
	}

	rows, err := tx.Query(ctx, `
		SELECT requested.id, sp.standard_path
		FROM unnest($1::uuid[]) WITH ORDINALITY AS requested(id, sort_order)
		JOIN standard_params sp ON sp.id = requested.id
		ORDER BY requested.sort_order`,
		uniqueIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve custom command standard path ids: %w", err)
	}
	resolvedPaths := make(map[uuid.UUID]string, len(uniqueIDs))
	for rows.Next() {
		var id uuid.UUID
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan custom command standard path id: %w", err)
		}
		resolvedPaths[id] = path
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate custom command standard path ids: %w", err)
	}
	rows.Close()
	if len(resolvedPaths) != len(uniqueIDs) {
		return nil, fmt.Errorf("one or more standard_path_ids not found: %w", commonerrors.ErrInvalidInput)
	}

	effective := append([]string(nil), existingPaths...)
	effectiveSet := make(map[string]struct{}, len(effective))
	for _, path := range effective {
		effectiveSet[path] = struct{}{}
	}
	addedIDs := make([]uuid.UUID, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		path := resolvedPaths[id]
		if _, exists := effectiveSet[path]; exists {
			continue
		}
		effectiveSet[path] = struct{}{}
		effective = append(effective, path)
		addedIDs = append(addedIDs, id)
	}

	if err := updateCustomCommandParamPaths(ctx, tx, commandID, effective); err != nil {
		return nil, err
	}
	if err := syncCustomCommandPaths(ctx, tx, commandID, effective); err != nil {
		return nil, fmt.Errorf("sync custom command paths: %w", err)
	}

	created := make([]MMLCustomCommandPath, 0, len(addedIDs))
	for _, pathID := range addedIDs {
		var p MMLCustomCommandPath
		if err := tx.QueryRow(ctx, `
			SELECT id, command_id, standard_path_id, default_selected,
			       sort_order, created_at, updated_at
			FROM mml_custom_command_paths
			WHERE command_id = $1 AND standard_path_id = $2`,
			commandID,
			pathID,
		).Scan(
			&p.ID,
			&p.CommandID,
			&p.StandardPathID,
			&p.DefaultSelected,
			&p.SortOrder,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("read created custom command path: %w", err)
		}
		created = append(created, p)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return created, nil
}

// Update 改单条关联的 default_selected / sort_order（nil 字段不动）。
func (r *PgCustomCommandPathRepository) Update(ctx context.Context, commandID, pathID uuid.UUID, defaultSelected *bool, sortOrder *int) (*MMLCustomCommandPath, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existingPaths, err := lockCustomCommandParamPaths(ctx, tx, commandID)
	if err != nil {
		return nil, err
	}
	if err := syncCustomCommandPaths(ctx, tx, commandID, existingPaths); err != nil {
		return nil, fmt.Errorf("sync custom command paths before update: %w", err)
	}

	var p MMLCustomCommandPath
	err = tx.QueryRow(ctx, `
		UPDATE mml_custom_command_paths
		SET default_selected = COALESCE($3, default_selected),
		    sort_order = COALESCE($4, sort_order),
		    updated_at = now()
		WHERE id = $1 AND command_id = $2
		RETURNING id, command_id, standard_path_id, default_selected, sort_order, created_at, updated_at`,
		pathID, commandID, defaultSelected, sortOrder).Scan(
		&p.ID, &p.CommandID, &p.StandardPathID, &p.DefaultSelected, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("custom command path %s not found: %w", pathID, commonerrors.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("update custom command path: %w", err)
	}

	effective, err := listLinkedCustomCommandParamPaths(ctx, tx, commandID)
	if err != nil {
		return nil, err
	}
	if err := updateCustomCommandParamPaths(ctx, tx, commandID, effective); err != nil {
		return nil, err
	}
	if err := syncCustomCommandPaths(ctx, tx, commandID, effective); err != nil {
		return nil, fmt.Errorf("normalize custom command path order: %w", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT id, command_id, standard_path_id, default_selected,
		       sort_order, created_at, updated_at
		FROM mml_custom_command_paths
		WHERE id = $1 AND command_id = $2`,
		pathID,
		commandID,
	).Scan(
		&p.ID,
		&p.CommandID,
		&p.StandardPathID,
		&p.DefaultSelected,
		&p.SortOrder,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("read updated custom command path: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &p, nil
}

// Delete 删单条关联（WHERE 限定 command_id）。
func (r *PgCustomCommandPathRepository) Delete(ctx context.Context, commandID, pathID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existingPaths, err := lockCustomCommandParamPaths(ctx, tx, commandID)
	if err != nil {
		return err
	}
	if err := syncCustomCommandPaths(ctx, tx, commandID, existingPaths); err != nil {
		return fmt.Errorf("sync custom command paths before delete: %w", err)
	}

	ct, err := tx.Exec(ctx,
		`DELETE FROM mml_custom_command_paths WHERE id = $1 AND command_id = $2`,
		pathID, commandID)
	if err != nil {
		return fmt.Errorf("delete custom command path: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("custom command path %s not found: %w", pathID, commonerrors.ErrNotFound)
	}

	effective, err := listLinkedCustomCommandParamPaths(ctx, tx, commandID)
	if err != nil {
		return err
	}
	if err := updateCustomCommandParamPaths(ctx, tx, commandID, effective); err != nil {
		return err
	}
	if err := syncCustomCommandPaths(ctx, tx, commandID, effective); err != nil {
		return fmt.Errorf("normalize custom command paths after delete: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
