package attention

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubRankedSource struct {
	name        string
	rank        int
	permission  Permission
	result      SourceResult
	err         error
	lastLimit   int
	lastOffset  int
	windowCalls int
}

func (s *stubRankedSource) Name() string           { return s.name }
func (s *stubRankedSource) Rank() int              { return s.rank }
func (s *stubRankedSource) Permission() Permission { return s.permission }
func (s *stubRankedSource) ListPrefix(_ context.Context, _ Scope, limit int) (SourceResult, error) {
	s.lastLimit = limit
	if s.err != nil {
		return SourceResult{}, s.err
	}
	result := s.result
	if len(result.Items) > limit {
		result.Items = append([]Item(nil), result.Items[:limit]...)
	}
	return result, nil
}

func (s *stubRankedSource) ListWindow(_ context.Context, _ Scope, offset, limit int) (SourceResult, error) {
	s.lastOffset = offset
	s.windowCalls++
	if s.err != nil {
		return SourceResult{}, s.err
	}
	result := SourceResult{Total: s.result.Total, Items: []Item{}}
	if offset >= len(s.result.Items) {
		return result, nil
	}
	end := min(offset+limit, len(s.result.Items))
	result.Items = append(result.Items, s.result.Items[offset:end]...)
	return result, nil
}

type stubPermissionChecker struct {
	allowed map[string]bool
	errors  map[string]error
}

func (s stubPermissionChecker) CheckPermission(_ context.Context, _ uuid.UUID, resource, action string) (bool, error) {
	key := resource + ":" + action
	if err := s.errors[key]; err != nil {
		return false, err
	}
	return s.allowed[key], nil
}

func itemAt(id string, kind Kind, severity string, at time.Time) Item {
	item := Item{ID: id, SourceID: id, Kind: kind, Severity: severity}
	if kind == KindActiveAlarm {
		item.OccurredAt = &at
	} else {
		item.CreatedAt = &at
	}
	return item
}

func TestService_Get_RanksAndTruncatesEachSection(t *testing.T) {
	base := time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)
	abnormal := &stubRankedSource{name: "alarms", result: SourceResult{Total: 3, Items: []Item{
		itemAt("critical-old", KindActiveAlarm, "critical", base),
		itemAt("critical-new", KindActiveAlarm, "critical", base.Add(time.Minute)),
		itemAt("major", KindActiveAlarm, "major", base.Add(2*time.Minute)),
	}}}
	candidateTodo := &stubRankedSource{name: "candidate", result: SourceResult{Total: 1, Items: []Item{
		itemAt("candidate", KindDeviceAccessReview, "", base),
	}}}
	svc := NewService([]RankedSource{abnormal}, []RankedSource{candidateTodo}, nil, zap.NewNop())
	generatedAt := base.Add(time.Hour)
	svc.now = func() time.Time { return generatedAt }

	result, err := svc.Get(context.Background(), Scope{IsSuperAdmin: true}, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Abnormalities.Total)
	assert.Equal(t, []string{"critical-new", "critical-old"}, []string{
		result.Abnormalities.Items[0].ID, result.Abnormalities.Items[1].ID,
	})
	assert.Equal(t, int64(1), result.Todos.Total)
	assert.Equal(t, []string{"candidate"}, []string{result.Todos.Items[0].ID})
	assert.Equal(t, generatedAt, result.GeneratedAt)
	assert.Equal(t, 2, abnormal.lastLimit)
	assert.Equal(t, 2, candidateTodo.lastLimit)
}

