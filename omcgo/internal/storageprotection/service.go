package storageprotection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Service struct {
	repo    Repository
	usage   UsageProvider
	metrics *Metrics
	logger  *zap.Logger
	now     func() time.Time
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
}

// Start periodically evaluates all enabled policies. Admission checks still
// perform an immediate evaluation, so a newly configured policy does not wait
// for the next tick before becoming effective.
func (s *Service) Start(ctx context.Context, interval time.Duration) {
	if s == nil || s.repo == nil || s.usage == nil {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	done := s.done
	s.mu.Unlock()
	go func() {
		defer close(done)
		s.evaluateAll(workerCtx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				s.evaluateAll(workerCtx)
			}
		}
	}()
}

func (s *Service) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

func (s *Service) evaluateAll(ctx context.Context) {
	policies, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Warn("list storage protection policies failed", zap.Error(err))
		return
	}
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}
		if _, err := s.Check(ctx, policy.TargetType, policy.TargetID, policy.WriteScope); err != nil {
			s.logger.Warn("evaluate storage protection policy failed", zap.String("policy_id", policy.ID), zap.Error(err))
		}
	}
}

func NewService(repo Repository, usage UsageProvider, metrics *Metrics, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: repo, usage: usage, metrics: metrics, logger: logger.Named("storage-protection"), now: time.Now}
}

// Check implements WriteAdmission. A missing policy is deliberately fail-open:
// monitoring alone must not unexpectedly block existing deployments.
func (s *Service) Check(ctx context.Context, targetType TargetType, targetID string, scope WriteScope) (AdmissionDecision, error) {
	if s == nil || s.repo == nil {
		return AdmissionDecision{Allowed: true, State: StateNormal, Reason: "storage protection is not configured", ObservedAt: time.Now()}, nil
	}
	policy, err := s.repo.GetEnabledPolicy(ctx, targetType, targetID, scope)
	if err != nil {
		return AdmissionDecision{}, fmt.Errorf("load storage protection policy: %w", err)
	}
	if policy == nil {
		return AdmissionDecision{Allowed: true, State: StateNormal, Reason: "no enabled storage protection policy", ObservedAt: s.now()}, nil
	}
	if err := validatePolicy(policy); err != nil {
		return AdmissionDecision{}, err
	}
	s.setPolicyInfo(policy)
	if s.usage == nil {
		return s.handleUnknown(ctx, policy, scope, "storage usage provider is unavailable")
	}
	usage, usageErr := s.usage.Snapshot(ctx, targetType, targetID)
	if usageErr != nil || !usage.Available {
		reason := "storage usage is unavailable"
		if usageErr != nil {
			reason = usageErr.Error()
		}
		if s.metrics != nil {
			s.metrics.CheckFailuresTotal.WithLabelValues(string(targetType), targetID, string(scope)).Inc()
		}
		return s.handleUnknown(ctx, policy, scope, reason)
	}
	s.observeUsage(policy, scope, usage)
	previous := policy.CurrentState
	if previous == StateUnknown {
		previous = StateNormal
		policy.StateObservations = 0
	}
	nextState, observations, reason := evaluateState(policy, usage.UsedRatio, usage.ObservedAt)
	policy.CurrentState = nextState
	policy.StateObservations = observations
	policy.LastObservedRatio = floatPtr(usage.UsedRatio)
	policy.LastObservedAt = timePtr(usage.ObservedAt)
	if policy.CurrentState != previous {
		now := s.now()
		policy.LastStateChangedAt = &now
	}
	if err := s.repo.UpdateState(ctx, policy); err != nil {
		return AdmissionDecision{}, fmt.Errorf("persist storage protection state: %w", err)
	}
	if policy.CurrentState != previous {
		event := Event{PolicyID: policy.ID, TargetType: targetType, TargetID: targetID, WriteScope: scope, PreviousState: previous, NewState: policy.CurrentState, Reason: transitionReason(previous, policy.CurrentState, reason), ObservedRatio: floatPtr(usage.UsedRatio), PolicyVersion: policy.Version, OperatorID: "system", CreatedAt: s.now()}
		if err := s.repo.RecordEvent(ctx, event); err != nil {
			return AdmissionDecision{}, fmt.Errorf("record storage protection transition: %w", err)
		}
	}
	return s.decision(policy, usage.ObservedAt, reason), nil
}

