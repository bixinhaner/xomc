package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

type emailWorkerRepositoryStub struct {
	delivery   DomainDelivery
	content    EmailDeliveryContent
	authErr    error
	loadErr    error
	attemptNo  int
	completion EmailAttemptCompletion
	calls      []string
}

func (s *emailWorkerRepositoryStub) ClaimEmailDeliveries(_ context.Context, _ EmailDeliveryClaimRequest) ([]DomainDelivery, error) {
	s.calls = append(s.calls, "claim")
	return []DomainDelivery{s.delivery}, nil
}

func (s *emailWorkerRepositoryStub) AuthorizeSend(_ context.Context, _ uuid.UUID, _ time.Time) (AuthorizedDelivery, error) {
	s.calls = append(s.calls, "authorize")
	return AuthorizedDelivery{DomainDelivery: s.delivery}, s.authErr
}

func (s *emailWorkerRepositoryStub) StartEmailAttempt(_ context.Context, deliveryID uuid.UUID, _ string, now time.Time) (DomainDeliveryAttempt, error) {
	s.calls = append(s.calls, "attempt")
	return DomainDeliveryAttempt{ID: uuid.New(), DeliveryID: deliveryID, AttemptNo: s.attemptNo, StartedAt: now, Result: "started"}, nil
}

func (s *emailWorkerRepositoryStub) LoadEmailDeliveryContent(_ context.Context, _ AuthorizedDelivery) (EmailDeliveryContent, error) {
	s.calls = append(s.calls, "content")
	return s.content, s.loadErr
}

func TestRenderEmailDelivery_DigestUsesAggregateValues(t *testing.T) {
	start := time.Date(2026, 8, 5, 14, 0, 0, 0, time.UTC)
	end := start.Add(15 * time.Minute)
	content := EmailDeliveryContent{
		Template: DomainTemplateVersion{
			Channel:  TemplateChannelEmail,
			Subject:  `{{.event_count}} alarms`,
			TextBody: `{{.window_started_at}} - {{.window_ends_at}}`,
		},
		DigestEventCount: 4, DigestWindowStartedAt: &start, DigestWindowEndsAt: &end,
	}

	subject, body, err := renderEmailDelivery(content)

	require.NoError(t, err)
	require.Equal(t, "4 alarms", subject)
	require.Equal(t, "2026-08-05T14:00:00Z - 2026-08-05T14:15:00Z", body)
}

func (s *emailWorkerRepositoryStub) FinishEmailAttempt(_ context.Context, completion EmailAttemptCompletion) error {
	s.calls = append(s.calls, "finish")
	s.completion = completion
	return nil
}

type emailUnprotectorStub struct {
	address string
	err     error
	calls   *[]string
}

func (s emailUnprotectorStub) Unprotect(string, []byte, int) (string, error) {
	*s.calls = append(*s.calls, "decrypt")
	return s.address, s.err
}

type singleEmailTransportStub struct {
	recipient string
	subject   string
	body      string
	err       error
	calls     *[]string
}

func (s *singleEmailTransportStub) SendOne(_ context.Context, recipient, subject, body string) error {
	*s.calls = append(*s.calls, "smtp")
	s.recipient, s.subject, s.body = recipient, subject, body
	return s.err
}

