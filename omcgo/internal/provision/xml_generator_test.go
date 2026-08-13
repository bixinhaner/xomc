package provision

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/quicksettings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGenerateAutoStartXMLMatchesCMCCContract(t *testing.T) {
	generatedAt := time.Date(2026, 7, 30, 8, 9, 10, 0, time.UTC)
	doc := AutoStartXMLDocument{
		NetworkType: "NR", Vendor: "BAICELLS", SerialNumber: "SN<&1",
		GeneratedAt: generatedAt, DataModelVersion: "v1.7",
		VendorSpecific: base64.StdEncoding.EncodeToString([]byte("<private/>")),
		Parameters: []ResolvedParameter{
			{ParameterID: "gNBName", TRPath: "Device.Services.Name", Value: "R&D"},
			{ParameterID: "PCI", TRPath: "Device.Services.Cell.1.PCI", Value: "123"},
		},
	}

	first, err := GenerateAutoStartXML(doc)
	require.NoError(t, err)
	second, err := GenerateAutoStartXML(doc)
	require.NoError(t, err)
	require.Equal(t, first, second)

	xmlText := string(first)
	require.Contains(t, xmlText, `<autoConfigFile generateTime="2026-07-30T08:09:10.000" networkType="NR" serialNumber="SN&lt;&amp;1" vendor="BAICELLS">`)
	require.Contains(t, xmlText, `<dataModelSpecific version="v1.7">`)
	require.Contains(t, xmlText, `<config name="Device.Services.Name" value="R&amp;D"></config>`)
	require.Contains(t, xmlText, `<vendorSpecific>PHByaXZhdGUvPg==</vendorSpecific>`)
	require.Less(t, strings.Index(xmlText, "Device.Services.Cell.1.PCI"), strings.Index(xmlText, "Device.Services.Name"))
	require.NotContains(t, xmlText, "SetParameterValues")

	var decoded any
	require.NoError(t, xml.Unmarshal(first, &decoded))
}

