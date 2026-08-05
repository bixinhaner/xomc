package notification

import (
	"context"

	"github.com/omcgo/omcgo/internal/core/event"
)

type LifecycleApplyState string

const (
	LifecycleApplied   LifecycleApplyState = "applied"
	LifecycleDuplicate LifecycleApplyState = "duplicate"
	LifecycleIgnored   LifecycleApplyState = "ignored"
	LifecycleWaiting   LifecycleApplyState = "waiting"
)

type LifecycleApplyResult struct {
	State        LifecycleApplyState
	AppliedCount int
}

type LifecycleRepository interface {
	ApplyLifecycle(context.Context, event.AlarmLifecyclePayload) (LifecycleApplyResult, error)
}
