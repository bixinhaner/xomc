package carrier

import (
	"fmt"
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestBuildGeofenceDeactivationCapabilityAcceptsAdminCellStateAsCombinedControl(t *testing.T) {
	capability, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/MLN/TC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "1", Writable: true},
			{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1"},
		},
		[]GeofenceControlMapping{
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
				"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState",
				"READ_WRITE",
			),
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"READ_ONLY",
			),
			activeMapping(
				"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
				"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
				"READ_WRITE",
			),
		},
	)

	require.NoError(t, err)
	require.Equal(t, []GeofenceControlParameter{
		{
			Path: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", Value: "0",
			Role: GeofenceRoleAdminRF, AccessProven: true,
		},
		{
			Path: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", Value: "0",
			Role: GeofenceRoleIPSec, AccessProven: true,
		},
	}, capability.Controls)
	require.Equal(t, GeofenceRoleOpState, capability.Terminals[0].Role)
	require.True(t, capability.Terminals[0].AccessProven)
}

func TestBuildGeofenceDeactivationCapabilityDoesNotShare452EvidenceWithBLN(t *testing.T) {
	_, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/CR-B4860/TC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1"},
		},
		[]GeofenceControlMapping{
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
				"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState",
				"READ_WRITE",
			),
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"READ_ONLY",
			),
		},
	)

	require.ErrorContains(t, err, "no model-proven writable cell administration control")
}

func TestBuildGeofenceDeactivationCapabilityRequiresParamModel(t *testing.T) {
	_, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/MLN/DC", model.TechLTE, false,
		[]model.DeviceParameter{{
			ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
			Writable:      true,
		}},
		nil,
	)

	require.ErrorContains(t, err, "has no ParamModel mappings")
}

func TestBuildGeofenceDeactivationCapabilityRejectsRadioOnlyBLQ(t *testing.T) {
	_, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/BAIBLQ/SC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1", Writable: true},
		},
		[]GeofenceControlMapping{
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
				"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
				"READ_WRITE",
			),
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"READ_ONLY",
			),
		},
	)

	require.ErrorContains(t, err, "no model-proven writable cell administration control")
}

func TestBuildGeofenceDeactivationCapabilityResolvesBLQPrivateAdminState(t *testing.T) {
	capability, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/BAIBLQ/SC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.DeviceInfo.FAP_adminstate", ParameterValue: "true"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1"},
		},
		[]GeofenceControlMapping{
			activeMapping(
				"Device.Services.FAPService.1.FAPControl.LTE.AdminState",
				"Device.DeviceInfo.FAP_adminstate",
				"READ_WRITE",
			),
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
				"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
				"READ_WRITE",
			),
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"READ_ONLY",
			),
			activeMapping(
				"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
				"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
				"READ_WRITE",
			),
		},
	)

	require.NoError(t, err)
	require.Equal(t, GeofenceControlParameter{
		Path:              "Device.Services.FAPService.1.FAPControl.LTE.AdminState",
		SnapshotPath:      "Device.DeviceInfo.FAP_adminstate",
		Value:             "0",
		Role:              GeofenceRoleAdmin,
		AccessProven:      true,
		AppliesToAllCells: true,
	}, capability.Controls[0])
	require.Equal(t, "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", capability.Controls[1].Path)
	require.Len(t, capability.Terminals, 1)
}

func TestBuildGeofenceDeactivationCapabilityUsesConfiguredCellCount(t *testing.T) {
	parameters := []model.DeviceParameter{
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", ParameterValue: "1"},
		{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1"},
	}
	for instance := 1; instance <= 3; instance++ {
		parameters = append(parameters,
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.RFTxStatus", instance), ParameterValue: "1"},
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.OpState", instance), ParameterValue: "1"},
		)
	}
	capability, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/MLN/TC", model.TechLTE, false, parameters, mlnCombinedMappings(),
	)

	require.NoError(t, err)
	require.Equal(t, []string{
		"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
	}, controlParameterPaths(capability.Controls))
	require.Equal(t, []string{
		"Device.Services.FAPService.1.FAPControl.LTE.OpState",
	}, controlParameterPaths(capability.Terminals))
}