func TestService_Get_PermissionDeniedIsExcludedAndPermissionFailureIsPartial(t *testing.T) {
	allowedPermission := Permission{Resource: "allowed", Action: "GET"}
	deniedPermission := Permission{Resource: "denied", Action: "GET"}
	brokenPermission := Permission{Resource: "broken", Action: "GET"}
	allowed := &stubRankedSource{name: "allowed", permission: allowedPermission, result: SourceResult{Total: 1, Items: []Item{{ID: "visible", SourceID: "visible"}}}}
	denied := &stubRankedSource{name: "denied", permission: deniedPermission, result: SourceResult{Total: 99, Items: []Item{{ID: "secret"}}}}
	broken := &stubRankedSource{name: "broken", permission: brokenPermission}
	svc := NewService([]RankedSource{allowed, denied, broken}, nil, stubPermissionChecker{
		allowed: map[string]bool{"allowed:GET": true},
		errors:  map[string]error{"broken:GET": errors.New("rbac unavailable")},
	}, zap.NewNop())

	result, err := svc.Get(context.Background(), Scope{UserID: uuid.New()}, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, SectionPartial, result.Abnormalities.Status)
	assert.Equal(t, int64(1), result.Abnormalities.Total)
	require.Len(t, result.Abnormalities.Items, 1)
	assert.Equal(t, "visible", result.Abnormalities.Items[0].ID)
	assert.Equal(t, 0, denied.lastLimit, "a denied source must never be queried")
}

func TestService_Get_AllEligibleSourcesFailReturnsErrorSection(t *testing.T) {
	source := &stubRankedSource{name: "broken", err: errors.New("database unavailable")}
	svc := NewService([]RankedSource{source}, nil, nil, zap.NewNop())

	result, err := svc.Get(context.Background(), Scope{IsSuperAdmin: true}, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, SectionError, result.Abnormalities.Status)
	assert.Zero(t, result.Abnormalities.Total)
	assert.Empty(t, result.Abnormalities.Items)
}

func TestService_GetPageUsesBoundedPrefixAndStableOrdering(t *testing.T) {
	base := time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)
	left := &stubRankedSource{name: "left", rank: 400, result: SourceResult{Total: 3, Items: []Item{
		itemAt("a", KindActiveAlarm, "critical", base.Add(3*time.Minute)),
		itemAt("b", KindActiveAlarm, "critical", base.Add(2*time.Minute)),
		itemAt("c", KindActiveAlarm, "critical", base.Add(time.Minute)),
	}}}
	right := &stubRankedSource{name: "right", rank: 300, result: SourceResult{Total: 2, Items: []Item{
		itemAt("d", KindActiveAlarm, "major", base.Add(6*time.Minute)),
		itemAt("e", KindActiveAlarm, "major", base.Add(5*time.Minute)),
	}}}
	svc := NewService([]RankedSource{left, right}, nil, nil, zap.NewNop())

	page, err := svc.GetPage(context.Background(), Scope{IsSuperAdmin: true}, SectionAbnormalities, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), page.Total)
	assert.Equal(t, []string{"c", "d"}, []string{page.Items[0].ID, page.Items[1].ID})
	assert.Equal(t, 2, left.lastLimit)
	assert.Equal(t, 2, right.lastLimit)
}

func TestService_GetPageDeepPageDoesNotLoadTheWholePrefix(t *testing.T) {
	items := make([]Item, 250)
	for index := range items {
		items[index] = Item{ID: fmt.Sprintf("item-%03d", index), SourceID: fmt.Sprintf("item-%03d", index)}
	}
	source := &stubRankedSource{name: "deep", rank: 500, result: SourceResult{Total: 250, Items: items}}
	svc := NewService(nil, []RankedSource{source}, nil, zap.NewNop())

	page, err := svc.GetPage(context.Background(), Scope{IsSuperAdmin: true}, SectionTodos, 100, 2)

	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	assert.Equal(t, []string{"item-198", "item-199"}, []string{page.Items[0].ID, page.Items[1].ID})
	assert.Equal(t, 2, source.lastLimit, "count discovery must stay bounded by page_size")
	assert.Equal(t, 198, source.lastOffset)
	assert.Equal(t, 1, source.windowCalls)
}

func TestService_RejectsInvalidParameters(t *testing.T) {
	svc := NewService(nil, nil, nil, zap.NewNop())
	_, err := svc.Get(context.Background(), Scope{}, 0, 2)
	assert.Error(t, err)
	_, err = svc.GetPage(context.Background(), Scope{}, Section("unknown"), 1, 20)
	assert.Error(t, err)
	_, err = svc.GetPage(context.Background(), Scope{}, SectionTodos, 1, 51)
	assert.Error(t, err)
}
