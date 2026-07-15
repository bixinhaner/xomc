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
	Category string `json:"category"`
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
