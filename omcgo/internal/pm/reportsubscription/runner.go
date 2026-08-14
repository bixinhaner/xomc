package reportsubscription

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/notification"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
)

type exportCreator interface {
	Create(ctx context.Context, request pmexport.CreateRequest) (*pmexport.Task, error)
}

type exportReader interface {
	Get(ctx context.Context, id uuid.UUID) (*pmexport.Task, error)
}

type objectReader interface {
	GetObject(ctx context.Context, bucket, object string, options minio.GetObjectOptions) (*minio.Object, error)
}

type Runner struct {
	repo         Repository
	exports      exportCreator
	exportReader exportReader
	objects      objectReader
	mailer       *notification.Mailer
	history      *notification.HistoryService
	logger       *zap.Logger
	pollInterval time.Duration
	waitTimeout  time.Duration
}

type RunnerDeps struct {
	Repository   Repository
	Exports      exportCreator
	ExportReader exportReader
	Objects      objectReader
	Mailer       *notification.Mailer
	History      *notification.HistoryService
	Logger       *zap.Logger
}

func NewRunner(deps RunnerDeps) *Runner {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Runner{
		repo: deps.Repository, exports: deps.Exports, exportReader: deps.ExportReader,
		objects: deps.Objects, mailer: deps.Mailer, history: deps.History,
		logger: logger.Named("pm.report.runner"), pollInterval: time.Second, waitTimeout: 5 * time.Minute,
	}
}

var _ asyncjob.JobRunner = (*Runner)(nil)

func (r *Runner) JobType() string { return JobType }

func (r *Runner) Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error) {
	var payload JobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode report job payload: %w", err)
	}
	runID, err := uuid.Parse(payload.RunID)
	if err != nil {
		return nil, fmt.Errorf("parse report run id: %w", err)
	}
	run, err := r.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load report run: %w", err)
	}
	if run.Status == RunStatusSent {
		return json.Marshal(map[string]string{"run_id": runID.String(), "status": RunStatusSent})
	}
	if err := r.repo.MarkRunRunning(ctx, runID); err != nil {
		return nil, fmt.Errorf("mark report run running: %w", err)
	}
	exportTask, err := r.resolveExport(ctx, run)
	if err != nil {
		return nil, r.failRun(ctx, runID, RunStatusExportFailed, err)
	}
	finalAttempt := job.MaxAttempts > 0 && job.Attempt >= job.MaxAttempts
	if err := r.deliver(ctx, run, exportTask, finalAttempt); err != nil {
		return nil, r.failRun(ctx, runID, RunStatusDeliveryFailed, err)
	}
	return json.Marshal(map[string]string{"run_id": runID.String(), "status": RunStatusSent})
}

func (r *Runner) failRun(ctx context.Context, runID uuid.UUID, status string, runErr error) error {
	if markErr := r.repo.MarkRunFailed(ctx, runID, status, runErr.Error()); markErr != nil {
		return fmt.Errorf("report run failed (%v), mark %s failed: %w", runErr, status, markErr)
	}
	return fmt.Errorf("report run %s: %w", status, runErr)
}

func (r *Runner) resolveExport(ctx context.Context, run *Run) (*pmexport.Task, error) {
	if r.exports == nil || r.exportReader == nil {
		return nil, fmt.Errorf("report export dependencies are not fully wired")
	}
	if run.ExportTaskID != nil {
		exportTask, err := r.exportReader.Get(ctx, *run.ExportTaskID)
		if err != nil {
			return nil, fmt.Errorf("load linked KPI report export: %w", err)
		}
		if exportTask.Status != pmexport.StatusFailed {
			return r.waitExport(ctx, exportTask.ID)
		}
	}
	params, err := buildExportParams(run)
	if err != nil {
		return nil, err
	}
	exportTask, err := r.exports.Create(ctx, pmexport.CreateRequest{
		TaskName:   "KPI_Report_" + run.QueryTemplateName,
		SourceType: pmexport.SourceKpiQuery,
		Params:     params,
		CreateUser: "system",
	})
	if err != nil {
		return nil, fmt.Errorf("create KPI report export: %w", err)
	}
	if err := r.repo.MarkRunExportTask(ctx, run.ID, exportTask.ID); err != nil {
		return nil, fmt.Errorf("link KPI report export: %w", err)
	}
	run.ExportTaskID = &exportTask.ID
	return r.waitExport(ctx, exportTask.ID)
}

