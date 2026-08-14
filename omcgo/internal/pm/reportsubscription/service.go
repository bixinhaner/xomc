package reportsubscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/pm/querytemplate"
)

type Service struct {
	repo      Repository
	templates querytemplate.Repository
	timezone  TimezoneProvider
	now       func() time.Time
}

type TimezoneProvider interface {
	Location(ctx context.Context) *time.Location
}

func NewService(repo Repository, templates querytemplate.Repository, timezone TimezoneProvider) *Service {
	return &Service{repo: repo, templates: templates, timezone: timezone, now: time.Now}
}

func (s *Service) Get(ctx context.Context, templateID, callerID uuid.UUID, superAdmin bool) (*SubscriptionView, error) {
	template, err := s.authorize(ctx, templateID, callerID, superAdmin, false)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.GetByTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	return toSubscriptionView(item, canManageReport(template, callerID, superAdmin)), nil
}

func (s *Service) Upsert(ctx context.Context, templateID, callerID uuid.UUID, superAdmin bool, input UpsertInput) (*Subscription, error) {
	if _, err := s.authorize(ctx, templateID, callerID, superAdmin, true); err != nil {
		return nil, err
	}
	if !ValidPeriod(input.Period) {
		return nil, fmt.Errorf("%w: invalid period", ErrInvalid)
	}
	sendTimes, err := NormalizeSendTimes(input.SendTimes)
	if err != nil {
		return nil, err
	}
	recipients, err := notification.NormalizeRecipients(input.Recipients)
	if err != nil || len(recipients) > 50 {
		return nil, fmt.Errorf("%w: invalid recipients", ErrInvalid)
	}
	location := s.location(ctx)
	nextRun, err := NextRunAt(s.now(), sendTimes, location)
	if err != nil {
		return nil, err
	}
	input.SendTimes = sendTimes
	input.Recipients = recipients
	input.TimezoneName = location.String()
	input.CreatedBy = callerID
	if input.Enabled {
		input.NextRunAt = &nextRun
	} else {
		input.NextRunAt = nil
	}
	return s.repo.Upsert(ctx, templateID, input)
}

func (s *Service) location(ctx context.Context) *time.Location {
	if s.timezone == nil {
		return time.UTC
	}
	location := s.timezone.Location(ctx)
	if location == nil {
		return time.UTC
	}
	return location
}

func (s *Service) Delete(ctx context.Context, templateID, callerID uuid.UUID, superAdmin bool) error {
	if _, err := s.authorize(ctx, templateID, callerID, superAdmin, true); err != nil {
		return err
	}
	return s.repo.Delete(ctx, templateID)
}

func (s *Service) ListRuns(ctx context.Context, templateID, callerID uuid.UUID, superAdmin bool, limit int) ([]RunView, error) {
	template, err := s.authorize(ctx, templateID, callerID, superAdmin, false)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListRuns(ctx, templateID, limit)
	if err != nil {
		return nil, err
	}
	includeSensitive := canManageReport(template, callerID, superAdmin)
	views := make([]RunView, 0, len(items))
	for i := range items {
		views = append(views, toRunView(&items[i], includeSensitive))
	}
	return views, nil
}

func toSubscriptionView(item *Subscription, includeSensitive bool) *SubscriptionView {
	view := &SubscriptionView{
		ID: item.ID, QueryTemplateID: item.QueryTemplateID, Enabled: item.Enabled,
		Period: item.Period, SendTimes: item.SendTimes, TimezoneName: item.TimezoneName,
		NextRunAt: item.NextRunAt, LastRunAt: item.LastRunAt, LastStatus: item.LastStatus,
	}
	if includeSensitive {
		view.Recipients = item.Recipients
		view.LastError = item.LastError
	}
	return view
}

func toRunView(item *Run, includeSensitive bool) RunView {
	view := RunView{
		ID: item.ID, QueryTemplateID: item.QueryTemplateID, QueryTemplateName: item.QueryTemplateName,
		Period: item.Period, WindowStart: item.WindowStart, WindowEnd: item.WindowEnd,
		Status: item.Status, AttachmentName: item.AttachmentName, StartedAt: item.StartedAt,
		FinishedAt: item.FinishedAt, CreatedAt: item.CreatedAt,
	}
	if includeSensitive {
		view.Recipients = item.Recipients
		view.ErrorMessage = item.ErrorMessage
	}
	return view
}

func (s *Service) authorize(ctx context.Context, templateID, callerID uuid.UUID, superAdmin, write bool) (*querytemplate.Template, error) {
	template, err := s.templates.Get(ctx, templateID)
	if err != nil {
		if errors.Is(err, querytemplate.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("load query template: %w", err)
	}
	allowed := querytemplate.CanRead(template, callerID, superAdmin)
	if write {
		allowed = canManageReport(template, callerID, superAdmin)
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return template, nil
}

func canManageReport(template *querytemplate.Template, callerID uuid.UUID, superAdmin bool) bool {
	if superAdmin {
		return true
	}
	return template.Visibility == querytemplate.VisibilityPrivate && template.CreatorID == callerID
}
