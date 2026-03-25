package soap

import (
	"bytes"
	"fmt"
	"text/template"
)

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
)

func init() {
	InformResponseTmpl = template.Must(template.New("InformResponse").Parse(informResponseXML))
	GetParameterValuesTmpl = template.Must(template.New("GetParameterValues").Parse(getParameterValuesXML))
	SetParameterValuesTmpl = template.Must(template.New("SetParameterValues").Parse(setParameterValuesXML))
	GetParameterNamesTmpl = template.Must(template.New("GetParameterNames").Parse(getParameterNamesXML))
	AddObjectTmpl = template.Must(template.New("AddObject").Parse(addObjectXML))
	DeleteObjectTmpl = template.Must(template.New("DeleteObject").Parse(deleteObjectXML))
	DownloadTmpl = template.Must(template.New("Download").Parse(downloadXML))
	UploadTmpl = template.Must(template.New("Upload").Parse(uploadXML))
	RebootTmpl = template.Must(template.New("Reboot").Parse(rebootXML))
	FactoryResetTmpl = template.Must(template.New("FactoryReset").Parse(factoryResetXML))
	ScheduleInformTmpl = template.Must(template.New("ScheduleInform").Parse(scheduleInformXML))
	FaultResponseTmpl = template.Must(template.New("Fault").Parse(faultResponseXML))
	TransferCompleteRespTmpl = template.Must(template.New("TransferCompleteResponse").Parse(transferCompleteResponseXML))
	AutonomousTransferCompleteRespTmpl = template.Must(template.New("AutonomousTransferCompleteResponse").Parse(autonomousTransferCompleteResponseXML))
	GetParameterAttributesTmpl = template.Must(template.New("GetParameterAttributes").Parse(getParameterAttributesXML))
	SetParameterAttributesTmpl = template.Must(template.New("SetParameterAttributes").Parse(setParameterAttributesXML))
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
	`{{- if .NoMoreRequests}}<cwmp:NoMoreRequests>{{.NoMoreRequests}}</cwmp:NoMoreRequests>{{end}}` +
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
	`<cwmp:ParameterNames SOAP-ENC:arrayType="xsd:string[{{len .Params}}]">` +
	`{{- range .Params}}<string>{{.Name}}</string>{{end}}` +
	`</cwmp:ParameterNames>` +
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
	`<cwmp:ParameterPath>{{.Path}}</cwmp:ParameterPath>` +
	`<cwmp:NextLevel>{{if .NextLevel}}true{{else}}false{{end}}</cwmp:NextLevel>` +
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

const downloadXML = soapEnvelopeOpen +
	`<cwmp:Download>` +
	`<cwmp:CommandKey>{{.CommandKey}}</cwmp:CommandKey>` +
	`<cwmp:FileType>{{.FileType}}</cwmp:FileType>` +
	`<cwmp:URL>{{.URL}}</cwmp:URL>` +
	`<cwmp:Username>{{.Username}}</cwmp:Username>` +
	`<cwmp:Password>{{.Password}}</cwmp:Password>` +
	`<cwmp:FileSize>{{.FileSize}}</cwmp:FileSize>` +
	`<cwmp:TargetFileName>{{.TargetFileName}}</cwmp:TargetFileName>` +
	`<cwmp:DelaySeconds>{{.DelaySeconds}}</cwmp:DelaySeconds>` +
	`<cwmp:SuccessURL></cwmp:SuccessURL>` +
	`<cwmp:FailureURL></cwmp:FailureURL>` +
	`</cwmp:Download>` + soapEnvelopeClose

const uploadXML = soapEnvelopeOpen +
	`<cwmp:Upload>` +
	`<cwmp:CommandKey>{{.CommandKey}}</cwmp:CommandKey>` +
	`<cwmp:FileType>{{.FileType}}</cwmp:FileType>` +
	`<cwmp:URL>{{.URL}}</cwmp:URL>` +
	`<cwmp:Username>{{.Username}}</cwmp:Username>` +
	`<cwmp:Password>{{.Password}}</cwmp:Password>` +
	`<cwmp:DelaySeconds>{{.DelaySeconds}}</cwmp:DelaySeconds>` +
	`</cwmp:Upload>` + soapEnvelopeClose

const rebootXML = soapEnvelopeOpen +
	`<cwmp:Reboot>` +
	`<cwmp:CommandKey>{{.CommandKey}}</cwmp:CommandKey>` +
	`</cwmp:Reboot>` + soapEnvelopeClose

const factoryResetXML = soapEnvelopeOpen +
	`<cwmp:FactoryReset/>` + soapEnvelopeClose

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

