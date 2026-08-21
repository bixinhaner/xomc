package provision

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"go.uber.org/zap"
)

// OnlineDevicePageLister supplies keyset pages of online devices. Keyset
// pagination avoids skipping rows when online state changes during the scan.
type OnlineDevicePageLister interface {
	ListOnlineDevices(ctx context.Context, afterID uuid.UUID, limit int) ([]*model.Device, error)
}

// StartupSyncSubmitter submits through the same durable full-sync entry used by
// a real device.online event.
type StartupSyncSubmitter interface {
	SubmitStartupDeviceOnlineFullSync(ctx context.Context, dev *model.Device, idempotencyKey, sourceEventID string) (*DeviceOnlineFullSyncResult, error)
}

// StartupSyncer performs one full-parameter reconciliation for every online
// device after OMC startup. It is independent of the periodic-sync switch.
type StartupSyncer struct {
	lister    OnlineDevicePageLister
	submitter StartupSyncSubmitter
	leader    LeaderElector
	logger    *zap.Logger
	pageSize  int
	runID     uuid.UUID
	retryWait func(context.Context, time.Duration) error
}

type redeployPendingDevice struct {
	device         *model.Device
	idempotencyKey string
	sourceEventID  string
}

const redeployRetryDelay = 10 * time.Second

func NewStartupSyncer(lister OnlineDevicePageLister, submitter StartupSyncSubmitter, leader LeaderElector, pageSize int, logger *zap.Logger) *StartupSyncer {
	if pageSize <= 0 || pageSize > 1000 {
		pageSize = 200
	}
	return &StartupSyncer{lister: lister, submitter: submitter, leader: leader, logger: logger, pageSize: pageSize, runID: uuid.New(), retryWait: waitStartupRetry}
}

func (s *StartupSyncer) Run(ctx context.Context) error {
	if s.lister == nil || s.submitter == nil {
		return nil
	}
	if s.leader != nil {
		acquired, err := s.leader.TryAcquire(ctx)
		if err != nil {
			return fmt.Errorf("acquire startup parameter sync leader: %w", err)
		}
		if !acquired {
			s.logger.Info("OMC redeploy parameter sync skipped on non-leader replica",
				zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()))
			return nil
		}
		defer func() {
			if err := s.leader.Release(context.Background()); err != nil {
				s.logger.Warn("OMC redeploy parameter sync leader release failed",
					zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()), zap.Error(err))
			}
		}()
	}

	startedAt := time.Now()
	s.logger.Info("OMC redeploy full parameter sync started",
		zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()))
	enqueued := 0
	pending := make([]redeployPendingDevice, 0)
	afterID := uuid.Nil
	for page := 1; ; page++ {
		var devices []*model.Device
		for attempt := 1; ; attempt++ {
			var err error
			devices, err = s.lister.ListOnlineDevices(ctx, afterID, s.pageSize)
			if err == nil {
				break
			}
			s.logger.Warn("OMC redeploy online-device page query failed; retrying",
				zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()),
				zap.Int("page", page), zap.Int("attempt", attempt), zap.Error(err))
			if err := s.retryWait(ctx, redeployRetryDelay); err != nil {
				return err
			}
		}
		for _, dev := range devices {
			if device.IsUPSProductClass(dev.ProductClass) {
				s.logger.Info("OMC redeploy parameter sync skipped for UPS device",
					zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()),
					zap.String("device_id", dev.ID.String()), zap.String("device_sn", dev.SerialNumber))
				continue
			}
			key := "omc-redeploy:" + uuid.NewSHA1(s.runID, []byte(dev.ID.String())).String()
			sourceEventID := "omc-redeploy:" + s.runID.String() + ":" + dev.ID.String()
			result, err := s.submitWithRetry(ctx, dev, key, sourceEventID)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				s.logger.Warn("OMC redeploy parameter sync remains pending after submission attempts",
					zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()),
					zap.String("device_id", dev.ID.String()), zap.String("device_sn", dev.SerialNumber), zap.Error(err))
				pending = append(pending, redeployPendingDevice{device: dev, idempotencyKey: key, sourceEventID: sourceEventID})
				continue
			}
			if result != nil {
				enqueued++
			} else {
				pending = append(pending, redeployPendingDevice{device: dev, idempotencyKey: key, sourceEventID: sourceEventID})
			}
		}
		if len(devices) < s.pageSize {
			break
		}
		afterID = devices[len(devices)-1].ID
	}
	for round := 1; len(pending) > 0; round++ {
		s.logger.Warn("OMC redeploy parameter sync retry round scheduled",
			zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()),
			zap.Int("round", round), zap.Int("pending_devices", len(pending)))
		if err := s.retryWait(ctx, redeployRetryDelay); err != nil {
			return err
		}
		next := make([]redeployPendingDevice, 0, len(pending))
		for _, item := range pending {
			result, err := s.submitWithRetry(ctx, item.device, item.idempotencyKey, item.sourceEventID)
			if err != nil || result == nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				next = append(next, item)
				continue
			}
			enqueued++
		}
		pending = next
	}
	s.logger.Info("OMC redeploy full parameter sync completed",
		zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()),
		zap.Int("enqueued", enqueued), zap.Duration("duration", time.Since(startedAt)))
	return nil
}

func (s *StartupSyncer) submitWithRetry(ctx context.Context, dev *model.Device, key, sourceEventID string) (*DeviceOnlineFullSyncResult, error) {
	delays := []time.Duration{0, time.Second, 3 * time.Second}
	var lastErr error
	for attempt, delay := range delays {
		if delay > 0 {
			if err := s.retryWait(ctx, delay); err != nil {
				return nil, err
			}
		}
		result, err := s.submitter.SubmitStartupDeviceOnlineFullSync(ctx, dev, key, sourceEventID)
		if err == nil {
			return result, nil
		}
		lastErr = err
		s.logger.Warn("OMC redeploy parameter sync submission attempt failed",
			zap.String("trigger", "omc_redeploy"), zap.String("redeploy_id", s.runID.String()),
			zap.Int("attempt", attempt+1), zap.String("device_id", dev.ID.String()), zap.Error(err))
	}
	return nil, lastErr
}

func waitStartupRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
