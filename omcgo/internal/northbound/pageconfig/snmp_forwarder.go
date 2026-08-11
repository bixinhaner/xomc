package pageconfig

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	nbsnmp "github.com/omcgo/omcgo/internal/northbound/snmp"
)

const snmpForwarderQueue = "northbound-page-config-snmp"
const snmpMaxNotificationID = 2147483647

type SNMPAlarmForwarder struct {
	svc    *Service
	bus    event.EventBus
	logger *zap.Logger

	mu   sync.Mutex
	subs []event.Subscription
}

func NewSNMPAlarmForwarder(svc *Service, bus event.EventBus, logger *zap.Logger) *SNMPAlarmForwarder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SNMPAlarmForwarder{svc: svc, bus: bus, logger: logger.Named("page-config-snmp-forwarder")}
}

func (f *SNMPAlarmForwarder) Start() error {
	if f == nil || f.svc == nil || f.bus == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.subs) > 0 {
		return nil
	}
	for _, subject := range alarmRealtimeSubjects() {
		subject := subject
		sub, err := f.bus.QueueSubscribe(subject, alarmRealtimeQueueForSubject(subject), func(ctx context.Context, evt event.Event) error {
			return f.handleAlarmEvent(ctx, subject, evt)
		})
		if err != nil {
			for _, existing := range f.subs {
				_ = existing.Unsubscribe()
			}
			f.subs = nil
			return fmt.Errorf("subscribe SNMP alarm forwarder %s: %w", subject, err)
		}
		f.subs = append(f.subs, sub)
	}
	f.logger.Info("northbound page-config SNMP alarm forwarder started")
	return nil
}

func (f *SNMPAlarmForwarder) Stop() error {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var firstErr error
	for _, sub := range f.subs {
		if err := sub.Unsubscribe(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	f.subs = nil
	return firstErr
}

func (f *SNMPAlarmForwarder) HandleAlarmEvent(ctx context.Context, subject string, evt event.Event) error {
	return f.handleAlarmEvent(ctx, subject, evt)
}

func (f *SNMPAlarmForwarder) handleAlarmEvent(ctx context.Context, subject string, evt event.Event) error {
	var alarm model.Alarm
	if err := evt.DecodePayload(&alarm); err != nil {
		f.logger.Warn("decode alarm event for SNMP forward failed",
			zap.String("subject", subject),
			zap.String("event_id", evt.ID),
			zap.Error(err))
		return nil
	}
	snmpAlarm := snmpAlarmFromModel(alarm, subject)
	events, err := f.svc.SendSNMPAlarm(ctx, snmpAlarm)
	if err != nil {
		f.logger.Warn("send SNMP alarm failed",
			zap.String("subject", subject),
			zap.String("alarm_id", alarm.ID.String()),
			zap.Error(err))
		return nil
	}
	if len(events) > 0 {
		f.logger.Debug("SNMP alarm forwarded",
			zap.String("subject", subject),
			zap.String("alarm_id", alarm.ID.String()),
			zap.Int("target_count", len(events)))
	}
	return nil
}

func snmpAlarmFromModel(alarm model.Alarm, subject string) *nbsnmp.AlarmEvent {
	occurTime := alarm.RaisedAt
	severity := alarmSeverityText(alarm.Severity)
	notificationType := "1"
	if subject == event.SubjectAlarmCleared || alarm.Status == model.AlarmCleared {
		notificationType = "0"
		if alarm.ClearedAt != nil && !alarm.ClearedAt.IsZero() {
			occurTime = *alarm.ClearedAt
		}
	}
	if occurTime.IsZero() {
		occurTime = time.Now()
	}

	extra := make(map[string]string, len(alarm.AdditionalInfo)+14)
	for k, v := range alarm.AdditionalInfo {
		extra[k] = v
	}
	extra["notificationType"] = notificationType
	extra["notificationID"] = strconv.Itoa(snmpNotificationIDFromAlarm(alarm))
	extra["alarmUniqueId"] = firstNonEmpty(alarm.AlarmIdentifier, alarm.ID.String())
	extra["equipmentSDN"] = alarm.DeviceSN
	extra["equipmentName"] = firstNonEmpty(stringPtrValue(alarm.DeviceName), alarm.DeviceSN)
	extra["equipmentClass"] = firstNonEmpty(stringPtrValue(alarm.Technology), string(alarm.Carrier))
	extra["objectSDN"] = firstNonEmpty(stringPtrValue(alarm.AlarmSource), alarm.DeviceSN)
	extra["objectInstanceName"] = firstNonEmpty(stringPtrValue(alarm.NetworkLocation), extra["objectSDN"])
	extra["objectClass"] = firstNonEmpty(stringPtrValue(alarm.EventType), alarm.AlarmType)
	extra["additionalText"] = alarm.Description
	extra["deviceVendorOUI"] = firstNonEmpty(extra["deviceVendorOUI"], "Baicells")
	extra["specificProblemID"] = firstNonEmpty(alarm.AlarmIdentifier, alarm.AlarmType)
	extra["specificProblem"] = firstNonEmpty(alarm.Description, alarm.AlarmType)
	extra["alarmType"] = alarm.AlarmType
	extra["probableCause"] = firstNonEmpty(stringPtrValue(alarm.ProbableCause), stringPtrValue(alarm.ExplicitCause))
	extra["additionalInformation"] = compactAdditionalInfo(alarm.AdditionalInfo)

	alarmID := strings.TrimSpace(alarm.ID.String())
	if alarmID == "" || alarmID == "00000000-0000-0000-0000-000000000000" {
		alarmID = alarm.AlarmIdentifier
	}
	return &nbsnmp.AlarmEvent{
		AlarmID:      alarmID,
		DeviceSerial: alarm.DeviceSN,
		Severity:     severity,
		AlarmType:    alarm.AlarmType,
		Carrier:      string(alarm.Carrier),
		OccurTime:    occurTime,
		Extra:        extra,
	}
}

func snmpNotificationIDFromAlarm(alarm model.Alarm) int {
	if alarm.AdditionalInfo != nil {
		for _, key := range []string{"notificationID", "notification_id", "alarmSeq", "alarm_seq", "sequence_id"} {
			if value := strings.TrimSpace(alarm.AdditionalInfo[key]); value != "" {
				if parsed, ok := parseSNMPNotificationID(value); ok {
					return parsed
				}
				if digits := digitsOnly(value); digits != "" {
					if parsed, ok := parseSNMPNotificationID(digits); ok {
						return parsed
					}
				}
			}
		}
	}
	seed := firstNonEmpty(alarm.ID.String(), alarm.AlarmIdentifier, alarm.DeviceSN, alarm.AlarmType)
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	id := int(hash.Sum32() & uint32(snmpMaxNotificationID))
	if id <= 0 {
		return 1
	}
	return id
}

func parseSNMPNotificationID(value string) (int, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	if parsed > snmpMaxNotificationID {
		parsed = ((parsed - 1) % snmpMaxNotificationID) + 1
	}
	return int(parsed), true
}

func alarmSeverityText(severity model.AlarmSeverity) string {
	switch severity {
	case model.AlarmCritical:
		return "critical"
	case model.AlarmMajor:
		return "major"
	case model.AlarmMinor:
		return "minor"
	case model.AlarmWarning:
		return "warning"
	default:
		return "unknown"
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func compactAdditionalInfo(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		parts = append(parts, key+"="+value)
	}
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}
