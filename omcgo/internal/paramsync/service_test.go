package paramsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
)

type stubPlanner struct {
	plan *Plan
	err  error
}

func (p stubPlanner) Plan(context.Context, PlanCommand) (*Plan, error) { return p.plan, p.err }

type failingDispatcher struct{ err error }

func (d failingDispatcher) Dispatch(context.Context, *SyncRun, *Plan) ([]*task.Task, error) {
	return nil, d.err
}

type memoryRequestRepo struct {
	requests map[uuid.UUID]*SyncRequest
	byKey    map[string]*SyncRequest
	active   *SyncRun
	gateErr  error
	backoff  time.Time
}

func newMemoryRequestRepo() *memoryRequestRepo {
	return &memoryRequestRepo{requests: map[uuid.UUID]*SyncRequest{}, byKey: map[string]*SyncRequest{}}
}

func (r *memoryRequestRepo) CreateRequest(_ context.Context, req *SyncRequest) error {
	if req.IdempotencyKey != nil {
		key := req.CallerType + ":" + *req.IdempotencyKey
		if _, exists := r.byKey[key]; exists {
			return errors.New("duplicate")
		}
		r.byKey[key] = req
	}
	r.requests[req.ID] = req
	return nil
}

func (r *memoryRequestRepo) GetRequest(_ context.Context, id uuid.UUID) (*SyncRequest, error) {
	req, ok := r.requests[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return req, nil
}

func (r *memoryRequestRepo) FindRequestByIdempotency(_ context.Context, callerType, key string) (*SyncRequest, error) {
	req, ok := r.byKey[callerType+":"+key]
	if !ok {
		return nil, errors.New("not found")
	}
	return req, nil
}

func (r *memoryRequestRepo) CompleteRequest(_ context.Context, id uuid.UUID, status RequestStatus, code ResultCode, activeRunID *uuid.UUID, message string) error {
	req := r.requests[id]
	req.Status, req.ResultCode, req.ActiveRunID, req.ErrorMessage = status, code, activeRunID, message
	return nil
}

func (r *memoryRequestRepo) QueueRequest(_ context.Context, id uuid.UUID, nextAttemptAt time.Time) error {
	req := r.requests[id]
	req.Status, req.ResultCode, req.ActiveRunID, req.NextAttemptAt = RequestStatusQueued, ResultCodeActiveSyncExists, nil, nextAttemptAt
	return nil
}

func (r *memoryRequestRepo) ClaimQueuedRequests(_ context.Context, now time.Time, limit int) ([]*SyncRequest, error) {
	var queued []*SyncRequest
	for _, req := range r.requests {
		if req.Status == RequestStatusQueued && !req.NextAttemptAt.After(now) {
			queued = append(queued, req)
			if len(queued) == limit {
				break
			}
		}
	}
	return queued, nil
}

func (r *memoryRequestRepo) FailRunAndRequest(_ context.Context, runID, requestID uuid.UUID, code ResultCode, message string) error {
	req := r.requests[requestID]
	req.Status, req.ResultCode, req.ActiveRunID, req.ErrorMessage = RequestStatusFailed, code, nil, message
	if r.active != nil && r.active.ID == runID {
		r.active.Status = RunStatusFailed
		r.active = nil
	}
	return nil
}

func (r *memoryRequestRepo) CreateOrDeduplicateRun(_ context.Context, req *SyncRequest) (*StartResult, error) {
	if r.active != nil {
		return &StartResult{Request: req, Deduplicated: true, ActiveRunID: &r.active.ID, ActiveRun: r.active}, nil
	}
	run := &SyncRun{ID: uuid.New(), RequestID: req.ID, DeviceID: req.DeviceID, SyncScope: req.SyncScope, Status: RunStatusPlanning}
	req.RunID, req.ActiveRunID, req.Status = &run.ID, &run.ID, RequestStatusRunning
	r.active = run
	return &StartResult{Request: req, Run: run}, nil
}

func (r *memoryRequestRepo) GetRun(context.Context, uuid.UUID) (*SyncRun, error) {
	return r.active, nil
}
func (r *memoryRequestRepo) GetActiveRunByDevice(context.Context, uuid.UUID) (*SyncRun, error) {
	if r.active == nil {
		return nil, errors.New("not found")
	}
	return r.active, nil
}
func (r *memoryRequestRepo) ListRequestsByDevice(_ context.Context, deviceID uuid.UUID, _ int) ([]*SyncRequest, error) {
	var out []*SyncRequest
	for _, req := range r.requests {
		if req.DeviceID == deviceID {
			out = append(out, req)
		}
	}
	return out, nil
}
func (r *memoryRequestRepo) InsertTaskResultIfAbsent(context.Context, *TaskResult) (bool, error) {
	return true, nil
}
func (r *memoryRequestRepo) ClaimRunForFinalize(context.Context, uuid.UUID) (bool, error) {
	return true, nil
}

func (r *memoryRequestRepo) CreateAutomaticRequest(ctx context.Context, req *SyncRequest, now time.Time) (bool, error) {
	if r.gateErr != nil {
		return false, r.gateErr
	}
	if !r.backoff.IsZero() {
		req.Status = RequestStatusRejected
		req.ResultCode = ResultCodeAutomaticBackoff
		req.ErrorMessage = "automatic parameter sync backed off"
		req.CompletedAt = &now
		return false, r.CreateRequest(ctx, req)
	}
	return true, r.CreateRequest(ctx, req)
}

func submitCommand(reason TriggerReason) SubmitCommand {
	return SubmitCommand{DeviceID: uuid.New(), DeviceSN: "SN-1", CallerType: "api", TriggerReason: reason, Scope: SyncScopeFull}
}

func TestService_SubmitReturnsDurableRequestAndRunIDs(t *testing.T) {
	repo := newMemoryRequestRepo()
	service := NewService(repo, stubPlanner{plan: &Plan{Batches: []TaskBatch{{Paths: []string{"Device."}}}}})
	got, err := service.Submit(context.Background(), submitCommand(TriggerManual))
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, got.RequestID)
	assert.NotNil(t, got.RunID)
	assert.Equal(t, RequestStatusRunning, got.Status)
	assert.Equal(t, 1, got.TaskCount)
}

