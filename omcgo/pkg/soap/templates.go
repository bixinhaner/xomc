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
	InformResponseTmpl         *template.Template
	GetParameterValuesTmpl     *template.Template
	SetParameterValuesTmpl     *template.Template
	GetParameterNamesTmpl      *template.Template
	AddObjectTmpl              *template.Template
	DeleteObjectTmpl           *template.Template
	DownloadTmpl               *template.Template
	UploadTmpl                 *template.Template
	RebootTmpl                 *template.Template
	FactoryResetTmpl           *template.Template
	ScheduleInformTmpl         *template.Template
	FaultResponseTmpl          *template.Template
	TransferCompleteRespTmpl              *template.Template
	AutonomousTransferCompleteRespTmpl    *template.Template
	GetParameterAttributesTmpl            *template.Template
	SetParameterAttributesTmpl            *template.Template
	GetRPCMethodsTmpl                     *template.Template
)

func init() {
	InformResponseTmpl = template.Must(template.New("InformResponse").Parse(informResponseXML))
	GetParameterValuesTmpl = template.Must(template.New("GetParameterValues").Parse(getParameterValuesXML))
	SetParameterValuesTmpl = template.Must(template.New("SetParameterValues").Parse(setParameterValuesXML))
	GetParameterNamesTmpl = template.Must(template.New("GetParameterNames").Parse(getParameterNamesXML))
	AddObjectTmpl = template.Must(template.New("AddObject").Parse(addObjectXML))
	DeleteObjectTmpl = template.Must(template.New("DeleteObject").Parse(deleteObjectXML))
	DownloadTmpl = template.Must(template.New("Download").Funcs(soapFuncMap).Parse(downloadXML))
	UploadTmpl = template.Must(template.New("Upload").Funcs(soapFuncMap).Parse(uploadXML))
	RebootTmpl = template.Must(template.New("Reboot").Parse(rebootXML))
	FactoryResetTmpl = template.Must(template.New("FactoryReset").Parse(factoryResetXML))
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
	NoMoreRequests int    `json:"no_more_requests"` // 0=more requests coming, 1=last request
}