func (s *Service) handleUnknown(ctx context.Context, policy *Policy, scope WriteScope, reason string) (AdmissionDecision, error) {
	previous := policy.CurrentState
	if previous == "" {
		previous = StateNormal
	}
	policy.CurrentState = StateUnknown
	policy.StateObservations = 0
	now := s.now()
	policy.LastObservedAt = &now
	if previous != StateUnknown {
		policy.LastStateChangedAt = &now
	}
	if err := s.repo.UpdateState(ctx, policy); err != nil {
		return AdmissionDecision{}, fmt.Errorf("persist unknown storage protection state: %w", err)
	}
	if previous != StateUnknown {
		event := Event{
			PolicyID: policy.ID, TargetType: policy.TargetType, TargetID: policy.TargetID,
			WriteScope: scope, PreviousState: previous, NewState: StateUnknown,
			Reason: reason, PolicyVersion: policy.Version, OperatorID: "system", CreatedAt: now,
		}
		if err := s.repo.RecordEvent(ctx, event); err != nil {
			return AdmissionDecision{}, fmt.Errorf("record storage protection unknown transition: %w", err)
		}
	}
	return s.unknownDecision(policy, reason), nil
}

func (s *Service) unknownDecision(policy *Policy, reason string) AdmissionDecision {
	if s.metrics != nil {
		s.metrics.AdmissionState.WithLabelValues(string(policy.TargetType), policy.TargetID, string(policy.WriteScope)).Set(stateValue(StateUnknown))
	}
	allowed := policy.UnknownBehavior == UnknownAllowWithAlarm
	if !allowed && s.metrics != nil {
		s.metrics.WriteRejectedTotal.WithLabelValues(string(policy.TargetType), policy.TargetID, string(policy.WriteScope), "unknown").Inc()
	}
	return AdmissionDecision{Allowed: allowed, State: StateUnknown, Reason: reason, RetryAfter: time.Duration(policy.CheckIntervalSeconds) * time.Second, ObservedAt: s.now()}
}

func (s *Service) observeUsage(policy *Policy, scope WriteScope, usage UsageSnapshot) {
	if s.metrics == nil {
		return
	}
	labels := []string{string(policy.TargetType), policy.TargetID, string(scope)}
	s.setPolicyInfo(policy)
	s.metrics.CapacityBytes.WithLabelValues(labels...).Set(float64(usage.CapacityBytes))
	s.metrics.UsedBytes.WithLabelValues(labels...).Set(float64(usage.UsedBytes))
	s.metrics.UsedRatio.WithLabelValues(labels...).Set(usage.UsedRatio)
}

func (s *Service) setPolicyInfo(policy *Policy) {
	if s.metrics == nil || policy == nil {
		return
	}
	s.metrics.PolicyInfo.WithLabelValues(
		string(policy.TargetType), policy.TargetID, string(policy.WriteScope),
		fmt.Sprintf("%t", policy.Enabled), string(policy.UnknownBehavior),
	).Set(1)
}

