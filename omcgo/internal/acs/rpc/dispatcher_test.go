package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticTransferProvider struct {
	snapshot transfercfg.Snapshot
}

func (s staticTransferProvider) Snapshot(context.Context) transfercfg.Snapshot {
	return s.snapshot
}

type staticAddressResolver struct {
	decision      transfercfg.AddressDecision
	seenDeviceID  uuid.UUID
	seenDirection transfercfg.TransferDirection
}

func (s *staticAddressResolver) Resolve(
	_ context.Context,
	deviceID uuid.UUID,
	direction transfercfg.TransferDirection,
) (transfercfg.AddressDecision, error) {
	s.seenDeviceID = deviceID
	s.seenDirection = direction
	return s.decision, nil
}

type staticDownloadDeviceLookup struct {
	device *model.Device
}

func (s staticDownloadDeviceLookup) GetBySerialNumber(_ context.Context, _ string) (*model.Device, error) {
	return s.device, nil
}

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
		"X_BAICELLS_COM_PasswordReset",
		"X_COMMON_COM_PasswordReset",
		"GetParameterAttributes",
		"SetParameterAttributes",
		"GetRPCMethods",
	}

	assert.Len(t, d.handlers, len(expectedMethods))
	for _, method := range expectedMethods {
		_, ok := d.handlers[method]
		assert.True(t, ok, "handler missing for %s", method)
	}
}

