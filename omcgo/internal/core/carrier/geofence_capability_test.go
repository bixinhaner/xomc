package carrier

import (
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
