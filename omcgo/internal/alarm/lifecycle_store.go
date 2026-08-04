package alarm

import (
	"context"
	"errors"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

var ErrAlarmVersionConflict = errors.New("alarm lifecycle version conflict")

// LifecycleStore atomically persists one managed-element alarm transition and
// its canonical Outbox fact in the primary PostgreSQL database.
type LifecycleStore interface {
	PersistRaised(context.Context, *model.Alarm) (event.AlarmLifecyclePayload, error)
	PersistUpdated(context.Context, *model.Alarm, []event.AlarmChangeField) (event.AlarmLifecyclePayload, error)
	PersistAcknowledged(context.Context, *model.Alarm) (event.AlarmLifecyclePayload, error)
	PersistUnacknowledged(context.Context, *model.Alarm) (event.AlarmLifecyclePayload, error)
	PersistCleared(context.Context, *model.Alarm) (event.AlarmLifecyclePayload, error)
}