func TestGenerateAutoStartXMLUsesLegacyParamContractForLTEAndGSM(t *testing.T) {
	for _, networkType := range []string{"LTE", "GSM"} {
		t.Run(networkType, func(t *testing.T) {
			content, err := GenerateAutoStartXML(AutoStartXMLDocument{
				NetworkType: networkType,
				Parameters: []ResolvedParameter{
					{ParameterID: "logical-b", TRPath: "Device.Services.Cell.1.PCI", Value: "415"},
					{ParameterID: "logical-a", TRPath: "Device.ManagementServer.URL", Value: "http://acs?a=1&b=2"},
				},
			})
			require.NoError(t, err)
			assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>

<auto_start>
    <param>
        <name>Device.ManagementServer.URL</name>
        <value>http://acs?a=1&amp;b=2</value>
    </param>
    <param>
        <name>Device.Services.Cell.1.PCI</name>
        <value>415</value>
    </param>
</auto_start>`, string(content))
			assert.NotContains(t, string(content), "logical-a")
			assert.NotContains(t, string(content), "autoConfigFile")
			assert.NotContains(t, string(content), "<config")
		})
	}
}

func TestNetworkProfileIdentifiesGSM(t *testing.T) {
	networkType, version := networkProfile(&model.Device{Technology: model.TechGSM}, "BSC")
	assert.Equal(t, "GSM", networkType)
	assert.Equal(t, "v1.0", version)
}

func TestGenerateAutoStartXMLRejectsUnresolvedAndConflictingPaths(t *testing.T) {
	base := AutoStartXMLDocument{
		NetworkType: "NR", Vendor: "BAICELLS", SerialNumber: "SN1",
		GeneratedAt: time.Now(), DataModelVersion: "v1.7",
	}
	unresolved := base
	unresolved.Parameters = []ResolvedParameter{{ParameterID: "PCI", TRPath: "Device.Cell.{i}.PCI", Value: "1"}}
	_, err := GenerateAutoStartXML(unresolved)
	require.ErrorContains(t, err, "unresolved instance")

	conflict := base
	conflict.Parameters = []ResolvedParameter{
		{ParameterID: "PCI", TRPath: "Device.Cell.1.PCI", Value: "1"},
		{ParameterID: "PCI", TRPath: "Device.Cell.1.PCI", Value: "2"},
	}
	_, err = GenerateAutoStartXML(conflict)
	require.ErrorContains(t, err, "conflicting values")
}

func TestCompilePolicyParametersUsesQuickSettingsAndConcreteInstances(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		ID: uuid.New(), SelfConfigEnabled: true,
		Config: []byte(`{
			"dataModelVersion":"v1.7",
			"vendorSpecific":"<vendor><x>1</x></vendor>",
			"paramConfigList":[{
				"serialNumber":"SN-NR-1",
				"gnbName":"gNB-Beijing-001",
				"pci":321,
				"amfList":[{"amfIp":"10.10.10.1"},{"amfIp":"10.10.10.2"}]
			}]
		}`),
	}
	device := &model.Device{
		ID: uuid.New(), SerialNumber: "SN-NR-1", Technology: model.TechNR,
	}
	groups := []quicksettings.Group{
		{ID: "core", Params: []quicksettings.Param{
			{Name: "gNBName", StandardPath: "Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBName"},
			{Name: "AmfIP1", StandardPath: "Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP1"},
		}},
		{ID: "cell", Params: []quicksettings.Param{
			{Name: "PCI", StandardPath: "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.PhyCellID"},
		}},
	}

	got, err := CompilePolicyParameters(policy, device, "BaiBNQ", groups)
	require.NoError(t, err)
	assert.Equal(t, "NR", got.NetworkType)
	assert.Equal(t, "v1.7", got.DataModelVersion)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("<vendor><x>1</x></vendor>")), got.VendorSpecific)

	byPath := make(map[string]string)
	for _, param := range got.Parameters {
		byPath[param.TRPath] = param.Value
	}
	assert.Equal(t, "gNB-Beijing-001", byPath["Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBName"])
	assert.Equal(t, "321", byPath["Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID"])
	assert.Equal(t, "10.10.10.1", byPath["Device.Services.FAPService.1.FAPControl.NR.AMFPoolConfigParam.1.AmfIP1"])
	assert.Equal(t, "10.10.10.2", byPath["Device.Services.FAPService.1.FAPControl.NR.AMFPoolConfigParam.2.AmfIP1"])
}

func TestCompilePolicyParametersRejectsUnknownCustomPath(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"SN1",
			"customParams":[{"trPath":"Device.Unregistered.Secret","value":"x"}]
		}]}`),
	}
	_, err := CompilePolicyParameters(policy, &model.Device{
		SerialNumber: "SN1", Technology: model.TechNR,
	}, "BaiBNQ", []quicksettings.Group{{Params: []quicksettings.Param{{
		Name: "PCI", StandardPath: "Device.Cell.{i}.PCI",
	}}}})
	require.ErrorContains(t, err, "not registered in quick settings")
}

func TestCompilePolicyParametersIgnoresRetiredPlugAndPlayFields(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"SN1",
			"sheetParameters":{
				"DEVICE":[{"NTP Enable":true,"Time Zone Term":"CET-1"}],
				"INTERFACE":[{"OMC IP":"198.51.100.10"}]
			}
		}]}`),
	}
	got, err := CompilePolicyParameters(policy, &model.Device{
		SerialNumber: "SN1", Technology: model.TechNR,
	}, "BaiBNQ", []quicksettings.Group{{Params: []quicksettings.Param{
		{Name: "Enable", StandardPath: "Device.Time.Enable", Type: "boolean"},
		{Name: "LocalTimeZoneName", StandardPath: "Device.Time.LocalTimeZoneName"},
		{Name: "OMCIP", StandardPath: "Device.ManagementServer.OMCIP"},
	}}})
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	require.Equal(t, "Device.Time.Enable", got.Parameters[0].TRPath)
}

func TestCompilePolicyParametersAcceptsBaiBNQDefaultWorkbookDeviceSheet(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data",
		registry,
		zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"1202000534228JB0007",
			"sheetParameters":{"DEVICE":[{
				"Serial Number":"1202000534228JB0007",
				"NTP Enable":true,
				"Local Time Zone":"Asia/Shanghai",
				"Time Zone Term":"CET-1",
				"Periodic Inform Enable":true,
				"Periodic Inform Interval":30
			}]}
		}]}`),
	}
	mappings := []parammodel.ParamMapping{
		{StandardPath: "Device.ManagementServer.PeriodicInformEnable", EntryType: "parameter", Access: "READ_WRITE", DataType: "BOOLEAN", IsSupported: true},
		{StandardPath: "Device.ManagementServer.PeriodicInformInterval", EntryType: "parameter", Access: "READ_WRITE", DataType: "U_INT", IsSupported: true},
	}
	got, err := CompilePolicyParametersWithMappings(
		policy,
		&model.Device{SerialNumber: "1202000534228JB0007", Technology: model.TechNR},
		"BaiBNQ",
		registry.GetByParamModel("BaiBNQ"),
		mappings,
	)
	require.NoError(t, err)

	byPath := make(map[string]string)
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "1", byPath["Device.Time.Enable"])
	assert.Equal(t, "Asia/Shanghai", byPath["Device.Time.LocalTimeZoneName"])
	assert.Equal(t, "1", byPath["Device.ManagementServer.PeriodicInformEnable"])
	assert.Equal(t, "30", byPath["Device.ManagementServer.PeriodicInformInterval"])
}

func TestCompilePolicyParametersAcceptsBaiBNQDefaultWorkbookGNBLengthHeader(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data",
		registry,
		zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"1202000534228JB0007",
			"sheetParameters":{"CELL":[{
				"*Serial Number":"1202000534228JB0007",
				"*gNB Lenth":32
			}]}
		}]}`),
	}
	got, err := CompilePolicyParameters(
		policy,
		&model.Device{SerialNumber: "1202000534228JB0007", Technology: model.TechNR},
		"BaiBNQ",
		registry.GetByParamModel("BaiBNQ"),
	)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t,
		"Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBIdLength",
		got.Parameters[0].TRPath,
	)
	assert.Equal(t, "32", got.Parameters[0].Value)
}

func TestCompilePolicyParametersAcceptsBaiBNQTDDWorkbookFieldsWithProductMappings(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data",
		registry,
		zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	const tddBase = "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.TddULDLConfigurationCommon."
	mappings := make([]parammodel.ParamMapping, 0, 10)
	for _, pattern := range []string{"pattern1", "pattern2"} {
		for _, leaf := range []string{
			"DlULTransmissionPeriodicity",
			"NrofDownlinkSlots",
			"NrofDownlinkSymbols",
			"NrofUplinkSlots",
			"NrofUplinkSymbols",
		} {
			mappings = append(mappings, parammodel.ParamMapping{
				StandardPath: tddBase + pattern + "." + leaf,
				EntryType:    "parameter",
				Access:       "READ_WRITE",
				DataType:     "U_INT",
				IsSupported:  true,
			})
		}
	}

	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"1202000534228JB0007",
			"sheetParameters":{"CELL":[{
				"*Serial Number":"1202000534228JB0007",
				"DL ULTransmissionPeriodicity1":5,
				"Nrof DownlinkSlots1":7,
				"Nrof DownlinkSymbols1":6,
				"Nrof  UplinkSlots1":2,
				"Nrof  UplinkSymbols1":4,
				"DL ULTransmissionPeriodicity2":6,
				"Nrof  DownlinkSlots2":8,
				"Nrof  DownlinkSymbols2":5,
				"Nrof  UplinkSlots2":3,
				"Nrof  UplinkSymbols2":2
			}]}
		}]}`),
	}
	got, err := CompilePolicyParametersWithMappings(
		policy,
		&model.Device{SerialNumber: "1202000534228JB0007", Technology: model.TechNR},
		"BaiBNQ",
		registry.GetByParamModel("BaiBNQ"),
		mappings,
	)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 10)

	byPath := make(map[string]string, len(got.Parameters))
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "7", byPath["Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern1.NrofDownlinkSlots"])
	assert.Equal(t, "8", byPath["Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern2.NrofDownlinkSlots"])
}

