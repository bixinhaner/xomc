// Package errors 已在 errors.go 中声明包注释。
// 此文件将 global 包中的错误码常量按领域分组再导出。
// 内部代码优先导入此包的常量，不直接引用 global。
package errors

import "github.com/omcgo/omcgo/global"

// Re-export error codes from global for backwards compatibility.

// Device Management (1000-1999)
const (
	ErrCodeDeviceNotFound     = global.ErrCodeDeviceNotFound
	ErrCodeDeviceDuplicate    = global.ErrCodeDeviceDuplicate
	ErrCodeDeviceInvalidInput = global.ErrCodeDeviceInvalidInput
	ErrCodeDeviceOffline      = global.ErrCodeDeviceOffline
	ErrCodeDeviceRebootFailed = global.ErrCodeDeviceRebootFailed
)

// Data Model / Configuration (2000-2999)
const (
	ErrCodeDataModelNotFound     = global.ErrCodeDataModelNotFound
	ErrCodeDataModelInvalidScope = global.ErrCodeDataModelInvalidScope
	ErrCodeDataModelImportFailed = global.ErrCodeDataModelImportFailed
	ErrCodeTemplateNotFound      = global.ErrCodeTemplateNotFound
	ErrCodeTemplateInvalidParams = global.ErrCodeTemplateInvalidParams
	ErrCodeConfigSyncFailed      = global.ErrCodeConfigSyncFailed
)

// ACS / TR069 (3000-3999)
const (
	ErrCodeACSSessionTimeout  = global.ErrCodeACSSessionTimeout
	ErrCodeACSSessionNotFound = global.ErrCodeACSSessionNotFound
	ErrCodeACSCommandFailed   = global.ErrCodeACSCommandFailed
	ErrCodeACSConnectionReset = global.ErrCodeACSConnectionReset
)

// Performance Management (4000-4999)
const (
	ErrCodePMCounterNotFound   = global.ErrCodePMCounterNotFound
	ErrCodePMKPICalcFailed     = global.ErrCodePMKPICalcFailed
	ErrCodePMThresholdNotFound = global.ErrCodePMThresholdNotFound
	ErrCodePMKPINotFound       = global.ErrCodePMKPINotFound
)

// Alarm Management (5000-5999)
const (
	ErrCodeAlarmNotFound       = global.ErrCodeAlarmNotFound
	ErrCodeAlarmAlreadyAcked   = global.ErrCodeAlarmAlreadyAcked
	ErrCodeAlarmRuleNotFound   = global.ErrCodeAlarmRuleNotFound
	ErrCodeAlarmAlreadyCleared = global.ErrCodeAlarmAlreadyCleared
)

// Auto Provisioning (6000-6999)
const (
	ErrCodeProvisionTaskNotFound = global.ErrCodeProvisionTaskNotFound
	ErrCodeProvisionNoTemplate   = global.ErrCodeProvisionNoTemplate
	ErrCodeProvisionRetryFailed  = global.ErrCodeProvisionRetryFailed
)

// Admin / Auth / RBAC (7000-7999)
const (
	ErrCodeAuthInvalidCredentials = global.ErrCodeAuthInvalidCredentials
	ErrCodeAuthTokenExpired       = global.ErrCodeAuthTokenExpired
	ErrCodeAuthTokenInvalid       = global.ErrCodeAuthTokenInvalid
	ErrCodeUserNotFound           = global.ErrCodeUserNotFound
	ErrCodeUserDuplicate          = global.ErrCodeUserDuplicate
	ErrCodeUserLocked             = global.ErrCodeUserLocked
	ErrCodeRoleNotFound           = global.ErrCodeRoleNotFound
	ErrCodeRoleInUse              = global.ErrCodeRoleInUse
	ErrCodePermissionDenied       = global.ErrCodePermissionDenied
	ErrCodeAuthCaptchaRequired    = global.ErrCodeAuthCaptchaRequired
	ErrCodeAuthCaptchaInvalid     = global.ErrCodeAuthCaptchaInvalid
	ErrCodeAuthAccountLocked      = global.ErrCodeAuthAccountLocked
	ErrCodeAPIKeyNotFound         = global.ErrCodeAPIKeyNotFound
	ErrCodeAPIKeyRevoked          = global.ErrCodeAPIKeyRevoked
	ErrCodeAPIKeyExpired          = global.ErrCodeAPIKeyExpired
	ErrCodeAPIKeyForbidden        = global.ErrCodeAPIKeyForbidden
)

