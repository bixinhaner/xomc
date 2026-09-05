package agentassistant

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/authz"
	"go.uber.org/zap"
)

// Remote is deliberately small: a generic planner and immutable run snapshots.
// Assistant ownership, publish transactions and business triggers stay local.
type Remote interface {
	AssistantConnection(context.Context) (string, error)
	PlanAssistant(context.Context, PlanRequest) (PlanResponse, error)
	SubmitAssistant(context.Context, string, ExecutionRequest) (RemoteRun, error)
	GetAssistantRun(context.Context, string, string) (RemoteRun, error)
	CancelAssistantRun(context.Context, string, string) error
}
type Service struct {
	repo        *Repository
	remote      Remote
	provider    CapabilityProvider
	permissions PermissionChecker
	groups      authz.VisibleGroupsResolver
	devices     authz.GroupReader
	logger      *zap.Logger
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	startOnce   sync.Once
}

func NewService(repo *Repository, remote Remote, provider CapabilityProvider, permissions PermissionChecker, groups authz.VisibleGroupsResolver, devices authz.GroupReader, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: repo, remote: remote, provider: provider, permissions: permissions, groups: groups, devices: devices, logger: logger.Named("assistant")}
}

type Catalog struct {
	Capabilities    []Capability      `json:"capabilities"`
	Events          []EventCapability `json:"events"`
	Connected       bool              `json:"connected"`
	ConnectionError string            `json:"connectionError,omitempty"`
}