func TestBuildParameterDefinitionsPrefersUniqueQuickSettingNamesOverMappingLeafCollisions(t *testing.T) {
	quickPaths := map[string]string{
		"gNBIdLength":         "Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBIdLength",
		"TAC":                 "Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}.TAC",
		"PLMNID":              "Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}.PLMNList.{i}.PLMNID",
		"SsbSubcarrierOffset": "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.SSB.SsbSubcarrierOffset",
	}
	groups := []quicksettings.Group{{ID: "colliding-fields"}}
	for name, path := range quickPaths {
		groups[0].Params = append(groups[0].Params, quicksettings.Param{Name: name, StandardPath: path})
	}
	mappings := []parammodel.ParamMapping{
		{StandardPath: quickPaths["gNBIdLength"], EntryType: "parameter", IsSupported: true},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.NRCell.{i}.gNBIdLength", EntryType: "parameter", IsSupported: true},
		{StandardPath: quickPaths["TAC"], EntryType: "parameter", IsSupported: true},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.NRCell.{i}.TAC", EntryType: "parameter", IsSupported: true},
		{StandardPath: quickPaths["PLMNID"], EntryType: "parameter", IsSupported: true},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.NRCell.{i}.PLMNID", EntryType: "parameter", IsSupported: true},
		{StandardPath: quickPaths["SsbSubcarrierOffset"], EntryType: "parameter", IsSupported: true},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.InterFreq.Carrier.{i}.SsbSubcarrierOffset", EntryType: "parameter", IsSupported: true},
	}

	_, aliases := buildParameterDefinitions(groups, mappings)
	for alias, name := range map[string]string{
		"gNB ID Length": "gNBIdLength",
		"TAC":           "TAC",
		"PLMN ID":       "PLMNID",
		"kSSB":          "SsbSubcarrierOffset",
	} {
		assert.Equal(t, quickPaths[name], aliases[normalizeParameterKey(alias)], alias)
	}
}

func TestBaiBNQWorkbookTemplateHeadersResolveAgainstProductionDefinitions(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data",
		registry,
		zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	type xmlMapping struct {
		StandardPath string `xml:"standardPath,attr"`
		Access       string `xml:"access,attr"`
		DataType     string `xml:"type,attr"`
		Supported    string `xml:"supported,attr"`
	}
	var document struct {
		Params []xmlMapping `xml:"parameters>param"`
	}
	raw, err := os.ReadFile("../../data/param-mappings/BaiBNQ.xml")
	require.NoError(t, err)
	require.NoError(t, xml.Unmarshal(raw, &document))
	mappings := make([]parammodel.ParamMapping, 0, len(document.Params))
	for _, item := range document.Params {
		mappings = append(mappings, parammodel.ParamMapping{
			StandardPath: item.StandardPath,
			EntryType:    "parameter",
			Access:       item.Access,
			DataType:     item.DataType,
			IsSupported:  !strings.EqualFold(item.Supported, "false"),
		})
	}
	_, aliases := buildParameterDefinitions(registry.GetByParamModel("BaiBNQ"), mappings)

	// Keep this contract aligned with the public gNB workbook in
	// omcmb/webcode/src/pages/device/PlugAndPlay/paramConfigTemplate.ts.
	templateHeaders := map[string][]string{
		"DEVICE": {
			"Serial Number", "URL", "Periodic Inform Enable", "Periodic Inform Time",
			"Periodic Inform Interval", "NTP Enable", "NTP Server1", "NTP Server2",
			"NTP Server3", "NTP Server4", "NTP Server5", "Local Time Zone",
			"PpsTimeMode",
		},
		"CELL": {
			"*Serial Number", "gNB Name", "*gNB ID", "*gNB Lenth", "*PCI",
			"SSB Frequency", "Freq BandIndicator", "NRARFCNDL", "NRARFCNUL",
			"DLBandwidth", "ULBandwidth", "Duplex Mode", "DLAntNum", "ULAntNum",
			"DL ULTransmissionPeriodicity1", "Nrof DownlinkSlots1",
			"Nrof DownlinkSymbols1", "Nrof  UplinkSlots1", "Nrof  UplinkSymbols1",
			"DL ULTransmissionPeriodicity2", "Nrof  DownlinkSlots2",
			"Nrof  DownlinkSymbols2", "Nrof  UplinkSlots2", "Nrof  UplinkSymbols2",
			"Prach RootSequenceIndex", "Prach RootSequenceValue",
			"SubcarrierSpacing(UL)", "SubcarrierSpacing(DL)", "PowerModify",
			"OffsetToPointA", "SsbSubcarrierOffset",
		},
		"PLMN": {
			"Serial Number", "*NCI", "*TAC", "*RANAC", "*PLMN ID", "*PRIMARY",
			"SD", "SD Value", "AMF IP:DEFAULT", "NguBindInterface",
		},
		"INTERFACE": {
			"Serial Number", "Interface Name", "Address Type", "IP Address",
			"Subnet Mask", "Prefix Length", "Gateway", "Bear Type", "Vlan Name",
			"Vlan ID",
		},
		"IPSEC": {
			"Serial Number", "TUNNEL_ENABLE", "TUNNEL_GATEWAY", "LEFT_AUTH",
			"RIGHT_AUTH", "RIGHT_SUBNET", "LEFT_IDENTIFIER", "RIGHT_IDENTIFIER",
			"LEFTSOURCEIP", "LEFT_SUBNET", "FRAGMENTATION", "IKE_ENCRYPTION",
			"IKE_DH_GROUP", "IKE_AUTHENTICATION", "ESP_ENCRYPTION", "ESP_DH_GROUP",
			"ESP_AUTHENTICATION", "KEYLIFE", "IKELIFETIME", "REKEYMARGIN",
			"DPDACTION", "DPDDELAY", "LEFT_INTERFACE", "FORCEENCAPS",
		},
	}
	for sheet, headers := range templateHeaders {
		for _, header := range headers {
			normalized := normalizeParameterKey(header)
			if _, ignored := ignoredPolicyFields[normalized]; ignored {
				continue
			}
			if _, optional := optionalPlanningFields[normalized]; optional {
				continue
			}
			assert.Contains(t, aliases, normalized, "%s.%s", sheet, header)
		}
	}
}

