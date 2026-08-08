package storageprotection

import (
	"context"
	"time"
)

type Repository interface {
	GetEnabledPolicy(ctx context.Context, targetType TargetType, targetID string, scope WriteScope) (*Policy, error)
	List(ctx context.Context) ([]Policy, error)
	Save(ctx context.Context, policy *Policy) (*Policy, error)
	UpdateState(ctx context.Context, policy *Policy) error
	RecordEvent(ctx context.Context, event Event) error
	ListEvents(ctx context.Context, targetType TargetType, targetID string, limit int) ([]Event, error)
	CleanupEvents(ctx context.Context, before time.Time, keepLatest int) (int64, error)
}

type UsageProvider interface {
	Snapshot(ctx context.Context, targetType TargetType, targetID string) (UsageSnapshot, error)
}

type UsageTargetLister interface {
	ListTargets(ctx context.Context) ([]UsageSnapshot, error)
}

type WriteAdmission interface {
	Check(ctx context.Context, targetType TargetType, targetID string, scope WriteScope) (AdmissionDecision, error)
}
