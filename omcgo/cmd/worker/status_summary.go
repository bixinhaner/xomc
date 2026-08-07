package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/notification"
)

const (
	zedSummaryPollInterval = 5 * time.Second
	zedSummaryBatchSize    = 10
	zedSummaryRetryDelay   = time.Minute
)

type zedStatusSummarySource interface {
	ComputeTechnologyStatusSummary(context.Context) (*device.TechnologyStatusSummarySet, error)
}

type zedStatusSummaryConfigReader interface {
	Get(context.Context, uuid.UUID) (*notification.StatusSummaryRuntimeConfig, error)
}

type zedSummaryRecipientProtector interface {
	Protect(channel, address string) ([]byte, int, []byte, error)
	Unprotect(channel string, ciphertext []byte, keyVersion int) (string, error)
}

func startZedStatusSummaryWorker(
	ctx context.Context,
	repo notification.StatusSummaryRunRepository,
	configReader zedStatusSummaryConfigReader,
	source zedStatusSummarySource,
	sender *notification.EmailSender,
	protector zedSummaryRecipientProtector,
	logger *zap.Logger,
) {
	if repo == nil || configReader == nil || source == nil || sender == nil || protector == nil {
		logger.Warn("Zed Mobile status summary worker disabled: dependencies are incomplete")
		return
	}
	go func() {
		ticker := time.NewTicker(zedSummaryPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if err := runZedStatusSummaryCycle(ctx, repo, configReader, source, sender, protector, now, logger); err != nil {
					logger.Warn("Zed Mobile status summary cycle failed", zap.Error(err))
				}
			}
		}
	}()
	logger.Info("Zed Mobile status summary worker started")
}

func runZedStatusSummaryCycle(
	ctx context.Context,
	repo notification.StatusSummaryRunRepository,
	configReader zedStatusSummaryConfigReader,
	source zedStatusSummarySource,
	sender *notification.EmailSender,
	protector zedSummaryRecipientProtector,
	now time.Time,
	logger *zap.Logger,
) error {
	runtimeConfig, err := configReader.Get(ctx, notification.StatusSummaryConfigID())
	if errors.Is(err, commonerrors.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load Zed summary config: %w", err)
	}
	if !runtimeConfig.Enabled {
		return nil
	}
	location, err := time.LoadLocation(runtimeConfig.TimeZone)
	if err != nil {
		return fmt.Errorf("load Zed summary time zone: %w", err)
	}
	cfg := appconfig.ZedSummaryConfig{Enabled: runtimeConfig.Enabled, SendTime: runtimeConfig.SendTime, TimeZone: runtimeConfig.TimeZone}
	parsed, err := time.Parse("15:04", strings.TrimSpace(cfg.SendTime))
	if err != nil {
		return fmt.Errorf("parse Zed summary send time: %w", err)
	}
	localNow := now.In(location)
	if localNow.Hour() < parsed.Hour() || (localNow.Hour() == parsed.Hour() && localNow.Minute() < parsed.Minute()) {
		return nil
	}
	runKey := "zed-mobile-status-summary:" + cfg.TimeZone + ":" + localNow.Format("2006-01-02")
	exists, err := repo.HasRun(ctx, runKey)
	if err != nil {
		return fmt.Errorf("check Zed summary run: %w", err)
	}
	if !exists {
		status, err := source.ComputeTechnologyStatusSummary(ctx)
		if err != nil {
			return fmt.Errorf("compute Zed summary status: %w", err)
		}
		snapshot := zedSummarySnapshot(status, now.UTC(), cfg.TimeZone)
		if _, created, err := repo.EnsureRun(ctx, runKey, snapshot, runtimeConfig.Recipients); err != nil {
			return fmt.Errorf("ensure Zed summary run: %w", err)
		} else if created {
			logger.Info("Zed Mobile status summary run created", zap.String("run_key", runKey))
		}
	}
	runs, err := repo.ClaimRuns(ctx, now.UTC(), zedSummaryBatchSize)
	if err != nil {
		return fmt.Errorf("claim Zed summary runs: %w", err)
	}
	for _, run := range runs {
		if err := processZedStatusSummaryRun(ctx, repo, sender, protector, run, location, now.UTC()); err != nil {
			logger.Warn("process Zed summary run failed", zap.String("run_id", run.ID.String()), zap.Error(err))
		}
	}
	return nil
}