func TestCompilePolicyParametersAcceptsCurrentBaiBNQWorkbookWithProductMappings(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data",
		registry,
		zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	type xmlMapping struct {
		StandardPath string `xml:"standardPath,attr"`
		Access       string `xml:"access,attr"`
		DataType     string `xml:"type,attr"`
		Supported    string `xml:"supported,attr"`
	}
	var document struct {
		Params []xmlMapping `xml:"parameters>param"`
	}
	raw, err := os.ReadFile("../../data/param-mappings/BaiBNQ.xml")
	require.NoError(t, err)
	require.NoError(t, xml.Unmarshal(raw, &document))
	mappings := make([]parammodel.ParamMapping, 0, len(document.Params))
	for _, item := range document.Params {
		mappings = append(mappings, parammodel.ParamMapping{
			StandardPath: item.StandardPath,
			EntryType:    "parameter",
			Access:       item.Access,
			DataType:     item.DataType,
			IsSupported:  !strings.EqualFold(item.Supported, "false"),
		})
	}

	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{
			"serialNumber":"1202000534228JB0007",
			"sheetParameters":{
				"DEVICE":[{"Periodic Inform Enable":true,"Periodic Inform Interval":30,"Time Zone Term":"CET-1"}],
				"CELL":[{
					"*PCI":"4556","*gNB ID":"21","DLAntNum":"4","ULAntNum":"4",
					"gNB Name":"212","NRARFCNDL":"687676","NRARFCNUL":"67676776",
					"*gNB Lenth":"23","DLBandwidth":"133","ULBandwidth":"133",
					"PowerModify":"23","OffsetToPointA":"3434","SsbSubcarrierOffset":"3",
					"SubcarrierSpacing(DL)":"1","SubcarrierSpacing(UL)":"1",
					"DL ULTransmissionPeriodicity1":"1","Nrof DownlinkSlots1":"1",
					"Nrof DownlinkSymbols1":"1","Nrof  UplinkSlots1":"1","Nrof  UplinkSymbols1":"1",
					"DL ULTransmissionPeriodicity2":"1","Nrof  DownlinkSlots2":"1",
					"Nrof  DownlinkSymbols2":"1","Nrof  UplinkSlots2":"1","Nrof  UplinkSymbols2":"1"
				}],
				"PLMN":[{
					"*NCI":"34354656","*TAC":"3","*PLMN ID":"46000",
					"AMF IP:DEFAULT":"172.17.1.23","NguBindInterface":"172.17.1.23"
				}]
			}
		}]
	}`)}
	got, err := CompilePolicyParametersWithMappings(
		policy,
		&model.Device{SerialNumber: "1202000534228JB0007", Technology: model.TechNR},
		"BaiBNQ",
		registry.GetByParamModel("BaiBNQ"),
		mappings,
	)
	require.NoError(t, err)

	byPath := make(map[string]string, len(got.Parameters))
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "172.17.1.23", byPath["Device.Services.FAPService.1.FAPControl.NR.AMFPoolConfigParam.1.AmfIP1"])
	assert.Equal(t, "3", byPath["Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.TAC"])
	assert.Equal(t, "46000", byPath["Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.PLMNList.1.PLMNID"])
}

func TestCompilePolicyParametersUsesProductEnumValuesForLegacyLTEBandwidth(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data", registry, zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	values := "25,50,75,100"
	labels := "CELL_BW_25(5M),CELL_BW_50(10M),CELL_BW_75(15M),CELL_BW_100(20M)"
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{"serialNumber":"BLQ-001","bandWidth":"n50"}]
	}`)}
	got, err := CompilePolicyParametersWithMappings(
		policy,
		&model.Device{SerialNumber: "BLQ-001", Technology: model.TechLTE},
		"BLQ",
		registry.GetByParamModel("BLQ"),
		[]parammodel.ParamMapping{{
			StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth",
			EntryType:    "parameter", Access: "READ_WRITE", DataType: "INT", IsSupported: true,
			EnumValues: &values, EnumLabels: &labels,
		}},
	)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "50", got.Parameters[0].Value)
}

