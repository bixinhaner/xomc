package redact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSensitivePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"Device.ManagementServer.Password", true},
		{"InternetGatewayDevice.ManagementServer.ConnectionRequestPassword", true},
		{"Device.ManagementServer.STUNPassword", true},
		{"Device.Services.X_VENDOR.KeyPassphrase", true},
		{"Device.WiFi.AccessPoint.1.Security.PreSharedKey", true},
		{"Device.WiFi.AccessPoint.1.Security.WPAPreSharedKey", true},
		{"Device.ManagementServer.Username", false},
		{"Device.DeviceInfo.SerialNumber", false},
		{"Password", true},
		{"", false},
		{"Device.Foo.client_secret", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSensitivePath(tt.path))
		})
	}
}

func TestRedactXML_PasswordElement(t *testing.T) {
	// Download RPC body carries <Password> directly.
	in := `<cwmp:Download xmlns:cwmp="urn:dslforum-org:cwmp-1-0">` +
		`<Username>cpe-user</Username>` +
		`<Password>s3cr3tValue</Password>` +
		`<URL>http://acs/fw.bin</URL>` +
		`</cwmp:Download>`

	out := RedactXML(in)

	assert.NotContains(t, out, "s3cr3tValue", "raw password must not survive redaction")
	assert.Contains(t, out, "cpe-user", "non-sensitive username must be preserved")
	assert.Contains(t, out, "http://acs/fw.bin", "non-sensitive URL must be preserved")
	// MaskString("s3cr3tValue") = "s3***ue"
	assert.Contains(t, out, MaskString("s3cr3tValue"))
}

func TestRedactXML_ParameterValueStruct(t *testing.T) {
	// GetParameterValuesResponse: secret lives in <Value>, named by <Name>.
	in := `<cwmp:GetParameterValuesResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0">` +
		`<ParameterList>` +
		`<ParameterValueStruct>` +
		`<Name>Device.ManagementServer.Password</Name>` +
		`<Value>topSecretPw</Value>` +
		`</ParameterValueStruct>` +
		`<ParameterValueStruct>` +
		`<Name>Device.DeviceInfo.SerialNumber</Name>` +
		`<Value>SN-12345</Value>` +
		`</ParameterValueStruct>` +
		`</ParameterList>` +
		`</cwmp:GetParameterValuesResponse>`

	out := RedactXML(in)

	assert.NotContains(t, out, "topSecretPw", "password value must be masked")
	assert.Contains(t, out, "SN-12345", "non-sensitive value must be preserved")
	assert.Contains(t, out, "Device.ManagementServer.Password", "the Name path itself is not a secret")
}

func TestRedactXML_MultipleParamsIndependent(t *testing.T) {
	// Two consecutive structs: only the sensitive one is masked, and the
	// pending-sensitive flag must reset between structs.
	in := `<ParameterList>` +
		`<ParameterValueStruct><Name>Device.X.STUNPassword</Name><Value>aaaSecretaaa</Value></ParameterValueStruct>` +
		`<ParameterValueStruct><Name>Device.X.Interval</Name><Value>plainValue30</Value></ParameterValueStruct>` +
		`</ParameterList>`

	out := RedactXML(in)

	assert.NotContains(t, out, "aaaSecretaaa")
	assert.Contains(t, out, "plainValue30", "interval value must not be masked after a sensitive struct")
}

func TestRedactXML_EmptyAndMalformed(t *testing.T) {
	assert.Equal(t, "", RedactXML(""), "empty input returned unchanged")

	// Not well-formed XML: returned verbatim so caller can still store raw bytes.
	malformed := "this is <not> well-formed </xml"
	assert.Equal(t, malformed, RedactXML(malformed))
}

func TestRedactXML_PreservesStructure(t *testing.T) {
	in := `<a><b>keep</b><Password>hidden-value</Password></a>`
	out := RedactXML(in)
	require.True(t, strings.Contains(out, "keep"))
	assert.NotContains(t, out, "hidden-value")
	// Structure must remain parseable: tag count preserved.
	assert.Equal(t, strings.Count(in, "<"), strings.Count(out, "<"))
}
