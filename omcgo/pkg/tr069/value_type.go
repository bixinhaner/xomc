package tr069

import "strings"

const signallingTraceEnablePath = "Device.DeviceInfo.SignallingTrace.Enable"

// XSDType maps parameter-model data types to their TR-069 SOAP wire types.
func XSDType(dataType string) string {
	switch canonicalDataType(dataType) {
	case "boolean", "bool":
		return "xsd:boolean"
	case "unsignedint", "unsignedinteger", "u_int", "uint", "uint32", "uint64":
		return "xsd:unsignedInt"
	case "int", "integer", "int32", "int64", "uniqueint":
		return "xsd:int"
	case "date_time", "datetime":
		return "xsd:dateTime"
	case "u_long", "unsignedlong", "ulong":
		return "xsd:unsignedLong"
	case "long":
		return "xsd:long"
	default:
		// STRING / enum / list types are transported as strings.
		return "xsd:string"
	}
}

// XSDTypeForPath applies known device compatibility overrides after resolving
// the data type from the concrete product parameter model.
func XSDTypeForPath(path, dataType string) string {
	if strings.TrimSpace(path) == signallingTraceEnablePath {
		return "xsd:string"
	}
	return XSDType(dataType)
}

// NormalizeValueForPath converts boolean form values to the numeric literals
// accepted by Baicells devices while preserving all other parameter values.
func NormalizeValueForPath(path, value, dataType string) string {
	if strings.TrimSpace(path) == signallingTraceEnablePath {
		dataType = "boolean"
	}
	if canonicalDataType(dataType) != "boolean" && canonicalDataType(dataType) != "bool" {
		return value
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1":
		return "1"
	case "false", "0":
		return "0"
	default:
		return value
	}
}

func canonicalDataType(dataType string) string {
	canonical := strings.ToLower(strings.TrimSpace(dataType))
	if cut := strings.IndexAny(canonical, "([{"); cut >= 0 {
		canonical = canonical[:cut]
	}
	return canonical
}
