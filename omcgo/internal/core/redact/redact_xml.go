package redact

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// IsSensitivePath reports whether a dotted TR-069 parameter path refers to a
// secret. Unlike IsSensitive (which matches a whole field name), this checks
// the *last* path segment, so "Device.ManagementServer.Password" and
// "InternetGatewayDevice.ManagementServer.ConnectionRequestPassword" are both
// recognised as sensitive.
//
// In addition to the canonical SensitiveKeys list, it treats any segment that
// ends in "Password", "Passwd", "Secret", "PreSharedKey" or "PrivateKey"
// (case-insensitive) as sensitive, since TR-069 data models prefix these with
// a qualifier ("STUNPassword", "ConnectionRequestPassword", "KeyPassphrase",
// "WPAPreSharedKey", ...).
func IsSensitivePath(path string) bool {
	if path == "" {
		return false
	}
	seg := path
	if i := strings.LastIndexByte(path, '.'); i >= 0 {
		seg = path[i+1:]
	}
	if IsSensitive(seg) {
		return true
	}
	low := strings.ToLower(seg)
	for _, suffix := range sensitivePathSuffixes {
		if strings.HasSuffix(low, suffix) {
			return true
		}
	}
	return false
}

// sensitivePathSuffixes are lower-cased suffixes that mark a TR-069 parameter
// path segment as secret regardless of its qualifier prefix.
var sensitivePathSuffixes = []string{
	"password",
	"passwd",
	"passphrase",
	"secret",
	"presharedkey",
	"privatekey",
}

// RedactXML masks credential values inside a SOAP/CWMP XML document so that the
// document can be persisted (e.g. in a TR-069 message trace) without leaking
// CPE passwords.
//
// Two redaction patterns are applied:
//
//  1. The character data of any element whose *local* name is sensitive
//     (e.g. <Password>, <cwmp:Password>) is replaced with MaskString.
//  2. Inside a <ParameterValueStruct> the <Value> is masked whenever the
//     sibling <Name> holds a sensitive dotted path (e.g.
//     Device.ManagementServer.Password). This covers GetParameterValuesResponse
//     and SetParameterValues bodies where the secret lives in <Value>, not in
//     an element named after the secret.
//
// The function never returns an error and never panics: trace capture is a
// diagnostic side-channel and must not affect the main request path. If the
// input is not well-formed XML (or is empty) it is returned unchanged so the
// caller can still store the raw bytes for debugging.
func RedactXML(in string) string {
	if in == "" {
		return in
	}
	dec := xml.NewDecoder(strings.NewReader(in))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)

	// elemStack tracks local element names so we know when char data belongs
	// to a sensitive element.
	var elemStack []string
	// withinParamStruct counts nested ParameterValueStruct depth; >0 means the
	// Name/Value sibling rule is active.
	withinParamStruct := 0
	// pendingNameSensitive marks that the most recent <Name> within the current
	// ParameterValueStruct referred to a secret, so the matching <Value> must be
	// masked.
	var pendingNameSensitive bool

	for {
		tok, err := dec.Token()
		if err != nil {
			// io.EOF or a parse error: flush what we have. On a parse error we
			// fall back to the original input to avoid emitting a partial doc.
			if err.Error() != "EOF" {
				return in
			}
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			local := t.Name.Local
			elemStack = append(elemStack, local)
			if local == "ParameterValueStruct" {
				withinParamStruct++
			}
			if err := enc.EncodeToken(t); err != nil {
				return in
			}

		case xml.EndElement:
			if len(elemStack) > 0 {
				elemStack = elemStack[:len(elemStack)-1]
			}
			if t.Name.Local == "ParameterValueStruct" && withinParamStruct > 0 {
				withinParamStruct--
				pendingNameSensitive = false
			}
			if err := enc.EncodeToken(t); err != nil {
				return in
			}

		case xml.CharData:
			out := t
			if len(elemStack) > 0 {
				cur := elemStack[len(elemStack)-1]
				switch {
				case IsSensitive(cur):
					// Pattern 1: <Password>secret</Password>.
					out = xml.CharData(MaskString(string(bytes.TrimSpace(t))))
				case withinParamStruct > 0 && cur == "Name":
					// Remember whether the named parameter is sensitive so the
					// sibling <Value> can be masked.
					pendingNameSensitive = IsSensitivePath(strings.TrimSpace(string(t)))
				case withinParamStruct > 0 && cur == "Value" && pendingNameSensitive:
					out = xml.CharData(MaskString(string(bytes.TrimSpace(t))))
				}
			}
			if err := enc.EncodeToken(out); err != nil {
				return in
			}

		default:
			if err := enc.EncodeToken(tok); err != nil {
				return in
			}
		}
	}

	if err := enc.Flush(); err != nil {
		return in
	}
	return buf.String()
}
