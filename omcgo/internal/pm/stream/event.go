package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
)

const SchemaVersion = 1

var ErrInvalidEvent = errors.New("invalid PM aggregation event")

func ValidateEvent(payload event.PMAggregationNormalizedPayload) error {
	if payload.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: unsupported schema version %d", ErrInvalidEvent, payload.SchemaVersion)
	}
	if payload.EventID == uuid.Nil || payload.SourceFileID == uuid.Nil ||
		payload.IngestBatchID == uuid.Nil || payload.DeviceID == uuid.Nil {
		return fmt.Errorf("%w: required UUID is empty", ErrInvalidEvent)
	}
	if payload.DeviceSN == "" || payload.WindowStart.IsZero() || !payload.WindowEnd.After(payload.WindowStart) {
		return fmt.Errorf("%w: invalid device or window", ErrInvalidEvent)
	}
	start := payload.WindowStart.UTC()
	if start.Second() != 0 || start.Nanosecond() != 0 || start.Minute()%15 != 0 ||
		payload.WindowEnd.Sub(payload.WindowStart) != 15*time.Minute {
		return fmt.Errorf("%w: window must be one aligned 15-minute slot", ErrInvalidEvent)
	}
	for _, measurement := range payload.Measurements {
		for _, metric := range measurement.Metrics {
			if metric.MetricPath == "" {
				return fmt.Errorf("%w: metric path is empty", ErrInvalidEvent)
			}
			if metric.MetricType != "counter" && metric.MetricType != "kpi" {
				return fmt.Errorf("%w: unsupported metric type %q", ErrInvalidEvent, metric.MetricType)
			}
			if math.IsNaN(metric.Value) || math.IsInf(metric.Value, 0) {
				return fmt.Errorf("%w: metric %s is not finite", ErrInvalidEvent, metric.MetricPath)
			}
		}
	}
	return nil
}

func MarshalEvent(payload event.PMAggregationNormalizedPayload, maxBytes int) ([]byte, error) {
	if err := ValidateEvent(payload); err != nil {
		return nil, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal PM aggregation event: %w", err)
	}
	if maxBytes > 0 && len(data) > maxBytes {
		return nil, fmt.Errorf("%w: payload bytes %d exceed limit %d", ErrInvalidEvent, len(data), maxBytes)
	}
	return data, nil
}
