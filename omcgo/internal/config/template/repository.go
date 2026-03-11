package template

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// ConfigTemplateRepository defines the persistence interface for configuration templates.
type ConfigTemplateRepository interface {
	Create(ctx context.Context, t *ConfigTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*ConfigTemplate, error)
	Update(ctx context.Context, t *ConfigTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ConfigTemplateFilter) (*model.ListResponse[ConfigTemplate], error)
	FindByCarrierTech(ctx context.Context, carrier model.CarrierCode, tech model.Technology, templateType TemplateType) ([]ConfigTemplate, error)
	FindBestMatch(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, templateType TemplateType) (*ConfigTemplate, error)
}
