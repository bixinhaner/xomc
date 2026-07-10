package mml

import (
	"context"
	"fmt"
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

type fakeScriptValidationRepo struct {
	commands          map[string]ValidationCommand
	devices           map[string]*model.Device
	commandBatchCalls int
	deviceBatchCalls  int
}

func (f *fakeScriptValidationRepo) LoadCommandsByCodes(_ context.Context, _ []string, _ ValidationActor) (map[string]ValidationCommand, error) {
	f.commandBatchCalls++
	return f.commands, nil
}

func (f *fakeScriptValidationRepo) LoadDevicesBySNs(_ context.Context, _ []string) (map[string]*model.Device, error) {
	f.deviceBatchCalls++
	return f.devices, nil
}

func TestScriptImportValidator_BatchesAndRejectsUnknownParameter(t *testing.T) {
	repo := &fakeScriptValidationRepo{
		commands: map[string]ValidationCommand{
			"MOD DEVICE_INFO": {
				CommandCode: "MOD DEVICE_INFO", OperationType: "MOD", RPCMethod: "SetParameterValues",
				ParamRefs: []MMLParamRef{{ParamCode: "USER_LABEL", IsWritable: true, ValueType: "string"}},
			},
		},
		devices: map[string]*model.Device{"SN1": {SerialNumber: "SN1", ProductClass: "PC1", IsOnline: true}},
	}
	parsed, parseIssues := ParseScriptTXT([]byte("MOD DEVICE_INFO:UNKNOWN=A;SN1\n"))
	require.Empty(t, parseIssues)

	result, err := NewScriptImportValidator(repo).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})
	require.NoError(t, err)
	require.Equal(t, 1, repo.commandBatchCalls)
	require.Equal(t, 1, repo.deviceBatchCalls)
	require.Empty(t, result.PlanItems)
	require.Equal(t, "MML_PARAMETER_UNKNOWN", result.Issues[0].Code)
}

