package alarm

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"go.opentelemetry.io/otel/attribute"
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

func resolveAlarmBusinessTime(alarm *model.Alarm, fallback time.Time) time.Time {
	if alarm == nil {
		return fallback
	}
	if !alarm.LastUpdatedAt.IsZero() {
		return alarm.LastUpdatedAt
	}
	if !alarm.RaisedAt.IsZero() {
		return alarm.RaisedAt
	}
	return fallback
}

func normalizeAlarmOccurrenceFields(alarm *model.Alarm, fallback time.Time) {
	if alarm.RaisedAt.IsZero() {
		alarm.RaisedAt = fallback
	}
	if alarm.AckCount <= 0 {
		alarm.AckCount = 1
	}
	if alarm.FirstRaisedAt.IsZero() {
		alarm.FirstRaisedAt = alarm.RaisedAt
	}
	if alarm.LastUpdatedAt.IsZero() {
		alarm.LastUpdatedAt = resolveAlarmBusinessTime(alarm, fallback)
	}
}

func applyIncomingAlarmState(target *model.Alarm, incoming *model.Alarm, fallback time.Time) {
	if target.FirstRaisedAt.IsZero() && !target.RaisedAt.IsZero() {
		target.FirstRaisedAt = target.RaisedAt
	}
	if target.RaisedAt.IsZero() && !incoming.RaisedAt.IsZero() {
		target.RaisedAt = incoming.RaisedAt
	}
	target.Severity = incoming.Severity
	target.Description = incoming.Description
	target.AckCount = max(target.AckCount, 1) + 1
	target.LastUpdatedAt = resolveAlarmBusinessTime(incoming, fallback)
	if incoming.Status != "" {
		target.Status = incoming.Status
		target.AcknowledgedAt = incoming.AcknowledgedAt
		target.AcknowledgedBy = incoming.AcknowledgedBy
		target.AckNote = incoming.AckNote
	}
}

