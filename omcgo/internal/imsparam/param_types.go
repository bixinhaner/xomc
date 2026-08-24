// Package imsparam 提供核心网（IMS Core）文件类型注册表与文件库。
//
// 背景：核心网文件传输（imscore_filetask.txt）合并为两个任务模板——
// 核心网文件采集（Upload RPC）/ 核心网文件下发（Download RPC）。方向由 RPC
// 方法本身表达，CWMP FileType 统一字面值 "Ims File"，具体类型由 SOAP 报文
// <ParameterType> 标签承载，取值为 FT_ImsCore_*（imscore_filetask.txt 的类型
// 标识）。本包是注册表的单一事实源：
//
//   - 后端校验（UFTE CreateTask / ACS upload handler / 派发器）
//   - HTTP 暴露（GET /imsparam/param-types，前端下拉用）
//
// 设计文档：docs/design/imscore-file-transfer.md
package imsparam

import "strings"

// CWMPFileTypeImsParam 是核心网侧要求的所有文件传输（参数/日志/License/恢复/
// 备份/鉴权，上传与下载）在 CWMP Download/Upload RPC 里统一使用的 FileType
// 字面值；方向由 RPC 方法表达，具体类型由报文 <ParameterType> 标签区分。
const CWMPFileTypeImsParam = "ImsCore Parameters File"

// UploadQueryFileTypeAlias 是 ACS 上传 URL（FileUploadService）query 参数
// fileType 使用的别名；子类型由 paramType query 参数携带（FT_ImsCore_*）。
const UploadQueryFileTypeAlias = "IMS_PARAM"

// FileTypeDef 描述一个核心网文件类型（FT_ImsCore_*）。
type FileTypeDef struct {
	// Code 是类型标识（核心网侧 FTx → FT_ImsCore 改造后），如
	// "FT_ImsCore_User_Setting_UD"，作为 SOAP <ParameterType> 标签值。
	Code string `json:"code"`
	// Name 是功能名称（中文），来自 imscore_filetask.txt。
	Name string `json:"name"`
	// DownloadSupported 表示是否支持下发（OMC → 核心网，Download RPC；
	// 核心网视角「下载」）。imscore_filetask.txt Download 段共 9 种。
	DownloadSupported bool `json:"downloadSupported"`
	// UploadSupported 表示是否支持采集（核心网 → OMC，Upload RPC；
	// 核心网视角「上传」）。imscore_filetask.txt Upload 段共 17 种。
	UploadSupported bool `json:"uploadSupported"`
}

// fileTypes 按 imscore_filetask.txt 顺序注册（Upload 段 + Download 段去重）。
var fileTypes = []FileTypeDef{
	// UD 双向（imscore_filetask.txt 两段重叠的 7 种）
	{Code: "FT_ImsCore_User_Setting_UD", Name: "开户数据", UploadSupported: true, DownloadSupported: true},
	{Code: "FT_ImsCore_Ims_User_Setting_UD", Name: "IMS用户数据", UploadSupported: true, DownloadSupported: true},
	{Code: "FT_ImsCore_Pbx_User_Setting_UD", Name: "PBX用户数据", UploadSupported: true, DownloadSupported: true},
	{Code: "FT_ImsCore_User_Apn_Setting_UD", Name: "用户APN设置", UploadSupported: true, DownloadSupported: true},
	{Code: "FT_ImsCore_Ue_IMEI_UD", Name: "UE IMEI", UploadSupported: true, DownloadSupported: true},
	{Code: "FT_ImsCore_Ue_Route_Setting_UD", Name: "UE 后路由设置", UploadSupported: true, DownloadSupported: true},
	{Code: "FT_ImsCore_Pcrf_Policy_Setting_UD", Name: "流量策略设置", UploadSupported: true, DownloadSupported: true},
	// 仅上传
	{Code: "FT_ImsCore_User_Location_Info_U", Name: "用户位置信息", UploadSupported: true},
	{Code: "FT_ImsCore_eNBgNB_Location_Info_U", Name: "基站位置信息", UploadSupported: true},
	{Code: "FT_ImsCore_Signaling_Events_U", Name: "信令事件", UploadSupported: true},
	{Code: "FT_ImsCore_Sip_Events_U", Name: "SIP事件", UploadSupported: true},
	{Code: "FT_ImsCore_Cdr_U", Name: "呼叫详单", UploadSupported: true},
	{Code: "FT_ImsCore_Operation_Logs_U", Name: "操作日志", UploadSupported: true},
	{Code: "FT_ImsCore_Download_Auth_U", Name: "授权认证文件", UploadSupported: true},
	{Code: "FT_ImsCore_Backups_U", Name: "核心网服务备份", UploadSupported: true},
	{Code: "FT_ImsCore_Core_Logs_U", Name: "核心网Core日志", UploadSupported: true},
	{Code: "FT_ImsCore_Web_Logs_U", Name: "核心网Web日志", UploadSupported: true},
	// 仅下发（文件库承载：License / 恢复）
	{Code: "FT_ImsCore_Upload_License_D", Name: "授权License", DownloadSupported: true},
	{Code: "FT_ImsCore_Recovery_D", Name: "核心网服务恢复", DownloadSupported: true},
}

// ParamTypes 返回全部文件类型定义（imscore_filetask.txt 顺序）。
func ParamTypes() []FileTypeDef {
	out := make([]FileTypeDef, len(fileTypes))
	copy(out, fileTypes)
	return out
}

// Lookup 按 Code（大小写不敏感）查文件类型定义。
func Lookup(code string) (FileTypeDef, bool) {
	trimmed := strings.TrimSpace(code)
	for _, item := range fileTypes {
		if strings.EqualFold(item.Code, trimmed) {
			return item, true
		}
	}
	return FileTypeDef{}, false
}

// NormalizeParamType 规范化类型标签：命中注册表返回规范的 Code
// （FT_ImsCore_* 精确大小写），否则原样去空白返回。
func NormalizeParamType(code string) string {
	if def, ok := Lookup(code); ok {
		return def.Code
	}
	return strings.TrimSpace(code)
}

// DisplayName 返回功能名称（如 "IMS用户数据"）；未知 code 返回原样串。
func DisplayName(code string) string {
	if def, ok := Lookup(code); ok {
		return def.Name
	}
	return code
}

// IsLogType 判断是否日志类（code 以 _Logs_U 结尾）——采集落地文件名用
// .log 后缀，其余 .dat。
func IsLogType(code string) bool {
	return strings.HasSuffix(strings.TrimSpace(code), "_Logs_U")
}