func TestService_SubmitConflictIsRejectedForUserAndDeduplicatedForAutomation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		reason TriggerReason
		want   RequestStatus
	}{
		{name: "manual", reason: TriggerManual, want: RequestStatusRejected},
		{name: "automatic", reason: TriggerPeriodic, want: RequestStatusDeduplicated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMemoryRequestRepo()
			repo.active = &SyncRun{ID: uuid.New(), Status: RunStatusExecuting, SyncScope: SyncScopeFull}
			service := NewService(repo, stubPlanner{plan: &Plan{Batches: []TaskBatch{{Paths: []string{"Device."}}}}})
			got, err := service.Submit(context.Background(), submitCommand(tc.reason))
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Status)
			assert.Equal(t, ResultCodeActiveSyncExists, got.ResultCode)
			assert.Equal(t, repo.active.ID, *got.ActiveRunID)
		})
	}
}

func TestService_SubmitDoesNotDeduplicateOntoIncompatiblePartialRun(t *testing.T) {
	repo := newMemoryRequestRepo()
	repo.active = &SyncRun{ID: uuid.New(), Status: RunStatusExecuting, SyncScope: SyncScopePartial}
	service := NewService(repo, stubPlanner{plan: &Plan{Batches: []TaskBatch{{Paths: []string{"Device."}}}}})
	cmd := submitCommand(TriggerPeriodic)
	cmd.Scope = SyncScopeFull

	got, err := service.Submit(context.Background(), cmd)

	require.NoError(t, err)
	assert.Equal(t, RequestStatusQueued, got.Status)
	assert.Equal(t, ResultCodeActiveSyncExists, got.ResultCode)
	assert.Nil(t, got.ActiveRunID, "queued request must not claim ownership of the incompatible active run")
}

