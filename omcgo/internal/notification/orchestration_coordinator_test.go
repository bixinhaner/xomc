package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

type orchestrationPendingStub struct {
	payloads []event.AlarmLifecyclePayload
	saved    []uuid.UUID
}

func (s *orchestrationPendingStub) ListPendingAppliedEvents(context.Context, uuid.UUID) ([]event.AlarmLifecyclePayload, error) {
	return s.payloads, nil
}

func (s *orchestrationPendingStub) SaveDecision(_ context.Context, eventID uuid.UUID, _ OrchestrationDecision) (bool, error) {
	s.saved = append(s.saved, eventID)
	return true, nil
}

type orchestrationInputStub struct{}

func (orchestrationInputStub) BuildOrchestrationInput(_ context.Context, payload event.AlarmLifecyclePayload) (OrchestrationInput, error) {
	return OrchestrationInput{
		Now: payload.OccurredAt, Event: payload,
		Occurrence: DomainOccurrence{OccurrenceID: payload.OccurrenceID, ScheduleGeneration: 1},
	}, nil
}

func TestOrchestrationCoordinator_ProcessesDrainedGapEventsInRepositoryOrder(t *testing.T) {
	now := time.Now().UTC()
	occurrenceID := uuid.New()
	raised := orchestrationCoordinatorPayload(now, occurrenceID, 1, event.AlarmLifecycleRaised)
	updated := orchestrationCoordinatorPayload(now.Add(time.Second), occurrenceID, 2, event.AlarmLifecycleUpdated)
	repository := &orchestrationPendingStub{payloads: []event.AlarmLifecyclePayload{raised, updated}}

	err := NewOrchestrationCoordinator(repository, orchestrationInputStub{}).ProcessOccurrence(context.Background(), occurrenceID)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{raised.EventID, updated.EventID}, repository.saved)
}

func orchestrationCoordinatorPayload(now time.Time, occurrenceID uuid.UUID, version int64, lifecycle event.AlarmLifecycleType) event.AlarmLifecyclePayload {
	return event.AlarmLifecyclePayload{
		SchemaVersion: event.AlarmLifecycleSchemaVersion, EventID: uuid.New(), OccurrenceID: occurrenceID,
		AlarmVersion: version, LifecycleType: lifecycle, OccurredAt: now,
		Snapshot: event.AlarmLifecycleSnapshot{
			AlarmID: occurrenceID, DeviceID: uuid.New(), Severity: model.AlarmMajor,
			Status: model.AlarmActive, RaisedAt: now,
		},
	}
}
