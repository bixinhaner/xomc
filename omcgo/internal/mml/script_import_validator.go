package mml

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ValidationActor is the authenticated user whose custom-command visibility is
// used for TXT import validation.
type ValidationActor struct {
	Username        string
	UserID          uuid.UUID
	VisibleGroupIDs []uuid.UUID
}

// ValidationCommand is the authoritative command snapshot used during import.
// It intentionally contains only data needed to validate and persist plan rows.
type ValidationCommand struct {
	CommandCode    string
	OperationType  string
	RPCMethod      string
	TargetObject   string
	TargetPaths    []string
	RequireConfirm bool
	Disabled       bool
	ParamRefs      []MMLParamRef
}

// ScriptValidationSummary is persisted with a successful import and lets the
// caller show a concise result without re-walking all plan rows.
type ScriptValidationSummary struct {
	TotalLines   int `json:"total_lines"`
	ValidLines   int `json:"valid_lines"`
	DeviceCount  int `json:"device_count"`
	ErrorCount   int `json:"error_count"`
	WarningCount int `json:"warning_count"`
}

// ScriptValidationResult contains line issues plus immutable execution rows.
// Lines with an error do not produce a plan item; warnings remain executable.
type ScriptValidationResult struct {
	PlanItems []MMLPlanItem           `json:"plan_items"`
	Summary   ScriptValidationSummary `json:"summary"`
	Issues    []ScriptIssue           `json:"issues"`
}

type ScriptImportValidator struct {
	repo ScriptValidationRepository
}

func NewScriptImportValidator(repo ScriptValidationRepository) *ScriptImportValidator {
	return &ScriptImportValidator{repo: repo}
}

func (v *ScriptImportValidator) Validate(ctx context.Context, parsed *ParsedScript, actor ValidationActor) (*ScriptValidationResult, error) {
	if parsed == nil {
		return nil, fmt.Errorf("validate parsed script: parsed script is nil")
	}
	if v == nil || v.repo == nil {
		return nil, fmt.Errorf("validate parsed script: repository is nil")
	}

	codes, sns := parsedScriptKeys(parsed.Lines)
	commands, err := v.repo.LoadCommandsByCodes(ctx, codes, actor)
	if err != nil {
		return nil, fmt.Errorf("load validation commands: %w", err)
	}
	devices, err := v.repo.LoadDevicesBySNs(ctx, sns)
	if err != nil {
		return nil, fmt.Errorf("load validation devices: %w", err)
	}

	result := &ScriptValidationResult{PlanItems: make([]MMLPlanItem, 0, len(parsed.Lines)), Issues: make([]ScriptIssue, 0)}
	for _, line := range parsed.Lines {
		lineIssues, command, device := validateScriptLine(line, commands, devices)
		result.Issues = append(result.Issues, lineIssues...)
		if !lineHasError(lineIssues) {
			result.PlanItems = append(result.PlanItems, buildValidationPlanItem(line, command))
		}
		_ = device // device is checked in validateScriptLine; retaining it documents the capability boundary.
	}

	result.Summary = summarizeValidation(parsed.Lines, result.PlanItems, result.Issues)
	return result, nil
}

func parsedScriptKeys(lines []ParsedScriptLine) ([]string, []string) {
	codeSeen := make(map[string]struct{}, len(lines))
	snSeen := make(map[string]struct{}, len(lines))
	codes := make([]string, 0, len(lines))
	sns := make([]string, 0, len(lines))
	for _, line := range lines {
		if _, exists := codeSeen[line.CommandCode]; !exists {
			codeSeen[line.CommandCode] = struct{}{}
			codes = append(codes, line.CommandCode)
		}
		if _, exists := snSeen[line.DeviceSN]; !exists {
			snSeen[line.DeviceSN] = struct{}{}
			sns = append(sns, line.DeviceSN)
		}
	}
	return codes, sns
}

func validateScriptLine(line ParsedScriptLine, commands map[string]ValidationCommand, devices map[string]*model.Device) ([]ScriptIssue, ValidationCommand, *model.Device) {
	issues := make([]ScriptIssue, 0, 4)
	command, commandOK := commands[line.CommandCode]
	if !commandOK {
		return append(issues, validationIssue(line, "MML_COMMAND_NOT_FOUND", IssueError, "command_code", "command is not available to the current user")), command, nil
	}
	if command.Disabled {
		issues = append(issues, validationIssue(line, "MML_COMMAND_DISABLED", IssueError, "command_code", "command is disabled"))
	}

	device, deviceOK := devices[line.DeviceSN]
	if !deviceOK || device == nil {
		issues = append(issues, validationIssue(line, "MML_DEVICE_NOT_FOUND", IssueError, "device_sn", "device does not exist"))
	} else if !device.IsOnline {
		issues = append(issues, validationIssue(line, "MML_DEVICE_OFFLINE", IssueWarning, "device_sn", "device is offline"))
	}

	issues = append(issues, validateLineParameters(line, command)...)
	if command.RequireConfirm {
		issues = append(issues, validationIssue(line, "MML_COMMAND_CONFIRM_REQUIRED", IssueWarning, "command_code", "command requires execution confirmation"))
	}
	return issues, command, device
}

