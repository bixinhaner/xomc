package paramsync

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type convergenceDecision struct {
	Ready  bool
	Status RunStatus
}

type convergenceBlockReason string

const (
	convergenceBlockPlanNotDispatched     convergenceBlockReason = "plan_not_dispatched"
	convergenceBlockTerminalResultMissing convergenceBlockReason = "terminal_task_missing_result"
	convergenceBlockDeviceTaskActive      convergenceBlockReason = "device_task_active"
)

type convergenceResult struct {
	Finalized     bool
	Failed        bool
	Drift         bool
	Status        RunStatus
	BlockedReason convergenceBlockReason
}

func classifyRunBlockReason(
	run SyncRun,
	counts authoritativeRunCounts,
) convergenceBlockReason {
	if run.ExpectedTaskCount > counts.expected ||
		(run.ExpectedTaskCount == 0 && counts.expected == 0) {
		return convergenceBlockPlanNotDispatched
	}
	if counts.terminal > counts.processed {
		return convergenceBlockTerminalResultMissing
	}
	if counts.terminal < counts.expected {
		return convergenceBlockDeviceTaskActive
	}
	return ""
}

func decideRunConvergence(run SyncRun) convergenceDecision {
	undispatched := run.ExpectedTaskCount == 0 && run.Status != RunStatusCancelling
	if run.Status.Terminal() || undispatched || !run.ReadyToFinalize() {
		return convergenceDecision{}
	}
	if run.Status == RunStatusCancelling || run.FailedTaskCount > 0 {
		return convergenceDecision{Ready: true, Status: RunStatusFailed}
	}
	return convergenceDecision{Ready: true, Status: RunStatusSucceeded}
}

func convergeRunTx(
	ctx context.Context,
	tx pgx.Tx,
	runID uuid.UUID,
	now time.Time,
) (convergenceResult, error) {
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return convergenceResult{}, err
	}
	counts, err := loadAuthoritativeRunCounts(ctx, tx, runID)
	if err != nil {
		return convergenceResult{}, err
	}
	drift := run.ExpectedTaskCount != counts.expected ||
		run.TerminalTaskCount != counts.terminal ||
		run.ProcessedTaskCount != counts.processed ||
		run.FailedTaskCount != counts.failed
	blockedReason := classifyRunBlockReason(*run, counts)
	applyAuthoritativeRunCounts(run, counts)
	if blockedReason == convergenceBlockPlanNotDispatched &&
		!run.Status.Terminal() && run.Status != RunStatusPlanning && run.Status != RunStatusEnqueuing {
		result, convergeErr := failIncompleteDispatchTx(ctx, tx, run, counts, now)
		result.Drift = drift
		result.BlockedReason = blockedReason
		return result, convergeErr
	}
	result, err := convergeLoadedRunTx(ctx, tx, run, run.Status, drift, now)
	result.Drift = drift
	result.BlockedReason = blockedReason
	return result, err
}

func failIncompleteDispatchTx(
	ctx context.Context,
	tx pgx.Tx,
	run *SyncRun,
	counts authoritativeRunCounts,
	now time.Time,
) (convergenceResult, error) {
	const message = "parameter sync task plan was only partially dispatched"
	if run.Status != RunStatusCancelling {
		if err := beginCancellingRun(ctx, tx, run, message, now); err != nil {
			return convergenceResult{}, err
		}
	}
	if err := cancelUnsentRunTasks(ctx, tx, run, now); err != nil {
		return convergenceResult{}, err
	}
	var err error
	counts, err = loadAuthoritativeRunCounts(ctx, tx, run.ID)
	if err != nil {
		return convergenceResult{}, err
	}
	applyAuthoritativeRunCounts(run, counts)
	// The planned cardinality remains larger than durable task rows by design.
	// Once every row that actually exists has a durable terminal result, fail
	// the run with truthful planned/actual counters instead of waiting forever
	// for task rows that were never created.
	if counts.terminal == counts.expected && counts.processed == counts.expected {
		if err := finalizeConvergedFailedRun(ctx, tx, run, now); err != nil {
			return convergenceResult{}, err
		}
		return convergenceResult{Finalized: true, Failed: true, Status: RunStatusFailed}, nil
	}
	if err := updateRunProgress(ctx, tx, run, RunStatusCancelling); err != nil {
		return convergenceResult{}, err
	}
	return convergenceResult{Failed: true, Status: RunStatusCancelling}, nil
}

func convergeLoadedRunTx(
	ctx context.Context,
	tx pgx.Tx,
	run *SyncRun,
	progressStatus RunStatus,
	persistProgress bool,
	now time.Time,
) (convergenceResult, error) {
	decision := decideRunConvergence(*run)
	if !decision.Ready {
		if persistProgress {
			if err := updateRunProgress(ctx, tx, run, progressStatus); err != nil {
				return convergenceResult{}, err
			}
		}
		return convergenceResult{Status: progressStatus}, nil
	}
	if decision.Status == RunStatusFailed {
		if run.Status != RunStatusCancelling {
			if err := beginCancellingRun(ctx, tx, run, run.ErrorMessage, now); err != nil {
				return convergenceResult{}, err
			}
			if err := cancelUnsentRunTasks(ctx, tx, run, now); err != nil {
				return convergenceResult{}, err
			}
			counts, err := loadAuthoritativeRunCounts(ctx, tx, run.ID)
			if err != nil {
				return convergenceResult{}, err
			}
			applyAuthoritativeRunCounts(run, counts)
		}
		if !run.ReadyToFinalize() {
			if err := updateRunProgress(ctx, tx, run, RunStatusCancelling); err != nil {
				return convergenceResult{}, err
			}
			return convergenceResult{Failed: true, Status: RunStatusCancelling}, nil
		}
		if err := finalizeConvergedFailedRun(ctx, tx, run, now); err != nil {
			return convergenceResult{}, err
		}
		return convergenceResult{
			Finalized: true,
			Failed:    true,
			Status:    RunStatusFailed,
		}, nil
	}
	if err := markRunProcessing(ctx, tx, run); err != nil {
		return convergenceResult{}, err
	}
	if err := finalizeSuccessfulRun(ctx, tx, run, now); err != nil {
		return convergenceResult{}, err
	}
	return convergenceResult{
		Finalized: true,
		Status:    RunStatusSucceeded,
	}, nil
}
