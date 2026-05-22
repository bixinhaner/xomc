package mml

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================
// search_repository.go — 命令搜索（Bundle C）
//
// 设计：单条 SQL 跨表 ILIKE，按 commandID 聚合匹配到的 path / description，
// 让前端一次拿到命令 + 命中的 path 列表（用于 AddTemplateModal 显示 highlight 摘要）。
//
// 搜索维度：
//   1. command_code      （如 "LST DEVICE_INFO"）
//   2. logical_code      （如 "DEVICE_INFO"）
//   3. command_name_i18n.zh-CN / en-US （动词前缀 + 对象名）
//   4. logical_name_i18n.zh-CN / en-US （命令中文名权威表）
//   5. standard_path     （子字段绑定的 TR-181 path）
//   6. description       （path 中文含义说明，seed/156 回填）
//
// 性能：当前 catalog 190 条 standard 命令 × ~5 sub_fields ≈ 1000 行 join，
// 全表 ILIKE 在 dev 库下 < 20ms。规模扩到 10k 命令时需加 GIN 全文索引 + ts_query。
// ============================================================

// SearchCommandRow 是单条命令的搜索结果行。
type SearchCommandRow struct {
	CommandID      uuid.UUID         `json:"command_id"`
	CommandCode    string            `json:"command_code"`
	LogicalCode    string            `json:"logical_code"`
	OperationType  string            `json:"operation_type"`
	DisplayName    string            `json:"display_name"`   // lang 派生 "<OP> <command_zh_name>"
	LogicalName    string            `json:"logical_name"`   // lang 派生
	GroupID        uuid.UUID         `json:"group_id"`
	GroupCode      string            `json:"group_code"`     // "chapter:SA" 等
	GroupName      string            `json:"group_name"`     // lang 派生
	ChapterCode    string            `json:"chapter_code"`   // "SA" / "SB" / ... 用于排序
	MatchedPaths   []string          `json:"matched_paths"`  // ILIKE 命中的 path 列表（去重）
	MatchReasons   []string          `json:"match_reasons"`  // command_name / command_code / path / description
	LabelI18n      map[string]string `json:"-"`
	LogicalI18n    map[string]string `json:"-"`
	GroupNameI18n  map[string]string `json:"-"`
}

// SearchRepository 提供命令搜索能力。
type SearchRepository interface {
	SearchCommands(ctx context.Context, query string, limit int) ([]SearchCommandRow, error)
}

// PgSearchRepository PostgreSQL 实现。
type PgSearchRepository struct {
	pool *pgxpool.Pool
}

// NewPgSearchRepository 构造 PgSearchRepository。
func NewPgSearchRepository(pool *pgxpool.Pool) *PgSearchRepository {
	return &PgSearchRepository{pool: pool}
}

var _ SearchRepository = (*PgSearchRepository)(nil)

