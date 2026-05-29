package parammodel

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanDiscoveredObjectInserts_AddsMissingTemplateParents(t *testing.T) {
	productID := uuid.New()
	objects := []ParamMapping{
		{PrivatePath: "Device.Services.FAPService.{i}.", EntryType: "object", StandardPath: "Device.Services.FAPService.{i}."},
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.", EntryType: "object", StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell."},
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.", EntryType: "object", StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}."},
	}
	discovered := []discoveredMappingRef{
		{
			ProductID:       productID,
			SoftwareVersion: "1.0",
			PrivatePath:     "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.7.CID",
		},
	}

	got := planDiscoveredObjectInserts(objects, discovered)
	require.Len(t, got, 3)

	paths := make([]string, 0, len(got))
	for _, item := range got {
		assert.Equal(t, productID, item.ProductID)
		assert.Equal(t, "1.0", item.SoftwareVersion)
		paths = append(paths, item.Mapping.PrivatePath)
	}
	assert.ElementsMatch(t, []string{
		"Device.Services.FAPService.{i}.",
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.",
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.",
	}, paths)
}

func TestPlanDiscoveredObjectInserts_SkipsExistingTemplateParents(t *testing.T) {
	productID := uuid.New()
	objects := []ParamMapping{
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.", EntryType: "object"},
		{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.", EntryType: "object"},
	}
	discovered := []discoveredMappingRef{
		{
			ProductID:       productID,
			SoftwareVersion: "1.0",
			PrivatePath:     "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.7.CID",
		},
		{
			ProductID:       productID,
			SoftwareVersion: "1.0",
			PrivatePath:     "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.",
		},
	}

	got := planDiscoveredObjectInserts(objects, discovered)
	require.Len(t, got, 1)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.", got[0].Mapping.PrivatePath)
}