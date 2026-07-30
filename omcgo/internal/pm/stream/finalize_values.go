package stream

import (
	"errors"
	"fmt"

	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
)

type finalizedMetric struct {
	Definition  ContributionValue
	MetricID    string
	MetricType  string
	Operation   AggregationOp
	Value       float64
	SampleCount int64
	// FormulaComplete is independent from window data completeness.
	FormulaComplete bool
}

type counterGroup struct {
	base     ContributionValue
	counters map[string]float64
	samples  map[string]int64
}

func buildFinalizedMetrics(
	version *TaskVersionSnapshot,
	state WindowState,
) ([]finalizedMetric, bool, error) {
	if version == nil {
		return nil, true, fmt.Errorf("PM aggregation task version snapshot missing")
	}
	groups := make(map[string]*counterGroup)
	var out []finalizedMetric
	for _, accumulator := range state.Accumulators {
		if accumulator.Count <= 0 {
			continue
		}
		definition := accumulator.Definition
		key := aggregationGroupKey(definition)
		group := groups[key]
		if group == nil {
			base := definition
			base.MetricPath = ""
			base.MetricType = ""
			base.Value = 0
			group = &counterGroup{
				base: base, counters: make(map[string]float64), samples: make(map[string]int64),
			}
			groups[key] = group
		}
		value := accumulatorValue(accumulator)
		group.counters[definition.MetricPath] = value
		group.samples[definition.MetricPath] = accumulator.Count
		if output, ok := version.Metrics[definition.MetricPath]; ok && output.MetricType == "counter" {
			out = append(out, finalizedMetric{
				Definition: definition, MetricID: output.MetricID,
				MetricType: "counter", Operation: definition.Operation,
				Value: value, SampleCount: accumulator.Count, FormulaComplete: true,
			})
		}
	}

	formulaIncomplete := false
	type compiledKPI struct {
		rule    MetricRule
		formula *expr.Formula
	}
	compiled := make([]compiledKPI, 0)
	for _, output := range version.Metrics {
		if output.MetricType != "kpi" {
			continue
		}
		formula, err := expr.Parse(output.Formula)
		if err != nil {
			return nil, true, fmt.Errorf("parse KPI %s formula: %w", output.MetricPath, err)
		}
		compiled = append(compiled, compiledKPI{rule: output, formula: formula})
	}
	for _, group := range groups {
		for _, item := range compiled {
			output := item.rule
			value, err := item.formula.Evaluate(group.counters)
			if err != nil {
				var missing *expr.MissingCounterError
				if errors.As(err, &missing) || errors.Is(err, expr.ErrDivByZero) {
					formulaIncomplete = true
					continue
				}
				return nil, true, fmt.Errorf("evaluate KPI %s formula: %w", output.MetricPath, err)
			}
			definition := group.base
			definition.MetricPath = output.MetricPath
			definition.MetricType = "kpi"
			definition.Operation = AggregationFormula
			sampleCount := minimumDependencySamples(output.Dependencies, group.samples)
			out = append(out, finalizedMetric{
				Definition: definition, MetricID: output.MetricID,
				MetricType: "kpi", Operation: AggregationFormula,
				Value: value, SampleCount: sampleCount, FormulaComplete: true,
			})
		}
	}
	return out, formulaIncomplete, nil
}

func aggregationGroupKey(value ContributionValue) string {
	return fmt.Sprintf(
		"%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s",
		value.Dimension, value.DimensionKey, value.DimensionName,
		value.ObjectLDN, value.DeviceOUI, value.DeviceSN, value.Technology,
	)
}

func minimumDependencySamples(dependencies []string, samples map[string]int64) int64 {
	var minimum int64
	for _, dependency := range dependencies {
		count := samples[dependency]
		if minimum == 0 || count < minimum {
			minimum = count
		}
	}
	return minimum
}
