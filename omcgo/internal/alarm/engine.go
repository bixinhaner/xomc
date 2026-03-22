package alarm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// AlarmEngine implements alarm processing, deduplication, and lifecycle management.
type AlarmEngine struct {
	store           AlarmStore
	redisStore      *RedisAlarmStore
	carrierRegistry *carrier.CarrierRegistry
	eventBus        event.EventBus
	logger          *zap.Logger
}

// NewAlarmEngine creates a new AlarmEngine.
func NewAlarmEngine(
	store AlarmStore,
	redisStore *RedisAlarmStore,
	carrierRegistry *carrier.CarrierRegistry,
	eventBus event.EventBus,
	logger *zap.Logger,
) *AlarmEngine {
	return &AlarmEngine{
		store:           store,
		redisStore:      redisStore,
		carrierRegistry: carrierRegistry,
		eventBus:        eventBus,
		logger:          logger,
	}
}

// Process handles an incoming alarm: maps severity, deduplicates, and persists.
func (e *AlarmEngine) Process(ctx context.Context, alarm *model.Alarm) error {
	// 1. Map severity via carrier adapter
	if e.carrierRegistry != nil {
		c, err := e.carrierRegistry.Get(alarm.Carrier)
		if err != nil {
			e.logger.Warn("carrier not found, using original severity",
				zap.String("carrier", string(alarm.Carrier)),
				zap.String("alarm_code", alarm.AlarmCode))
		} else {
			mapped := c.AlarmSeverityMapping(alarm.AlarmCode)
			if mapped != 0 {
				alarm.Severity = mapped
			}
		}
	}

	// 2. Deduplication check via Redis
	if e.redisStore != nil {
		exists, err := e.redisStore.Exists(ctx, alarm.DeviceSN, alarm.AlarmCode)
		if err != nil {
			e.logger.Warn("redis dedup check failed, falling back to DB",
				zap.Error(err))
		} else if exists {
			existingIDStr, _ := e.redisStore.Get(ctx, alarm.DeviceSN, alarm.AlarmCode)
			if existingIDStr != "" {
				existingID, parseErr := uuid.Parse(existingIDStr)
				if parseErr == nil {
					existing, getErr := e.store.GetActiveByID(ctx, existingID)
					if getErr == nil {
						existing.RaisedAt = alarm.RaisedAt
						existing.Severity = alarm.Severity
						existing.Description = alarm.Description
						if updateErr := e.store.UpdateActive(ctx, existing); updateErr != nil {
							return fmt.Errorf("update existing alarm: %w", updateErr)
						}
						e.logger.Debug("deduplicated alarm updated",
							zap.String("device_sn", alarm.DeviceSN),
							zap.String("alarm_code", alarm.AlarmCode))
						return nil
					}
				}
			}
		}
	}

	// 3. Also check DB in case Redis missed it
	existing, err := e.store.GetActiveByDeviceAndCode(ctx, alarm.DeviceSN, alarm.AlarmCode)
	if err == nil && existing != nil {
		existing.RaisedAt = alarm.RaisedAt
		existing.Severity = alarm.Severity
		existing.Description = alarm.Description
		if updateErr := e.store.UpdateActive(ctx, existing); updateErr != nil {
			return fmt.Errorf("update existing alarm: %w", updateErr)
		}
		if e.redisStore != nil {
			if redisErr := e.redisStore.Set(ctx, alarm.DeviceSN, alarm.AlarmCode, existing.ID.String()); redisErr != nil {
				e.logger.Warn("redis set alarm dedup key", zap.Error(redisErr))
			}
		}
		return nil
	}

	// 4. New alarm
	now := time.Now()
	alarm.ID = uuid.New()
	alarm.Status = model.AlarmActive
	alarm.CreatedAt = now
	alarm.UpdatedAt = now

	if err := e.store.SaveActive(ctx, alarm); err != nil {
		return fmt.Errorf("save active alarm: %w", err)
	}

	if e.redisStore != nil {
		if redisErr := e.redisStore.Set(ctx, alarm.DeviceSN, alarm.AlarmCode, alarm.ID.String()); redisErr != nil {
			e.logger.Warn("redis set new alarm dedup key", zap.Error(redisErr))
		}
	}

	// 5. Publish alarm.raised event
	if e.eventBus != nil {
		evt, err := event.NewEvent(event.SubjectAlarmRaised, alarm)
		if err == nil {
			if pubErr := e.eventBus.Publish(ctx, event.SubjectAlarmRaised, evt); pubErr != nil {
				e.logger.Warn("publish alarm.raised failed", zap.Error(pubErr))
			}
		}
	}

	e.logger.Info("new alarm raised",
		zap.String("alarm_id", alarm.ID.String()),
		zap.String("device_sn", alarm.DeviceSN),
		zap.String("alarm_code", alarm.AlarmCode),
		zap.Int("severity", int(alarm.Severity)))

	return nil
}

