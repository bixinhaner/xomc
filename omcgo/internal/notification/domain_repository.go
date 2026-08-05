package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InboxRepository owns only idempotent Inbox insertion and occurrence reads.
// Ordered application is added by the lifecycle consumer in Task 8 so this
// baseline does not pre-encode orchestration policy.
type InboxRepository interface {
	InsertEvent(context.Context, *DomainEvent) (bool, error)
	GetOccurrence(context.Context, uuid.UUID) (*DomainOccurrence, error)
}

type RuleVersionRepository interface {
	GetRuleVersion(context.Context, uuid.UUID) (*DomainRuleVersion, error)
}

type TemplateVersionRepository interface {
	GetTemplateVersion(context.Context, uuid.UUID) (*DomainTemplateVersion, error)
}

type ScheduleRepository interface {
	InsertSchedule(context.Context, *DomainSchedule) (bool, error)
	ClaimDue(context.Context, ScheduleClaimRequest) ([]DomainSchedule, error)
	MarkCompleted(context.Context, uuid.UUID, string, time.Time) error
	MarkCancelled(context.Context, uuid.UUID, string, time.Time) error
	Release(context.Context, uuid.UUID, string, time.Time, time.Time) error
}

type ScheduleClaimRequest struct {
	WorkerID      string
	Now           time.Time
	LeaseDuration time.Duration
	Limit         int
}

type DeliveryRepository interface {
	InsertDelivery(context.Context, *DomainDelivery) (bool, error)
	InsertAttempt(context.Context, *DomainDeliveryAttempt) (bool, error)
}
