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

func TestMaterializePolicyParametersRemovesFieldsExcludedFromCommonConfiguration(t *testing.T) {
	device := model.Device{ID: uuid.New(), SerialNumber: "SN-1"}
	policy := &PlugAndPlayPolicy{SelfConfigEnabled: true, Config: json.RawMessage(`{
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
	policy := &PlugAndPlayPolicy{Config: json.RawMessage(`{
		"commonParamConfig":{
			"deviceType":"gNB","gnbIdLength":24,
			"gnbIdAllocation":{"start":1,"end":16777216,"step":1},
			"pciAllocation":{"start":0,"end":1007,"step":1}
		}
	}`)}
	require.ErrorContains(t, validatePolicyCommonParameters(policy), "24-bit maximum")
}