// Acknowledge marks an alarm as acknowledged.
func (e *AlarmEngine) Acknowledge(ctx context.Context, alarmID uuid.UUID, by string) error {
	alarm, err := e.store.GetActiveByID(ctx, alarmID)
	if err != nil {
		return fmt.Errorf("get alarm: %w", err)
	}

	if alarm.Status != model.AlarmActive {
		return fmt.Errorf("alarm is not in active state, current: %s", alarm.Status)
	}

	now := time.Now()
	alarm.Status = model.AlarmAcknowledged
	alarm.AcknowledgedAt = &now
	alarm.AcknowledgedBy = by

	if err := e.store.UpdateActive(ctx, alarm); err != nil {
		return fmt.Errorf("update alarm: %w", err)
	}

	if e.eventBus != nil {
		evt, err := event.NewEvent(event.SubjectAlarmAcknowledged, alarm)
		if err == nil {
			if pubErr := e.eventBus.Publish(ctx, event.SubjectAlarmAcknowledged, evt); pubErr != nil {
				e.logger.Warn("publish alarm.acknowledged event", zap.Error(pubErr))
			}
		}
	}

	e.logger.Info("alarm acknowledged",
		zap.String("alarm_id", alarmID.String()),
		zap.String("by", by))

	return nil
}

// Clear marks an alarm as cleared, archives to history, and removes from active.
func (e *AlarmEngine) Clear(ctx context.Context, alarmID uuid.UUID) error {
	alarm, err := e.store.GetActiveByID(ctx, alarmID)
	if err != nil {
		return fmt.Errorf("get alarm: %w", err)
	}

	if alarm.Status == model.AlarmCleared {
		return fmt.Errorf("alarm is already cleared")
	}

	now := time.Now()
	alarm.Status = model.AlarmCleared
	alarm.ClearedAt = &now

	// Archive to history
	if err := e.store.Archive(ctx, alarm); err != nil {
		return fmt.Errorf("archive alarm: %w", err)
	}

	// Remove from active
	if err := e.store.RemoveActive(ctx, alarmID); err != nil {
		return fmt.Errorf("remove active alarm: %w", err)
	}

	// Remove from Redis
	if e.redisStore != nil {
		if redisErr := e.redisStore.Delete(ctx, alarm.DeviceSN, alarm.AlarmCode); redisErr != nil {
			e.logger.Warn("redis delete alarm dedup key", zap.Error(redisErr))
		}
	}

	if e.eventBus != nil {
		evt, err := event.NewEvent(event.SubjectAlarmCleared, alarm)
		if err == nil {
			if pubErr := e.eventBus.Publish(ctx, event.SubjectAlarmCleared, evt); pubErr != nil {
				e.logger.Warn("publish alarm.cleared event", zap.Error(pubErr))
			}
		}
	}

	e.logger.Info("alarm cleared",
		zap.String("alarm_id", alarmID.String()),
		zap.String("device_sn", alarm.DeviceSN))

	return nil
}

// AutoClear clears an alarm by device serial and alarm code (device-initiated).
func (e *AlarmEngine) AutoClear(ctx context.Context, deviceSN, alarmCode string) error {
	alarm, err := e.store.GetActiveByDeviceAndCode(ctx, deviceSN, alarmCode)
	if err != nil {
		return fmt.Errorf("get alarm by device and code: %w", err)
	}
	if alarm == nil {
		return nil // No active alarm to clear
	}
	return e.Clear(ctx, alarm.ID)
}

// Store returns the underlying AlarmStore.
func (e *AlarmEngine) Store() AlarmStore {
	return e.store
}
