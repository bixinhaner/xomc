package tr069

import "fmt"

// CWMP fault codes per TR069 specification (9000-9013).
const (
	FaultMethodNotSupported    = 9000
	FaultRequestDenied         = 9001
	FaultInternalError         = 9002
	FaultInvalidArguments      = 9003
	FaultResourcesExceeded     = 9004
	FaultInvalidParameterName  = 9005
	FaultInvalidParameterType  = 9006
	FaultInvalidParameterValue = 9007
	FaultNotWritable           = 9008
	FaultNotificationRejected  = 9009
	FaultDownloadFailure       = 9010
	FaultUploadFailure         = 9011
	FaultFileTransferAuth      = 9012
	FaultFileTransferProtocol  = 9013
)

// faultMessages maps fault codes to their standard descriptions.
var faultMessages = map[int]string{
	FaultMethodNotSupported:    "Method not supported",
	FaultRequestDenied:         "Request denied",
	FaultInternalError:         "Internal error",
	FaultInvalidArguments:      "Invalid arguments",
	FaultResourcesExceeded:     "Resources exceeded",
	FaultInvalidParameterName:  "Invalid parameter name",
	FaultInvalidParameterType:  "Invalid parameter type",
	FaultInvalidParameterValue: "Invalid parameter value",
	FaultNotWritable:           "Attempt to set a non-writable parameter",
	FaultNotificationRejected:  "Notification request rejected",
	FaultDownloadFailure:       "Download failure",
	FaultUploadFailure:         "Upload failure",
	FaultFileTransferAuth:      "File transfer server authentication failure",
	FaultFileTransferProtocol:  "Unsupported protocol for file transfer",
}

// Fault represents a CWMP fault.
type Fault struct {
	FaultCode   int    `xml:"FaultCode" json:"fault_code"`
	FaultString string `xml:"FaultString" json:"fault_string"`
}

// NewFault creates a new Fault with the given code and optional custom message.
func NewFault(code int, message string) *Fault {
	if message == "" {
		message = faultMessages[code]
	}
	return &Fault{FaultCode: code, FaultString: message}
}

func (f *Fault) Error() string {
	return fmt.Sprintf("CWMP Fault %d: %s", f.FaultCode, f.FaultString)
}
