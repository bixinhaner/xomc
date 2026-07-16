package mml

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

type fakeScriptValidationRepo struct {
	commands          map[string]ValidationCommand
	devices           map[string]*model.Device
	standardPaths     map[string]struct{}
	commandBatchCalls int
	deviceBatchCalls  int
	pathBatchCalls    int
}

func (f *fakeScriptValidationRepo) LoadCommandsByCodes(_ context.Context, _ []string, _ ValidationActor) (map[string]ValidationCommand, error) {
	f.commandBatchCalls++
	return f.commands, nil
}

func (f *fakeScriptValidationRepo) LoadDevicesBySNs(_ context.Context, _ []string) (map[string]*model.Device, error) {
	f.deviceBatchCalls++
	return f.devices, nil
}

func (f *fakeScriptValidationRepo) LoadStandardPathSupport(_ context.Context, lookups []StandardPathLookup) (map[string]bool, error) {
	f.pathBatchCalls++
	result := make(map[string]bool, len(lookups))
	for _, lookup := range lookups {
		result[lookup.key()] = standardPathLookupSupportedBySet(lookup, f.standardPaths)
	}
	return result, nil
}

func legacyParsedScriptForTest(t *testing.T, line string) *ParsedScript {
	t.Helper()
	parsedLine := legacyParsedLineForTest(t, 1, line, 1)
	return &ParsedScript{Lines: []ParsedScriptLine{parsedLine}}
}

func legacyParsedLineForTest(t *testing.T, lineNo int, line string, order int) ParsedScriptLine {
	t.Helper()
	parts := strings.Split(line, ";")
	require.Len(t, parts, 2)
	commandRaw := strings.TrimSpace(parts[0])
	deviceSN := strings.TrimSpace(parts[1])
	firstSpace := strings.IndexAny(commandRaw, " \t")
	require.GreaterOrEqual(t, firstSpace, 0)
	operation := normalizeScriptOperation(strings.ToUpper(strings.TrimSpace(commandRaw[:firstSpace])))
	require.NotEmpty(t, operation)
	commandCode, parameters, err := parseLegacyScriptCommand(operation, strings.TrimSpace(commandRaw[firstSpace:]))
	require.NoError(t, err)
	return ParsedScriptLine{
		LineNo:        lineNo,
		RawLine:       line,
		DeviceSN:      deviceSN,
		Order:         order,
		CommandCode:   commandCode,
		OperationType: operation,
		Parameters:    parameters,
	}
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
	parsed := legacyParsedScriptForTest(t, "MOD DEVICE_INFO:UNKNOWN=A;SN1")

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
			parsed := legacyParsedScriptForTest(t, tt.line)
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
	parsed := &ParsedScript{Lines: make([]ParsedScriptLine, 0, MaxScriptLines)}
	for i := 0; i < MaxScriptDevices; i++ {
		sn := fmt.Sprintf("SN%03d", i)
		devices[sn] = &model.Device{SerialNumber: sn, IsOnline: true}
		for j := 0; j < 10; j++ {
			lineNo := len(parsed.Lines) + 1
			parsed.Lines = append(parsed.Lines, legacyParsedLineForTest(t, lineNo, "LST DEVICE_INFO;"+sn, j+1))
		}
	}
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

func TestScriptImportValidator_ProducesRawPathPlanItemsWithoutCommandLookup(t *testing.T) {
	parsed, issues := ParseScriptTXT([]byte(strings.Join([]string{
		"LST Device.IP.Interface.1.Enable;SN1",
		"MOD Device.IP.Interface.1.Enable=true;SN1",
		"ADD Device.IP.Interface.1.IPv4Address.:IPAddress=192.168.1.10;SN1",
		"RMV Device.IP.Interface.1.IPv4Address.3.;SN1",
	}, "\n") + "\n"))
	require.Empty(t, issues)
	repo := &fakeScriptValidationRepo{
		devices: map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}},
		standardPaths: map[string]struct{}{
			"Device.IP.Interface.{i}.Enable":                    {},
			"Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress": {},
		},
	}

	result, err := NewScriptImportValidator(repo).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})

	require.NoError(t, err)
	require.Empty(t, result.Issues)
	require.Len(t, result.PlanItems, 4)
	require.Equal(t, 0, repo.commandBatchCalls)
	require.Equal(t, 1, repo.pathBatchCalls)
	require.Equal(t, "RAW LST", result.PlanItems[0].Command["command_code"])
	require.Equal(t, []string{"Device.IP.Interface.1.Enable"}, result.PlanItems[0].Command["param_paths"])
	require.Equal(t, "RAW MOD", result.PlanItems[1].Command["command_code"])
	require.Equal(t, map[string]interface{}{"Device.IP.Interface.1.Enable": "true"}, result.PlanItems[1].Command["parameters"])
	require.Equal(t, "RAW ADD", result.PlanItems[2].Command["command_code"])
	require.Equal(t, map[string]interface{}{"IPAddress": "192.168.1.10"}, result.PlanItems[2].Command["parameters"])
	require.Equal(t, "RAW RMV", result.PlanItems[3].Command["command_code"])
}