func (r *Runner) deliver(ctx context.Context, run *Run, exportTask *pmexport.Task, finalAttempt bool) error {
	if r.objects == nil || r.mailer == nil {
		return fmt.Errorf("report delivery dependencies are not fully wired")
	}
	attachmentName := safeReportFilename(run.QueryTemplateName, run.WindowStart, run.WindowEnd)
	dedupKey := "pm-report:" + run.ID.String()
	message := notification.EmailMessage{
		To:      run.Recipients,
		Subject: fmt.Sprintf("KPI report / KPI 报表 - %s", run.QueryTemplateName),
		TextBody: fmt.Sprintf(
			"Template / 模板: %s\nPeriod / 周期: %s\nWindow / 数据窗口: %s - %s\n",
			run.QueryTemplateName, run.Period, run.WindowStart.Format(time.RFC3339), run.WindowEnd.Format(time.RFC3339),
		),
		Attachments: []notification.Attachment{{
			Filename:    attachmentName,
			ContentType: "text/csv; charset=utf-8",
			Size:        exportTask.FileSize,
			Open: func(openCtx context.Context) (io.ReadCloser, error) {
				object, err := r.objects.GetObject(openCtx, exportTask.Bucket, exportTask.FilePath, minio.GetObjectOptions{})
				if err != nil {
					return nil, fmt.Errorf("open KPI report object: %w", err)
				}
				return object, nil
			},
		}},
	}
	deliveries, err := r.mailer.SendMessageBatch(ctx, message, notification.DeliveryMetadata{
		BusinessType: "pm_query_report", BusinessID: run.ID.String(), DedupKey: dedupKey,
		FinalAttempt: finalAttempt,
	})
	if err != nil {
		return fmt.Errorf("send KPI report email: %w", err)
	}
	if len(deliveries) == 0 {
		return fmt.Errorf("send KPI report email: no successful deliveries")
	}
	if err := r.repo.MarkRunSent(ctx, run.ID, deliveries[0].HistoryID, attachmentName); err != nil {
		return fmt.Errorf("mark report run sent: %w", err)
	}
	return nil
}

func (r *Runner) waitExport(ctx context.Context, id uuid.UUID) (*pmexport.Task, error) {
	waitCtx, cancel := context.WithTimeout(ctx, r.waitTimeout)
	defer cancel()
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		task, err := r.exportReader.Get(waitCtx, id)
		if err != nil {
			return nil, fmt.Errorf("load KPI report export: %w", err)
		}
		switch task.Status {
		case pmexport.StatusSucceeded:
			return task, nil
		case pmexport.StatusFailed:
			return nil, fmt.Errorf("KPI report export failed: %s", task.Error)
		}
		select {
		case <-waitCtx.Done():
			return nil, fmt.Errorf("wait KPI report export: %w", waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func buildExportParams(run *Run) ([]byte, error) {
	var templatePayload map[string]any
	if err := json.Unmarshal(run.QueryPayload, &templatePayload); err != nil {
		return nil, fmt.Errorf("decode query template payload: %w", err)
	}
	technology := ""
	switch strings.ToUpper(fmt.Sprint(templatePayload["device_type"])) {
	case "ENB":
		technology = "lte"
	case "GNB":
		technology = "nr"
	case "GSM":
		technology = "gsm"
	}
	params := map[string]any{
		"granularity":  string(run.Period),
		"dimension":    "device",
		"device_sns":   templatePayload["device_sns"],
		"metric_paths": templatePayload["metric_paths"],
		"start_time":   run.WindowStart.Format(time.RFC3339),
		"end_time":     run.WindowEnd.Format(time.RFC3339),
	}
	if technology != "" {
		params["technologies"] = []string{technology}
	}
	out, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode KPI report export params: %w", err)
	}
	return out, nil
}

func safeReportFilename(name string, start, end time.Time) string {
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\\/:*?"<>|;`, r) || r < 32 {
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	if name == "" {
		name = "KPI_Report"
	}
	return fmt.Sprintf("%s_%s_%s.csv", name, start.Format("20060102_1504"), end.Format("20060102_1504"))
}