func TestScriptImportValidator_ReportsAuthoritativeValidationIssues(t *testing.T) {
	baseCommand := ValidationCommand{
		CommandCode: "MOD DEVICE_INFO", OperationType: "MOD", RPCMethod: "SetParameterValues",
		ParamRefs: []MMLParamRef{
			{ParamCode: "REQUIRED", IsWritable: true, IsRequired: true, ValueType: "string"},
			{ParamCode: "READ_ONLY", IsWritable: false, ValueType: "string"},
			{ParamCode: "COUNT", IsWritable: true, ValueType: "integer", ValueConstraint: map[string]interface{}{"min": 1, "max": 10}},
			{ParamCode: "MODE", IsWritable: true, ValueType: "string", ValueConstraint: map[string]interface{}{"enum": []string{"A", "B"}}},
			{ParamCode: "LABEL", IsWritable: true, ValueType: "string", ValueConstraint: map[string]interface{}{"regex": "^[A-Z]+$"}},
		},
	}

	tests := []struct {
		name     string
		line     string
		commands map[string]ValidationCommand
		devices  map[string]*model.Device
		code     string
		severity IssueSeverity
	}{
		{"unknown command", "LST UNKNOWN;SN1", map[string]ValidationCommand{}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_COMMAND_NOT_FOUND", IssueError},
		{"disabled command", "LST DEVICE_INFO;SN1", map[string]ValidationCommand{"LST DEVICE_INFO": {CommandCode: "LST DEVICE_INFO", Disabled: true}}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_COMMAND_DISABLED", IssueError},
		{"required missing", "MOD DEVICE_INFO:COUNT=2;SN1", map[string]ValidationCommand{"MOD DEVICE_INFO": baseCommand}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_PARAMETER_REQUIRED", IssueError},
		{"read only", "MOD DEVICE_INFO:REQUIRED=X,READ_ONLY=Y;SN1", map[string]ValidationCommand{"MOD DEVICE_INFO": baseCommand}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_PARAMETER_READ_ONLY", IssueError},
		{"type", "MOD DEVICE_INFO:REQUIRED=X,COUNT=not-a-number;SN1", map[string]ValidationCommand{"MOD DEVICE_INFO": baseCommand}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_PARAMETER_TYPE_INVALID", IssueError},
		{"range", "MOD DEVICE_INFO:REQUIRED=X,COUNT=11;SN1", map[string]ValidationCommand{"MOD DEVICE_INFO": baseCommand}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_PARAMETER_RANGE_INVALID", IssueError},
		{"enum", "MOD DEVICE_INFO:REQUIRED=X,MODE=C;SN1", map[string]ValidationCommand{"MOD DEVICE_INFO": baseCommand}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_PARAMETER_ENUM_INVALID", IssueError},
		{"regex", "MOD DEVICE_INFO:REQUIRED=X,LABEL=lower;SN1", map[string]ValidationCommand{"MOD DEVICE_INFO": baseCommand}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_PARAMETER_REGEX_INVALID", IssueError},
		{"device missing", "LST DEVICE_INFO;MISSING", map[string]ValidationCommand{"LST DEVICE_INFO": {CommandCode: "LST DEVICE_INFO", OperationType: "LST"}}, map[string]*model.Device{}, "MML_DEVICE_NOT_FOUND", IssueError},
		{"offline warning", "LST DEVICE_INFO;SN1", map[string]ValidationCommand{"LST DEVICE_INFO": {CommandCode: "LST DEVICE_INFO", OperationType: "LST"}}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: false}}, "MML_DEVICE_OFFLINE", IssueWarning},
		{"confirm warning", "RMV DEVICE_INFO;SN1", map[string]ValidationCommand{"RMV DEVICE_INFO": {CommandCode: "RMV DEVICE_INFO", OperationType: "RMV", RequireConfirm: true}}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_COMMAND_CONFIRM_REQUIRED", IssueWarning},
		{"unsigned int rejects non-numeric", "MOD DEVICE_INFO:REQUIRED=X,COUNT=not-a-number;SN1", map[string]ValidationCommand{"MOD DEVICE_INFO": {CommandCode: "MOD DEVICE_INFO", OperationType: "MOD", ParamRefs: []MMLParamRef{{ParamCode: "REQUIRED", IsWritable: true, IsRequired: true, ValueType: "string"}, {ParamCode: "COUNT", IsWritable: true, ValueType: "unsignedint"}}}}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_PARAMETER_TYPE_INVALID", IssueError},
		{"unsigned int rejects negative", "MOD DEVICE_INFO:REQUIRED=X,COUNT=-1;SN1", map[string]ValidationCommand{"MOD DEVICE_INFO": {CommandCode: "MOD DEVICE_INFO", OperationType: "MOD", ParamRefs: []MMLParamRef{{ParamCode: "REQUIRED", IsWritable: true, IsRequired: true, ValueType: "string"}, {ParamCode: "COUNT", IsWritable: true, ValueType: "unsignedint"}}}}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_PARAMETER_TYPE_INVALID", IssueError},
		{"ambiguous custom code", "LST CUSTOM;SN1", map[string]ValidationCommand{"LST CUSTOM": {CommandCode: "LST CUSTOM", OperationType: "LST", Ambiguous: true}}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_COMMAND_AMBIGUOUS", IssueError},
		{"operation mismatch", "LST CUSTOM;SN1", map[string]ValidationCommand{"LST CUSTOM": {CommandCode: "LST CUSTOM", OperationType: "MOD"}}, map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}}, "MML_COMMAND_OPERATION_MISMATCH", IssueError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, issues := ParseScriptTXT([]byte(tt.line + "\n"))
			require.Empty(t, issues)
			result, err := NewScriptImportValidator(&fakeScriptValidationRepo{commands: tt.commands, devices: tt.devices}).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})
			require.NoError(t, err)
			require.Len(t, result.Issues, 1)
			require.Equal(t, tt.code, result.Issues[0].Code)
			require.Equal(t, tt.severity, result.Issues[0].Severity)
		})
	}
}

func TestStandardValidationRules_ProduceRuntimeRegexAndEnumConstraints(t *testing.T) {
	boolean := standardValidationRules("boolean", nil, nil)
	require.Equal(t, "(?i)^(true|false|0|1)$", boolean.JsRegex)
	require.Equal(t, []interface{}{"true", "false", "0", "1"}, boolean.ValueConstraint["enum"])

	min, max := int64(1), int64(10)
	unsigned := standardValidationRules("unsignedint", &min, &max)
	require.Equal(t, "^[0-9]+$", unsigned.JsRegex)
	require.Equal(t, float64(1), unsigned.ValueConstraint["min"])
	require.Equal(t, float64(10), unsigned.ValueConstraint["max"])
}

func TestScriptImportValidator_ProducesPlanAndUsesOneBatchPerResource(t *testing.T) {
	commands := map[string]ValidationCommand{
		"LST DEVICE_INFO": {CommandCode: "LST DEVICE_INFO", OperationType: "LST", RPCMethod: "GetParameterValues"},
	}
	devices := make(map[string]*model.Device, MaxScriptDevices)
	var script string
	for i := 0; i < MaxScriptDevices; i++ {
		sn := fmt.Sprintf("SN%03d", i)
		devices[sn] = &model.Device{SerialNumber: sn, IsOnline: true}
		for j := 0; j < 10; j++ {
			script += "LST DEVICE_INFO;" + sn + "\n"
		}
	}
	parsed, issues := ParseScriptTXT([]byte(script))
	require.Empty(t, issues)
	require.Len(t, parsed.Lines, MaxScriptLines)
	repo := &fakeScriptValidationRepo{commands: commands, devices: devices}

	result, err := NewScriptImportValidator(repo).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})
	require.NoError(t, err)
	require.Empty(t, result.Issues)
	require.Len(t, result.PlanItems, MaxScriptLines)
	require.Equal(t, 1, repo.commandBatchCalls)
	require.Equal(t, 1, repo.deviceBatchCalls)
	require.Equal(t, "LST DEVICE_INFO", result.PlanItems[0].Command["command_code"])
	require.Equal(t, 10, result.PlanItems[9].Order)
}
