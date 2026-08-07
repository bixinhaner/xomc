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
	logMu   sync.RWMutex
	logGate LogAdmissionController
	cancel  context.CancelFunc
	done    chan struct{}
}

// LogAdmissionController is implemented by the process logger gate. Keeping
// this small interface here avoids coupling the storage-protection domain to a
// concrete logging package while allowing app/acs/worker to share identical
// behavior.
type LogAdmissionController interface {
	SetBlocked(bool)
}

// SetLogAdmissionController connects storage state to the service logger. The
// controller is intentionally optional so storage-protection unit tests and
// command-line tools can use the service without a logger gate.
func (s *Service) SetLogAdmissionController(controller LogAdmissionController) {
	if s == nil {
		return
	}
	s.logMu.Lock()
	s.logGate = controller
	s.logMu.Unlock()
}

func (s *Service) setLogAdmissionBlocked(blocked bool) {
	if s == nil {
		return
	}
	s.logMu.RLock()
	controller := s.logGate
	s.logMu.RUnlock()
	if controller != nil {
		controller.SetBlocked(blocked)
	}
}

// syncLogAdmission mirrors the physical storage state into the process-local
// log gate. Unknown is fail-closed for logs: if the service cannot establish
// that the filesystem is safe, continuing to emit unbounded logs could make a
// full disk worse. A normal/warning state re-opens the gate.
func (s *Service) syncLogAdmission(policy *Policy) {
	if policy == nil {
		s.setLogAdmissionBlocked(false)
		return
	}
	s.setLogAdmissionBlocked(policy.CurrentState == StateBlocked || policy.CurrentState == StateUnknown)
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
	hasEnabledPolicy := false
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}
		hasEnabledPolicy = true
		if _, err := s.Check(ctx, policy.TargetType, policy.TargetID, policy.WriteScope); err != nil {
			s.logger.Warn("evaluate storage protection policy failed", zap.String("policy_id", policy.ID), zap.Error(err))
		}
	}
	if !hasEnabledPolicy {
		// A policy can be disabled or deleted while a previous state was
		// blocked. Do not leave the process logger permanently closed.
		s.setLogAdmissionBlocked(false)
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
	// Business callers may still identify the destination as MinIO or another
	// logical component. Those components share one physical filesystem in the
	// current deployment, so every admission check uses the canonical target.
	targetType, targetID = TargetFilesystem, UnifiedStorageTargetID
	if s == nil || s.repo == nil {
		if s != nil {
			s.setLogAdmissionBlocked(false)
		}
		return AdmissionDecision{Allowed: true, State: StateNormal, Reason: "storage protection is not configured", ObservedAt: time.Now()}, nil
	}
	policy, err := s.repo.GetEnabledPolicy(ctx, targetType, targetID, scope)
	if err != nil {
		return AdmissionDecision{}, fmt.Errorf("load storage protection policy: %w", err)
	}
	if policy == nil {
		s.setLogAdmissionBlocked(false)
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
	reason = usageReason(reason, usage)
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
	s.syncLogAdmission(policy)
	if policy.CurrentState != previous {
		event := Event{PolicyID: policy.ID, TargetType: targetType, TargetID: targetID, WriteScope: scope, PreviousState: previous, NewState: policy.CurrentState, Reason: transitionReason(previous, policy.CurrentState, reason), ObservedRatio: floatPtr(usage.UsedRatio), PolicyVersion: policy.Version, OperatorID: "system", CreatedAt: s.now()}
		if err := s.repo.RecordEvent(ctx, event); err != nil {
			return AdmissionDecision{}, fmt.Errorf("record storage protection transition: %w", err)
		}
	}
	return s.decision(policy, scope, usage.ObservedAt, reason), nil
}

func usageReason(reason string, usage UsageSnapshot) string {
	if usage.Reason == "" {
		return reason
	}
	if reason == "" {
		return usage.Reason
	}
	return reason + "; " + usage.Reason
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
	s.syncLogAdmission(policy)
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
	return s.unknownDecision(policy, scope, reason), nil
}

func (s *Service) unknownDecision(policy *Policy, scope WriteScope, reason string) AdmissionDecision {
	if s.metrics != nil {
		s.metrics.AdmissionState.WithLabelValues(string(policy.TargetType), policy.TargetID, string(scope)).Set(stateValue(StateUnknown))
	}
	allowed := policy.UnknownBehavior == UnknownAllowWithAlarm
	if !allowed && s.metrics != nil {
		s.metrics.WriteRejectedTotal.WithLabelValues(string(policy.TargetType), policy.TargetID, string(scope), "unknown").Inc()
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

func (s *Service) decision(policy *Policy, scope WriteScope, observedAt time.Time, reason string) AdmissionDecision {
	s.syncLogAdmission(policy)
	allowed := policy.CurrentState != StateBlocked
	if policy.CurrentState == StateUnknown {
		allowed = policy.UnknownBehavior == UnknownAllowWithAlarm
	}
	if s.metrics != nil {
		labels := []string{string(policy.TargetType), policy.TargetID, string(scope)}
		s.metrics.AdmissionState.WithLabelValues(labels...).Set(stateValue(policy.CurrentState))
		if policy.CurrentState == StateBlocked {
			s.metrics.WriteRejectedTotal.WithLabelValues(string(policy.TargetType), policy.TargetID, string(scope), "blocked").Inc()
		} else if policy.CurrentState == StateWarning {
			s.metrics.WriteDegradedTotal.WithLabelValues(string(policy.TargetType), policy.TargetID, string(scope), "warning").Inc()
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
	writeScopes := make([]WriteScope, 0)
	seenScopes := make(map[WriteScope]struct{})
	var globalPolicy *Policy
	for _, policy := range policies {
		if policy.TargetType != TargetFilesystem || policy.TargetID != UnifiedStorageTargetID {
			continue
		}
		if _, ok := seenScopes[policy.WriteScope]; !ok {
			writeScopes = append(writeScopes, policy.WriteScope)
			seenScopes[policy.WriteScope] = struct{}{}
		}
		if policy.Enabled && (globalPolicy == nil || policy.WriteScope == WriteScopeAll) {
			copy := policy
			globalPolicy = &copy
		}
	}
	usageTargets := []UsageSnapshot{{TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, Mountpoint: UnifiedStorageMountpoint, Available: false, Reason: "storage usage provider is unavailable"}}
	if lister, ok := s.usage.(UsageTargetLister); ok {
		usageTargets, err = lister.ListTargets(ctx)
		if err != nil {
			return nil, fmt.Errorf("list storage protection usage targets: %w", err)
		}
	} else if s.usage != nil {
		usage, usageErr := s.usage.Snapshot(ctx, TargetFilesystem, UnifiedStorageTargetID)
		if usageErr != nil {
			usage = UsageSnapshot{TargetType: TargetFilesystem, TargetID: UnifiedStorageTargetID, Mountpoint: UnifiedStorageMountpoint, Available: false, Reason: usageErr.Error()}
		}
		usageTargets = []UsageSnapshot{usage}
	}

	targets := make([]TargetSnapshot, 0, len(usageTargets))
	for _, usage := range usageTargets {
		targets = append(targets, TargetSnapshot{
			TargetType: usage.TargetType, TargetID: usage.TargetID, Mountpoint: usage.Mountpoint, ProtectedPaths: append([]string(nil), usage.ProtectedPaths...),
			CapacityBytes: usage.CapacityBytes, UsedBytes: usage.UsedBytes, UsedRatio: usage.UsedRatio,
			Available: usage.Available, Reason: usage.Reason, ObservedAt: usage.ObservedAt,
			CurrentState: targetState(globalPolicy, usage), WriteScopes: append([]WriteScope(nil), writeScopes...),
		})
	}
	return targets, nil
}

func targetState(policy *Policy, usage UsageSnapshot) State {
	if policy == nil || !policy.Enabled {
		return StateNormal
	}
	if !usage.Available {
		return StateUnknown
	}
	if usage.UsedRatio >= float64(policy.BlockUsedPercent)/100 {
		return StateBlocked
	}
	if usage.UsedRatio >= float64(policy.WarnUsedPercent)/100 {
		return StateWarning
	}
	return StateNormal
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
