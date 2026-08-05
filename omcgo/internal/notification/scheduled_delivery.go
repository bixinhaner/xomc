package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ScheduledDeliveryRepository atomically materializes the delivery snapshot
// carried by a claimed schedule and links the two durable records.
type ScheduledDeliveryRepository interface {
	CreateScheduledDelivery(context.Context, DomainSchedule, string, time.Time) (uuid.UUID, error)
}

type ScheduledDeliveryHandler struct {
	repository ScheduledDeliveryRepository
	workerID   string
	now        func() time.Time
}

func NewScheduledDeliveryHandler(repository ScheduledDeliveryRepository, workerID string) *ScheduledDeliveryHandler {
	return &ScheduledDeliveryHandler{
		repository: repository,
		workerID:   workerID,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (h *ScheduledDeliveryHandler) Handle(ctx context.Context, schedule DomainSchedule) error {
	if h == nil || h.repository == nil {
		return fmt.Errorf("create scheduled notification delivery: repository is required")
	}
	if h.workerID == "" {
		return fmt.Errorf("create scheduled notification delivery: worker ID is required")
	}
	if _, err := h.repository.CreateScheduledDelivery(ctx, schedule, h.workerID, h.now()); err != nil {
		return fmt.Errorf("create delivery from notification schedule %s: %w", schedule.ID, err)
	}
	return nil
}

func deliveryFromSchedule(schedule DomainSchedule, now time.Time) DomainDelivery {
	return DomainDelivery{
		ID:                   schedule.ID,
		EventID:              schedule.EventID,
		OccurrenceID:         schedule.OccurrenceID,
		RuleVersionID:        schedule.RuleVersionID,
		TemplateVersionID:    schedule.TemplateVersionID,
		ChannelConfigID:      schedule.ChannelConfigID,
		Channel:              schedule.Channel,
		DispatchKind:         schedule.DispatchKind,
		SequenceNo:           schedule.SequenceNo,
		RecipientType:        schedule.RecipientType,
		AddressCiphertext:    schedule.AddressCiphertext,
		AddressKeyVersion:    schedule.AddressKeyVersion,
		RecipientFingerprint: schedule.RecipientFingerprint,
		FlowState:            "queued",
		DeliveryResult:       "none",
		AvailableAt:          now,
		NextAttemptAt:        now,
		OccurrenceVersion:    schedule.CreatedEventVersion,
		ScheduleGeneration:   schedule.Generation,
	}
}
