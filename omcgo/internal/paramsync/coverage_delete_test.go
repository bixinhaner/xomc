package paramsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrozenCoveragePathPredicateUsesExactMappedLeaf(t *testing.T) {
	sql, args, err := frozenCoveragePathPredicate(CoverageScope{Mappings: []FrozenMapping{{StandardPath: "Device.Info.Serial", IsStorable: true}}}).ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "parameter_path = ?")
	assert.Equal(t, []any{"Device.Info.Serial"}, args)
}

func TestFrozenCoveragePathPredicateOnlyMatchesMappedRuntimeInstances(t *testing.T) {
	sql, args, err := frozenCoveragePathPredicate(CoverageScope{Mappings: []FrozenMapping{
		{StandardPath: "Device.Radio.{i}.Enable", IsStorable: true},
		{StandardPath: "Device.Radio.{i}.Secret", IsStorable: false},
	}}).ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "parameter_path ~ ?")
	assert.Equal(t, []any{`^Device\.Radio\.[0-9]+\.Enable$`}, args)
}

func TestRecoveredPrivateLeafMarksOnlyItsFrozenStandardCoverageIncomplete(t *testing.T) {
	coverage := []CoverageScope{
		{Path: "Device.Radio.{i}.Enable", Complete: true, Mappings: []FrozenMapping{{
			StandardPath: "Device.Radio.{i}.Enable", PrivatePath: "Vendor.Cell.{i}.Enabled", IsStorable: true,
		}}},
		{Path: "Device.Info.Serial", Complete: true, Mappings: []FrozenMapping{{
			StandardPath: "Device.Info.Serial", PrivatePath: "Vendor.Info.Serial", IsStorable: true,
		}}},
	}

	indexes := recoveredIncompleteCoverageIndexes(coverage, storedTaskResult{
		Recovered: true, FaultCode: 9005, BadPath: "Vendor.Cell.7.Enabled",
		RequestedNames: []string{"Vendor.Cell.7.Enabled"},
	})

	require.Equal(t, []int{0}, indexes)
}

func TestRecoveredPrivateObjectPrefixMarksItsFrozenSubtreeIncomplete(t *testing.T) {
	coverage := []CoverageScope{{
		Path: "Device.Radio.", Complete: true, Subtree: true,
		Mappings: []FrozenMapping{{
			StandardPath: "Device.Radio.{i}.Enable", PrivatePath: "Vendor.Cell.{i}.Enabled", IsStorable: true,
		}},
	}}

	indexes := recoveredIncompleteCoverageIndexes(coverage, storedTaskResult{
		Recovered: true, FaultCode: 9005, BadPath: "Vendor.Cell.7.",
		RequestedNames: []string{"Vendor.Cell.7."},
	})

	require.Equal(t, []int{0}, indexes)
}

func TestRecoveredUnknownFaultFallsBackToThisTaskRequestedCoverage(t *testing.T) {
	coverage := []CoverageScope{
		{Path: "Device.Radio.", Complete: true, Subtree: true, Mappings: []FrozenMapping{{
			StandardPath: "Device.Radio.{i}.Enable", PrivatePath: "Vendor.Cell.{i}.Enabled", IsStorable: true,
		}}},
		{Path: "Device.Info.Serial", Complete: true, Mappings: []FrozenMapping{{
			StandardPath: "Device.Info.Serial", PrivatePath: "Vendor.Info.Serial", IsStorable: true,
		}}},
	}

	indexes := recoveredIncompleteCoverageIndexes(coverage, storedTaskResult{
		Recovered: true, FaultCode: 9005, BadPath: "Vendor.Unparseable.FaultText",
		RequestedNames: []string{"Vendor.Info.Serial"},
	})

	require.Equal(t, []int{1}, indexes)
}

func TestRecoveredUnresolvableTaskConservativelyMarksAllCoverageIncomplete(t *testing.T) {
	coverage := []CoverageScope{
		{Path: "Device.Radio.", Complete: true},
		{Path: "Device.Info.Serial", Complete: true},
	}

	indexes := recoveredIncompleteCoverageIndexes(coverage, storedTaskResult{
		Recovered: true, FaultCode: 9005, BadPath: "Vendor.Unknown.Path",
		RequestedNames: []string{"Vendor.Unknown.Request"},
	})

	require.Equal(t, []int{0, 1}, indexes)
}
