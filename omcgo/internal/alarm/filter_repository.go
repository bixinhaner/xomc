package alarm

import (
	"context"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

type AlarmFilterRuleRepository interface {
	Create(ctx context.Context, rule *AlarmFilterRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*AlarmFilterRule, error)
	Update(ctx context.Context, rule *AlarmFilterRule) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter AlarmFilterRuleFilter) (*model.ListResponse[AlarmFilterRule], error)
	Toggle(ctx context.Context, id uuid.UUID) error
	ListEnabled(ctx context.Context) ([]AlarmFilterRule, error)
}