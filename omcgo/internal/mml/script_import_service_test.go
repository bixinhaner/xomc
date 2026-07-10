package mml

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeImportedScriptRepo struct {
	mu          sync.Mutex
	scripts     map[uuid.UUID]*MMLScript
	bySession   map[uuid.UUID]*MMLScript
	createErr   error
	replaceErr  error
	createCalls int
}

func newFakeImportedScriptRepo() *fakeImportedScriptRepo {
	return &fakeImportedScriptRepo{scripts: map[uuid.UUID]*MMLScript{}, bySession: map[uuid.UUID]*MMLScript{}}
}
func (r *fakeImportedScriptRepo) Create(ctx context.Context, s *MMLScript) error {
	return r.CreateImported(ctx, s)
}
func (r *fakeImportedScriptRepo) CreateImported(_ context.Context, s *MMLScript) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	if r.createErr != nil {
		return r.createErr
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = s.CreatedAt
	cp := *s
	r.scripts[s.ID] = &cp
	r.bySession[s.ImportSessionID] = &cp
	return nil
}
func (r *fakeImportedScriptRepo) GetByID(_ context.Context, id uuid.UUID) (*MMLScript, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.scripts[id]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (r *fakeImportedScriptRepo) GetByImportSessionID(_ context.Context, id uuid.UUID) (*MMLScript, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.bySession[id]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (r *fakeImportedScriptRepo) Update(_ context.Context, _ *MMLScript) error          { return nil }
func (r *fakeImportedScriptRepo) UpdateLifecycle(_ context.Context, _ *MMLScript) error { return nil }
func (r *fakeImportedScriptRepo) UpdateLastRun(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (r *fakeImportedScriptRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.scripts, id)
	return nil
}
func (r *fakeImportedScriptRepo) List(context.Context, ScriptFilter) (*model.ListResponse[MMLScript], error) {
	return nil, nil
}
func (r *fakeImportedScriptRepo) ReplaceImported(_ context.Context, s *MMLScript, expected time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.replaceErr != nil {
		return r.replaceErr
	}
	old, ok := r.scripts[s.ID]
	if !ok {
		return commonerrors.ErrNotFound
	}
	if !old.UpdatedAt.Equal(expected) {
		return ErrScriptVersionConflict
	}
	s.UpdatedAt = time.Now()
	cp := *s
	r.scripts[s.ID] = &cp
	r.bySession[s.ImportSessionID] = &cp
	return nil
}
func (r *fakeImportedScriptRepo) UpdateMetadata(_ context.Context, id uuid.UUID, name, description string, tags []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.scripts[id]
	if !ok {
		return commonerrors.ErrNotFound
	}
	s.ScriptName, s.Description, s.Tags = name, description, tags
	s.UpdatedAt = time.Now()
	return nil
}

type fakeImportValidator struct {
	result *ScriptValidationResult
	err    error
}

func (v *fakeImportValidator) Validate(context.Context, *ParsedScript, ValidationActor) (*ScriptValidationResult, error) {
	return v.result, v.err
}

type fakeImportSessions struct {
	session                                 *ImportSession
	putSession                              *ImportSession
	claimCalls, releaseCalls, finalizeCalls int
	claimErr                                error
	consumed                                bool
}

func (s *fakeImportSessions) Put(_ context.Context, _ string, session *ImportSession) (string, error) {
	s.putSession = session
	return "token", nil
}
func (s *fakeImportSessions) Get(_ context.Context, _ string, _ string) (*ImportSession, error) {
	if s.consumed {
		return nil, ErrImportTokenConsumed
	}
	return s.session, nil
}
func (s *fakeImportSessions) Claim(_ context.Context, _ string, _ string, _ string) (*ImportSession, error) {
	s.claimCalls++
	if s.consumed {
		return nil, ErrImportTokenConsumed
	}
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	return s.session, nil
}
func (s *fakeImportSessions) GetConsumed(context.Context, string, string) (*ImportSession, error) {
	if !s.consumed {
		return nil, ErrImportTokenExpired
	}
	return s.session, nil
}
func (s *fakeImportSessions) Release(context.Context, string, string, string) error {
	s.releaseCalls++
	return nil
}
func (s *fakeImportSessions) Finalize(context.Context, string, string, string) error {
	s.finalizeCalls++
	s.consumed = true
	return nil
}

func importedServiceSession() *ImportSession {
	return &ImportSession{ID: uuid.New(), OriginalFilename: "巡检.txt", NormalizedContent: "LST DEVICE_INFO;SN1\n", ContentSHA256: "sha-a", ValidationVersion: ValidationVersion, Validation: ScriptValidationResult{PlanItems: []MMLPlanItem{{LineNo: 1, DeviceSN: "SN1", Order: 1, CommandCode: "LST DEVICE_INFO"}}, Summary: ScriptValidationSummary{TotalLines: 1, ValidLines: 1, DeviceCount: 1}}}
}

func TestScriptImportService_ValidationErrorsDoNotCreateToken(t *testing.T) {
	sessions := &fakeImportSessions{}
	svc := NewScriptImportService(newFakeImportedScriptRepo(), &fakeImportValidator{}, sessions, zap.NewNop())
	response, err := svc.ValidateScriptImport(context.Background(), "alice", "bad.txt", []byte("not a command\n"))
	require.NoError(t, err)
	require.Empty(t, response.ValidationToken)
	require.NotEmpty(t, response.Issues)
	require.Nil(t, sessions.putSession)
}

func TestScriptImportService_ValidationWarningsStillCreateToken(t *testing.T) {
	sessions := &fakeImportSessions{}
	validator := &fakeImportValidator{result: &ScriptValidationResult{PlanItems: []MMLPlanItem{{LineNo: 1, DeviceSN: "SN1", Order: 1, CommandCode: "LST DEVICE_INFO"}}, Summary: ScriptValidationSummary{TotalLines: 1, ValidLines: 1, DeviceCount: 1}, Issues: []ScriptIssue{{Code: "MML_DEVICE_OFFLINE", Severity: IssueWarning, LineNo: 1}}}}
	svc := NewScriptImportService(newFakeImportedScriptRepo(), validator, sessions, zap.NewNop())
	response, err := svc.ValidateScriptImport(context.Background(), "alice", "ok.txt", []byte("LST DEVICE_INFO;SN1\n"))
	require.NoError(t, err)
	require.Equal(t, "token", response.ValidationToken)
	require.NotNil(t, sessions.putSession)
	require.Len(t, sessions.putSession.Validation.Issues, 1)
}

func TestScriptImportService_DoesNotConsumeTokenWhenRepositoryFails(t *testing.T) {
	sessions := &fakeImportSessions{session: importedServiceSession()}
	repo := newFakeImportedScriptRepo()
	repo.createErr = errors.New("pg unavailable")
	svc := NewScriptImportService(repo, &fakeImportValidator{}, sessions, zap.NewNop())
	_, err := svc.CreateScriptFromImport(context.Background(), "alice", SaveImportedScriptRequest{ValidationToken: "token", ScriptName: "巡检"})
	require.ErrorContains(t, err, "pg unavailable")
	require.Equal(t, 1, sessions.claimCalls)
	require.Equal(t, 1, sessions.releaseCalls)
	require.Equal(t, 0, sessions.finalizeCalls)
}

func TestScriptImportService_RejectsValidationErrorsAndMissingName(t *testing.T) {
	session := importedServiceSession()
	session.Validation.Issues = []ScriptIssue{{Code: "MML_COMMAND_NOT_FOUND", Severity: IssueError}}
	sessions := &fakeImportSessions{session: session}
	repo := newFakeImportedScriptRepo()
	svc := NewScriptImportService(repo, &fakeImportValidator{}, sessions, zap.NewNop())
	_, err := svc.CreateScriptFromImport(context.Background(), "alice", SaveImportedScriptRequest{ValidationToken: "token", ScriptName: "巡检"})
	require.Error(t, err)
	require.Equal(t, 0, repo.createCalls)
	require.Equal(t, 0, sessions.finalizeCalls)
	_, err = svc.CreateScriptFromImport(context.Background(), "alice", SaveImportedScriptRequest{ValidationToken: "token"})
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestScriptImportService_PersistsOnlyAuthoritativeSessionFields(t *testing.T) {
	sessions := &fakeImportSessions{session: importedServiceSession()}
	repo := newFakeImportedScriptRepo()
	svc := NewScriptImportService(repo, &fakeImportValidator{}, sessions, zap.NewNop())
	got, err := svc.CreateScriptFromImport(context.Background(), "alice", SaveImportedScriptRequest{ValidationToken: "token", ScriptName: "巡检", Description: "desc", Tags: []string{"a"}, RequestID: "req-1"})
	require.NoError(t, err)
	require.Equal(t, sessions.session.NormalizedContent, got.Content)
	require.Equal(t, sessions.session.ContentSHA256, got.ContentSHA256)
	require.Equal(t, sessions.session.Validation.PlanItems, got.PlanItems)
	require.Equal(t, "alice", got.Creator)
	require.Equal(t, 1, sessions.finalizeCalls)
}

func TestScriptImportService_ReplayIsIdempotent(t *testing.T) {
	sessions := &fakeImportSessions{session: importedServiceSession()}
	repo := newFakeImportedScriptRepo()
	svc := NewScriptImportService(repo, &fakeImportValidator{}, sessions, zap.NewNop())
	req := SaveImportedScriptRequest{ValidationToken: "token", ScriptName: "巡检", RequestID: "req-1"}
	one, err := svc.CreateScriptFromImport(context.Background(), "alice", req)
	require.NoError(t, err)
	sessions.consumed = false
	two, err := svc.CreateScriptFromImport(context.Background(), "alice", req)
	require.NoError(t, err)
	require.Equal(t, one.ID, two.ID)
	require.Equal(t, 1, repo.createCalls)
}

func TestScriptImportService_ReplayAfterConsumedTokenIsIdempotent(t *testing.T) {
	sessions := &fakeImportSessions{session: importedServiceSession()}
	repo := newFakeImportedScriptRepo()
	svc := NewScriptImportService(repo, &fakeImportValidator{}, sessions, zap.NewNop())
	req := SaveImportedScriptRequest{ValidationToken: "token", ScriptName: "巡检", RequestID: "req-1"}
	one, err := svc.CreateScriptFromImport(context.Background(), "alice", req)
	require.NoError(t, err)
	two, err := svc.CreateScriptFromImport(context.Background(), "alice", req)
	require.NoError(t, err)
	require.Equal(t, one.ID, two.ID)
	require.Equal(t, 1, repo.createCalls)
}

func TestScriptImportService_ReplaceUsesOptimisticVersion(t *testing.T) {
	sessions := &fakeImportSessions{session: importedServiceSession()}
	repo := newFakeImportedScriptRepo()
	svc := NewScriptImportService(repo, &fakeImportValidator{}, sessions, zap.NewNop())
	created, err := svc.CreateScriptFromImport(context.Background(), "alice", SaveImportedScriptRequest{ValidationToken: "token", ScriptName: "旧", RequestID: "req-1"})
	require.NoError(t, err)
	sessions.consumed = false
	_, err = svc.ReplaceScriptFromImport(context.Background(), created.ID, "alice", ReplaceImportedScriptRequest{ValidationToken: "token", ScriptName: "新", ExpectedUpdatedAt: created.UpdatedAt, RequestID: "req-2"})
	require.NoError(t, err)
	sessions.consumed = false
	_, err = svc.ReplaceScriptFromImport(context.Background(), created.ID, "alice", ReplaceImportedScriptRequest{ValidationToken: "token", ScriptName: "再次", ExpectedUpdatedAt: created.UpdatedAt, RequestID: "req-3"})
	require.ErrorIs(t, err, ErrScriptVersionConflict)
}