func TestScriptImportValidator_ValidatesExpandedDeviceSNPlanItems(t *testing.T) {
	parsed, issues := ParseScriptTXT([]byte("LST Device.IP.Interface.1.Enable;SN1,SN2\n"))
	require.Empty(t, issues)
	repo := &fakeScriptValidationRepo{
		devices: map[string]*model.Device{
			"SN1": {SerialNumber: "SN1", IsOnline: true},
			"SN2": {SerialNumber: "SN2", IsOnline: true},
		},
		standardPaths: map[string]struct{}{"Device.IP.Interface.{i}.Enable": {}},
	}

	result, err := NewScriptImportValidator(repo).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})

	require.NoError(t, err)
	require.Empty(t, result.Issues)
	require.Len(t, result.PlanItems, 2)
	require.Equal(t, 0, repo.commandBatchCalls)
	require.Equal(t, 1, repo.deviceBatchCalls)
	require.Equal(t, 1, repo.pathBatchCalls)
	require.Equal(t, []string{"SN1", "SN2"}, []string{result.PlanItems[0].DeviceSN, result.PlanItems[1].DeviceSN})
	require.Equal(t, []int{1, 1}, []int{result.PlanItems[0].Order, result.PlanItems[1].Order})
	require.Equal(t, "RAW LST", result.PlanItems[0].Command["command_code"])
	require.Equal(t, "RAW LST", result.PlanItems[1].Command["command_code"])
}

func TestScriptImportValidator_RejectsRawPathMissingFromStandardParams(t *testing.T) {
	parsed, issues := ParseScriptTXT([]byte("LST Device.NotInStandardParams.1.Enable;SN1\n"))
	require.Empty(t, issues)
	repo := &fakeScriptValidationRepo{
		devices: map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}},
	}

	result, err := NewScriptImportValidator(repo).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})

	require.NoError(t, err)
	require.Empty(t, result.PlanItems)
	require.Len(t, result.Issues, 1)
	require.Equal(t, "MML_PATH_NOT_FOUND", result.Issues[0].Code)
}

func TestScriptImportValidator_PrivateRawPathSkipsStandardParamsLookup(t *testing.T) {
	parsed, issues := ParseScriptTXT([]byte("LST PRIVATE:InternetGatewayDevice.DeviceInfo.X_VENDOR_NotRegistered;SN1\n"))
	require.Empty(t, issues)
	repo := &fakeScriptValidationRepo{
		devices: map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}},
	}

	result, err := NewScriptImportValidator(repo).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})

	require.NoError(t, err)
	require.Empty(t, result.Issues)
	require.Len(t, result.PlanItems, 1)
	require.Equal(t, 0, repo.commandBatchCalls)
	require.Equal(t, 1, repo.deviceBatchCalls)
	require.Equal(t, 0, repo.pathBatchCalls)
	require.Equal(t, rawPathModePrivate, result.PlanItems[0].Command["raw_path_mode"])
	require.Equal(t, []string{"InternetGatewayDevice.DeviceInfo.X_VENDOR_NotRegistered"}, result.PlanItems[0].Command["param_paths"])
}

func TestScriptImportValidator_DoesNotResolveLegacyCommandCodesByDefault(t *testing.T) {
	parsed, issues := ParseScriptTXT([]byte("LST DEVICE_INFO;SN1\n"))
	require.Empty(t, issues)
	repo := &fakeScriptValidationRepo{
		devices: map[string]*model.Device{"SN1": {SerialNumber: "SN1", IsOnline: true}},
		commands: map[string]ValidationCommand{
			"LST DEVICE_INFO": {CommandCode: "LST DEVICE_INFO", OperationType: "LST"},
		},
	}

	result, err := NewScriptImportValidator(repo).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})

	require.NoError(t, err)
	require.Equal(t, 0, repo.commandBatchCalls)
	require.Len(t, result.Issues, 1)
	require.Equal(t, "MML_PATH_INVALID", result.Issues[0].Code)
	require.Empty(t, result.PlanItems)
}
