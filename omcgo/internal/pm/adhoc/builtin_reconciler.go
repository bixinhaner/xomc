package adhoc

import (
	"context"
	"errors"
	"fmt"
	"time"

	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
)

type BuiltinReconcileResult struct {
	Definitions int
	Saved       int
	Changed     int
	Empty       int
	Failed      int
}

type BuiltinReconciler struct {
	list                      func(context.Context) ([]Task, error)
	resolveEnabledMetricPaths func(context.Context, string) ([]string, error)
	resolveRules              func(context.Context, string, []string) ([]pmstream.MetricRule, error)
	resolveCounters           func(context.Context, string, []pmstream.MetricRule) ([]pmstream.CounterRule, error)
	resolveMembers            func(context.Context, *Task) ([]pmstream.TaskMember, error)
	save                      func(context.Context, pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error)
	purgeObsolete             func(context.Context) (int, error)
}

func NewBuiltinReconciler(repo *PgRepository) *BuiltinReconciler {
	return &BuiltinReconciler{
		list: func(ctx context.Context) ([]Task, error) {
			builtin := true
			return repo.List(ctx, ListFilter{IsBuiltin: &builtin, IncludeAll: true})
		},
		resolveEnabledMetricPaths: repo.resolveEnabledStreamingMetricPaths,
		resolveRules:              repo.resolveStreamingRules,
		resolveCounters:           repo.resolveStreamingCounters,
		resolveMembers:            repo.resolveStreamingMembers,
		save: func(ctx context.Context, req pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error) {
			if repo.streamRepo == nil {
				return nil, errors.New("PM streaming task repository is not configured")
			}
			return repo.streamRepo.Save(ctx, req)
		},
		purgeObsolete: func(ctx context.Context) (int, error) {
			if repo.streamRepo == nil {
				return 0, errors.New("PM streaming task repository is not configured")
			}
			return repo.streamRepo.PurgeObsoleteBuiltinDeviceTasks(ctx)
		},
	}
}

func (r *BuiltinReconciler) Reconcile(ctx context.Context) (BuiltinReconcileResult, error) {
	var result BuiltinReconcileResult
	if r.purgeObsolete != nil {
		removed, err := r.purgeObsolete(ctx)
		if err != nil {
			return result, fmt.Errorf("purge obsolete built-in PM device tasks: %w", err)
		}
		result.Changed += removed
	}
	tasks, err := r.list(ctx)
	if err != nil {
		return result, fmt.Errorf("list builtin PM aggregation tasks: %w", err)
	}
	var reconcileErrors []error
	for i := range tasks {
		task := &tasks[i]
		if !task.IsBuiltin || task.Mode != ModeContinuous {
			continue
		}
		result.Definitions++
		empty, changed, saveErr := r.saveStreamingDefinition(ctx, task, time.Time{})
		if saveErr != nil {
			result.Failed++
			reconcileErrors = append(reconcileErrors, builtinReconcileError(task, saveErr))
			continue
		}
		result.Saved++
		if empty {
			result.Empty++
		}
		if changed {
			result.Changed++
		}
	}
	return result, errors.Join(reconcileErrors...)
}

func builtinReconcileError(task *Task, err error) error {
	return fmt.Errorf("reconcile builtin PM aggregation task %s (%s): %w", task.ID, task.Name, err)
}

func (r *BuiltinReconciler) saveStreamingDefinition(ctx context.Context, task *Task, effectiveFrom time.Time) (bool, bool, error) {
	outputMetricPaths := task.MetricPaths
	if r.resolveEnabledMetricPaths != nil {
		enabledPaths, resolveErr := r.resolveEnabledMetricPaths(ctx, task.Technology)
		if resolveErr != nil {
			return false, false, resolveErr
		}
		outputMetricPaths = streamingOutputMetricPaths(task.MetricPaths, enabledPaths)
	}
	rules, resolveErr := r.resolveRules(ctx, task.Technology, outputMetricPaths)
	if resolveErr != nil {
		return false, false, resolveErr
	}
	counters, resolveErr := r.resolveCounters(ctx, task.Technology, rules)
	if resolveErr != nil {
		return false, false, resolveErr
	}
	members, resolveErr := r.resolveMembers(ctx, task)
	if resolveErr != nil {
		return false, false, resolveErr
	}
	creator := task.Creator
	if creator == "" {
		creator = "system"
	}
	snapshot, saveErr := r.save(ctx, pmstream.SaveTaskRequest{
		TaskID: task.ID, Name: task.Name, Enabled: task.Status != StatusCanceled,
		Visibility: string(normalizeVisibility(task.Visibility)), Creator: creator,
		Technology: task.Technology, Dimension: pmstream.Dimension(task.Dimension),
		Granularities: streamingRollupGranularities(), ObjectLDNs: task.ObjectLDNs,
		Metrics: rules, Counters: counters, Members: members, EffectiveFrom: effectiveFrom,
	})
	if saveErr != nil {
		return false, false, saveErr
	}
	return len(members) == 0, snapshot != nil && snapshot.NewVersion, nil
}
