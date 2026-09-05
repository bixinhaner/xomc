package agentassistant

import (
	"context"
	"fmt"
	"net/url"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/attention"
	"github.com/omcgo/omcgo/internal/authz"
)

// AttentionSource is an owner-only projection of the existing run ledger. It
// never copies private reports into the legacy group-visible finding store.
type AttentionSource struct{ repo *Repository }

func NewAttentionSource(repo *Repository) *AttentionSource { return &AttentionSource{repo: repo} }
func (s *AttentionSource) Name() string                    { return "assistant_result" }
func (s *AttentionSource) Rank() int                       { return 360 }
func (s *AttentionSource) Permission() attention.Permission {
	return attention.Permission{OwnerOnly: true}
}
func (s *AttentionSource) ListPrefix(ctx context.Context, scope attention.Scope, limit int) (attention.SourceResult, error) {
	return s.ListWindow(ctx, scope, 0, limit)
}
func (s *AttentionSource) ListWindow(ctx context.Context, scope attention.Scope, offset, limit int) (attention.SourceResult, error) {
	digest := scopeDigest(&admin.Claims{UserID: scope.UserID, IsSuperAdmin: scope.IsSuperAdmin}, scope.VisibleGroups)
	q := sql.Select(runColumns).From("agent_assistant_runs").Where(sq.Eq{"owner_id": scope.UserID, "notify_visible": true, "read_at": nil}).Where(sq.GtOrEq{"created_at": time.Now().Add(-7 * 24 * time.Hour)}).Where(sq.Expr("principal->>'scopeDigest'=?", digest))
	if !scope.IsSuperAdmin {
		q = q.Where(sq.Expr("EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id=agent_assistant_runs.owner_id AND ur.role_id::text=principal->>'roleId')"))
		groups, ungrouped := authz.SplitVisibleGroups(scope.VisibleGroups)
		deviceScope := sq.Select("1").From("devices d").Where("d.id::text=agent_assistant_runs.request->'definition'->'scope'->>'deviceId'").Where(sq.Eq{"d.deleted_at": nil})
		member := sq.Expr("EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id=d.id AND m.group_id = ANY(?))", groups)
		if ungrouped {
			deviceScope = deviceScope.Where(sq.Or{member, sq.Expr("NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id=d.id)")})
		} else {
			deviceScope = deviceScope.Where(member)
		}
		q = q.Where(sq.Or{sq.Expr("request->'definition'->'scope'->>'kind'='visible'"), sq.Expr("EXISTS (?)", deviceScope)})
	}
	count, args, err := q.RemoveColumns().Column("COUNT(*)").ToSql()
	if err != nil {
		return attention.SourceResult{}, fmt.Errorf("build inbox count: %w", err)
	}
	var total int64
	if err = s.repo.pool.QueryRow(ctx, count, args...).Scan(&total); err != nil {
		return attention.SourceResult{}, fmt.Errorf("count assistant inbox: %w", err)
	}
	text, args, err := q.OrderBy("created_at DESC").Limit(uint64(max(1, min(limit, 50)))).Offset(uint64(max(0, offset))).ToSql()
	if err != nil {
		return attention.SourceResult{}, fmt.Errorf("build assistant inbox: %w", err)
	}
	rows, err := s.repo.pool.Query(ctx, text, args...)
	if err != nil {
		return attention.SourceResult{}, fmt.Errorf("list assistant inbox: %w", err)
	}
	runs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Run, error) { return scanRun(row) })
	if err != nil {
		return attention.SourceResult{}, err
	}
	items := make([]attention.Item, 0, len(runs))
	for _, run := range runs {
		title, summary, severity := run.Request.Definition.Name, run.ErrorCode, "medium"
		if run.Output != nil {
			title = run.Output.Title
			summary = run.Output.Summary
			if run.Output.Outcome == "no_change" {
				severity = "info"
			}
		}
		items = append(items, attention.Item{ID: "assistant:" + run.ID, Kind: attention.KindAssistantResult, Source: "assistant", SourceID: run.ID, Title: title, Summary: summary, Severity: severity, Priority: severity, CreatedAt: &run.CreatedAt, DetailRoute: "/system/active-intelligence?assistant=" + url.QueryEscape(run.AssistantID) + "&run=" + url.QueryEscape(run.ID), AllowedActions: []attention.Action{attention.ActionViewAssistant}})
	}
	return attention.SourceResult{Total: total, Items: items}, nil
}