// Process handles an incoming alarm: maps severity, deduplicates, and persists.
func (e *AlarmEngine) Process(ctx context.Context, alarm *model.Alarm) (err error) {
	// issue #20：告警 raise 路径补 span。卡顿/延迟告警（dedup 查 Redis、落库、发事件
	// 任一环节慢）此前在 trace 上不可见；named return + defer 统一把任一分支的错误
	// 记到 span，无需改每个 return。tracing no-op 时零开销。
	ctx, span := tracing.StartSpan(ctx, tracing.AlarmTracerName, "Alarm Process",
		attribute.String("alarm.device_sn", alarm.DeviceSN),
		attribute.String("alarm.identifier", alarm.AlarmIdentifier),
		attribute.String("alarm.carrier", string(alarm.Carrier)),
	)
	defer func() {
		tracing.RecordError(span, err)
		span.End()
	}()

	// 0. Apply user-defined filter rules (W2 T-0011 接生产路径)
	//    ignore     → 直接返回，不入库
	//    auto_clear → 直接返回，不入库（设备-发起的清除走 AutoClear 路径）
	//    auto_ack / notify_webhook → 仅修改 alarm 状态或派发 webhook，仍继续走 dedup + 入库
	var pendingEmailRule *AlarmFilterRule
	if e.filterEngine != nil {
		result, err := e.filterEngine.evaluateAlarm(ctx, alarm, alarm.DeviceID)
		if err != nil {
			e.logger.Warn("filter engine processing failed, falling through to default flow",
				zap.Error(err),
				zap.String("alarm_identifier", alarm.AlarmIdentifier))
		} else if result != nil && result.Handled {
			if result.Action == FilterActionNotifyEmail {
				pendingEmailRule = result.matchedRule
			}
			switch result.Action {
			case FilterActionIgnore:
				e.logger.Debug("alarm short-circuited by filter",
					zap.String("action", result.Action),
					zap.String("alarm_identifier", alarm.AlarmIdentifier))
				return nil
			case FilterActionAutoClear:
				if err := e.archiveAutoClearedAlarm(ctx, alarm); err != nil {
					return err
				}
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
			} else if alarm.Severity == 0 {
				alarm.Severity = model.AlarmWarning
			}
		}
	}

	dedupKey := activeAlarmMatchKey(alarm)

	// 2. Deduplication check via Redis
	if e.redisStore != nil {
		exists, err := e.redisStore.Exists(ctx, alarm.DeviceSN, dedupKey)
		if err != nil {
			e.logger.Warn("redis dedup check failed, falling back to DB",
				zap.Error(err))
		} else if exists {
			existingIDStr, _ := e.redisStore.Get(ctx, alarm.DeviceSN, dedupKey)
			if existingIDStr != "" {
				existingID, parseErr := uuid.Parse(existingIDStr)
				if parseErr == nil {
					existing, getErr := e.store.GetActiveByID(ctx, existingID)
					if getErr == nil {
						applyIncomingAlarmState(existing, alarm, time.Now())
						if updateErr := e.store.UpdateActive(ctx, existing); updateErr != nil {
							return fmt.Errorf("update existing alarm: %w", updateErr)
						}
						e.syncDeviceSeverityStatsAsync(existing.DeviceID)
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
	existing, err := loadMatchingActiveAlarm(ctx, e.store, alarm)
	if err == nil && existing != nil {
		applyIncomingAlarmState(existing, alarm, time.Now())
		if updateErr := e.store.UpdateActive(ctx, existing); updateErr != nil {
			return fmt.Errorf("update existing alarm: %w", updateErr)
		}
		e.syncDeviceSeverityStatsAsync(existing.DeviceID)
		if e.redisStore != nil {
			if redisErr := e.redisStore.Set(ctx, alarm.DeviceSN, dedupKey, existing.ID.String()); redisErr != nil {
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
	normalizeAlarmOccurrenceFields(alarm, now)
	alarm.CreatedAt = now
	alarm.UpdatedAt = now

	if err := e.store.SaveActive(ctx, alarm); err != nil {
		return fmt.Errorf("save active alarm: %w", err)
	}
	if e.redisStore != nil && alarm.DeviceID.String() != "00000000-0000-0000-0000-000000000000" {
		_ = e.redisStore.IncrementActiveAlarmCount(ctx, alarm.DeviceID.String())
	}

	// Record metrics for new alarm
	if e.metrics != nil {
		sev := severityLabel(alarm.Severity)
		e.metrics.ReceivedTotal.WithLabelValues(sev).Inc()
		e.metrics.ActiveTotal.WithLabelValues(sev, string(alarm.Carrier)).Inc()
	}

	if e.redisStore != nil {
		if redisErr := e.redisStore.Set(ctx, alarm.DeviceSN, dedupKey, alarm.ID.String()); redisErr != nil {
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
	if pendingEmailRule != nil {
		e.filterEngine.dispatchEmail(ctx, alarm, pendingEmailRule)
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

	return e.clearActiveAlarm(ctx, alarm)
}

func (e *AlarmEngine) clearActiveAlarm(ctx context.Context, alarm *model.Alarm) error {
	now := time.Now()
	alarm.Status = model.AlarmCleared
	alarm.ClearedAt = &now

	// Archive to history
	if err := e.store.Archive(ctx, alarm); err != nil {
		return fmt.Errorf("archive alarm: %w", err)
	}

	// Remove from active
	if err := e.store.RemoveActive(ctx, alarm.ID); err != nil {
		return fmt.Errorf("remove active alarm: %w", err)
	}
	if e.redisStore != nil && alarm.DeviceID.String() != "00000000-0000-0000-0000-000000000000" {
		_ = e.redisStore.DecrementActiveAlarmCount(ctx, alarm.DeviceID.String())
	}

	// Decrement active alarm gauge
	if e.metrics != nil {
		e.metrics.ActiveTotal.WithLabelValues(severityLabel(alarm.Severity), string(alarm.Carrier)).Dec()
	}

	// Remove from Redis

	// Remove from Redis
	if e.redisStore != nil {
		if redisErr := e.redisStore.Delete(ctx, alarm.DeviceSN, activeAlarmMatchKey(alarm)); redisErr != nil {
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

	if e.filterEngine != nil {
		e.filterEngine.NotifyClearedAlarm(ctx, alarm, alarm.DeviceID)
	}

	e.logger.Info("alarm cleared",
		zap.String("alarm_id", alarm.ID.String()),
		zap.String("device_sn", alarm.DeviceSN))

	return nil
}

// AutoClear clears a device-originated alarm using the same instance-level key
// as creation and update flows.
func (e *AlarmEngine) AutoClear(ctx context.Context, alarm *model.Alarm) error {
	alarms, err := e.store.GetActiveByDeviceSN(ctx, alarm.DeviceSN)
	if err != nil {
		return fmt.Errorf("get alarm by device and identifier: %w", err)
	}

	existing := findMatchingActiveAlarm(alarms, alarm)
	if existing == nil {
		if e.metrics != nil {
			e.metrics.ReconciliationTotal.WithLabelValues("clear_exact_miss").Inc()
		}
		candidates := findActiveAlarmsByIdentifier(alarms, alarm)
		switch len(candidates) {
		case 0:
			e.logger.Debug("alarm clear exact match missed",
				zap.String("device_sn", alarm.DeviceSN),
				zap.String("alarm_identifier", alarm.AlarmIdentifier),
				zap.String("match_key", activeAlarmMatchKey(alarm)))
			return nil
		case 1:
			existing = candidates[0]
			if e.metrics != nil {
				e.metrics.ReconciliationTotal.WithLabelValues("clear_unique_fallback").Inc()
			}
			e.logger.Info("alarm clear resolved by unique identifier fallback",
				zap.String("device_sn", alarm.DeviceSN),
				zap.String("alarm_identifier", alarm.AlarmIdentifier),
				zap.String("alarm_id", existing.ID.String()),
				zap.String("match_key", activeAlarmMatchKey(alarm)))
		default:
			candidateIDs := make([]string, 0, len(candidates))
			for _, candidate := range candidates {
				candidateIDs = append(candidateIDs, candidate.ID.String())
			}
			if e.metrics != nil {
				e.metrics.ReconciliationTotal.WithLabelValues("clear_ambiguous").Inc()
			}
			e.logger.Warn("alarm clear ambiguous; sync requested",
				zap.String("device_sn", alarm.DeviceSN),
				zap.String("alarm_identifier", alarm.AlarmIdentifier),
				zap.String("match_key", activeAlarmMatchKey(alarm)),
				zap.Int("candidate_count", len(candidates)),
				zap.Strings("candidate_ids", candidateIDs))
			return nil
		}
	}

	clearedBy := "system"
	clearNote := "auto-cleared"
	existing.ClearedBy = &clearedBy
	existing.ClearNote = &clearNote
	return e.clearActiveAlarm(ctx, existing)
}

func (e *AlarmEngine) archiveAutoClearedAlarm(ctx context.Context, alarm *model.Alarm) error {
	now := time.Now()
	clearNote := "auto-cleared by alarm rule"
	if alarm.ClearNote != nil && *alarm.ClearNote != "" {
		clearNote = *alarm.ClearNote
	}
	clearedBy := "system"
	if alarm.ClearedBy != nil && *alarm.ClearedBy != "" {
		clearedBy = *alarm.ClearedBy
	}

	existing, err := loadMatchingActiveAlarm(ctx, e.store, alarm)
	if err != nil {
		if !errors.Is(err, commonerrors.ErrNotFound) {
			return fmt.Errorf("lookup auto-cleared alarm: %w", err)
		}
	}

	if existing != nil {
		existing.Status = model.AlarmCleared
		existing.ClearedAt = &now
		existing.ClearedBy = &clearedBy
		existing.ClearNote = &clearNote
		if err := e.store.Archive(ctx, existing); err != nil {
			return fmt.Errorf("archive existing auto-cleared alarm: %w", err)
		}
		if err := e.store.RemoveActive(ctx, existing.ID); err != nil {
			return fmt.Errorf("remove existing auto-cleared alarm: %w", err)
		}
		if e.redisStore != nil {
			if err := e.redisStore.Delete(ctx, existing.DeviceSN, activeAlarmMatchKey(existing)); err != nil {
				e.logger.Warn("redis delete auto-cleared alarm", zap.Error(err))
			}
		}
		if e.metrics != nil {
			e.metrics.ActiveTotal.WithLabelValues(severityLabel(existing.Severity), string(existing.Carrier)).Dec()
		}
		return nil
	}

	archived := *alarm
	if archived.ID == uuid.Nil {
		archived.ID = uuid.New()
	}
	if archived.RaisedAt.IsZero() {
		archived.RaisedAt = now
	}
	if archived.FirstRaisedAt.IsZero() {
		archived.FirstRaisedAt = archived.RaisedAt
	}
	archived.Status = model.AlarmCleared
	archived.ClearedAt = &now
	archived.ClearedBy = &clearedBy
	archived.ClearNote = &clearNote
	if archived.CreatedAt.IsZero() {
		archived.CreatedAt = now
	}
	archived.UpdatedAt = now
	if archived.LastUpdatedAt.IsZero() {
		archived.LastUpdatedAt = now
	}

	if err := e.store.Archive(ctx, &archived); err != nil {
		return fmt.Errorf("archive new auto-cleared alarm: %w", err)
	}

	return nil
}

// UpdateFromSync updates an existing alarm's attributes during sync without publishing events.
// Used by the sync processor when remote alarm properties have changed.
func (e *AlarmEngine) UpdateFromSync(ctx context.Context, alarm *model.Alarm) error {
	alarm.AckCount = max(alarm.AckCount, 1) + 1
	alarm.LastUpdatedAt = resolveAlarmBusinessTime(alarm, time.Now())
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
func (e *AlarmEngine) ClearBySync(ctx context.Context, alarm *model.Alarm) (err error) {
	// issue #20：告警 clear 路径补 span，让"该清未清"的延迟清除（archive/remove 慢）可定位。
	ctx, span := tracing.StartSpan(ctx, tracing.AlarmTracerName, "Alarm ClearBySync",
		attribute.String("alarm.device_sn", alarm.DeviceSN),
		attribute.String("alarm.identifier", alarm.AlarmIdentifier),
	)
	defer func() {
		tracing.RecordError(span, err)
		span.End()
	}()

	now := time.Now()
	alarm.Status = model.AlarmCleared
	alarm.ClearedAt = &now
	if err := e.store.Archive(ctx, alarm); err != nil {
		return fmt.Errorf("sync archive alarm: %w", err)
	}
	if err := e.store.RemoveActive(ctx, alarm.ID); err != nil {
		return fmt.Errorf("sync remove active alarm: %w", err)
	}
	if e.redisStore != nil && alarm.DeviceID.String() != "00000000-0000-0000-0000-000000000000" {
		_ = e.redisStore.DecrementActiveAlarmCount(ctx, alarm.DeviceID.String())
	}
	if e.redisStore != nil {
		if err := e.redisStore.Delete(ctx, alarm.DeviceSN, activeAlarmMatchKey(alarm)); err != nil {
			e.logger.Warn("redis delete alarm on sync clear", zap.Error(err))
		}
	}
	if e.metrics != nil {
		e.metrics.ActiveTotal.WithLabelValues(severityLabel(alarm.Severity), string(alarm.Carrier)).Dec()
	}
	if e.filterEngine != nil {
		e.filterEngine.NotifyClearedAlarm(ctx, alarm, alarm.DeviceID)
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
	existing, err := loadMatchingActiveAlarm(ctx, e.store, alarm)
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
	existing.AckCount = max(existing.AckCount, 1) + 1
	existing.LastUpdatedAt = resolveAlarmBusinessTime(alarm, time.Now())
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
	e.syncDeviceSeverityStatsAsync(existing.DeviceID)

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