func validateLineParameters(line ParsedScriptLine, command ValidationCommand) []ScriptIssue {
	refs := make(map[string]MMLParamRef, len(command.ParamRefs))
	issues := make([]ScriptIssue, 0)
	for _, ref := range command.ParamRefs {
		refs[ref.ParamCode] = ref
		if ref.IsRequired && (line.OperationType == "MOD" || line.OperationType == "ADD") {
			if _, provided := line.Parameters[ref.ParamCode]; !provided {
				issues = append(issues, validationIssue(line, "MML_PARAMETER_REQUIRED", IssueError, ref.ParamCode, "required parameter is missing"))
			}
		}
	}
	for code, value := range line.Parameters {
		ref, ok := refs[code]
		if !ok {
			issues = append(issues, validationIssue(line, "MML_PARAMETER_UNKNOWN", IssueError, code, "parameter is not defined by command"))
			continue
		}
		if (line.OperationType == "MOD" || line.OperationType == "ADD") && !ref.IsWritable {
			issues = append(issues, validationIssue(line, "MML_PARAMETER_READ_ONLY", IssueError, code, "parameter is read-only"))
			continue
		}
		if !validParameterType(value, ref.ValueType) {
			issues = append(issues, validationIssue(line, "MML_PARAMETER_TYPE_INVALID", IssueError, code, "parameter value has an invalid type"))
			continue
		}
		issues = append(issues, validateParameterConstraint(line, code, value, ref.ValueConstraint)...)
	}
	return issues
}

func validParameterType(value, valueType string) bool {
	switch strings.ToLower(valueType) {
	case "", "string", "text":
		return true
	case "boolean", "bool":
		return strings.EqualFold(value, "true") || strings.EqualFold(value, "false") || value == "0" || value == "1"
	case "integer", "int", "int32", "int64", "uint", "uint32", "uint64":
		_, err := strconv.ParseInt(value, 10, 64)
		return err == nil
	case "number", "float", "float32", "float64", "decimal":
		_, err := strconv.ParseFloat(value, 64)
		return err == nil
	default:
		return true
	}
}

func validateParameterConstraint(line ParsedScriptLine, code, value string, constraint map[string]interface{}) []ScriptIssue {
	if len(constraint) == 0 {
		return nil
	}
	issues := make([]ScriptIssue, 0, 2)
	if values, ok := constraint["enum"]; ok && !constraintContains(values, value) {
		issues = append(issues, validationIssue(line, "MML_PARAMETER_ENUM_INVALID", IssueError, code, "parameter value is outside the allowed enum"))
	}
	if pattern, ok := constraint["regex"].(string); ok && pattern != "" {
		if re, err := regexp.Compile(pattern); err != nil || !re.MatchString(value) {
			issues = append(issues, validationIssue(line, "MML_PARAMETER_REGEX_INVALID", IssueError, code, "parameter value does not match the required pattern"))
		}
	}
	if min, max, constrained := numericBounds(constraint); constrained {
		if numeric, err := strconv.ParseFloat(value, 64); err == nil && (min != nil && numeric < *min || max != nil && numeric > *max) {
			issues = append(issues, validationIssue(line, "MML_PARAMETER_RANGE_INVALID", IssueError, code, "parameter value is outside the allowed range"))
		}
	}
	return issues
}

func constraintContains(values interface{}, value string) bool {
	switch typed := values.(type) {
	case []string:
		for _, candidate := range typed {
			if candidate == value {
				return true
			}
		}
	case []interface{}:
		for _, candidate := range typed {
			if fmt.Sprint(candidate) == value {
				return true
			}
		}
	}
	return false
}

func numericBounds(constraint map[string]interface{}) (min, max *float64, constrained bool) {
	if value, ok := constraint["min"]; ok {
		if number, ok := constraintNumber(value); ok {
			min = &number
			constrained = true
		}
	}
	if value, ok := constraint["max"]; ok {
		if number, ok := constraintNumber(value); ok {
			max = &number
			constrained = true
		}
	}
	return min, max, constrained
}

func constraintNumber(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case string:
		number, err := strconv.ParseFloat(typed, 64)
		return number, err == nil
	default:
		return 0, false
	}
}

func buildValidationPlanItem(line ParsedScriptLine, command ValidationCommand) MMLPlanItem {
	return MMLPlanItem{
		LineNo: line.LineNo, DeviceSN: line.DeviceSN, Order: line.Order, RawLine: line.RawLine,
		Command: map[string]interface{}{
			"command_code": command.CommandCode, "operation_type": command.OperationType,
			"rpc_method": command.RPCMethod, "target_object": command.TargetObject,
			"target_paths": command.TargetPaths,
			"parameters":   stringMapToAny(line.Parameters), "param_refs": command.ParamRefs,
		},
	}
}

func stringMapToAny(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func validationIssue(line ParsedScriptLine, code string, severity IssueSeverity, field, message string) ScriptIssue {
	return ScriptIssue{Code: code, Severity: severity, LineNo: line.LineNo, RawLine: line.RawLine, Field: field, Message: message}
}

func lineHasError(issues []ScriptIssue) bool {
	for _, issue := range issues {
		if issue.Severity == IssueError {
			return true
		}
	}
	return false
}

func summarizeValidation(lines []ParsedScriptLine, planItems []MMLPlanItem, issues []ScriptIssue) ScriptValidationSummary {
	devices := make(map[string]struct{}, len(lines))
	summary := ScriptValidationSummary{TotalLines: len(lines), ValidLines: len(planItems)}
	for _, line := range lines {
		devices[line.DeviceSN] = struct{}{}
	}
	summary.DeviceCount = len(devices)
	for _, issue := range issues {
		if issue.Severity == IssueError {
			summary.ErrorCount++
		} else if issue.Severity == IssueWarning {
			summary.WarningCount++
		}
	}
	return summary
}
