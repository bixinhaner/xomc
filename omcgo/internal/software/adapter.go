package software

import (
	"fmt"
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

// UpgradeAdapter isolates 4G/5G upgrade and rollback parameter differences
// from the executor logic. Each technology combination may have
// different rollback parameter paths and download file types.
//
// 所有 Rollback*Path 方法返回的都是 **standardPath**——下游必须经 ParamPathTranslator
// 翻译为目标设备 param_model 的私有路径后再下发 GPV / SPV。直接拿来当 cwmp:Name
// 是错的，会被不识别该路径的 CPE 拒收。
//
// This is a separate interface from carrier.Carrier to avoid modifying
// the core carrier package and all its adapters for this feature.
// In the future these methods could be migrated into carrier.Carrier.
type UpgradeAdapter interface {
	// RollbackEnableCheckPath 返回回退前需要 GET 校验的 standardPath；
	// 不需要校验时（5G/NR）返回空字符串。
	RollbackEnableCheckPath(tech model.Technology) string

	// RollbackParameterPath 返回触发回退的 SPV standardPath：
	//   - 4G/LTE: "Device.DeviceInfo.ROLLBACK_CONTROL"
	//   - 5G/NR:  "Device.SoftwareCtrl.ActivateEnable"
	// 调用方需要先用 ParamPathTranslator 把这个 standardPath 翻译成 device 私有路径再下发。
	RollbackParameterPath(tech model.Technology) string

	// RollbackParameterValue 根据 Translator 翻译后的 privatePath 决定 SPV 的值与
	// xsd 类型：
	//   - privatePath 形如 InternetGatewayDevice.DeviceInfo.RollBackEnable（TR-098
	//     系老风格 boolean 触发） → value="true", type="xsd:boolean"
	//   - 其它（典型 Device.DeviceInfo.X_COM_ROLLBACK_CONTROL / Device.SoftwareCtrl.
	//     ActivateEnable 等 string/int 触发） → value="1", type="xsd:string"
	// 5G/NR 不受 privatePath 影响，统一 "1" + xsd:string。
	RollbackParameterValue(tech model.Technology, privatePath string) (value, xsdType string)

	// RollbackNeedsEnableCheck returns true if the device must first be queried
	// to verify rollback is available before setting the rollback parameter.
	// 4G/LTE typically requires this check; 5G/NR does not.
	RollbackNeedsEnableCheck(tech model.Technology) bool

	// DownloadFileType returns the TR-069 FileType string for a given firmware file type.
	DownloadFileType(fileType FileType) string

	// DownloadCommandKey returns the command key for a Download RPC.
	DownloadCommandKey(subTaskID string) string
}

// DefaultUpgradeAdapter provides default 4G/5G upgrade behavior.
// Technology-specific differences can be implemented by creating custom adapters.
type DefaultUpgradeAdapter struct{}

// NewDefaultUpgradeAdapter creates a new DefaultUpgradeAdapter.
func NewDefaultUpgradeAdapter() *DefaultUpgradeAdapter {
	return &DefaultUpgradeAdapter{}
}

func (a *DefaultUpgradeAdapter) RollbackEnableCheckPath(tech model.Technology) string {
	if tech == model.TechLTE {
		// standardPath（字典里 ROLLBACK_ENABLE 无 X_COM 前缀），Translator 翻译到
		// 各 param_model 对应的私有 path（BM/BLQ/BSC/BTS/MLN/MLQ → X_COM_ROLLBACK_ENABLE）
		return "Device.DeviceInfo.ROLLBACK_ENABLE"
	}
	return ""
}

func (a *DefaultUpgradeAdapter) RollbackParameterPath(tech model.Technology) string {
	switch tech {
	case model.TechNR:
		return "Device.SoftwareCtrl.ActivateEnable"
	default: // LTE
		// standardPath（无 X_COM 前缀）。Translator 翻译到各 param_model 对应的私有
		// 路径再下发，详见 RollbackParameterValue 注释。
		return "Device.DeviceInfo.ROLLBACK_CONTROL"
	}
}

func (a *DefaultUpgradeAdapter) RollbackParameterValue(tech model.Technology, privatePath string) (string, string) {
	// 5G 不分私有路径，统一 string/1
	if tech == model.TechNR {
		return "1", "xsd:string"
	}
	// 4G：TR-098 系（InternetGatewayDevice.* + RollBackEnable 驼峰命名）走 boolean，
	// 其它（X_COM_ROLLBACK_CONTROL / 等 TR-181 风格）走 string/1
	lower := strings.ToLower(strings.TrimSpace(privatePath))
	if strings.HasPrefix(lower, "internetgatewaydevice.") && strings.Contains(lower, "rollbackenable") {
		return "true", "xsd:boolean"
	}
	return "1", "xsd:string"
}

func (a *DefaultUpgradeAdapter) RollbackNeedsEnableCheck(tech model.Technology) bool {
	return tech == model.TechLTE
}

// DownloadFileType returns the TR-069 FileType string for a given firmware file type.
func (a *DefaultUpgradeAdapter) DownloadFileType(fileType FileType) string {
	switch fileType {
	case FileTypeIMG:
		return "1 Firmware Upgrade Image"
	case FileTypePATCH:
		return "1 Firmware Upgrade Image"
	case FileTypeFPGA:
		return "1 Firmware Upgrade Image"
	default:
		return "1 Firmware Upgrade Image"
	}
}

// DownloadCommandKey returns the command key for a Download RPC.
func (a *DefaultUpgradeAdapter) DownloadCommandKey(subTaskID string) string {
	return fmt.Sprintf("Download Upgrade,%s", subTaskID)
}

// Is5G returns true if the device is a 5G/NR device.
// Determined by Technology field or ProductClass being BNQ/BNX.
func Is5G(dev *model.Device) bool {
	return dev.Technology == model.TechNR ||
		dev.ProductClass == "BNQ" ||
		dev.ProductClass == "BNX"
}
