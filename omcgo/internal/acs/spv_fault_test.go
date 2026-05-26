package acs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 真实样本：BAICELLS mBS31001 在 SET Device.DeviceInfo.UserLabel 时返回
// outer 9003 + inner 9005 AttributeIdNotFound。来自 protocol-2026-05-26T11-48.log。
const baicellsSPVFaultSample = `<?xml version="1.0"?>
<soap-env:Envelope xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"
                   xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap-env:Header>
    <cwmp:ID soap-env:mustUnderstand="1">ID:intrnl.unset.id.SetParameterValues1779767325.19266954</cwmp:ID>
  </soap-env:Header>
  <soap-env:Body>
    <soap-env:Fault>
      <faultcode>Client</faultcode>
      <faultstring>CWMP fault</faultstring>
      <detail>
        <cwmp:Fault>
          <FaultCode>9003</FaultCode>
          <FaultString>Invalid arguments</FaultString>
          <SetParameterValuesFault>
            <ParameterName>Device.DeviceInfo.UserLabel</ParameterName>
            <FaultCode>9005</FaultCode>
            <FaultString>AttributeIdNotFound : Device.DeviceInfo.UserLabel</FaultString>
          </SetParameterValuesFault>
        </cwmp:Fault>
      </detail>
    </soap-env:Fault>
  </soap-env:Body>
</soap-env:Envelope>`

// 多 path 情形：CPE 同时报告两条 path 不支持。
const multiSPVFaultSample = `<soap:Envelope xmlns:soap="x" xmlns:cwmp="y">
<soap:Body><soap:Fault><detail><cwmp:Fault>
<FaultCode>9003</FaultCode><FaultString>Invalid arguments</FaultString>
<SetParameterValuesFault>
  <ParameterName>Device.A.Foo</ParameterName>
  <FaultCode>9005</FaultCode>
  <FaultString>not found</FaultString>
</SetParameterValuesFault>
<SetParameterValuesFault>
  <ParameterName>Device.B.Bar</ParameterName>
  <FaultCode>9008</FaultCode>
  <FaultString>read-only</FaultString>
</SetParameterValuesFault>
</cwmp:Fault></detail></soap:Fault></soap:Body></soap:Envelope>`

func TestExtractSPVFaults_RealBaicellsSample(t *testing.T) {
	got := extractSPVFaults([]byte(baicellsSPVFaultSample))
	if assert.Len(t, got, 1) {
		assert.Equal(t, "Device.DeviceInfo.UserLabel", got[0].ParameterName)
		assert.Equal(t, 9005, got[0].FaultCode)
		assert.Equal(t, "AttributeIdNotFound : Device.DeviceInfo.UserLabel", got[0].FaultString)
	}
}

func TestExtractSPVFaults_MultiplePaths(t *testing.T) {
	got := extractSPVFaults([]byte(multiSPVFaultSample))
	if assert.Len(t, got, 2) {
		assert.Equal(t, "Device.A.Foo", got[0].ParameterName)
		assert.Equal(t, 9005, got[0].FaultCode)
		assert.Equal(t, "Device.B.Bar", got[1].ParameterName)
		assert.Equal(t, 9008, got[1].FaultCode)
	}
}

func TestExtractSPVFaults_NoSPVBlock_ReturnsNil(t *testing.T) {
	// 普通 SOAP fault（无 SetParameterValuesFault 块，如 GPV 失败）
	plain := `<soap:Fault><faultcode>Server</faultcode><faultstring>x</faultstring>
<detail><cwmp:Fault><FaultCode>9001</FaultCode><FaultString>boom</FaultString></cwmp:Fault></detail>
</soap:Fault>`
	got := extractSPVFaults([]byte(plain))
	assert.Nil(t, got)
}

func TestEnrichFaultMsgWithSPV_Empty_ReturnsBase(t *testing.T) {
	got := enrichFaultMsgWithSPV("[Client] Invalid arguments", nil)
	assert.Equal(t, "[Client] Invalid arguments", got)
}

func TestEnrichFaultMsgWithSPV_AppendsPerPathDetail(t *testing.T) {
	faults := []SPVFault{
		{ParameterName: "Device.DeviceInfo.UserLabel", FaultCode: 9005, FaultString: "AttributeIdNotFound"},
	}
	got := enrichFaultMsgWithSPV("[Client] Invalid arguments", faults)
	assert.Equal(t,
		"[Client] Invalid arguments — path faults: [Device.DeviceInfo.UserLabel: 9005 AttributeIdNotFound]",
		got)
}
