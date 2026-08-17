package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

type fakeAlarmEmailRunRepository struct {
	run          AlarmEmailRun
	subscription AlarmEmailSubscription
	setting      AlarmEmailGlobalSetting
	deliveries   []AlarmEmailDelivery
	runStatus    string
	processing   []uuid.UUID
}

func (f *fakeAlarmEmailRunRepository) GetRun(context.Context, uuid.UUID) (*AlarmEmailRun, error) {
	copy := f.run
	return &copy, nil
}
func (f *fakeAlarmEmailRunRepository) GetSubscription(context.Context, uuid.UUID) (*AlarmEmailSubscription, error) {
	copy := f.subscription
	return &copy, nil
}
func (f *fakeAlarmEmailRunRepository) GetGlobalSetting(context.Context) (*AlarmEmailGlobalSetting, error) {
	copy := f.setting
	return &copy, nil
}
func (f *fakeAlarmEmailRunRepository) MarkRunProcessing(context.Context, uuid.UUID) error {
	f.runStatus = AlarmEmailRunProcessing
	return nil
}
func (f *fakeAlarmEmailRunRepository) MarkRunResult(_ context.Context, _ uuid.UUID, status, _, _, _ string) error {
	f.runStatus = status
	return nil
}
func (f *fakeAlarmEmailRunRepository) EnsureDeliveries(_ context.Context, runID uuid.UUID, recipients []string) error {
	if len(f.deliveries) > 0 {
		return nil
	}
	for _, recipient := range recipients {
		f.deliveries = append(f.deliveries, AlarmEmailDelivery{ID: uuid.New(), RunID: runID, Recipient: recipient, Status: AlarmEmailDeliveryPending})
	}
	return nil
}
func (f *fakeAlarmEmailRunRepository) ListUnsentDeliveries(context.Context, uuid.UUID) ([]AlarmEmailDelivery, error) {
	var result []AlarmEmailDelivery
	for _, delivery := range f.deliveries {
		if delivery.Status != AlarmEmailDeliverySent {
			result = append(result, delivery)
		}
	}
	return result, nil
}
func (f *fakeAlarmEmailRunRepository) MarkDeliveryProcessing(_ context.Context, id uuid.UUID) error {
	f.processing = append(f.processing, id)
	return nil
}
func (f *fakeAlarmEmailRunRepository) MarkDeliverySent(_ context.Context, id uuid.UUID, _ time.Time) error {
	for index := range f.deliveries {
		if f.deliveries[index].ID == id {
			f.deliveries[index].Status = AlarmEmailDeliverySent
		}
	}
	return nil
}
func (f *fakeAlarmEmailRunRepository) MarkDeliveryFailed(_ context.Context, id uuid.UUID, _ string) error {
	for index := range f.deliveries {
		if f.deliveries[index].ID == id {
			f.deliveries[index].Status = AlarmEmailDeliveryFailed
		}
	}
	return nil
}

type fakeAlarmEmailReader struct{ items []AlarmEmailItem }

func (f fakeAlarmEmailReader) ListForEmailWindow(context.Context, *AlarmEmailSubscription, AlarmEmailWindow) ([]AlarmEmailItem, error) {
	return f.items, nil
}

type capturingAlarmEmailReader struct {
	items        []AlarmEmailItem
	subscription AlarmEmailSubscription
}

func (f *capturingAlarmEmailReader) ListForEmailWindow(_ context.Context, subscription *AlarmEmailSubscription, _ AlarmEmailWindow) ([]AlarmEmailItem, error) {
	f.subscription = *subscription
	return f.items, nil
}

type fakeAlarmEmailTransport struct {
	fail       map[string]error
	recipients []string
}

func (f *fakeAlarmEmailTransport) SendHTMLWithMessageID(_ context.Context, recipients []string, _, _, _ string) error {
	f.recipients = append(f.recipients, recipients[0])
	return f.fail[recipients[0]]
}

type fakeAlarmEmailScopeAuthorizer struct {
	err   error
	calls int
}

func (f *fakeAlarmEmailScopeAuthorizer) AuthorizeDeviceScope(context.Context, uuid.UUID, []uuid.UUID, []uuid.UUID) error {
	f.calls++
	return f.err
}

func newAlarmEmailRunnerFixture() (*fakeAlarmEmailRunRepository, *asyncjob.Job) {
	runID := uuid.New()
	subscriptionID := uuid.New()
	window := AlarmEmailWindow{Start: time.Now().Add(-10 * time.Minute), End: time.Now()}
	subscription := AlarmEmailSubscription{
		ID:                       subscriptionID,
		Name:                     "critical alarms",
		Enabled:                  true,
		Recipients:               []string{"one@example.com"},
		IncludeDefaultRecipients: true,
	}
	repository := &fakeAlarmEmailRunRepository{
		run: AlarmEmailRun{
			ID: runID, SubscriptionID: subscriptionID, Window: window,
			SubscriptionSnapshot: subscription,
			RecipientsSnapshot:   []string{"one@example.com", "two@example.com"},
		},
		subscription: subscription,
		setting:      AlarmEmailGlobalSetting{Enabled: true, DefaultRecipients: []string{"two@example.com"}},
	}
	payload, _ := json.Marshal(alarmEmailJobPayload{RunID: runID})
	return repository, &asyncjob.Job{ID: uuid.New(), JobType: AlarmEmailJobType, Payload: payload}
}