func TestBuildGeofenceDeactivationCapabilityFallsBackToMaximumCells(t *testing.T) {
	parameters := []model.DeviceParameter{{
		ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1",
	}}
	for instance := 1; instance <= 3; instance++ {
		parameters = append(parameters,
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.RFTxStatus", instance), ParameterValue: "1"},
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.OpState", instance), ParameterValue: "1"},
		)
	}
	capability, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/MLN/TC", model.TechLTE, false, parameters, mlnCombinedMappings(),
	)

	require.NoError(t, err)
	require.Len(t, capability.Controls, 4)
	require.Len(t, capability.Terminals, 3)
}

func TestBuildGeofenceDeactivationCapabilityUsesBLQGlobalAdminForMaximumCells(t *testing.T) {
	parameters := []model.DeviceParameter{
		{ParameterPath: "Device.DeviceInfo.FAP_adminstate", ParameterValue: "true"},
		{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1"},
	}
	for instance := 1; instance <= 2; instance++ {
		parameters = append(parameters,
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.RFTxStatus", instance), ParameterValue: "1"},
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.OpState", instance), ParameterValue: "1"},
		)
	}
	countMapping := activeMapping(
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
		"FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
		"READ_WRITE",
	)
	countMapping.EnumValues = "1,2"
	capability, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/mBS31001/DC", model.TechLTE, false, parameters, []GeofenceControlMapping{
			activeMapping("Device.Services.FAPService.1.FAPControl.LTE.AdminState", "Device.DeviceInfo.FAP_adminstate", "READ_WRITE"),
			activeMapping("Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", "READ_WRITE"),
			activeMapping("Device.Services.FAPService.{i}.FAPControl.LTE.OpState", "Device.Services.FAPService.{i}.FAPControl.LTE.OpState", "READ_ONLY"),
			activeMapping("Device.Services.FAPService.Ipsec.IPSEC_ENABLE", "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", "READ_WRITE"),
			countMapping,
		},
	)

	require.NoError(t, err)
	require.True(t, capability.Controls[0].AppliesToAllCells)
	require.Len(t, capability.Controls, 4)
	require.Len(t, capability.Terminals, 2)
}

func TestBuildGeofenceDeactivationCapabilityRejectsIncompleteModelMaximumSnapshot(t *testing.T) {
	_, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/MLN/SC", model.TechLTE, false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1"},
		},
		mlnCombinedMappings(),
	)

	require.ErrorContains(t, err, "OpState/model maximum instance count differs (1/3)")
}

func TestModelMaximumGeofenceCellsRespectsProductRadioModes(t *testing.T) {
	countMapping := activeMapping(
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
		"FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
		"READ_WRITE",
	)
	countMapping.EnumValues = "1,2,3"

	countMapping.ProductRadioModes = "SC"
	require.Equal(t, map[int]struct{}{1: {}}, modelMaximumGeofenceCellInstances([]GeofenceControlMapping{countMapping}))

	countMapping.ProductRadioModes = "SC,CA,DC"
	require.Equal(t, map[int]struct{}{1: {}, 2: {}, 3: {}}, modelMaximumGeofenceCellInstances([]GeofenceControlMapping{countMapping}))

	countMapping.EnumValues = ""
	countMapping.ProductRadioModes = "SC,CA,DC,TC"
	require.Equal(t, map[int]struct{}{1: {}, 2: {}, 3: {}}, modelMaximumGeofenceCellInstances([]GeofenceControlMapping{countMapping}))
}

