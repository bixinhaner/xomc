package soap

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeInform_Bootstrap(t *testing.T) {
	f, err := os.Open("../../test/fixtures/soap/inform_bootstrap.xml")
	require.NoError(t, err)
	defer f.Close()

	inform, cwmpID, err := DecodeInform(f)
	require.NoError(t, err)

	assert.Equal(t, "100001", cwmpID)
	assert.Equal(t, "TestVendor", inform.DeviceId.Manufacturer)
	assert.Equal(t, "001122", inform.DeviceId.OUI)
	assert.Equal(t, "SmallCell-LTE", inform.DeviceId.ProductClass)
	assert.Equal(t, "TEST-SN-001", inform.DeviceId.SerialNumber)
	assert.Len(t, inform.Event, 1)
	assert.Equal(t, "0 BOOTSTRAP", inform.Event[0].EventCode)
	assert.Equal(t, 1, inform.MaxEnvelopes)
	assert.Len(t, inform.ParameterList, 5)
}

func TestDecodeInform_Periodic(t *testing.T) {
	f, err := os.Open("../../test/fixtures/soap/inform_periodic.xml")
	require.NoError(t, err)
	defer f.Close()

	inform, cwmpID, err := DecodeInform(f)
	require.NoError(t, err)

	assert.Equal(t, "100002", cwmpID)
	assert.Equal(t, "TEST-SN-001", inform.DeviceId.SerialNumber)
	assert.Len(t, inform.Event, 1)
	assert.Equal(t, "2 PERIODIC", inform.Event[0].EventCode)
	assert.Len(t, inform.ParameterList, 3)
}

func TestRenderInformResponse(t *testing.T) {
	data := InformResponseData{ID: "test-id-123"}
	result, err := RenderResponse(InformResponseTmpl, data)
	require.NoError(t, err)

	xml := string(result)
	assert.Contains(t, xml, "test-id-123")
	assert.Contains(t, xml, "InformResponse")
	assert.Contains(t, xml, "MaxEnvelopes")
}

func TestSanitizeBareAmpersands(t *testing.T) {
	cases := []struct{ in, want string }{
		{"no amp here", "no amp here"},
		{"a&b", "a&amp;b"},
		{"?fileType=PM&filename=x.xml", "?fileType=PM&amp;filename=x.xml"},
		{"already &amp; escaped", "already &amp; escaped"},
		{"&lt;tag&gt; &quot;q&quot; &apos;a&apos;", "&lt;tag&gt; &quot;q&quot; &apos;a&apos;"},
		{"numeric &#65; and &#x41; ok", "numeric &#65; and &#x41; ok"},
		{"bad &# and &amp ; spaced", "bad &amp;# and &amp;amp ; spaced"},
		{"trailing &", "trailing &amp;"},
		{"&unknown;", "&amp;unknown;"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, string(SanitizeBareAmpersands([]byte(c.in))), "input=%q", c.in)
	}
}

// TestDecodeAutonomousTransferComplete_UnescapedAmpersand 复现真机 BLQ/MLQ 固件：
// AutonomousTransferComplete 的 TransferURL 含未转义 '&'（?fileType=PM&filename=...）。
// 旧实现 encoding/xml 直接 "invalid character entity" 失败 → ACS 回 400；
// 现 SanitizeBareAmpersands 兜底后应能正常解析出 URL。
func TestDecodeAutonomousTransferComplete_UnescapedAmpersand(t *testing.T) {
	xmlBody := `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
 <soap:Header><cwmp:ID soap:mustUnderstand="1">ATC-1</cwmp:ID></soap:Header>
 <soap:Body>
  <cwmp:AutonomousTransferComplete>
   <AnnounceURL></AnnounceURL>
   <TransferURL>http://172.19.1.173:8080/smallcell/FileUploadService?fileType=PM&filename=A20260615.xml</TransferURL>
   <FaultStruct><FaultCode>0</FaultCode><FaultString></FaultString></FaultStruct>
  </cwmp:AutonomousTransferComplete>
 </soap:Body>
</soap:Envelope>`
	atc, cwmpID, err := DecodeAutonomousTransferComplete(strings.NewReader(xmlBody))
	require.NoError(t, err)
	assert.Equal(t, "ATC-1", cwmpID)
	assert.Contains(t, atc.TransferURL, "fileType=PM&filename=A20260615.xml")
}
