package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/event"
)

type OrchestrationPendingRepository interface {
	ListPendingAppliedEvents(context.Context, uuid.UUID) ([]event.AlarmLifecyclePayload, error)
	SaveDecision(context.Context, uuid.UUID, OrchestrationDecision) (bool, error)
}

type OrchestrationInputBuilder interface {
	BuildOrchestrationInput(context.Context, event.AlarmLifecyclePayload) (OrchestrationInput, error)
}

type OccurrenceOrchestrator interface {
	ProcessOccurrence(context.Context, uuid.UUID) error
}

type OrchestrationCoordinator struct {
	repository OrchestrationPendingRepository
	builder    OrchestrationInputBuilder
}

func NewOrchestrationCoordinator(repository OrchestrationPendingRepository, builder OrchestrationInputBuilder) *OrchestrationCoordinator {
	return &OrchestrationCoordinator{repository: repository, builder: builder}
}

func (c *OrchestrationCoordinator) ProcessOccurrence(ctx context.Context, occurrenceID uuid.UUID) error {
	if c == nil || c.repository == nil || c.builder == nil {
		return fmt.Errorf("process notification occurrence orchestration: dependencies are required")
	}
	payloads, err := c.repository.ListPendingAppliedEvents(ctx, occurrenceID)
	if err != nil {
		return fmt.Errorf("list pending notification orchestration events: %w", err)
	}
	for _, payload := range payloads {
		input, err := c.builder.BuildOrchestrationInput(ctx, payload)
		if err != nil {
			return fmt.Errorf("build notification orchestration input for event %s: %w", payload.EventID, err)
		}
		decision, err := BuildOrchestrationDecision(input)
		if err != nil {
			return fmt.Errorf("evaluate notification orchestration event %s: %w", payload.EventID, err)
		}
		if _, err := c.repository.SaveDecision(ctx, payload.EventID, decision); err != nil {
			return fmt.Errorf("save notification orchestration event %s: %w", payload.EventID, err)
		}
	}
	return nil
}
