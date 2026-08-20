package regularreport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/notification"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
)

type runnerRepoStub struct {
	run        Run
	deliveries []Delivery
	statuses   []RunStatus
	prepare    func()
}

func (r *runnerRepoStub) ListEnabledTemplates(context.Context) ([]Template, error) { return nil, nil }
func (r *runnerRepoStub) EnsureRun(context.Context, EnsureRunRequest) (bool, error) {
	return false, nil
}
func (r *runnerRepoStub) GetRun(context.Context, uuid.UUID) (*Run, error) {
	copy := r.run
	return &copy, nil
}
func (r *runnerRepoStub) ListUnsentDeliveries(context.Context, uuid.UUID) ([]Delivery, error) {
	result := make([]Delivery, 0)
	for _, delivery := range r.deliveries {
		if delivery.Status != "sent" {
			result = append(result, delivery)
		}
	}
	return result, nil
}
func (r *runnerRepoStub) CountSentDeliveries(context.Context, uuid.UUID) (int, error) {
	count := 0
	for _, delivery := range r.deliveries {
		if delivery.Status == "sent" {
			count++
		}
	}
	return count, nil
}
func (r *runnerRepoStub) MarkRunProcessing(context.Context, uuid.UUID) error {
	r.run.Status = RunProcessing
	return nil
}
func (r *runnerRepoStub) MarkRunResult(_ context.Context, _ uuid.UUID, status RunStatus, subject, lastError string) error {
	r.run.Status, r.run.Subject, r.run.LastError = status, subject, lastError
	r.statuses = append(r.statuses, status)
	return nil
}
func (r *runnerRepoStub) MarkDeliveryResult(_ context.Context, id uuid.UUID, sent bool, _ string) error {
	for i := range r.deliveries {
		if r.deliveries[i].ID == id {
			r.deliveries[i].Attempt++
			if sent {
				r.deliveries[i].Status = "sent"
			} else {
				r.deliveries[i].Status = "failed"
			}
		}
	}
	return nil
}
func (r *runnerRepoStub) PrepareExportRetry(context.Context, uuid.UUID) error {
	if r.prepare != nil {
		r.prepare()
	}
	return nil
}

type exportRepoStub struct{ task *pmexport.Task }

func (e *exportRepoStub) Get(context.Context, uuid.UUID) (*pmexport.Task, error) {
	copy := *e.task
	return &copy, nil
}

type exportRunnerStub struct {
	task      *pmexport.Task
	calls     int
	failFirst bool
}

func (e *exportRunnerStub) Run(context.Context, *asyncjob.Job) (json.RawMessage, error) {
	e.calls++
	if e.failFirst && e.calls == 1 {
		e.task.Status = pmexport.StatusFailed
		e.task.Error = "temporary export failure"
		return json.RawMessage(`{"status":"failed"}`), nil
	}
	e.task.Status = pmexport.StatusSucceeded
	e.task.Bucket = "reports"
	e.task.FilePath = "kpi/report.csv"
	return json.RawMessage(`{}`), nil
}

type deviceScopeAuthorizerStub struct {
	err   error
	calls int
}

func (s *deviceScopeAuthorizerStub) AuthorizeSerialNumbers(context.Context, uuid.UUID, []string) error {
	s.calls++
	return s.err
}

type attachmentStoreStub struct{}

func (attachmentStoreStub) Read(context.Context, string, string, int64) ([]byte, error) {
	return []byte("time,kpi\n2026-08-17,1\n"), nil
}

type attachmentSenderStub struct {
	failRecipient string
	calls         map[string]int
}

func (s *attachmentSenderStub) SendWithAttachmentsMessageID(_ context.Context, to []string, _, _ string, attachments []notification.EmailAttachment, _ string) error {
	s.calls[to[0]]++
	if len(attachments) != 1 || len(attachments[0].Data) == 0 {
		return errors.New("missing attachment")
	}
	if to[0] == s.failRecipient {
		return errors.New("temporary smtp error")
	}
	return nil
}

