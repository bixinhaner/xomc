package catalogloader

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LinkHealthRow 是 mml_catalog_link_health 的一行视图。
// 对应 spec §R-2.5.2 标准参数树关联失败清单的一条记录。
type LinkHealthRow struct {
	ID              uuid.UUID  `json:"id"`
	SpecVersion     string     `json:"spec_version"`
	StandardPath    string     `json:"standard_path"`
	GroupCodeObject string     `json:"group_code_object"`
	OperationType   string     `json:"operation_type,omitempty"`
	FailureReason   string     `json:"failure_reason"` // A / B / C / D
	Notes           string     `json:"notes,omitempty"`
	Decision        string     `json:"decision,omitempty"`
	Owner           string     `json:"owner,omitempty"`
	DetectedAt      time.Time  `json:"detected_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

// LinkHealthListOptions 控制 ListLinkHealth 的过滤条件。
type LinkHealthListOptions struct {
	SpecVersion       string // 空表示全部 spec
	IncludeResolved   bool   // 默认 false：只返 resolved_at IS NULL 的未解决项
	Limit             int    // 0 → 默认 500；上限 5000 避免一次性巨大返回
}

// ListLinkHealth 查询 mml_catalog_link_health 表，按 spec_version / resolved 状态过滤。
// 排序：未解决优先（resolved_at IS NULL 在前），然后按 detected_at DESC。
//
// admin API GET /api/v1/admin/mml-catalog/link-health 直接消费本函数。
func ListLinkHealth(ctx context.Context, db *pgxpool.Pool, opt LinkHealthListOptions) ([]LinkHealthRow, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 500
	}
	if limit > 5000 {
		limit = 5000
	}

	// 用 1=1 + 可选 AND 拼装，简化条件多组合的代码。
	args := []any{}
	where := "WHERE 1=1"
	if opt.SpecVersion != "" {
		args = append(args, opt.SpecVersion)
		where += fmt.Sprintf(" AND spec_version = $%d", len(args))
	}
	if !opt.IncludeResolved {
		where += " AND resolved_at IS NULL"
	}
	args = append(args, limit)
	sqlText := fmt.Sprintf(`
		SELECT id, spec_version, standard_path, group_code_object,
		       COALESCE(operation_type, ''),
		       failure_reason,
		       COALESCE(notes, ''),
		       COALESCE(decision, ''),
		       COALESCE(owner, ''),
		       detected_at, resolved_at
		FROM mml_catalog_link_health
		%s
		ORDER BY (resolved_at IS NOT NULL), detected_at DESC
		LIMIT $%d
	`, where, len(args))

	rows, err := db.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("query mml_catalog_link_health: %w", err)
	}
	defer rows.Close()

	out := make([]LinkHealthRow, 0, limit)
	for rows.Next() {
		var r LinkHealthRow
		var resolved *time.Time
		if err := rows.Scan(
			&r.ID, &r.SpecVersion, &r.StandardPath, &r.GroupCodeObject,
			&r.OperationType,
			&r.FailureReason,
			&r.Notes,
			&r.Decision,
			&r.Owner,
			&r.DetectedAt, &resolved,
		); err != nil {
			return nil, fmt.Errorf("scan link_health row: %w", err)
		}
		r.ResolvedAt = resolved
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate link_health rows: %w", err)
	}
	return out, nil
}

// LinkHealthSummary 是按 spec / failure_reason 维度聚合的统计，admin UI 卡片用。
type LinkHealthSummary struct {
	SpecVersion       string `json:"spec_version"`
	Unresolved        int    `json:"unresolved"`
	Resolved          int    `json:"resolved"`
	ByReason          map[string]int `json:"by_reason"`         // A/B/C/D 各自的未解决数
}

// SummarizeLinkHealth 返回按 spec_version 聚合的统计。
// 用于 admin UI 顶部卡片快速展示"还有多少未解决"。
func SummarizeLinkHealth(ctx context.Context, db *pgxpool.Pool) ([]LinkHealthSummary, error) {
	const sql = `
		SELECT spec_version,
		       COUNT(*) FILTER (WHERE resolved_at IS NULL) AS unresolved,
		       COUNT(*) FILTER (WHERE resolved_at IS NOT NULL) AS resolved,
		       COUNT(*) FILTER (WHERE resolved_at IS NULL AND failure_reason = 'A') AS r_a,
		       COUNT(*) FILTER (WHERE resolved_at IS NULL AND failure_reason = 'B') AS r_b,
		       COUNT(*) FILTER (WHERE resolved_at IS NULL AND failure_reason = 'C') AS r_c,
		       COUNT(*) FILTER (WHERE resolved_at IS NULL AND failure_reason = 'D') AS r_d
		FROM mml_catalog_link_health
		GROUP BY spec_version
		ORDER BY spec_version
	`
	rows, err := db.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("summarize link_health: %w", err)
	}
	defer rows.Close()

	out := make([]LinkHealthSummary, 0, 4)
	for rows.Next() {
		var s LinkHealthSummary
		var rA, rB, rC, rD int
		if err := rows.Scan(&s.SpecVersion, &s.Unresolved, &s.Resolved, &rA, &rB, &rC, &rD); err != nil {
			return nil, fmt.Errorf("scan link_health summary: %w", err)
		}
		s.ByReason = map[string]int{"A": rA, "B": rB, "C": rC, "D": rD}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate link_health summary: %w", err)
	}
	return out, nil
}
