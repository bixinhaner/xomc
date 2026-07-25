package stream

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/require"
)

func validNormalizedEvent() event.PMAggregationNormalizedPayload {
	start := time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC)
	return event.PMAggregationNormalizedPayload{
		SchemaVersion: SchemaVersion,
		EventID:       uuid.New(),
		SourceFileID:  uuid.New(),
		IngestBatchID: uuid.New(),
		DeviceID:      uuid.New(),
		DeviceSN:      "SN-1",
		Technology:    "lte",
		WindowStart:   start,
		WindowEnd:     start.Add(slotDuration),
		Measurements: []event.PMAggregationMeasurement{{
			ObjectLDN: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.1",
			Metrics: []event.PMAggregationMetric{{
				MetricPath: "K001",
				MetricType: "kpi",
				Value:      12.5,
			}},
		}},
	}
}

func TestValidateEventAndPayloadLimit(t *testing.T) {
	payload := validNormalizedEvent()
	data, err := MarshalEvent(payload, 1<<20)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	_, err = MarshalEvent(payload, 8)
	require.ErrorIs(t, err, ErrInvalidEvent)
}

func TestValidateEventRejectsNonFiniteMetric(t *testing.T) {
	payload := validNormalizedEvent()
	payload.Measurements[0].Metrics[0].Value = math.NaN()
	err := ValidateEvent(payload)
	require.True(t, errors.Is(err, ErrInvalidEvent))
}