func TestEmailWorker_AuthorizesBeforeAttemptAndCompletesAccepted(t *testing.T) {
	now := time.Date(2026, 8, 5, 14, 0, 0, 0, time.UTC)
	repository := emailWorkerFixture(now)
	transport := &singleEmailTransportStub{calls: &repository.calls}
	worker := NewEmailWorker(repository, emailUnprotectorStub{address: "noc@example.com", calls: &repository.calls}, transport, "email-a", nil)
	worker.now = func() time.Time { return now }

	processed, err := worker.RunOnce(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.Equal(t, []string{"claim", "authorize", "decrypt", "content", "attempt", "smtp", "finish"}, repository.calls)
	require.Equal(t, "noc@example.com", transport.recipient)
	require.Equal(t, "POWER_FAIL major", transport.subject)
	require.Equal(t, "SC-001 active", transport.body)
	require.Equal(t, "completed", repository.completion.FlowState)
	require.Equal(t, "accepted", repository.completion.DeliveryResult)
	require.Equal(t, "accepted", repository.completion.AttemptResult)
	require.NotEqual(t, "delivered", repository.completion.DeliveryResult)
}

func TestEmailWorker_MissingTemplateVariableDeadLettersWithoutSMTP(t *testing.T) {
	now := time.Date(2026, 8, 5, 14, 10, 0, 0, time.UTC)
	repository := emailWorkerFixture(now)
	repository.content.Template.Subject = `{{.missing_variable}}`
	transport := &singleEmailTransportStub{calls: &repository.calls}
	worker := NewEmailWorker(repository, emailUnprotectorStub{address: "noc@example.com", calls: &repository.calls}, transport, "email-a", nil)
	worker.now = func() time.Time { return now }

	_, err := worker.RunOnce(context.Background())

	require.ErrorContains(t, err, "render email subject")
	require.NotContains(t, repository.calls, "smtp")
	require.Equal(t, "dead_letter", repository.completion.FlowState)
	require.Equal(t, "failed", repository.completion.DeliveryResult)
	require.Equal(t, "template_error", *repository.completion.ErrorCategory)
}

func TestEmailWorker_TransientFailureSchedulesBackoff(t *testing.T) {
	now := time.Date(2026, 8, 5, 14, 20, 0, 0, time.UTC)
	repository := emailWorkerFixture(now)
	repository.attemptNo = 2
	transport := &singleEmailTransportStub{
		calls: &repository.calls,
		err:   &EmailSendError{Category: EmailErrorTemporary, Retryable: true, Stage: "recipient", Err: errors.New("451 unavailable")},
	}
	worker := NewEmailWorker(repository, emailUnprotectorStub{address: "noc@example.com", calls: &repository.calls}, transport, "email-a", nil)
	worker.now = func() time.Time { return now }

	_, err := worker.RunOnce(context.Background())

	require.Error(t, err)
	require.Equal(t, "retry_wait", repository.completion.FlowState)
	require.Equal(t, "failed", repository.completion.DeliveryResult)
	require.NotNil(t, repository.completion.NextRetryAt)
	require.Greater(t, *repository.completion.NextRetryAt, now)
	require.Equal(t, EmailErrorTemporary, *repository.completion.ErrorCategory)
}

func TestEmailWorker_UnknownOutcomeDoesNotRetry(t *testing.T) {
	now := time.Date(2026, 8, 5, 14, 30, 0, 0, time.UTC)
	repository := emailWorkerFixture(now)
	transport := &singleEmailTransportStub{
		calls: &repository.calls,
		err:   &EmailSendError{Category: EmailErrorUnknown, OutcomeUnknown: true, Stage: "finalize_data", Err: errors.New("timeout")},
	}
	worker := NewEmailWorker(repository, emailUnprotectorStub{address: "noc@example.com", calls: &repository.calls}, transport, "email-a", nil)
	worker.now = func() time.Time { return now }

	_, err := worker.RunOnce(context.Background())

	require.Error(t, err)
	require.Equal(t, "completed", repository.completion.FlowState)
	require.Equal(t, "unknown", repository.completion.DeliveryResult)
	require.Equal(t, "unknown", repository.completion.AttemptResult)
	require.Nil(t, repository.completion.NextRetryAt)
}

func TestEmailWorker_FinalAuthorizationFenceSkipsAttempt(t *testing.T) {
	now := time.Date(2026, 8, 5, 14, 40, 0, 0, time.UTC)
	repository := emailWorkerFixture(now)
	repository.authErr = ErrDeliveryOccurrenceFenced
	transport := &singleEmailTransportStub{calls: &repository.calls}
	worker := NewEmailWorker(repository, emailUnprotectorStub{address: "noc@example.com", calls: &repository.calls}, transport, "email-a", nil)
	worker.now = func() time.Time { return now }

	processed, err := worker.RunOnce(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.Equal(t, []string{"claim", "authorize"}, repository.calls)
}

func emailWorkerFixture(now time.Time) *emailWorkerRepositoryStub {
	delivery := DomainDelivery{
		ID: uuid.New(), EventID: uuid.New(), OccurrenceID: uuid.New(), RuleVersionID: uuid.New(),
		TemplateVersionID: uuid.New(), ChannelConfigID: uuid.New(), Channel: TemplateChannelEmail,
		DispatchKind: DispatchKindInitial, AddressCiphertext: []byte("ciphertext"), AddressKeyVersion: 1,
		FlowState: "sending", DeliveryResult: "none", OccurrenceVersion: 1, ScheduleGeneration: 1,
	}
	payload := event.AlarmLifecyclePayload{
		OccurredAt: now,
		Snapshot: event.AlarmLifecycleSnapshot{
			DeviceID: uuid.New(), DeviceSN: "SC-001", Severity: model.AlarmMajor,
			AlarmIdentifier: "POWER_FAIL", Status: model.AlarmActive, RaisedAt: now,
		},
	}
	return &emailWorkerRepositoryStub{
		delivery: delivery, attemptNo: 1,
		content: EmailDeliveryContent{
			Template: DomainTemplateVersion{Channel: TemplateChannelEmail, Subject: `{{.alarm_identifier}} {{.severity}}`, TextBody: `{{.device_sn}} {{.status}}`},
			Payload:  payload,
		},
	}
}
