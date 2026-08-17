package alarm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

const (
	alarmEmailRealtimeRaisedQueue  = "alarm-email-realtime-raised"
	alarmEmailRealtimeClearedQueue = "alarm-email-realtime-cleared"
)

type AlarmEmailRealtimeSubscriber struct {
	repository AlarmEmailSchedulerRepository
	logger     *zap.Logger
}

func NewAlarmEmailRealtimeSubscriber(repository AlarmEmailSchedulerRepository, logger *zap.Logger) *AlarmEmailRealtimeSubscriber {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AlarmEmailRealtimeSubscriber{repository: repository, logger: logger.Named("alarm-email-realtime")}
}

func (s *AlarmEmailRealtimeSubscriber) Subscribe(bus event.EventBus) error {
	if bus == nil || s.repository == nil {
		return fmt.Errorf("alarm email realtime subscriber dependencies are required")
	}
	raisedSubscription, err := bus.QueueSubscribe(event.SubjectAlarmEmailRaised, alarmEmailRealtimeRaisedQueue, s.handleRaised)
	if err != nil {
		return fmt.Errorf("subscribe realtime alarm raised email: %w", err)
	}
	if _, err := bus.QueueSubscribe(event.SubjectAlarmEmailCleared, alarmEmailRealtimeClearedQueue, s.handleCleared); err != nil {
		if unsubscribeErr := raisedSubscription.Unsubscribe(); unsubscribeErr != nil {
			s.logger.Warn("unsubscribe partial realtime alarm email subscription", zap.Error(unsubscribeErr))
		}
		return fmt.Errorf("subscribe realtime alarm cleared email: %w", err)
	}
	s.logger.Info("realtime alarm email subscriber started")
	return nil
}

func (s *AlarmEmailRealtimeSubscriber) handleCleared(ctx context.Context, evt event.Event) error {
	var alarm model.Alarm
	if err := evt.DecodePayload(&alarm); err != nil {
		return fmt.Errorf("decode realtime cleared alarm email event: %w", err)
	}
	if alarm.ID == uuid.Nil || alarm.RaisedAt.IsZero() || alarm.ClearedAt == nil || alarm.ClearedAt.IsZero() {
		return fmt.Errorf("realtime cleared alarm email event is missing alarm identity, raised_at or cleared_at")
	}
	setting, err := s.repository.GetGlobalSetting(ctx)
	if err != nil {
		return fmt.Errorf("load realtime cleared alarm email setting: %w", err)
	}
	if !setting.Enabled {
		return nil
	}
	subscriptions, err := s.repository.ListEnabledSubscriptions(ctx, AlarmEmailIntervalRealtime)
	if err != nil {
		return fmt.Errorf("list realtime cleared alarm email subscriptions: %w", err)
	}
	window := alarmEmailRealtimeWindow(*alarm.ClearedAt)
	for i := range subscriptions {
		eligibleAt, err := AlarmEmailRealtimeScheduledAt(alarm.RaisedAt, subscriptions[i].ToleranceMinutes)
		if err != nil {
			s.logger.Warn("skip invalid realtime alarm email subscription",
				zap.String("subscription_id", subscriptions[i].ID.String()),
				zap.Error(err))
			continue
		}
		// The delayed raised-alarm run queries both active and cleared storage.
		// A clear before that run is therefore already covered and must not create
		// a duplicate email. Clears at or after the eligibility boundary need a
		// separate recovery notification.
		if alarm.ClearedAt.Before(eligibleAt) {
			continue
		}
		if _, _, err := s.repository.EnqueueRun(ctx, &subscriptions[i], setting.DefaultRecipients, window, *alarm.ClearedAt); err != nil {
			return fmt.Errorf("enqueue realtime cleared alarm email subscription %s: %w", subscriptions[i].ID, err)
		}
	}
	return nil
}

func (s *AlarmEmailRealtimeSubscriber) handleRaised(ctx context.Context, evt event.Event) error {
	var alarm model.Alarm
	if err := evt.DecodePayload(&alarm); err != nil {
		return fmt.Errorf("decode realtime alarm email event: %w", err)
	}
	if alarm.ID == uuid.Nil || alarm.RaisedAt.IsZero() {
		return fmt.Errorf("realtime alarm email event is missing alarm identity or raised_at")
	}
	setting, err := s.repository.GetGlobalSetting(ctx)
	if err != nil {
		return fmt.Errorf("load realtime alarm email setting: %w", err)
	}
	if !setting.Enabled {
		return nil
	}
	subscriptions, err := s.repository.ListEnabledSubscriptions(ctx, AlarmEmailIntervalRealtime)
	if err != nil {
		return fmt.Errorf("list realtime alarm email subscriptions: %w", err)
	}
	// PostgreSQL stores timestamptz at microsecond precision. Use one exact,
	// non-overlapping storage bucket so adjacent realtime events cannot be read
	// by two different runs.
	window := alarmEmailRealtimeWindow(alarm.RaisedAt)
	for i := range subscriptions {
		scheduledAt, err := AlarmEmailRealtimeScheduledAt(alarm.RaisedAt, subscriptions[i].ToleranceMinutes)
		if err != nil {
			s.logger.Warn("skip invalid realtime alarm email subscription",
				zap.String("subscription_id", subscriptions[i].ID.String()),
				zap.Error(err))
			continue
		}
		if _, _, err := s.repository.EnqueueRun(ctx, &subscriptions[i], setting.DefaultRecipients, window, scheduledAt); err != nil {
			return fmt.Errorf("enqueue realtime alarm email subscription %s: %w", subscriptions[i].ID, err)
		}
	}
	return nil
}

func alarmEmailRealtimeWindow(at time.Time) AlarmEmailWindow {
	start := at.Truncate(time.Microsecond)
	return AlarmEmailWindow{Start: start, End: start.Add(time.Microsecond)}
}