func TestBuildGeofenceDeactivationCapabilityUsesInUseCells(t *testing.T) {
	parameters := []model.DeviceParameter{{
		ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1",
	}}
	for instance := 1; instance <= 3; instance++ {
		inUse := "0"
		if instance == 1 {
			inUse = "1"
		}
		parameters = append(parameters,
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.InUse", instance), ParameterValue: inUse},
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.AdminState", instance), ParameterValue: "1"},
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.RFTxStatus", instance), ParameterValue: "1"},
			model.DeviceParameter{ParameterPath: fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.OpState", instance), ParameterValue: "1"},
		)
	}
	capability, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/BU1810", model.TechLTE, false, parameters, []GeofenceControlMapping{
			activeMapping("Device.Services.FAPService.{i}.FAPControl.LTE.AdminState", "Device.Services.FAPService.{i}.FAPControl.LTE.AdminState", "READ_WRITE"),
			activeMapping("Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", "READ_WRITE"),
			activeMapping("Device.Services.FAPService.{i}.FAPControl.LTE.OpState", "Device.Services.FAPService.{i}.FAPControl.LTE.OpState", "READ_ONLY"),
			activeMapping("Device.Services.FAPService.Ipsec.IPSEC_ENABLE", "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", "READ_WRITE"),
		},
	)

	require.NoError(t, err)
	require.Equal(t, []string{
		"Device.Services.FAPService.1.FAPControl.LTE.AdminState",
		"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus",
		"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
	}, controlParameterPaths(capability.Controls))
	require.Len(t, capability.Terminals, 1)
}

func TestBuildGeofenceDeactivationCapabilityUsesSeparateAdminForRadioEnableProducts(t *testing.T) {
	capability, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/BM/SC",
		model.TechLTE,
		false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.AdminState", ParameterValue: "1", Writable: true},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1", Writable: true},
		},
		[]GeofenceControlMapping{
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
				"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable",
				"READ_WRITE",
			),
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.AdminState",
				"Device.Services.FAPService.{i}.FAPControl.LTE.AdminState",
				"READ_WRITE",
			),
			activeMapping(
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
				"READ_ONLY",
			),
			activeMapping(
				"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
				"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
				"READ_WRITE",
			),
		},
	)

	require.NoError(t, err)
	require.Equal(t, GeofenceRoleAdmin, capability.Controls[0].Role)
	require.Equal(t, GeofenceRoleRF, capability.Controls[1].Role)
	require.Equal(t, GeofenceRoleIPSec, capability.Controls[2].Role)
}

func TestBuildGeofenceDeactivationCapabilityRequiresRFForEveryAdminCell(t *testing.T) {
	_, err := BuildGeofenceDeactivationCapabilityForSnapshotWithMappings(
		"FAP/BU1810", model.TechLTE, false,
		[]model.DeviceParameter{
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.AdminState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.AdminState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.OpState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.2.FAPControl.LTE.OpState", ParameterValue: "1"},
			{ParameterPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", ParameterValue: "1"},
		},
		[]GeofenceControlMapping{
			activeMapping("Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus", "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable", "READ_WRITE"),
			activeMapping("Device.Services.FAPService.{i}.FAPControl.LTE.AdminState", "Device.Services.FAPService.{i}.FAPControl.LTE.AdminState", "READ_WRITE"),
			activeMapping("Device.Services.FAPService.{i}.FAPControl.LTE.OpState", "Device.Services.FAPService.{i}.FAPControl.LTE.OpState", "READ_ONLY"),
			activeMapping("Device.Services.FAPService.Ipsec.IPSEC_ENABLE", "Device.Services.FAPService.Ipsec.IPSEC_ENABLE", "READ_WRITE"),
		},
	)

	require.ErrorContains(t, err, "RF/OpState instance count differs")
}

func activeMapping(standardPath, privatePath, access string) GeofenceControlMapping {
	return GeofenceControlMapping{
		StandardPath: standardPath,
		PrivatePath:  privatePath,
		EntryType:    "parameter",
		Access:       access,
		IsActive:     true,
		IsSupported:  true,
	}
}

func mlnCombinedMappings() []GeofenceControlMapping {
	countMapping := activeMapping(
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
		"FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells",
		"READ_WRITE",
	)
	countMapping.EnumValues = "1,2,3"
	countMapping.ProductRadioModes = "SC,CA,DC"
	return []GeofenceControlMapping{
		activeMapping(
			"Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus",
			"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState",
			"READ_WRITE",
		),
		activeMapping(
			"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
			"Device.Services.FAPService.{i}.FAPControl.LTE.OpState",
			"READ_ONLY",
		),
		activeMapping(
			"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
			"Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
			"READ_WRITE",
		),
		countMapping,
	}
}
