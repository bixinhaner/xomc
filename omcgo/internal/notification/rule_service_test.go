package notification

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type ruleRepositoryStub struct {
	rule                  *NotificationRule
	createInput           RuleDraftInput
	updateRevision        int64
	publishRevision       int64
	enableRevision        int64
	enableVersionID       uuid.UUID
	disableRevision       int64
	archiveRevision       int64
	savePublishedID       uuid.UUID
	savePublishedRevision int64
	savePublishedEnabled  bool
	err                   error
}

func (s *ruleRepositoryStub) List(context.Context) ([]NotificationRule, error) {
	if s.rule == nil {
		return nil, s.err
	}
	return []NotificationRule{*s.rule}, s.err
}
func (s *ruleRepositoryStub) Get(context.Context, uuid.UUID) (*NotificationRule, error) {
	return s.rule, s.err
}
func (s *ruleRepositoryStub) Create(_ context.Context, input RuleDraftInput, _ string) (*NotificationRule, error) {
	s.createInput = input
	return s.rule, s.err
}
func (s *ruleRepositoryStub) UpdateDraft(_ context.Context, _ uuid.UUID, revision int64, _ RuleDraftInput, _ string) (*NotificationRule, error) {
	s.updateRevision = revision
	return s.rule, s.err
}
func (s *ruleRepositoryStub) Publish(_ context.Context, _ uuid.UUID, revision int64, _ string) (*NotificationRule, error) {
	s.publishRevision = revision
	return s.rule, s.err
}
func (s *ruleRepositoryStub) Enable(_ context.Context, _ uuid.UUID, revision int64, versionID uuid.UUID) (*NotificationRule, error) {
	s.enableRevision, s.enableVersionID = revision, versionID
	return s.rule, s.err
}
func (s *ruleRepositoryStub) Disable(_ context.Context, _ uuid.UUID, revision int64) (*NotificationRule, error) {
	s.disableRevision = revision
	return s.rule, s.err
}
func (s *ruleRepositoryStub) Archive(_ context.Context, _ uuid.UUID, revision int64) (*NotificationRule, error) {
	s.archiveRevision = revision
	return s.rule, s.err
}
func (s *ruleRepositoryStub) SavePublished(
	_ context.Context,
	id uuid.UUID,
	revision int64,
	_ RuleDraftInput,
	enabled bool,
	_ string,
) (*NotificationRule, error) {
	s.savePublishedID = id
	s.savePublishedRevision = revision
	s.savePublishedEnabled = enabled
	return s.rule, s.err
}

func TestRuleService_CreateNormalizesDraftWithoutInventingPolicy(t *testing.T) {
	repository := &ruleRepositoryStub{rule: &NotificationRule{ID: uuid.New()}}
	service := NewRuleService(repository)

	_, err := service.Create(context.Background(), RuleDraftInput{Name: "  critical alarms  "}, "operator")
	require.NoError(t, err)
	require.Equal(t, "critical alarms", repository.createInput.Name)
	require.Equal(t, 100, repository.createInput.Priority)
	require.JSONEq(t, `{}`, string(repository.createInput.Policy))
}

func TestRuleService_RejectsInvalidDraft(t *testing.T) {
	service := NewRuleService(&ruleRepositoryStub{})
	_, err := service.Create(context.Background(), RuleDraftInput{Name: "", Policy: json.RawMessage(`{}`)}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	_, err = service.Create(context.Background(), RuleDraftInput{Name: "rule", Policy: json.RawMessage(`not-json`)}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestRuleService_PropagatesRevisionAndPublicationInvariants(t *testing.T) {
	wantErr := ErrRevisionMismatch
	repository := &ruleRepositoryStub{err: wantErr}
	service := NewRuleService(repository)
	ruleID, versionID := uuid.New(), uuid.New()

	_, err := service.UpdateDraft(context.Background(), ruleID, 7, RuleDraftInput{Name: "rule"}, "operator")
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, int64(7), repository.updateRevision)
	_, err = service.Publish(context.Background(), ruleID, 8, "operator")
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, int64(8), repository.publishRevision)
	_, err = service.Enable(context.Background(), ruleID, 9, versionID)
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, int64(9), repository.enableRevision)
	require.Equal(t, versionID, repository.enableVersionID)
	_, err = service.Disable(context.Background(), ruleID, 10)
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, int64(10), repository.disableRevision)
	_, err = service.Archive(context.Background(), ruleID, 11)
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, int64(11), repository.archiveRevision)
}

func TestRuleService_WrapsRepositoryErrors(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service := NewRuleService(&ruleRepositoryStub{err: wantErr})
	_, err := service.List(context.Background())
	require.ErrorIs(t, err, wantErr)
}

func TestRuleService_SavePublishedDelegatesOneAtomicMutation(t *testing.T) {
	ruleID := uuid.New()
	repository := &ruleRepositoryStub{rule: &NotificationRule{ID: ruleID}}
	service := NewRuleService(repository)

	_, err := service.SavePublished(context.Background(), ruleID, 7, RuleDraftInput{Name: "alarm mail"}, true, "operator")
	require.NoError(t, err)
	require.Equal(t, ruleID, repository.savePublishedID)
	require.Equal(t, int64(7), repository.savePublishedRevision)
	require.True(t, repository.savePublishedEnabled)
}
