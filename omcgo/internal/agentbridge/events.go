package agentbridge

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

const (
	XOMCPackageKey     = "com.baicells.xomc"
	XOMCPackageVersion = "1.0.0"
	XOMCPackageDigest  = "sha256:094a1ea9b43f83b92416c9c14933b54593d4cae9893495ae294c228d5bb43b06"
)

type HandbookMetadataProvider interface {
	HandbookMetadata() (map[string]any, error)
}

type EventSubscriber struct {
	bus      event.EventBus
	pool     *pgxpool.Pool
	outbox   *OutboxRepository
	targets  RuntimeTargetProvider
	handbook HandbookMetadataProvider
	logger   *zap.Logger
}

func NewEventSubscriber(bus event.EventBus, pool *pgxpool.Pool, outbox *OutboxRepository, targets RuntimeTargetProvider, handbook HandbookMetadataProvider, logger *zap.Logger) *EventSubscriber {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EventSubscriber{bus: bus, pool: pool, outbox: outbox, targets: targets, handbook: handbook, logger: logger.Named("agent-bridge-events")}
}

func (s *EventSubscriber) Subscribe() error {
	if s.bus == nil || s.outbox == nil || s.targets == nil || s.handbook == nil {
		return fmt.Errorf("agent event subscriber dependencies are incomplete")
	}
	if _, err := s.bus.QueueSubscribe(event.SubjectTaskFailed, "agent-bridge-task-failed", s.handleTaskFailed); err != nil {
		return fmt.Errorf("subscribe %s for agent bridge: %w", event.SubjectTaskFailed, err)
	}
	if _, err := s.bus.QueueSubscribe(event.SubjectDeviceAccessReviewRequired, "agent-bridge-access-review", s.handleAccessReview); err != nil {
		return fmt.Errorf("subscribe %s for agent bridge: %w", event.SubjectDeviceAccessReviewRequired, err)
	}
	return nil
}

func (s *EventSubscriber) HandleSevereAlarm(ctx context.Context, alarm *model.Alarm) error {
	if alarm == nil {
		return nil
	}
	evt, err := event.NewEvent(event.SubjectAlarmRaised, alarm)
	if err != nil {
		return fmt.Errorf("create severe alarm bridge event: %w", err)
	}
	return s.handleAlarmRaised(ctx, evt)
}

func (s *EventSubscriber) handleAccessReview(ctx context.Context, evt event.Event) error {
	var payload map[string]any
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode access review for agent bridge: %w", err)
	}
	resources := []ResourceRef{}
	if id := mapString(payload, "candidate_id"); id != "" {
		resources = append(resources, ResourceRef{Type: "candidate", ID: id, Role: "candidate"})
	}
	if id := mapString(payload, "device_id"); id != "" {
		resources = append(resources, ResourceRef{Type: "device", ID: id, Role: "device"})
	}
	if len(resources) == 0 {
		return fmt.Errorf("access review event has no candidate or device resource")
	}
	return s.enqueue(ctx, evt, "omc.device.access-review-required.v1", resources, payload)
}

func (s *EventSubscriber) handleAlarmRaised(ctx context.Context, evt event.Event) error {
	var alarm model.Alarm
	if err := evt.DecodePayload(&alarm); err != nil {
		return fmt.Errorf("decode severe alarm for agent bridge: %w", err)
	}
	if alarm.Severity != model.AlarmCritical && alarm.Severity != model.AlarmMajor && alarm.Severity != model.AlarmSeverity(31001) && alarm.Severity != model.AlarmSeverity(31002) {
		return nil
	}
	resources := []ResourceRef{{Type: "alarm", ID: alarm.ID.String(), Role: "alarm", Label: alarm.Description}}
	if alarm.DeviceID != (uuid.UUID{}) {
		resources = append(resources, ResourceRef{Type: "device", ID: alarm.DeviceID.String(), Role: "device", Label: alarm.DeviceSN})
	}
	severity := "major"
	if alarm.Severity == model.AlarmCritical || alarm.Severity == model.AlarmSeverity(31001) {
		severity = "critical"
	}
	data := map[string]any{"severity": severity, "alarmType": alarm.AlarmType, "alarmIdentifier": alarm.AlarmIdentifier, "description": alarm.Description, "deviceSerialNumber": alarm.DeviceSN, "raisedAt": alarm.RaisedAt}
	return s.enqueue(ctx, evt, "omc.alarm.severe-raised.v1", resources, data)
}

