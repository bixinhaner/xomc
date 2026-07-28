package stream

import (
	"time"

	"github.com/google/uuid"
)

// AggregationRule is a declarative filter for analytics dimensions. It has no
// scheduler state: the shared rollup pipeline evaluates effective versions when
// device-hour counter events arrive.
type AggregationRule struct {
	ID            uuid.UUID
	Name          string
	Enabled       bool
	Visibility    string
	Creator       string
	Technology    string
	Dimension     Dimension
	Granularities []Granularity
	ObjectLDNs    []string
	Metrics       []MetricRule
	Counters      []CounterRule
	Members       []TaskMember
}

// RuleVersionSnapshot is an immutable in-memory value. Updating a logical rule
// creates another snapshot with a new VersionID and effective interval.
type RuleVersionSnapshot struct {
	Rule          AggregationRule
	VersionID     uuid.UUID
	VersionNo     int
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}