func (s *Service) Catalog(ctx context.Context, c *admin.Claims) (Catalog, error) {
	caps, err := s.capabilities(ctx, principalFromClaims(c))
	if err != nil {
		return Catalog{}, err
	}
	_, connectionErr := s.remote.AssistantConnection(ctx)
	out := Catalog{Capabilities: caps, Events: Events, Connected: connectionErr == nil}
	if connectionErr != nil {
		out.ConnectionError = "ASSISTANT_NOT_CONNECTED"
	}
	return out, nil
}
func (s *Service) Create(ctx context.Context, c *admin.Claims, id, locale, timezone string) (Assistant, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Assistant{}, fmt.Errorf("ASSISTANT_INVALID_REQUEST")
	}
	if locale == "" {
		locale = "en-US"
	}
	if timezone == "" {
		timezone = "UTC"
	}
	if len(locale) > 30 {
		return Assistant{}, fmt.Errorf("ASSISTANT_INVALID_REQUEST")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return Assistant{}, fmt.Errorf("ASSISTANT_INVALID_TIMEZONE")
	}
	p := principalFromClaims(c)
	if _, err := s.repo.CurrentPrincipal(ctx, p); err != nil {
		return Assistant{}, err
	}
	existing, err := s.repo.Get(ctx, p.UserID, id)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Assistant{}, err
	}
	list, err := s.repo.List(ctx, p.UserID)
	if err != nil {
		return Assistant{}, err
	}
	if len(list) >= 100 {
		return Assistant{}, fmt.Errorf("ASSISTANT_LIMIT_REACHED")
	}
	return s.repo.Create(ctx, p.UserID, p.RoleID, id, locale, timezone)
}
func (s *Service) Plan(ctx context.Context, c *admin.Claims, id string, revision int, message string) (Assistant, error) {
	if strings.TrimSpace(message) == "" || len([]rune(message)) > 8000 {
		return Assistant{}, fmt.Errorf("ASSISTANT_INVALID_REQUEST")
	}
	a, err := s.repo.Get(ctx, c.UserID.String(), id)
	if err != nil {
		return a, err
	}
	if a.Revision != revision {
		return a, ErrConflict
	}
	p := principalFromClaims(c)
	caps, err := s.capabilities(ctx, p)
	if err != nil {
		return a, err
	}
	if len(caps) == 0 {
		return a, fmt.Errorf("ASSISTANT_NO_CAPABILITIES")
	}
	messages := a.Messages
	if len(messages) > 18 {
		messages = messages[len(messages)-18:]
	}
	// The draft is not modified until a valid planning response is received. A
	// timeout leaves the user's saved draft and active published version intact.
	plan, err := s.remote.PlanAssistant(ctx, PlanRequest{Message: message, Definition: a.Definition, Messages: messages, Capabilities: caps, Events: Events, Locale: a.Locale, Timezone: a.Timezone, ExternalUserID: p.UserID})
	if err != nil {
		return a, fmt.Errorf("ASSISTANT_PLANNING_FAILED: %w", err)
	}
	if plan.Readiness != "ready" && plan.Readiness != "needs_input" && plan.Readiness != "unsupported" {
		return a, fmt.Errorf("ASSISTANT_INVALID_MODEL_OUTPUT")
	}
	if plan.Readiness == "ready" {
		if len(plan.Questions) > 0 || len(plan.MissingCapabilities) > 0 {
			return a, fmt.Errorf("ASSISTANT_PLAN_NOT_READY")
		}
		if err = s.validate(ctx, p, plan.Definition); err != nil {
			return a, err
		}
	}
	a.Definition = plan.Definition
	a.Readiness = plan.Readiness
	a.Questions = plan.Questions
	a.MissingCapabilities = plan.MissingCapabilities
	a.RoleID = p.RoleID
	a.Messages = append(messages, Message{Role: "user", Text: message}, Message{Role: "assistant", Text: plan.Reply})
	return s.repo.SaveDraft(ctx, a, revision)
}
func (s *Service) SaveDefinition(ctx context.Context, c *admin.Claims, id string, revision int, d Definition) (Assistant, error) {
	a, err := s.repo.Get(ctx, c.UserID.String(), id)
	if err != nil {
		return a, err
	}
	if a.Revision != revision {
		return a, ErrConflict
	}
	p := principalFromClaims(c)
	if err = s.validate(ctx, p, &d); err != nil {
		return a, err
	}
	a.Definition = &d
	a.Readiness = "ready"
	a.Questions = []string{}
	a.MissingCapabilities = []string{}
	a.RoleID = p.RoleID
	return s.repo.SaveDraft(ctx, a, revision)
}
func (s *Service) Publish(ctx context.Context, c *admin.Claims, id string, revision int) (Assistant, error) {
	a, err := s.repo.Get(ctx, c.UserID.String(), id)
	if err != nil {
		return a, err
	}
	if a.Revision != revision {
		return a, ErrConflict
	}
	if _, err = s.remote.AssistantConnection(ctx); err != nil {
		return a, err
	}
	if err = s.validate(ctx, Principal{UserID: a.OwnerID, RoleID: a.RoleID}, a.Definition); err != nil {
		return a, err
	}
	p, err := s.bindPrincipal(ctx, Principal{UserID: a.OwnerID, RoleID: a.RoleID})
	if err != nil {
		return a, err
	}
	return s.repo.Publish(ctx, a.OwnerID, id, revision, p)
}
func (s *Service) State(ctx context.Context, c *admin.Claims, id, state string) (Assistant, error) {
	if state != "active" && state != "paused" {
		return Assistant{}, fmt.Errorf("ASSISTANT_INVALID_REQUEST")
	}
	a, err := s.repo.Get(ctx, c.UserID.String(), id)
	if err != nil {
		return a, err
	}
	if state == "active" {
		if a.PublishedRevision == nil || a.PublishedDefinition == nil {
			return a, ErrTrialRequired
		}
		p, _, _, err := s.repo.VersionPrincipal(ctx, id, *a.PublishedRevision)
		if err != nil {
			return a, err
		}
		if err = s.validate(ctx, p, a.PublishedDefinition); err != nil {
			return a, err
		}
		if _, err = s.remote.AssistantConnection(ctx); err != nil {
			return a, err
		}
	}
	return s.repo.SetState(ctx, a.OwnerID, id, state)
}
func (s *Service) StartRun(ctx context.Context, c *admin.Claims, id string, revision int, kind, requestID string) (Run, error) {
	if kind != "trial" && kind != "manual" {
		return Run{}, fmt.Errorf("ASSISTANT_INVALID_REQUEST")
	}
	if _, err := uuid.Parse(requestID); err != nil {
		return Run{}, fmt.Errorf("ASSISTANT_INVALID_REQUEST")
	}
	a, err := s.repo.Get(ctx, c.UserID.String(), id)
	if err != nil {
		return Run{}, err
	}
	if kind == "trial" && a.Revision != revision {
		return Run{}, ErrConflict
	}
	if kind == "trial" && a.Readiness != "ready" {
		return Run{}, fmt.Errorf("ASSISTANT_PLAN_NOT_READY")
	}
	return s.enqueue(ctx, a, kind, kind+":"+fmt.Sprint(revision)+":"+requestID, map[string]any{"kind": kind}, false)
}
func (s *Service) enqueue(ctx context.Context, a Assistant, kind, key string, trigger map[string]any, scheduled bool) (Run, error) {
	d := a.Definition
	p := Principal{UserID: a.OwnerID, RoleID: a.RoleID}
	revision := a.Revision
	locale, tz := a.Locale, a.Timezone
	if kind != "trial" {
		if a.PublishedRevision == nil || a.PublishedDefinition == nil || a.State != "active" {
			return Run{}, ErrTrialRequired
		}
		d = a.PublishedDefinition
		revision = *a.PublishedRevision
		var err error
		p, locale, tz, err = s.repo.VersionPrincipal(ctx, a.ID, revision)
		if err != nil {
			return Run{}, err
		}
	}
	if err := s.validate(ctx, p, d); err != nil {
		return Run{}, err
	}
	var err error
	p, err = s.bindPrincipal(ctx, p)
	if err != nil {
		return Run{}, err
	}
	connector, err := s.remote.AssistantConnection(ctx)
	if err != nil {
		return Run{}, err
	}
	metadata, err := s.provider.HandbookMetadata()
	if err != nil {
		return Run{}, fmt.Errorf("ASSISTANT_CAPABILITIES_UNAVAILABLE: %w", err)
	}
	digest, ok := metadata["handbookDigest"].(string)
	if !ok || digest == "" {
		return Run{}, fmt.Errorf("ASSISTANT_CAPABILITIES_UNAVAILABLE")
	}
	run := Run{ID: uuid.NewString(), AssistantID: a.ID, OwnerID: a.OwnerID, Revision: revision, Kind: kind, ConnectorID: connector, Principal: p}
	run.Request = ExecutionRequest{ContractVersion: "1.0", RunID: run.ID, AssistantID: a.ID, Revision: revision, Definition: *d, DefinitionDigest: DefinitionDigest(*d), HandbookDigest: digest, APIHandbook: metadata, Locale: locale, Timezone: tz, ExternalUserID: a.OwnerID, TriggerContext: trigger, Limits: Limits{TimeoutSeconds: 120, MaxToolCalls: 18, MaxOutputBytes: 32768}}
	return s.repo.Enqueue(ctx, a, run, key, scheduled)
}
func (s *Service) Runs(ctx context.Context, c *admin.Claims, id string) ([]Run, error) {
	runs, err := s.repo.Runs(ctx, c.UserID.String(), id)
	if err != nil {
		return nil, err
	}
	for i := range runs {
		if err = s.checkScope(ctx, runs[i].Principal, runs[i].Request.Definition.Scope); err != nil {
			runs[i].Output = nil
			runs[i].Tools = []ToolProgress{}
			runs[i].ErrorCode = "ASSISTANT_RESULT_ACCESS_REVOKED"
			runs[i].ErrorMessage = ""
		}
	}
	return runs, nil
}
func (s *Service) Run(ctx context.Context, c *admin.Claims, id string) (Run, error) {
	run, err := s.repo.GetRun(ctx, c.UserID.String(), id)
	if err != nil {
		return run, err
	}
	if err = s.checkScope(ctx, run.Principal, run.Request.Definition.Scope); err != nil {
		return Run{}, err
	}
	return run, nil
}
func (s *Service) OnEvent(ctx context.Context, event Event) error {
	if _, err := uuid.Parse(event.DeviceID); err != nil {
		return nil
	} // no authoritative resource, no analysis
	items, err := s.repo.Published(ctx, event.Type, false)
	if err != nil {
		return err
	}
	var retry error
	for _, a := range items {
		if a.PublishedDefinition == nil || a.PublishedRevision == nil || a.PublishedAt == nil || event.OccurredAt.Before(*a.PublishedAt) {
			continue
		}
		d := a.PublishedDefinition
		if !Matches(d.Trigger.Conditions, event.Data) || (d.Scope.Kind == "device" && d.Scope.DeviceID != event.DeviceID) {
			continue
		}
		p, _, _, err := s.repo.VersionPrincipal(ctx, a.ID, *a.PublishedRevision)
		if err != nil {
			continue
		}
		// The event must be visible to the creator before its payload leaves xOMC.
		if err = s.checkScope(ctx, p, Scope{Kind: "device", DeviceID: event.DeviceID}); err != nil {
			continue
		}
		key := fmt.Sprintf("event:%d:%s", *a.PublishedRevision, event.ID)
		_, err = s.enqueue(ctx, a, "event", key, map[string]any{"kind": "event", "eventId": event.ID, "eventType": event.Type, "data": event.Data, "deviceId": event.DeviceID}, false)
		if err != nil {
			if strings.Contains(err.Error(), "ALREADY_RUNNING") {
				retry = err
				continue
			}
			if isPolicyError(err) {
				_ = s.repo.Block(ctx, a.ID, *a.PublishedRevision, err)
			} else {
				retry = err
			}
		}
	}
	return retry
}
func isPolicyError(err error) bool {
	for _, key := range []string{"OWNER_UNAVAILABLE", "ROLE_UNAVAILABLE", "SCOPE_FORBIDDEN", "SCOPE_CHANGED", "SCOPE_NOT_RESOLVED", "UNKNOWN_CAPABILITY", "CAPABILITY_SCOPE_MISMATCH"} {
		if strings.Contains(err.Error(), key) {
			return true
		}
	}
	return false
}
func (s *Service) Start() {
	s.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		s.wg.Add(2)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				s.schedule(ctx)
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					s.poll(ctx)
				}
			}
		}()
	})
}
func (s *Service) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	return nil
}
func (s *Service) schedule(ctx context.Context) {
	items, err := s.repo.Published(ctx, "", true)
	if err != nil {
		if ctx.Err() == nil {
			s.logger.Warn("list due assistants", zap.Error(err))
		}
		return
	}
	for _, a := range items {
		if ctx.Err() != nil {
			return
		}
		if a.NextRunAt == nil || a.PublishedRevision == nil {
			continue
		}
		key := fmt.Sprintf("schedule:%d:%s", *a.PublishedRevision, a.NextRunAt.UTC().Format(time.RFC3339))
		_, err = s.enqueue(ctx, a, "schedule", key, map[string]any{"kind": "schedule", "scheduledAt": a.NextRunAt.UTC().Format(time.RFC3339)}, true)
		if err != nil && isPolicyError(err) {
			_ = s.repo.Block(ctx, a.ID, *a.PublishedRevision, err)
		}
		if err != nil && !strings.Contains(err.Error(), "ALREADY_RUNNING") {
			s.logger.Warn("schedule assistant", zap.String("assistant_id", a.ID), zap.Error(err))
		}
	}
}
func (s *Service) poll(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	run, err := s.repo.Claim(ctx)
	if errors.Is(err, ErrNotFound) {
		return
	}
	if err != nil {
		if parent.Err() == nil {
			s.logger.Warn("claim assistant run", zap.Error(err))
		}
		return
	}
	var remote RemoteRun
	if run.Status == "CANCELLING" {
		_, err = s.remote.SubmitAssistant(ctx, run.ConnectorID, run.Request)
		if err == nil {
			err = s.remote.CancelAssistantRun(ctx, run.ConnectorID, run.ID)
		}
		if err == nil {
			// Cancellation can race completion. The remote ledger is authoritative;
			// a successful cancel request does not imply a CANCELLED terminal state.
			remote, err = s.remote.GetAssistantRun(ctx, run.ConnectorID, run.ID)
		}
	} else {
		if err = s.validate(ctx, run.Principal, &run.Request.Definition); err == nil {
			// Submit is idempotent, including after an ambiguous response or restart.
			_, err = s.remote.SubmitAssistant(ctx, run.ConnectorID, run.Request)
			if err == nil {
				remote, err = s.remote.GetAssistantRun(ctx, run.ConnectorID, run.ID)
			}
		}
	}
	if parent.Err() != nil {
		return
	} // release by expiry, never pretend a shutdown failed the run
	if err != nil {
		code := "ASSISTANT_CONNECTION_RETRY"
		status := "RUNNING"
		if isPolicyError(err) || strings.Contains(err.Error(), "CONFLICT") || strings.Contains(err.Error(), "CONNECTOR_CHANGED") || time.Since(run.CreatedAt) > 15*time.Minute {
			status = "FAILED"
			code = "ASSISTANT_RUN_FAILED"
		}
		if run.Status == "CANCELLING" {
			status = "CANCELLING"
			code = "ASSISTANT_CANCEL_PENDING"
		}
		remote = RemoteRun{ID: run.ID, Status: status, Error: &RemoteError{Code: code, Message: err.Error(), Retryable: status != "FAILED"}, Tools: run.Tools}
	}
	if remote.Tools == nil {
		remote.Tools = []ToolProgress{}
	}
	saveCtx, saveCancel := context.WithTimeout(parent, 3*time.Second)
	defer saveCancel()
	if e := s.repo.Progress(saveCtx, run, remote, 2*time.Second); e != nil {
		s.logger.Warn("save assistant run", zap.String("run_id", run.ID), zap.Error(e))
	}
}
