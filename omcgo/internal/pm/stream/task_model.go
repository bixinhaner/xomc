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
	AggregationSum     AggregationOp = "sum"
	AggregationAvg     AggregationOp = "avg"
	AggregationMin     AggregationOp = "min"
	AggregationMax     AggregationOp = "max"
	AggregationFormula AggregationOp = "formula"
)

type MetricRule struct {
	MetricID     string
	MetricPath   string
	MetricType   string
	Aggregation  AggregationOp
	Formula      string
	Dependencies []string
}

// CounterRule is the composable state retained between aggregation levels.
// Average counters carry sum+count; KPI values never enter this path.
type CounterRule struct {
	MetricPath  string
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
	TaskID          uuid.UUID
	Name            string
	Enabled         bool
	Visibility      string
	Creator         string
	Technology      string
	Dimension       Dimension
	Granularities   []Granularity
	ObjectLDNs      []string
	Metrics         []MetricRule
	Counters        []CounterRule
	Members         []TaskMember
	Now             time.Time
	EffectiveFrom   time.Time
	PlannedEndAt    *time.Time
	SourceUpdatedAt time.Time
}

type TaskVersionSnapshot struct {
	TaskID        uuid.UUID
	VersionID     uuid.UUID
	VersionNo     int
	NewVersion    bool
	Name          string
	TaskEnabled   bool
	TaskDeletedAt *time.Time
	Enabled       bool
	Technology    string
	Dimension     Dimension
	Granularities []Granularity
	ObjectLDNs    map[string]struct{}
	EffectiveFrom time.Time
	// LineageEffectiveFrom is the first persisted effective_from for the task.
	// It is loaded without the matchable-history cutoff so synthetic stable
	// rollups never acquire a moving epoch as old catalog versions age out.
	LineageEffectiveFrom time.Time
	EffectiveTo          *time.Time
	PlannedEndAt         *time.Time
	Metrics              map[string]MetricRule
	Counters             map[string]CounterRule
	Members              map[uuid.UUID][]TaskMember
	// DimensionMemberCounts caches distinct device counts per dimension key.
	// It prevents all-member scans for every device-hour contribution.
	DimensionMemberCounts map[string]int64
	// DevicePipeline marks a synthetic catalog-backed version used by the
	// fixed raw-to-hour device pipeline. Each definition has an immutable ID.
	DevicePipeline bool
	// DeviceRollup marks the stable hour-to-day-to-week/month lineage. It is
	// indexed only by version ID and never matches raw PM events directly.
	DeviceRollup    bool
	RollupVersionID uuid.UUID
}

type RecoveryVersionState struct {
	TaskID         uuid.UUID
	TaskEnabled    bool
	TaskDeletedAt  *time.Time
	VersionID      uuid.UUID
	VersionEnabled bool
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
	PlannedEndAt   *time.Time
}
