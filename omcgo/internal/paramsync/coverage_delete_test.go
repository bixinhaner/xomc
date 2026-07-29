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
	// {i} 分支要同时带一个可走 (device_id, parameter_path varchar_pattern_ops)
	// 索引的 LIKE 前缀条件，避免整表逐行做正则匹配（线上巡检发现的性能问题）。
	assert.Contains(t, sql, "parameter_path LIKE ?")
	assert.Contains(t, sql, "parameter_path ~ ?")
	assert.Equal(t, []any{"Device.Radio.%", `^Device\.Radio\.[0-9]+\.Enable$`}, args)
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

func TestRecoveredMLNDCCarrier9005MarksRFMappingIncomplete(t *testing.T) {
	coverage := []CoverageScope{
		{Path: mlnDCRFStatusStandardPath, Complete: true, Mappings: []FrozenMapping{{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			IsStorable:   true,
		}}},
		{Path: "Device.Info.Serial", Complete: true, Mappings: []FrozenMapping{{
			StandardPath: "Device.Info.Serial",
			PrivatePath:  "Device.Info.Serial",
			IsStorable:   true,
		}}},
	}
	const missingCarrierPath = "Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.AdminCellState"

	indexes := recoveredIncompleteCoverageIndexes(coverage, storedTaskResult{
		Recovered:      true,
		FaultCode:      9005,
		BadPath:        missingCarrierPath,
		RequestedNames: []string{missingCarrierPath},
	})

	require.Equal(t, []int{0}, indexes)
}

func TestRecoveredMLNDCRFUnknownUsesIsolatedRequestWhenFaultPathDiffers(t *testing.T) {
	const (
		privatePath  = "Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.AdminCellState"
		standardPath = "Device.Services.FAPService.3.FAPControl.LTE.RFTxStatus"
	)
	coverage := []CoverageScope{{
		Path: mlnDCRFStatusStandardPath,
		Mappings: []FrozenMapping{{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			Access:       "readWrite",
			IsStorable:   true,
		}},
	}}

	value, ok := projectRecoveredMLNDCRFUnknown(storedTaskResult{
		Recovered:      true,
		FaultCode:      9005,
		BadPath:        "Device.Vendor.UnparseableFaultPath",
		BadPaths:       []string{privatePath},
		RequestedNames: []string{privatePath},
	}, coverage)

	require.True(t, ok)
	assert.Equal(t, standardPath, value.ParameterPath)
	assert.Equal(t, privatePath, value.PrivatePath)
	assert.Equal(t, "unknown", value.Value)
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
