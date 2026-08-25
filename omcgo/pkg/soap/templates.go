package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"text/template"
)

// xmlEscape 用 encoding/xml.EscapeText 把字符串里的 & < > " ' 等转义成 XML 实体。
// 模板里用户可控字段（URL/CommandKey/FileType/Username/Password/TargetFileName 等）
// 必须经此函数处理，否则未转义的 & 会让 CPE 端 XML 解析失败或截断 URL。
// 历史故障：Upload SOAP 的 URL 含 `?fileType=X&sn=Y&taskId=Z` 直接拼到模板，
// 设备收到后解析报错，既不回 UploadResponse 也不向 FileUploadService 上传。
func xmlEscape(s string) (string, error) {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var soapFuncMap = template.FuncMap{
	"xmlescape": xmlEscape,
}

// Pre-compiled SOAP response templates.
var (
	InformResponseTmpl                 *template.Template
	GetParameterValuesTmpl             *template.Template
	SetParameterValuesTmpl             *template.Template
	GetParameterNamesTmpl              *template.Template
	AddObjectTmpl                      *template.Template
	DeleteObjectTmpl                   *template.Template
	DownloadTmpl                       *template.Template
	UploadTmpl                         *template.Template
	RebootTmpl                         *template.Template
	FactoryResetTmpl                   *template.Template
	ResetLMTPasswordTmpl               *template.Template
	ScheduleInformTmpl                 *template.Template
	FaultResponseTmpl                  *template.Template
	TransferCompleteRespTmpl           *template.Template
	AutonomousTransferCompleteRespTmpl *template.Template
	GetParameterAttributesTmpl         *template.Template
	SetParameterAttributesTmpl         *template.Template
	GetRPCMethodsTmpl                  *template.Template
)

func init() {
	InformResponseTmpl = template.Must(template.New("InformResponse").Parse(informResponseXML))
	GetParameterValuesTmpl = template.Must(template.New("GetParameterValues").Parse(getParameterValuesXML))
	SetParameterValuesTmpl = template.Must(template.New("SetParameterValues").Funcs(soapFuncMap).Parse(setParameterValuesXML))
	GetParameterNamesTmpl = template.Must(template.New("GetParameterNames").Parse(getParameterNamesXML))
	AddObjectTmpl = template.Must(template.New("AddObject").Parse(addObjectXML))
	DeleteObjectTmpl = template.Must(template.New("DeleteObject").Parse(deleteObjectXML))
	DownloadTmpl = template.Must(template.New("Download").Funcs(soapFuncMap).Parse(downloadXML))
	UploadTmpl = template.Must(template.New("Upload").Funcs(soapFuncMap).Parse(uploadXML))
	RebootTmpl = template.Must(template.New("Reboot").Parse(rebootXML))
	FactoryResetTmpl = template.Must(template.New("FactoryReset").Parse(factoryResetXML))
	ResetLMTPasswordTmpl = template.Must(template.New("X_BAICELLS_COM_PasswordReset").Funcs(soapFuncMap).Parse(resetLMTPasswordXML))
	ScheduleInformTmpl = template.Must(template.New("ScheduleInform").Parse(scheduleInformXML))
	FaultResponseTmpl = template.Must(template.New("Fault").Parse(faultResponseXML))
	TransferCompleteRespTmpl = template.Must(template.New("TransferCompleteResponse").Parse(transferCompleteResponseXML))
	AutonomousTransferCompleteRespTmpl = template.Must(template.New("AutonomousTransferCompleteResponse").Parse(autonomousTransferCompleteResponseXML))
	GetParameterAttributesTmpl = template.Must(template.New("GetParameterAttributes").Parse(getParameterAttributesXML))
	SetParameterAttributesTmpl = template.Must(template.New("SetParameterAttributes").Parse(setParameterAttributesXML))
	GetRPCMethodsTmpl = template.Must(template.New("GetRPCMethods").Parse(getRPCMethodsXML))
}

// RenderResponse executes a SOAP template with the given data and returns the XML bytes.
func RenderResponse(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render SOAP template: %w", err)
	}
	return buf.Bytes(), nil
}

