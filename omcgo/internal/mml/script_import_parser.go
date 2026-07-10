package mml

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxScriptDevices       = 200
	MaxScriptLines         = 2000
	MaxScriptBytes         = 2 << 20
	MaxScriptPhysicalLines = MaxScriptLines
	MaxScriptIssues        = 100
	ValidationVersion      = "mml-txt-v1"
)

// IssueSeverity is the stable severity vocabulary returned by script import.
type IssueSeverity string

const (
	IssueError   IssueSeverity = "error"
	IssueWarning IssueSeverity = "warning"
)

// ScriptIssue identifies a file-level or physical-line parsing problem.
// File-level issues have a zero LineNo and empty RawLine because no source row
// is available to identify.
type ScriptIssue struct {
	Code     string        `json:"code"`
	Severity IssueSeverity `json:"severity"`
	LineNo   int           `json:"line_no,omitempty"`
	RawLine  string        `json:"raw_line,omitempty"`
	Field    string        `json:"field,omitempty"`
	Message  string        `json:"message,omitempty"`
}

// ParsedScript is the parser's deterministic representation of one TXT file.
type ParsedScript struct {
	NormalizedContent string             `json:"normalized_content"`
	SHA256            string             `json:"sha256"`
	Lines             []ParsedScriptLine `json:"lines"`
}

// ParsedScriptLine is one executable device-bound command from the TXT file.
type ParsedScriptLine struct {
	LineNo        int               `json:"line_no"`
	RawLine       string            `json:"raw_line"`
	DeviceSN      string            `json:"device_sn"`
	Order         int               `json:"order"`
	CommandCode   string            `json:"command_code"`
	OperationType string            `json:"operation_type"`
	Parameters    map[string]string `json:"parameters"`
}

// ParseScriptTXT validates TXT-level syntax without making any external calls.
// It preserves comments and blank lines in NormalizedContent, while Lines holds
// only executable rows. All content hashes are computed from canonical LF text
// with exactly one final newline.
func ParseScriptTXT(raw []byte) (*ParsedScript, []ScriptIssue) {
	if len(raw) > MaxScriptBytes {
		return nil, []ScriptIssue{fileIssue("MML_FILE_TOO_LARGE")}
	}

	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(raw) {
		return nil, []ScriptIssue{fileIssue("MML_FILE_ENCODING_INVALID")}
	}

	normalized := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\r", "\n")
	canonical := strings.TrimRight(normalized, "\n") + "\n"
	digest := sha256.Sum256([]byte(canonical))
	parsedScript := &ParsedScript{
		NormalizedContent: canonical,
		SHA256:            hex.EncodeToString(digest[:]),
		Lines:             make([]ParsedScriptLine, 0),
	}

	orders := make(map[string]int)
	devices := make(map[string]struct{})
	var deviceLimitIssue *ScriptIssue
	issues := make([]ScriptIssue, 0)
	forEachScriptPhysicalLine(normalized, func(lineNo int, physical string) bool {
		if lineNo > MaxScriptPhysicalLines {
			issues = append(issues, lineIssue("MML_FILE_TOO_LARGE", lineNo, physical, "physical line limit exceeded"))
			return false
		}
		parsed, issue := parseScriptPhysicalLine(lineNo, physical)
		if issue != nil {
			if len(issues) >= MaxScriptIssues {
				issues = append(issues, lineIssue("MML_FILE_TOO_LARGE", lineNo, physical, "parser issue limit exceeded"))
				return false
			}
			issues = append(issues, *issue)
			return true
		}
		if parsed == nil {
			return true
		}

		orders[parsed.DeviceSN]++
		parsed.Order = orders[parsed.DeviceSN]
		parsedScript.Lines = append(parsedScript.Lines, *parsed)
		if _, exists := devices[parsed.DeviceSN]; !exists {
			devices[parsed.DeviceSN] = struct{}{}
			if len(devices) > MaxScriptDevices && deviceLimitIssue == nil {
				issue := lineIssue("MML_FILE_TOO_LARGE", parsed.LineNo, parsed.RawLine, "")
				deviceLimitIssue = &issue
			}
		}
		return true
	})

	if len(parsedScript.Lines) == 0 && len(issues) == 0 {
		issues = append(issues, fileIssue("MML_FILE_EMPTY"))
	}
	if len(parsedScript.Lines) > MaxScriptLines {
		line := parsedScript.Lines[MaxScriptLines]
		issues = append(issues, lineIssue("MML_FILE_TOO_LARGE", line.LineNo, line.RawLine, ""))
	}
	if deviceLimitIssue != nil {
		issues = append(issues, *deviceLimitIssue)
	}

	return parsedScript, issues
}

