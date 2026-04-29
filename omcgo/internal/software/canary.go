// Package software — Canary upgrade strategy (T-0018 / R-101).
//
// Canary rollout subdivides a batch upgrade into staged percentages
// (default 1% → 10% → 50% → 100%) so operators can validate each stage
// before promoting the next. Failure-rate thresholds gate each stage:
// crossing the threshold auto-pauses the task and emits a metric/log
// (rollback is a separate action — see T-0021).
//
// Persistence: stored as `strategy`, `canary_stages JSONB`, `current_stage`,
// `stage_status`, `stage_history JSONB`, `auto_advance`, `auto_advance_minutes`
// columns on `upgrade_tasks` (migration 000045).
package software

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CanaryStage configures one rollout stage by cumulative percentage and the
// per-stage failure-rate threshold (percent integer).
type CanaryStage struct {
	Percent          int `json:"percent"`           // cumulative target (1..100)
	FailureThreshold int `json:"failure_threshold"` // failure rate ceiling (1..100)
}

// CanaryStrategy is the full per-task strategy descriptor.
// Stored in upgrade_tasks.canary_stages as JSON encoding of just []CanaryStage;
// auto_advance / auto_advance_minutes are stored in dedicated columns.
type CanaryStrategy struct {
	Stages             []CanaryStage `json:"stages"`
	AutoAdvance        bool          `json:"auto_advance"`
	AutoAdvanceMinutes int           `json:"auto_advance_minutes,omitempty"`
}

// DefaultCanaryStages is the recommended default progression: 1% → 10% → 50% → 100%
// with progressively-relaxed failure thresholds (early bugs are most painful).
var DefaultCanaryStages = []CanaryStage{
	{Percent: 1, FailureThreshold: 5},
	{Percent: 10, FailureThreshold: 10},
	{Percent: 50, FailureThreshold: 20},
	{Percent: 100, FailureThreshold: 30},
}

// Canary task constants and stage-status values.
const (
	StrategyFull   = "full"
	StrategyCanary = "canary"

	StageStatusPending   = "pending"
	StageStatusRunning   = "running"
	StageStatusPaused    = "paused"
	StageStatusAborted   = "aborted"
	StageStatusCompleted = "completed"
)

// StageHistoryEntry records the outcome of one stage transition for audit.
type StageHistoryEntry struct {
	Stage         int       `json:"stage"`           // 1-indexed
	Percent       int       `json:"percent"`
	DevicesInStage int      `json:"devices_in_stage"`
	SuccessCount  int       `json:"success_count"`
	FailCount     int       `json:"fail_count"`
	FailureRate   float64   `json:"failure_rate"`    // 0.0–1.0
	Action        string    `json:"action"`          // "advanced" / "paused" / "aborted" / "completed"
	At            time.Time `json:"at"`
	Reason        string    `json:"reason,omitempty"`
}

// ValidateStages checks a canary stage list for monotonicity and basic ranges.
// Returns nil when stages are usable.
func ValidateStages(stages []CanaryStage) error {
	if len(stages) == 0 {
		return fmt.Errorf("canary stages must not be empty")
	}
	last := 0
	for i, s := range stages {
		if s.Percent <= 0 || s.Percent > 100 {
			return fmt.Errorf("stage %d percent out of range [1,100]: %d", i+1, s.Percent)
		}
		if s.Percent <= last {
			return fmt.Errorf("stage %d percent must strictly increase (got %d after %d)", i+1, s.Percent, last)
		}
		if s.FailureThreshold < 1 || s.FailureThreshold > 100 {
			return fmt.Errorf("stage %d failure_threshold out of range [1,100]: %d", i+1, s.FailureThreshold)
		}
		last = s.Percent
	}
	if last != 100 {
		return fmt.Errorf("last stage must reach 100%% (got %d)", last)
	}
	return nil
}

// DevicesForStage returns the count of devices that should be active at the
// end of stage index (0-indexed). Cumulative — e.g. given 100 total devices and
// stages [1,10,50,100], DevicesForStage(0,100,stages) == 1.
func DevicesForStage(stageIdx int, total int, stages []CanaryStage) int {
	if stageIdx < 0 || stageIdx >= len(stages) {
		return 0
	}
	pct := stages[stageIdx].Percent
	count := total * pct / 100
	if count < 1 && pct > 0 && total > 0 {
		count = 1 // ensure even 1% on small batches yields ≥1 device
	}
	if count > total {
		count = total
	}
	return count
}

// FailureRate computes failure rate as a 0.0–1.0 float. Returns 0 when total is 0.
func FailureRate(failCount, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(failCount) / float64(total)
}

// MarshalStages and UnmarshalStages provide JSONB round-trip helpers.
func MarshalStages(stages []CanaryStage) (json.RawMessage, error) {
	b, err := json.Marshal(stages)
	if err != nil {
		return nil, fmt.Errorf("marshal canary stages: %w", err)
	}
	return b, nil
}

// UnmarshalStages parses a JSONB column. nil/empty input returns
// DefaultCanaryStages so callers always get a usable schedule.
func UnmarshalStages(raw json.RawMessage) ([]CanaryStage, error) {
	if len(raw) == 0 {
		return append([]CanaryStage(nil), DefaultCanaryStages...), nil
	}
	var stages []CanaryStage
	if err := json.Unmarshal(raw, &stages); err != nil {
		return nil, fmt.Errorf("unmarshal canary stages: %w", err)
	}
	if err := ValidateStages(stages); err != nil {
		return nil, err
	}
	return stages, nil
}

// CanaryFields captures just the canary-related task state used by service
// methods and the cron monitor — keeps the surface tight for tests.
type CanaryFields struct {
	TaskID             uuid.UUID
	Strategy           string
	Stages             []CanaryStage
	CurrentStage       int
	StageStatus        string
	StageHistory       []StageHistoryEntry
	AutoAdvance        bool
	AutoAdvanceMinutes int
	TotalCount         int
}

// IsCanary reports whether this task is on the canary path.
func (c *CanaryFields) IsCanary() bool {
	return c != nil && c.Strategy == StrategyCanary
}

// CurrentStageDescriptor returns the active stage definition, or nil when out of range.
func (c *CanaryFields) CurrentStageDescriptor() *CanaryStage {
	if c == nil || c.CurrentStage < 1 || c.CurrentStage > len(c.Stages) {
		return nil
	}
	s := c.Stages[c.CurrentStage-1]
	return &s
}
