package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/notification"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
	"github.com/omcgo/omcgo/internal/pm/querytemplate"
)

const (
	regularReportPollInterval  = 5 * time.Second
	regularReportBatchSize     = 20
	regularReportRetryDelay    = time.Minute
	regularReportMaxAttachment = 20 << 20
)

type reportRecipientDecryptor interface {
	Unprotect(channel string, ciphertext []byte, keyVersion int) (string, error)
}

func startRegularReportWorker(
	ctx context.Context,
	repo *querytemplate.PgRepository,
	exportService *pmexport.Service,
	exportRepo pmexport.Repository,
	minioClient *minio.Client,
	reportBucket string,
	sender *notification.EmailSender,
	decryptor reportRecipientDecryptor,
	tz *tzManager,
	logger *zap.Logger,
) {
	if repo == nil || exportService == nil || exportRepo == nil || minioClient == nil || sender == nil || decryptor == nil {
		logger.Warn("KPI regular report worker disabled: dependencies are incomplete")
		return
	}
	go func() {
		ticker := time.NewTicker(regularReportPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				location := time.UTC
				if tz != nil {
					location = tz.Current()
				}
				if err := runRegularReportCycle(ctx, repo, exportService, exportRepo, minioClient, reportBucket, sender, decryptor, now, location, logger); err != nil {
					logger.Warn("KPI regular report cycle failed", zap.Error(err))
				}
			}
		}
	}()
	logger.Info("KPI regular report worker started")
}

func runRegularReportCycle(
	ctx context.Context,
	repo *querytemplate.PgRepository,
	exportService *pmexport.Service,
	exportRepo pmexport.Repository,
	minioClient *minio.Client,
	reportBucket string,
	sender *notification.EmailSender,
	decryptor reportRecipientDecryptor,
	now time.Time,
	location *time.Location,
	logger *zap.Logger,
) error {
	claimed, err := repo.ClaimDueRegularReports(ctx, now, location, regularReportBatchSize)
	if err != nil {
		return fmt.Errorf("claim due KPI regular reports: %w", err)
	}
	for _, report := range claimed {
		params, paramsErr := querytemplate.BuildRegularReportExportParams(report.Payload, report.Period, report.WindowStart, report.WindowEnd)
		if paramsErr != nil {
			_ = repo.MarkRegularReportRunRetry(ctx, report.RunID, paramsErr.Error(), now.Add(regularReportRetryDelay))
			continue
		}
		task, createErr := exportService.Create(ctx, pmexport.CreateRequest{
			TaskName:   fmt.Sprintf("regular-report-%s-%s", report.TemplateName, report.RunID),
			SourceType: pmexport.SourceKpiQuery,
			Params:     params,
			CreateUser: "system",
		})
		if createErr != nil {
			_ = repo.MarkRegularReportRunRetry(ctx, report.RunID, createErr.Error(), now.Add(regularReportRetryDelay))
			continue
		}
		if setErr := repo.SetRegularReportExportTask(ctx, report.RunID, task.ID); setErr != nil {
			return fmt.Errorf("bind KPI report export task: %w", setErr)
		}
	}

	runs, err := repo.ClaimRegularReportRuns(ctx, now, regularReportBatchSize)
	if err != nil {
		return fmt.Errorf("claim KPI report executions: %w", err)
	}
	for _, run := range runs {
		if err := processRegularReportRun(ctx, repo, exportRepo, minioClient, reportBucket, sender, decryptor, run, now); err != nil {
			logger.Warn("KPI regular report execution failed", zap.String("run_id", run.ID.String()), zap.Error(err))
		}
	}
	return nil
}

