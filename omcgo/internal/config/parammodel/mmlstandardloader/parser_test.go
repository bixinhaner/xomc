package mmlstandardloader

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleXML = `<?xml version="1.0" ?>
<standardModel totalPaths="6" totalObjects="2" totalParams="4">
    <objects>
        <object standardPath="Device.DeviceInfo.EU." access="READ_ONLY" changeApplies="Immediate"/>
        <object standardPath="Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell." access="READ_WRITE" changeApplies="Immediate"/>
    </objects>
    <parameters>
        <param standardPath="Device.DeviceInfo.AntennaInfo.Azimuth" access="READ_WRITE" type="INT" changeApplies="Immediate" min="0" max="359"/>
        <param standardPath="Device.DeviceInfo.SerialNumber" access="READ_ONLY" type="STRING" changeApplies="Immediate" max="64"/>
        <param standardPath="Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI" access="READ_WRITE" type="U_INT" changeApplies="Immediate" min="0" max="503"/>
        <param standardPath="Device.FaultMgmt.CurrentAlarm.AlarmActive" access="READ_ONLY" type="BOOLEAN" changeApplies="Immediate"/>
    </parameters>
</standardModel>`

func TestParseStandardXML_BasicStructure(t *testing.T) {
	params, objects, err := ParseStandardXML(strings.NewReader(sampleXML))
	require.NoError(t, err)
	assert.Len(t, params, 4)
	assert.Len(t, objects, 2)
}

func TestParseStandardXML_NumericConstraints(t *testing.T) {
	params, _, err := ParseStandardXML(strings.NewReader(sampleXML))
	require.NoError(t, err)

	// Azimuth INT 0-359
	azimuth := findParam(t, params, "Device.DeviceInfo.AntennaInfo.Azimuth")
	require.NotNil(t, azimuth.Min, "INT param should parse min")
	require.NotNil(t, azimuth.Max)
	assert.Equal(t, int64(0), *azimuth.Min)
	assert.Equal(t, int64(359), *azimuth.Max)
	assert.True(t, azimuth.IsWritable(), "access=READ_WRITE → writable")
	assert.False(t, azimuth.HasInstanceIndex(), "no {i} in path")
}

func TestParseStandardXML_StringMaxLen(t *testing.T) {
	params, _, err := ParseStandardXML(strings.NewReader(sampleXML))
	require.NoError(t, err)

	sn := findParam(t, params, "Device.DeviceInfo.SerialNumber")
	require.NotNil(t, sn.MaxLen, "STRING param max=N → MaxLen")
	assert.Equal(t, 64, *sn.MaxLen)
	assert.Nil(t, sn.Min, "STRING param should not have numeric min")
	assert.False(t, sn.IsWritable(), "READ_ONLY")
}

func TestParseStandardXML_InstanceIndexDetection(t *testing.T) {
	params, _, err := ParseStandardXML(strings.NewReader(sampleXML))
	require.NoError(t, err)

	pci := findParam(t, params, "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI")
	assert.True(t, pci.HasInstanceIndex(), "path 含 {i} 应被检测到")
}

func TestStripInstanceIndex(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"Device.X.{i}.Y", "Device.X.Y"},
		{"Device.X.{i}.", "Device.X."},
		{"Device.DeviceInfo.AntennaInfo", "Device.DeviceInfo.AntennaInfo"},
		{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI",
			"Device.Services.FAPService.CellConfig.LTE.RAN.NeighborList.LTECell.PCI"},
	}
	for _, c := range cases {
		assert.Equal(t, c.out, StripInstanceIndex(c.in), c.in)
	}
}

// 端到端：解析仓库里真实的 standard-model.xml — 验证 1988 行可解析。
func TestParseStandardXMLFile_RealData(t *testing.T) {
	xmlPath := repoXMLPath(t)
	params, objects, err := ParseStandardXMLFile(xmlPath)
	require.NoError(t, err)
	assert.Greater(t, len(params), 1900, "real file should have ~1988 params")
	assert.Greater(t, len(objects), 10, "real file should have ~13 objects")

	// 抽样校验：必须含 Device.DeviceInfo.AntennaInfo.Azimuth
	found := false
	for _, p := range params {
		if p.StandardPath == "Device.DeviceInfo.AntennaInfo.Azimuth" {
			found = true
			break
		}
	}
	assert.True(t, found, "Azimuth 必须存在")

	rfTxStatus := findParam(t, params, "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus")
	assert.Equal(t, "U_INT", rfTxStatus.Type, "RFTxStatus must be sent as xsd:unsignedInt")
}

// ============================================================
// helpers
// ============================================================

func findParam(t *testing.T, params []ParamSpec, path string) ParamSpec {
	t.Helper()
	for _, p := range params {
		if p.StandardPath == path {
			return p
		}
	}
	t.Fatalf("param %q not found in parsed result", path)
	return ParamSpec{}
}

// repoXMLPath 解析仓库里 standard-model.xml 的绝对路径（按 _test.go 相对位置推断）。
// 不依赖 CWD，便于 go test ./... 与 IDE 跑。
func repoXMLPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// 本测试在 omcgo/internal/config/parammodel/mmlstandardloader/
	// → 向上 4 层到 omcgo/，再进 data/param-mappings/standard-model.xml
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..",
		"data", "param-mappings", "standard-model.xml")
}
