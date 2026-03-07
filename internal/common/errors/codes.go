package errors

// Error code ranges by domain. Each domain reserves a 1000-code block.
// Handlers can use these constants with NewBusinessError() for consistent,
// machine-readable error responses.

// Device Management (1000-1999)
const (
	ErrCodeDeviceNotFound     = 1001
	ErrCodeDeviceDuplicate    = 1002
	ErrCodeDeviceInvalidInput = 1003
	ErrCodeDeviceOffline      = 1004
	ErrCodeDeviceRebootFailed = 1005
)

// Data Model / Configuration (2000-2999)
const (
	ErrCodeDataModelNotFound      = 2001
	ErrCodeDataModelInvalidScope  = 2002
	ErrCodeDataModelImportFailed  = 2003
	ErrCodeTemplateNotFound       = 2004
	ErrCodeTemplateInvalidParams  = 2005
	ErrCodeConfigSyncFailed       = 2006
)

// ACS / TR069 (3000-3999)
const (
	ErrCodeACSSessionTimeout  = 3001
	ErrCodeACSSessionNotFound = 3002
	ErrCodeACSCommandFailed   = 3003
	ErrCodeACSConnectionReset = 3004
)

// Performance Management (4000-4999)
const (
	ErrCodePMCounterNotFound   = 4001
	ErrCodePMKPICalcFailed     = 4002
	ErrCodePMThresholdNotFound = 4003
	ErrCodePMKPINotFound       = 4004
)

// Alarm Management (5000-5999)
const (
	ErrCodeAlarmNotFound     = 5001
	ErrCodeAlarmAlreadyAcked = 5002
	ErrCodeAlarmRuleNotFound = 5003
	ErrCodeAlarmAlreadyCleared = 5004
)

// Auto Provisioning (6000-6999)
const (
	ErrCodeProvisionTaskNotFound = 6001
	ErrCodeProvisionNoTemplate   = 6002
	ErrCodeProvisionRetryFailed  = 6003
)

// Admin / Auth / RBAC (7000-7999)
const (
	ErrCodeAuthInvalidCredentials = 7001
	ErrCodeAuthTokenExpired       = 7002
	ErrCodeAuthTokenInvalid       = 7003
	ErrCodeUserNotFound           = 7004
	ErrCodeUserDuplicate          = 7005
	ErrCodeUserLocked             = 7006
	ErrCodeRoleNotFound           = 7007
	ErrCodeRoleInUse              = 7008
	ErrCodePermissionDenied       = 7009
)

// Software / Firmware (8000-8999)
const (
	ErrCodeFirmwareNotFound    = 8001
	ErrCodeFirmwareDuplicate   = 8002
	ErrCodeUpgradeTaskNotFound = 8003
	ErrCodeUpgradeTaskFailed   = 8004
)

// Northbound / OSS (9000-9999)
const (
	ErrCodeNBTargetNotFound = 9001
	ErrCodeNBPushFailed     = 9002
	ErrCodeNBSyncFailed     = 9003
	ErrCodeNBExportFailed   = 9004
)

// Interop Testing (10000-10999)
const (
	ErrCodeInteropTestFailed     = 10001
	ErrCodeInteropValidateFailed = 10002
	ErrCodeInteropCaseNotFound   = 10003
)