// forEachScriptPhysicalLine avoids allocating a string slice proportional to
// the number of physical rows. A final LF terminates the preceding physical
// row; unlike strings.Split, it does not create a synthetic empty row.
func forEachScriptPhysicalLine(content string, visit func(lineNo int, physical string) bool) {
	for start, lineNo := 0, 1; start < len(content); lineNo++ {
		next := strings.IndexByte(content[start:], '\n')
		if next < 0 {
			visit(lineNo, content[start:])
			return
		}
		next += start
		if !visit(lineNo, content[start:next]) {
			return
		}
		start = next + 1
	}
}

func fileIssue(code string) ScriptIssue {
	return ScriptIssue{Code: code, Severity: IssueError}
}

func lineIssue(code string, lineNo int, rawLine, message string) ScriptIssue {
	return ScriptIssue{Code: code, Severity: IssueError, LineNo: lineNo, RawLine: rawLine, Message: message}
}

func parseScriptPhysicalLine(lineNo int, rawLine string) (*ParsedScriptLine, *ScriptIssue) {
	trimmed := strings.TrimSpace(rawLine)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil, nil
	}

	semicolonIndexes, err := topLevelIndexes(trimmed, ';')
	if err != nil {
		issue := lineIssue("MML_LINE_FORMAT_INVALID", lineNo, rawLine, err.Error())
		return nil, &issue
	}
	if len(semicolonIndexes) == 0 {
		issue := lineIssue("MML_DEVICE_SN_REQUIRED", lineNo, rawLine, "command must end with ;SN")
		return nil, &issue
	}
	if len(semicolonIndexes) != 1 {
		issue := lineIssue("MML_LINE_FORMAT_INVALID", lineNo, rawLine, "a physical line may contain only one command")
		return nil, &issue
	}

	separator := semicolonIndexes[0]
	commandRaw := strings.TrimSpace(trimmed[:separator])
	deviceSN := strings.TrimSpace(trimmed[separator+1:])
	if deviceSN == "" {
		issue := lineIssue("MML_DEVICE_SN_REQUIRED", lineNo, rawLine, "serial number is required")
		return nil, &issue
	}
	if strings.Contains(deviceSN, ",") {
		issue := lineIssue("MML_DEVICE_SN_MULTIPLE", lineNo, rawLine, "a physical line may name only one serial number")
		return nil, &issue
	}
	if commandRaw == "" {
		issue := lineIssue("MML_LINE_FORMAT_INVALID", lineNo, rawLine, "command is required before ;SN")
		return nil, &issue
	}

	operation, commandCode, parameters, err := parseScriptCommand(commandRaw)
	if err != nil {
		issue := lineIssue("MML_LINE_FORMAT_INVALID", lineNo, rawLine, err.Error())
		return nil, &issue
	}
	return &ParsedScriptLine{
		LineNo:        lineNo,
		RawLine:       rawLine,
		DeviceSN:      deviceSN,
		CommandCode:   commandCode,
		OperationType: operation,
		Parameters:    parameters,
	}, nil
}

