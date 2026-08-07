package pageconfig

import (
	"fmt"
	"strings"
)

var supportedFormats = map[Domain]map[OutputFormat]struct{}{
	DomainCM:        {FormatCSV: {}},
	DomainPM:        {FormatCSV: {}},
	DomainMR:        {FormatXML: {}},
	DomainLOG:       {FormatTXT: {}, FormatCSV: {}},
	DomainInventory: {FormatCSV: {}},
}

var supportedObjects = map[Domain]map[string]struct{}{
	DomainCM:        {"CP": {}, "EP": {}, "CC": {}, "CE": {}, "COMS": {}},
	DomainPM:        {"PC": {}, "PE": {}},
	DomainMR:        {"MRO": {}, "MRE": {}, "MRS": {}},
	DomainLOG:       {"login": {}, "operation": {}, "login_fix": {}, "operation_fix": {}},
	DomainInventory: {"ENB": {}, "GNB": {}, "GSM": {}, "OMC": {}, "eNB": {}, "gNB": {}},
}

var supportedPeriods = map[Period]struct{}{
	Period15M: {},
	Period60M: {},
	Period24H: {},
	Period7D:  {},
	Period1MO: {},
}

func (c *Catalog) Validate(req ValidateRequest) ValidationResult {
	var result ValidationResult

	if _, ok := supportedFormats[req.Domain]; !ok {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported domain %q", req.Domain))
	} else if req.ProfileKind == "file" && req.Domain == DomainInventory {
		result.Errors = append(result.Errors, "inventory must use inventory profiles, not file profiles")
	}

	if formats, ok := supportedFormats[req.Domain]; ok {
		if _, ok := formats[req.Format]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("format %q is not supported by domain %q", req.Format, req.Domain))
		}
	}

	if _, ok := supportedPeriods[req.Period]; !ok {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported period %q", req.Period))
	}

	if req.CompressionEnabled {
		switch req.CompressionFormat {
		case CompressionZip, CompressionGz:
		default:
			result.Errors = append(result.Errors, fmt.Sprintf("unsupported compression format %q", req.CompressionFormat))
		}
	}

	allowedObjects := supportedObjects[req.Domain]
	for _, object := range req.Objects {
		code := strings.TrimSpace(object.Code)
		if code == "" {
			result.Errors = append(result.Errors, "object code is required")
			continue
		}
		if strings.EqualFold(code, "EGW") || strings.EqualFold(code, "PEGW") {
			result.Errors = append(result.Errors, "EGW/PEGW objects are not supported in current xomc northbound page config")
			continue
		}
		if len(allowedObjects) > 0 {
			if _, ok := allowedObjects[code]; !ok {
				result.Errors = append(result.Errors, fmt.Sprintf("object %q is not supported by domain %q", code, req.Domain))
			}
		}
	}

	for _, key := range req.Fields {
		if strings.TrimSpace(key) == "" {
			result.Errors = append(result.Errors, "field key must not be empty")
			continue
		}
		if !c.HasFieldKey(req.Domain, key) && req.Domain != DomainPM {
			result.Errors = append(result.Errors, fmt.Sprintf("unknown field key %q for domain %q", key, req.Domain))
		}
	}

	result.Valid = len(result.Errors) == 0
	return result
}
