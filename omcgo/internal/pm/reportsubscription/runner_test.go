package reportsubscription

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
)

type runnerRepoStub struct {
	run                *Run
	markRunningCalls   int
	linkedExportTaskID uuid.UUID
	failedStatus       string
	failedMessage      string
	markRunFailedErr   error
}

func (s *runnerRepoStub) GetByTemplate(context.Context, uuid.UUID) (*Subscription, error) {
	return nil, ErrNotFound
}
func (s *runnerRepoStub) Upsert(context.Context, uuid.UUID, UpsertInput) (*Subscription, error) {
	return nil, errors.New("unexpected Upsert call")
}
func (s *runnerRepoStub) Delete(context.Context, uuid.UUID) error { return nil }
func (s *runnerRepoStub) ListRuns(context.Context, uuid.UUID, int) ([]Run, error) {
	return nil, nil
}
func (s *runnerRepoStub) GetRun(context.Context, uuid.UUID) (*Run, error) { return s.run, nil }
func (s *runnerRepoStub) MarkRunRunning(context.Context, uuid.UUID) error {
	s.markRunningCalls++
	return nil
}
func (s *runnerRepoStub) MarkRunExportTask(_ context.Context, _ uuid.UUID, exportTaskID uuid.UUID) error {
	s.linkedExportTaskID = exportTaskID
	return nil
}
func (s *runnerRepoStub) MarkRunSent(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}
func (s *runnerRepoStub) MarkRunFailed(_ context.Context, _ uuid.UUID, status string, message string) error {
	s.failedStatus = status
	s.failedMessage = message
	return s.markRunFailedErr
}

type exportCreatorStub struct {
	task  *pmexport.Task
	err   error
	calls int
}

func (s *exportCreatorStub) Create(context.Context, pmexport.CreateRequest) (*pmexport.Task, error) {
	s.calls++
	return s.task, s.err
}

type exportReaderStub struct {
	tasks map[uuid.UUID]*pmexport.Task
}

func (s *exportReaderStub) Get(_ context.Context, id uuid.UUID) (*pmexport.Task, error) {
	task, ok := s.tasks[id]
	if !ok {
		return nil, pmexport.ErrNotFound
	}
	return task, nil
}

func TestBuildExportParamsUsesNaturalWindowAndTemplateSelection(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 8, 11, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	run := &Run{
		Period: PeriodHourly, WindowStart: start, WindowEnd: start.Add(time.Hour),
		QueryPayload: json.RawMessage(`{"device_sns":["SN1","SN2"],"metric_paths":["K1","K2"],"device_type":"ENB"}`),
	}
	raw, err := buildExportParams(run)
	require.NoError(t, err)
	var params map[string]any
	require.NoError(t, json.Unmarshal(raw, &params))
	require.Equal(t, "hourly", params["granularity"])
	require.Equal(t, "device", params["dimension"])
	require.Equal(t, []any{"SN1", "SN2"}, params["device_sns"])
	require.Equal(t, []any{"lte"}, params["technologies"])
	require.Equal(t, start.Format(time.RFC3339), params["start_time"])
}

func TestSafeReportFilename(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	name := safeReportFilename(`NOC/小时:报表`, start, start.Add(time.Hour))
	require.Equal(t, "NOC_小时_报表_20260811_1200_20260811_1300.csv", name)
}

func TestRunnerRunReturnsExecutionFailureForAsyncRetry(t *testing.T) {
	t.Parallel()
	runID := uuid.New()
	repo := &runnerRepoStub{run: &Run{ID: runID, Status: RunStatusPending}}
	runner := NewRunner(RunnerDeps{Repository: repo})
	payload, err := json.Marshal(JobPayload{RunID: runID.String()})
	require.NoError(t, err)

	result, err := runner.Run(context.Background(), &asyncjob.Job{Payload: payload})

	require.Nil(t, result)
	require.ErrorContains(t, err, "report run export_failed")
	require.Equal(t, 1, repo.markRunningCalls)
	require.Equal(t, RunStatusExportFailed, repo.failedStatus)
	require.Contains(t, repo.failedMessage, "report export dependencies are not fully wired")
}

func TestRunnerResolveExportReusesSucceededTask(t *testing.T) {
	t.Parallel()
	runID := uuid.New()
	exportID := uuid.New()
	creator := &exportCreatorStub{}
	reader := &exportReaderStub{tasks: map[uuid.UUID]*pmexport.Task{
		exportID: {ID: exportID, Status: pmexport.StatusSucceeded, Bucket: "exports", FilePath: "report.csv"},
	}}
	runner := NewRunner(RunnerDeps{Repository: &runnerRepoStub{}, Exports: creator, ExportReader: reader})

	task, err := runner.resolveExport(context.Background(), &Run{ID: runID, ExportTaskID: &exportID})

	require.NoError(t, err)
	require.Equal(t, exportID, task.ID)
	require.Zero(t, creator.calls, "a successful export must be reused on SMTP retry")
}

func TestRunnerResolveExportReplacesFailedTask(t *testing.T) {
	t.Parallel()
	runID := uuid.New()
	failedExportID := uuid.New()
	replacementID := uuid.New()
	repo := &runnerRepoStub{}
	creator := &exportCreatorStub{task: &pmexport.Task{ID: replacementID, Status: pmexport.StatusPending}}
	reader := &exportReaderStub{tasks: map[uuid.UUID]*pmexport.Task{
		failedExportID: {ID: failedExportID, Status: pmexport.StatusFailed},
		replacementID:  {ID: replacementID, Status: pmexport.StatusSucceeded, Bucket: "exports", FilePath: "retry.csv"},
	}}
	runner := NewRunner(RunnerDeps{Repository: repo, Exports: creator, ExportReader: reader})
	run := &Run{
		ID: runID, ExportTaskID: &failedExportID, QueryTemplateName: "hourly",
		Period: PeriodHourly, QueryPayload: json.RawMessage(`{"device_sns":["SN1"],"metric_paths":["K1"]}`),
	}

	task, err := runner.resolveExport(context.Background(), run)

	require.NoError(t, err)
	require.Equal(t, replacementID, task.ID)
	require.Equal(t, 1, creator.calls)
	require.Equal(t, replacementID, repo.linkedExportTaskID)
	require.Equal(t, replacementID, *run.ExportTaskID)
}
