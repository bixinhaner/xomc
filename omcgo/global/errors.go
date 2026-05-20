package global

// Error code ranges by domain. Each domain reserves a 1000-code block.

// Device Management (1000-1999)
const (
	ErrCodeDeviceNotFound     = 1001
	ErrCodeDeviceDuplicate    = 1002
	ErrCodeDeviceInvalidInput = 1003
	ErrCodeDeviceOffline      = 1004
	ErrCodeDeviceRebootFailed = 1005
)

// Device Group (1100-1199)
const (
	ErrCodeGroupNotFound       = 1101
	ErrCodeGroupDuplicate      = 1102
	ErrCodeGroupIsDefault      = 1103
	ErrCodeGroupLevelInvalid   = 1104
	ErrCodeGroupParentInvalid  = 1105
	ErrCodeGroupDeviceOnlyL2   = 1106
	ErrCodeGroupNameDuplicate  = 1107
)

// Device Registration (1200-1299)
const (
	ErrCodeRegistrationNotFound  = 1201
	ErrCodeRegistrationDuplicate = 1202
	ErrCodeRegistrationImportErr = 1203
	ErrCodeRegistrationInvalidSN = 1204
)

// Device Rules (1300-1399)
const (
	ErrCodeRuleNotFound          = 1301
	ErrCodeRulePriorityDuplicate = 1302
	ErrCodeRuleNotEnabled        = 1303
	ErrCodeRuleTaskNotFound      = 1304
	ErrCodeRuleTaskRunning       = 1305
	ErrCodeRuleInvalidParameter  = 1306
)

// Data Model / Configuration (2000-2999)
const (
	ErrCodeDataModelNotFound     = 2001
	ErrCodeDataModelInvalidScope = 2002
	ErrCodeDataModelImportFailed = 2003
	ErrCodeTemplateNotFound      = 2004
	ErrCodeTemplateInvalidParams = 2005
	ErrCodeConfigSyncFailed      = 2006
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
	ErrCodeAlarmNotFound      = 5001
	ErrCodeAlarmAlreadyAcked  = 5002
	ErrCodeAlarmRuleNotFound  = 5003
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
	ErrCodeAuthCaptchaRequired    = 7010
	ErrCodeAuthCaptchaInvalid     = 7011
	ErrCodeAuthAccountLocked      = 7012
	ErrCodeAPIKeyNotFound         = 7013
	ErrCodeAPIKeyRevoked          = 7014
	ErrCodeAPIKeyExpired          = 7015
	ErrCodeAPIKeyForbidden        = 7016
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

// Config Baseline/Task/Neighbor (11000-11999)
const (
	ErrCodeBaselineNotFound   = 11001
	ErrCodeBaselineDuplicate  = 11002
	ErrCodeConfigTaskNotFound = 11003
	ErrCodeNeighborNotFound   = 11004
)

// License (12000-12999)
const (
	ErrCodeLicenseNotFound       = 12001
	ErrCodeLicenseDuplicate      = 12002
	ErrCodeLicenseExpired        = 12003
	ErrCodeLicenseAlreadyActive  = 12004
	ErrCodeLicenseAlreadyRevoked = 12005

	// 12100-12109 段 — handler/service 业务异常（T-0100-P5-d 迁移自野生段 9100-9109）。
	// 命名约定：12100-12109 对应原 9100-9109 错误，保留语义一一对应方便回溯。
	ErrCodeLicenseRevokedActivate         = 12100 // 原 9100：尝试激活已 revoked 的 license
	ErrCodeLicenseQuotaLoad               = 12103 // 原 9103：加载 license quota 失败
	ErrCodeLicenseLogsServiceUnavail      = 12104 // 原 9104：log 服务未注入
	ErrCodeLicenseLogsListFailed          = 12105 // 原 9105：list 日志失败
	ErrCodeLicenseLogsByIDFailed          = 12106 // 原 9106：list 日志 by license id 失败
	ErrCodeLicenseExportFormatInvalid     = 12107 // 原 9107：单条导出 format 不合法
	ErrCodeLicenseBulkExportFormatInvalid = 12108 // 原 9108：批量导出 format 不合法
	ErrCodeLicenseSignatureVerifyFailed   = 12109 // 原 9109：strict 模式签名校验失败

	// 12110-12119 段 — System License 重构（F06-system-license-redesign PRD §5.3）。
	// 这一组错误码服务于 singleton system_license 模型；旧 multi-license 错误码
	// （12100-12109）在 Step 5 删除老 handler 时再下线。
	ErrCodeSystemLicenseIDExists      = 12110 // license_id 已存在于 current 或 history，拒绝重复上传
	ErrCodeSystemLicenseInvalidFormat = 12111 // license JSON 解析失败 / 必填字段缺失
	ErrCodeSystemLicenseDowngrade     = 12112 // 新 license 容量小于已用，需 force 或先降容（Step 3 enforcer 落地）
	ErrCodeSystemLicenseNotConfigured = 12113 // GetCurrent 时 system_license 表空
)

// Reports (13000-13999)
const (
	ErrCodeReportDefNotFound = 13001
	ErrCodeReportRecNotFound = 13002
	ErrCodeReportGenFailed   = 13003
)

// OpsTools (14000-14999)
const (
	ErrCodeOpsTemplateNotFound = 14001
	ErrCodeOpsTaskNotFound     = 14002
	ErrCodeOpsTaskInvalidState = 14003
	ErrCodeOpsCommandFailed    = 14004
)

// Backup FTP (15000-15999)
const (
	ErrCodeFTPConfigNotFound   = 15001
	ErrCodeFTPConnectionFailed = 15002
)

// MR Indicators (16000-16999)
const (
	ErrCodeMRIndicatorNotFound = 16001
	ErrCodeMRMappingNotFound   = 16002
	ErrCodeMRExportFailed      = 16003
)

// MML Console v2.3 catalog (17000-17999)
//
// 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.4
// 需求：R-8.4 / R-9.2 / R-9.3 — 结构化执行入参 + Translator 翻译 + 产品类型一致性
const (
	// R-8.4: execute-statements 入参设备包含多种 product_class
	ErrCodeDevicesMixedProductClass = 17001
	// R-8.4: execute-statements 入参设备为空或全部失效
	ErrCodeNoValidDevices = 17002
	// R-3: statement.operation_type 与 command 自身 op_type 不一致
	ErrCodeOperationMismatch = 17003
	// R-9.2: 结构化入参缺字段、paths 与 command 路径集不匹配、values key 不在 paths 内等
	ErrCodeInvalidStatementPayload = 17004
	// R-9.3: 全部 path 翻译失败（passthrough 关闭时；理论 passthrough 兜底不触发）
	ErrCodeTranslationFailedAll = 17005
	// R-9.3: devices.product_class 在 ProductRegistry 找不到匹配 product（孤儿设备）
	ErrCodeProductClassUnresolved = 17006

	// 用户私有模板命令名在同 owner 内重复（owner_user_id+command_name+private 三联唯一）
	// 关联：docs/design/mml-user-private-template-crud-20260520.md §3
	// HTTP 映射：→ 409 Conflict（通过 commonerrors.ErrAlreadyExists 链 wrap）
	ErrCodeTemplateNameDuplicated = 17008
)