func TestCompilePolicyParametersAcceptsMLNDefaultWorkbookSynchronizationMode(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data",
		registry,
		zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"120200055922C8B0068",
			"sheetParameters":{"1588_CONFIGURATION":[{
				"*SERIAL_NUMBER":"120200055922C8B0068",
				"*SYNCHRONIZATION_MODE":"GNSS"
			}]}
		}]}`),
	}
	got, err := CompilePolicyParameters(
		policy,
		&model.Device{SerialNumber: "120200055922C8B0068", Technology: model.TechLTE},
		"MLN",
		registry.GetByParamModel("MLN"),
	)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "Device.ManagementServer.tfcsManagerPrimsrc", got.Parameters[0].TRPath)
	assert.Equal(t, "3", got.Parameters[0].Value)
}

func TestCompilePolicyParametersSerializesMLNServingPLMNList(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data",
		registry,
		zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"120200055922C8B0068",
			"plmnConfigList":[{"plmnId":"46000"},{"plmnId":"46001"}]
		}]}`),
	}
	got, err := CompilePolicyParameters(
		policy,
		&model.Device{SerialNumber: "120200055922C8B0068", Technology: model.TechLTE},
		"MLN",
		registry.GetByParamModel("MLN"),
	)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList", got.Parameters[0].TRPath)
	assert.Equal(t, "46000,46001", got.Parameters[0].Value)
}

func TestCompilePolicyParametersExpandsMultiInstanceServingPLMNList(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"LTE-PLMN-001",
			"plmnConfigList":[{"plmnId":"46000"},{"plmnId":"46001"}]
		}]}`),
	}
	groups := []quicksettings.Group{{
		ObjectPath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.",
		Params:     []quicksettings.Param{{Name: "PLMNID", Leaf: "PLMNID"}},
	}}
	got, err := CompilePolicyParameters(
		policy,
		&model.Device{SerialNumber: "LTE-PLMN-001", Technology: model.TechLTE},
		"BLQ",
		groups,
	)
	require.NoError(t, err)

	byPath := make(map[string]string)
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "46000", byPath["Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID"])
	assert.Equal(t, "46001", byPath["Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.2.PLMNID"])
}

func TestCompilePolicyParametersSkipsUnsupportedMLN1588PlanningFields(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data",
		registry,
		zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"120200055922C8B0068",
			"sheetParameters":{"1588_CONFIGURATION":[{
				"*SERIAL_NUMBER":"120200055922C8B0068",
				"*1588_ENABLE":"1",
				"*SYNCHRONIZATION_MODE":"GNSS",
				"*SYNC_MODE":"TIME",
				"*MODE_SWITCH":"multicast",
				"*DOMAIN":"0",
				"*SYNC_INTERVAL":"-4",
				"*DELAY_INTERVAL":"0",
				"*ASYMMETRY":"0",
				"*STARTUP_TIME":"0",
				"UNICAST_SERVER_IP_ADDRESS":"192.0.2.10"
			}]}
		}]}`),
	}
	got, err := CompilePolicyParameters(
		policy,
		&model.Device{SerialNumber: "120200055922C8B0068", Technology: model.TechLTE},
		"MLN",
		registry.GetByParamModel("MLN"),
	)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "Device.ManagementServer.tfcsManagerPrimsrc", got.Parameters[0].TRPath)
	assert.Equal(t, "3", got.Parameters[0].Value)
}

