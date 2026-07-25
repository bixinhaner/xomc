package adhoc

import (
	"context"
	"errors"
	"fmt"

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
	list            func(context.Context) ([]Task, error)
	resolveRules    func(context.Context, string, []string) ([]pmstream.MetricRule, error)
	resolveCounters func(context.Context, string, []pmstream.MetricRule) ([]pmstream.CounterRule, error)
	resolveMembers  func(context.Context, *Task) ([]pmstream.TaskMember, error)
	save            func(context.Context, pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error)
}

func NewBuiltinReconciler(repo *PgRepository) *BuiltinReconciler {
	return &BuiltinReconciler{
		list: func(ctx context.Context) ([]Task, error) {
			builtin := true
			return repo.List(ctx, ListFilter{IsBuiltin: &builtin, IncludeAll: true})
		},
		resolveRules:    repo.resolveStreamingRules,
		resolveCounters: repo.resolveStreamingCounters,
		resolveMembers:  repo.resolveStreamingMembers,
		save: func(ctx context.Context, req pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error) {
			if repo.streamRepo == nil {
				return nil, errors.New("PM streaming task repository is not configured")
			}
			return repo.streamRepo.Save(ctx, req)
		},
	}
}

func (r *BuiltinReconciler) Reconcile(ctx context.Context) (BuiltinReconcileResult, error) {
	var result BuiltinReconcileResult
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
		rules, resolveErr := r.resolveRules(ctx, task.Technology, task.MetricPaths)
		if resolveErr != nil {
			result.Failed++
			reconcileErrors = append(reconcileErrors, builtinReconcileError(task, resolveErr))
			continue
		}
		counters, resolveErr := r.resolveCounters(ctx, task.Technology, rules)
		if resolveErr != nil {
			result.Failed++
			reconcileErrors = append(reconcileErrors, builtinReconcileError(task, resolveErr))
			continue
		}
		members, resolveErr := r.resolveMembers(ctx, task)
		if resolveErr != nil {
			result.Failed++
			reconcileErrors = append(reconcileErrors, builtinReconcileError(task, resolveErr))
			continue
		}
		if len(members) == 0 {
			result.Empty++
		}
		granularities := streamingRollupGranularities()
		creator := task.Creator
		if creator == "" {
			creator = "system"
		}
		snapshot, saveErr := r.save(ctx, pmstream.SaveTaskRequest{
			TaskID: task.ID, Name: task.Name, Enabled: task.Status != StatusCanceled,
			Visibility: string(normalizeVisibility(task.Visibility)), Creator: creator,
			Technology: task.Technology, Dimension: pmstream.Dimension(task.Dimension),
			Granularities: granularities, ObjectLDNs: task.ObjectLDNs,
			Metrics: rules, Counters: counters, Members: members,
		})
		if saveErr != nil {
			result.Failed++
			reconcileErrors = append(reconcileErrors, builtinReconcileError(task, saveErr))
			continue
		}
		result.Saved++
		if snapshot != nil && snapshot.NewVersion {
			result.Changed++
		}
	}
	return result, errors.Join(reconcileErrors...)
}

func builtinReconcileError(task *Task, err error) error {
	return fmt.Errorf("reconcile builtin PM aggregation task %s (%s): %w", task.ID, task.Name, err)
}
