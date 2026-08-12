package paramsync

import (
	"fmt"
	"strings"
	"testing"
	"time"

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
	require.Contains(t, sql, "parameter_path = ANY($2::text[])")
	require.Contains(t, sql, "parameter_path ~ $3")
	require.NotContains(t, args, "Device.Incomplete")
	require.Contains(t, args, deviceID.String())
	require.Contains(t, args, runID)
}

func TestFrozenCoveragePathPredicateUsesExactMappedLeaf(t *testing.T) {
	sql, args, err := frozenCoveragePathPredicate(CoverageScope{Mappings: []FrozenMapping{{StandardPath: "Device.Info.Serial", IsStorable: true}}}).ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "parameter_path = ANY(?::text[])")
	assert.Equal(t, []any{[]string{"Device.Info.Serial"}}, args)
}

func TestFrozenCoveragePathPredicateCollapsesExactLeavesIntoOneIndexedSet(t *testing.T) {
	sql, args, err := frozenCoveragePathPredicate(CoverageScope{Mappings: []FrozenMapping{
		{StandardPath: "Device.Info.Serial", IsStorable: true},
		{StandardPath: "Device.Info.Model", IsStorable: true},
		{StandardPath: "Device.Info.Software", IsStorable: true},
	}}).ToSql()
	require.NoError(t, err)
	require.Contains(t, sql, "parameter_path = ANY(?::text[])")
	require.NotContains(t, sql, " OR ")
	require.Equal(t, []any{[]string{
		"Device.Info.Serial", "Device.Info.Model", "Device.Info.Software",
	}}, args)
}

func TestFrozenCoveragePathPredicateOnlyMatchesMappedRuntimeInstances(t *testing.T) {
	sql, args, err := frozenCoveragePathPredicate(CoverageScope{Mappings: []FrozenMapping{
		{StandardPath: "Device.Radio.{i}.Enable", IsStorable: true},
		{StandardPath: "Device.Radio.{i}.Secret", IsStorable: false},
	}}).ToSql()
	require.NoError(t, err)
	// device_id 先把候选集收敛到单设备；所有实例模式合成一个正则参数，
	// 避免数百个 LIKE/regex OR 的规划开销与不稳定锁行顺序。
	assert.Contains(t, sql, "parameter_path ~ ?")
	assert.Equal(t, []any{`^(Device\.Radio\.[0-9]+\.Enable)$`}, args)
}

func TestFrozenCoveragePathPredicateCollapsesRuntimePatternsIntoOneRegex(t *testing.T) {
	sql, args, err := frozenCoveragePathPredicate(CoverageScope{Mappings: []FrozenMapping{
		{StandardPath: "Device.Radio.{i}.Enable", IsStorable: true},
		{StandardPath: "Device.Radio.{i}.Status", IsStorable: true},
	}}).ToSql()
	require.NoError(t, err)
	require.Contains(t, sql, "parameter_path ~ ?")
	require.Len(t, args, 1)
	require.Equal(t,
		`^(Device\.Radio\.[0-9]+\.Enable|Device\.Radio\.[0-9]+\.Status)$`,
		args[0],
	)
}

func TestBuildFullSyncReconcileDeleteKeepsSQLAndBindCountBounded(t *testing.T) {
	mappings := make([]FrozenMapping, 0, 700)
	for i := 0; i < 400; i++ {
		mappings = append(mappings, FrozenMapping{
			StandardPath: fmt.Sprintf("Device.Exact.P%d", i), IsStorable: true,
		})
	}
	for i := 0; i < 300; i++ {
		mappings = append(mappings, FrozenMapping{
			StandardPath: fmt.Sprintf("Device.Table.{i}.P%d", i), IsStorable: true,
		})
	}

	sql, args, ok, err := buildFullSyncReconcileDelete(uuid.New(), uuid.New(), []CoverageScope{{
		Path: "Device.", Complete: true, Mappings: mappings,
	}})
	require.NoError(t, err)
	require.True(t, ok)
	require.Less(t, len(sql), 500, "mapping cardinality must not expand SQL text")
	require.Len(t, args, 4, "device, exact array, runtime regex, and run are the only binds")
	require.Len(t, args[1], 400)
}

func TestBuildPartialSyncReconcileDeleteScopesEmptyObjectResponse(t *testing.T) {
	deviceID := uuid.New()
	runID := uuid.New()
	runStartedAt := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	neighborPath := "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."
	sql, args, ok, err := buildPartialSyncReconcileDelete(deviceID, runID, runStartedAt, []CoverageScope{{
		Path: neighborPath, Complete: true, Subtree: true,
		Mappings: []FrozenMapping{{
			StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.Enable",
			IsStorable:   true,
		}},
	}})

	require.NoError(t, err)
	require.True(t, ok)
	require.Contains(t, sql, "DELETE FROM device_parameters")
	require.Contains(t, sql, "parameter_path LIKE")
	require.Contains(t, sql, "last_updated_at <=")
	require.Contains(t, sql, "NOT EXISTS")
	require.Equal(t, []string{neighborPath + "%"}, args[1])
	require.Contains(t, args, runStartedAt)
	require.Contains(t, args, runID)
}

func TestBuildPartialSyncReconcileDeleteKeepsSQLAndBindCountBounded(t *testing.T) {
	coverage := make([]CoverageScope, 1000)
	for i := range coverage {
		coverage[i] = CoverageScope{
			Path: fmt.Sprintf("Device.Table.%d.", i), Complete: true, Subtree: true,
		}
	}

	sql, args, ok, err := buildPartialSyncReconcileDelete(uuid.New(), uuid.New(), time.Now(), coverage)

	require.NoError(t, err)
	require.True(t, ok)
	require.Less(t, len(sql), 500, "coverage cardinality must not expand SQL text")
	require.Len(t, args, 4, "device, prefix array, start time, and run are the only binds")
	require.Len(t, args[1], 1000)
}

func TestBuildPartialSyncReconcileDeleteDoesNotDeleteLeafOrIncompleteObject(t *testing.T) {
	deviceID := uuid.New()
	runID := uuid.New()
	coverage := []CoverageScope{
		{Path: "Device.Radio.1.Enable", Complete: false, Subtree: false},
		{Path: "Device.Radio.1.", Complete: false, Subtree: true},
	}

	_, _, ok, err := buildPartialSyncReconcileDelete(deviceID, runID, time.Now(), coverage)
	require.NoError(t, err)
	assert.False(t, ok)
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
