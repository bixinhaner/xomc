package parammodel

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestXMLParamEntry_SupportedAttribute 验证 T-0103 新加的 supported XML 属性正确解析。
//
// 字典侧约定：缺省 / 任意非 "false" 字面值 → 视为 supported=true（默认含义），
// 仅严格 "false"（含大小写不敏感比较）→ 视为 false。
// 这里只断言原始字符串透传到 xmlParamEntry.Supported；
// 字符串 → bool 的转换在 loader.batchInsertMappings 由 EqualFold 完成。
func TestXMLParamEntry_SupportedAttribute(t *testing.T) {
	const snippet = `<?xml version="1.0"?>
<parameterModel paramModel="TestModel" totalEntries="3">
  <parameters>
    <param name="A.Default" standardPath="A.Default" type="STRING"/>
    <param name="A.SupportedFalse" standardPath="A.SupportedFalse" type="STRING" supported="false"/>
    <param name="A.SupportedTrue"  standardPath="A.SupportedTrue"  type="STRING" supported="true"/>
  </parameters>
</parameterModel>`

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal([]byte(snippet), &doc))
	require.Len(t, doc.Params, 3)

	byName := map[string]xmlParamEntry{}
	for _, p := range doc.Params {
		byName[p.Name] = p
	}

	// 缺省 → 空串（loader 后续 EqualFold("","false")=false → IsSupported=true）
	assert.Equal(t, "", byName["A.Default"].Supported)
	// 显式 false
	assert.Equal(t, "false", byName["A.SupportedFalse"].Supported)
	// 显式 true
	assert.Equal(t, "true", byName["A.SupportedTrue"].Supported)
}

func TestBMNeighborListHasPrivateArfcnAlias(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BM.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	const privatePath = "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.X_COM_EUTRAULEarfcn"
	const standardPath = "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.EUTRACarrierARFCN"

	for _, param := range doc.Params {
		if param.Name == privatePath {
			assert.Equal(t, standardPath, param.StandardPath)
			return
		}
	}

	t.Fatalf("expected BM.xml to define alias %s -> %s", privatePath, standardPath)
}

func TestBMNeighborListHasWritableX2Flag(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BM.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	const path = "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.X2Flag"
	for _, param := range doc.Params {
		if param.Name == path {
			assert.Equal(t, path, param.StandardPath)
			assert.Equal(t, "READ_WRITE", param.Access)
			assert.Equal(t, "U_INT", param.DataType)
			assert.Equal(t, "0,1", param.EnumValues)
			assert.Equal(t, "SON,Manual", param.EnumLabels)
			return
		}
	}

	t.Fatalf("expected BM.xml to define writable X2Flag mapping at %s", path)
}

func TestBMWANUsesIPInterfaceAliases(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BM.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	wantObjects := map[string]string{
		"Device.IP.Interface.{i}.":                 "Device.Ethernet.Interface.{i}.",
		"Device.IP.Interface.{i}.IPv4Address.{i}.": "Device.Ethernet.Interface.{i}.IPv4Address.{i}.",
		"Device.IP.Interface.{i}.IPv6Address.{i}.": "Device.Ethernet.Interface.{i}.IPv6Address.{i}.",
	}
	for privatePath, standardPath := range wantObjects {
		found := false
		for _, object := range doc.Objects {
			if object.Name != privatePath {
				continue
			}
			found = true
			assert.Equal(t, standardPath, object.StandardPath)
		}
		require.True(t, found, "expected BM WAN object alias %s", privatePath)
	}

	wantParams := map[string]struct {
		standardPath string
		access       string
	}{
		"Device.IP.Interface.{i}.Enable":                         {"Device.Ethernet.Interface.{i}.Enable", "READ_WRITE"},
		"Device.IP.Interface.{i}.Name":                           {"Device.Ethernet.Interface.{i}.Name", "READ_ONLY"},
		"Device.IP.Interface.{i}.Status":                         {"Device.Ethernet.Interface.{i}.Status", "READ_ONLY"},
		"Device.DeviceInfo.X_COM_MACAddress":                     {"Device.Ethernet.Interface.{i}.MACAddress", "READ_ONLY"},
		"Device.IP.Interface.{i}.IPv4Address.{i}.AddressingType": {"Device.Ethernet.Interface.{i}.IPv4Address.{i}.AddressingType", "READ_ONLY"},
		"Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress":      {"Device.Ethernet.Interface.{i}.IPv4Address.{i}.IPAddress", "READ_WRITE"},
		"Device.IP.Interface.{i}.IPv4Address.{i}.SubnetMask":     {"Device.Ethernet.Interface.{i}.IPv4Address.{i}.SubnetMask", "READ_WRITE"},
		"Device.IP.Interface.{i}.IPv6Address.{i}.IPAddress":      {"Device.Ethernet.Interface.{i}.IPv6Address.{i}.IPAddress", "READ_WRITE"},
		"Device.IP.Interface.{i}.IPv6Address.{i}.Origin":         {"Device.Ethernet.Interface.{i}.IPv6Address.{i}.Origin", "READ_ONLY"},
	}
	found := make(map[string]xmlParamEntry, len(wantParams))
	for _, param := range doc.Params {
		if _, ok := wantParams[param.Name]; ok {
			found[param.Name] = param
		}
	}
	for privatePath, want := range wantParams {
		param, ok := found[privatePath]
		require.True(t, ok, "expected BM WAN parameter alias %s", privatePath)
		assert.Equal(t, want.standardPath, param.StandardPath)
		assert.Equal(t, want.access, param.Access)
	}
}