func (s *Service) decision(policy *Policy, observedAt time.Time, reason string) AdmissionDecision {
	allowed := policy.CurrentState != StateBlocked
	if policy.CurrentState == StateUnknown {
		allowed = policy.UnknownBehavior == UnknownAllowWithAlarm
	}
	if s.metrics != nil {
		labels := []string{string(policy.TargetType), policy.TargetID, string(policy.WriteScope)}
		s.metrics.AdmissionState.WithLabelValues(labels...).Set(stateValue(policy.CurrentState))
		if policy.CurrentState == StateBlocked {
			s.metrics.WriteRejectedTotal.WithLabelValues(string(policy.TargetType), policy.TargetID, string(policy.WriteScope), "blocked").Inc()
		} else if policy.CurrentState == StateWarning {
			s.metrics.WriteDegradedTotal.WithLabelValues(string(policy.TargetType), policy.TargetID, string(policy.WriteScope), "warning").Inc()
		}
	}
	return AdmissionDecision{Allowed: allowed, State: policy.CurrentState, Reason: reason, RetryAfter: time.Duration(policy.CheckIntervalSeconds) * time.Second, ObservedAt: observedAt}
}

func (s *Service) ListPolicies(ctx context.Context) ([]Policy, error) {
	if s == nil || s.repo == nil {
		return []Policy{}, nil
	}
	return s.repo.List(ctx)
}

func (s *Service) ListTargets(ctx context.Context) ([]TargetSnapshot, error) {
	if s == nil || s.repo == nil {
		return []TargetSnapshot{}, nil
	}
	policies, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list storage protection targets: %w", err)
	}
	type targetEntry struct {
		snapshot TargetSnapshot
		seen     map[WriteScope]struct{}
	}
	entries := make(map[string]*targetEntry)
	order := make([]string, 0)
	for _, policy := range policies {
		key := string(policy.TargetType) + "\x00" + policy.TargetID
		entry, ok := entries[key]
		if !ok {
			entry = &targetEntry{
				snapshot: TargetSnapshot{TargetType: policy.TargetType, TargetID: policy.TargetID, CurrentState: StateNormal},
				seen:     make(map[WriteScope]struct{}),
			}
			entries[key] = entry
			order = append(order, key)
		}
		if _, ok := entry.seen[policy.WriteScope]; !ok {
			entry.snapshot.WriteScopes = append(entry.snapshot.WriteScopes, policy.WriteScope)
			entry.seen[policy.WriteScope] = struct{}{}
		}
		if policy.CurrentState == StateBlocked || (policy.CurrentState == StateWarning && entry.snapshot.CurrentState != StateBlocked) || (policy.CurrentState == StateUnknown && entry.snapshot.CurrentState == StateNormal) {
			entry.snapshot.CurrentState = policy.CurrentState
		}
	}
	targets := make([]TargetSnapshot, 0, len(order))
	for _, key := range order {
		snapshot := entries[key].snapshot
		if s.usage != nil {
			usage, usageErr := s.usage.Snapshot(ctx, snapshot.TargetType, snapshot.TargetID)
			if usageErr != nil {
				snapshot.Reason = usageErr.Error()
			} else {
				snapshot.CapacityBytes = usage.CapacityBytes
				snapshot.UsedBytes = usage.UsedBytes
				snapshot.UsedRatio = usage.UsedRatio
				snapshot.Available = usage.Available
				snapshot.Reason = usage.Reason
				snapshot.ObservedAt = usage.ObservedAt
			}
		}
		targets = append(targets, snapshot)
	}
	return targets, nil
}

func (s *Service) ListEvents(ctx context.Context, targetType TargetType, targetID string, limit int) ([]Event, error) {
	if s == nil || s.repo == nil {
		return []Event{}, nil
	}
	return s.repo.ListEvents(ctx, targetType, targetID, limit)
}

func (s *Service) SavePolicy(ctx context.Context, policy *Policy) (*Policy, error) {
	if err := validatePolicy(policy); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("storage protection repository is not configured")
	}
	return s.repo.Save(ctx, policy)
}

func floatPtr(value float64) *float64    { return &value }
func timePtr(value time.Time) *time.Time { return &value }

type invalidPolicyError struct{ reason string }

func (e *invalidPolicyError) Error() string { return e.reason }
func errInvalidPolicy(reason string) error  { return &invalidPolicyError{reason: reason} }

func IsValidationError(err error) bool {
	_, ok := err.(*invalidPolicyError)
	return ok
}
