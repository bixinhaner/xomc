package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// TemplateService is the business-logic layer for notification templates.
// It validates input, normalizes defaults, and delegates persistence to the
// repository. Higher-level dispatchers (alarm pipeline, scheduled jobs, etc.)
// can resolve templates by ID or by name and render them at send time.
type TemplateService struct {
	repo   TemplateRepository
	logger *zap.Logger
}

// NewTemplateService creates a TemplateService.
func NewTemplateService(repo TemplateRepository, logger *zap.Logger) *TemplateService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TemplateService{
		repo:   repo,
		logger: logger.Named("notification-template"),
	}
}

// List returns templates matching the filter.
func (s *TemplateService) List(ctx context.Context, filter NotificationTemplateFilter) (*model.ListResponse[NotificationTemplate], error) {
	return s.repo.List(ctx, filter)
}

// GetByID returns a template by UUID.
func (s *TemplateService) GetByID(ctx context.Context, id uuid.UUID) (*NotificationTemplate, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByName returns a template by its unique name. Useful for dispatchers that
// reference templates by symbolic name (e.g. "alarm-critical-zh").
func (s *TemplateService) GetByName(ctx context.Context, name string) (*NotificationTemplate, error) {
	if name == "" {
		return nil, commonerrors.ErrInvalidInput
	}
	return s.repo.GetByName(ctx, name)
}

// Create validates the request, applies defaults, and persists a new template.
func (s *TemplateService) Create(ctx context.Context, req CreateTemplateRequest) (*NotificationTemplate, error) {
	if err := validateTemplateChannel(req.Channel); err != nil {
		return nil, err
	}

	tpl := &NotificationTemplate{
		ID:        uuid.New(),
		Name:      strings.TrimSpace(req.Name),
		Channel:   req.Channel,
		Language:  defaultLanguage(req.Language),
		Subject:   req.Subject,
		Body:      req.Body,
		Variables: normalizeVariables(req.Variables),
		Enabled:   true,
	}
	if req.Enabled != nil {
		tpl.Enabled = *req.Enabled
	}

	if tpl.Name == "" {
		return nil, commonerrors.ErrInvalidInput
	}

	if err := s.repo.Create(ctx, tpl); err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	return tpl, nil
}

// Update applies a partial update to an existing template.
func (s *TemplateService) Update(ctx context.Context, id uuid.UUID, req UpdateTemplateRequest) (*NotificationTemplate, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return nil, commonerrors.ErrInvalidInput
		}
		tpl.Name = trimmed
	}
	if req.Channel != nil {
		if err := validateTemplateChannel(*req.Channel); err != nil {
			return nil, err
		}
		tpl.Channel = *req.Channel
	}
	if req.Language != nil {
		tpl.Language = defaultLanguage(*req.Language)
	}
	if req.Subject != nil {
		tpl.Subject = *req.Subject
	}
	if req.Body != nil {
		tpl.Body = *req.Body
	}
	if req.Variables != nil {
		tpl.Variables = normalizeVariables(req.Variables)
	}
	if req.Enabled != nil {
		tpl.Enabled = *req.Enabled
	}

	if err := s.repo.Update(ctx, tpl); err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}
	return tpl, nil
}

// Delete removes a template by UUID.
func (s *TemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// validateTemplateChannel rejects channels not in the supported allow-list. The
// gin binding validator catches this on the request side; the service-side check
// guards against direct callers (jobs, tests).
func validateTemplateChannel(channel string) error {
	switch channel {
	case TemplateChannelEmail, TemplateChannelSMS, TemplateChannelWebhook:
		return nil
	default:
		return fmt.Errorf("%w: unsupported channel %q", commonerrors.ErrInvalidInput, channel)
	}
}

// defaultLanguage returns zh-CN when the supplied language is blank.
func defaultLanguage(language string) string {
	if strings.TrimSpace(language) == "" {
		return TemplateLanguageZhCN
	}
	return language
}

// normalizeVariables ensures the slice is non-nil and that each entry is trimmed
// and unique. Order of first occurrence is preserved.
func normalizeVariables(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