func TestMLQPLMNListObjectIsWritable(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "MLQ.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	const path = "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList."
	for _, object := range doc.Objects {
		if object.StandardPath == path {
			assert.Equal(t, path, object.Name)
			assert.Equal(t, "READ_WRITE", object.Access)
			return
		}
	}

	t.Fatalf("expected MLQ.xml to define writable PLMNList object at %s", path)
}

func TestBMUpTimeUsesStandardPath(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BM.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	const path = "Device.DeviceInfo.UpTime"
	matches := 0

	for _, param := range doc.Params {
		if param.StandardPath == path {
			matches++
			assert.Equal(t, path, param.Name)
			assert.Equal(t, "U_INT", param.DataType)
		}
	}

	assert.Equal(t, 1, matches, "BM must expose uptime only via standard Device.DeviceInfo.UpTime")
}

func TestBaiBNQGNBNameIsWritableNRCommonPath(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BaiBNQ.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	const path = "Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBName"

	for _, param := range doc.Params {
		if param.Name == path {
			assert.Equal(t, path, param.StandardPath)
			assert.Equal(t, "READ_WRITE", param.Access)
			assert.Equal(t, "STRING", param.DataType)
			return
		}
	}

	t.Fatalf("expected BaiBNQ.xml to define writable gNBName mapping at %s", path)
}

func TestBaiBNQIncludesNRWANInterfaceParameters(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BaiBNQ.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	const interfaceObject = "Device.Ethernet.Interface.{i}."
	foundObject := false
	for _, object := range doc.Objects {
		if object.Name == interfaceObject {
			foundObject = true
			assert.Equal(t, interfaceObject, object.StandardPath)
			assert.Equal(t, "READ_WRITE", object.Access)
		}
	}
	require.True(t, foundObject, "NR must explicitly support WAN interface AddObject/DeleteObject")

	want := map[string]string{
		"Device.Ethernet.Interface.{i}.Enable":         "READ_WRITE",
		"Device.Ethernet.Interface.{i}.UserLabel":      "READ_WRITE",
		"Device.Ethernet.Interface.{i}.Name":           "READ_ONLY",
		"Device.Ethernet.Interface.{i}.Status":         "READ_WRITE",
		"Device.Ethernet.Interface.{i}.MACAddress":     "READ_ONLY",
		"Device.Ethernet.Interface.{i}.MaxBitRate":     "READ_WRITE",
		"Device.Ethernet.Interface.{i}.SignTransMedia": "READ_ONLY",
		"Device.Ethernet.Interface.{i}.DuplexMode":     "READ_WRITE",
		"Device.Ethernet.Interface.{i}.PortLocation":   "READ_ONLY",
		"Device.Ethernet.Interface.{i}.interfaceType":  "READ_WRITE",
	}
	found := make(map[string]xmlParamEntry, len(want))
	for _, param := range doc.Params {
		if _, ok := want[param.StandardPath]; ok {
			found[param.StandardPath] = param
		}
	}

	for path, access := range want {
		param, ok := found[path]
		require.True(t, ok, "expected BaiBNQ.xml to define NR WAN parameter %s", path)
		assert.Equal(t, path, param.Name)
		assert.Equal(t, access, param.Access)
	}
}

func TestBaiBNQTopLevelDeviceInfoUsesStandardPaths(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BaiBNQ.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	want := map[string]string{
		"Device.DeviceInfo.HardwareVersion": "Device.DeviceInfo.HardwareVersion",
		"Device.DeviceInfo.ModelName":       "Device.DeviceInfo.ModelName",
		"Device.DeviceInfo.SoftwareVersion": "Device.DeviceInfo.SoftwareVersion",
		"Device.DeviceInfo.UpTime":          "Device.DeviceInfo.UpTime",
	}
	got := make(map[string]string, len(want))
	for _, param := range doc.Params {
		if _, ok := want[param.Name]; ok {
			got[param.Name] = param.StandardPath
		}
	}

	for privatePath, standardPath := range want {
		require.Contains(t, got, privatePath, "expected BaiBNQ.xml to define %s", privatePath)
		assert.Equal(t, standardPath, got[privatePath])
	}
}

