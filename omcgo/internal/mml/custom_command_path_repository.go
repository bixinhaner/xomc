package mml

import (
	"context"
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
func (r *PgCustomCommandPathRepository) ListByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCustomCommandPathView, error) {
	const q = `
		SELECT ccp.id, ccp.command_id, ccp.standard_path_id,
		       sp.standard_path, sp.entry_type,
		       COALESCE(sp.access, ''), COALESCE(sp.data_type, ''), COALESCE(sp.description, ''),
		       sp.min_value, sp.max_value,
		       ccp.default_selected, ccp.sort_order
		FROM mml_custom_command_paths ccp
		JOIN standard_params sp ON sp.id = ccp.standard_path_id
		WHERE ccp.command_id = $1
		ORDER BY ccp.sort_order ASC, sp.standard_path ASC`
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
			&v.DefaultSelected, &v.SortOrder); err != nil {
			return nil, fmt.Errorf("scan custom command path: %w", err)
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom command paths: %w", err)
	}
	return items, nil
}

// BatchCreate 把 standardPathIDs 追加到命令的 path 集合（sort_order 接现有末尾）。
// 已关联的 path（命中 uq_ccp_command_path）静默跳过，返回实际新建的行。
func (r *PgCustomCommandPathRepository) BatchCreate(ctx context.Context, commandID uuid.UUID, standardPathIDs []uuid.UUID) ([]MMLCustomCommandPath, error) {
	if len(standardPathIDs) == 0 {
		return nil, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var nextOrder int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM mml_custom_command_paths WHERE command_id = $1`,
		commandID).Scan(&nextOrder); err != nil {
		return nil, fmt.Errorf("query next sort_order: %w", err)
	}

	created := make([]MMLCustomCommandPath, 0, len(standardPathIDs))
	for _, pathID := range standardPathIDs {
		var p MMLCustomCommandPath
		err := tx.QueryRow(ctx, `
			INSERT INTO mml_custom_command_paths (command_id, standard_path_id, sort_order)
			VALUES ($1, $2, $3)
			ON CONFLICT (command_id, standard_path_id) DO NOTHING
			RETURNING id, command_id, standard_path_id, default_selected, sort_order, created_at, updated_at`,
			commandID, pathID, nextOrder).Scan(
			&p.ID, &p.CommandID, &p.StandardPathID, &p.DefaultSelected, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			continue // 已关联，跳过，不占用 sort_order
		}
		if err != nil {
			return nil, fmt.Errorf("insert custom command path: %w", err)
		}
		created = append(created, p)
		nextOrder++
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return created, nil
}

// Update 改单条关联的 default_selected / sort_order（nil 字段不动）。
func (r *PgCustomCommandPathRepository) Update(ctx context.Context, commandID, pathID uuid.UUID, defaultSelected *bool, sortOrder *int) (*MMLCustomCommandPath, error) {
	var p MMLCustomCommandPath
	err := r.pool.QueryRow(ctx, `
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
	return &p, nil
}

// Delete 删单条关联（WHERE 限定 command_id）。
func (r *PgCustomCommandPathRepository) Delete(ctx context.Context, commandID, pathID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM mml_custom_command_paths WHERE id = $1 AND command_id = $2`,
		pathID, commandID)
	if err != nil {
		return fmt.Errorf("delete custom command path: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("custom command path %s not found: %w", pathID, commonerrors.ErrNotFound)
	}
	return nil
}
