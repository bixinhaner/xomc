package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/omcgo/omcgo/pkg/tr069"
)

// SanitizeBareAmpersands 把 XML 文本里"裸"的 '&'（不构成合法实体引用的）转义为 '&amp;'，
// 已合法的实体（&amp; &lt; &gt; &quot; &apos; 与数字 &#NN; / &#xHH;）原样保留。
// 用于容忍部分 CPE 固件在 XML 里回传 URL 时未转义 '&'（如 ?fileType=PM&filename=...）的情况——
// encoding/xml 默认严格拒绝裸 '&' 会报 "invalid character entity"，导致整条报文解析失败。
func SanitizeBareAmpersands(b []byte) []byte {
	if !bytes.ContainsRune(b, '&') {
		return b
	}
	out := make([]byte, 0, len(b)+16)
	for i := 0; i < len(b); i++ {
		if b[i] != '&' {
			out = append(out, b[i])
			continue
		}
		if startsWithXMLEntity(b[i+1:]) {
			out = append(out, '&')
		} else {
			out = append(out, "&amp;"...)
		}
	}
	return out
}

// startsWithXMLEntity 判断 '&' 之后的字节是否构成合法 XML 实体引用（到首个 ';' 为止）。
func startsWithXMLEntity(s []byte) bool {
	semi := -1
	for j := 0; j < len(s) && j <= 10; j++ {
		if s[j] == ';' {
			semi = j
			break
		}
	}
	if semi <= 0 {
		return false
	}
	name := s[:semi]
	switch string(name) {
	case "amp", "lt", "gt", "quot", "apos":
		return true
	}
	if name[0] != '#' {
		return false
	}
	digits := name[1:]
	if len(digits) == 0 {
		return false
	}
	if digits[0] == 'x' || digits[0] == 'X' {
		digits = digits[1:]
		if len(digits) == 0 {
			return false
		}
		for _, c := range digits {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// DecodeInform stream-parses a SOAP Inform message and returns the InformMessage and CWMP ID.
func DecodeInform(r io.Reader) (*tr069.InformMessage, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var inform tr069.InformMessage
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		local := se.Name.Local

		// Extract CWMP ID from header
		if local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if local == "Body" {
			inBody = true
			continue
		}

		if inBody && local == "Inform" {
			if err := decoder.DecodeElement(&inform, &se); err != nil {
				return nil, "", fmt.Errorf("decode Inform: %w", err)
			}
			return &inform, cwmpID, nil
		}
	}

	return nil, "", fmt.Errorf("no Inform element found in SOAP body")
}

// DecodeTransferComplete stream-parses a SOAP TransferComplete message.
func DecodeTransferComplete(r io.Reader) (*tr069.TransferComplete, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var tc tr069.TransferComplete
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "TransferComplete" {
			if err := decoder.DecodeElement(&tc, &se); err != nil {
				return nil, "", fmt.Errorf("decode TransferComplete: %w", err)
			}
			return &tc, cwmpID, nil
		}
	}

	return nil, "", fmt.Errorf("no TransferComplete element found in SOAP body")
}

// DecodeGetParameterValuesResponse stream-parses a SOAP GetParameterValuesResponse.
func DecodeGetParameterValuesResponse(r io.Reader) ([]tr069.ParameterValueStruct, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var resp tr069.GetParameterValuesResponse
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "GetParameterValuesResponse" {
			if err := decoder.DecodeElement(&resp, &se); err != nil {
				return nil, "", fmt.Errorf("decode GetParameterValuesResponse: %w", err)
			}
			return resp.ParameterList, cwmpID, nil
		}
	}

	return nil, "", fmt.Errorf("no GetParameterValuesResponse element found in SOAP body")
}

// DecodeSetParameterValuesResponse stream-parses a SOAP SetParameterValuesResponse.
// Returns the status code (0=applied immediately, 1=requires reboot).
func DecodeSetParameterValuesResponse(r io.Reader) (int, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var resp tr069.SetParameterValuesResponse
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return -1, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "SetParameterValuesResponse" {
			if err := decoder.DecodeElement(&resp, &se); err != nil {
				return -1, "", fmt.Errorf("decode SetParameterValuesResponse: %w", err)
			}
			return resp.Status, cwmpID, nil
		}
	}

	return -1, "", fmt.Errorf("no SetParameterValuesResponse element found in SOAP body")
}

