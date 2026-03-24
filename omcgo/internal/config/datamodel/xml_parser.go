package datamodel

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// --- XML structures matching CPE-exported parameter model ---

// XMLParameterModel is the root element of a CPE-exported parameter model XML.
type XMLParameterModel struct {
	XMLName      xml.Name      `xml:"parameterModel"`
	GenerateTime string        `xml:"generateTime,attr"`
	Vendor       string        `xml:"vendor,attr"`
	NetworkType  string        `xml:"networkType,attr"`
	SerialNumber string        `xml:"serialNumber,attr"`
	ModelVersion string        `xml:"modelVersion,attr"`
	TotalEntries int           `xml:"totalEntries,attr"`
	Objects      XMLObjects    `xml:"objects"`
	Parameters   XMLParameters `xml:"parameters"`
}

// XMLObjects wraps the list of object definitions.
type XMLObjects struct {
	Items []XMLObject `xml:"object"`
}

// XMLObject represents an object node in the parameter model.
type XMLObject struct {
	Name         string `xml:"name,attr"`
	Access       string `xml:"access,attr"`
	MaxInstances string `xml:"maxInstances,attr"`
	IsList       string `xml:"isList,attr"`
}

// XMLParameters wraps the list of parameter definitions.
type XMLParameters struct {
	Items []XMLParam `xml:"param"`
}

// XMLParam represents a single parameter definition in the XML.
type XMLParam struct {
	Name          string `xml:"name,attr"`
	Access        string `xml:"access,attr"`
	Type          string `xml:"type,attr"`
	Min           string `xml:"min,attr"`
	Max           string `xml:"max,attr"`
	Notify        string `xml:"notify,attr"`
	ForcedInform  string `xml:"forcedInform,attr"`
	DefaultValue  string `xml:"defaultValue,attr"`
	ChangeApplies string `xml:"changeApplies,attr"`
	IsList        string `xml:"isList,attr"`
}

// --- Parsed intermediate representation ---

// ParsedParameterModel is the intermediate representation after XML parsing.
type ParsedParameterModel struct {
	Vendor       string       // OUI, e.g. "48BF74"
	NetworkType  string       // "LTE" or "NR"
	SerialNumber string       // Source device serial number
	ModelVersion string       // Model version, e.g. "1.0"
	TotalEntries int          // Declared total entries
	GenerateTime time.Time    // When the model was generated
	Objects      []ObjectInfo // Object node definitions
	Parameters   []Parameter  // Parameter definitions (converted to internal format)
}

// ParseParameterModelXML parses a CPE-exported parameter model XML and returns
// a ParsedParameterModel. It uses streaming XML decoding for memory efficiency.
func ParseParameterModelXML(reader io.Reader) (*ParsedParameterModel, error) {
	var xmlModel XMLParameterModel
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&xmlModel); err != nil {
		return nil, fmt.Errorf("decode parameter model XML: %w", err)
	}

	result := &ParsedParameterModel{
		Vendor:       xmlModel.Vendor,
		NetworkType:  xmlModel.NetworkType,
		SerialNumber: xmlModel.SerialNumber,
		ModelVersion: xmlModel.ModelVersion,
		TotalEntries: xmlModel.TotalEntries,
	}

	// Parse generate time (ISO 8601 with timezone).
	if xmlModel.GenerateTime != "" {
		t, err := time.Parse(time.RFC3339, xmlModel.GenerateTime)
		if err != nil {
			// Try alternate formats.
			t, err = time.Parse("2006-01-02T15:04:05-07:00", xmlModel.GenerateTime)
			if err != nil {
				// Store zero time but don't fail the parse.
				result.GenerateTime = time.Time{}
			} else {
				result.GenerateTime = t
			}
		} else {
			result.GenerateTime = t
		}
	}

	// Convert XML objects to ObjectInfo.
	result.Objects = make([]ObjectInfo, 0, len(xmlModel.Objects.Items))
	for _, obj := range xmlModel.Objects.Items {
		maxInst, _ := strconv.Atoi(obj.MaxInstances)
		result.Objects = append(result.Objects, ObjectInfo{
			Name:         obj.Name,
			Access:       obj.Access,
			MaxInstances: maxInst,
			IsList:       parseBool(obj.IsList),
		})
	}

	// Convert XML params to Parameter.
	result.Parameters = make([]Parameter, 0, len(xmlModel.Parameters.Items))
	for _, p := range xmlModel.Parameters.Items {
		param := convertXMLParam(p)
		result.Parameters = append(result.Parameters, param)
	}

	return result, nil
}

