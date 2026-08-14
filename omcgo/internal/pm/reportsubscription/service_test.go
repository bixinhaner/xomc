package reportsubscription

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/querytemplate"
)

type serviceRepoStub struct {
	inputs       []UpsertInput
	subscription *Subscription
	runs         []Run
}

func (s *serviceRepoStub) GetByTemplate(context.Context, uuid.UUID) (*Subscription, error) {
	if s.subscription == nil {
		return nil, ErrNotFound
	}
	return s.subscription, nil
}
func (s *serviceRepoStub) Upsert(_ context.Context, templateID uuid.UUID, input UpsertInput) (*Subscription, error) {
	s.inputs = append(s.inputs, input)
	return &Subscription{QueryTemplateID: templateID, TimezoneName: input.TimezoneName, NextRunAt: input.NextRunAt}, nil
}
func (s *serviceRepoStub) Delete(context.Context, uuid.UUID) error { return nil }
func (s *serviceRepoStub) ListRuns(context.Context, uuid.UUID, int) ([]Run, error) {
	return s.runs, nil
}
func (s *serviceRepoStub) GetRun(context.Context, uuid.UUID) (*Run, error) { return nil, ErrNotFound }
func (s *serviceRepoStub) MarkRunRunning(context.Context, uuid.UUID) error { return nil }
func (s *serviceRepoStub) MarkRunExportTask(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (s *serviceRepoStub) MarkRunSent(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}
func (s *serviceRepoStub) MarkRunFailed(context.Context, uuid.UUID, string, string) error {
	return nil
}

type templateRepoStub struct {
	template *querytemplate.Template
}

func (s *templateRepoStub) Create(context.Context, querytemplate.CreateRequest) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (s *templateRepoStub) Get(context.Context, uuid.UUID) (*querytemplate.Template, error) {
	return s.template, nil
}
func (s *templateRepoStub) List(context.Context, querytemplate.ListFilter) ([]querytemplate.Template, int, error) {
	return nil, 0, nil
}
func (s *templateRepoStub) Update(context.Context, uuid.UUID, querytemplate.UpdateRequest) error {
	return nil
}
func (s *templateRepoStub) Delete(context.Context, uuid.UUID) error { return nil }

type mutableTimezoneProvider struct {
	location *time.Location
}

func (p *mutableTimezoneProvider) Location(context.Context) *time.Location { return p.location }

func TestServiceUpsertUsesCurrentTimezoneOnEverySave(t *testing.T) {
	t.Parallel()
	templateID := uuid.New()
	userID := uuid.New()
	repo := &serviceRepoStub{}
	timezone := &mutableTimezoneProvider{location: time.FixedZone("UTC+8", 8*60*60)}
	service := NewService(repo, &templateRepoStub{template: &querytemplate.Template{
		ID: templateID, Visibility: querytemplate.VisibilityPrivate, CreatorID: userID,
	}}, timezone)
	service.now = func() time.Time { return time.Date(2026, 8, 11, 0, 30, 0, 0, time.UTC) }
	input := UpsertInput{
		Enabled: true, Period: PeriodDaily, SendTimes: []string{"08:00"}, Recipients: []string{"noc@example.com"},
	}

	_, err := service.Upsert(context.Background(), templateID, userID, false, input)
	require.NoError(t, err)
	require.Equal(t, "UTC+8", repo.inputs[0].TimezoneName)
	require.Equal(t, time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), repo.inputs[0].NextRunAt.UTC())

	timezone.location = time.FixedZone("UTC-5", -5*60*60)
	_, err = service.Upsert(context.Background(), templateID, userID, false, input)
	require.NoError(t, err)
	require.Equal(t, "UTC-5", repo.inputs[1].TimezoneName)
	require.Equal(t, time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC), repo.inputs[1].NextRunAt.UTC())
}

func TestServicePublicTemplateReaderGetsRedactedReportData(t *testing.T) {
	t.Parallel()
	ownerID := uuid.New()
	readerID := uuid.New()
	templateID := uuid.New()
	lastError := "smtp authentication failed"
	repo := &serviceRepoStub{
		subscription: &Subscription{
			ID: uuid.New(), QueryTemplateID: templateID, Period: PeriodDaily,
			Recipients: []string{"noc@example.com"}, LastError: &lastError,
		},
		runs: []Run{{
			ID: uuid.New(), QueryTemplateID: templateID, QueryTemplateName: "public",
			QueryPayload: []byte(`{"device_sns":["private-device"]}`),
			Recipients:   []string{"noc@example.com"}, Status: RunStatusDeliveryFailed,
			ErrorMessage: &lastError,
		}},
	}
	service := NewService(repo, &templateRepoStub{template: &querytemplate.Template{
		ID: templateID, Visibility: querytemplate.VisibilityPublic, CreatorID: ownerID,
	}}, nil)

	subscription, err := service.Get(context.Background(), templateID, readerID, false)
	require.NoError(t, err)
	require.Empty(t, subscription.Recipients)
	require.Nil(t, subscription.LastError)

	runs, err := service.ListRuns(context.Background(), templateID, readerID, false, 20)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Empty(t, runs[0].Recipients)
	require.Nil(t, runs[0].ErrorMessage)
	require.Equal(t, RunStatusDeliveryFailed, runs[0].Status)
}

func TestServiceTemplateWriterGetsReportRecipientsAndErrors(t *testing.T) {
	t.Parallel()
	ownerID := uuid.New()
	templateID := uuid.New()
	lastError := "smtp unavailable"
	repo := &serviceRepoStub{
		subscription: &Subscription{QueryTemplateID: templateID, Recipients: []string{"noc@example.com"}, LastError: &lastError},
		runs:         []Run{{QueryTemplateID: templateID, Recipients: []string{"noc@example.com"}, ErrorMessage: &lastError}},
	}
	service := NewService(repo, &templateRepoStub{template: &querytemplate.Template{
		ID: templateID, Visibility: querytemplate.VisibilityPrivate, CreatorID: ownerID,
	}}, nil)

	subscription, err := service.Get(context.Background(), templateID, ownerID, false)
	require.NoError(t, err)
	require.Equal(t, []string{"noc@example.com"}, subscription.Recipients)
	require.Equal(t, lastError, *subscription.LastError)
	runs, err := service.ListRuns(context.Background(), templateID, ownerID, false, 20)
	require.NoError(t, err)
	require.Equal(t, []string{"noc@example.com"}, runs[0].Recipients)
	require.Equal(t, lastError, *runs[0].ErrorMessage)
}

func TestServicePublicTemplateRequiresSuperAdminToManageReport(t *testing.T) {
	t.Parallel()
	ownerID := uuid.New()
	templateID := uuid.New()
	service := NewService(&serviceRepoStub{}, &templateRepoStub{template: &querytemplate.Template{
		ID: templateID, Visibility: querytemplate.VisibilityPublic, CreatorID: ownerID,
	}}, nil)

	_, err := service.Upsert(context.Background(), templateID, ownerID, false, UpsertInput{
		Enabled: true, Period: PeriodDaily, SendTimes: []string{"08:00"}, Recipients: []string{"noc@example.com"},
	})
	require.ErrorIs(t, err, ErrForbidden)
}