type UploadData struct {
	ID             string `json:"id"`
	CommandKey     string `json:"command_key"`
	FileType       string `json:"file_type"`
	URL            string `json:"url"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	DelaySeconds   int    `json:"delay_seconds"`
	Md5            string `json:"md5"`
	RawMode        string `json:"raw_mode"`
	NoMoreRequests int    `json:"no_more_requests"` // 0=more requests coming, 1=last request
}

type RebootData struct {
	ID             string
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

const setParameterValuesXML = soapEnvelopeOpen +
	`<cwmp:SetParameterValues>` +
	`<cwmp:ParameterList SOAP-ENC:arrayType="cwmp:ParameterValueStruct[{{len .Params}}]">` +
	`{{- range .Params}}<cwmp:ParameterValueStruct>` +
	`<cwmp:Name>{{.Name}}</cwmp:Name>` +
	`<cwmp:Value xsi:type="{{.Type}}">{{.Value}}</cwmp:Value>` +
	`</cwmp:ParameterValueStruct>{{end}}` +
	`</cwmp:ParameterList>` +
	`<cwmp:ParameterKey>{{.Key}}</cwmp:ParameterKey>` +
	`</cwmp:SetParameterValues>` + soapEnvelopeClose

const getParameterNamesXML = soapEnvelopeOpen +
	`<cwmp:GetParameterNames>` +
	`<ParameterPath>{{.Path}}</ParameterPath>` +
	`<NextLevel>{{if .NextLevel}}true{{else}}false{{end}}</NextLevel>` +
	`</cwmp:GetParameterNames>` + soapEnvelopeClose

const addObjectXML = soapEnvelopeOpen +
	`<cwmp:AddObject>` +
	`<cwmp:ObjectName>{{.ObjectName}}</cwmp:ObjectName>` +
	`<cwmp:ParameterKey>{{.Key}}</cwmp:ParameterKey>` +
	`</cwmp:AddObject>` + soapEnvelopeClose

const deleteObjectXML = soapEnvelopeOpen +
	`<cwmp:DeleteObject>` +
	`<cwmp:ObjectName>{{.ObjectName}}</cwmp:ObjectName>` +
	`<cwmp:ParameterKey>{{.Key}}</cwmp:ParameterKey>` +
	`</cwmp:DeleteObject>` + soapEnvelopeClose

// TR-069 §A.3.2.8 Download / §A.3.2.9 Upload: CWMP schema 使用
// elementFormDefault="unqualified"，RPC 参数子元素必须是无前缀（unqualified），
// 否则严格解析的 CPE 会把带 cwmp: 前缀的子元素视为未知元素而忽略，
// 导致 CPE 收到 SOAP 后既不回 UploadResponse 也不实际下载/上传。
const downloadXML = soapEnvelopeOpen +
	`<cwmp:Download>` +
	`<CommandKey>{{.CommandKey | xmlescape}}</CommandKey>` +
	`<FileType>{{.FileType | xmlescape}}</FileType>` +
	`<URL>{{.URL | xmlescape}}</URL>` +
	`<Username>{{.Username | xmlescape}}</Username>` +
	`<Password>{{.Password | xmlescape}}</Password>` +
	`<FileSize>{{.FileSize}}</FileSize>` +
	`<TargetFileName>{{.TargetFileName | xmlescape}}</TargetFileName>` +
	`<DelaySeconds>{{.DelaySeconds}}</DelaySeconds>` +
	`<SuccessURL></SuccessURL>` +
	`<FailureURL></FailureURL>` +
	`</cwmp:Download>` + soapEnvelopeClose

const uploadXML = soapEnvelopeOpen +
	`<cwmp:Upload>` +
	`<CommandKey>{{.CommandKey | xmlescape}}</CommandKey>` +
	`<FileType>{{.FileType | xmlescape}}</FileType>` +
	`<URL>{{.URL | xmlescape}}</URL>` +
	`<Username>{{.Username | xmlescape}}</Username>` +
	`<Password>{{.Password | xmlescape}}</Password>` +
	`<DelaySeconds>{{.DelaySeconds}}</DelaySeconds>` +
	`</cwmp:Upload>` + soapEnvelopeClose

const rebootXML = soapEnvelopeOpen +
	`<cwmp:Reboot>` +
	`<cwmp:CommandKey>{{.CommandKey}}</cwmp:CommandKey>` +
	`</cwmp:Reboot>` + soapEnvelopeClose

const factoryResetXML = soapEnvelopeOpen +
	`<cwmp:FactoryReset/>` + soapEnvelopeClose

// TR-069 §A.3.1.2 GetRPCMethods 请求：空标签，无参数；
// 响应由 CPE 返回 supported methods 列表（由 ACS 的 inform/response 处理回路读取）。
const getRPCMethodsXML = soapEnvelopeOpen +
	`<cwmp:GetRPCMethods/>` + soapEnvelopeClose

const scheduleInformXML = soapEnvelopeOpen +
	`<cwmp:ScheduleInform>` +
	`<cwmp:DelaySeconds>{{.DelaySeconds}}</cwmp:DelaySeconds>` +
	`<cwmp:CommandKey>{{.CommandKey}}</cwmp:CommandKey>` +
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

const getParameterAttributesXML = soapEnvelopeOpen +
	`<cwmp:GetParameterAttributes>` +
	`<cwmp:ParameterNames SOAP-ENC:arrayType="xsd:string[{{len .Params}}]">` +
	`{{- range .Params}}<string>{{.Name}}</string>{{end}}` +
	`</cwmp:ParameterNames>` +
	`</cwmp:GetParameterAttributes>` + soapEnvelopeClose

const setParameterAttributesXML = soapEnvelopeOpen +
	`<cwmp:SetParameterAttributes>` +
	`<cwmp:ParameterList SOAP-ENC:arrayType="cwmp:SetParameterAttributesStruct[{{len .Params}}]">` +
	`{{- range .Params}}<cwmp:SetParameterAttributesStruct>` +
	`<cwmp:Name>{{.Name}}</cwmp:Name>` +
	`<cwmp:NotificationChange>{{if .NotificationChange}}true{{else}}false{{end}}</cwmp:NotificationChange>` +
	`<cwmp:Notification>{{.Notification}}</cwmp:Notification>` +
	`<cwmp:AccessListChange>{{if .AccessListChange}}true{{else}}false{{end}}</cwmp:AccessListChange>` +
	`<cwmp:AccessList>{{range .AccessList}}<string>{{.}}</string>{{end}}</cwmp:AccessList>` +
	`</cwmp:SetParameterAttributesStruct>{{end}}` +
	`</cwmp:ParameterList>` +
	`</cwmp:SetParameterAttributes>` + soapEnvelopeClose

