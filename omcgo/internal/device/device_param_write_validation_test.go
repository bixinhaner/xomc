package device

import (
	"testing"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newParameterWriteModelForTest(mappings []parammodel.ParamMapping) *parameterWriteModel {
	set := &parammodel.MappingSet{Mappings: mappings}
	return &parameterWriteModel{
		validator:  parammodel.NewMappingValidator(set),
		translator: parammodel.NewTranslator(set, nil, nil),
	}
}

func TestParameterWriteValidationAcceptsMLNStandardTunnelPath(t *testing.T) {
	writeModel := newParameterWriteModelForTest([]parammodel.ParamMapping{{
		StandardPath: "Device.FAP.Ipsec.{i}.TUNNEL_ENABLE",
		PrivatePath:  "Device.FAP.Ipsec.{i}.TUNNEL_CONFIG_TUNNELENABLE",
		EntryType:    "parameter",
		Access:       "READ_WRITE",
		DataType:     "BOOLEAN",
	}})

	result := validateParameterWrites(writeModel, []ParameterValueItem{{
		Path:  "Device.FAP.Ipsec.1.TUNNEL_ENABLE",
		Value: "0",
		Type:  "BOOLEAN",
	}})

	require.Empty(t, result.Errors)
	assert.False(t, result.RebootRequired)
}

func TestParameterWriteValidationStillAcceptsPrivateRuntimePath(t *testing.T) {
	writeModel := newParameterWriteModelForTest([]parammodel.ParamMapping{{
		StandardPath: "Device.FAP.Ipsec.{i}.TUNNEL_ENABLE",
		PrivatePath:  "Device.FAP.Ipsec.{i}.TUNNEL_CONFIG_TUNNELENABLE",
		EntryType:    "parameter",
		Access:       "READ_WRITE",
		DataType:     "BOOLEAN",
	}})

	result := validateParameterWrites(writeModel, []ParameterValueItem{{
		Path:  "Device.FAP.Ipsec.1.TUNNEL_CONFIG_TUNNELENABLE",
		Value: "1",
		Type:  "BOOLEAN",
	}})

	require.Empty(t, result.Errors)
}

func TestParameterWriteValidationRejectsUnknownPathUsingOriginalPath(t *testing.T) {
	writeModel := newParameterWriteModelForTest([]parammodel.ParamMapping{{
		StandardPath: "Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
		PrivatePath:  "Device.Services.FAPService.Ipsec.IPSEC_ENABLE",
		EntryType:    "parameter",
		Access:       "READ_WRITE",
		DataType:     "BOOLEAN",
	}})

	const originalPath = "Device.FAP.Ipsec.1.UNKNOWN"
	result := validateParameterWrites(writeModel, []ParameterValueItem{{
		Path:  originalPath,
		Value: "1",
		Type:  "BOOLEAN",
	}})

	require.Len(t, result.Errors, 1)
	assert.Equal(t, "not_found", result.Errors[0].Code)
	assert.Equal(t, originalPath, result.Errors[0].Path)
}

func TestParameterWriteValidationRejectsTranslatedReadOnlyPathUsingOriginalPath(t *testing.T) {
	writeModel := newParameterWriteModelForTest([]parammodel.ParamMapping{{
		StandardPath: "Device.FAP.Ipsec.{i}.TUNNEL_ENABLE",
		PrivatePath:  "Device.FAP.Ipsec.{i}.TUNNEL_CONFIG_TUNNELENABLE",
		EntryType:    "parameter",
		Access:       "READ_ONLY",
		DataType:     "BOOLEAN",
	}})

	const originalPath = "Device.FAP.Ipsec.1.TUNNEL_ENABLE"
	result := validateParameterWrites(writeModel, []ParameterValueItem{{
		Path:  originalPath,
		Value: "0",
		Type:  "BOOLEAN",
	}})

	require.Len(t, result.Errors, 1)
	assert.Equal(t, "not_writable", result.Errors[0].Code)
	assert.Equal(t, originalPath, result.Errors[0].Path)
}

func TestParameterWriteValidationDetectsTranslatedRebootRequirement(t *testing.T) {
	writeModel := newParameterWriteModelForTest([]parammodel.ParamMapping{{
		StandardPath:  "Device.FAP.Ipsec.{i}.TUNNEL_ENABLE",
		PrivatePath:   "Device.FAP.Ipsec.{i}.TUNNEL_CONFIG_TUNNELENABLE",
		EntryType:     "parameter",
		Access:        "READ_WRITE",
		DataType:      "BOOLEAN",
		ChangeApplies: "RebootRequired",
	}})

	result := validateParameterWrites(writeModel, []ParameterValueItem{{
		Path:  "Device.FAP.Ipsec.1.TUNNEL_ENABLE",
		Value: "1",
		Type:  "BOOLEAN",
	}})

	require.Empty(t, result.Errors)
	assert.True(t, result.RebootRequired)
}

func TestRebootTargetForIMSCorePaths(t *testing.T) {
	assert.Equal(t, 1, rebootTargetForPath("Device.ImsCore.BaseConfig.HOST_IPv4"))
	assert.Equal(t, 2, rebootTargetForPath("Device.ImsCore.BaseConfig.WEB_LOG_LEVEL"))
	assert.Equal(t, 3, rebootTargetForPath("Device.ImsCore.CoreInfoConfig.1.CORE_SYN_IP"))
}

func TestMergeRebootTarget(t *testing.T) {
	assert.Equal(t, 1, mergeRebootTarget(0, 1))
	assert.Equal(t, 3, mergeRebootTarget(1, 2))
	assert.Equal(t, 2, mergeRebootTarget(2, 2))
}
