package datamodel

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDataModelForIterator() *DataModel {
	objects := []ObjectInfo{
		{Name: "Device.", Access: "READ_WRITE", MaxInstances: 0},
		{Name: "Device.DeviceInfo.", Access: "READ_WRITE", MaxInstances: 0},
		{Name: "Device.ManagementServer.", Access: "READ_WRITE", MaxInstances: 0},
		{Name: "Device.Services.FAPService.{12}.", Access: "READ_WRITE", MaxInstances: 0},
		{Name: "Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.", Access: "READ_WRITE", MaxInstances: 6},
		{Name: "Device.Services.FAPService.{12}.CellConfig.LTE.RAN.NeighborList.LTECell.{256}.", Access: "READ_WRITE", MaxInstances: 256},
	}
	params := []Parameter{
		{Path: "Device.DeviceInfo.Manufacturer", Type: "string"},
		{Path: "Device.ManagementServer.URL", Type: "string"},
		{Path: "Device.Services.FAPService.{12}.CellConfig.LTE.RAN.Common.CellIdentity", Type: "unsignedInt"},
		{Path: "Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.PLMNID", Type: "string"},
	}

	objJSON, _ := json.Marshal(objects)
	paramJSON, _ := json.Marshal(params)

	return &DataModel{
		ObjectTree:    objJSON,
		ParameterTree: paramJSON,
	}
}

func TestNewParameterTreeIterator(t *testing.T) {
	dm := newTestDataModelForIterator()
	it, err := NewParameterTreeIterator(dm)
	require.NoError(t, err)
	assert.NotNil(t, it)

	multiObjs := it.GetMultiInstanceObjects()
	assert.GreaterOrEqual(t, len(multiObjs), 3) // FAPService, PLMNList, NeighborList

	// Check depth ordering.
	for i := 1; i < len(multiObjs); i++ {
		assert.GreaterOrEqual(t, multiObjs[i].Depth, multiObjs[i-1].Depth)
	}
}

func TestBuildSyncPlan_WithMultiInstance(t *testing.T) {
	dm := newTestDataModelForIterator()
	it, err := NewParameterTreeIterator(dm)
	require.NoError(t, err)

	plan := it.BuildSyncPlan()
	assert.NotNil(t, plan)

	// Should have GPN requests for multi-instance objects.
	assert.Greater(t, len(plan.Phase1GPNs), 0)

	// Should have static prefixes for non-multi-instance objects.
	assert.Greater(t, len(plan.StaticPrefixes), 0)

	// Verify depth-0 GPNs include FAPService.
	depth0 := 0
	for _, gpn := range plan.Phase1GPNs {
		if gpn.Depth == 0 {
			depth0++
			assert.False(t, ContainsPlaceholder(gpn.BasePath))
		}
	}
	assert.Greater(t, depth0, 0)
}

func TestBuildSyncPlan_NoMultiInstance(t *testing.T) {
	objects := []ObjectInfo{
		{Name: "Device.", Access: "READ_WRITE"},
		{Name: "Device.DeviceInfo.", Access: "READ_WRITE"},
		{Name: "Device.ManagementServer.", Access: "READ_WRITE"},
	}
	params := []Parameter{
		{Path: "Device.DeviceInfo.Manufacturer", Type: "string"},
	}

	objJSON, _ := json.Marshal(objects)
	paramJSON, _ := json.Marshal(params)
	dm := &DataModel{ObjectTree: objJSON, ParameterTree: paramJSON}

	it, err := NewParameterTreeIterator(dm)
	require.NoError(t, err)

	plan := it.BuildSyncPlan()
	assert.Len(t, plan.Phase1GPNs, 0)
	assert.Greater(t, len(plan.StaticPrefixes), 0)
}

func TestExpandGPNsForDepth(t *testing.T) {
	dm := newTestDataModelForIterator()
	it, err := NewParameterTreeIterator(dm)
	require.NoError(t, err)

	plan := it.BuildSyncPlan()

	// Depth 0 GPNs can be sent directly.
	depth0 := it.ExpandGPNsForDepth(plan, 0, InstanceMap{})
	assert.Greater(t, len(depth0), 0)

	// Depth 1 GPNs need parent instances.
	instances := InstanceMap{
		"Device.Services.FAPService.": {1, 2},
	}
	depth1 := it.ExpandGPNsForDepth(plan, 1, instances)
	// Should have expanded GPNs for each parent instance x each depth-1 object.
	assert.Greater(t, len(depth1), 0)
	for _, gpn := range depth1 {
		assert.False(t, ContainsPlaceholder(gpn.BasePath), "expanded GPN should not contain placeholders: %s", gpn.BasePath)
	}
}

func TestBuildGPVPrefixes(t *testing.T) {
	dm := newTestDataModelForIterator()
	it, err := NewParameterTreeIterator(dm)
	require.NoError(t, err)

	plan := it.BuildSyncPlan()

	instances := InstanceMap{
		"Device.Services.FAPService.": {1, 2},
	}

	prefixes := it.BuildGPVPrefixes(plan, instances)
	assert.Greater(t, len(prefixes), 0)

	// Should include static prefixes.
	hasStatic := false
	for _, p := range prefixes {
		if !ContainsPlaceholder(p) {
			hasStatic = true
		}
	}
	assert.True(t, hasStatic)

	// Should include instance prefixes.
	hasInstance := false
	for _, p := range prefixes {
		if p == "Device.Services.FAPService.1." || p == "Device.Services.FAPService.2." {
			hasInstance = true
		}
	}
	assert.True(t, hasInstance)
}

func TestMaxDepth(t *testing.T) {
	dm := newTestDataModelForIterator()
	it, err := NewParameterTreeIterator(dm)
	require.NoError(t, err)

	assert.Equal(t, 1, it.MaxDepth())
}
