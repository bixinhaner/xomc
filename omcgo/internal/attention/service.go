package attention

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PermissionChecker interface {
	CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
}

type Service struct {
	abnormalitySources []RankedSource
	todoSources        []RankedSource
	permissions        PermissionChecker
	logger             *zap.Logger
	now                func() time.Time
}

func NewService(
	abnormalitySources []RankedSource,
	todoSources []RankedSource,
	permissions PermissionChecker,
	logger *zap.Logger,
) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	abnormalitySources = orderedSources(abnormalitySources)
	todoSources = orderedSources(todoSources)
	return &Service{
		abnormalitySources: abnormalitySources,
		todoSources:        todoSources,
		permissions:        permissions,
		logger:             logger.Named("attention"),
		now:                time.Now,
	}
}

func (s *Service) Get(ctx context.Context, scope Scope, abnormalLimit, todoLimit int) (*Response, error) {
	if abnormalLimit < 1 || abnormalLimit > 5 || todoLimit < 1 || todoLimit > 5 {
		return nil, fmt.Errorf("attention summary limit out of range")
	}

	var abnormalities, todos AttentionSection
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		abnormalities = s.loadSection(ctx, scope, s.abnormalitySources, abnormalLimit)
	}()
	go func() {
		defer wg.Done()
		todos = s.loadSection(ctx, scope, s.todoSources, todoLimit)
	}()
	wg.Wait()

	return &Response{
		Abnormalities: abnormalities,
		Todos:         todos,
		GeneratedAt:   s.now(),
	}, nil
}

func (s *Service) GetPage(ctx context.Context, scope Scope, section Section, page, pageSize int) (*Page, error) {
	if page < 1 || pageSize < 1 || pageSize > 50 {
		return nil, fmt.Errorf("attention page parameters out of range")
	}
	if page > int(^uint(0)>>1)/pageSize {
		return nil, fmt.Errorf("attention page offset overflows")
	}

	sources := s.abnormalitySources
	if section == SectionTodos {
		sources = s.todoSources
	} else if section != SectionAbnormalities {
		return nil, fmt.Errorf("unknown attention section %q", section)
	}

	return s.loadPage(ctx, scope, sources, page, pageSize), nil
}

func orderedSources(sources []RankedSource) []RankedSource {
	result := append([]RankedSource(nil), sources...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Rank() != result[j].Rank() {
			return result[i].Rank() > result[j].Rank()
		}
		return result[i].Name() < result[j].Name()
	})
	return result
}

func (s *Service) loadPage(ctx context.Context, scope Scope, sources []RankedSource, page, pageSize int) *Page {
	loads := make([]sourceLoad, len(sources))
	var wg sync.WaitGroup
	for index, source := range sources {
		wg.Add(1)
		go func(index int, source RankedSource) {
			defer wg.Done()
			allowed, err := s.allowed(ctx, scope, source.Permission())
			if err != nil {
				loads[index] = sourceLoad{eligible: true, err: fmt.Errorf("check %s permission: %w", source.Name(), err)}
				return
			}
			if !allowed {
				return
			}
			result, err := source.ListPrefix(ctx, scope, pageSize)
			loads[index] = sourceLoad{eligible: true, result: result, err: err}
		}(index, source)
	}
	wg.Wait()

	eligible, failed := 0, 0
	var total int64
	for index, load := range loads {
		if !load.eligible {
			continue
		}
		eligible++
		if load.err != nil {
			failed++
			s.logger.Error("load attention source count failed", zap.String("source", sources[index].Name()), zap.Error(load.err))
			continue
		}
		total += load.result.Total
	}

	offset := int64((page - 1) * pageSize)
	items := make([]Item, 0, pageSize)
	for index, load := range loads {
		if !load.eligible || load.err != nil || len(items) >= pageSize {
			continue
		}
		if offset >= load.result.Total {
			offset -= load.result.Total
			continue
		}

		remaining := pageSize - len(items)
		available := int(load.result.Total - offset)
		limit := min(remaining, available)
		window := load.result
		if offset > 0 {
			var err error
			window, err = sources[index].ListWindow(ctx, scope, int(offset), limit)
			if err != nil {
				failed++
				total -= load.result.Total
				s.logger.Error("load attention source window failed", zap.String("source", sources[index].Name()), zap.Error(err))
				continue
			}
		}
		if len(window.Items) > limit {
			window.Items = window.Items[:limit]
		}
		items = append(items, window.Items...)
		offset = 0
	}

	items = deduplicate(items)
	if items == nil {
		items = []Item{}
	}
	return &Page{
		Status: sectionStatus(eligible, failed), Total: total, Items: items,
		Page: page, PageSize: pageSize,
	}
}