func TestAlarmEmailJobRunnerUsesEnqueueSnapshotAfterSubscriptionChanges(t *testing.T) {
	repository, job := newAlarmEmailRunnerFixture()
	originalDeviceID := uuid.New()
	repository.run.SubscriptionSnapshot.DeviceIDs = []uuid.UUID{originalDeviceID}
	repository.subscription.Recipients = []string{"changed@example.com"}
	repository.subscription.DeviceIDs = []uuid.UUID{uuid.New()}
	repository.setting.DefaultRecipients = []string{"changed-default@example.com"}

	reader := &capturingAlarmEmailReader{items: []AlarmEmailItem{{AlarmIdentifier: "11184", RaisedAt: time.Now()}}}
	transport := &fakeAlarmEmailTransport{fail: map[string]error{}}
	runner := NewAlarmEmailJobRunner(repository, reader, transport, "Test OMC", time.UTC, nil)

	_, err := runner.Run(context.Background(), job)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{originalDeviceID}, reader.subscription.DeviceIDs)
	require.ElementsMatch(t, []string{"one@example.com", "two@example.com"}, transport.recipients)
}

func TestAlarmEmailJobRunnerSendsEachRecipientAndKeepsClearedAlarm(t *testing.T) {
	repository, job := newAlarmEmailRunnerFixture()
	clearedAt := time.Now().Add(-time.Minute)
	transport := &fakeAlarmEmailTransport{fail: map[string]error{}}
	runner := NewAlarmEmailJobRunner(repository, fakeAlarmEmailReader{items: []AlarmEmailItem{
		{AlarmIdentifier: "11184", RaisedAt: time.Now().Add(-5 * time.Minute)},
		{AlarmIdentifier: "11217", RaisedAt: time.Now().Add(-4 * time.Minute), ClearedAt: &clearedAt},
	}}, transport, "Test OMC", time.UTC, nil)

	result, err := runner.Run(context.Background(), job)
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"sent"}`, string(result))
	require.Equal(t, AlarmEmailRunSent, repository.runStatus)
	require.ElementsMatch(t, []string{"one@example.com", "two@example.com"}, transport.recipients)
}

func TestAlarmEmailJobRunnerRetriesOnlyFailedRecipient(t *testing.T) {
	repository, job := newAlarmEmailRunnerFixture()
	transport := &fakeAlarmEmailTransport{fail: map[string]error{"two@example.com": errors.New("temporary smtp error")}}
	runner := NewAlarmEmailJobRunner(repository, fakeAlarmEmailReader{items: []AlarmEmailItem{{AlarmIdentifier: "11184", RaisedAt: time.Now()}}}, transport, "Test OMC", time.UTC, nil)

	_, err := runner.Run(context.Background(), job)
	require.Error(t, err)
	require.Equal(t, AlarmEmailRunPartialFailed, repository.runStatus)

	transport.recipients = nil
	_, err = runner.Run(context.Background(), job)
	require.Error(t, err)
	require.Equal(t, []string{"two@example.com"}, transport.recipients)
	require.Equal(t, AlarmEmailRunPartialFailed, repository.runStatus)

	transport.fail = map[string]error{}
	transport.recipients = nil
	_, err = runner.Run(context.Background(), job)
	require.NoError(t, err)
	require.Equal(t, []string{"two@example.com"}, transport.recipients)
	require.Equal(t, AlarmEmailRunSent, repository.runStatus)
}

func TestAlarmEmailJobRunnerDoesNotSendWhenNoAlarmMatches(t *testing.T) {
	repository, job := newAlarmEmailRunnerFixture()
	transport := &fakeAlarmEmailTransport{fail: map[string]error{}}
	runner := NewAlarmEmailJobRunner(repository, fakeAlarmEmailReader{}, transport, "Test OMC", time.UTC, nil)

	result, err := runner.Run(context.Background(), job)
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"empty"}`, string(result))
	require.Empty(t, transport.recipients)
	require.Empty(t, repository.deliveries)
	require.Equal(t, AlarmEmailRunSent, repository.runStatus)
}

func TestAlarmEmailJobRunnerRejectsRevokedDeviceScopeBeforeSending(t *testing.T) {
	t.Parallel()
	repository, job := newAlarmEmailRunnerFixture()
	creatorID := uuid.New()
	repository.run.SubscriptionSnapshot.CreatedBy = &creatorID
	repository.run.SubscriptionSnapshot.DeviceGroupIDs = []uuid.UUID{uuid.New()}
	transport := &fakeAlarmEmailTransport{fail: map[string]error{}}
	authorizer := &fakeAlarmEmailScopeAuthorizer{err: errors.New("device scope revoked")}
	runner := NewAlarmEmailJobRunner(
		repository,
		fakeAlarmEmailReader{items: []AlarmEmailItem{{AlarmIdentifier: "10"}}},
		transport,
		"OMC",
		time.UTC,
		nil,
	)
	runner.SetScopeAuthorizer(authorizer)

	_, err := runner.Run(context.Background(), job)
	require.ErrorContains(t, err, "revalidate alarm email device scope")
	require.Equal(t, 1, authorizer.calls)
	require.Empty(t, transport.recipients)
	require.Equal(t, AlarmEmailRunFailed, repository.runStatus)
}
