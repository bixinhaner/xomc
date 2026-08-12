package paramsync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

var (
	ErrInvalidSubmitCommand       = errors.New("invalid parameter sync submit command")
	ErrRequestIdempotencyConflict = errors.New("parameter sync request idempotency conflict")
)

type RequestPlanner interface {
	Plan(ctx context.Context, cmd PlanCommand) (*Plan, error)
}

type Service struct {
	repo       Repository
	planner    RequestPlanner
	dispatcher PlannedTaskDispatcher
	metrics    *Metrics
	now        func() time.Time
}

type queuedRequestRepository interface {
	QueueRequest(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time) error
	ClaimQueuedRequests(ctx context.Context, now time.Time, limit int) ([]*SyncRequest, error)
}

type automaticAdmissionRepository interface {
	TryReserveAutomaticAdmission(ctx context.Context, req *SyncRequest, now time.Time) (bool, error)
}

type automaticAdmissionTaskRepository interface {
	AdjustAutomaticAdmissionTasks(ctx context.Context, req *SyncRequest, plannedTasks int, now time.Time) (bool, error)
}

type automaticAdmissionQueueRepository interface {
	QueueRequestAndReleaseAutomaticAdmission(ctx context.Context, req *SyncRequest, nextAttemptAt time.Time, now time.Time) error
}

func (s *Service) WithMetrics(metrics *Metrics) *Service {
	s.metrics = metrics
	return s
}

func (s *Service) WithDispatcher(dispatcher PlannedTaskDispatcher) *Service {
	s.dispatcher = dispatcher
	return s
}

