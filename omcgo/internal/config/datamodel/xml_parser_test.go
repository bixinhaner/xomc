package datamodel

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseParameterModelXML_Basic(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<parameterModel generateTime="2026-03-24T00:55:06+02:00" vendor="48BF74" networkType="LTE"
                serialNumber="1202000588233HB0039-LTE" modelVersion="1.0" totalEntries="5">
    <objects>
        <object name="Device." access="READ_WRITE" maxInstances="0" isList="false" />
        <object name="Device.DeviceInfo." access="READ_WRITE" maxInstances="0" isList="false" />
    </objects>
    <parameters>
        <param name="Device.DeviceInfo.SoftwareVersion" access="READ_ONLY" type="STRING" min="0" max="64"
               notify="ACTIVE_NOTIFICATION" forcedInform="true" changeApplies="Immediate" isList="false" />
        <param name="Device.ManagementServer.URL" access="READ_WRITE" type="STRING" min="0" max="256"
               notify="NO_NOTIFICATION" forcedInform="false" changeApplies="Immediate" isList="false" />
        <param name="Device.DeviceInfo.UpTime" access="READ_ONLY" type="U_INT"
               notify="NO_NOTIFICATION" forcedInform="false" defaultValue="FAP" changeApplies="Immediate" isList="false" />
    </parameters>
</parameterModel>`

	result, err := ParseParameterModelXML(strings.NewReader(xmlData))
	require.NoError(t, err)

	assert.Equal(t, "48BF74", result.Vendor)
	assert.Equal(t, "LTE", result.NetworkType)
	assert.Equal(t, "1202000588233HB0039-LTE", result.SerialNumber)
	assert.Equal(t, "1.0", result.ModelVersion)
	assert.Equal(t, 5, result.TotalEntries)
	assert.False(t, result.GenerateTime.IsZero())

	// Objects
	require.Len(t, result.Objects, 2)
	assert.Equal(t, "Device.", result.Objects[0].Name)
	assert.Equal(t, "READ_WRITE", result.Objects[0].Access)

	// Parameters
	require.Len(t, result.Parameters, 3)

	// Check SoftwareVersion parameter
	sv := result.Parameters[0]
	assert.Equal(t, "Device.DeviceInfo.SoftwareVersion", sv.Path)
	assert.Equal(t, "string", sv.Type)
	assert.False(t, sv.Writable)
	assert.Equal(t, "ACTIVE_NOTIFICATION", sv.Notify)
	assert.True(t, sv.ForcedInform)
	assert.Equal(t, "Immediate", sv.ChangeApplies)
	assert.NotNil(t, sv.Constraints)
	assert.Equal(t, 64, sv.Constraints.MaxLength)
	assert.Equal(t, "device_info", sv.Category)

	// Check URL parameter
	url := result.Parameters[1]
	assert.Equal(t, "Device.ManagementServer.URL", url.Path)
	assert.True(t, url.Writable)
	assert.Equal(t, "NO_NOTIFICATION", url.Notify)
	assert.False(t, url.ForcedInform)
	assert.Equal(t, "management", url.Category)

	// Check UpTime parameter (U_INT type)
	uptime := result.Parameters[2]
	assert.Equal(t, "unsignedInt", uptime.Type)
	assert.Equal(t, "FAP", uptime.DefaultValue)
}

func TestParseParameterModelXML_RealFile(t *testing.T) {
	const testFile = "/tmp/test_datamodel_export_v2.xml"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("test file %s not found, skipping", testFile)
	}

	f, err := os.Open(testFile)
	require.NoError(t, err)
	defer f.Close()

	result, err := ParseParameterModelXML(f)
	require.NoError(t, err)

	assert.Equal(t, "48BF74", result.Vendor)
	assert.Equal(t, "LTE", result.NetworkType)
	assert.Greater(t, len(result.Objects), 10, "should have many objects")
	assert.Greater(t, len(result.Parameters), 100, "should have many parameters")

	// Validate all parameters have paths
	for _, p := range result.Parameters {
		assert.NotEmpty(t, p.Path, "parameter should have a path")
		assert.NotEmpty(t, p.Type, "parameter should have a type")
	}

	// Validate all objects have names
	for _, o := range result.Objects {
		assert.NotEmpty(t, o.Name, "object should have a name")
	}

	errs := ValidateXMLModel(result)
	assert.Empty(t, errs, "real file should pass validation")
}

func TestNormalizeParamType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"STRING", "string"},
		{"U_INT", "unsignedInt"},
		{"INT", "int"},
		{"BOOLEAN", "boolean"},
		{"DATE_TIME", "dateTime"},
		{"BASE64", "base64"},
		{"UNKNOWN", "string"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, normalizeParamType(tt.input), "input: %s", tt.input)
	}
}

func TestBuildConstraints(t *testing.T) {
	// String type: min/max → MaxLength
	c := buildConstraints("STRING", "0", "64")
	require.NotNil(t, c)
	assert.Equal(t, 64, c.MaxLength)
	assert.Nil(t, c.MinValue)

	// Numeric type: min/max → MinValue/MaxValue
	c = buildConstraints("U_INT", "1", "65535")
	require.NotNil(t, c)
	assert.Equal(t, int64(1), *c.MinValue)
	assert.Equal(t, int64(65535), *c.MaxValue)

	// Empty → nil
	c = buildConstraints("STRING", "", "")
	assert.Nil(t, c)
}

func TestValidateXMLModel(t *testing.T) {
	// Valid model
	m := &ParsedParameterModel{
		Vendor:      "48BF74",
		NetworkType: "LTE",
		Parameters:  []Parameter{{Path: "Device.Test", Type: "string"}},
	}
	assert.Empty(t, ValidateXMLModel(m))

	// Missing vendor
	m.Vendor = ""
	errs := ValidateXMLModel(m)
	assert.Contains(t, errs[0], "vendor")

	// No parameters
	m.Vendor = "48BF74"
	m.Parameters = nil
	errs = ValidateXMLModel(m)
	assert.Contains(t, errs[0], "no parameters")
}

func TestDetectCategory(t *testing.T) {
	assert.Equal(t, "management", detectCategory("Device.ManagementServer.URL"))
	assert.Equal(t, "device_info", detectCategory("Device.DeviceInfo.SoftwareVersion"))
	assert.Equal(t, "radio", detectCategory("Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity"))
	assert.Equal(t, "alarm", detectCategory("Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier"))
	assert.Equal(t, "time", detectCategory("Device.Time.NTPServer1"))
	assert.Equal(t, "", detectCategory("Device.SomeUnknown.Param"))
}