func sectionStatus(eligible, failed int) SectionStatus {
	if failed > 0 && failed >= eligible {
		return SectionError
	}
	if failed > 0 {
		return SectionPartial
	}
	return SectionOK
}

type sourceLoad struct {
	eligible bool
	result   SourceResult
	err      error
}

func (s *Service) loadSection(ctx context.Context, scope Scope, sources []RankedSource, prefixLimit int) AttentionSection {
	loads := make([]sourceLoad, len(sources))
	var wg sync.WaitGroup
	for index, source := range sources {
		wg.Add(1)
		go func(index int, source RankedSource) {
			defer wg.Done()
			allowed, err := s.allowed(ctx, scope, source.Permission())
			if err != nil {
				loads[index] = sourceLoad{eligible: true, err: fmt.Errorf("check %s permission: %w", source.Name(), err)}
				return
			}
			if !allowed {
				return
			}
			result, err := source.ListPrefix(ctx, scope, prefixLimit)
			loads[index] = sourceLoad{eligible: true, result: result, err: err}
		}(index, source)
	}
	wg.Wait()

	eligible, failed := 0, 0
	var total int64
	items := make([]Item, 0, prefixLimit)
	for index, load := range loads {
		if !load.eligible {
			continue
		}
		eligible++
		if load.err != nil {
			failed++
			s.logger.Error("load attention source failed", zap.String("source", sources[index].Name()), zap.Error(load.err))
			continue
		}
		total += load.result.Total
		items = append(items, load.result.Items...)
	}

	sort.SliceStable(items, func(i, j int) bool { return less(items[i], items[j]) })
	items = deduplicate(items)
	if len(items) > prefixLimit {
		items = items[:prefixLimit]
	}
	if items == nil {
		items = []Item{}
	}

	status := sectionStatus(eligible, failed)
	if status == SectionError {
		total = 0
		items = []Item{}
	}
	return AttentionSection{Status: status, Total: total, Items: items}
}

func (s *Service) allowed(ctx context.Context, scope Scope, permission Permission) (bool, error) {
	if scope.IsSuperAdmin {
		return true, nil
	}
	if s.permissions == nil || scope.UserID == uuid.Nil {
		return false, nil
	}
	return s.permissions.CheckPermission(ctx, scope.UserID, permission.Resource, permission.Action)
}

func less(left, right Item) bool {
	leftRank, rightRank := itemRank(left), itemRank(right)
	if leftRank != rightRank {
		return leftRank > rightRank
	}
	leftTime, rightTime := itemTime(left), itemTime(right)
	if left.Kind == KindActiveAlarm {
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
	} else if !leftTime.Equal(rightTime) {
		return leftTime.Before(rightTime)
	}
	return left.SourceID < right.SourceID
}

func itemRank(item Item) int {
	switch item.Kind {
	case KindActiveAlarm:
		if item.Severity == "critical" {
			return 400
		}
		return 300
	case KindDeviceAccessReview:
		return 200
	default:
		return 0
	}
}

func itemTime(item Item) time.Time {
	if item.OccurredAt != nil {
		return *item.OccurredAt
	}
	if item.CreatedAt != nil {
		return *item.CreatedAt
	}
	return time.Time{}
}

func deduplicate(items []Item) []Item {
	seen := make(map[string]struct{}, len(items))
	result := make([]Item, 0, len(items))
	for _, item := range items {
		if _, exists := seen[item.ID]; exists {
			continue
		}
		seen[item.ID] = struct{}{}
		result = append(result, item)
	}
	return result
}