// DecodeDownloadResponse stream-parses a SOAP DownloadResponse.
// Returns status (0=complete, 1=in progress) and any fault string.
func DecodeDownloadResponse(r io.Reader) (int, string, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var inBody bool

	type downloadResponse struct {
		Status       int    `xml:"Status"`
		StartTime    string `xml:"StartTime"`
		CompleteTime string `xml:"CompleteTime"`
	}
	var resp downloadResponse

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return -1, "", "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "DownloadResponse" {
			if err := decoder.DecodeElement(&resp, &se); err != nil {
				return -1, "", "", fmt.Errorf("decode DownloadResponse: %w", err)
			}
			return resp.Status, resp.CompleteTime, cwmpID, nil
		}
	}

	return -1, "", "", fmt.Errorf("no DownloadResponse element found in SOAP body")
}

// DecodeAutonomousTransferComplete stream-parses a SOAP AutonomousTransferComplete message.
func DecodeAutonomousTransferComplete(r io.Reader) (*tr069.AutonomousTransferComplete, string, error) {
	// 部分 CPE 固件在 AutonomousTransferComplete 里回传上传 URL 时未对查询串的 '&'
	// （?fileType=PM&filename=...）做 XML 转义，导致 encoding/xml 报 invalid character
	// entity、整条完成通知解析失败、ACS 回 400。先把裸 '&' 修成 '&amp;' 再解析以容忍之。
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, "", fmt.Errorf("read SOAP body: %w", err)
	}
	decoder := xml.NewDecoder(bytes.NewReader(SanitizeBareAmpersands(raw)))

	var cwmpID string
	var atc tr069.AutonomousTransferComplete
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "AutonomousTransferComplete" {
			if err := decoder.DecodeElement(&atc, &se); err != nil {
				return nil, "", fmt.Errorf("decode AutonomousTransferComplete: %w", err)
			}
			return &atc, cwmpID, nil
		}
	}

	return nil, "", fmt.Errorf("no AutonomousTransferComplete element found in SOAP body")
}

// DecodeGetParameterNamesResponse stream-parses a SOAP GetParameterNamesResponse.
func DecodeGetParameterNamesResponse(r io.Reader) ([]tr069.ParameterInfoStruct, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var resp tr069.GetParameterNamesResponse
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "GetParameterNamesResponse" {
			if err := decoder.DecodeElement(&resp, &se); err != nil {
				return nil, "", fmt.Errorf("decode GetParameterNamesResponse: %w", err)
			}
			return resp.ParameterList, cwmpID, nil
		}
	}

	return nil, "", fmt.Errorf("no GetParameterNamesResponse element found in SOAP body")
}

// DecodeAddObjectResponse stream-parses a SOAP AddObjectResponse.
// Returns instanceNumber, status, and cwmpID.
func DecodeAddObjectResponse(r io.Reader) (int, int, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var resp tr069.AddObjectResponse
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, -1, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "AddObjectResponse" {
			if err := decoder.DecodeElement(&resp, &se); err != nil {
				return 0, -1, "", fmt.Errorf("decode AddObjectResponse: %w", err)
			}
			return resp.InstanceNumber, resp.Status, cwmpID, nil
		}
	}

	return 0, -1, "", fmt.Errorf("no AddObjectResponse element found in SOAP body")
}

// DecodeDeleteObjectResponse stream-parses a SOAP DeleteObjectResponse.
// Returns status and cwmpID.
func DecodeDeleteObjectResponse(r io.Reader) (int, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var resp tr069.DeleteObjectResponse
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return -1, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "DeleteObjectResponse" {
			if err := decoder.DecodeElement(&resp, &se); err != nil {
				return -1, "", fmt.Errorf("decode DeleteObjectResponse: %w", err)
			}
			return resp.Status, cwmpID, nil
		}
	}

	return -1, "", fmt.Errorf("no DeleteObjectResponse element found in SOAP body")
}

