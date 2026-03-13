package template

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// ConfigTemplateService provides business logic for configuration templates.
type ConfigTemplateService struct {
	repo   ConfigTemplateRepository
	logger *zap.Logger
}

// NewConfigTemplateService creates a new ConfigTemplateService.
func NewConfigTemplateService(repo ConfigTemplateRepository, logger *zap.Logger) *ConfigTemplateService {
	return &ConfigTemplateService{
		repo:   repo,
		logger: logger,
	}
}

// Match finds the best-matching active provisioning template for a device.
// Priority: carrier + tech + product_class > carrier + tech (no product_class).
// Returns nil, nil if no matching template is found.
func (s *ConfigTemplateService) Match(ctx context.Context, device *model.Device) (*ConfigTemplate, error) {
	if device == nil {
		return nil, fmt.Errorf("match template: device is nil")
	}

	t, err := s.repo.FindBestMatch(ctx, device.Carrier, device.Technology, device.ProductClass, TemplateProvisioning)
	if err != nil {
		return nil, fmt.Errorf("match template for device %s: %w", device.SerialNumber, err)
	}

	if t != nil {
		s.logger.Debug("template matched for device",
			zap.String("device_sn", device.SerialNumber),
			zap.String("template_id", t.ID.String()),
			zap.String("template_name", t.Name),
		)
	}

	return t, nil
}

// ListTemplates lists templates with filtering and pagination.
func (s *ConfigTemplateService) ListTemplates(ctx context.Context, filter ConfigTemplateFilter) (*model.ListResponse[ConfigTemplate], error) {
	return s.repo.List(ctx, filter)
}