func mapString(value map[string]any, key string) string {
	return strings.TrimSpace(fmt.Sprint(value[key]))
}

func (s *EventSubscriber) enqueue(ctx context.Context, evt event.Event, eventType string, resources []ResourceRef, data map[string]any) error {
	target, err := s.targets.GetRuntimeTarget(ctx)
	if err != nil {
		return fmt.Errorf("load agent bridge target for event: %w", err)
	}
	if !target.Enabled {
		return nil
	}
	metadata, err := s.handbook.HandbookMetadata()
	if err != nil {
		return fmt.Errorf("resolve agent handbook metadata: %w", err)
	}
	digest, _ := metadata["handbookDigest"].(string)
	if strings.TrimSpace(digest) == "" {
		return fmt.Errorf("resolve agent handbook metadata: handbook digest is missing")
	}
	traceID := strings.TrimSpace(evt.Metadata["trace_id"])
	if traceID == "" {
		traceID = evt.ID
	}
	data["apiHandbook"] = metadata
	return s.outbox.Enqueue(ctx, ConnectorEventEnvelope{ContractVersion: ContractVersion, EventID: evt.ID, EventType: eventType, Source: target.InstanceName, OccurredAt: evt.Timestamp.UTC(), TraceID: traceID, IntegrationPack: PackRef{Key: XOMCPackageKey, Version: XOMCPackageVersion, Digest: XOMCPackageDigest}, HandbookDigest: digest, Resources: resources, Data: data})
}

func (s *EventSubscriber) handleTaskFailed(ctx context.Context, evt event.Event) error {
	var value task.Task
	if err := evt.DecodePayload(&value); err != nil {
		return fmt.Errorf("decode task failure for agent bridge: %w", err)
	}
	target, err := s.targets.GetRuntimeTarget(ctx)
	if err != nil {
		return fmt.Errorf("load agent bridge target for event: %w", err)
	}
	if !target.Enabled {
		return nil
	}
	handbookMetadata, err := s.handbook.HandbookMetadata()
	if err != nil {
		return fmt.Errorf("resolve agent handbook metadata: %w", err)
	}
	handbookDigest, _ := handbookMetadata["handbookDigest"].(string)
	if strings.TrimSpace(handbookDigest) == "" {
		return fmt.Errorf("resolve agent handbook metadata: handbook digest is missing")
	}
	deviceID, err := s.deviceID(ctx, value.DeviceSN)
	if err != nil {
		return err
	}
	resources := []ResourceRef{{Type: "task", ID: value.ID, Role: "task", Label: value.Description}}
	if deviceID != "" {
		resources = append(resources, ResourceRef{Type: "device", ID: deviceID, Role: "device", Label: value.DeviceSN})
	}
	traceID := strings.TrimSpace(evt.Metadata["trace_id"])
	if traceID == "" {
		traceID = evt.ID
	}
	envelope := ConnectorEventEnvelope{
		ContractVersion: ContractVersion,
		EventID:         evt.ID, EventType: "omc.task.failed.v1", Source: target.InstanceName,
		OccurredAt: evt.Timestamp.UTC(), TraceID: traceID,
		IntegrationPack: PackRef{Key: XOMCPackageKey, Version: XOMCPackageVersion, Digest: XOMCPackageDigest},
		HandbookDigest:  handbookDigest, Resources: resources,
		Data: map[string]any{
			"taskType": value.Method, "errorCode": value.ErrorCode, "failureMessage": value.ErrorMessage,
			"deviceSerialNumber": value.DeviceSN, "source": value.Source, "retryCount": value.RetryCount,
			"apiHandbook": handbookMetadata,
		},
	}
	if err := s.outbox.Enqueue(ctx, envelope); err != nil {
		return fmt.Errorf("enqueue task failure for agent analysis: %w", err)
	}
	return nil
}

func (s *EventSubscriber) deviceID(ctx context.Context, serialNumber string) (string, error) {
	query, args, err := psql.Select("id").From("devices").
		Where(sq.Eq{"serial_number": serialNumber}).Where(sq.Expr("deleted_at IS NULL")).Limit(1).ToSql()
	if err != nil {
		return "", fmt.Errorf("build agent event device lookup: %w", err)
	}
	var id string
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return "", nil
		}
		return "", fmt.Errorf("resolve task device for agent event: %w", err)
	}
	return id, nil
}
