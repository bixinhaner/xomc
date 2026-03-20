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
	ID             string
	CommandKey     string
	FileType       string
	URL            string
	Username       string
	Password       string
	FileSize       int64
	TargetFileName string
	DelaySeconds   int
	NoMoreRequests int // 0=more requests coming, 1=last request
}

type UploadData struct {
	ID             string
	CommandKey     string
	FileType       string
	URL            string
	Username       string
	Password       string
	DelaySeconds   int
	NoMoreRequests int // 0=more requests coming, 1=last request
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

type SetParameterAttributeData struct {
	Name               string
	NotificationChange bool
	Notification       int
	AccessListChange   bool
	AccessList         []string
}

// XML Templates

const soapEnvelopeOpen = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">{{.ID}}</cwmp:ID>{{if .NoMoreRequests}}
    <cwmp:NoMoreRequests>{{.NoMoreRequests}}</cwmp:NoMoreRequests>{{end}}
  </soap:Header>
  <soap:Body>`

const soapEnvelopeClose = `
  </soap:Body>
</soap:Envelope>`

const informResponseXML = soapEnvelopeOpen + `
    <cwmp:InformResponse>
      <MaxEnvelopes>1</MaxEnvelopes>{{if .CurrentTime}}
      <CurrentTime>{{.CurrentTime}}</CurrentTime>{{end}}
    </cwmp:InformResponse>` + soapEnvelopeClose

const getParameterValuesXML = soapEnvelopeOpen + `
    <cwmp:GetParameterValues>
      <ParameterNames soap:arrayType="xsd:string[{{len .Params}}]">
        {{- range .Params}}
        <string>{{.Name}}</string>
        {{- end}}
      </ParameterNames>
    </cwmp:GetParameterValues>` + soapEnvelopeClose

const setParameterValuesXML = soapEnvelopeOpen + `
    <cwmp:SetParameterValues>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[{{len .Params}}]">
        {{- range .Params}}
        <ParameterValueStruct>
          <Name>{{.Name}}</Name>
          <Value xsi:type="{{.Type}}">{{.Value}}</Value>
        </ParameterValueStruct>
        {{- end}}
      </ParameterList>
      <ParameterKey>{{.Key}}</ParameterKey>
    </cwmp:SetParameterValues>` + soapEnvelopeClose

const getParameterNamesXML = soapEnvelopeOpen + `
    <cwmp:GetParameterNames>
      <ParameterPath>{{.Path}}</ParameterPath>
      <NextLevel>{{if .NextLevel}}true{{else}}false{{end}}</NextLevel>
    </cwmp:GetParameterNames>` + soapEnvelopeClose

const addObjectXML = soapEnvelopeOpen + `
    <cwmp:AddObject>
      <ObjectName>{{.ObjectName}}</ObjectName>
      <ParameterKey>{{.Key}}</ParameterKey>
    </cwmp:AddObject>` + soapEnvelopeClose

const deleteObjectXML = soapEnvelopeOpen + `
    <cwmp:DeleteObject>
      <ObjectName>{{.ObjectName}}</ObjectName>
      <ParameterKey>{{.Key}}</ParameterKey>
    </cwmp:DeleteObject>` + soapEnvelopeClose

const downloadXML = soapEnvelopeOpen + `
    <cwmp:Download>
      <CommandKey>{{.CommandKey}}</CommandKey>
      <FileType>{{.FileType}}</FileType>
      <URL>{{.URL}}</URL>
      <Username>{{.Username}}</Username>
      <Password>{{.Password}}</Password>
      <FileSize>{{.FileSize}}</FileSize>
      <TargetFileName>{{.TargetFileName}}</TargetFileName>
      <DelaySeconds>{{.DelaySeconds}}</DelaySeconds>
      <SuccessURL></SuccessURL>
      <FailureURL></FailureURL>
    </cwmp:Download>` + soapEnvelopeClose

const uploadXML = soapEnvelopeOpen + `
    <cwmp:Upload>
      <CommandKey>{{.CommandKey}}</CommandKey>
      <FileType>{{.FileType}}</FileType>
      <URL>{{.URL}}</URL>
      <Username>{{.Username}}</Username>
      <Password>{{.Password}}</Password>
      <DelaySeconds>{{.DelaySeconds}}</DelaySeconds>
    </cwmp:Upload>` + soapEnvelopeClose

const rebootXML = soapEnvelopeOpen + `
    <cwmp:Reboot>
      <CommandKey>{{.CommandKey}}</CommandKey>
    </cwmp:Reboot>` + soapEnvelopeClose

const factoryResetXML = soapEnvelopeOpen + `
    <cwmp:FactoryReset/>` + soapEnvelopeClose

const scheduleInformXML = soapEnvelopeOpen + `
    <cwmp:ScheduleInform>
      <DelaySeconds>{{.DelaySeconds}}</DelaySeconds>
      <CommandKey>{{.CommandKey}}</CommandKey>
    </cwmp:ScheduleInform>` + soapEnvelopeClose

const faultResponseXML = soapEnvelopeOpen + `
    <soap:Fault>
      <faultcode>Client</faultcode>
      <faultstring>CWMP fault</faultstring>
      <detail>
        <cwmp:Fault>
          <FaultCode>{{.FaultCode}}</FaultCode>
          <FaultString>{{.FaultString}}</FaultString>
        </cwmp:Fault>
      </detail>
    </soap:Fault>` + soapEnvelopeClose

const transferCompleteResponseXML = soapEnvelopeOpen + `
    <cwmp:TransferCompleteResponse/>` + soapEnvelopeClose

const autonomousTransferCompleteResponseXML = soapEnvelopeOpen + `
    <cwmp:AutonomousTransferCompleteResponse/>` + soapEnvelopeClose

const getParameterAttributesXML = soapEnvelopeOpen + `
    <cwmp:GetParameterAttributes>
      <ParameterNames soap:arrayType="xsd:string[{{len .Params}}]">
        {{- range .Params}}
        <string>{{.Name}}</string>
        {{- end}}
      </ParameterNames>
    </cwmp:GetParameterAttributes>` + soapEnvelopeClose

const setParameterAttributesXML = soapEnvelopeOpen + `
    <cwmp:SetParameterAttributes>
      <ParameterList soap:arrayType="cwmp:SetParameterAttributesStruct[{{len .Params}}]">
        {{- range .Params}}
        <SetParameterAttributesStruct>
          <Name>{{.Name}}</Name>
          <NotificationChange>{{if .NotificationChange}}true{{else}}false{{end}}</NotificationChange>
          <Notification>{{.Notification}}</Notification>
          <AccessListChange>{{if .AccessListChange}}true{{else}}false{{end}}</AccessListChange>
          <AccessList>
            {{- range .AccessList}}
            <string>{{.}}</string>
            {{- end}}
          </AccessList>
        </SetParameterAttributesStruct>
        {{- end}}
      </ParameterList>
    </cwmp:SetParameterAttributes>` + soapEnvelopeClose

