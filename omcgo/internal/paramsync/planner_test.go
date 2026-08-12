package paramsync

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
)

type staticMappings struct{ set *parammodel.MappingSet }

func (s staticMappings) GetMappingSet(context.Context, *model.Device) (*parammodel.MappingSet, error) {
	return s.set, nil
}

type staticReadUnsupportedPaths struct {
	productID uuid.UUID
	firmware  string
	paths     []string
}

func (s staticReadUnsupportedPaths) ListReadUnsupportedPaths(_ context.Context, productID uuid.UUID, firmware string) ([]string, error) {
	if productID != s.productID || firmware != s.firmware {
		return nil, nil
	}
	return append([]string(nil), s.paths...), nil
}

func TestPlanner_FullSyncFiltersLearnedLeaf9005ByProductAndFirmware(t *testing.T) {
	productID := uuid.New()
	set := &parammodel.MappingSet{
		ProductID: productID, SoftwareVersion: "FW-1", Mappings: []parammodel.ParamMapping{
			{StandardPath: "Device.Good", PrivatePath: "Vendor.Good", IsStorable: true, IsSupported: true},
			{StandardPath: "Device.Bad", PrivatePath: "Vendor.Bad", IsStorable: true, IsSupported: true},
		},
	}
	planner := NewPlanner(staticMappings{set: set}, 50).WithUnsupportedPaths(staticReadUnsupportedPaths{
		productID: productID, firmware: "FW-1", paths: []string{"Device.Bad"},
	})

	full, err := planner.Plan(context.Background(), PlanCommand{Device: &model.Device{}, Scope: SyncScopeFull})
	require.NoError(t, err)
	assert.Equal(t, []string{"Vendor.Good"}, flattenPaths(full.Batches))
	assert.Equal(t, []CoverageScope{{
		Path: "Device.Good", Complete: true,
		Mappings: []FrozenMapping{{StandardPath: "Device.Good", PrivatePath: "Vendor.Good", IsStorable: true}},
	}}, full.Coverage)

	partial, err := planner.Plan(context.Background(), PlanCommand{
		Device: &model.Device{}, Scope: SyncScopeReadback, RequestedPaths: []string{"Device.Bad"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Vendor.Bad"}, flattenPaths(partial.Batches),
		"an explicit readback remains a supported way to re-probe a learned path")

	otherFirmware := *set
	otherFirmware.SoftwareVersion = "FW-2"
	other, err := NewPlanner(staticMappings{set: &otherFirmware}, 50).
		WithUnsupportedPaths(staticReadUnsupportedPaths{productID: productID, firmware: "FW-1", paths: []string{"Device.Bad"}}).
		Plan(context.Background(), PlanCommand{Device: &model.Device{}, Scope: SyncScopeFull})
	require.NoError(t, err)
	assert.Equal(t, []string{"Vendor.Bad", "Vendor.Good"}, flattenPaths(other.Batches),
		"a fault learned for FW-1 must not suppress the path on FW-2")
}

func TestLearnableRecovered9005RequiresExactRequestedLeafAndFrozenMapping(t *testing.T) {
	coverage := []CoverageScope{{Mappings: []FrozenMapping{
		{StandardPath: "Device.Bad", PrivatePath: "Vendor.Bad", IsStorable: true},
		{StandardPath: "Device.Table.{i}.Value", PrivatePath: "Vendor.Table.{i}.Value", IsStorable: true},
	}}}

	path, ok := learnableRecovered9005StandardPath(storedTaskResult{
		Recovered: true, FaultCode: 9005, BadPath: "Vendor.Bad", RequestedNames: []string{"Vendor.Bad", "Vendor.Good"},
	}, coverage)
	require.True(t, ok)
	assert.Equal(t, "Device.Bad", path)

	_, ok = learnableRecovered9005StandardPath(storedTaskResult{
		Recovered: true, FaultCode: 9005, BadPath: "Vendor.Table.7.Value", RequestedNames: []string{"Vendor.Table."},
	}, coverage)
	assert.False(t, ok, "a child fault returned for an object-prefix request is not a certain unsupported leaf")

	_, ok = learnableRecovered9005StandardPath(storedTaskResult{
		Recovered: true, FaultCode: 9005, BadPath: "Vendor.Table.", RequestedNames: []string{"Vendor.Table."},
	}, coverage)
	assert.False(t, ok, "object-prefix and zero-instance faults must not be learned permanently")

	_, ok = learnableRecovered9005StandardPath(storedTaskResult{
		Recovered: true, FaultCode: 9005, BadPath: "Vendor.Unknown", RequestedNames: []string{"Vendor.Unknown"},
	}, coverage)
	assert.False(t, ok, "an unmatched private path cannot be safely persisted as a standard path")
}

func TestPlanner_ScopeIsStructuralAndDeterministic(t *testing.T) {
	set := &parammodel.MappingSet{Source: parammodel.MappingSourceDiscovered, SoftwareVersion: "1.2.3", Mappings: []parammodel.ParamMapping{
		{StandardPath: "Device.Radio.{i}.Enable", PrivatePath: "Device.X.Radio.{i}.Enable", IsStorable: true, IsSupported: true},
		{StandardPath: "Device.Info.Serial", PrivatePath: "Device.X.Info.Serial", IsStorable: true, IsSupported: true},
		{StandardPath: "Device.Secret", PrivatePath: "Device.X.Secret", IsStorable: false, IsSupported: true},
	}}
	planner := NewPlanner(staticMappings{set: set}, 50)
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-1"}

	full, err := planner.Plan(context.Background(), PlanCommand{Device: dev, Scope: SyncScopeFull})
	require.NoError(t, err)
	assert.Equal(t, []string{"Device.X.Info.Serial", "Device.X.Radio."}, flattenPaths(full.Batches))
	assert.Equal(t, []CoverageScope{
		{Path: "Device.Info.Serial", Complete: true, Mappings: []FrozenMapping{{StandardPath: "Device.Info.Serial", PrivatePath: "Device.X.Info.Serial", IsStorable: true}}},
		{Path: "Device.Radio.", Complete: true, Subtree: true, Mappings: []FrozenMapping{{StandardPath: "Device.Radio.{i}.Enable", PrivatePath: "Device.X.Radio.{i}.Enable", IsStorable: true}}},
	}, full.Coverage)

	partial, err := planner.Plan(context.Background(), PlanCommand{
		Device: dev, Scope: SyncScopePartial, RequestedPaths: []string{"Device.Radio.7.Enable"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Device.X.Radio."}, flattenPaths(partial.Batches))
	assert.False(t, partial.Coverage[0].Complete)
	require.Len(t, partial.Coverage[0].Mappings, 1)
	assert.Equal(t, "Device.Radio.{i}.Enable", partial.Coverage[0].Mappings[0].StandardPath)

	privatePartial, err := planner.Plan(context.Background(), PlanCommand{
		Device: dev, Scope: SyncScopePartial, RequestedPaths: []string{"Device.X.Radio.7.Enable"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Device.X.Radio."}, flattenPaths(privatePartial.Batches))
	assert.Equal(t, []string{"Device.Radio.7.Enable"}, privatePartial.RequestedPaths)

	readback, err := planner.Plan(context.Background(), PlanCommand{
		Device: dev, Scope: SyncScopeReadback, RequestedPaths: []string{"Device.Info.Serial"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Device.X.Info.Serial"}, flattenPaths(readback.Batches))

	objectReadback, err := planner.Plan(context.Background(), PlanCommand{
		Device: dev, Scope: SyncScopeReadback, RequestedPaths: []string{"Device.Radio.7."},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Device.X.Radio.7."}, flattenPaths(objectReadback.Batches))
	require.Len(t, objectReadback.Coverage, 1)
	assert.True(t, objectReadback.Coverage[0].Subtree)
	assert.False(t, objectReadback.Coverage[0].Complete, "SPV/AddObject readback must not delete absent sibling data")
	require.Len(t, objectReadback.Coverage[0].Mappings, 1)

	privateObjectReadback, err := planner.Plan(context.Background(), PlanCommand{
		Device: dev, Scope: SyncScopeReadback, RequestedPaths: []string{"Device.X.Radio.7."},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Device.Radio.7."}, privateObjectReadback.RequestedPaths)
	assert.Equal(t, []string{"Device.X.Radio.7."}, flattenPaths(privateObjectReadback.Batches))
	assert.True(t, privateObjectReadback.Coverage[0].Subtree)
	require.Len(t, privateObjectReadback.Coverage[0].Mappings, 1)
}

func TestBuildCoverage_SeparatesNestedMultiInstanceObjectsAndMakesPartialObjectAuthoritative(t *testing.T) {
	mappings := []parammodel.ParamMapping{
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.Enable", PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.Enable", EntryType: "parameter", IsStorable: true, IsSupported: true},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.Enable", PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.Enable", EntryType: "parameter", IsStorable: true, IsSupported: true},
	}

	full := buildCoverage(mappings, SyncScopeFull, nil)
	require.Len(t, full, 2)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.", full[0].Path)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.", full[1].Path)
	assert.True(t, full[0].Complete)
	assert.True(t, full[1].Complete)
	require.Len(t, full[0].Mappings, 1)
	require.Len(t, full[1].Mappings, 1)
	assert.NotEqual(t, full[0].Mappings[0].StandardPath, full[1].Mappings[0].StandardPath)

	partial := buildCoverage(mappings, SyncScopePartial, []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.",
	})
	require.Len(t, partial, 1)
	assert.Equal(t, "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.", partial[0].Path)
	assert.True(t, partial[0].Subtree)
	assert.True(t, partial[0].Complete, "an explicit object-prefix refresh is an authoritative snapshot")
}

func TestBuildCoverage_AddsObjectBoundaryToPartialObjectWithoutTrailingDot(t *testing.T) {
	mappings := []parammodel.ParamMapping{{
		StandardPath: "Device.Radio.{i}.", PrivatePath: "Device.Radio.{i}.",
		EntryType: "object", IsStorable: true, IsSupported: true,
	}}

	coverage := buildCoverage(mappings, SyncScopePartial, []string{"Device.Radio.1"})

	require.Len(t, coverage, 1)
	assert.Equal(t, "Device.Radio.1.", coverage[0].Path)
	assert.True(t, coverage[0].Subtree)
	assert.True(t, coverage[0].Complete)
}

func TestPlanner_MLNDCFullSyncQueriesAuthoritativeRFStateForEveryCarrier(t *testing.T) {
	set := &parammodel.MappingSet{Mappings: []parammodel.ParamMapping{{
		StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
		PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState",
		IsStorable:   true,
		IsSupported:  true,
	}}}

	plan, err := NewPlanner(staticMappings{set: set}, 50).Plan(
		context.Background(),
		PlanCommand{
			Device: &model.Device{ProductClass: "FAP/MLN/DC"},
			Scope:  SyncScopeFull,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState",
	}, flattenPaths(plan.Batches))
}

func TestPlanner_MLNDCRFPathsScaleByCarrierCount(t *testing.T) {
	assert.Equal(t, []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState",
		"Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.AdminCellState",
	}, mlnDCRFStatusPaths(SyncScopeFull, nil, 3))
}

func TestPlanner_MLNDCRFPathsAreFaultIsolated(t *testing.T) {
	set := &parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			IsStorable:   true,
			IsSupported:  true,
		},
		{
			StandardPath: "Device.Safe.Status",
			PrivatePath:  "Device.Safe.Status",
			IsStorable:   true,
			IsSupported:  true,
		},
	}}

	plan, err := NewPlanner(staticMappings{set: set}, 50).Plan(
		context.Background(),
		PlanCommand{Device: &model.Device{ProductClass: mlnDCProductClass}, Scope: SyncScopeFull},
	)

	require.NoError(t, err)
	require.Len(t, plan.Batches, 3)
	assert.Equal(t, []string{"Device.Safe.Status"}, plan.Batches[0].Paths)
	assert.Equal(t, []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
	}, plan.Batches[1].Paths)
	assert.Equal(t, []string{
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState",
	}, plan.Batches[2].Paths)
}

func TestPlanner_MLNDCReadbackOnlyQueriesRequestedRFCarrier(t *testing.T) {
	set := &parammodel.MappingSet{Mappings: []parammodel.ParamMapping{{
		StandardPath: mlnDCRFStatusStandardPath,
		PrivatePath:  mlnDCRFStatusPrivatePath,
		IsStorable:   true,
		IsSupported:  true,
	}}}

	plan, err := NewPlanner(staticMappings{set: set}, 50).Plan(
		context.Background(),
		PlanCommand{
			Device: &model.Device{ProductClass: mlnDCProductClass},
			Scope:  SyncScopeReadback,
			RequestedPaths: []string{
				"Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus",
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState",
	}, flattenPaths(plan.Batches))
}

func TestPlanner_NonDCProductKeepsExistingRFPlanning(t *testing.T) {
	set := &parammodel.MappingSet{Mappings: []parammodel.ParamMapping{{
		StandardPath: mlnDCRFStatusStandardPath,
		PrivatePath:  mlnDCRFStatusPrivatePath,
		IsStorable:   true,
		IsSupported:  true,
	}}}

	plan, err := NewPlanner(staticMappings{set: set}, 50).Plan(
		context.Background(),
		PlanCommand{Device: &model.Device{ProductClass: "FAP/MLN/SC"}, Scope: SyncScopeFull},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
	}, flattenPaths(plan.Batches))
}

func TestProjectTaskValuesMapsMLNRFStateToStandardPath(t *testing.T) {
	coverage := []CoverageScope{{
		Path: "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
		Mappings: []FrozenMapping{{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
			PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState",
			Access:       "readWrite",
			DataType:     "boolean",
			IsStorable:   true,
		}},
	}}

	got := projectTaskValues([]tr069.ParameterValueStruct{
		{
			Name:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
			Value: "1",
			Type:  "xsd:unsignedInt",
		},
	}, coverage)

	require.Len(t, got, 1)
	assert.Equal(t, projectedValue{
		ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		PrivatePath:   "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
		Value:         "1",
		ParameterType: "unsignedInt",
		Writable:      true,
		FAPInstance:   1,
		ParamGroup:    "fap_control",
	}, got[0])
}

func TestProjectTaskValuesUsesFrozenMappingSemantics(t *testing.T) {
	coverage := []CoverageScope{{
		Path: "Device.Radio.", Complete: true, Subtree: true,
		Mappings: []FrozenMapping{
			{StandardPath: "Device.Radio.{i}.Enable", PrivatePath: "Device.X.Radio.{i}.Enable", Access: "readWrite", DataType: "boolean", IsStorable: true},
			{StandardPath: "Device.Radio.{i}.Secret", PrivatePath: "Device.X.Radio.{i}.Secret", IsStorable: false},
		},
	}}

	got := projectTaskValues([]tr069.ParameterValueStruct{
		{Name: "Device.X.Radio.7.Enable", Value: "1", Type: "xsd:boolean"},
		{Name: "Device.X.Radio.7.Secret", Value: "hidden", Type: "xsd:string"},
		{Name: "Device.X.Vendor.Extension", Value: "kept", Type: "xsd:unsignedInt"},
	}, coverage)

	require.Len(t, got, 2)
	assert.Equal(t, projectedValue{ParameterPath: "Device.Radio.7.Enable", PrivatePath: "Device.X.Radio.7.Enable", Value: "1", ParameterType: "boolean", Writable: true, FAPInstance: 0, ParamGroup: "other"}, got[0])
	assert.Equal(t, projectedValue{ParameterPath: "Device.X.Vendor.Extension", PrivatePath: "Device.X.Vendor.Extension", Value: "kept", ParameterType: "unsignedInt", Writable: false, FAPInstance: 0, ParamGroup: "other"}, got[1])
}

// TestDedupeProjectedValuesByPath_KeepsLastAndOriginalOrder 覆盖批量 INSERT ... ON CONFLICT
// 上批前必须去重的场景：不同 private path（如不同实例）被 translator 映射到同一个
// StandardPath 时（多实例映射未命中兜底等已知场景），同一条多行 INSERT 语句里对同一冲突
// 目标 DO UPDATE 两次会被 PG 拒绝（"ON CONFLICT DO UPDATE command cannot affect row a
// second time"），所以必须先去重、且要保留"后写覆盖前写"的原逐条执行语义。
func TestDedupeProjectedValuesByPath_KeepsLastAndOriginalOrder(t *testing.T) {
	values := []projectedValue{
		{ParameterPath: "Device.Radio.1.Enable", Value: "first"},
		{ParameterPath: "Device.Radio.2.Enable", Value: "only"},
		{ParameterPath: "Device.Radio.1.Enable", Value: "last-wins"},
	}

	got := dedupeProjectedValuesByPath(values)

	require.Len(t, got, 2)
	assert.Equal(t, projectedValue{ParameterPath: "Device.Radio.1.Enable", Value: "last-wins"}, got[0])
	assert.Equal(t, projectedValue{ParameterPath: "Device.Radio.2.Enable", Value: "only"}, got[1])
}

func TestDedupeProjectedValuesByPath_NoDuplicatesReturnsSameValues(t *testing.T) {
	values := []projectedValue{
		{ParameterPath: "Device.Radio.1.Enable", Value: "a"},
		{ParameterPath: "Device.Radio.2.Enable", Value: "b"},
	}

	got := dedupeProjectedValuesByPath(values)

	assert.Equal(t, values, got)
}

func TestPlanner_BatchesStayBelowNATSBudget(t *testing.T) {
	mappings := make([]parammodel.ParamMapping, 0, 1000)
	for i := range 1000 {
		path := "Device.Param." + uuid.NewSHA1(uuid.Nil, []byte{byte(i)}).String()
		mappings = append(mappings, parammodel.ParamMapping{StandardPath: path, PrivatePath: path, IsStorable: true, IsSupported: true})
	}
	planner := NewPlanner(staticMappings{set: &parammodel.MappingSet{Mappings: mappings}}, 1000)
	plan, err := planner.Plan(context.Background(), PlanCommand{Device: &model.Device{}, Scope: SyncScopeFull})
	require.NoError(t, err)
	for _, batch := range plan.Batches {
		assert.LessOrEqual(t, batch.EstimatedPayloadBytes, GPVNATSPayloadBudgetBytes)
	}
}

func TestPlanner_UsesLegacyPathBBatchIsolation(t *testing.T) {
	set := &parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{StandardPath: "Device.Scalar.B", PrivatePath: "Device.Scalar.B", IsStorable: true, IsSupported: true},
		{StandardPath: "Device.Table.{i}.Value", PrivatePath: "Device.X.Table.{i}.Value", IsStorable: true, IsSupported: true},
		{StandardPath: "Device.Scalar.A", PrivatePath: "Device.Scalar.A", IsStorable: true, IsSupported: true},
	}}
	planner := NewPlanner(staticMappings{set: set}, 50)

	plan, err := planner.Plan(context.Background(), PlanCommand{Device: &model.Device{}, Scope: SyncScopeFull})
	require.NoError(t, err)
	require.Len(t, plan.Batches, 2)
	assert.Equal(t, []string{"Device.Scalar.A", "Device.Scalar.B"}, plan.Batches[0].Paths)
	assert.Equal(t, []string{"Device.X.Table."}, plan.Batches[1].Paths,
		"Path B keeps an enumerable object prefix in its own GPV batch")
}

func flattenPaths(batches []TaskBatch) []string {
	var paths []string
	for _, batch := range batches {
		paths = append(paths, batch.Paths...)
	}
	return paths
}
