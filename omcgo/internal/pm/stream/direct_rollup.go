package stream

import (
	"sort"
)

const defaultRollupBatchValues = 500

func rollupTargets(source Granularity) []Granularity {
	switch source {
	case GranularityHourly:
		return []Granularity{GranularityDaily}
	case GranularityDaily:
		return []Granularity{GranularityWeekly, GranularityMonthly}
	default:
		return nil
	}
}

func compactCounterValues(
	version *TaskVersionSnapshot,
	accumulators []Accumulator,
) []ContributionValue {
	values := make([]ContributionValue, 0, len(accumulators))
	for _, accumulator := range accumulators {
		rule, ok := version.Counters[accumulator.Definition.MetricPath]
		if !ok || accumulator.Count <= 0 {
			continue
		}
		value := accumulator.Definition
		value.MetricType = "counter"
		value.Operation = rule.Aggregation
		value.Value = 0
		value.Sum = accumulator.Sum
		value.Count = accumulator.Count
		value.Min = accumulator.Min
		value.Max = accumulator.Max
		value.Composed = true
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		leftKey := aggregationGroupKey(values[i]) + "\x1f" + values[i].MetricPath
		rightKey := aggregationGroupKey(values[j]) + "\x1f" + values[j].MetricPath
		return leftKey < rightKey
	})
	return values
}

func containsGranularity(values []Granularity, target Granularity) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
