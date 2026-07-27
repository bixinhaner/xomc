package adhoc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
	now             func() time.Time
}

var hiddenDeviceBuiltinTaskIDs = map[string]uuid.UUID{
	"lte": uuid.MustParse("0184dddd-0005-4000-8000-000000000001"),
	"nr":  uuid.MustParse("0184dddd-0005-4000-8000-000000000002"),
	"gsm": uuid.MustParse("0184dddd-0005-4000-8000-000000000003"),
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
		hidden, ok := hiddenDeviceDefinition(task)
		if !ok {
			continue
		}
		result.Definitions++
		empty, changed, saveErr = r.saveStreamingDefinition(ctx, &hidden, hiddenDeviceEffectiveFrom(r.currentTime()))
		if saveErr != nil {
			result.Failed++
			reconcileErrors = append(reconcileErrors, builtinReconcileError(&hidden, saveErr))
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
	rules, resolveErr := r.resolveRules(ctx, task.Technology, task.MetricPaths)
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

func (r *BuiltinReconciler) currentTime() time.Time {
	if r.now != nil {
		return r.now().UTC()
	}
	return time.Now().UTC()
}

func hiddenDeviceEffectiveFrom(now time.Time) time.Time {
	return now.UTC().Truncate(time.Hour)
}

func hiddenDeviceDefinition(source *Task) (Task, bool) {
	if source.Dimension != DimensionNetwork || source.Technology == "" {
		return Task{}, false
	}
	technology := strings.ToLower(source.Technology)
	taskID, ok := hiddenDeviceBuiltinTaskIDs[technology]
	if !ok {
		taskID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("omcgo.pm.hidden-device."+technology))
	}
	creator := source.Creator
	if creator == "" {
		creator = "system"
	}
	return Task{
		ID: taskID, Name: "内置-设备-" + strings.ToUpper(technology),
		Mode: ModeContinuous, MetricPaths: append([]string(nil), source.MetricPaths...),
		Dimension: DimensionDevice, Technology: source.Technology,
		IsBuiltin: true, Visibility: VisibilityPrivate, Status: source.Status,
		Creator: creator, ExpireDays: source.ExpireDays,
	}, true
}
