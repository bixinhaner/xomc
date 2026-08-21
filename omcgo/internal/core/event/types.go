// Package event 提供事件驱动架构的核心抽象。
// 包括事件封装类型、EventBus 接口和两种实现：
//   - ChannelEventBus：单进程内内存通道，用于测试
//   - NATSEventBus：基于 NATS JetStream 的跨服务广播，用于生产
//
// 事件主题常量定义在 subjects.go。
package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MetadataDeviceSN identifies the originating device without changing an
// event's protocol payload. It is primarily used by device-originated RPC
// events whose CWMP body does not contain the serial number.
const MetadataDeviceSN = "device_sn"

// MetadataGPVOwner identifies the intended owner for a GPV response without
// forcing secondary consumers to decode the full parameter_values payload.
const MetadataGPVOwner = "gpv_owner"

const (
	GPVOwnerParamSync        = "param_sync"
	GPVOwnerAlarmSync        = "alarm_sync"
	GPVOwnerDeviceAccess     = "device_access"
	GPVOwnerSoftwareRollback = "software_rollback"
	GPVOwnerProvision        = "provision"
	GPVOwnerDeviceRPC        = "device_rpc"
)

// Event 是所有事件的标准封装。
// Payload 为 JSON 序列化的事件载荷，封装前后可通过 NewEvent 和 DecodePayload 操作。
// Metadata 可附加平台信息（如 trace_id、来源服务）。
type Event struct {
	ID        string            `json:"id"`
	Subject   string            `json:"subject"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// EventHandler processes a received event.
type EventHandler func(ctx context.Context, event Event) error

// SysConfigSavedPayload is the payload for sys.config.saved control-plane events.
// Category matches sys_configs.category and lets subscribers invalidate only the
// runtime policy slice they care about.
type SysConfigSavedPayload struct {
	Category      string    `json:"category"`
	BatchID       uuid.UUID `json:"batch_id,omitempty"`
	ConfigVersion int64     `json:"config_version,omitempty"`
}

// ParamSyncTaskResultPayload is a lightweight canonical terminal-result event.
// ResultRef points at durable storage; raw SOAP and large parameter lists must
// never be embedded in this event.
type ParamSyncTaskResultPayload struct {
	EventID      string    `json:"event_id"`
	RequestID    uuid.UUID `json:"request_id"`
	RunID        uuid.UUID `json:"run_id"`
	TaskID       string    `json:"task_id"`
	DeviceID     uuid.UUID `json:"device_id,omitempty"`
	DeviceSN     string    `json:"device_sn"`
	Success      bool      `json:"success"`
	ResultRef    string    `json:"result_ref"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

type ParamSyncRequestedPayload struct {
	DeviceSN       string   `json:"device_sn"`
	TriggerReason  string   `json:"trigger_reason"`
	RequestedPaths []string `json:"requested_paths"`
	IdempotencyKey string   `json:"idempotency_key"`
}

// DeviceLocationObservedPayload is the immutable position snapshot consumed by
// geofence evaluation. All quality fields belong to the same parameter sync
// snapshot as the coordinates.
type DeviceLocationObservedPayload struct {
	DeviceID                   uuid.UUID  `json:"device_id"`
	SerialNumber               string     `json:"serial_number"`
	Carrier                    string     `json:"carrier"`
	ObservationVersion         int64      `json:"observation_version"`
	Latitude                   float64    `json:"latitude"`
	Longitude                  float64    `json:"longitude"`
	GPSHeight                  *float64   `json:"gps_height,omitempty"`
	GPSLockStatus              *string    `json:"gps_lock_status,omitempty"`
	SatelliteCount             *int       `json:"satellite_count,omitempty"`
	AccuracyMeters             *float64   `json:"accuracy_meters,omitempty"`
	ObservedAt                 time.Time  `json:"observed_at"`
	ReceivedAt                 time.Time  `json:"received_at"`
	DeviceReportedAt           *time.Time `json:"device_reported_at,omitempty"`
	SourcePath                 string     `json:"source_path"`
	PreviousObservationVersion *int64     `json:"previous_observation_version,omitempty"`
	MovementDistanceMeters     *float64   `json:"movement_distance_meters,omitempty"`
	ElapsedSeconds             *float64   `json:"elapsed_seconds,omitempty"`
	ImpliedSpeedMPS            *float64   `json:"implied_speed_mps,omitempty"`
}

type GeofenceLifecycleReevaluatePayload struct {
	DeviceID   uuid.UUID `json:"device_id"`
	GeofenceID uuid.UUID `json:"geofence_id"`
	Reason     string    `json:"reason"`
}

// GeofenceLifecycleDeactivationPayload records the safety action required when
// an enabled fence is disabled or archived while a device's last confirmed
// binding state is inside. The outbox event ID is the idempotency boundary.
type GeofenceLifecycleDeactivationPayload struct {
	DeviceID     uuid.UUID `json:"device_id"`
	SerialNumber string    `json:"serial_number"`
	GeofenceID   uuid.UUID `json:"geofence_id"`
	TargetStatus string    `json:"target_status"`
	Reason       string    `json:"reason"`
}

// GeofenceEvaluationPayload is emitted for every completed or deterministically
// failed binding evaluation. The event identity is carried by the Event
// envelope; these fields provide the stable business identity used by
// downstream history and metrics consumers.
type GeofenceEvaluationPayload struct {
	DeviceID             uuid.UUID `json:"device_id"`
	SerialNumber         string    `json:"serial_number"`
	Carrier              string    `json:"carrier"`
	BindingID            uuid.UUID `json:"binding_id"`
	RuleType             string    `json:"rule_type"`
	GeofenceID           uuid.UUID `json:"geofence_id"`
	GeofenceVersionID    uuid.UUID `json:"geofence_version_id"`
	ObservationVersion   int64     `json:"observation_version"`
	StateVersion         int64     `json:"state_version"`
	EvaluationID         uuid.UUID `json:"evaluation_id"`
	ConfirmedState       string    `json:"confirmed_state"`
	EvaluationHealth     string    `json:"evaluation_health"`
	ReasonCode           string    `json:"reason_code"`
	ErrorCode            string    `json:"error_code,omitempty"`
	OccurredAt           time.Time `json:"occurred_at"`
	SignedDistanceMeters *float64  `json:"signed_distance_meters,omitempty"`
}

// GeofenceDeviceStatePayload represents a device-wide effective-state edge.
// Consumers must use EffectiveStateVersion as the action-level idempotency
// boundary rather than a single binding evaluation ID.
type GeofenceDeviceStatePayload struct {
	DeviceID              uuid.UUID  `json:"device_id"`
	SerialNumber          string     `json:"serial_number"`
	Carrier               string     `json:"carrier"`
	TriggerBindingID      *uuid.UUID `json:"binding_id,omitempty"`
	TriggerEvaluationID   *uuid.UUID `json:"evaluation_id,omitempty"`
	ObservationVersion    int64      `json:"observation_version"`
	EffectiveState        string     `json:"effective_state"`
	RequiredActionLevel   string     `json:"required_action_level"`
	EffectiveStateVersion int64      `json:"effective_state_version"`
	EvaluationHealth      string     `json:"evaluation_health"`
	ReasonCode            string     `json:"reason_code"`
	OccurredAt            time.Time  `json:"occurred_at"`
}

// PMAggregationMetric 是已完成 15 分钟计算、可直接进入窗口累加的有限数值。
type PMAggregationMetric struct {
	MetricPath string  `json:"metric_path"`
	MetricType string  `json:"metric_type"`
	StatisType string  `json:"statis_type"`
	Value      float64 `json:"value"`
}

// PMAggregationMeasurement 保留对象和 CounterGroup 语义，不引用时序库行。
type PMAggregationMeasurement struct {
	ObjectLDN    string                `json:"object_ldn"`
	CounterGroup string                `json:"counter_group"`
	Metrics      []PMAggregationMetric `json:"metrics"`
}

// PMAggregationNormalizedPayload 是 PM 入库事务写入 outbox 的版本化标准事件。
type PMAggregationNormalizedPayload struct {
	SchemaVersion int                        `json:"schema_version"`
	EventID       uuid.UUID                  `json:"event_id"`
	SourceFileID  uuid.UUID                  `json:"source_file_id"`
	IngestBatchID uuid.UUID                  `json:"ingest_batch_id"`
	DeviceID      uuid.UUID                  `json:"device_id"`
	DeviceOUI     string                     `json:"device_oui"`
	DeviceSN      string                     `json:"device_sn"`
	Technology    string                     `json:"technology"`
	WindowStart   time.Time                  `json:"window_start"`
	WindowEnd     time.Time                  `json:"window_end"`
	Measurements  []PMAggregationMeasurement `json:"measurements"`
}

// PMAggregationTaskVersionChangedPayload 是轻量控制面刷新通知。
type PMAggregationTaskVersionChangedPayload struct {
	TaskID        uuid.UUID `json:"task_id"`
	TaskVersionID uuid.UUID `json:"task_version_id"`
	EffectiveFrom time.Time `json:"effective_from"`
}

// NewEvent creates a new Event with a generated ID and current timestamp.
func NewEvent(subject string, payload interface{}) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:        uuid.New().String(),
		Subject:   subject,
		Payload:   data,
		Timestamp: time.Now(),
	}, nil
}

// DecodePayload unmarshals the event payload into the target.
func (e *Event) DecodePayload(target interface{}) error {
	return json.Unmarshal(e.Payload, target)
}
