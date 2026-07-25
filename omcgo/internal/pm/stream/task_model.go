package stream

import (
	"time"

	"github.com/google/uuid"
)

type Granularity string

const (
	GranularityHourly  Granularity = "hourly"
	GranularityDaily   Granularity = "daily"
	GranularityWeekly  Granularity = "weekly"
	GranularityMonthly Granularity = "monthly"
)

type Dimension string

const (
	DimensionDevice         Dimension = "device"
	DimensionAggregateGroup Dimension = "aggregate_group"
	DimensionDeviceGroup    Dimension = "device_group"
	DimensionProduct        Dimension = "product"
	DimensionBand           Dimension = "band"
	DimensionNetwork        Dimension = "network"
)

type AggregationOp string

const (
	AggregationSum AggregationOp = "sum"
	AggregationAvg AggregationOp = "avg"
	AggregationMin AggregationOp = "min"
	AggregationMax AggregationOp = "max"
)

type MetricRule struct {
	MetricID    string
	MetricPath  string
	MetricType  string
	Aggregation AggregationOp
}

type TaskMember struct {
	DeviceID      uuid.UUID
	DeviceSN      string
	DimensionKey  string
	DimensionName string
	ObjectLDN     string
}

type SaveTaskRequest struct {
	TaskID        uuid.UUID
	Name          string
	Enabled       bool
	Visibility    string
	Creator       string
	Technology    string
	Dimension     Dimension
	Granularities []Granularity
	ObjectLDNs    []string
	Metrics       []MetricRule
	Members       []TaskMember
	Now           time.Time
}

type TaskVersionSnapshot struct {
	TaskID        uuid.UUID
	VersionID     uuid.UUID
	VersionNo     int
	Name          string
	Enabled       bool
	Technology    string
	Dimension     Dimension
	Granularities []Granularity
	ObjectLDNs    map[string]struct{}
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Metrics       map[string]MetricRule
	Members       map[uuid.UUID][]TaskMember
}
