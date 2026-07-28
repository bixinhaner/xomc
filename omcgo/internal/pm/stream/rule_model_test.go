package stream

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregationRuleContainsDefinitionOnly(t *testing.T) {
	typ := reflect.TypeOf(AggregationRule{})
	for _, forbidden := range []string{"Mode", "CronExpr", "Status", "NextRunAt", "LastRunAt"} {
		_, found := typ.FieldByName(forbidden)
		require.False(t, found, "aggregation rule must not contain scheduler field %s", forbidden)
	}
}

func TestRuleVersionSnapshotIsImmutableValue(t *testing.T) {
	require.NotPanics(t, func() {
		_ = RuleVersionSnapshot{
			Rule: AggregationRule{Name: "设备组 LTE", Enabled: true},
		}
	})
}
