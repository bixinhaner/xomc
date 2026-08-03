package stream

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
)

type ContributionValue struct {
	Dimension     Dimension     `json:"d"`
	DimensionKey  string        `json:"dk"`
	DimensionName string        `json:"dn,omitempty"`
	ObjectLDN     string        `json:"o,omitempty"`
	DeviceOUI     string        `json:"oui,omitempty"`
	DeviceSN      string        `json:"sn,omitempty"`
	Technology    string        `json:"t,omitempty"`
	MetricPath    string        `json:"m"`
	MetricType    string        `json:"mt"`
	Operation     AggregationOp `json:"op"`
	Value         float64       `json:"v,omitempty"`
	Sum           float64       `json:"s,omitempty"`
	Count         int64         `json:"c,omitempty"`
	Min           float64       `json:"n,omitempty"`
	Max           float64       `json:"x,omitempty"`
	Composed      bool          `json:"cp,omitempty"`
}

type Contribution struct {
	Key                   WindowKey
	SourceFileID          string
	DeviceID              string
	SlotStart             time.Time
	ExpectedSlots         int64
	SourceExpectedSlots   int64
	SourceReceivedSlots   int64
	SourceIncompleteSlots int64
	VersionEffectiveFrom  time.Time
	VersionEffectiveTo    *time.Time
	Rollup                bool
	RollupChunkIndex      int
	RollupChunkCount      int
	Values                []ContributionValue
}

type Matcher struct {
	location LocationProvider
}

type LocationProvider func() *time.Location

func fixedLocationProvider(location *time.Location) LocationProvider {
	if location == nil {
		location = time.UTC
	}
	return func() *time.Location {
		return location
	}
}

func currentLocation(provider LocationProvider) *time.Location {
	if provider == nil {
		return time.UTC
	}
	location := provider()
	if location == nil {
		return time.UTC
	}
	return location
}

func NewMatcher(location *time.Location) *Matcher {
	return NewMatcherWithLocationProvider(fixedLocationProvider(location))
}

func NewMatcherWithLocationProvider(location LocationProvider) *Matcher {
	return &Matcher{location: location}
}

func (m *Matcher) Location() *time.Location {
	if m == nil {
		return time.UTC
	}
	return currentLocation(m.location)
}

func (m *Matcher) Match(
	payload event.PMAggregationNormalizedPayload,
	snapshot *TaskSnapshot,
) ([]Contribution, error) {
	return m.MatchGranularity(payload, snapshot, GranularityHourly)
}

// MatchGranularity rebuilds a Redis window directly from retained original
// 15-minute events. Normal ingestion only calls Match (hourly).
func (m *Matcher) MatchGranularity(
	payload event.PMAggregationNormalizedPayload,
	snapshot *TaskSnapshot,
	target Granularity,
) ([]Contribution, error) {
	if snapshot == nil {
		return nil, nil
	}
	versions := snapshot.ByDevice[payload.DeviceID]
	out := make([]Contribution, 0)
	for _, version := range versions {
		if !version.DevicePipeline || !version.Enabled || (version.Technology != "" &&
			!strings.EqualFold(version.Technology, payload.Technology)) {
			continue
		}
		members := version.Members[payload.DeviceID]
		if len(members) == 0 {
			continue
		}
		for _, granularity := range version.Granularities {
			// Raw 15-minute PM events only feed hourly windows. Coarser
			// windows consume finalized child counter events.
			if granularity != target {
				continue
			}
			window, err := WindowFor(payload.WindowStart, target, m.Location())
			if err != nil {
				return nil, err
			}
			if window.Start.Before(version.EffectiveFrom) ||
				(version.EffectiveTo != nil && !window.Start.Before(*version.EffectiveTo)) ||
				(version.PlannedEndAt != nil && !window.Start.Before(*version.PlannedEndAt)) {
				continue
			}
			contribution := Contribution{
				Key: WindowKey{
					TaskID: version.TaskID, TaskVersionID: version.VersionID,
					EntityKey:   payload.DeviceID.String(),
					Granularity: granularity, Start: window.Start, End: window.End,
				},
				SourceFileID:         payload.SourceFileID.String(),
				DeviceID:             payload.DeviceID.String(),
				SlotStart:            payload.WindowStart.UTC(),
				ExpectedSlots:        4,
				VersionEffectiveFrom: version.EffectiveFrom,
				VersionEffectiveTo:   version.EffectiveTo,
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
						if metric.MetricType != "counter" {
							continue
						}
						counterRule, ok := version.Counters[metric.MetricPath]
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
							MetricType: "counter", Operation: counterRule.Aggregation, Value: metric.Value,
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

func expectedChildWindows(window Window, target Granularity, location *time.Location) int64 {
	switch target {
	case GranularityDaily:
		return 24
	case GranularityWeekly:
		return 7
	case GranularityMonthly:
		localStart := window.Start.In(location)
		localEnd := window.End.In(location)
		days := int64(0)
		for cursor := localStart; cursor.Before(localEnd); cursor = cursor.AddDate(0, 0, 1) {
			days++
		}
		return days
	default:
		return 0
	}
}

func (c Contribution) Validate() error {
	if c.Key.TaskVersionID == uuid.Nil || c.Key.EntityKey == "" || c.SourceFileID == "" ||
		c.DeviceID == "" || c.SlotStart.IsZero() || !c.Key.End.After(c.Key.Start) {
		return fmt.Errorf("invalid PM aggregation contribution")
	}
	return nil
}
