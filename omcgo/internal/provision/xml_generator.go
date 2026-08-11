package provision

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ResolvedParameter is the only parameter shape consumed by the automatic
// start XML renderer. Business form keys and spreadsheet headers must be
// resolved and validated before they reach this layer.
type ResolvedParameter struct {
	ParameterID    string `json:"parameter_id"`
	TRPath         string `json:"trpath"`
	Value          string `json:"value"`
	Source         string `json:"source,omitempty"`
	SourceLocation string `json:"source_location,omitempty"`
}

// AutoStartXMLDocument contains the CMCC automatic-start file metadata and its
// already-resolved TR-069 parameters.
type AutoStartXMLDocument struct {
	NetworkType      string
	Vendor           string
	SerialNumber     string
	GeneratedAt      time.Time
	DataModelVersion string
	Parameters       []ResolvedParameter
	VendorSpecific   string
}

type autoConfigFileXML struct {
	XMLName        xml.Name             `xml:"autoConfigFile"`
	GenerateTime   string               `xml:"generateTime,attr"`
	NetworkType    string               `xml:"networkType,attr"`
	SerialNumber   string               `xml:"serialNumber,attr"`
	Vendor         string               `xml:"vendor,attr"`
	DataModel      dataModelSpecificXML `xml:"dataModelSpecific"`
	VendorSpecific string               `xml:"vendorSpecific"`
}

type dataModelSpecificXML struct {
	Version string      `xml:"version,attr"`
	Config  []configXML `xml:"config"`
}

type configXML struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type legacyAutoStartXML struct {
	XMLName xml.Name         `xml:"auto_start"`
	Params  []legacyParamXML `xml:"param"`
}

type legacyParamXML struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

// GenerateAutoStartXML renders the CMCC autoConfigFile contract. The output is
// deterministic for a fixed GeneratedAt and input parameter set.
func GenerateAutoStartXML(doc AutoStartXMLDocument) ([]byte, error) {
	doc.NetworkType = strings.TrimSpace(doc.NetworkType)
	doc.Vendor = strings.TrimSpace(doc.Vendor)
	doc.SerialNumber = strings.TrimSpace(doc.SerialNumber)
	doc.DataModelVersion = strings.TrimSpace(doc.DataModelVersion)
	if doc.NetworkType != "NR" && doc.NetworkType != "LTE" && doc.NetworkType != "GSM" && doc.NetworkType != "LTENR" {
		return nil, fmt.Errorf("unsupported automatic-start network type %q", doc.NetworkType)
	}
	legacyFormat := doc.NetworkType == "LTE" || doc.NetworkType == "GSM"
	if !legacyFormat {
		if doc.Vendor == "" {
			return nil, fmt.Errorf("device Manufacturer is required")
		}
		if doc.SerialNumber == "" {
			return nil, fmt.Errorf("device serial number is required")
		}
		if doc.DataModelVersion == "" {
			return nil, fmt.Errorf("data model version is required")
		}
		if doc.GeneratedAt.IsZero() {
			return nil, fmt.Errorf("generation time is required")
		}
	}

	params := append([]ResolvedParameter(nil), doc.Parameters...)
	sort.Slice(params, func(i, j int) bool { return params[i].TRPath < params[j].TRPath })
	configs := make([]configXML, 0, len(params))
	seen := make(map[string]string, len(params))
	for _, param := range params {
		path := strings.TrimSpace(param.TRPath)
		if path == "" {
			return nil, fmt.Errorf("parameter %q has an empty TR path", param.ParameterID)
		}
		if strings.Contains(path, "{i}") {
			return nil, fmt.Errorf("parameter %q contains unresolved instance placeholder: %s", param.ParameterID, path)
		}
		if previous, exists := seen[path]; exists {
			if previous != param.Value {
				return nil, fmt.Errorf("conflicting values for TR path %s", path)
			}
			continue
		}
		seen[path] = param.Value
		configs = append(configs, configXML{Name: path, Value: param.Value})
	}
	if legacyFormat {
		legacyParams := make([]legacyParamXML, 0, len(configs))
		for _, config := range configs {
			legacyParams = append(legacyParams, legacyParamXML{Name: config.Name, Value: config.Value})
		}
		return encodeLegacyAutoStartXML(legacyAutoStartXML{Params: legacyParams})
	}

	payload := autoConfigFileXML{
		GenerateTime: doc.GeneratedAt.UTC().Format("2006-01-02T15:04:05.000"),
		NetworkType:  doc.NetworkType,
		SerialNumber: doc.SerialNumber,
		Vendor:       doc.Vendor,
		DataModel: dataModelSpecificXML{
			Version: doc.DataModelVersion,
			Config:  configs,
		},
		VendorSpecific: doc.VendorSpecific,
	}
	var body bytes.Buffer
	body.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>`)
	body.WriteByte('\n')
	encoder := xml.NewEncoder(&body)
	encoder.Indent("", "    ")
	if err := encoder.Encode(payload); err != nil {
		return nil, fmt.Errorf("encode automatic-start XML: %w", err)
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("flush automatic-start XML: %w", err)
	}
	return body.Bytes(), nil
}

func encodeLegacyAutoStartXML(payload legacyAutoStartXML) ([]byte, error) {
	var body bytes.Buffer
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	body.WriteString("\n\n")
	encoder := xml.NewEncoder(&body)
	encoder.Indent("", "    ")
	if err := encoder.Encode(payload); err != nil {
		return nil, fmt.Errorf("encode legacy automatic-start XML: %w", err)
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("flush legacy automatic-start XML: %w", err)
	}
	return body.Bytes(), nil
}