func NewService(repo Repository, planner RequestPlanner) *Service {
	return &Service{repo: repo, planner: planner, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Submit(ctx context.Context, cmd SubmitCommand) (*SubmitResult, error) {
	if cmd.DeviceID == uuid.Nil || strings.TrimSpace(cmd.DeviceSN) == "" || cmd.Scope == "" || !cmd.TriggerReason.Valid() {
		return nil, ErrInvalidSubmitCommand
	}
	if cmd.CallerType == "" {
		cmd.CallerType = "system"
	}
	if cmd.Priority <= 0 {
		cmd.Priority = 10
	}
	now := s.now()
	req := &SyncRequest{
		ID: uuid.New(), DeviceID: cmd.DeviceID, DeviceSN: cmd.DeviceSN, CallerType: cmd.CallerType,
		TriggerReason: cmd.TriggerReason, SyncScope: cmd.Scope, RequestedPaths: append([]string(nil), cmd.RequestedPaths...),
		Status: RequestStatusAccepted, Priority: cmd.Priority, NextAttemptAt: now, DeadlineAt: cmd.DeadlineAt,
		CreatedAt: now, UpdatedAt: now,
	}
	if cmd.IdempotencyKey != "" {
		req.IdempotencyKey = &cmd.IdempotencyKey
	}
	if cmd.CampaignID != nil {
		campaignID := *cmd.CampaignID
		req.CampaignID = &campaignID
	}
	if cmd.SourceEventID != "" {
		req.SourceEventID = &cmd.SourceEventID
	}
	if cmd.OriginEventType != "" {
		req.OriginEventType = &cmd.OriginEventType
	}
	if cmd.ModelUploadIntentID != nil {
		req.ModelUploadIntentID = cmd.ModelUploadIntentID
	}
	if cmd.ModelUploadStatus != "" {
		req.ModelUploadStatus = &cmd.ModelUploadStatus
	}
	allowed := true
	var createErr error
	if cmd.TriggerReason.Automatic() {
		allowed, createErr = s.repo.CreateAutomaticRequest(ctx, req, now)
	} else {
		createErr = s.repo.CreateRequest(ctx, req)
	}
	if createErr != nil {
		if cmd.IdempotencyKey != "" {
			existing, findErr := s.repo.FindRequestByIdempotency(ctx, cmd.CallerType, cmd.IdempotencyKey)
			if findErr == nil {
				return resultFromRequest(existing, 0), nil
			}
		}
		return nil, fmt.Errorf("create parameter sync request: %w", createErr)
	}
	if s.metrics != nil {
		s.metrics.RequestsTotal.WithLabelValues(string(cmd.Scope), string(cmd.TriggerReason), string(req.Status)).Inc()
	}
	if !allowed {
		return resultFromRequest(req, 0), nil
	}
	return s.startRequest(ctx, req)
}

func (s *Service) startRequest(ctx context.Context, req *SyncRequest) (*SubmitResult, error) {
	plan, err := s.planner.Plan(ctx, PlanCommand{
		RunID: uuid.New(), Device: &model.Device{ID: req.DeviceID, SerialNumber: req.DeviceSN},
		Scope: req.SyncScope, RequestedPaths: req.RequestedPaths, TriggerReason: req.TriggerReason, Priority: req.Priority,
	})
	if err != nil {
		code := ResultCodeResultProcessingFailed
		status := RequestStatusFailed
		if errors.Is(err, ErrMappingUnavailable) {
			code, status = ResultCodePathBUnavailable, RequestStatusRejected
		}
		if completeErr := s.repo.CompleteRequest(ctx, req.ID, status, code, nil, err.Error()); completeErr != nil {
			return nil, errors.Join(fmt.Errorf("plan parameter sync: %w", err), completeErr)
		}
		req.Status, req.ResultCode, req.ErrorMessage = status, code, err.Error()
		return resultFromRequest(req, 0), nil
	}
	if plan == nil || len(plan.Batches) == 0 {
		if err := s.repo.CompleteRequest(ctx, req.ID, RequestStatusSucceeded, ResultCodeNoStorablePath, nil, ""); err != nil {
			return nil, err
		}
		req.Status, req.ResultCode = RequestStatusSucceeded, ResultCodeNoStorablePath
		return resultFromRequest(req, 0), nil
	}
	if req.TriggerReason.Automatic() {
		admission, supported := s.repo.(automaticAdmissionTaskRepository)
		if supported {
			allowed, adjustErr := admission.AdjustAutomaticAdmissionTasks(ctx, req, len(plan.Batches), s.now())
			if adjustErr != nil {
				return nil, adjustErr
			}
			if !allowed {
				return resultFromRequest(req, 0), nil
			}
		}
	}

	start, err := s.repo.CreateOrDeduplicateRun(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("start parameter sync run: %w", err)
	}
	if start.Deduplicated {
		status := RequestStatusDeduplicated
		compatible := start.ActiveRun != nil && start.ActiveRun.SyncScope.IsFull()
		if req.TriggerReason == TriggerManual {
			status = RequestStatusRejected
		}
		if !compatible && req.TriggerReason != TriggerManual {
			if queuedRepo, ok := s.repo.(queuedRequestRepository); ok && start.ActiveRunID != nil {
				nextAttemptAt := s.now().Add(time.Second)
				if admission, supported := s.repo.(automaticAdmissionQueueRepository); supported && req.TriggerReason.Automatic() {
					if err := admission.QueueRequestAndReleaseAutomaticAdmission(ctx, req, nextAttemptAt, s.now()); err != nil {
						return nil, err
					}
				} else if err := queuedRepo.QueueRequest(ctx, req.ID, nextAttemptAt); err != nil {
					return nil, err
				}
				req.Status, req.ResultCode, req.ActiveRunID, req.NextAttemptAt = RequestStatusQueued, ResultCodeActiveSyncExists, nil, nextAttemptAt
				return resultFromRequest(req, 0), nil
			}
			status = RequestStatusRejected
		}
		activeRunID := start.ActiveRunID
		if !compatible {
			activeRunID = nil
		}
		if err := s.repo.CompleteRequest(ctx, req.ID, status, ResultCodeActiveSyncExists, activeRunID, "active parameter sync already exists"); err != nil {
			return nil, err
		}
		req.Status, req.ResultCode, req.ActiveRunID = status, ResultCodeActiveSyncExists, activeRunID
		return resultFromRequest(req, 0), nil
	}
	if s.dispatcher != nil {
		tasks, dispatchErr := s.dispatcher.Dispatch(ctx, start.Run, plan)
		if dispatchErr != nil {
			if completeErr := s.repo.FailRunAndRequest(ctx, start.Run.ID, req.ID, ResultCodeResultProcessingFailed, dispatchErr.Error()); completeErr != nil {
				return nil, errors.Join(dispatchErr, completeErr)
			}
			return nil, fmt.Errorf("dispatch parameter sync plan: %w", dispatchErr)
		}
		return resultFromRequest(start.Request, len(tasks)), nil
	}
	return resultFromRequest(start.Request, len(plan.Batches)), nil
}

// DispatchQueued retries durable requests that could not acquire the per-device
// run gate. Claiming advances next_attempt_at before planning so multiple APP
// instances cannot hot-loop the same request.
func (s *Service) DispatchQueued(ctx context.Context, limit int) (int, error) {
	return s.DispatchQueuedConcurrent(ctx, limit, 1)
}

// DispatchQueuedConcurrent retries claimed durable requests with bounded
// concurrency. Planning one full parameter-sync request may touch the model
// library and create many durable tasks; processing a cold-start claim
// sequentially makes the maintenance deadline cancel the unvisited tail.
func (s *Service) DispatchQueuedConcurrent(ctx context.Context, limit, workers int) (int, error) {
	repo, ok := s.repo.(queuedRequestRepository)
	if !ok {
		return 0, nil
	}
	requests, err := repo.ClaimQueuedRequests(ctx, s.now(), limit)
	if err != nil {
		return 0, err
	}
	return runConcurrentQueuedRequests(requests, workers, func(req *SyncRequest) (bool, error) {
		if req.TriggerReason.Automatic() {
			admission, supported := s.repo.(automaticAdmissionRepository)
			if supported {
				allowed, reserveErr := admission.TryReserveAutomaticAdmission(ctx, req, s.now())
				if reserveErr != nil {
					return false, reserveErr
				}
				if !allowed {
					return false, nil
				}
			}
		}
		result, dispatchErr := s.startRequest(ctx, req)
		if dispatchErr != nil {
			return false, dispatchErr
		}
		return result.Status == RequestStatusRunning || result.Status.Terminal(), nil
	})
}

func runConcurrentQueuedRequests(
	requests []*SyncRequest,
	workers int,
	dispatch func(*SyncRequest) (bool, error),
) (int, error) {
	if len(requests) == 0 {
		return 0, nil
	}
	if workers < 1 {
		workers = 1
	}
	if workers > len(requests) {
		workers = len(requests)
	}
	type result struct {
		dispatched bool
		err        error
	}
	jobs := make(chan *SyncRequest)
	results := make(chan result, len(requests))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range jobs {
				dispatched, err := dispatch(req)
				results <- result{dispatched: dispatched, err: err}
			}
		}()
	}
	for _, req := range requests {
		jobs <- req
	}
	close(jobs)
	wg.Wait()
	close(results)

	dispatched := 0
	var errs []error
	for item := range results {
		if item.dispatched {
			dispatched++
		}
		if item.err != nil {
			errs = append(errs, item.err)
		}
	}
	return dispatched, errors.Join(errs...)
}

