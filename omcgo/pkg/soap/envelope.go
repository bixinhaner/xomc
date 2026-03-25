package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// SOAP/CWMP XML namespace constants.
const (
	NSEnvelope = "http://schemas.xmlsoap.org/soap/envelope/"
	NSCWMP     = "urn:dslforum-org:cwmp-1-0"
)

// Envelope represents a SOAP envelope.
type Envelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Header  Header   `xml:"Header"`
	Body    Body     `xml:"Body"`
}

// Header represents a SOAP header.
type Header struct {
	ID             string `xml:"ID"`
	NoMoreRequests string `xml:"NoMoreRequests,omitempty"`
	SessionTimeout int    `xml:"SessionTimeout,omitempty"`
}

// Body represents a SOAP body containing raw XML for the RPC payload.
type Body struct {
	Content []byte `xml:",innerxml"`
}

// RPCMethod identifies which RPC method is in the SOAP body.
type RPCMethod string

const (
	MethodInform                  RPCMethod = "Inform"
	MethodInformResponse          RPCMethod = "InformResponse"
	MethodGetParameterValues      RPCMethod = "GetParameterValues"
	MethodGetParameterValuesResp  RPCMethod = "GetParameterValuesResponse"
	MethodSetParameterValues      RPCMethod = "SetParameterValues"
	MethodSetParameterValuesResp  RPCMethod = "SetParameterValuesResponse"
	MethodGetParameterNames       RPCMethod = "GetParameterNames"
	MethodGetParameterNamesResp   RPCMethod = "GetParameterNamesResponse"
	MethodAddObject               RPCMethod = "AddObject"
	MethodAddObjectResp           RPCMethod = "AddObjectResponse"
	MethodDeleteObject            RPCMethod = "DeleteObject"
	MethodDeleteObjectResp        RPCMethod = "DeleteObjectResponse"
	MethodDownload                RPCMethod = "Download"
	MethodDownloadResp            RPCMethod = "DownloadResponse"
	MethodUpload                  RPCMethod = "Upload"
	MethodUploadResp              RPCMethod = "UploadResponse"
	MethodReboot                  RPCMethod = "Reboot"
	MethodRebootResp              RPCMethod = "RebootResponse"
	MethodFactoryReset            RPCMethod = "FactoryReset"
	MethodFactoryResetResp        RPCMethod = "FactoryResetResponse"
	MethodScheduleInform          RPCMethod = "ScheduleInform"
	MethodScheduleInformResp      RPCMethod = "ScheduleInformResponse"
	MethodGetParameterAttributes      RPCMethod = "GetParameterAttributes"
	MethodGetParameterAttributesResp  RPCMethod = "GetParameterAttributesResponse"
	MethodSetParameterAttributes      RPCMethod = "SetParameterAttributes"
	MethodSetParameterAttributesResp  RPCMethod = "SetParameterAttributesResponse"
	MethodAutonomousTransferComplete     RPCMethod = "AutonomousTransferComplete"
	MethodAutonomousTransferCompleteResp RPCMethod = "AutonomousTransferCompleteResponse"
	MethodTransferComplete        RPCMethod = "TransferComplete"
	MethodTransferCompleteResp    RPCMethod = "TransferCompleteResponse"
	MethodFault                   RPCMethod = "Fault"
)

// ParseEnvelope parses a SOAP envelope from a reader, extracting the header
// and raw body content for further processing.
func ParseEnvelope(r io.Reader) (*Envelope, error) {
	decoder := xml.NewDecoder(r)
	var env Envelope

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse SOAP envelope: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "Envelope":
			continue
		case "Header":
			if err := decoder.DecodeElement(&env.Header, &se); err != nil {
				return nil, fmt.Errorf("parse SOAP header: %w", err)
			}
		case "Body":
			if err := decoder.DecodeElement(&env.Body, &se); err != nil {
				return nil, fmt.Errorf("parse SOAP body: %w", err)
			}
			return &env, nil
		}
	}

	return &env, nil
}

// DetectRPCMethod identifies the RPC method from raw SOAP body content
// by parsing the XML and extracting the first child element of <SOAP-ENV:Body>.
// Per SOAP/CWMP spec, the RPC method is always the first (and only) child element of Body.
func DetectRPCMethod(bodyContent []byte) RPCMethod {
	decoder := xml.NewDecoder(bytes.NewReader(bodyContent))
	var inBody bool

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody {
			// The first child element of SOAP-ENV:Body is the RPC method.
			// e.g., <cwmp:Inform>, <cwmp:GetParameterValuesResponse>, <cwmp:FactoryReset/>
			return RPCMethod(se.Name.Local)
		}
	}

	return ""
}