// convertXMLParam converts an XMLParam to the internal Parameter format.
func convertXMLParam(xp XMLParam) Parameter {
	p := Parameter{
		Path:          xp.Name,
		Type:          normalizeParamType(xp.Type),
		Writable:      xp.Access == "READ_WRITE",
		Notify:        xp.Notify,
		ForcedInform:  parseBool(xp.ForcedInform),
		DefaultValue:  xp.DefaultValue,
		ChangeApplies: xp.ChangeApplies,
		IsList:        parseBool(xp.IsList),
	}

	// Build constraints from min/max values.
	p.Constraints = buildConstraints(xp.Type, xp.Min, xp.Max)

	// Auto-detect category from path.
	p.Category = detectCategory(xp.Name)

	return p
}

// normalizeParamType maps XML type names to TR-069 standard type names.
func normalizeParamType(xmlType string) string {
	switch strings.ToUpper(xmlType) {
	case "STRING":
		return "string"
	case "U_INT", "UINT", "UNSIGNED_INT":
		return "unsignedInt"
	case "INT", "INTEGER":
		return "int"
	case "BOOLEAN", "BOOL":
		return "boolean"
	case "DATE_TIME", "DATETIME":
		return "dateTime"
	case "BASE64":
		return "base64"
	case "LONG":
		return "long"
	case "UNSIGNED_LONG", "U_LONG":
		return "unsignedLong"
	default:
		return "string"
	}
}

// buildConstraints creates parameter constraints from min/max XML attributes.
// For string types, min/max represent string length; for numeric types, they represent value range.
func buildConstraints(xmlType, minStr, maxStr string) *Constraints {
	if minStr == "" && maxStr == "" {
		return nil
	}

	c := &Constraints{}
	isString := strings.ToUpper(xmlType) == "STRING"

	if isString {
		// For strings, max is max length.
		if maxStr != "" {
			if maxLen, err := strconv.Atoi(maxStr); err == nil && maxLen > 0 {
				c.MaxLength = maxLen
			}
		}
	} else {
		// For numeric types, min/max are value range.
		if minStr != "" {
			if v, err := strconv.ParseInt(minStr, 10, 64); err == nil {
				c.MinValue = &v
			}
		}
		if maxStr != "" {
			if v, err := strconv.ParseInt(maxStr, 10, 64); err == nil {
				c.MaxValue = &v
			}
		}
	}

	// Return nil if no constraints were actually set.
	if c.MaxLength == 0 && c.MinValue == nil && c.MaxValue == nil {
		return nil
	}
	return c
}

// detectCategory guesses the parameter category from the path.
func detectCategory(path string) string {
	upper := strings.ToUpper(path)
	switch {
	case strings.Contains(upper, "MANAGEMENTSERVER"):
		return "management"
	case strings.Contains(upper, "DEVICEINFO"):
		return "device_info"
	case strings.Contains(upper, "FAPSERVICE") || strings.Contains(upper, "CELLCONFIG"):
		return "radio"
	case strings.Contains(upper, "TIME.") || strings.Contains(upper, "NTP"):
		return "time"
	case strings.Contains(upper, "FAULTMGMT") || strings.Contains(upper, "ALARM"):
		return "alarm"
	case strings.Contains(upper, "PERFMGMT"):
		return "pm"
	case strings.Contains(upper, "TUNNEL") || strings.Contains(upper, "IPSEC"):
		return "transport"
	case strings.Contains(upper, "NEIGHBO"):
		return "neighbor"
	default:
		return ""
	}
}

// parseBool interprets common boolean string representations.
func parseBool(s string) bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// ValidateXMLModel performs basic validation on a parsed parameter model.
func ValidateXMLModel(m *ParsedParameterModel) []string {
	var errs []string

	if m.Vendor == "" {
		errs = append(errs, "vendor (OUI) is required in XML")
	}
	if m.NetworkType == "" {
		errs = append(errs, "networkType is required in XML")
	}
	if len(m.Parameters) == 0 {
		errs = append(errs, "no parameters found in XML")
	}
	if m.TotalEntries > 0 {
		actual := len(m.Objects) + len(m.Parameters)
		if actual == 0 {
			errs = append(errs, "XML declares totalEntries but contains no objects or parameters")
		}
	}

	return errs
}