// DecodeUploadResponse stream-parses a SOAP UploadResponse.
// Returns status (0=complete, 1=in progress), completeTime, and cwmpID.
func DecodeUploadResponse(r io.Reader) (int, string, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var inBody bool

	type uploadResponse struct {
		Status       int    `xml:"Status"`
		StartTime    string `xml:"StartTime"`
		CompleteTime string `xml:"CompleteTime"`
	}
	var resp uploadResponse

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return -1, "", "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "UploadResponse" {
			if err := decoder.DecodeElement(&resp, &se); err != nil {
				return -1, "", "", fmt.Errorf("decode UploadResponse: %w", err)
			}
			return resp.Status, resp.CompleteTime, cwmpID, nil
		}
	}

	return -1, "", "", fmt.Errorf("no UploadResponse element found in SOAP body")
}

// DecodeRebootResponse stream-parses a SOAP RebootResponse.
// RebootResponse is empty; this just extracts the cwmpID.
func DecodeRebootResponse(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "RebootResponse" {
			return cwmpID, nil
		}
	}

	return "", fmt.Errorf("no RebootResponse element found in SOAP body")
}

// DecodeFactoryResetResponse stream-parses a SOAP FactoryResetResponse.
// FactoryResetResponse is empty; this just extracts the cwmpID.
func DecodeFactoryResetResponse(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "FactoryResetResponse" {
			return cwmpID, nil
		}
	}

	return "", fmt.Errorf("no FactoryResetResponse element found in SOAP body")
}

// DecodeGetParameterAttributesResponse stream-parses a SOAP GetParameterAttributesResponse.
func DecodeGetParameterAttributesResponse(r io.Reader) ([]tr069.ParameterAttributeStruct, string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var resp tr069.GetParameterAttributesResponse
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "GetParameterAttributesResponse" {
			if err := decoder.DecodeElement(&resp, &se); err != nil {
				return nil, "", fmt.Errorf("decode GetParameterAttributesResponse: %w", err)
			}
			return resp.ParameterList, cwmpID, nil
		}
	}

	return nil, "", fmt.Errorf("no GetParameterAttributesResponse element found in SOAP body")
}

// DecodeSetParameterAttributesResponse stream-parses a SOAP SetParameterAttributesResponse.
// SetParameterAttributesResponse is empty; this just extracts the cwmpID.
func DecodeSetParameterAttributesResponse(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)

	var cwmpID string
	var inBody bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode SOAP: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			continue
		}

		if se.Name.Local == "Body" {
			inBody = true
			continue
		}

		if inBody && se.Name.Local == "SetParameterAttributesResponse" {
			return cwmpID, nil
		}
	}

	return "", fmt.Errorf("no SetParameterAttributesResponse element found in SOAP body")
}

// DetectMethod reads just enough of the SOAP body to determine the RPC method name.
func DetectMethod(r io.Reader) (RPCMethod, string, []byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", "", nil, fmt.Errorf("read request body: %w", err)
	}

	// Empty body = empty POST (session continuation)
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return "", "", data, nil
	}

	method := DetectRPCMethod(data)

	// Extract CWMP ID
	cwmpID := ""
	decoder := xml.NewDecoder(strings.NewReader(trimmed))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local == "ID" {
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				cwmpID = id
			}
			break
		}
	}

	return method, cwmpID, data, nil
}

// HeaderInfo contains extracted SOAP header information.
type HeaderInfo struct {
	ID             string
	SessionTimeout int
	NoMoreRequests string
}

// ParseHeader extracts header information from a SOAP message.
// Returns HeaderInfo with ID, SessionTimeout, and NoMoreRequests if present.
func ParseHeader(r io.Reader) (HeaderInfo, error) {
	decoder := xml.NewDecoder(r)
	var info HeaderInfo

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return info, fmt.Errorf("parse SOAP header: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "ID":
			var id string
			if err := decoder.DecodeElement(&id, &se); err == nil {
				info.ID = id
			}
		case "SessionTimeout":
			var timeout int
			if err := decoder.DecodeElement(&timeout, &se); err == nil {
				info.SessionTimeout = timeout
			}
		case "NoMoreRequests":
			var nmr string
			if err := decoder.DecodeElement(&nmr, &se); err == nil {
				info.NoMoreRequests = nmr
			}
		case "Body":
			// Stop parsing once we reach the body
			return info, nil
		}
	}

	return info, nil
}