// SearchCommands 跨表 ILIKE 搜索命令。
//
// query 空 → 返回空切片（避免误返回全表 LIMIT 50 给用户造成"无意义结果"幻觉）。
// limit 上限固定 200（防止前端误传超大值导致服务端 OOM）。
//
// 聚合策略：按 command_id 分组，把所有命中 path / description 的 standardPath 聚合
// 到 matched_paths 数组（去重、按字典序排序）；match_reasons 聚合命中维度。
func (r *PgSearchRepository) SearchCommands(ctx context.Context, query string, limit int) ([]SearchCommandRow, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []SearchCommandRow{}, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	pattern := "%" + q + "%"

	const sqlText = `
WITH name_hits AS (
    SELECT c.id
    FROM mml_commands c
    WHERE c.deprecated_at IS NULL
      AND c.source = 'standard'
      AND (
            c.command_code ILIKE $1
         OR COALESCE(c.logical_code, '') ILIKE $1
         OR COALESCE(c.command_name_i18n->>'zh-CN', '') ILIKE $1
         OR COALESCE(c.command_name_i18n->>'en-US', '') ILIKE $1
         OR COALESCE(c.logical_name_i18n->>'zh-CN', '') ILIKE $1
         OR COALESCE(c.logical_name_i18n->>'en-US', '') ILIKE $1
      )
),
path_hits AS (
    SELECT DISTINCT c.id, sp.standard_path,
           CASE WHEN sp.standard_path ILIKE $1 THEN 'path'
                ELSE 'description' END AS reason
    FROM mml_commands c
    JOIN mml_command_sub_fields csf ON csf.command_id = c.id
    JOIN standard_params sp ON sp.id = csf.standard_path_id
    WHERE c.deprecated_at IS NULL
      AND c.source = 'standard'
      AND (sp.standard_path ILIKE $1 OR COALESCE(sp.description, '') ILIKE $1)
),
matched_ids AS (
    SELECT id FROM name_hits
    UNION
    SELECT id FROM path_hits
)
SELECT
    c.id,
    c.command_code,
    COALESCE(c.logical_code, '')                AS logical_code,
    c.operation_type,
    COALESCE(c.command_name_i18n, '{}'::jsonb)  AS command_name_i18n,
    COALESCE(c.logical_name_i18n, '{}'::jsonb)  AS logical_name_i18n,
    g.id                                        AS group_id,
    g.group_code,
    COALESCE(g.name_i18n, '{}'::jsonb)          AS group_name_i18n,
    COALESCE(g.chapter_code, '')                AS chapter_code,
    COALESCE(
        (
            SELECT array_agg(DISTINCT ph.standard_path ORDER BY ph.standard_path)
            FROM path_hits ph
            WHERE ph.id = c.id
        ),
        '{}'::text[]
    ) AS matched_paths,
    EXISTS (SELECT 1 FROM name_hits nh WHERE nh.id = c.id)                AS hit_name,
    EXISTS (SELECT 1 FROM path_hits ph WHERE ph.id = c.id AND ph.reason='path')        AS hit_path,
    EXISTS (SELECT 1 FROM path_hits ph WHERE ph.id = c.id AND ph.reason='description') AS hit_desc
FROM mml_commands c
JOIN mml_command_groups g ON g.id = c.group_id
WHERE c.id IN (SELECT id FROM matched_ids)
ORDER BY g.chapter_code NULLS LAST, g.display_order, c.operation_type, c.command_code
LIMIT $2`

	rows, err := r.pool.Query(ctx, sqlText, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search commands: %w", err)
	}
	defer rows.Close()

	out := make([]SearchCommandRow, 0, limit)
	for rows.Next() {
		var (
			row              SearchCommandRow
			cmdNameI18nBytes []byte
			logNameI18nBytes []byte
			grpNameI18nBytes []byte
			hitName          bool
			hitPath          bool
			hitDesc          bool
		)
		if err := rows.Scan(
			&row.CommandID,
			&row.CommandCode,
			&row.LogicalCode,
			&row.OperationType,
			&cmdNameI18nBytes,
			&logNameI18nBytes,
			&row.GroupID,
			&row.GroupCode,
			&grpNameI18nBytes,
			&row.ChapterCode,
			&row.MatchedPaths,
			&hitName,
			&hitPath,
			&hitDesc,
		); err != nil {
			return nil, fmt.Errorf("scan search row: %w", err)
		}
		row.LabelI18n = parseI18nMap(cmdNameI18nBytes)
		row.LogicalI18n = parseI18nMap(logNameI18nBytes)
		row.GroupNameI18n = parseI18nMap(grpNameI18nBytes)
		if hitName {
			row.MatchReasons = append(row.MatchReasons, "command_name")
		}
		if hitPath {
			row.MatchReasons = append(row.MatchReasons, "path")
		}
		if hitDesc {
			row.MatchReasons = append(row.MatchReasons, "description")
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search rows: %w", err)
	}
	return out, nil
}

// parseI18nMap 安全解析 JSONB i18n 列；空 / 错误 → 空 map。
func parseI18nMap(b []byte) map[string]string {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	m := make(map[string]string)
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}
