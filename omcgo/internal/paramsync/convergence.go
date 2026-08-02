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

type convergenceResult struct {
	Finalized bool
	Failed    bool
	Drift     bool
	Status    RunStatus
}

func decideRunConvergence(run SyncRun) convergenceDecision {
	if run.Status.Terminal() || !run.ReadyToFinalize() {
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
	applyAuthoritativeRunCounts(run, counts)
	result, err := convergeLoadedRunTx(ctx, tx, run, run.Status, drift, now)
	result.Drift = drift
	return result, err
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
