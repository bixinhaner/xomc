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

// Template data types
type InformResponseData struct {
	ID string
}

type GetParameterValuesData struct {
	ID     string
	Params []ParameterNameData
}

type ParameterNameData struct {
	Name string
}

type SetParameterValuesData struct {
	ID     string
	Key    string
	Params []ParameterSetData
}

type ParameterSetData struct {
	Name  string
	Value string
	Type  string
}

type GetParameterNamesData struct {
	ID        string
	Path      string
	NextLevel bool
}

type AddObjectData struct {
	ID         string
	ObjectName string
	Key        string
}

type DeleteObjectData struct {
	ID         string
	ObjectName string
	Key        string
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
}

type UploadData struct {
	ID           string
	CommandKey   string
	FileType     string
	URL          string
	Username     string
	Password     string
	DelaySeconds int
}

type RebootData struct {
	ID         string
	CommandKey string
}

type ScheduleInformData struct {
	ID           string
	DelaySeconds int
	CommandKey   string
}

type FaultData struct {
	ID          string
	FaultCode   int
	FaultString string
}

type GetParameterAttributesData struct {
	ID     string
	Params []ParameterNameData
}

type SetParameterAttributesData struct {
	ID     string
	Params []SetParameterAttributeData
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
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">{{.ID}}</cwmp:ID>
  </soap:Header>
  <soap:Body>`

const soapEnvelopeClose = `
  </soap:Body>
</soap:Envelope>`

const informResponseXML = soapEnvelopeOpen + `
    <cwmp:InformResponse>
      <MaxEnvelopes>1</MaxEnvelopes>
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

