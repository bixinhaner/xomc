package provision

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func TestMaterializePolicyParametersAllocatesByStableFleetOrder(t *testing.T) {
	created := time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC)
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,"freqBandIndicator":78,
			"gnbIdAllocation":{"start":100,"end":110,"step":1,"reserved":[{"start":101,"end":101}]},
			"pciAllocation":{"start":10,"end":20,"step":2,"reserved":[{"start":12,"end":12}]}
		},
		"paramConfigList":[{"serialNumber":"SN-OVERRIDE","gnbId":"100","pci":10}]
	}`)}
	fleet := []model.Device{
		{ID: uuid.New(), SerialNumber: "SN-NEW", CreatedAt: created.Add(time.Minute)},
		{ID: uuid.New(), SerialNumber: "SN-OVERRIDE", CreatedAt: created},
	}

	materialized, err := materializePolicyParameters(policy, &fleet[0], fleet)
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	rows := mapSlice(root["paramConfigList"])
	require.Len(t, rows, 1)
	require.Equal(t, "SN-NEW", rows[0]["serialNumber"])
	require.Equal(t, "102", rows[0]["gnbId"])
	require.Equal(t, float64(14), rows[0]["pci"])
	require.Equal(t, float64(78), rows[0]["freqBandIndicator"])
}

func TestMaterializePolicyParametersMergesDeviceOverrideOnCommonValues(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,"freqBandIndicator":78,"dlbandwidth":"100MHz",
			"gnbIdAllocation":{"start":1,"end":10,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1}
		},
		"paramConfigList":[{"serialNumber":"SN-1","dlbandwidth":"80MHz","tac":3}]
	}`)}

	materialized, err := materializePolicyParameters(policy, &device, []model.Device{device})
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	row := mapSlice(root["paramConfigList"])[0]
	require.Equal(t, float64(78), row["freqBandIndicator"])
	require.Equal(t, "80MHz", row["dlbandwidth"])
	require.Equal(t, float64(3), row["tac"])
	require.Equal(t, "1", row["gnbId"])
	require.Equal(t, float64(0), row["pci"])
}

func TestMaterializePolicyParametersTreatsLegacyCommonConfigAsCommonMode(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,"freqBandIndicator":78,
			"gnbIdAllocation":{"start":1,"end":10,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1}
		},
		"paramConfigList":[{"serialNumber":"SN-1","dlbandwidth":"80MHz"}]
	}`)}

	materialized, err := materializePolicyParameters(policy, &device, []model.Device{device})
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	row := mapSlice(root["paramConfigList"])[0]
	require.Equal(t, float64(78), row["freqBandIndicator"])
	require.Equal(t, "80MHz", row["dlbandwidth"])
}

func TestMaterializePolicyParametersSpecifiedModeDoesNotMergeCommonValues(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"specified",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,"freqBandIndicator":78,
			"gnbIdAllocation":{"start":1,"end":10,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1}
		},
		"paramConfigList":[{"serialNumber":"SN-1","dlbandwidth":"80MHz","tac":3}]
	}`)}

	materialized, err := materializePolicyParameters(policy, &device, []model.Device{device})
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	row := mapSlice(root["paramConfigList"])[0]
	require.NotContains(t, row, "freqBandIndicator")
	require.Equal(t, "80MHz", row["dlbandwidth"])
	require.Equal(t, float64(3), row["tac"])
}

func TestMaterializePolicyParametersPreservesCommonRadioInstanceList(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-MULTI-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":10,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1},
			"sheetParameters":{"CELL":[
				{"Cell Index":1,"*PCI":"10","SSB Frequency":"633984"},
				{"Cell Index":3,"*PCI":"12","SSB Frequency":"634560"}
			]}
		}
	}`)}

	materialized, err := materializePolicyParameters(policy, &device, []model.Device{device})
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	row := mapSlice(root["paramConfigList"])[0]
	cells := mapSlice(mapValue(row["sheetParameters"])["CELL"])
	require.Len(t, cells, 2)
	require.Equal(t, float64(1), cells[0]["Cell Index"])
	require.Equal(t, "633984", cells[0]["SSB Frequency"])
	require.Equal(t, float64(3), cells[1]["Cell Index"])
	require.Equal(t, "634560", cells[1]["SSB Frequency"])
}

func TestMaterializePolicyParametersDeviceRadioInstanceListOverridesCommonList(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-MULTI-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"eNB",
			"sheetParameters":{"CELL":[
				{"*CELL_NUMBER":1,"*PCI":"10"},
				{"*CELL_NUMBER":2,"*PCI":"11"}
			]}
		},
		"paramConfigList":[{
			"serialNumber":"SN-MULTI-1",
			"sheetParameters":{"CELL":[{"*CELL_NUMBER":4,"*PCI":"20"}]}
		}]
	}`)}

	materialized, err := materializePolicyParameters(policy, &device, []model.Device{device})
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	cells := mapSlice(mapValue(mapSlice(root["paramConfigList"])[0]["sheetParameters"])["CELL"])
	require.Len(t, cells, 1)
	require.Equal(t, float64(4), cells[0]["*CELL_NUMBER"])
	require.Equal(t, "20", cells[0]["*PCI"])
}