func TestService_SubmitIdempotencyReturnsOriginalRequest(t *testing.T) {
	repo := newMemoryRequestRepo()
	service := NewService(repo, stubPlanner{plan: &Plan{Batches: []TaskBatch{{Paths: []string{"Device."}}}}})
	cmd := submitCommand(TriggerManual)
	cmd.IdempotencyKey = "same-click"
	first, err := service.Submit(context.Background(), cmd)
	require.NoError(t, err)
	second, err := service.Submit(context.Background(), cmd)
	require.NoError(t, err)
	assert.Equal(t, first.RequestID, second.RequestID)
}

func TestService_SubmitNoStorablePathCompletesWithoutRun(t *testing.T) {
	repo := newMemoryRequestRepo()
	service := NewService(repo, stubPlanner{plan: &Plan{}})
	got, err := service.Submit(context.Background(), submitCommand(TriggerDeviceOnline))
	require.NoError(t, err)
	assert.Equal(t, RequestStatusSucceeded, got.Status)
	assert.Equal(t, ResultCodeNoStorablePath, got.ResultCode)
	assert.Nil(t, got.RunID)
}

func TestService_SubmitMappingUnavailableIsObservable(t *testing.T) {
	repo := newMemoryRequestRepo()
	service := NewService(repo, stubPlanner{err: ErrMappingUnavailable})
	got, err := service.Submit(context.Background(), submitCommand(TriggerManual))
	require.NoError(t, err)
	assert.Equal(t, RequestStatusRejected, got.Status)
	assert.Equal(t, ResultCodePathBUnavailable, got.ResultCode)
}

func TestService_SubmitAutomaticGateFailureDoesNotLeakAcceptedRequest(t *testing.T) {
	repo := newMemoryRequestRepo()
	repo.gateErr = errors.New("gate unavailable")
	service := NewService(repo, stubPlanner{plan: &Plan{Batches: []TaskBatch{{Paths: []string{"Device."}}}}})

	_, err := service.Submit(context.Background(), submitCommand(TriggerDeviceOnline))

	require.ErrorContains(t, err, "gate unavailable")
	assert.Empty(t, repo.requests)
}

func TestService_SubmitAutomaticBackoffIsPersistedAsRejected(t *testing.T) {
	repo := newMemoryRequestRepo()
	repo.backoff = time.Now().Add(time.Hour)
	service := NewService(repo, stubPlanner{plan: &Plan{Batches: []TaskBatch{{Paths: []string{"Device."}}}}})

	got, err := service.Submit(context.Background(), submitCommand(TriggerDeviceOnline))

	require.NoError(t, err)
	assert.Equal(t, RequestStatusRejected, got.Status)
	assert.Equal(t, ResultCodeAutomaticBackoff, got.ResultCode)
	require.Len(t, repo.requests, 1)
	for _, req := range repo.requests {
		assert.Equal(t, RequestStatusRejected, req.Status)
		assert.NotNil(t, req.CompletedAt)
	}
}

func TestService_SubmitDispatchFailureReleasesActiveRun(t *testing.T) {
	repo := newMemoryRequestRepo()
	service := NewService(repo, stubPlanner{plan: &Plan{Batches: []TaskBatch{{Paths: []string{"Device."}}}}}).
		WithDispatcher(failingDispatcher{err: errors.New("queue unavailable")})

	_, err := service.Submit(context.Background(), submitCommand(TriggerManual))

	require.ErrorContains(t, err, "queue unavailable")
	assert.Nil(t, repo.active, "a failed dispatch must not leave a full run holding the active-device gate")

	service.dispatcher = nil
	result, retryErr := service.Submit(context.Background(), submitCommand(TriggerManual))
	require.NoError(t, retryErr)
	assert.Equal(t, RequestStatusRunning, result.Status)
}
