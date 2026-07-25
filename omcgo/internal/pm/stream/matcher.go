package stream

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
)

type ContributionValue struct {
	Dimension     Dimension
	DimensionKey  string
	DimensionName string
	ObjectLDN     string
	DeviceOUI     string
	DeviceSN      string
	Technology    string
	MetricPath    string
	MetricType    string
	Operation     AggregationOp
	Value         float64
}

type Contribution struct {
	Key           WindowKey
	SourceFileID  string
	DeviceID      string
	SlotStart     time.Time
	ExpectedSlots int64
	Values        []ContributionValue
}

type Matcher struct {
	location *time.Location
}

func NewMatcher(location *time.Location) *Matcher {
	if location == nil {
		location = time.UTC
	}
	return &Matcher{location: location}
}

func (m *Matcher) Match(
	payload event.PMAggregationNormalizedPayload,
	snapshot *TaskSnapshot,
) ([]Contribution, error) {
	if snapshot == nil {
		return nil, nil
	}
	versions := snapshot.ByDevice[payload.DeviceID]
	out := make([]Contribution, 0)
	for _, version := range versions {
		if !version.Enabled || (version.Technology != "" &&
			!strings.EqualFold(version.Technology, payload.Technology)) {
			continue
		}
		members := version.Members[payload.DeviceID]
		if len(members) == 0 {
			continue
		}
		for _, granularity := range version.Granularities {
			window, err := WindowFor(payload.WindowStart, granularity, m.location)
			if err != nil {
				return nil, err
			}
			if window.Start.Before(version.EffectiveFrom) ||
				(version.EffectiveTo != nil && !window.Start.Before(*version.EffectiveTo)) {
				continue
			}
			contribution := Contribution{
				Key: WindowKey{
					TaskID: version.TaskID, TaskVersionID: version.VersionID,
					Granularity: granularity, Start: window.Start, End: window.End,
				},
				SourceFileID:  payload.SourceFileID.String(),
				DeviceID:      payload.DeviceID.String(),
				SlotStart:     payload.WindowStart.UTC(),
				ExpectedSlots: expectedSlots(window, len(version.Members)),
			}
			for _, measurement := range payload.Measurements {
				if len(version.ObjectLDNs) > 0 {
					if _, ok := version.ObjectLDNs[measurement.ObjectLDN]; !ok {
						continue
					}
				}
				for _, member := range members {
					if member.ObjectLDN != "" &&
						member.ObjectLDN != measurement.ObjectLDN &&
						!strings.Contains(measurement.ObjectLDN, "Cellid="+member.ObjectLDN) {
						continue
					}
					for _, metric := range measurement.Metrics {
						rule, ok := version.Metrics[metric.MetricPath]
						if !ok {
							continue
						}
						deviceOUI, deviceSN := "", ""
						if version.Dimension == DimensionDevice {
							deviceOUI, deviceSN = payload.DeviceOUI, payload.DeviceSN
						}
						contribution.Values = append(contribution.Values, ContributionValue{
							Dimension: version.Dimension, DimensionKey: member.DimensionKey,
							DimensionName: member.DimensionName,
							ObjectLDN:     measurement.ObjectLDN,
							DeviceOUI:     deviceOUI, DeviceSN: deviceSN, Technology: payload.Technology,
							MetricPath: metric.MetricPath,
							MetricType: metric.MetricType, Operation: rule.Aggregation, Value: metric.Value,
						})
					}
				}
			}
			if len(contribution.Values) > 0 {
				out = append(out, contribution)
			}
		}
	}
	return out, nil
}

func (c Contribution) Validate() error {
	if c.Key.TaskVersionID == uuid.Nil || c.SourceFileID == "" ||
		c.DeviceID == "" || c.SlotStart.IsZero() || !c.Key.End.After(c.Key.Start) {
		return fmt.Errorf("invalid PM aggregation contribution")
	}
	return nil
}
