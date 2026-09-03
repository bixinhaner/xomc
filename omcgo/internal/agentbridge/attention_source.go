package agentbridge

import (
	"context"
	"fmt"
	"net/url"

	"github.com/omcgo/omcgo/internal/attention"
)

// AttentionSource projects active, visible findings into the existing dashboard
// attention stream. The finding repository remains authoritative for visibility
// and per-user dismissal state.
type AttentionSource struct {
	repo *FindingRepository
}

func NewAttentionSource(repo *FindingRepository) *AttentionSource {
	return &AttentionSource{repo: repo}
}

func (s *AttentionSource) Name() string { return "agent_finding" }
func (s *AttentionSource) Rank() int    { return 350 }

func (s *AttentionSource) Permission() attention.Permission {
	return attention.Permission{Resource: "/api/v1/agent/findings/:id", Action: "GET"}
}

func (s *AttentionSource) ListPrefix(ctx context.Context, scope attention.Scope, limit int) (attention.SourceResult, error) {
	return s.ListWindow(ctx, scope, 0, limit)
}

func (s *AttentionSource) ListWindow(ctx context.Context, scope attention.Scope, offset, limit int) (attention.SourceResult, error) {
	if s == nil || s.repo == nil {
		return attention.SourceResult{}, fmt.Errorf("agent finding attention repository is not configured")
	}
	findings, total, err := s.repo.List(ctx, FindingListOptions{
		UserID: scope.UserID, SuperAdmin: scope.IsSuperAdmin, VisibleGroups: scope.VisibleGroups,
		Limit: limit, Offset: offset,
	})
	if err != nil {
		return attention.SourceResult{}, fmt.Errorf("list agent findings for attention: %w", err)
	}
	items := make([]attention.Item, 0, len(findings))
	for _, finding := range findings {
		createdAt := finding.CreatedAt
		target := findingTarget(finding.Resources)
		items = append(items, attention.Item{
			ID: "abnormal:agent_finding:" + finding.ID, Kind: attention.KindAgentFinding,
			Source: "agent", SourceID: finding.ID, Title: finding.Title, Summary: finding.Summary,
			Severity: finding.Severity, Priority: finding.Severity, Target: target,
			CreatedAt: &createdAt, DetailRoute: "/dashboard?agentFinding=" + url.QueryEscape(finding.ID),
			AllowedActions: []attention.Action{attention.ActionViewFinding},
		})
	}
	return attention.SourceResult{Total: total, Items: items}, nil
}

func findingTarget(resources []ResourceRef) *attention.Target {
	for _, resource := range resources {
		if resource.Type != "device" {
			continue
		}
		return &attention.Target{Type: resource.Type, ID: resource.ID, Name: resource.Label}
	}
	return nil
}
