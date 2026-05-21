package alarm

import (
	"context"
	"fmt"
	"strconv"
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
	metrics         *AlarmMetrics
	filterEngine    *FilterEngine // optional; nil 时跳过过滤逻辑（向后兼容）
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

// SetMetrics attaches Prometheus metrics to the engine.
func (e *AlarmEngine) SetMetrics(m *AlarmMetrics) {
	e.metrics = m
}

// SetFilterEngine 注入过滤引擎，使所有入站告警在落库前先经 FilterEngine.ProcessAlarm。
// 调用方负责构造 FilterEngine（带 dispatcher / dead-letter / metrics）。nil 时跳过过滤。
func (e *AlarmEngine) SetFilterEngine(fe *FilterEngine) {
	e.filterEngine = fe
}

// severityLabel converts an AlarmSeverity to a Prometheus label string.
func severityLabel(s model.AlarmSeverity) string {
	return strconv.Itoa(int(s))
}

func applyIncomingAlarmState(target *model.Alarm, incoming *model.Alarm) {
	target.RaisedAt = incoming.RaisedAt
	target.Severity = incoming.Severity
	target.Description = incoming.Description
	if incoming.Status != "" {
		target.Status = incoming.Status
		target.AcknowledgedAt = incoming.AcknowledgedAt
		target.AcknowledgedBy = incoming.AcknowledgedBy
	}
}

// Process handles an incoming alarm: maps severity, deduplicates, and persists.
func (e *AlarmEngine) Process(ctx context.Context, alarm *model.Alarm) error {
	// 0. Apply user-defined filter rules (W2 T-0011 接生产路径)
	//    ignore     → 直接返回，不入库
	//    auto_clear → 直接返回，不入库（设备-发起的清除走 AutoClear 路径）
	//    auto_ack / notify_webhook → 仅修改 alarm 状态或派发 webhook，仍继续走 dedup + 入库
	if e.filterEngine != nil {
		result, err := e.filterEngine.ProcessAlarm(ctx, alarm, alarm.DeviceID)
		if err != nil {
			e.logger.Warn("filter engine processing failed, falling through to default flow",
				zap.Error(err),
				zap.String("alarm_identifier", alarm.AlarmIdentifier))
		} else if result != nil && result.Handled {
			switch result.Action {
			case FilterActionIgnore, FilterActionAutoClear:
				e.logger.Debug("alarm short-circuited by filter",
					zap.String("action", result.Action),
					zap.String("alarm_identifier", alarm.AlarmIdentifier))
				return nil
			}
		}
	}

	// 1. Map severity via carrier adapter
	if e.carrierRegistry != nil {
		c, err := e.carrierRegistry.Get(alarm.Carrier)
		if err != nil {
			e.logger.Warn("carrier not found, using original severity",
				zap.String("carrier", string(alarm.Carrier)),
				zap.String("alarm_identifier", alarm.AlarmIdentifier))
		} else {
			mapped := c.AlarmSeverityMapping(alarm.AlarmIdentifier)
			if mapped != 0 {
				alarm.Severity = mapped
			}
		}
	}

	// 2. Deduplication check via Redis
	if e.redisStore != nil {
		exists, err := e.redisStore.Exists(ctx, alarm.DeviceSN, alarm.AlarmIdentifier)
		if err != nil {
			e.logger.Warn("redis dedup check failed, falling back to DB",
				zap.Error(err))
		} else if exists {
			existingIDStr, _ := e.redisStore.Get(ctx, alarm.DeviceSN, alarm.AlarmIdentifier)
			if existingIDStr != "" {
				existingID, parseErr := uuid.Parse(existingIDStr)
				if parseErr == nil {
					existing, getErr := e.store.GetActiveByID(ctx, existingID)
					if getErr == nil {
						applyIncomingAlarmState(existing, alarm)
					if updateErr := e.store.UpdateActive(ctx, existing); updateErr != nil {
						return fmt.Errorf("update existing alarm: %w", updateErr)
					}
					e.logger.Debug("deduplicated alarm updated",
							zap.String("device_sn", alarm.DeviceSN),
							zap.String("alarm_identifier", alarm.AlarmIdentifier))
						return nil
					}
				}
			}
		}
	}

	// 3. Also check DB in case Redis missed it
	existing, err := e.store.GetActiveByDeviceAndIdentifier(ctx, alarm.DeviceSN, alarm.AlarmIdentifier)
	if err == nil && existing != nil {
		applyIncomingAlarmState(existing, alarm)
		if updateErr := e.store.UpdateActive(ctx, existing); updateErr != nil {
			return fmt.Errorf("update existing alarm: %w", updateErr)
		}
		if e.redisStore != nil {
			if redisErr := e.redisStore.Set(ctx, alarm.DeviceSN, alarm.AlarmIdentifier, existing.ID.String()); redisErr != nil {
				e.logger.Warn("redis set alarm dedup key", zap.Error(redisErr))
			}
		}
		return nil
	}

	// 4. New alarm
	now := time.Now()
	alarm.ID = uuid.New()
	if alarm.Status == "" {
		alarm.Status = model.AlarmActive
	}
	alarm.CreatedAt = now
	alarm.UpdatedAt = now

	if err := e.store.SaveActive(ctx, alarm); err != nil {
		return fmt.Errorf("save active alarm: %w", err)
	}

	// Record metrics for new alarm
	if e.metrics != nil {
		sev := severityLabel(alarm.Severity)
		e.metrics.ReceivedTotal.WithLabelValues(sev).Inc()
		e.metrics.ActiveTotal.WithLabelValues(sev, string(alarm.Carrier)).Inc()
	}

	if e.redisStore != nil {
		if redisErr := e.redisStore.Set(ctx, alarm.DeviceSN, alarm.AlarmIdentifier, alarm.ID.String()); redisErr != nil {
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
		zap.String("alarm_identifier", alarm.AlarmIdentifier),
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
	alarm.AcknowledgedBy = &by

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

	// Decrement active alarm gauge
	if e.metrics != nil {
		e.metrics.ActiveTotal.WithLabelValues(severityLabel(alarm.Severity), string(alarm.Carrier)).Dec()
	}

	// Remove from Redis

	// Remove from Redis
	if e.redisStore != nil {
		if redisErr := e.redisStore.Delete(ctx, alarm.DeviceSN, alarm.AlarmIdentifier); redisErr != nil {
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

// AutoClear clears an alarm by device serial and alarm identifier (device-initiated).
func (e *AlarmEngine) AutoClear(ctx context.Context, deviceSN, alarmIdentifier string) error {
	alarm, err := e.store.GetActiveByDeviceAndIdentifier(ctx, deviceSN, alarmIdentifier)
	if err != nil {
		return fmt.Errorf("get alarm by device and identifier: %w", err)
	}
	if alarm == nil {
		return nil // No active alarm to clear
	}
	return e.Clear(ctx, alarm.ID)
}

// UpdateFromSync updates an existing alarm's attributes during sync without publishing events.
// Used by the sync processor when remote alarm properties have changed.
func (e *AlarmEngine) UpdateFromSync(ctx context.Context, alarm *model.Alarm) error {
	alarm.LastUpdatedAt = time.Now()
	if err := e.store.UpdateActive(ctx, alarm); err != nil {
		return fmt.Errorf("sync update alarm: %w", err)
	}
	e.logger.Debug("alarm updated from sync",
		zap.String("alarm_id", alarm.ID.String()),
		zap.String("alarm_identifier", alarm.AlarmIdentifier),
		zap.String("device_sn", alarm.DeviceSN))
	return nil
}

// ClearBySync clears an alarm during sync (device no longer reports it) without publishing events.
// The sync processor will publish a batch event after processing all diffs.
func (e *AlarmEngine) ClearBySync(ctx context.Context, alarm *model.Alarm) error {
	now := time.Now()
	alarm.Status = model.AlarmCleared
	alarm.ClearedAt = &now
	if err := e.store.Archive(ctx, alarm); err != nil {
		return fmt.Errorf("sync archive alarm: %w", err)
	}
	if err := e.store.RemoveActive(ctx, alarm.ID); err != nil {
		return fmt.Errorf("sync remove active alarm: %w", err)
	}
	if e.redisStore != nil {
		if err := e.redisStore.Delete(ctx, alarm.DeviceSN, alarm.AlarmIdentifier); err != nil {
			e.logger.Warn("redis delete alarm on sync clear", zap.Error(err))
		}
	}
	if e.metrics != nil {
		e.metrics.ActiveTotal.WithLabelValues(severityLabel(alarm.Severity), string(alarm.Carrier)).Dec()
	}
	e.logger.Debug("alarm cleared from sync",
		zap.String("alarm_id", alarm.ID.String()),
		zap.String("alarm_identifier", alarm.AlarmIdentifier),
		zap.String("device_sn", alarm.DeviceSN))
	return nil
}

// UpdateByEvent handles a ChangedAlarm ExpeditedEvent notification.
// It updates mutable fields (Severity, Description, EventType, ProbableCause, AdditionalInfo)
// on the existing active alarm and publishes an alarm.updated event for northbound push.
// If no matching active alarm is found, it falls back to Process() (race with NewAlarm).
func (e *AlarmEngine) UpdateByEvent(ctx context.Context, alarm *model.Alarm) error {
	existing, err := e.store.GetActiveByDeviceAndIdentifier(ctx, alarm.DeviceSN, alarm.AlarmIdentifier)
	if err != nil {
		return fmt.Errorf("lookup alarm for update: %w", err)
	}

	// No matching active alarm — treat as new (race: NewAlarm not yet processed)
	if existing == nil {
		e.logger.Warn("UpdateByEvent: no active alarm found, falling back to Process",
			zap.String("device_sn", alarm.DeviceSN),
			zap.String("alarm_identifier", alarm.AlarmIdentifier))
		return e.Process(ctx, alarm)
	}

	// Update mutable fields
	oldSeverity := existing.Severity
	existing.Severity = alarm.Severity
	existing.Description = alarm.Description
	existing.EventType = alarm.EventType
	existing.ProbableCause = alarm.ProbableCause
	existing.LastUpdatedAt = time.Now()
	if alarm.AdditionalInfo != nil {
		if existing.AdditionalInfo == nil {
			existing.AdditionalInfo = make(map[string]string)
		}
		for k, v := range alarm.AdditionalInfo {
			existing.AdditionalInfo[k] = v
		}
	}

	if err := e.store.UpdateActive(ctx, existing); err != nil {
		return fmt.Errorf("update alarm by event: %w", err)
	}

	// Adjust metrics if severity changed
	if e.metrics != nil && oldSeverity != existing.Severity {
		e.metrics.ActiveTotal.WithLabelValues(severityLabel(oldSeverity), string(existing.Carrier)).Dec()
		e.metrics.ActiveTotal.WithLabelValues(severityLabel(existing.Severity), string(existing.Carrier)).Inc()
	}

	// Publish alarm.updated event for northbound push
	if e.eventBus != nil {
		evt, err := event.NewEvent(event.SubjectAlarmUpdated, existing)
		if err == nil {
			if pubErr := e.eventBus.Publish(ctx, event.SubjectAlarmUpdated, evt); pubErr != nil {
				e.logger.Warn("publish alarm.updated event", zap.Error(pubErr))
			}
		}
	}

	e.logger.Info("alarm updated by expedited event",
		zap.String("alarm_id", existing.ID.String()),
		zap.String("device_sn", existing.DeviceSN),
		zap.String("alarm_identifier", existing.AlarmIdentifier),
		zap.Int("old_severity", int(oldSeverity)),
		zap.Int("new_severity", int(existing.Severity)))

	return nil
}

// Store returns the underlying AlarmStore.
func (e *AlarmEngine) Store() AlarmStore {
	return e.store
}
