package attention

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/deviceaccess"
)

const candidateSourcePageSize = 200

type CandidateReader interface {
	ListCandidates(context.Context, deviceaccess.ManagementFilter) ([]deviceaccess.CandidateItem, int64, error)
}

type CandidateSource struct {
	reader CandidateReader
	now    func() time.Time
}

func NewCandidateSource(reader CandidateReader) *CandidateSource {
	return &CandidateSource{reader: reader, now: time.Now}
}

func (s *CandidateSource) Name() string { return "device_access_review" }

func (s *CandidateSource) Rank() int { return 200 }

func (s *CandidateSource) Permission() Permission {
	return Permission{Resource: "/api/v1/device-access/candidates/:candidateID/review", Action: "POST"}
}

func (s *CandidateSource) ListPrefix(ctx context.Context, scope Scope, limit int) (SourceResult, error) {
	return s.ListWindow(ctx, scope, 0, limit)
}

func (s *CandidateSource) ListWindow(ctx context.Context, scope Scope, offset, limit int) (SourceResult, error) {
	if s == nil || s.reader == nil {
		return SourceResult{}, fmt.Errorf("candidate attention reader is not configured")
	}
	if offset < 0 || limit < 1 {
		return SourceResult{Items: []Item{}}, nil
	}

	now := s.now()
	items := make([]Item, 0, limit)
	var total int64
	pageSize := min(candidateSourcePageSize, limit)
	page := offset/pageSize + 1
	withinPage := offset % pageSize
	firstQuery := true
	for len(items) < limit {
		candidates, count, err := s.reader.ListCandidates(ctx, deviceaccess.ManagementFilter{
			Status: "pending", ExpiresAfter: &now,
			Page: page, PageSize: pageSize,
			SortBy: "first_seen_at", SortDir: "asc",
			VisibleGroups: scope.VisibleGroups,
		})
		if err != nil {
			return SourceResult{}, fmt.Errorf("list pending device access candidates: %w", err)
		}
		if firstQuery {
			total = count
			firstQuery = false
		}
		start := min(withinPage, len(candidates))
		for index := start; index < len(candidates) && len(items) < limit; index++ {
			items = append(items, mapCandidate(candidates[index]))
		}
		if len(candidates) < pageSize || int64(offset+len(items)) >= total {
			break
		}
		page++
		withinPage = 0
	}
	return SourceResult{Total: total, Items: items}, nil
}

func mapCandidate(candidate deviceaccess.CandidateItem) Item {
	createdAt := candidate.FirstSeenAt
	targetName := strings.TrimSpace(candidate.SerialNumber)
	return Item{
		ID:   "todo:device_access_review:" + candidate.ID.String(),
		Kind: KindDeviceAccessReview, Source: "device_access", SourceID: candidate.ID.String(),
		Title: targetName, Summary: strings.ToUpper(strings.TrimSpace(candidate.Carrier)), Priority: "normal", Risk: "safe",
		Target:    &Target{Type: "device_candidate", ID: candidate.ID.String(), Name: targetName, SerialNumber: candidate.SerialNumber},
		CreatedAt: &createdAt,
		DetailRoute: "/device/access-control?tab=candidates&operator=" + url.QueryEscape(candidate.Carrier) +
			"&candidateId=" + candidate.ID.String() + "&reviewStatus=pending",
		AllowedActions: []Action{ActionReviewCandidate},
	}
}
