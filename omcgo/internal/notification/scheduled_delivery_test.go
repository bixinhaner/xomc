package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type scheduledDeliveryRepositoryStub struct {
	deliveryID uuid.UUID
	schedule   DomainSchedule
	workerID   string
	now        time.Time
	err        error
}

func (s *scheduledDeliveryRepositoryStub) CreateScheduledDelivery(
	_ context.Context,
	schedule DomainSchedule,
	workerID string,
	now time.Time,
) (uuid.UUID, error) {
	s.schedule, s.workerID, s.now = schedule, workerID, now
	return s.deliveryID, s.err
}

func TestDeliveryFromSchedule_PreservesImmutableSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 5, 13, 0, 0, 0, time.UTC)
	schedule := DomainSchedule{
		ID: uuid.New(), EventID: uuid.New(), OccurrenceID: uuid.New(), RuleVersionID: uuid.New(),
		TemplateVersionID: uuid.New(), ChannelConfigID: uuid.New(), Channel: "email",
		DispatchKind: DispatchKindRepeat, SequenceNo: 3, RecipientType: RecipientTargetUser,
		AddressCiphertext: []byte("ciphertext"), AddressKeyVersion: 2,
		RecipientFingerprint: []byte("fingerprint"), Generation: 4, CreatedEventVersion: 7,
	}

	delivery := deliveryFromSchedule(schedule, now)

	require.Equal(t, schedule.ID, delivery.ID)
	require.Equal(t, schedule.EventID, delivery.EventID)
	require.Equal(t, schedule.OccurrenceID, delivery.OccurrenceID)
	require.Equal(t, schedule.RuleVersionID, delivery.RuleVersionID)
	require.Equal(t, schedule.TemplateVersionID, delivery.TemplateVersionID)
	require.Equal(t, schedule.ChannelConfigID, delivery.ChannelConfigID)
	require.Equal(t, schedule.Channel, delivery.Channel)
	require.Equal(t, schedule.DispatchKind, delivery.DispatchKind)
	require.Equal(t, schedule.SequenceNo, delivery.SequenceNo)
	require.Equal(t, schedule.RecipientType, delivery.RecipientType)
	require.Equal(t, schedule.AddressCiphertext, delivery.AddressCiphertext)
	require.Equal(t, schedule.AddressKeyVersion, delivery.AddressKeyVersion)
	require.Equal(t, schedule.RecipientFingerprint, delivery.RecipientFingerprint)
	require.Equal(t, "queued", delivery.FlowState)
	require.Equal(t, "none", delivery.DeliveryResult)
	require.Equal(t, now, delivery.AvailableAt)
	require.Equal(t, now, delivery.NextAttemptAt)
	require.Equal(t, schedule.CreatedEventVersion, delivery.OccurrenceVersion)
	require.Equal(t, schedule.Generation, delivery.ScheduleGeneration)
}

func TestScheduledDeliveryHandler_UsesClaimOwnerAndWrapsFailure(t *testing.T) {
	now := time.Date(2026, 8, 5, 13, 10, 0, 0, time.UTC)
	schedule := DomainSchedule{ID: uuid.New()}
	wantErr := errors.New("database unavailable")
	repository := &scheduledDeliveryRepositoryStub{err: wantErr}
	handler := NewScheduledDeliveryHandler(repository, "scheduler-a")
	handler.now = func() time.Time { return now }

	err := handler.Handle(context.Background(), schedule)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, schedule, repository.schedule)
	require.Equal(t, "scheduler-a", repository.workerID)
	require.Equal(t, now, repository.now)
}
