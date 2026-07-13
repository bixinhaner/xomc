package mml

import (
	"strings"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

type scriptIssueDisplayText struct {
	zh string
	en string
}

var scriptIssueDisplayMessages = map[string]scriptIssueDisplayText{
	"MML_FILE_TOO_LARGE":                     {zh: "文件超过大小或行数限制", en: "File exceeds the size or line limit"},
	"MML_FILE_ENCODING_INVALID":              {zh: "文件必须使用 UTF-8 编码", en: "File must be UTF-8 encoded"},
	"MML_FILE_EMPTY":                         {zh: "文件中没有可执行命令", en: "The file has no executable command"},
	"MML_LINE_FORMAT_INVALID":                {zh: "命令格式不正确", en: "Command format is invalid"},
	"MML_DEVICE_SN_REQUIRED":                 {zh: "命令末尾必须使用 ;设备SN", en: "Command must end with ;device SN"},
	"MML_DEVICE_SN_MULTIPLE":                 {zh: "每行只能填写一个设备SN", en: "Each line may contain only one device SN"},
	"MML_COMMAND_NOT_FOUND":                  {zh: "当前用户不可用该命令", en: "The command is not available to the current user"},
	"MML_COMMAND_AMBIGUOUS":                  {zh: "命令配置重复，请联系管理员处理", en: "Multiple visible commands use this command code"},
	"MML_COMMAND_DISABLED":                   {zh: "命令已停用", en: "The command is disabled"},
	"MML_COMMAND_OPERATION_MISMATCH":         {zh: "命令操作类型与脚本行不一致", en: "The command operation does not match the script line"},
	"MML_COMMAND_CONFIRM_REQUIRED":           {zh: "该命令执行前需要确认", en: "The command requires confirmation before execution"},
	"MML_DEVICE_NOT_FOUND":                   {zh: "设备不存在", en: "Device does not exist"},
	"MML_DEVICE_OFFLINE":                     {zh: "设备当前离线", en: "Device is offline"},
	"MML_PATH_INVALID":                       {zh: "标准路径格式不正确", en: "Standard path format is invalid"},
	"MML_PATH_REQUIRED":                      {zh: "请填写标准路径", en: "Standard path is required"},
	"MML_PATH_COUNT_INVALID":                 {zh: "对象路径数量不符合操作要求", en: "The number of object paths does not match the operation"},
	"MML_PARAMETER_REQUIRED":                 {zh: "缺少必填参数或参数值", en: "Required parameter or value is missing"},
	"MML_PARAMETER_UNKNOWN":                  {zh: "参数不在该命令支持范围内", en: "Parameter is not supported by this command"},
	"MML_PARAMETER_READ_ONLY":                {zh: "参数为只读，不能修改", en: "Parameter is read-only"},
	"MML_PARAMETER_TYPE_INVALID":             {zh: "参数值类型不正确", en: "Parameter value type is invalid"},
	"MML_PARAMETER_ENUM_INVALID":             {zh: "参数值不在允许范围内", en: "Parameter value is outside the allowed enum"},
	"MML_PARAMETER_REGEX_INVALID":            {zh: "参数值格式不符合要求", en: "Parameter value does not match the required pattern"},
	"MML_PARAMETER_RANGE_INVALID":            {zh: "参数值超出允许范围", en: "Parameter value is outside the allowed range"},
	"MML_OPERATION_UNSUPPORTED":              {zh: "标准路径模式仅支持 LST、MOD、ADD、RMV", en: "Standard path mode supports only LST, MOD, ADD, and RMV"},
	"MML_SCRIPT_VALIDATION_FAILED":           {zh: "脚本校验失败", en: "Script validation failed"},
	"MML_SCRIPT_EXECUTION_VALIDATION_FAILED": {zh: "脚本执行前校验失败", en: "Script execution preflight validation failed"},
}

func localizeImportValidationResponse(result *ImportValidationResponse, locale appcontext.Locale) *ImportValidationResponse {
	if result == nil {
		return nil
	}
	localized := *result
	localized.Issues = localizeScriptIssues(result.Issues, locale)
	return &localized
}

func localizeScriptValidationResult(result *ScriptValidationResult, locale appcontext.Locale) *ScriptValidationResult {
	if result == nil {
		return nil
	}
	localized := *result
	localized.Issues = localizeScriptIssues(result.Issues, locale)
	return &localized
}

func localizeScriptIssues(issues []ScriptIssue, locale appcontext.Locale) []ScriptIssue {
	if len(issues) == 0 {
		return issues
	}
	localized := make([]ScriptIssue, len(issues))
	for i, issue := range issues {
		localized[i] = issue
		localized[i].DisplayMessage = scriptIssueDisplayMessage(issue, locale)
	}
	return localized
}

func scriptIssueDisplayMessage(issue ScriptIssue, locale appcontext.Locale) string {
	code := strings.TrimSpace(issue.Code)
	if text, ok := scriptIssueDisplayMessages[code]; ok {
		if locale == appcontext.LocaleEN {
			return text.en
		}
		return text.zh
	}
	if message := strings.TrimSpace(issue.DisplayMessage); message != "" {
		return message
	}
	if message := strings.TrimSpace(issue.Message); message != "" {
		return message
	}
	if locale == appcontext.LocaleEN {
		return "Validation failed"
	}
	return "校验未通过"
}
