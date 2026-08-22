package paramsync

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
)

// TaskTerminalBridge converts every terminal param_sync task state, including
// sweeper-driven expired/cancelled states, into one canonical lightweight result.
type TaskTerminalBridge struct {
	bus  event.EventBus
	subs []event.Subscription
}

func NewTaskTerminalBridge(bus event.EventBus) *TaskTerminalBridge {
	return &TaskTerminalBridge{bus: bus}
}

func (b *TaskTerminalBridge) Start() error {
	for _, subject := range []string{event.SubjectTaskCompleted, event.SubjectTaskFailed, event.SubjectTaskCancelled} {
		queue := "param-sync-terminal-" + strings.ReplaceAll(subject, ".", "-")
		sub, err := b.bus.PullSubscribe(subject, queue, b.Handle)
		if err != nil {
			return fmt.Errorf("subscribe parameter sync terminal task %s: %w", subject, err)
		}
		b.subs = append(b.subs, sub)
	}
	return nil
}

func (b *TaskTerminalBridge) Handle(ctx context.Context, evt event.Event) error {
	var terminal task.Task
	if err := evt.DecodePayload(&terminal); err != nil {
		return fmt.Errorf("decode terminal task for parameter sync: %w", err)
	}
	if terminal.Source != task.TaskSourceParamSync {
		return nil
	}
	runID, err := uuid.Parse(terminal.SourceID)
	if err != nil {
		return fmt.Errorf("parameter sync terminal task has invalid run id: %w", err)
	}
	requestID, err := uuid.Parse(terminal.CreatorID)
	if err != nil {
		return fmt.Errorf("parameter sync terminal task has invalid request id: %w", err)
	}
	payload := event.ParamSyncTaskResultPayload{
		EventID: terminal.ID + ":" + string(terminal.Status), RequestID: requestID, RunID: runID,
		TaskID: terminal.ID, DeviceSN: terminal.DeviceSN, Success: terminal.Status == task.TaskStatusCompleted,
		ResultRef: "device_tasks:" + terminal.ID, ErrorCode: canonicalTaskErrorCode(terminal.ErrorCode), ErrorMessage: terminal.ErrorMessage,
	}
	canonical, err := event.NewEvent(event.SubjectParamSyncTaskResult, payload)
	if err != nil {
		return fmt.Errorf("create canonical parameter sync result: %w", err)
	}
	if err := b.bus.Publish(ctx, event.SubjectParamSyncTaskResult, canonical); err != nil {
		return fmt.Errorf("publish canonical parameter sync result: %w", err)
	}
	return nil
}

func canonicalTaskErrorCode(code int) string {
	if code == 0 {
		return ""
	}
	return strconv.Itoa(code)
}

func (b *TaskTerminalBridge) Stop() error {
	var first error
	for _, sub := range b.subs {
		if err := sub.Unsubscribe(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