func (s *Service) GetRequest(ctx context.Context, requestID uuid.UUID) (*SyncRequest, error) {
	return s.repo.GetRequest(ctx, requestID)
}

func (s *Service) GetRun(ctx context.Context, runID uuid.UUID) (*SyncRun, error) {
	return s.repo.GetRun(ctx, runID)
}

func (s *Service) FindRequestByIdempotency(ctx context.Context, callerType, key string) (*SyncRequest, error) {
	return s.repo.FindRequestByIdempotency(ctx, callerType, key)
}

func (s *Service) GetActiveRun(ctx context.Context, deviceID uuid.UUID) (*SyncRun, error) {
	return s.repo.GetActiveRunByDevice(ctx, deviceID)
}

func (s *Service) ListHistory(ctx context.Context, deviceID uuid.UUID, limit int) ([]*SyncRequest, error) {
	return s.repo.ListRequestsByDevice(ctx, deviceID, limit)
}

func resultFromRequest(req *SyncRequest, taskCount int) *SubmitResult {
	return &SubmitResult{
		RequestID: req.ID, RunID: req.RunID, Status: req.Status, ResultCode: req.ResultCode,
		Scope: req.SyncScope, TriggerReason: req.TriggerReason, TaskCount: taskCount, ActiveRunID: req.ActiveRunID,
	}
}