func TestCompilePolicyParametersCompilesSupported1588PlanningFields(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"LTE-PTP-001",
			"sheetParameters":{"1588_CONFIGURATION":[{
				"*SERIAL_NUMBER":"LTE-PTP-001",
				"*1588_ENABLE":"1",
				"*SYNC_MODE":"TIME",
				"*DOMAIN":"24",
				"UNICAST_SERVER_IP_ADDRESS":"192.0.2.10"
			}]}
		}]}`),
	}
	mappings := []parammodel.ParamMapping{
		{StandardPath: "Device.DeviceInfo.X_COM_1588SyncEnable", EntryType: "parameter", Access: "READ_WRITE", DataType: "BOOLEAN", IsSupported: true},
		{StandardPath: "Device.DeviceInfo.X_COM_PTP1588syncsMode", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
		{StandardPath: "Device.DeviceInfo.X_COM_PTP1588DomainNum", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
		{StandardPath: "Device.DeviceInfo.X_COM_PTP1588UnicastAddr", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
	}

	got, err := CompilePolicyParametersWithMappings(
		policy,
		&model.Device{SerialNumber: "LTE-PTP-001", Technology: model.TechLTE},
		"MLQ",
		nil,
		mappings,
	)
	require.NoError(t, err)

	byPath := make(map[string]string)
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "1", byPath["Device.DeviceInfo.X_COM_1588SyncEnable"])
	assert.Equal(t, "TIME", byPath["Device.DeviceInfo.X_COM_PTP1588syncsMode"])
	assert.Equal(t, "24", byPath["Device.DeviceInfo.X_COM_PTP1588DomainNum"])
	assert.Equal(t, "192.0.2.10", byPath["Device.DeviceInfo.X_COM_PTP1588UnicastAddr"])
}

func TestCompilePolicyParametersFallsBackToProductParameterMappings(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"SN-PRODUCT-MAPPING",
			"sheetParameters":{"DEVICE":[{
				"Serial Number":"SN-PRODUCT-MAPPING",
				"NTP Enable":true,
				"Local Time Zone":"Asia/Shanghai",
				"Time Zone Term":"CET-1",
				"Periodic Inform Enable":true,
				"Periodic Inform Interval":30
			}]}
		}]}`),
	}
	mappings := []parammodel.ParamMapping{
		{StandardPath: "Device.Time.Enable", EntryType: "parameter", Access: "READ_WRITE", DataType: "BOOLEAN", IsSupported: true},
		{StandardPath: "Device.Time.LocalTimeZoneName", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
		{StandardPath: "Device.ManagementServer.PeriodicInformEnable", EntryType: "parameter", Access: "READ_WRITE", DataType: "BOOLEAN", IsSupported: true},
		{StandardPath: "Device.ManagementServer.PeriodicInformInterval", EntryType: "parameter", Access: "READ_WRITE", DataType: "U_INT", IsSupported: true},
	}

	got, err := CompilePolicyParametersWithMappings(
		policy,
		&model.Device{SerialNumber: "SN-PRODUCT-MAPPING", Technology: model.TechNR},
		"BaiBNQ",
		nil,
		mappings,
	)
	require.NoError(t, err)

	byPath := make(map[string]string)
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "1", byPath["Device.Time.Enable"])
	assert.Equal(t, "Asia/Shanghai", byPath["Device.Time.LocalTimeZoneName"])
	assert.Equal(t, "1", byPath["Device.ManagementServer.PeriodicInformEnable"])
	assert.Equal(t, "30", byPath["Device.ManagementServer.PeriodicInformInterval"])
}

func TestCompilePolicyParametersUsesSerialNumberOnlyForDeviceMatching(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"SN-PRODUCT-MAPPING",
			"sheetParameters":{"DEVICE":[{
				"Serial Number":"SN-PRODUCT-MAPPING",
				"NTP Enable":true
			}]},
			"customParams":[{
				"trPath":"Device.DeviceInfo.MU.1.SerialNumber",
				"value":"MUST-NOT-BE-DOWNLOADED"
			}]
		}]}`),
	}
	mappings := []parammodel.ParamMapping{
		{
			StandardPath: "Device.DeviceInfo.MU.{i}.SerialNumber",
			EntryType:    "parameter", Access: "READ_ONLY", DataType: "STRING", IsSupported: true,
		},
		{
			StandardPath: "Device.Time.Enable",
			EntryType:    "parameter", Access: "READ_WRITE", DataType: "BOOLEAN", IsSupported: true,
		},
	}

	got, err := CompilePolicyParametersWithMappings(
		policy,
		&model.Device{SerialNumber: "SN-PRODUCT-MAPPING", Technology: model.TechNR},
		"BaiBNQ",
		nil,
		mappings,
	)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "Device.Time.Enable", got.Parameters[0].TRPath)
	assert.Equal(t, "1", got.Parameters[0].Value)
}

func TestCompilePolicyParametersRejectsUnknownImportedParameter(t *testing.T) {
	policy := &PlugAndPlayPolicy{
		SelfConfigEnabled: true,
		Config: []byte(`{"paramConfigList":[{
			"serialNumber":"SN-UNKNOWN-IMPORT",
			"sheetParameters":{"DEVICE":[{
				"Serial Number":"SN-UNKNOWN-IMPORT",
				"Unsupported Template Field":"value"
			}]}
		}]}`),
	}

	_, err := CompilePolicyParameters(policy, &model.Device{
		SerialNumber: "SN-UNKNOWN-IMPORT", Technology: model.TechNR,
	}, "BaiBNQ", []quicksettings.Group{{Params: []quicksettings.Param{{
		Name: "PCI", StandardPath: "Device.Cell.{i}.PCI",
	}}}})
	require.ErrorContains(t, err, "Unsupported Template Field")
	require.ErrorContains(t, err, "is not registered")
}

func TestCompilePolicyParametersAcceptsNewNRPlanningFields(t *testing.T) {
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{"serialNumber":"NR-PLAN-001","sheetParameters":{
			"CELL":[{"PowerModify":"20"}],
			"PLMN":[{"NguBindInterface":"192.0.2.10"}],
			"IPSEC":[{"LEFT_AUTH":"psk","FORCEENCAPS":"yes"}]
		}}]}`)}
	mappings := []parammodel.ParamMapping{
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PowerModify", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
		{StandardPath: "Device.FAP.NguIpBind{i}.BindInterface", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
		{StandardPath: "Device.FAP.Ipsec.{i}.TUNNEL_LEFT_AUTH", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
		{StandardPath: "Device.FAP.Ipsec.{i}.FORCEENCAPS", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
	}
	got, err := CompilePolicyParametersWithMappings(policy,
		&model.Device{SerialNumber: "NR-PLAN-001", Technology: model.TechNR},
		"BaiBNQ", nil, mappings)
	require.NoError(t, err)

	byPath := make(map[string]string)
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "20", byPath["Device.Services.FAPService.1.CellConfig.1.NR.RAN.PowerModify"])
	assert.Equal(t, "192.0.2.10", byPath["Device.FAP.NguIpBind1.BindInterface"])
	assert.Equal(t, "psk", byPath["Device.FAP.Ipsec.1.TUNNEL_LEFT_AUTH"])
	assert.Equal(t, "yes", byPath["Device.FAP.Ipsec.1.FORCEENCAPS"])
}

func TestCompilePolicyParametersCombinesGSMIPAAndUnitID(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data", registry, zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{"serialNumber":"GSM-PLAN-001","sheetParameters":{"GSM":[{
			"IPA":"6969","Unit ID":"36","Remote IP":"198.51.100.10","Bind IP":"192.0.2.20",
			"WAN IP":"192.0.2.20","Synchronization":"GNSS","OMC":"192.0.2.100"
		}]}}]}`)}
	got, err := CompilePolicyParameters(policy,
		&model.Device{SerialNumber: "GSM-PLAN-001", Technology: model.TechGSM},
		"BM", registry.GetByParamModel("BM"))
	require.NoError(t, err)

	byPath := make(map[string]string)
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "6969-36", byPath["Device.Services.GsmBTSCellDT.1.IPAUnitID"])
	assert.Equal(t, "198.51.100.10", byPath["Device.Services.GsmBTSCellDT.1.OMLRemoteIP"])
	assert.Equal(t, "192.0.2.20", byPath["Device.Services.GsmBTSCellDT.1.GsmBtsBindMib"])
	assert.Equal(t, "1", byPath["Device.FAP.Synchronization.PpsTimeMode"])
}

func TestCompilePolicyParametersUsesBSCSpecificGSMPaths(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data", registry, zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{"serialNumber":"BSC-PLAN-001","sheetParameters":{"GSM":[{
			"IPA":"6969","Unit ID":"36","Remote IP":"198.51.100.10"
		}]}}]}`)}
	got, err := CompilePolicyParameters(policy,
		&model.Device{SerialNumber: "BSC-PLAN-001", Technology: model.TechGSM},
		"BSC", registry.GetByParamModel("BSC"))
	require.NoError(t, err)

	byPath := make(map[string]string)
	for _, parameter := range got.Parameters {
		byPath[parameter.TRPath] = parameter.Value
	}
	assert.Equal(t, "6969-36", byPath["DeviceGSM.Bts.1.IpaUnitId"])
	assert.Equal(t, "198.51.100.10", byPath["DeviceGSM.Bts.1.IpaRslIp"])
}

