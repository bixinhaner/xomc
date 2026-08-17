package regularreport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/notification"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
)

const maxAttachmentBytes int64 = 20 << 20

type ExportRunner interface {
	Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error)
}

type ExportRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*pmexport.Task, error)
}

type AttachmentStore interface {
	Read(ctx context.Context, bucket, object string, maxBytes int64) ([]byte, error)
}

type AttachmentSender interface {
	SendWithAttachmentsMessageID(ctx context.Context, to []string, subject, body string, attachments []notification.EmailAttachment, messageID string) error
}

type DeviceScopeAuthorizer interface {
	AuthorizeSerialNumbers(ctx context.Context, userID uuid.UUID, serialNumbers []string) error
}

type Runner struct {
	repo       Repository
	exportRepo ExportRepository
	exporter   ExportRunner
	store      AttachmentStore
	sender     AttachmentSender
	omcName    string
	location   func() *time.Location
	scope      DeviceScopeAuthorizer
	logger     *zap.Logger
}

func (r *Runner) SetScopeAuthorizer(authorizer DeviceScopeAuthorizer) {
	r.scope = authorizer
}

func NewRunner(
	repo Repository,
	exportRepo ExportRepository,
	exporter ExportRunner,
	store AttachmentStore,
	sender AttachmentSender,
	omcName string,
	location func() *time.Location,
	logger *zap.Logger,
) *Runner {
	if strings.TrimSpace(omcName) == "" {
		omcName = "OMC"
	}
	if location == nil {
		location = func() *time.Location { return time.UTC }
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Runner{repo: repo, exportRepo: exportRepo, exporter: exporter, store: store, sender: sender, omcName: omcName, location: location, logger: logger.Named("kpi-regular-report")}
}

func (r *Runner) JobType() string { return JobType }

func (r *Runner) Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error) {
	var payload JobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("parse KPI report job payload: %w", err)
	}
	runID, err := uuid.Parse(payload.RunID)
	if err != nil {
		return nil, fmt.Errorf("invalid KPI report run_id: %w", err)
	}
	run, err := r.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load KPI report run: %w", err)
	}
	if run.Status == RunSent {
		return json.Marshal(map[string]any{"run_id": run.ID, "status": run.Status})
	}
	if err := r.repo.MarkRunProcessing(ctx, run.ID); err != nil {
		return nil, fmt.Errorf("mark KPI report processing: %w", err)
	}

	task, err := r.exportRepo.Get(ctx, run.ExportTaskID)
	if err != nil {
		return nil, r.failRun(ctx, run, fmt.Errorf("load KPI export task: %w", err))
	}
	if r.scope != nil {
		serialNumbers, err := exportDeviceSerialNumbers(task.Params)
		if err != nil {
			return nil, r.failRun(ctx, run, err)
		}
		if err := r.scope.AuthorizeSerialNumbers(ctx, run.CreatorID, serialNumbers); err != nil {
			return nil, r.failRun(ctx, run, fmt.Errorf("revalidate KPI report device scope: %w", err))
		}
	}
	if (task.Status == pmexport.StatusRunning || task.Status == pmexport.StatusFailed) && job.Attempt > 1 {
		if err := r.repo.PrepareExportRetry(ctx, task.ID); err != nil {
			return nil, r.failRun(ctx, run, fmt.Errorf("prepare KPI export retry: %w", err))
		}
		task.Status = pmexport.StatusPending
	}
	if task.Status == pmexport.StatusPending {
		exportPayload, err := pmexport.BuildJobPayload(task.ID)
		if err != nil {
			return nil, r.failRun(ctx, run, err)
		}
		if _, err := r.exporter.Run(ctx, &asyncjob.Job{Payload: exportPayload}); err != nil {
			return nil, r.failRun(ctx, run, fmt.Errorf("generate KPI report export: %w", err))
		}
		task, err = r.exportRepo.Get(ctx, run.ExportTaskID)
		if err != nil {
			return nil, r.failRun(ctx, run, fmt.Errorf("reload KPI export task: %w", err))
		}
	}
	if task.Status == pmexport.StatusFailed {
		return nil, r.failRun(ctx, run, fmt.Errorf("KPI export failed: %s", task.Error))
	}
	if task.Status != pmexport.StatusSucceeded || task.Bucket == "" || task.FilePath == "" {
		return nil, r.failRun(ctx, run, fmt.Errorf("KPI export task is not ready: %s", task.Status))
	}

	attachment, err := r.store.Read(ctx, task.Bucket, task.FilePath, maxAttachmentBytes)
	if err != nil {
		return nil, r.failRun(ctx, run, fmt.Errorf("read KPI report attachment: %w", err))
	}
	deliveries, err := r.repo.ListUnsentDeliveries(ctx, run.ID)
	if err != nil {
		return nil, fmt.Errorf("list KPI report deliveries: %w", err)
	}
	if len(deliveries) == 0 {
		if err := r.repo.MarkRunResult(ctx, run.ID, RunSent, run.Subject, ""); err != nil {
			return nil, fmt.Errorf("mark KPI report sent: %w", err)
		}
		return json.Marshal(map[string]any{"run_id": run.ID, "status": RunSent})
	}
	previouslySent, err := r.repo.CountSentDeliveries(ctx, run.ID)
	if err != nil {
		return nil, fmt.Errorf("count sent KPI report deliveries: %w", err)
	}

	location := r.location()
	if location == nil {
		location = time.UTC
	}
	subject := fmt.Sprintf("[%s] KPI Regular Report - %s - %s", r.omcName, run.TemplateName, periodLabel(run.Period))
	body := fmt.Sprintf(
		"OMC: %s\nTemplate: %s\nPeriod: %s\nWindow: %s - %s\nTimezone: %s\n",
		r.omcName, run.TemplateName, periodLabel(run.Period),
		run.WindowStart.In(location).Format("2006-01-02 15:04:05"),
		run.WindowEnd.In(location).Format("2006-01-02 15:04:05"), location.String(),
	)
	filename := path.Base(task.FilePath)
	if filename == "." || filename == "/" || filename == "" {
		filename = "kpi-report.csv"
	}
	mailAttachment := notification.EmailAttachment{Filename: filename, ContentType: "text/csv; charset=utf-8", Data: attachment}

	failed := make([]string, 0)
	sentNow := 0
	for _, delivery := range deliveries {
		messageID := fmt.Sprintf("<kpi-report-%s@omc.local>", delivery.ID)
		sendErr := r.sender.SendWithAttachmentsMessageID(ctx, []string{delivery.Recipient}, subject, body, []notification.EmailAttachment{mailAttachment}, messageID)
		if sendErr != nil {
			failed = append(failed, delivery.Recipient)
			if err := r.repo.MarkDeliveryResult(ctx, delivery.ID, false, sendErr.Error()); err != nil {
				return nil, fmt.Errorf("mark KPI report delivery failed: %w", err)
			}
			continue
		}
		if err := r.repo.MarkDeliveryResult(ctx, delivery.ID, true, ""); err != nil {
			return nil, fmt.Errorf("mark KPI report delivery sent: %w", err)
		}
		sentNow++
	}
	if len(failed) > 0 {
		lastError := fmt.Sprintf("%d recipient(s) failed", len(failed))
		status := RunFailed
		if previouslySent+sentNow > 0 {
			status = RunPartialFailed
		}
		if err := r.repo.MarkRunResult(ctx, run.ID, status, subject, lastError); err != nil {
			return nil, fmt.Errorf("mark KPI report partial failure: %w", err)
		}
		return nil, errors.New(lastError)
	}
	if err := r.repo.MarkRunResult(ctx, run.ID, RunSent, subject, ""); err != nil {
		return nil, fmt.Errorf("mark KPI report sent: %w", err)
	}
	return json.Marshal(map[string]any{"run_id": run.ID, "status": RunSent})
}

func (r *Runner) failRun(ctx context.Context, run *Run, err error) error {
	if markErr := r.repo.MarkRunResult(ctx, run.ID, RunFailed, run.Subject, err.Error()); markErr != nil {
		return fmt.Errorf("%v; mark KPI report failed: %w", err, markErr)
	}
	return err
}

func exportDeviceSerialNumbers(params []byte) ([]string, error) {
	var request struct {
		DeviceSNs []string `json:"device_sns"`
	}
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, fmt.Errorf("parse KPI report export device scope: %w", err)
	}
	return request.DeviceSNs, nil
}

func periodLabel(period Period) string {
	switch period {
	case Period15Min:
		return "15Min"
	case PeriodHourly:
		return "Hour"
	case PeriodDaily:
		return "Day"
	default:
		return string(period)
	}
}
