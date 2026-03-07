package soap

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/omcgo/omcgo/pkg/tr069"
)

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