// HeaderData contains common SOAP header fields used by all templates.
// NoMoreRequests: 0 = ACS will send more requests, 1 = this is the last request.
type HeaderData struct {
	ID             string
	NoMoreRequests int // 0=more requests coming, 1=last request (omit for default 0)
}

// InformResponseData contains data for InformResponse template.
type InformResponseData struct {
	ID             string
	CurrentTime    string // ACS current time for CPE synchronization (optional per TR069 spec)
	NoMoreRequests int    // 0=more requests coming, 1=last request
}

type GetParameterValuesData struct {
	ID             string
	Params         []ParameterNameData
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type ParameterNameData struct {
	Name string
}

type SetParameterValuesData struct {
	ID             string
	Key            string
	Params         []ParameterSetData
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type ParameterSetData struct {
	Name  string
	Value string
	Type  string
}

type GetParameterNamesData struct {
	ID             string
	Path           string
	NextLevel      bool
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type AddObjectData struct {
	ID             string
	ObjectName     string
	Key            string
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type DeleteObjectData struct {
	ID             string
	ObjectName     string
	Key            string
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type DownloadData struct {
	ID             string `json:"id"`
	CommandKey     string `json:"command_key"`
	FileType       string `json:"file_type"`
	URL            string `json:"url"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	FileSize       int64  `json:"file_size"`
	TargetFileName string `json:"target_file_name"`
	DelaySeconds   int    `json:"delay_seconds"`
	Md5            string `json:"md5"`
	RawMode        string `json:"raw_mode"`
	// ParamType 是核心网参数配置（ImsCore Parameters File）的细分标签（FT_ImsCore_*），
	// 渲染为 SOAP 报文的 <ParameterType> 元素。
	// 非核心网任务留空，模板条件渲染 → 报文不含该标签，与既有厂商 CPE 完全兼容。
	ParamType string `json:"param_type"`
	// LogType 是核心网日志（Ims Log File）的细分标签（FT_ImsCore_*_Logs_U），
	// 渲染为 <LogType> 元素；与 ParamType 按文件大类互斥。
	LogType        string `json:"log_type"`
	NoMoreRequests int    `json:"no_more_requests"` // 0=more requests coming, 1=last request
}

type UploadData struct {
	ID           string `json:"id"`
	CommandKey   string `json:"command_key"`
	FileType     string `json:"file_type"`
	URL          string `json:"url"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DelaySeconds int    `json:"delay_seconds"`
	Md5          string `json:"md5"`
	RawMode      string `json:"raw_mode"`
	// ParamType 同上：渲染为 <ParameterType> 元素，非核心网任务留空（不渲染）。
	ParamType string `json:"param_type"`
	// LogType 同上：渲染为 <LogType> 元素（日志类任务用）。
	LogType        string `json:"log_type"`
	NoMoreRequests int    `json:"no_more_requests"` // 0=more requests coming, 1=last request
}

type RebootData struct {
	ID             string
	CommandKey     string
	RebootTarget   int
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type ResetLMTPasswordData struct {
	ID             string
	Method         string
	CommandKey     string
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type ScheduleInformData struct {
	ID             string
	DelaySeconds   int
	CommandKey     string
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type FaultData struct {
	ID             string
	FaultCode      int
	FaultString    string
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type GetParameterAttributesData struct {
	ID             string
	Params         []ParameterNameData
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type SetParameterAttributesData struct {
	ID             string
	Params         []SetParameterAttributeData
	NoMoreRequests int // 0=more requests coming, 1=last request
}

// FactoryResetData contains data for FactoryReset template.
type FactoryResetData struct {
	ID             string
	NoMoreRequests int // 0=more requests coming, 1=last request
}

// GetRPCMethodsData contains data for GetRPCMethods template.
// TR-069 §A.3.1.2: ACS 询问 CPE 支持的 RPC 方法列表，请求体仅含空标签
// `<cwmp:GetRPCMethods/>`，无任何参数。
type GetRPCMethodsData struct {
	ID             string
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type SetParameterAttributeData struct {
	Name               string
	NotificationChange bool
	Notification       int
	AccessListChange   bool
	AccessList         []string
}

// XML Templates — Compact mode: no whitespace between elements to avoid CPE parsing issues.

const soapEnvelopeOpen = `<?xml version="1.0" encoding="UTF-8"?>` +
	`<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"` +
	` xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/"` +
	` xmlns:cwmp="urn:dslforum-org:cwmp-1-0"` +
	` xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` +
	` xmlns:xsd="http://www.w3.org/2001/XMLSchema">` +
	`<SOAP-ENV:Header>` +
	`<cwmp:ID SOAP-ENV:mustUnderstand="1">{{.ID}}</cwmp:ID>` +
	// TR-069 §3.7.1.5 NoMoreRequests 在 ACS→CPE 请求中必须显式出现。实测厂商 CPE
	// （baicells/MMMM 系列）在该 header 缺失时直接静默丢弃 Upload/Download：
	// 既不回 RPC 响应也不向 FileUploadService PUT，HTTP session 看似正常但业务卡死。
	// 因此不要再用 `if NoMoreRequests` 条件跳过 0；直接渲染（0 = 还会有后续请求）。
	`<cwmp:NoMoreRequests>{{.NoMoreRequests}}</cwmp:NoMoreRequests>` +
	`</SOAP-ENV:Header>` +
	`<SOAP-ENV:Body>`

const soapEnvelopeClose = `</SOAP-ENV:Body></SOAP-ENV:Envelope>`

const informResponseXML = soapEnvelopeOpen +
	`<cwmp:InformResponse>` +
	`<cwmp:MaxEnvelopes>1</cwmp:MaxEnvelopes>` +
	`{{- if .CurrentTime}}<cwmp:CurrentTime>{{.CurrentTime}}</cwmp:CurrentTime>{{end}}` +
	`</cwmp:InformResponse>` + soapEnvelopeClose

const getParameterValuesXML = soapEnvelopeOpen +
	`<cwmp:GetParameterValues>` +
	`<ParameterNames SOAP-ENC:arrayType="xsd:string[{{len .Params}}]">` +
	`{{- range .Params}}<string>{{.Name}}</string>{{end}}` +
	`</ParameterNames>` +
	`</cwmp:GetParameterValues>` + soapEnvelopeClose

// setParameterValuesXML
//
// TR-069 §A.3.2.1 SetParameterValues: CWMP schema 使用 elementFormDefault="unqualified"，
// RPC 参数子元素（ParameterList / ParameterValueStruct / Name / Value / ParameterKey）
// 必须是无前缀（unqualified）；只有 RPC 顶层元素 cwmp:SetParameterValues 与 arrayType 里
// 指向类型定义的 cwmp:ParameterValueStruct[N] 带 cwmp: 前缀。
//
// 历史教训（双重）：
//  1. URL 含 `&` 未 xmlescape，CPE XML parser 报错（与下方 Download/Upload 模板同根因）
//  2. 内层元素带 cwmp: 前缀，严格按 spec 校验的 CPE（baicells/FAP/BU1810 实测）
//     直接把 ParameterList 当未知元素忽略，回 "Empty parameter list" SOAP Fault，
//     RPC handler 整个失败。所有用户可控字段同时过 xmlescape。
const setParameterValuesXML = soapEnvelopeOpen +
	`<cwmp:SetParameterValues>` +
	`<ParameterList SOAP-ENC:arrayType="cwmp:ParameterValueStruct[{{len .Params}}]">` +
	`{{- range .Params}}<ParameterValueStruct>` +
	`<Name>{{.Name | xmlescape}}</Name>` +
	`<Value xsi:type="{{.Type}}">{{.Value | xmlescape}}</Value>` +
	`</ParameterValueStruct>{{end}}` +
	`</ParameterList>` +
	`<ParameterKey>{{.Key | xmlescape}}</ParameterKey>` +
	`</cwmp:SetParameterValues>` + soapEnvelopeClose

const getParameterNamesXML = soapEnvelopeOpen +
	`<cwmp:GetParameterNames>` +
	`<ParameterPath>{{.Path}}</ParameterPath>` +
	`<NextLevel>{{if .NextLevel}}true{{else}}false{{end}}</NextLevel>` +
	`</cwmp:GetParameterNames>` + soapEnvelopeClose

// AddObject / DeleteObject — RPC 参数子元素同样 unqualified（参考 SetParameterValues 注释）。
const addObjectXML = soapEnvelopeOpen +
	`<cwmp:AddObject>` +
	`<ObjectName>{{.ObjectName}}</ObjectName>` +
	`<ParameterKey>{{.Key}}</ParameterKey>` +
	`</cwmp:AddObject>` + soapEnvelopeClose

const deleteObjectXML = soapEnvelopeOpen +
	`<cwmp:DeleteObject>` +
	`<ObjectName>{{.ObjectName}}</ObjectName>` +
	`<ParameterKey>{{.Key}}</ParameterKey>` +
	`</cwmp:DeleteObject>` + soapEnvelopeClose

// TR-069 §A.3.2.8 Download / §A.3.2.9 Upload: CWMP schema 使用
// elementFormDefault="unqualified"，RPC 参数子元素必须是无前缀（unqualified），
// 否则严格解析的 CPE 会把带 cwmp: 前缀的子元素视为未知元素而忽略，
// 导致 CPE 收到 SOAP 后既不回 UploadResponse 也不实际下载/上传。
const downloadXML = soapEnvelopeOpen +
	`<cwmp:Download>` +
	`<CommandKey>{{.CommandKey | xmlescape}}</CommandKey>` +
	`<FileType>{{.FileType | xmlescape}}</FileType>` +
	// ParameterType：核心网参数配置的 vendor 扩展标签（FT_ImsCore_*）。仅非空时渲染，
	// 其它 Download（固件/配置恢复/license）不带此标签，保持既有厂商兼容。
	`{{- if .ParamType}}<ParameterType>{{.ParamType | xmlescape}}</ParameterType>{{end}}` +
	`{{- if .LogType}}<LogType>{{.LogType | xmlescape}}</LogType>{{end}}` +
	`<URL>{{.URL | xmlescape}}</URL>` +
	`<Username>{{.Username | xmlescape}}</Username>` +
	`<Password>{{.Password | xmlescape}}</Password>` +
	`<FileSize>{{.FileSize}}</FileSize>` +
	`<TargetFileName>{{.TargetFileName | xmlescape}}</TargetFileName>` +
	`<DelaySeconds>{{.DelaySeconds}}</DelaySeconds>` +
	// MD5：CPE 下载后做文件完整性校验的必填字段。三类下行文件的 MD5 来源不同——
	// 固件升级 / license 在上传入库时算好（fw.MD5Val / lic.MD5），配置文件恢复在
	// 派发时读 MinIO 文件流现算（RestoreService.computeSourceMD5）。值由上层塞进
	// Download params 的 md5 字段 → soap.DownloadData.Md5 → 此处渲染。
	`<Md5>{{.Md5 | xmlescape}}</Md5>` +
	`<SuccessURL></SuccessURL>` +
	`<FailureURL></FailureURL>` +
	`</cwmp:Download>` + soapEnvelopeClose

const uploadXML = soapEnvelopeOpen +
	`<cwmp:Upload>` +
	`<CommandKey>{{.CommandKey | xmlescape}}</CommandKey>` +
	`<FileType>{{.FileType | xmlescape}}</FileType>` +
	// ParameterType：同上，核心网参数采集的 vendor 扩展标签，仅非空时渲染。
	`{{- if .ParamType}}<ParameterType>{{.ParamType | xmlescape}}</ParameterType>{{end}}` +
	`{{- if .LogType}}<LogType>{{.LogType | xmlescape}}</LogType>{{end}}` +
	`<URL>{{.URL | xmlescape}}</URL>` +
	`<Username>{{.Username | xmlescape}}</Username>` +
	`<Password>{{.Password | xmlescape}}</Password>` +
	`<DelaySeconds>{{.DelaySeconds}}</DelaySeconds>` +
	`</cwmp:Upload>` + soapEnvelopeClose

// Reboot — CommandKey 子元素 unqualified（参考 SetParameterValues 注释）。
const rebootXML = soapEnvelopeOpen +
	`<cwmp:Reboot>` +
	`<CommandKey>{{.CommandKey}}</CommandKey>` +
	`{{- if .RebootTarget}}<X_BaiCells_ImsCore_RebootTarget>{{.RebootTarget}}</X_BaiCells_ImsCore_RebootTarget>{{end}}` +
	`</cwmp:Reboot>` + soapEnvelopeClose

const factoryResetXML = soapEnvelopeOpen +
	`<cwmp:FactoryReset/>` + soapEnvelopeClose

const resetLMTPasswordXML = soapEnvelopeOpen +
	`<cwmp:{{.Method}}>` +
	`<CommandKey>{{.CommandKey | xmlescape}}</CommandKey>` +
	`</cwmp:{{.Method}}>` + soapEnvelopeClose

// TR-069 §A.3.1.2 GetRPCMethods 请求：空标签，无参数；
// 响应由 CPE 返回 supported methods 列表（由 ACS 的 inform/response 处理回路读取）。
const getRPCMethodsXML = soapEnvelopeOpen +
	`<cwmp:GetRPCMethods/>` + soapEnvelopeClose

// ScheduleInform — DelaySeconds / CommandKey 子元素 unqualified（参考 SetParameterValues 注释）。
const scheduleInformXML = soapEnvelopeOpen +
	`<cwmp:ScheduleInform>` +
	`<DelaySeconds>{{.DelaySeconds}}</DelaySeconds>` +
	`<CommandKey>{{.CommandKey}}</CommandKey>` +
	`</cwmp:ScheduleInform>` + soapEnvelopeClose

const faultResponseXML = soapEnvelopeOpen +
	`<SOAP-ENV:Fault>` +
	`<faultcode>Client</faultcode>` +
	`<faultstring>CWMP fault</faultstring>` +
	`<detail><cwmp:Fault>` +
	`<cwmp:FaultCode>{{.FaultCode}}</cwmp:FaultCode>` +
	`<cwmp:FaultString>{{.FaultString}}</cwmp:FaultString>` +
	`</cwmp:Fault></detail>` +
	`</SOAP-ENV:Fault>` + soapEnvelopeClose

const transferCompleteResponseXML = soapEnvelopeOpen +
	`<cwmp:TransferCompleteResponse/>` + soapEnvelopeClose

const autonomousTransferCompleteResponseXML = soapEnvelopeOpen +
	`<cwmp:AutonomousTransferCompleteResponse/>` + soapEnvelopeClose

// GetParameterAttributes / SetParameterAttributes — RPC 参数子元素同样 unqualified
// （参考 SetParameterValues 注释）。arrayType 里的类型名（cwmp:SetParameterAttributesStruct）
// 保留 cwmp: 前缀（指向类型定义）。
const getParameterAttributesXML = soapEnvelopeOpen +
	`<cwmp:GetParameterAttributes>` +
	`<ParameterNames SOAP-ENC:arrayType="xsd:string[{{len .Params}}]">` +
	`{{- range .Params}}<string>{{.Name}}</string>{{end}}` +
	`</ParameterNames>` +
	`</cwmp:GetParameterAttributes>` + soapEnvelopeClose

const setParameterAttributesXML = soapEnvelopeOpen +
	`<cwmp:SetParameterAttributes>` +
	`<ParameterList SOAP-ENC:arrayType="cwmp:SetParameterAttributesStruct[{{len .Params}}]">` +
	`{{- range .Params}}<SetParameterAttributesStruct>` +
	`<Name>{{.Name}}</Name>` +
	`<NotificationChange>{{if .NotificationChange}}true{{else}}false{{end}}</NotificationChange>` +
	`<Notification>{{.Notification}}</Notification>` +
	`<AccessListChange>{{if .AccessListChange}}true{{else}}false{{end}}</AccessListChange>` +
	`<AccessList>{{range .AccessList}}<string>{{.}}</string>{{end}}</AccessList>` +
	`</SetParameterAttributesStruct>{{end}}` +
	`</ParameterList>` +
	`</cwmp:SetParameterAttributes>` + soapEnvelopeClose