// Software / Firmware (8000-8999)
const (
	ErrCodeFirmwareNotFound    = global.ErrCodeFirmwareNotFound
	ErrCodeFirmwareDuplicate   = global.ErrCodeFirmwareDuplicate
	ErrCodeUpgradeTaskNotFound = global.ErrCodeUpgradeTaskNotFound
	ErrCodeUpgradeTaskFailed   = global.ErrCodeUpgradeTaskFailed
)

// Northbound / OSS (9000-9999)
const (
	ErrCodeNBTargetNotFound = global.ErrCodeNBTargetNotFound
	ErrCodeNBPushFailed     = global.ErrCodeNBPushFailed
	ErrCodeNBSyncFailed     = global.ErrCodeNBSyncFailed
	ErrCodeNBExportFailed   = global.ErrCodeNBExportFailed
)

// Interop Testing (10000-10999)
const (
	ErrCodeInteropTestFailed     = global.ErrCodeInteropTestFailed
	ErrCodeInteropValidateFailed = global.ErrCodeInteropValidateFailed
	ErrCodeInteropCaseNotFound   = global.ErrCodeInteropCaseNotFound
)

// Config Baseline/Task/Neighbor (11000-11999)
const (
	ErrCodeBaselineNotFound   = global.ErrCodeBaselineNotFound
	ErrCodeBaselineDuplicate  = global.ErrCodeBaselineDuplicate
	ErrCodeConfigTaskNotFound = global.ErrCodeConfigTaskNotFound
	ErrCodeNeighborNotFound   = global.ErrCodeNeighborNotFound
)

// License (12000-12999)
const (
	ErrCodeLicenseNotFound       = global.ErrCodeLicenseNotFound
	ErrCodeLicenseDuplicate      = global.ErrCodeLicenseDuplicate
	ErrCodeLicenseExpired        = global.ErrCodeLicenseExpired
	ErrCodeLicenseAlreadyActive  = global.ErrCodeLicenseAlreadyActive
	ErrCodeLicenseAlreadyRevoked = global.ErrCodeLicenseAlreadyRevoked
	ErrCodeSystemLicenseNotActive        = global.ErrCodeSystemLicenseNotActive
	ErrCodeSystemLicenseHardwareMismatch = global.ErrCodeSystemLicenseHardwareMismatch
	ErrCodeSystemLicenseCapacityExceeded = global.ErrCodeSystemLicenseCapacityExceeded
)

// Reports (13000-13999)
const (
	ErrCodeReportDefNotFound = global.ErrCodeReportDefNotFound
	ErrCodeReportRecNotFound = global.ErrCodeReportRecNotFound
	ErrCodeReportGenFailed   = global.ErrCodeReportGenFailed
)

// OpsTools (14000-14999)
const (
	ErrCodeOpsTemplateNotFound = global.ErrCodeOpsTemplateNotFound
	ErrCodeOpsTaskNotFound     = global.ErrCodeOpsTaskNotFound
	ErrCodeOpsTaskInvalidState = global.ErrCodeOpsTaskInvalidState
	ErrCodeOpsCommandFailed    = global.ErrCodeOpsCommandFailed
)

// Backup FTP (15000-15999)
const (
	ErrCodeFTPConfigNotFound   = global.ErrCodeFTPConfigNotFound
	ErrCodeFTPConnectionFailed = global.ErrCodeFTPConnectionFailed
)

// MR Indicators (16000-16999)
const (
	ErrCodeMRIndicatorNotFound = global.ErrCodeMRIndicatorNotFound
	ErrCodeMRMappingNotFound   = global.ErrCodeMRMappingNotFound
	ErrCodeMRExportFailed      = global.ErrCodeMRExportFailed
)
