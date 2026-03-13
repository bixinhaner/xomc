package alarm

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// AlarmRuleRepository defines the interface for alarm rule persistence.
type AlarmRuleRepository interface {
	Create(ctx context.Context, rule *AlarmRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*AlarmRule, error)
	Update(ctx context.Context, rule *AlarmRule) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter AlarmRuleFilter) (*model.ListResponse[AlarmRule], error)
}