func parseScriptCommand(raw string) (operation, commandCode string, parameters map[string]string, err error) {
	firstSpace := -1
	for index, r := range raw {
		if unicode.IsSpace(r) {
			firstSpace = index
			break
		}
	}
	if firstSpace < 0 {
		return "", "", nil, fmt.Errorf("operation and command code must be separated by whitespace")
	}

	operation = strings.ToUpper(strings.TrimSpace(raw[:firstSpace]))
	if !validScriptOperation(operation) {
		return "", "", nil, fmt.Errorf("unsupported operation %q", operation)
	}
	rest := strings.TrimSpace(raw[firstSpace:])
	if rest == "" {
		return "", "", nil, fmt.Errorf("command code is required")
	}

	colonIndex, err := firstTopLevelIndex(rest, ':')
	if err != nil {
		return "", "", nil, err
	}
	code := rest
	parametersRaw := ""
	if colonIndex >= 0 {
		code = strings.TrimSpace(rest[:colonIndex])
		parametersRaw = strings.TrimSpace(rest[colonIndex+1:])
	}
	if code == "" || len(strings.Fields(code)) != 1 || !isValidCommandCode(strings.ToUpper(code)) {
		return "", "", nil, fmt.Errorf("invalid command code %q", code)
	}

	parameters = make(map[string]string)
	if parametersRaw != "" {
		parameters, err = parseScriptParameters(parametersRaw)
		if err != nil {
			return "", "", nil, err
		}
	}
	return operation, operation + " " + strings.ToUpper(code), parameters, nil
}

func validScriptOperation(operation string) bool {
	switch operation {
	case "LST", "MOD", "ADD", "RMV":
		return true
	default:
		return false
	}
}

func parseScriptParameters(raw string) (map[string]string, error) {
	tokens, err := splitTopLevel(raw, ',')
	if err != nil {
		return nil, err
	}
	parameters := make(map[string]string, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			return nil, fmt.Errorf("parameter token is empty")
		}
		equalsIndex, err := firstTopLevelIndex(token, '=')
		if err != nil {
			return nil, err
		}
		if equalsIndex < 0 {
			return nil, fmt.Errorf("parameter %q is missing =", token)
		}
		key := strings.TrimSpace(token[:equalsIndex])
		if key == "" {
			return nil, fmt.Errorf("parameter key is empty")
		}
		if _, exists := parameters[key]; exists {
			return nil, fmt.Errorf("duplicate parameter key %q", key)
		}
		value, err := normalizeScriptParameterValue(strings.TrimSpace(token[equalsIndex+1:]))
		if err != nil {
			return nil, err
		}
		parameters[key] = value
	}
	return parameters, nil
}

func normalizeScriptParameterValue(value string) (string, error) {
	if len(value) < 2 {
		return value, nil
	}
	quote := value[0]
	if quote != '\'' && quote != '"' {
		return value, nil
	}
	if value[len(value)-1] != quote {
		return "", fmt.Errorf("unterminated quoted parameter value")
	}
	return strings.ReplaceAll(value[1:len(value)-1], "\\"+string(quote), string(quote)), nil
}

func splitTopLevel(raw string, separator rune) ([]string, error) {
	indexes, err := topLevelIndexes(raw, separator)
	if err != nil {
		return nil, err
	}
	parts := make([]string, 0, len(indexes)+1)
	start := 0
	for _, index := range indexes {
		parts = append(parts, raw[start:index])
		start = index + len(string(separator))
	}
	return append(parts, raw[start:]), nil
}

func firstTopLevelIndex(raw string, separator rune) (int, error) {
	indexes, err := topLevelIndexes(raw, separator)
	if err != nil || len(indexes) == 0 {
		return -1, err
	}
	return indexes[0], nil
}

// topLevelIndexes finds separators outside both quote styles and balanced
// braces. Backslash escapes are honored inside quoted strings.
func topLevelIndexes(raw string, separator rune) ([]int, error) {
	indexes := make([]int, 0)
	var quote rune
	escaped := false
	braceDepth := 0
	for index, r := range raw {
		switch {
		case quote != 0:
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '{':
			braceDepth++
		case r == '}':
			if braceDepth == 0 {
				return nil, fmt.Errorf("unmatched closing brace")
			}
			braceDepth--
		case r == separator && braceDepth == 0:
			indexes = append(indexes, index)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted value")
	}
	if braceDepth != 0 {
		return nil, fmt.Errorf("unterminated brace value")
	}
	return indexes, nil
}
