package rpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDispatcher_AllHandlersRegistered(t *testing.T) {
	d := NewDispatcher()

	expectedMethods := []string{
		"GetParameterValues",
		"SetParameterValues",
		"GetParameterNames",
		"AddObject",
		"DeleteObject",
		"Download",
		"Upload",
		"Reboot",
		"FactoryReset",
		"GetParameterAttributes",
		"SetParameterAttributes",
		"GetRPCMethods",
	}

	assert.Len(t, d.handlers, 12)
	for _, method := range expectedMethods {
		_, ok := d.handlers[method]
		assert.True(t, ok, "handler missing for %s", method)
	}
}

func TestDispatcher_BuildRequest_UnknownMethod(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "NonExistentMethod",
		CommandKey: "key-1",
		Params:     json.RawMessage(`{}`),
	}

	_, err := d.BuildRequest(cmd, "cwmp-id-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown RPC method")
}

func TestGetParameterValuesHandler(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "GetParameterValues",
		CommandKey: "test-key-1",
		Params:     json.RawMessage(`{"names":["Device.DeviceInfo.ModelName","Device.DeviceInfo.SerialNumber"]}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-1")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "cwmp:GetParameterValues")
	assert.Contains(t, body, "Device.DeviceInfo.ModelName")
	assert.Contains(t, body, "Device.DeviceInfo.SerialNumber")
	// SOAP header cwmp:ID should be the cwmpID parameter, not CommandKey
	assert.Contains(t, body, "cwmp-id-1")
}

func TestSetParameterValuesHandler(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "SetParameterValues",
		CommandKey: "test-key-2",
		Params:     json.RawMessage(`{"values":[{"name":"Device.ManagementServer.PeriodicInformInterval","value":"300"}]}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-2")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "cwmp:SetParameterValues")
	assert.Contains(t, body, "Device.ManagementServer.PeriodicInformInterval")
	assert.Contains(t, body, "300")
	// Default type should be xsd:string when not specified
	assert.Contains(t, body, "xsd:string")
}

func TestSetParameterValuesHandler_WithType(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "SetParameterValues",
		CommandKey: "test-key-3",
		Params:     json.RawMessage(`{"values":[{"name":"Device.ManagementServer.PeriodicInformInterval","value":"300","type":"xsd:unsignedInt"}]}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-3")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "xsd:unsignedInt")
	assert.NotContains(t, body, "xsd:string")
}

func TestGetParameterNamesHandler(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "GetParameterNames",
		CommandKey: "test-key-4",
		Params:     json.RawMessage(`{"path":"Device.DeviceInfo.","next_level":true}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-4")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "cwmp:GetParameterNames")
	assert.Contains(t, body, "Device.DeviceInfo.")
	assert.Contains(t, body, "<cwmp:NextLevel>true</cwmp:NextLevel>")
}

func TestRebootHandler(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "Reboot",
		CommandKey: "reboot-key-1",
		Params:     json.RawMessage(`{}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-5")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "cwmp:Reboot")
	assert.Contains(t, body, "<cwmp:CommandKey>reboot-key-1</cwmp:CommandKey>")
}

func TestFactoryResetHandler(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "FactoryReset",
		CommandKey: "reset-key-1",
		Params:     json.RawMessage(`{}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-6")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "cwmp:FactoryReset")
}

// TestGetRPCMethodsHandler verifies the TR-069 §A.3.1.2 GetRPCMethods RPC
// renders as an empty <cwmp:GetRPCMethods/> tag with the cwmp:ID header.
// Triggered from ops side by action="get_rpc_methods" (T-0102-c map).
func TestGetRPCMethodsHandler(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "GetRPCMethods",
		CommandKey: "rpc-methods-key-1", // CommandKey 不用于此 RPC，但 schema 上保留
		Params:     json.RawMessage(`{}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-grpcm")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "<cwmp:GetRPCMethods/>")
	// cwmp:ID 走 SOAP Header 注入
	assert.Contains(t, body, "cwmp-id-grpcm")
	// 不应有任何 body 内 child element（empty self-closing tag）
	assert.NotContains(t, body, "</cwmp:GetRPCMethods>")
}

func TestDownloadHandler(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "Download",
		CommandKey: "dl-key-1",
		Params: json.RawMessage(`{
			"file_type": "1 Firmware Upgrade Image",
			"url": "http://fileserver.example.com/firmware.bin",
			"username": "dluser",
			"password": "dlpass",
			"file_size": 1048576,
			"target_file_name": "firmware.bin",
			"delay_seconds": 0
		}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-7")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "cwmp:Download")
	assert.Contains(t, body, "1 Firmware Upgrade Image")
	assert.Contains(t, body, "http://fileserver.example.com/firmware.bin")
	assert.Contains(t, body, "<cwmp:CommandKey>dl-key-1</cwmp:CommandKey>")
	assert.Contains(t, body, "<cwmp:Username>dluser</cwmp:Username>")
	assert.Contains(t, body, "<cwmp:Password>dlpass</cwmp:Password>")
	assert.Contains(t, body, "<cwmp:FileSize>1048576</cwmp:FileSize>")
	assert.Contains(t, body, "<cwmp:TargetFileName>firmware.bin</cwmp:TargetFileName>")
}
