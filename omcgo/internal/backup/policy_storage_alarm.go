package backup

import (
	"context"
	"errors"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/event"
)

// StorageThresholdAlarmPayload mirrors FailureAlarmPayload schema for
// consistency with the F04 alarm engine consumers (route by source/identifier).
// Same payload shape is used for both raised and cleared transitions; the
// EventBus subject distinguishes the two (alarm.raised vs alarm.cleared).
type StorageThresholdAlarmPayload struct {
	Source           string `json:"source"`     // always "backup"
	Severity         string `json:"severity"`   // "major" — disk fill = data-loss precursor
	Identifier       string `json:"identifier"` // "backup_storage_threshold_exceeded"
	Summary          string `json:"summary"`
	BucketName       string `json:"bucket_name"` // "config-backup"
	UsedBytes        int64  `json:"used_bytes"`
	CapacityBytes    int64  `json:"capacity_bytes"`
	UsagePercent     int    `json:"usage_percent"`     // 0..N (may exceed 100 when over capacity)
	ThresholdPercent int    `json:"threshold_percent"` // policy.AlertThresholdPercent
	AlertEmail       string `json:"alert_email,omitempty"`
}

// StorageInfo is the input bundle PolicyMonitor passes to
// PublishStorageThresholdAlarm — keeps the function signature small.
type StorageInfo struct {
	BucketName       string
	UsedBytes        int64
	CapacityBytes    int64
	UsagePercent     int
	ThresholdPercent int
	// Severity is the policy-driven alarm severity (T-0084). Caller
	// (PolicyMonitor) injects from policy.AlertSeverity. Empty string
	// falls back to alarmSeverityMajor via severityOrDefault.
	Severity   string
	AlertEmail string
}

// StorageTransition encodes the edge-trigger state change PolicyMonitor
// detected (above→below or below→above). The publisher routes to the
// matching EventBus subject and metric label.
type StorageTransition int

const (
	// TransitionRaised is below→above: bucket usage just crossed the threshold.
	TransitionRaised StorageTransition = iota
	// TransitionCleared is above→below: bucket usage just dropped below the
	// threshold. Publishes alarm.cleared so the F04 engine can close the open
	// alarm by identifier.
	TransitionCleared
)

// storageThresholdIdentifier is the canonical alarm identifier used for
// disk threshold alarms. F04 alarm engine deduplicates raise/clear pairs by
// (source, identifier) — picking a stable string here is essential.
const storageThresholdIdentifier = "backup_storage_threshold_exceeded"

// PublishStorageThresholdAlarm publishes alarm.raised or alarm.cleared based
// on the supplied transition. Caller (PolicyMonitor) is responsible for
// detecting the edge in its in-memory state; this function only routes.
//
// Severity is hardcoded to "major" — policy-driven severity (warning/major/
// critical) is the T-0084 followup; T-0082 deliberately punts schema work.
func PublishStorageThresholdAlarm(
	ctx context.Context,
	bus event.EventBus,
	metrics *PolicyMetrics,
	transition StorageTransition,
	info StorageInfo,
) error {
	if bus == nil {
		return fmt.Errorf("nil event bus: %w", errors.New("invalid input"))
	}

	subject, kind, summary := storageAlarmRoute(transition, info)
	payload := StorageThresholdAlarmPayload{
		Source:           alarmSourceBackup,
		Severity:         severityOrDefault(info.Severity),
		Identifier:       storageThresholdIdentifier,
		Summary:          summary,
		BucketName:       info.BucketName,
		UsedBytes:        info.UsedBytes,
		CapacityBytes:    info.CapacityBytes,
		UsagePercent:     info.UsagePercent,
		ThresholdPercent: info.ThresholdPercent,
		AlertEmail:       info.AlertEmail,
	}

	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		metrics.RecordStorageThresholdAlarm("error")
		return fmt.Errorf("build storage alarm event: %w", err)
	}
	if err := bus.Publish(ctx, subject, evt); err != nil {
		metrics.RecordStorageThresholdAlarm("error")
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	metrics.RecordStorageThresholdAlarm(kind)
	return nil
}

// storageAlarmRoute maps a transition to (subject, metric kind, human summary).
// Centralised so the publisher and tests share one source of truth.
func storageAlarmRoute(t StorageTransition, info StorageInfo) (subject, kind, summary string) {
	switch t {
	case TransitionRaised:
		return event.SubjectAlarmRaised, "raised", fmt.Sprintf(
			"Backup storage usage reached %d%% (threshold %d%%, capacity %d GB)",
			info.UsagePercent, info.ThresholdPercent, info.CapacityBytes/(1024*1024*1024),
		)
	case TransitionCleared:
		return event.SubjectAlarmCleared, "cleared", fmt.Sprintf(
			"Backup storage usage dropped to %d%% (below threshold %d%%)",
			info.UsagePercent, info.ThresholdPercent,
		)
	default:
		// Unknown transition is a programmer bug; route to raised with a
		// flagged summary so it's loud in production rather than silent.
		return event.SubjectAlarmRaised, "raised", fmt.Sprintf(
			"Backup storage alarm: unknown transition %d", int(t),
		)
	}
}