func TestMaterializePolicyParametersExpandsCommonNetworkInstancesInListOrder(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":10,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1},
			"networkInterfaces":[{
				"name":"eth","ipv4Addresses":[
					{"addressingType":"Static","ipAddress":"172.17.1.14","subnetMask":"255.255.255.0","portType":"Other"},
					{"addressingType":"DHCP"}
				],
				"vlans":[{"name":"wan100","id":"100","enable":"1","ipv6Addresses":[{"origin":"Static","ipAddress":"2001:db8::1","prefixLength":"64"}]}]
			}],
			"customParams":[{"trPath":"Device.Ethernet.Interface.1.Name","value":"override-eth"}]
		}
	}`)}

	materialized, err := materializePolicyParameters(policy, &device, []model.Device{device})
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	row := mapSlice(root["paramConfigList"])[0]
	require.NotContains(t, row, "networkInterfaces")
	values := make(map[string]string)
	for _, item := range mapSlice(row["customParams"]) {
		values[valueString(item["trPath"])] = valueString(item["value"])
	}
	require.Equal(t, "override-eth", values["Device.Ethernet.Interface.1.Name"])
	require.Equal(t, "172.17.1.14", values["Device.Ethernet.Interface.1.IPv4Address.1.IPAddress"])
	require.Equal(t, "DHCP", values["Device.Ethernet.Interface.1.IPv4Address.2.AddressingType"])
	require.Equal(t, "wan100", values["Device.Ethernet.Interface.1.VlanInterface.1.Name"])
	require.Equal(t, "2001:db8::1", values["Device.Ethernet.Interface.1.VlanInterface.1.IPv6Address.1.IPAddress"])
}

func TestMaterializePolicyParametersDoesNotTurnMappedWorkbookValuesIntoCustomParameters(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-MAPPED-1"}
	const path = "Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.SsbFrequency"
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":10,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1}
		},
		"paramConfigList":[{
			"serialNumber":"SN-MAPPED-1",
			"sheetParameters":{"CELL":[{"SSB Frequency":"2324"}]},
			"workbookMappings":[{
				"sheet":"CELL","header":"SSB Frequency",
				"trPath":"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.SsbFrequency"
			}],
			"networkParameterValues":{
				"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.SsbFrequency":"23244232"
			}
		}]
	}`)}

	materialized, err := materializePolicyParameters(policy, &device, []model.Device{device})
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	row := mapSlice(root["paramConfigList"])[0]
	for _, item := range mapSlice(row["customParams"]) {
		require.NotEqual(t, path, valueString(item["trPath"]))
	}
}

func TestMaterializePolicyParametersRemovesFieldsExcludedFromCommonConfiguration(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":10,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1},
			"sheetParameters":{
				"DEVICE":[{"URL":"http://acs.example.com","Periodic Inform Interval":300,"Time Zone Term":"CET-1"}],
				"INTERFACE":[{"Interface Name":"eth0","Address Type":"IPv4","IP Address":"10.0.0.2","Vlan Name":"wan","OMC IP":"10.0.0.3"}],
				"IPSEC":[{"FORCEENCAPS":"1","TUNNEL_ENABLE":"1"}]
			}
		}
	}`)}

	materialized, err := materializePolicyParameters(policy, &device, []model.Device{device})
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(materialized.Config, &root))
	row := mapSlice(root["paramConfigList"])[0]
	sheets := mapValue(row["sheetParameters"])
	require.Equal(t, "http://acs.example.com", mapSlice(sheets["DEVICE"])[0]["URL"])
	require.NotContains(t, mapSlice(sheets["DEVICE"])[0], "Time Zone Term")
	require.NotContains(t, mapSlice(sheets["INTERFACE"])[0], "Interface Name")
	require.NotContains(t, mapSlice(sheets["INTERFACE"])[0], "Address Type")
	require.NotContains(t, mapSlice(sheets["INTERFACE"])[0], "Vlan Name")
	require.NotContains(t, mapSlice(sheets["INTERFACE"])[0], "OMC IP")
	require.Equal(t, "10.0.0.2", mapSlice(sheets["INTERFACE"])[0]["IP Address"])
	require.NotContains(t, mapSlice(sheets["IPSEC"])[0], "FORCEENCAPS")
	require.Equal(t, "1", mapSlice(sheets["IPSEC"])[0]["TUNNEL_ENABLE"])
}

func TestMaterializePolicyParametersRejectsDuplicateExplicitPCI(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":10,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1}
		},
		"paramConfigList":[
			{"serialNumber":"SN-1","pci":20},
			{"serialNumber":"SN-2","pci":20}
		]
	}`)}

	_, err := materializePolicyParameters(policy, &device, []model.Device{
		device, {ID: uuid.New(), SerialNumber: "SN-2"},
	})
	require.ErrorContains(t, err, "PCI 20")
}

func TestMaterializePolicyParametersRejectsExhaustedRange(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-2"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":1,"step":1},
			"pciAllocation":{"start":0,"end":10,"step":1}
		}
	}`)}

	_, err := materializePolicyParameters(policy, &device, []model.Device{
		{ID: uuid.New(), SerialNumber: "SN-1"}, device,
	})
	require.ErrorContains(t, err, "range is exhausted")
}

func TestValidatePolicyCommonParametersEnforcesGNBIDLength(t *testing.T) {
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"common",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":16777216,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1}
		}
	}`)}
	require.ErrorContains(t, validatePolicyCommonParameters(policy), "24-bit maximum")
}

func TestValidatePolicyCommonParametersIgnoresSpecifiedMode(t *testing.T) {
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
		"paramConfigMode":"specified",
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":16777216,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1}
		}
	}`)}

	require.NoError(t, validatePolicyCommonParameters(policy))
}
