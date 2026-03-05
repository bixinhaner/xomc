package soap

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
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

// DetectRPCMethod identifies the RPC method from raw SOAP body content.
func DetectRPCMethod(bodyContent []byte) RPCMethod {
	s := string(bodyContent)

	// Check for known method names in the body XML
	methods := []RPCMethod{
		MethodInform,
		MethodTransferComplete,
		MethodGetParameterValuesResp,
		MethodSetParameterValuesResp,
		MethodGetParameterNamesResp,
		MethodAddObjectResp,
		MethodDeleteObjectResp,
		MethodDownloadResp,
		MethodUploadResp,
		MethodRebootResp,
		MethodFactoryResetResp,
		MethodScheduleInformResp,
		MethodFault,
	}

	for _, m := range methods {
		if strings.Contains(s, string(m)) {
			return m
		}
	}

	return ""
}
