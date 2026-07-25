package streamtest

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
)

type Generator struct {
	Devices    int
	Metrics    int
	SlotStart  time.Time
	Technology string
}

func (g Generator) Event(deviceIndex int) (event.PMAggregationNormalizedPayload, error) {
	if g.Devices <= 0 || deviceIndex < 0 || deviceIndex >= g.Devices || g.Metrics <= 0 {
		return event.PMAggregationNormalizedPayload{}, fmt.Errorf("invalid PM stream load generator bounds")
	}
	technology := g.Technology
	if technology == "" {
		technology = "lte"
	}
	start := g.SlotStart.UTC().Truncate(15 * time.Minute)
	if start.IsZero() {
		start = time.Now().UTC().Truncate(15 * time.Minute)
	}
	deviceSN := fmt.Sprintf("LOAD-%08d", deviceIndex+1)
	deviceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(deviceSN))
	sourceID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(deviceSN+"|"+start.Format(time.RFC3339)))
	metrics := make([]event.PMAggregationMetric, 0, g.Metrics)
	for index := 0; index < g.Metrics; index++ {
		metrics = append(metrics, event.PMAggregationMetric{
			MetricPath: fmt.Sprintf("C%04d", index+1),
			MetricType: "counter",
			StatisType: "sum",
			Value:      float64(deviceIndex+index) + 0.5,
		})
	}
	return event.PMAggregationNormalizedPayload{
		SchemaVersion: pmstream.SchemaVersion,
		EventID:       uuid.New(),
		SourceFileID:  sourceID,
		IngestBatchID: uuid.New(),
		DeviceID:      deviceID,
		DeviceOUI:     "LOAD",
		DeviceSN:      deviceSN,
		Technology:    technology,
		WindowStart:   start,
		WindowEnd:     start.Add(15 * time.Minute),
		Measurements: []event.PMAggregationMeasurement{{
			ObjectLDN:    fmt.Sprintf("Device.Services.FAPService.1.CellConfig.%s.RAN.RF.1", technology),
			CounterGroup: "loadtest",
			Metrics:      metrics,
		}},
	}, nil
}