func processRegularReportRun(
	ctx context.Context,
	repo *querytemplate.PgRepository,
	exportRepo pmexport.Repository,
	minioClient *minio.Client,
	reportBucket string,
	sender *notification.EmailSender,
	decryptor reportRecipientDecryptor,
	run querytemplate.RegularReportRun,
	now time.Time,
) error {
	if run.ExportTaskID == nil {
		return retryRegularReportRun(ctx, repo, run.ID, "export task is not bound", now)
	}
	task, err := exportRepo.Get(ctx, *run.ExportTaskID)
	if err != nil {
		return retryRegularReportRun(ctx, repo, run.ID, fmt.Sprintf("load export task: %v", err), now)
	}
	switch task.Status {
	case pmexport.StatusPending, pmexport.StatusRunning:
		return repo.MarkRegularReportRunRetry(ctx, run.ID, "export is still running", now.Add(10*time.Second))
	case pmexport.StatusFailed:
		if err := repo.ClearRegularReportExportTask(ctx, run.ID); err != nil {
			return err
		}
		return retryRegularReportRun(ctx, repo, run.ID, task.Error, now)
	case pmexport.StatusSucceeded:
		// continue
	default:
		return retryRegularReportRun(ctx, repo, run.ID, "unsupported export state", now)
	}

	if err := repo.MarkRegularReportRunSending(ctx, run.ID); err != nil {
		return err
	}
	bucket := task.Bucket
	if bucket == "" {
		bucket = reportBucket
	}
	object, err := minioClient.GetObject(ctx, bucket, task.FilePath, minio.GetObjectOptions{})
	if err != nil {
		return retryRegularReportRun(ctx, repo, run.ID, fmt.Sprintf("load exported file: %v", err), now)
	}
	defer object.Close()
	content, err := io.ReadAll(io.LimitReader(object, regularReportMaxAttachment+1))
	if err != nil {
		return retryRegularReportRun(ctx, repo, run.ID, fmt.Sprintf("read exported file: %v", err), now)
	}
	if len(content) > regularReportMaxAttachment {
		return retryRegularReportRun(ctx, repo, run.ID, "exported file exceeds attachment limit", now)
	}

	recipients, err := repo.ListRegularReportRunRecipients(ctx, run.ID)
	if err != nil {
		return err
	}
	if err := repo.RequeueStaleRegularReportRecipients(ctx, run.ID, now); err != nil {
		return err
	}
	if len(recipients) == 0 {
		return retryRegularReportRun(ctx, repo, run.ID, "regular report has no recipients", now)
	}
	subject := "OMC KPI 定时报表"
	body := fmt.Sprintf("统计窗口：%s 至 %s\n本邮件由 OMC 自动发送。", run.WindowStart.Format(time.RFC3339), run.WindowEnd.Format(time.RFC3339))
	for _, recipient := range recipients {
		if recipient.State == "succeeded" {
			continue
		}
		claimed, err := repo.ClaimRegularReportRecipient(ctx, run.ID, recipient.ID)
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		address, err := decryptor.Unprotect("email", recipient.Ciphertext, recipient.KeyVersion)
		if err != nil {
			_ = repo.MarkRegularReportRecipientFailed(ctx, recipient.ID, err.Error())
			continue
		}
		err = sender.SendOneWithAttachments(ctx, address, subject, body, []notification.EmailAttachment{{
			Filename: fmt.Sprintf("kpi-report-%s.csv", run.ID), ContentType: "text/csv; charset=utf-8", Data: content,
		}})
		if err != nil {
			_ = repo.MarkRegularReportRecipientFailed(ctx, recipient.ID, err.Error())
			continue
		}
		if err := repo.MarkRegularReportRecipientSucceeded(ctx, recipient.ID); err != nil {
			return err
		}
	}

	finalRecipients, err := repo.ListRegularReportRunRecipients(ctx, run.ID)
	if err != nil {
		return err
	}
	for _, recipient := range finalRecipients {
		if recipient.State != "succeeded" {
			return retryRegularReportRun(ctx, repo, run.ID, "one or more recipients failed", now.Add(regularReportRetryDelay))
		}
	}
	return repo.MarkRegularReportRunSucceeded(ctx, run.ID)
}

func retryRegularReportRun(ctx context.Context, repo *querytemplate.PgRepository, runID uuid.UUID, message string, next time.Time) error {
	if next.Equal(time.Time{}) {
		next = time.Now().Add(regularReportRetryDelay)
	}
	return repo.MarkRegularReportRunRetry(ctx, runID, message, next)
}