func TestRunnerRetriesOnlyFailedRecipientsWithoutRegenerating(t *testing.T) {
	t.Parallel()
	runID, exportID := uuid.New(), uuid.New()
	repo := &runnerRepoStub{
		run: Run{ID: runID, TemplateName: "Radio KPI", Period: PeriodDaily, ExportTaskID: exportID, WindowStart: time.Now().Add(-24 * time.Hour), WindowEnd: time.Now(), Status: RunPending},
		deliveries: []Delivery{
			{ID: uuid.New(), RunID: runID, Recipient: "ok@example.com", Status: "pending"},
			{ID: uuid.New(), RunID: runID, Recipient: "retry@example.com", Status: "pending"},
		},
	}
	exportRepo := &exportRepoStub{task: &pmexport.Task{ID: exportID, Status: pmexport.StatusPending}}
	exporter := &exportRunnerStub{task: exportRepo.task}
	sender := &attachmentSenderStub{failRecipient: "retry@example.com", calls: map[string]int{}}
	runner := NewRunner(repo, exportRepo, exporter, attachmentStoreStub{}, sender, "Test OMC", func() *time.Location { return time.UTC }, nil)
	payload, err := json.Marshal(JobPayload{RunID: runID.String()})
	require.NoError(t, err)

	_, err = runner.Run(context.Background(), &asyncjob.Job{Payload: payload, Attempt: 1})
	require.ErrorContains(t, err, "1 recipient")
	require.Equal(t, 1, exporter.calls)
	require.Equal(t, 1, sender.calls["ok@example.com"])
	require.Equal(t, 1, sender.calls["retry@example.com"])
	require.Equal(t, RunPartialFailed, repo.run.Status)

	sender.failRecipient = ""
	_, err = runner.Run(context.Background(), &asyncjob.Job{Payload: payload, Attempt: 2})
	require.NoError(t, err)
	require.Equal(t, 1, exporter.calls, "retry must reuse the existing object")
	require.Equal(t, 1, sender.calls["ok@example.com"], "successful recipient must not be resent")
	require.Equal(t, 2, sender.calls["retry@example.com"])
	require.Equal(t, RunSent, repo.run.Status)
}

func TestRunnerRetriesFailedExportInsteadOfFinishingAsyncJob(t *testing.T) {
	t.Parallel()
	runID, exportID := uuid.New(), uuid.New()
	task := &pmexport.Task{ID: exportID, Status: pmexport.StatusPending}
	repo := &runnerRepoStub{
		run:        Run{ID: runID, TemplateName: "Radio KPI", Period: PeriodDaily, ExportTaskID: exportID, WindowStart: time.Now().Add(-24 * time.Hour), WindowEnd: time.Now(), Status: RunPending},
		deliveries: []Delivery{{ID: uuid.New(), RunID: runID, Recipient: "ops@example.com", Status: "pending"}},
		prepare:    func() { task.Status = pmexport.StatusPending },
	}
	exportRepo := &exportRepoStub{task: task}
	exporter := &exportRunnerStub{task: task, failFirst: true}
	sender := &attachmentSenderStub{calls: map[string]int{}}
	runner := NewRunner(repo, exportRepo, exporter, attachmentStoreStub{}, sender, "Test OMC", func() *time.Location { return time.UTC }, nil)
	payload, err := json.Marshal(JobPayload{RunID: runID.String()})
	require.NoError(t, err)

	_, err = runner.Run(context.Background(), &asyncjob.Job{Payload: payload, Attempt: 1, MaxAttempts: 3})
	require.ErrorContains(t, err, "KPI export failed")
	require.Equal(t, RunFailed, repo.run.Status)

	_, err = runner.Run(context.Background(), &asyncjob.Job{Payload: payload, Attempt: 2, MaxAttempts: 3})
	require.NoError(t, err)
	require.Equal(t, 2, exporter.calls)
	require.Equal(t, RunSent, repo.run.Status)
}

func TestRunnerRejectsRevokedDeviceScopeBeforeExport(t *testing.T) {
	t.Parallel()
	runID, exportID, creatorID := uuid.New(), uuid.New(), uuid.New()
	repo := &runnerRepoStub{
		run: Run{ID: runID, CreatorID: creatorID, TemplateName: "Radio KPI", Period: PeriodDaily, ExportTaskID: exportID, WindowStart: time.Now().Add(-24 * time.Hour), WindowEnd: time.Now(), Status: RunPending},
	}
	exportRepo := &exportRepoStub{task: &pmexport.Task{ID: exportID, Status: pmexport.StatusPending, Params: []byte(`{"device_sns":["SN-1"]}`)}}
	exporter := &exportRunnerStub{task: exportRepo.task}
	authorizer := &deviceScopeAuthorizerStub{err: errors.New("device scope revoked")}
	runner := NewRunner(repo, exportRepo, exporter, attachmentStoreStub{}, &attachmentSenderStub{calls: map[string]int{}}, "Test OMC", func() *time.Location { return time.UTC }, nil)
	runner.SetScopeAuthorizer(authorizer)
	payload, err := json.Marshal(JobPayload{RunID: runID.String()})
	require.NoError(t, err)

	_, err = runner.Run(context.Background(), &asyncjob.Job{Payload: payload, Attempt: 1, MaxAttempts: 3})
	require.ErrorContains(t, err, "revalidate KPI report device scope")
	require.Equal(t, 1, authorizer.calls)
	require.Equal(t, 0, exporter.calls)
	require.Equal(t, RunFailed, repo.run.Status)
}
