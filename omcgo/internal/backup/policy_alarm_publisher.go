package backup

import (
	"context"
	"errors"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/event"
)

// FailureAlarmPayload is the event payload published on backup task failure
// when the active BackupPolicy has alert_on_failure=true. The alarm engine
// (F04) subscribes to event.SubjectAlarmRaised and routes by `source` /
// `identifier` to the appropriate notification channel (T-0007 EmailDispatcher
// for `alert_email`).
type FailureAlarmPayload struct {
	Source       string `json:"source"`        // always "backup"
	Severity     string `json:"severity"`      // "major" — backup data loss
	Identifier   string `json:"identifier"`    // "backup_task_failed"
	Summary      string `json:"summary"`
	TaskID       string `json:"task_id"`
	TargetCount  int    `json:"target_count"`
	ErrorMessage string `json:"error_message,omitempty"`
	AlertEmail   string `json:"alert_email,omitempty"` // routing hint for alarm engine
}

// PublishFailureAlarm publishes alarm.raised when policy.AlertOnFailure=true.
// On opt-out it short-circuits (still records the metric outcome).
//
// Errors are returned but the executor should not abort on alarm failure —
// the backup task itself has already failed; alarm publish is best-effort
// auxiliary signaling.
func PublishFailureAlarm(
	ctx context.Context,
	policyService PolicyGetter,
	bus event.EventBus,
	metrics *PolicyMetrics,
	task *BackupTask,
) error {
	// Defensive guards (review fix HIGH-1): callers from contexts other than
	// the executor failure path may legitimately pass nil deps.
	if policyService == nil {
		return fmt.Errorf("nil policyService: %w", errors.New("invalid input"))
	}
	if bus == nil {
		return fmt.Errorf("nil event bus: %w", errors.New("invalid input"))
	}
	if task == nil {
		return fmt.Errorf("nil task: %w", errors.New("invalid input"))
	}

	policy, err := policyService.Get(ctx)
	if err != nil {
		metrics.RecordFailureAlarm("error")
		return fmt.Errorf("get backup policy: %w", err)
	}
	if !policy.AlertOnFailure {
		metrics.RecordFailureAlarm("skipped")
		return nil
	}

	errMsg := ""
	if task.ErrorMessage != nil {
		errMsg = *task.ErrorMessage
	}
	payload := FailureAlarmPayload{
		Source: "backup",
		// TODO(T-0076): policy-driven severity (warning/major/critical). Phase 1
		// hard-codes "major" because backup failure usually warrants ops attention;
		// future PR maps policy.alert_threshold_percent or a new severity field.
		Severity:     "major",
		Identifier:   "backup_task_failed",
		Summary:      fmt.Sprintf("Backup task failed for %d target(s)", len(task.TargetIDs)),
		TaskID:       task.ID.String(),
		TargetCount:  len(task.TargetIDs),
		ErrorMessage: errMsg,
		AlertEmail:   policy.AlertEmail,
	}

	evt, err := event.NewEvent(event.SubjectAlarmRaised, payload)
	if err != nil {
		metrics.RecordFailureAlarm("error")
		return fmt.Errorf("build alarm event: %w", err)
	}
	if err := bus.Publish(ctx, event.SubjectAlarmRaised, evt); err != nil {
		metrics.RecordFailureAlarm("error")
		return fmt.Errorf("publish alarm.raised: %w", err)
	}
	metrics.RecordFailureAlarm("published")
	return nil
}

// PolicyGetter is the narrow contract PublishFailureAlarm consumes. *PolicyService
// satisfies it (declared at use-site to keep the dependency direction one-way).
type PolicyGetter interface {
	Get(ctx context.Context) (*BackupPolicy, error)
}