func TestUECountVendorPathsMapToStandardCurrentCount(t *testing.T) {
	tests := []struct {
		model       string
		privatePath string
	}{
		{model: "BaiBNQ", privatePath: "Device.DeviceInfo.X_CUSTOM_COM_UeNumber"},
		{model: "BSC", privatePath: "Device.DeviceInfo.X_COM_UE_Count"},
		{model: "BTS", privatePath: "Device.DeviceInfo.X_COM_UE_Count"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", tt.model+".xml")
			body, err := os.ReadFile(xmlPath)
			require.NoError(t, err)

			var doc xmlParameterModel
			require.NoError(t, xml.Unmarshal(body, &doc))

			for _, param := range doc.Params {
				if param.Name != tt.privatePath {
					continue
				}
				assert.Equal(t, "Device.DeviceInfo.UE_Count", param.StandardPath)
				assert.Equal(t, "READ_ONLY", param.Access)
				return
			}

			t.Fatalf("expected %s.xml to define UE count alias %s", tt.model, tt.privatePath)
		})
	}
}

func TestBaiBNQNguFallbackIsWritable(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BaiBNQ.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	var mappings []ParamMapping
	for _, param := range doc.Params {
		mappings = append(mappings, ParamMapping{
			PrivatePath:   param.Name,
			StandardPath:  param.StandardPath,
			EntryType:     "parameter",
			Access:        param.Access,
			DataType:      param.DataType,
			ChangeApplies: param.ChangeApplies,
		})
	}

	validator := NewMappingValidator(&MappingSet{Mappings: mappings})
	const path = "Device.LAN_HostConfigManagement.IPInterface.NgapMgmt.NguLocalIpAddrList"
	require.NotNil(t, validator.LookupParam(path), "expected BaiBNQ.xml to define %s", path)
	assert.Nil(t, validator.ValidateValue(path, "172.19.3.81"), path)
}

func TestBaiBNQLTEIdleReselectionCarrierObjectIsWritable(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BaiBNQ.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	const path = "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.IdleMode.EUTRA.Carrier."

	for _, object := range doc.Objects {
		if object.Name == path {
			assert.Equal(t, path, object.StandardPath)
			assert.Equal(t, "READ_WRITE", object.Access)
			return
		}
	}

	t.Fatalf("expected BaiBNQ.xml to define writable LTE idle reselection carrier object at %s", path)
}

func TestBaiBNQLTEIdleReselectionCarrierRanges(t *testing.T) {
	xmlPath := filepath.Join("..", "..", "..", "data", "param-mappings", "BaiBNQ.xml")
	body, err := os.ReadFile(xmlPath)
	require.NoError(t, err)

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal(body, &doc))

	const prefix = "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.IdleMode.EUTRA.Carrier.{i}."
	want := map[string]struct {
		dataType string
		min      string
		max      string
	}{
		"CellReselectionPriority": {"U_INT", "0", "7"},
		"EUTRACarrierARFCN":       {"U_INT", "0", "3279165"},
		"QRxLevMin":               {"INT", "-70", "-22"},
		"ThreshXHigh":             {"U_INT", "0", "31"},
		"ThreshXLow":              {"U_INT", "0", "31"},
	}

	found := make(map[string]xmlParamEntry)
	for _, param := range doc.Params {
		for leaf := range want {
			if param.Name == prefix+leaf {
				found[leaf] = param
			}
		}
	}

	for leaf, expectation := range want {
		param, ok := found[leaf]
		require.True(t, ok, "expected BaiBNQ.xml to define %s%s", prefix, leaf)
		assert.Equal(t, expectation.dataType, param.DataType, leaf)
		assert.Equal(t, expectation.min, param.Min, leaf)
		assert.Equal(t, expectation.max, param.Max, leaf)
	}
}

func TestNormalizeEnumCSV_ReplacesFullWidthComma(t *testing.T) {
	assert.Equal(t, "PSK,SIM,CERT,OTHER", normalizeEnumCSV(" PSK，SIM, ，CERT，OTHER "))
}

func TestXMLParamEntryCarriesValueRules(t *testing.T) {
	const snippet = `<?xml version="1.0"?>
<parameterModel paramModel="TestModel">
  <parameters>
    <param name="Device.Test.Value" standardPath="Device.Test.Value"
      type="STRING" min="1" max="8" defaultValue="abc"
      validationPattern="/^abc/" enumValues="a,b" enumLabels="A,B"/>
  </parameters>
</parameterModel>`

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal([]byte(snippet), &doc))
	require.Len(t, doc.Params, 1)
	entry := doc.Params[0]
	assert.Equal(t, "1", entry.Min)
	assert.Equal(t, "8", entry.Max)
	assert.Equal(t, "abc", entry.DefaultValue)
	assert.Equal(t, "/^abc/", entry.ValidationPattern)
	assert.Equal(t, "a,b", entry.EnumValues)
	assert.Equal(t, "A,B", entry.EnumLabels)
}