func protectZedSummaryRecipients(protector zedSummaryRecipientProtector, addresses []string) ([]notification.ProtectedStatusSummaryRecipient, error) {
	seen := make(map[string]struct{}, len(addresses))
	result := make([]notification.ProtectedStatusSummaryRecipient, 0, len(addresses))
	for _, raw := range addresses {
		address := strings.ToLower(strings.TrimSpace(raw))
		if address == "" {
			continue
		}
		if _, ok := seen[address]; ok {
			continue
		}
		seen[address] = struct{}{}
		ciphertext, keyVersion, fingerprint, err := protector.Protect("email", address)
		if err != nil {
			return nil, fmt.Errorf("protect Zed summary recipient: %w", err)
		}
		result = append(result, notification.ProtectedStatusSummaryRecipient{Ciphertext: ciphertext, KeyVersion: keyVersion, Fingerprint: fingerprint})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("Zed summary recipients are empty")
	}
	return result, nil
}

func zedSummarySnapshot(status *device.TechnologyStatusSummarySet, generatedAt time.Time, timeZone string) notification.ZedStatusSummarySnapshot {
	snapshot := notification.ZedStatusSummarySnapshot{
		GeneratedAt:  generatedAt,
		TimeZone:     timeZone,
		ByTechnology: make(map[model.Technology]notification.StatusSummaryCounts, 3),
	}
	if status == nil {
		return snapshot
	}
	for technology, counts := range status.ByTechnology {
		snapshot.ByTechnology[technology] = notification.StatusSummaryCounts{Total: counts.Total, Online: counts.Online, Activated: counts.Activated}
		snapshot.ExcludedCPE += counts.ExcludedCPE
	}
	return snapshot
}

func processZedStatusSummaryRun(
	ctx context.Context,
	repo notification.StatusSummaryRunRepository,
	sender *notification.EmailSender,
	protector zedSummaryRecipientProtector,
	run notification.StatusSummaryRun,
	location *time.Location,
	now time.Time,
) error {
	if err := repo.RequeueStaleRecipients(ctx, run.ID, now); err != nil {
		return err
	}
	recipients, err := repo.ListRecipients(ctx, run.ID)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return repo.MarkRunRetry(ctx, run.ID, "Zed summary run has no recipients", now.Add(zedSummaryRetryDelay))
	}
	subject, body, err := notification.RenderZedStatusSummaryEmail(run.Snapshot, location)
	if err != nil {
		return repo.MarkRunRetry(ctx, run.ID, err.Error(), now.Add(zedSummaryRetryDelay))
	}
	for _, recipient := range recipients {
		if recipient.State == "succeeded" {
			continue
		}
		claimed, err := repo.ClaimRecipient(ctx, run.ID, recipient.ID)
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		attempt, err := repo.StartRecipientAttempt(ctx, recipient.ID, now)
		if err != nil {
			return err
		}
		address, err := protector.Unprotect("email", recipient.Ciphertext, recipient.KeyVersion)
		if err != nil {
			if finishErr := repo.FinishRecipientAttempt(ctx, attempt.ID, now, "failed", stringPointer("recipient_data_error"), stringPointer(err.Error())); finishErr != nil {
				return finishErr
			}
			if markErr := repo.MarkRecipientFailed(ctx, recipient.ID, err.Error()); markErr != nil {
				return markErr
			}
			continue
		}
		if err := sender.SendOne(ctx, address, subject, body); err != nil {
			category := "smtp_error"
			unknown := false
			if sendErr, ok := err.(*notification.EmailSendError); ok {
				category = sendErr.Category
				unknown = sendErr.OutcomeUnknown
			}
			if unknown {
				if finishErr := repo.FinishRecipientAttempt(ctx, attempt.ID, now, "unknown", &category, stringPointer(err.Error())); finishErr != nil {
					return finishErr
				}
				if markErr := repo.MarkRecipientUnknown(ctx, recipient.ID, err.Error()); markErr != nil {
					return markErr
				}
				continue
			}
			if finishErr := repo.FinishRecipientAttempt(ctx, attempt.ID, now, "failed", &category, stringPointer(err.Error())); finishErr != nil {
				return finishErr
			}
			if markErr := repo.MarkRecipientFailed(ctx, recipient.ID, err.Error()); markErr != nil {
				return markErr
			}
			continue
		}
		if err := repo.FinishRecipientAttempt(ctx, attempt.ID, now, "accepted", nil, nil); err != nil {
			return err
		}
		if err := repo.MarkRecipientSucceeded(ctx, recipient.ID); err != nil {
			return err
		}
	}
	finalRecipients, err := repo.ListRecipients(ctx, run.ID)
	if err != nil {
		return err
	}
	for _, recipient := range finalRecipients {
		if recipient.State == "unknown" {
			return repo.MarkRunFailed(ctx, run.ID, "one or more Zed summary recipients have unknown SMTP outcome")
		}
		if recipient.State != "succeeded" {
			return repo.MarkRunRetry(ctx, run.ID, "one or more Zed summary recipients failed", now.Add(zedSummaryRetryDelay))
		}
	}
	return repo.MarkRunSucceeded(ctx, run.ID)
}

func stringPointer(value string) *string { return &value }