func TestCompilePolicyParametersUsesWorkbookCustomMapping(t *testing.T) {
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{
			"serialNumber":"NR-CUSTOM-001",
			"sheetParameters":{"DEVICE":[{"Custom Header":"custom-value"}]},
			"workbookMappings":[{
				"sheet":"DEVICE","header":"Custom Header","trPath":"Device.Custom.1.Value"
			}]
		}]
	}`)}
	mappings := []parammodel.ParamMapping{{
		StandardPath: "Device.Custom.{i}.Value", EntryType: "parameter",
		Access: "READ_WRITE", DataType: "STRING", IsSupported: true,
	}}
	got, err := CompilePolicyParametersWithMappings(policy,
		&model.Device{SerialNumber: "NR-CUSTOM-001", Technology: model.TechNR},
		"BaiBNQ", nil, mappings)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "Device.Custom.1.Value", got.Parameters[0].TRPath)
	assert.Equal(t, "custom-value", got.Parameters[0].Value)
}

func TestCompilePolicyParametersAllowsConcreteWorkbookPathForRegisteredAlias(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data", registry, zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{
			"serialNumber":"NR-WORKBOOK-ALIAS-001",
			"sheetParameters":{"CELL":[{"SSB Frequency":"633984"}]},
			"workbookMappings":[{
				"sheet":"CELL","header":"SSB Frequency",
				"trPath":"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.SsbFrequency"
			}]
		}]
	}`)}
	got, err := CompilePolicyParameters(policy,
		&model.Device{SerialNumber: "NR-WORKBOOK-ALIAS-001", Technology: model.TechNR},
		"BaiBNQ", registry.GetByParamModel("BaiBNQ"))
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.SsbFrequency", got.Parameters[0].TRPath)
	assert.Equal(t, "633984", got.Parameters[0].Value)
}

func TestWorkbookMappingsAcceptConcretePathsForEveryQuickSettingAlias(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data", registry, zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	for _, modelName := range registry.KnownParamModels() {
		groups := registry.GetByParamModel(modelName)
		for _, group := range groups {
			for _, param := range group.Params {
				template := param.StandardPath
				if template == "" && group.ObjectPath != "" && param.Leaf != "" {
					template = group.ObjectPath + param.Leaf
				}
				if template == "" {
					continue
				}
				header := param.TitleEn
				if header == "" {
					header = param.TitleZh
				}
				if header == "" {
					header = param.Name
				}
				concretePath := strings.ReplaceAll(template, "{i}", "1")
				t.Run(modelName+"/"+group.ID+"/"+param.Name, func(t *testing.T) {
					definitions, aliases := buildParameterDefinitions(groups, nil)
					row := map[string]any{
						"sheetParameters": map[string]any{"TEST": []any{map[string]any{header: "value"}}},
						"workbookMappings": []any{map[string]any{
							"sheet": "TEST", "header": header, "trPath": concretePath,
						}},
					}
					mappedAliases, mapErr := applyWorkbookParameterMappings(row, definitions, aliases)
					require.NoError(t, mapErr)
					definitionKey := mappedAliases[normalizeParameterKey("TEST."+header)]
					require.NotEmpty(t, definitionKey)
					assert.Equal(t, concretePath, definitions[definitionKey].Template)
				})
			}
		}
	}
}

