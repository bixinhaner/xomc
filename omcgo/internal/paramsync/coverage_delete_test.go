package paramsync

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFullSyncReconcileDeleteCombinesCompleteCoverageIntoOneStatement(t *testing.T) {
	deviceID := uuid.New()
	runID := uuid.New()
	coverage := []CoverageScope{
		{Path: "Device.Info.Serial", Complete: true, Mappings: []FrozenMapping{{StandardPath: "Device.Info.Serial", IsStorable: true}}},
		{Path: "Device.Radio.", Complete: true, Mappings: []FrozenMapping{{StandardPath: "Device.Radio.{i}.Enable", IsStorable: true}}},
		{Path: "Device.Incomplete", Complete: false, Mappings: []FrozenMapping{{StandardPath: "Device.Incomplete", IsStorable: true}}},
	}

	sql, args, ok, err := buildFullSyncReconcileDelete(deviceID, runID, coverage)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, strings.Count(sql, "DELETE FROM device_parameters"))
	require.Contains(t, sql, "parameter_path = $2 OR")
	require.Contains(t, sql, "parameter_path LIKE $3")
	require.Contains(t, sql, "parameter_path ~ $4")
	require.NotContains(t, args, "Device.Incomplete")
	require.Contains(t, args, deviceID.String())
	require.Contains(t, args, runID)
}

func TestFrozenCoveragePathPredicateUsesExactMappedLeaf(t *testing.T) {
	sql, args, err := frozenCoveragePathPredicate(CoverageScope{Mappings: []FrozenMapping{{StandardPath: "Device.Info.Serial", IsStorable: true}}}).ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "parameter_path = ?")
	assert.Equal(t, []any{"Device.Info.Serial"}, args)
}

func TestFrozenCoveragePathPredicateCollapsesExactLeavesIntoOneIndexedSet(t *testing.T) {
	sql, args, err := frozenCoveragePathPredicate(CoverageScope{Mappings: []FrozenMapping{
		{StandardPath: "Device.Info.Serial", IsStorable: true},
		{StandardPath: "Device.Info.Model", IsStorable: true},
		{StandardPath: "Device.Info.Software", IsStorable: true},
	}}).ToSql()
	require.NoError(t, err)
	require.Contains(t, sql, "parameter_path IN (?,?,?)")
	require.NotContains(t, sql, " OR ")
	require.Equal(t, []any{
		"Device.Info.Serial", "Device.Info.Model", "Device.Info.Software",
	}, args)
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

func TestProjectRecoveredMLNDCRFUnknownRequiresExactIsolatedLeaf(t *testing.T) {
	coverage := []CoverageScope{{
		Mappings: []FrozenMapping{{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			IsStorable:   true,
		}},
	}}
	privatePath := "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState"

	value, ok := projectRecoveredMLNDCRFUnknown(storedTaskResult{
		Recovered:      true,
		FaultCode:      9005,
		BadPath:        privatePath,
		RequestedNames: []string{privatePath},
	}, coverage)

	require.True(t, ok)
	assert.Equal(t, "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus", value.ParameterPath)
	assert.Equal(t, privatePath, value.PrivatePath)
	assert.Equal(t, "unknown", value.Value)
}

func TestProjectRecoveredMLNDCRFUnknownRejectsAmbiguousRecovery(t *testing.T) {
	coverage := []CoverageScope{{
		Mappings: []FrozenMapping{{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			IsStorable:   true,
		}},
	}}
	privatePath := "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState"

	tests := []storedTaskResult{
		{
			Recovered:      true,
			FaultCode:      9005,
			BadPath:        privatePath,
			RequestedNames: []string{privatePath, "Device.Other"},
		},
		{
			Recovered:      true,
			FaultCode:      9005,
			BadPath:        "Device.Other",
			RequestedNames: []string{privatePath},
		},
		{
			Recovered:      true,
			FaultCode:      9007,
			BadPath:        privatePath,
			RequestedNames: []string{privatePath},
		},
	}
	for _, stored := range tests {
		_, ok := projectRecoveredMLNDCRFUnknown(stored, coverage)
		assert.False(t, ok)
	}
}
