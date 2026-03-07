package soap

import (
	"os"
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