func TestWorkbookMappingsAllowSameHeaderInDifferentSheets(t *testing.T) {
	definitions := map[string]parameterDefinition{
		"Device.Custom.1.Value": {ID: "first", Template: "Device.Custom.1.Value"},
		"Device.Other.1.Value":  {ID: "second", Template: "Device.Other.1.Value"},
	}
	row := map[string]any{
		"sheetParameters": map[string]any{
			"FIRST":  []any{map[string]any{"Enable": "1"}},
			"SECOND": []any{map[string]any{"Enable": "0"}},
		},
		"workbookMappings": []any{
			map[string]any{"sheet": "FIRST", "header": "Enable", "trPath": "Device.Custom.1.Value"},
			map[string]any{"sheet": "SECOND", "header": "Enable", "trPath": "Device.Other.1.Value"},
		},
	}
	aliases, err := applyWorkbookParameterMappings(row, definitions, nil)
	require.NoError(t, err)
	assert.Equal(t, "Device.Custom.1.Value", aliases[normalizeParameterKey("FIRST.Enable")])
	assert.Equal(t, "Device.Other.1.Value", aliases[normalizeParameterKey("SECOND.Enable")])
}

func TestStrictWorkbookMappingsAllowRegisteredLegacyAliasFallback(t *testing.T) {
	registry := quicksettings.NewRegistry()
	loader := quicksettings.NewLoader(
		appconfig.QuickSettingsLoaderConfig{Directory: "quicksettings"},
		"../../data", registry, zap.NewNop(),
	)
	_, err := loader.LoadOnce(context.Background())
	require.NoError(t, err)

	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{
			"serialNumber":"NR-LEGACY-ALIAS-001",
			"sheetParameters":{"CELL":[{"*gNB ID":"12"}]},
			"workbookMappings":[{
				"sheet":"DEVICE","header":"gNB ID",
				"trPath":"Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBId"
			}]
		}]
	}`)}
	got, err := CompilePolicyParameters(policy,
		&model.Device{SerialNumber: "NR-LEGACY-ALIAS-001", Technology: model.TechNR},
		"BaiBNQ", registry.GetByParamModel("BaiBNQ"))
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBId", got.Parameters[0].TRPath)
	assert.Equal(t, "12", got.Parameters[0].Value)
}

func TestCompilePolicyParametersRejectsConflictingMappingsForSameSheetColumn(t *testing.T) {
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{
			"serialNumber":"NR-WORKBOOK-CONFLICT-001",
			"sheetParameters":{"DEVICE":[{"Mapped":"value"}]},
			"workbookMappings":[
				{"sheet":"DEVICE","header":"Mapped","trPath":"Device.Custom.1.Value"},
				{"sheet":"DEVICE","header":"Mapped","trPath":"Device.Other.1.Value"}
			]
		}]
	}`)}
	mappings := []parammodel.ParamMapping{
		{StandardPath: "Device.Custom.{i}.Value", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
		{StandardPath: "Device.Other.{i}.Value", EntryType: "parameter", Access: "READ_WRITE", DataType: "STRING", IsSupported: true},
	}
	_, err := CompilePolicyParametersWithMappings(policy,
		&model.Device{SerialNumber: "NR-WORKBOOK-CONFLICT-001", Technology: model.TechNR},
		"BaiBNQ", nil, mappings)
	require.ErrorContains(t, err, "workbook parameter column Mapped maps to conflicting TRPaths")
}

func TestCompilePolicyParametersRejectsUnmappedWorkbookColumn(t *testing.T) {
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{
			"serialNumber":"NR-CUSTOM-002",
			"sheetParameters":{"DEVICE":[{"Mapped":"ok","Unmapped":"bad"}]},
			"workbookMappings":[{
				"sheet":"DEVICE","header":"Mapped","trPath":"Device.Custom.1.Value"
			}]
		}]
	}`)}
	mappings := []parammodel.ParamMapping{{
		StandardPath: "Device.Custom.{i}.Value", EntryType: "parameter",
		Access: "READ_WRITE", DataType: "STRING", IsSupported: true,
	}}
	_, err := CompilePolicyParametersWithMappings(policy,
		&model.Device{SerialNumber: "NR-CUSTOM-002", Technology: model.TechNR},
		"BaiBNQ", nil, mappings)
	require.ErrorContains(t, err, "workbook parameter column has no mapping: DEVICE.Unmapped")
}

func TestCompilePolicyParametersIgnoresBlankWorkbookParameterMapping(t *testing.T) {
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: []byte(`{
		"paramConfigList":[{
			"serialNumber":"NR-CUSTOM-003",
			"sheetParameters":{"DEVICE":[{"Mapped":"ok","Optional Unsupported":""}]},
			"workbookMappings":[
				{"sheet":"DEVICE","header":"Mapped","trPath":"Device.Custom.1.Value"},
				{"sheet":"DEVICE","header":"Optional Unsupported","trPath":"Device.Unsupported.1.Value"}
			]
		}]
	}`)}
	mappings := []parammodel.ParamMapping{{
		StandardPath: "Device.Custom.{i}.Value", EntryType: "parameter",
		Access: "READ_WRITE", DataType: "STRING", IsSupported: true,
	}}
	got, err := CompilePolicyParametersWithMappings(policy,
		&model.Device{SerialNumber: "NR-CUSTOM-003", Technology: model.TechNR},
		"BaiBNQ", nil, mappings)
	require.NoError(t, err)
	require.Len(t, got.Parameters, 1)
	assert.Equal(t, "Device.Custom.1.Value", got.Parameters[0].TRPath)
	assert.Equal(t, "ok", got.Parameters[0].Value)
}