func TestResetLMTPasswordHandler_BaicellsMethod(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "X_BAICELLS_COM_PasswordReset",
		CommandKey: "password-reset-key",
		Params:     json.RawMessage(`{}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-reset")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "cwmp:X_BAICELLS_COM_PasswordReset")
	assert.Contains(t, body, "<CommandKey>password-reset-key</CommandKey>")
	assert.Contains(t, body, "cwmp-id-reset")
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
	assert.Contains(t, body, "<ParameterNames SOAP-ENC:arrayType=\"xsd:string[2]\">")
	assert.NotContains(t, body, "<cwmp:ParameterNames")
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
	assert.Contains(t, body, "<ParameterPath>Device.DeviceInfo.</ParameterPath>")
	assert.Contains(t, body, "Device.DeviceInfo.")
	assert.Contains(t, body, "<NextLevel>true</NextLevel>")
	assert.NotContains(t, body, "<cwmp:ParameterPath>")
	assert.NotContains(t, body, "<cwmp:NextLevel>")
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
	// TR-069 §A.3.2.10: 子元素 unqualified（参考 templates.go rebootXML 注释）
	assert.Contains(t, body, "<CommandKey>reboot-key-1</CommandKey>")
	assert.NotContains(t, body, "<cwmp:CommandKey>")
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

func TestResetLMTPasswordHandler(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "X_BAICELLS_COM_PasswordReset",
		CommandKey: "reset-lmt-password-key-1",
		Params:     json.RawMessage(`{}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-reset-lmt-password")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "<cwmp:X_BAICELLS_COM_PasswordReset>")
	assert.Contains(t, body, "<CommandKey>reset-lmt-password-key-1</CommandKey>")
	assert.Contains(t, body, "</cwmp:X_BAICELLS_COM_PasswordReset>")
	assert.NotContains(t, body, "cwmp:SetParameterValues")
	assert.NotContains(t, body, "X_COM_Localweb_password")
}

func TestResetLMTPasswordHandler_LegacyCommonMethodStillBuildsBaicellsPayload(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method:     "X_COMMON_COM_PasswordReset",
		CommandKey: "legacy-reset-key-1",
		Params:     json.RawMessage(`{}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-reset-lmt-password-legacy")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "<cwmp:X_COMMON_COM_PasswordReset>")
	assert.Contains(t, body, "<CommandKey>legacy-reset-key-1</CommandKey>")
	assert.Contains(t, body, "</cwmp:X_COMMON_COM_PasswordReset>")
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
			"delay_seconds": 0,
			"md5": "d41d8cd98f00b204e9800998ecf8427e"
		}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-7")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "cwmp:Download")
	assert.Contains(t, body, "1 Firmware Upgrade Image")
	assert.Contains(t, body, "http://fileserver.example.com/firmware.bin")
	assert.Contains(t, body, "<CommandKey>dl-key-1</CommandKey>")
	assert.Contains(t, body, "<Username>dluser</Username>")
	assert.Contains(t, body, "<Password>dlpass</Password>")
	assert.Contains(t, body, "<FileSize>1048576</FileSize>")
	assert.Contains(t, body, "<TargetFileName>firmware.bin</TargetFileName>")
	// Download 报文必须携带 MD5（params.md5 → soap.DownloadData.Md5 → <Md5>）。
	assert.Contains(t, body, "<Md5>d41d8cd98f00b204e9800998ecf8427e</Md5>")
}

// TestDownloadHandler_RuntimeTransferConfigOverride 验证 runtime transfer config
// 只用于 BaseURL/Path 拼接，**不**把 Username/Password 注入 SOAP Download —— 这是
// 产品决策（与 Upload 对齐，CPE 不通过 HTTP Basic Auth 取文件，详见 dispatcher.go
// DownloadHandler.BuildRequest 注释）。本测试是反向断言，防止该决策被无意回退。
//
// 凭据来源仅认 Params.Username/Password（上层任务派发时显式塞入）；transfercfg
// 提供的 Username/Password 即便配置了也应被忽略。
func TestDownloadHandler_RuntimeTransferConfigOverride(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{
		TransferConfigProvider: staticTransferProvider{snapshot: transfercfg.Snapshot{
			Download: transfercfg.DownloadSettings{
				BaseURL:  "https://gateway.example.com:9443/omc/",
				Path:     "/smallcell/FileDownloadService",
				Username: "runtime-user",
				Password: "runtime-pass",
			},
		}},
	})
	cmd := &Command{
		Method:     "Download",
		CommandKey: "dl-key-2",
		Params: json.RawMessage(`{
			"file_type": "1 Firmware Upgrade Image",
			"url": "firmware/QAFA/V1/正式 pkg.bin",
			"file_size": 2048,
			"target_file_name": "pkg.bin",
			"delay_seconds": 0
		}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-id-runtime")
	require.NoError(t, err)
	body := string(result)
	// BaseURL/Path 拼接生效
	assert.Contains(t, body, "https://gateway.example.com:9443/omc/smallcell/FileDownloadService/firmware/QAFA/V1/%E6%AD%A3%E5%BC%8F%20pkg.bin")
	// 凭据**不**应被注入（runtime config 里的 Username/Password 不进 SOAP）
	assert.NotContains(t, body, "runtime-user",
		"runtime transfer config 的 Username 不应注入 SOAP（产品决策）")
	assert.NotContains(t, body, "runtime-pass",
		"runtime transfer config 的 Password 不应注入 SOAP（产品决策）")
	// 模板渲染时应为空标签
	assert.Contains(t, body, "<Username></Username>")
	assert.Contains(t, body, "<Password></Password>")
}

func TestDownloadHandler_NormalizesLegacyConfigBackupBucketInURL(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{
		DownloadBaseURL: "https://gateway.example.com",
		DownloadPath:    "/smallcell/FileDownloadService",
	})
	cmd := &Command{
		Method: "Download",
		Params: json.RawMessage(`{
			"file_type": "3 Vendor Configuration File",
			"url": "config_backup/backup/2026/07/24/SN001_CFG.xml"
		}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-config-restore")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "https://gateway.example.com/smallcell/FileDownloadService/config-backup/backup/2026/07/24/SN001_CFG.xml")
	assert.NotContains(t, body, "FileDownloadService/config_backup/")
}

func TestDownloadHandler_NormalizesLegacyConfigBackupBucketInAbsoluteURL(t *testing.T) {
	d := NewDispatcher()
	cmd := &Command{
		Method: "Download",
		Params: json.RawMessage(`{
			"file_type": "3 Vendor Configuration File",
			"url": "https://gateway.example.com/smallcell/FileDownloadService/config_backup/backup/SN001_CFG.xml"
		}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-config-restore")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "https://gateway.example.com/smallcell/FileDownloadService/config-backup/backup/SN001_CFG.xml")
	assert.NotContains(t, body, "FileDownloadService/config_backup/")
}

func TestDownloadHandler_PreservesExternalURLsContainingLegacyBucketSegment(t *testing.T) {
	d := NewDispatcher()
	for _, externalURL := range []string{
		"https://vendor.example/files/config_backup/fw.bin",
		"ftp://vendor.example/files/config_backup/fw.bin",
	} {
		t.Run(externalURL, func(t *testing.T) {
			cmd := &Command{
				Method: "Download",
				Params: json.RawMessage(fmt.Sprintf(`{
					"file_type": "1 Firmware Upgrade Image",
					"url": %q
				}`, externalURL)),
			}

			result, err := d.BuildRequest(cmd, "cwmp-vendor-download")

			require.NoError(t, err)
			assert.Contains(t, string(result), "<URL>"+externalURL+"</URL>")
		})
	}
}

func TestDownloadHandler_ConfigRestoreReResolvesAbsoluteURLAtSOAPBuild(t *testing.T) {
	deviceID := uuid.New()
	resolver := &staticAddressResolver{decision: transfercfg.AddressDecision{
		Direction: transfercfg.TransferDirectionDownload,
		Protocol:  transfercfg.TransferProtocolHTTPS,
		BaseURL:   "https://fresh.example.com:9443/secure",
	}}
	d := NewDispatcher(DispatcherConfig{
		TransferConfigProvider: staticTransferProvider{snapshot: transfercfg.Snapshot{
			Download: transfercfg.DownloadSettings{
				BaseURL:      "http://old.example.com",
				HTTPSBaseURL: "https://fresh.example.com:9443/secure",
				Path:         "/smallcell/FileDownloadService",
			},
		}},
		TransferAddressResolver: resolver,
		DownloadDeviceLookup:    staticDownloadDeviceLookup{device: &model.Device{ID: deviceID, SerialNumber: "SN001"}},
	})
	cmd := &Command{
		DeviceSN:   "SN001",
		Method:     "Download",
		CommandKey: "CONFIG_RESTORE_29800000_SN001",
		Params: json.RawMessage(`{
			"file_type": "10 48BF74 Configuration File",
			"url": "http://stale.example.com/old/smallcell/FileDownloadService/config-snapshots/DG298/restore/SN001_CFG.xml",
			"target_file_name": "SN001_CFG.xml",
			"md5": "a2bd0c39a47fbbc6a4c294eeac762b62"
		}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-config-restore")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "https://fresh.example.com:9443/secure/smallcell/FileDownloadService/config-snapshots/DG298/restore/SN001_CFG.xml")
	assert.NotContains(t, body, "stale.example.com")
	assert.Equal(t, deviceID, resolver.seenDeviceID)
	assert.Equal(t, transfercfg.TransferDirectionDownload, resolver.seenDirection)
}

func TestDownloadHandler_NonConfigRestoreAbsoluteURLIsNotReResolved(t *testing.T) {
	resolver := &staticAddressResolver{decision: transfercfg.AddressDecision{
		Direction: transfercfg.TransferDirectionDownload,
		Protocol:  transfercfg.TransferProtocolHTTPS,
		BaseURL:   "https://fresh.example.com:9443/secure",
	}}
	d := NewDispatcher(DispatcherConfig{
		TransferConfigProvider: staticTransferProvider{snapshot: transfercfg.Snapshot{
			Download: transfercfg.DownloadSettings{Path: "/smallcell/FileDownloadService"},
		}},
		TransferAddressResolver: resolver,
		DownloadDeviceLookup:    staticDownloadDeviceLookup{device: &model.Device{ID: uuid.New(), SerialNumber: "SN001"}},
	})
	cmd := &Command{
		DeviceSN:   "SN001",
		Method:     "Download",
		CommandKey: "firmware-download",
		Params: json.RawMessage(`{
			"file_type": "1 Firmware Upgrade Image",
			"url": "http://vendor.example.com/smallcell/FileDownloadService/firmware/pkg.bin"
		}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-firmware")

	require.NoError(t, err)
	assert.Contains(t, string(result), "http://vendor.example.com/smallcell/FileDownloadService/firmware/pkg.bin")
	assert.Equal(t, uuid.Nil, resolver.seenDeviceID)
}

func TestDownloadHandler_LicenseReResolvesAbsoluteURLAtSOAPBuild(t *testing.T) {
	deviceID := uuid.New()
	resolver := &staticAddressResolver{decision: transfercfg.AddressDecision{
		Direction: transfercfg.TransferDirectionDownload,
		Protocol:  transfercfg.TransferProtocolHTTPS,
		BaseURL:   "https://fresh-license.example.com:9443/secure",
	}}
	d := NewDispatcher(DispatcherConfig{
		TransferConfigProvider: staticTransferProvider{snapshot: transfercfg.Snapshot{
			Download: transfercfg.DownloadSettings{Path: "/smallcell/FileDownloadService"},
		}},
		TransferAddressResolver: resolver,
		DownloadDeviceLookup:    staticDownloadDeviceLookup{device: &model.Device{ID: deviceID, SerialNumber: "SN-LIC"}},
	})
	cmd := &Command{
		DeviceSN:   "SN-LIC",
		Method:     "Download",
		CommandKey: "LICENSE_UPGRADE_29900000_SN-LIC",
		Params: json.RawMessage(`{
			"file_type": "License File",
			"url": "http://stale.example.com/old/smallcell/FileDownloadService/device-licenses/SN-LIC.lic",
			"target_file_name": "SN-LIC.lic",
			"md5": "d41d8cd98f00b204e9800998ecf8427e"
		}`),
	}

	result, err := d.BuildRequest(cmd, "cwmp-license")

	require.NoError(t, err)
	body := string(result)
	assert.Contains(t, body, "https://fresh-license.example.com:9443/secure/smallcell/FileDownloadService/device-licenses/SN-LIC.lic")
	assert.NotContains(t, body, "stale.example.com")
	assert.Equal(t, deviceID, resolver.seenDeviceID)
	assert.Equal(t, transfercfg.TransferDirectionDownload, resolver.seenDirection)
}

func TestDownloadHandler_SoftwareUpgradeReResolvesAbsoluteURLAtSOAPBuild(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fileType  string
		staleURL  string
		wantURL   string
		notInBody string
	}{
		{
			name:      "IMG",
			fileType:  "1 Firmware Upgrade Image",
			staleURL:  "http://stale.example.com/old/smallcell/FileDownloadService/firmware/img/product/V2.0.0/fw.bin",
			wantURL:   "https://fresh-upgrade.example.com:9443/secure/smallcell/FileDownloadService/firmware/img/product/V2.0.0/fw.bin",
			notInBody: "stale.example.com",
		},
		{
			name:      "PATCH with encoded name",
			fileType:  "X 48BF74 Software Upgrade Patch",
			staleURL:  "http://stale.example.com/old/smallcell/FileDownloadService/firmware/patch/product%20A/V2.0.0/%E8%A1%A5%20%E4%B8%81.bin",
			wantURL:   "https://fresh-upgrade.example.com:9443/secure/smallcell/FileDownloadService/firmware/patch/product%20A/V2.0.0/%E8%A1%A5%20%E4%B8%81.bin",
			notInBody: "stale.example.com",
		},
		{
			name:      "FPGA",
			fileType:  "Firmware Upgrade Fpga",
			staleURL:  "firmware/fpga/product/V2.0.0/fpga.bin",
			wantURL:   "https://fresh-upgrade.example.com:9443/secure/smallcell/FileDownloadService/firmware/fpga/product/V2.0.0/fpga.bin",
			notInBody: "http://old.example.com",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deviceID := uuid.New()
			resolver := &staticAddressResolver{decision: transfercfg.AddressDecision{
				Direction: transfercfg.TransferDirectionDownload,
				Protocol:  transfercfg.TransferProtocolHTTPS,
				BaseURL:   "https://fresh-upgrade.example.com:9443/secure",
			}}
			d := NewDispatcher(DispatcherConfig{
				TransferConfigProvider: staticTransferProvider{snapshot: transfercfg.Snapshot{
					Download: transfercfg.DownloadSettings{
						BaseURL:      "http://old.example.com",
						HTTPSBaseURL: "https://fresh-upgrade.example.com:9443/secure",
						Path:         "/smallcell/FileDownloadService",
					},
				}},
				TransferAddressResolver: resolver,
				DownloadDeviceLookup:    staticDownloadDeviceLookup{device: &model.Device{ID: deviceID, SerialNumber: "SN-UPG"}},
			})
			cmd := &Command{
				DeviceSN:   "SN-UPG",
				Method:     "Download",
				CommandKey: "30000000-0000-4000-8000-000000000001",
				Params: json.RawMessage(fmt.Sprintf(`{
						"command_key": "Download Upgrade,30000000-0000-4000-8000-000000000001",
						"file_type": %q,
						"url": %q,
						"transfer_policy_managed": true,
						"target_file_name": "pkg.bin",
						"md5": "d41d8cd98f00b204e9800998ecf8427e"
				}`, tc.fileType, tc.staleURL)),
			}

			result, err := d.BuildRequest(cmd, "cwmp-upgrade")

			require.NoError(t, err)
			body := string(result)
			assert.Contains(t, body, tc.wantURL)
			assert.NotContains(t, body, tc.notInBody)
			assert.Contains(t, body, "<CommandKey>30000000-0000-4000-8000-000000000001</CommandKey>")
			assert.Equal(t, deviceID, resolver.seenDeviceID)
			assert.Equal(t, transfercfg.TransferDirectionDownload, resolver.seenDirection)
		})
	}
}

func TestDownloadHandler_SoftwareUpgradeDoesNotReResolveAPOrExternalVendorURL(t *testing.T) {
	for _, tc := range []struct {
		name    string
		url     string
		managed bool
	}{
		{
			name:    "AP firmware object",
			url:     "firmware/ap/product/V2.0.0/ap.bin",
			managed: true,
		},
		{
			name: "external vendor URL",
			url:  "http://vendor.example.com/downloads/firmware/patch/pkg.bin",
		},
		{
			name: "external vendor URL with FileDownloadService-looking path",
			url:  "http://vendor.example.com/smallcell/FileDownloadService/firmware/patch/pkg.bin",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &staticAddressResolver{decision: transfercfg.AddressDecision{
				Direction: transfercfg.TransferDirectionDownload,
				Protocol:  transfercfg.TransferProtocolHTTPS,
				BaseURL:   "https://fresh-upgrade.example.com:9443/secure",
			}}
			d := NewDispatcher(DispatcherConfig{
				TransferConfigProvider: staticTransferProvider{snapshot: transfercfg.Snapshot{
					Download: transfercfg.DownloadSettings{
						BaseURL: "http://old.example.com",
						Path:    "/smallcell/FileDownloadService",
					},
				}},
				TransferAddressResolver: resolver,
				DownloadDeviceLookup:    staticDownloadDeviceLookup{device: &model.Device{ID: uuid.New(), SerialNumber: "SN-UPG"}},
			})
			cmd := &Command{
				DeviceSN:   "SN-UPG",
				Method:     "Download",
				CommandKey: "30000000-0000-4000-8000-000000000002",
				Params: json.RawMessage(fmt.Sprintf(`{
						"command_key": "Download Upgrade,30000000-0000-4000-8000-000000000002",
						"file_type": "1 Firmware Upgrade Image",
						"url": %q,
						"transfer_policy_managed": %t
					}`, tc.url, tc.managed)),
			}

			result, err := d.BuildRequest(cmd, "cwmp-upgrade")

			require.NoError(t, err)
			assert.Contains(t, string(result), tc.url)
			assert.Equal(t, uuid.Nil, resolver.seenDeviceID)
		})
	}
}
