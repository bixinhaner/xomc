package mml

import (
	_ "embed"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed assets/MMLTemplate.txt
var scriptImportTemplateForParserTest string

func TestParseScriptTXT_DerivesPerDeviceOrder(t *testing.T) {
	raw := []byte("\xef\xbb\xbf# note\r\nLST Device.DeviceInfo.SoftwareVersion;SN1\r\nMOD Device.DeviceInfo.X_VENDOR_Label=A;SN2\r\nLST Device.DeviceInfo.HardwareVersion;SN2\r\n")

	got, issues := ParseScriptTXT(raw)

	require.Empty(t, issues)
	require.Equal(t, "# note\nLST Device.DeviceInfo.SoftwareVersion;SN1\nMOD Device.DeviceInfo.X_VENDOR_Label=A;SN2\nLST Device.DeviceInfo.HardwareVersion;SN2\n", got.NormalizedContent)
	require.Len(t, got.SHA256, 64)
	require.Equal(t, []int{1, 1, 2}, []int{got.Lines[0].Order, got.Lines[1].Order, got.Lines[2].Order})
	require.Equal(t, []int{2, 3, 4}, []int{got.Lines[0].LineNo, got.Lines[1].LineNo, got.Lines[2].LineNo})
	require.Equal(t, []string{"LST PATH", "MOD PATH", "LST PATH"}, []string{
		got.Lines[0].CommandCode, got.Lines[1].CommandCode, got.Lines[2].CommandCode,
	})
}

func TestParseScriptTXT_ExpandsCommaSeparatedDeviceSNs(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"LST Device.DeviceInfo.SoftwareVersion;SN1, SN2",
		"MOD Device.DeviceInfo.X_VENDOR_Label=A;SN1",
	}, "\n") + "\n")

	got, issues := ParseScriptTXT(raw)

	require.Empty(t, issues)
	require.Len(t, got.Lines, 3)
	require.Equal(t, []string{"SN1", "SN2", "SN1"}, []string{
		got.Lines[0].DeviceSN,
		got.Lines[1].DeviceSN,
		got.Lines[2].DeviceSN,
	})
	require.Equal(t, []int{1, 1, 2}, []int{
		got.Lines[0].Order,
		got.Lines[1].Order,
		got.Lines[2].Order,
	})
	require.Equal(t, []int{1, 1, 2}, []int{
		got.Lines[0].LineNo,
		got.Lines[1].LineNo,
		got.Lines[2].LineNo,
	})
	require.Equal(t, "LST Device.DeviceInfo.SoftwareVersion;SN1, SN2", got.Lines[1].RawLine)
}

func TestParseScriptTXT_TemplateDocumentsSupportedSyntaxAndExamplesParse(t *testing.T) {
	require.Contains(t, scriptImportTemplateForParserTest, "本文档为 MML TXT 脚本说明和示例")
	require.Contains(t, scriptImportTemplateForParserTest, "行首 # 表示注释行")
	require.Contains(t, scriptImportTemplateForParserTest, "请删除行首 #，并填写正确的参数、值以及设备SN")
	require.Contains(t, scriptImportTemplateForParserTest, "支持操作")
	require.Contains(t, scriptImportTemplateForParserTest, "LST 查询")
	require.Contains(t, scriptImportTemplateForParserTest, "对象或前缀路径")
	require.Contains(t, scriptImportTemplateForParserTest, "MOD 修改")
	require.Contains(t, scriptImportTemplateForParserTest, "ADD 新增")
	require.Contains(t, scriptImportTemplateForParserTest, "RMV 删除")
	require.NotContains(t, scriptImportTemplateForParserTest, "DEL")
	require.Contains(t, scriptImportTemplateForParserTest, "使用标准 PATH")
	require.Contains(t, scriptImportTemplateForParserTest, "PRIVATE:")
	require.Contains(t, scriptImportTemplateForParserTest, "多个设备SN用英文逗号分隔")
	require.Contains(t, scriptImportTemplateForParserTest, "ADD 后的参数名是新对象内的相对参数名")
	require.NotContains(t, scriptImportTemplateForParserTest, "PATH:")
	require.NotContains(t, scriptImportTemplateForParserTest, "操作 命令编码")
	require.NotContains(t, scriptImportTemplateForParserTest, "兼容命令编码")
	require.NotContains(t, scriptImportTemplateForParserTest, "LST DEVICE_INFO;DEVICE_SN")
	require.Contains(t, scriptImportTemplateForParserTest, "# LST Device.DeviceInfo.SoftwareVersion;DEVICE_SN")

	safeTemplate := strings.ReplaceAll(scriptImportTemplateForParserTest, "DEVICE_SN", "SN-TEMPLATE-1")
	safeTemplate = strings.ReplaceAll(safeTemplate, "SECOND_SN", "SN-TEMPLATE-2")
	got, issues := ParseScriptTXT([]byte(safeTemplate))

	require.Len(t, issues, 1)
	require.Equal(t, "MML_FILE_EMPTY", issues[0].Code)
	require.Empty(t, got.Lines)

	executableTemplate := uncommentTemplateExamples(scriptImportTemplateForParserTest)
	executableTemplate = strings.ReplaceAll(executableTemplate, "DEVICE_SN", "SN-TEMPLATE-1")
	executableTemplate = strings.ReplaceAll(executableTemplate, "SECOND_SN", "SN-TEMPLATE-2")
	got, issues = ParseScriptTXT([]byte(executableTemplate))

	require.Empty(t, issues)
	require.NotEmpty(t, got.Lines)
	require.Equal(t, []string{"LST", "MOD", "ADD", "RMV"}, []string{
		got.Lines[0].OperationType,
		got.Lines[1].OperationType,
		got.Lines[2].OperationType,
		got.Lines[3].OperationType,
	})
	require.Equal(t, "RMV Device.IP.Interface.1.IPv4Address.3.;SN-TEMPLATE-1", got.Lines[3].RawLine)
	require.Equal(t, []string{"Device.IP.Interface.1.IPv4Address."}, got.Lines[2].ParamPaths)
	require.Equal(t, map[string]string{"IPAddress": "192.168.1.10", "SubnetMask": "255.255.255.0"}, got.Lines[2].Parameters)
	require.Equal(t, rawPathModePrivate, got.Lines[4].RawPathMode)
	require.Equal(t, []string{"SN-TEMPLATE-1", "SN-TEMPLATE-2"}, []string{got.Lines[5].DeviceSN, got.Lines[6].DeviceSN})
}

func TestParseScriptTXT_EnglishTemplateDocumentsSupportedSyntaxAndExamplesParse(t *testing.T) {
	template := string(scriptImportTemplateForLocale("en-US"))
	require.Contains(t, template, "MML TXT Script Template")
	require.Contains(t, template, "This document provides MML TXT script instructions and examples")
	require.Contains(t, template, "Lines starting with # are comments")
	require.Contains(t, template, "remove the leading # and fill in the correct parameters, values, and device SNs")
	require.Contains(t, template, "Supported operations")
	require.Contains(t, template, "LST query")
	require.Contains(t, template, "object or prefix path")
	require.Contains(t, template, "MOD modify")
	require.Contains(t, template, "ADD add")
	require.Contains(t, template, "RMV remove")
	require.NotContains(t, template, "DEL")
	require.NotContains(t, template, "支持操作")
	require.Contains(t, template, "Standard PATH")
	require.Contains(t, template, "PRIVATE:")
	require.Contains(t, template, "Separate multiple device SNs with English commas")
	require.Contains(t, template, "ADD follow-up values use relative parameter names")
	require.NotContains(t, template, "PATH:")
	require.NotContains(t, template, "Operation command_code")
	require.NotContains(t, template, "Compatible command-code")
	require.NotContains(t, template, "LST DEVICE_INFO;DEVICE_SN")
	require.Contains(t, template, "# LST Device.DeviceInfo.SoftwareVersion;DEVICE_SN")

	safeTemplate := strings.ReplaceAll(template, "DEVICE_SN", "SN-TEMPLATE-1")
	safeTemplate = strings.ReplaceAll(safeTemplate, "SECOND_SN", "SN-TEMPLATE-2")
	got, issues := ParseScriptTXT([]byte(safeTemplate))

	require.Len(t, issues, 1)
	require.Equal(t, "MML_FILE_EMPTY", issues[0].Code)
	require.Empty(t, got.Lines)

	executableTemplate := uncommentTemplateExamples(template)
	executableTemplate = strings.ReplaceAll(executableTemplate, "DEVICE_SN", "SN-TEMPLATE-1")
	executableTemplate = strings.ReplaceAll(executableTemplate, "SECOND_SN", "SN-TEMPLATE-2")
	got, issues = ParseScriptTXT([]byte(executableTemplate))

	require.Empty(t, issues)
	require.NotEmpty(t, got.Lines)
	require.Equal(t, []string{"LST", "MOD", "ADD", "RMV"}, []string{
		got.Lines[0].OperationType,
		got.Lines[1].OperationType,
		got.Lines[2].OperationType,
		got.Lines[3].OperationType,
	})
	require.Equal(t, rawPathModePrivate, got.Lines[4].RawPathMode)
	require.Equal(t, []string{"SN-TEMPLATE-1", "SN-TEMPLATE-2"}, []string{got.Lines[5].DeviceSN, got.Lines[6].DeviceSN})
}

func uncommentTemplateExamples(template string) string {
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		trimmed := strings.TrimPrefix(line, "# ")
		if !strings.Contains(trimmed, "DEVICE_SN") && !strings.Contains(trimmed, "SECOND_SN") {
			continue
		}
		if strings.HasPrefix(trimmed, "LST ") ||
			strings.HasPrefix(trimmed, "MOD ") ||
			strings.HasPrefix(trimmed, "ADD ") ||
			strings.HasPrefix(trimmed, "RMV ") {
			lines[i] = trimmed
		}
	}
	return strings.Join(lines, "\n")
}

func TestParseScriptTXT_NormalizesCROnlyLineEndings(t *testing.T) {
	got, issues := ParseScriptTXT([]byte("# note\r\rLST DEVICE_INFO;SN1\r"))

	require.Empty(t, issues)
	require.Equal(t, "# note\n\nLST DEVICE_INFO;SN1\n", got.NormalizedContent)
	require.Len(t, got.Lines, 1)
	require.Equal(t, 3, got.Lines[0].LineNo)
}

func TestParseScriptTXT_CanonicalizesTerminalNewlinesForSHA256(t *testing.T) {
	withoutTerminalNewline, withoutIssues := ParseScriptTXT([]byte("LST DEVICE_INFO;SN1"))
	withTerminalNewlines, withIssues := ParseScriptTXT([]byte("LST DEVICE_INFO;SN1\r\n\r\n"))

	require.Empty(t, withoutIssues)
	require.Empty(t, withIssues)
	require.Equal(t, "LST DEVICE_INFO;SN1\n", withoutTerminalNewline.NormalizedContent)
	require.Equal(t, withoutTerminalNewline.NormalizedContent, withTerminalNewlines.NormalizedContent)
	require.Equal(t, withoutTerminalNewline.SHA256, withTerminalNewlines.SHA256)
}

func TestParseScriptTXT_RejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		raw      []byte
		wantCode string
		lineNo   int
		rawLine  string
	}{
		{
			name:     "empty file",
			raw:      []byte("\r\n# comment\n\n"),
			wantCode: "MML_FILE_EMPTY",
		},
		{
			name:     "non utf8",
			raw:      []byte{0xff},
			wantCode: "MML_FILE_ENCODING_INVALID",
		},
		{
			name:     "missing serial number",
			raw:      []byte("LST DEVICE_INFO\n"),
			wantCode: "MML_DEVICE_SN_REQUIRED",
			lineNo:   1,
			rawLine:  "LST DEVICE_INFO",
		},
		{
			name:     "two commands in one physical line",
			raw:      []byte("LST DEVICE_INFO;SN1;MOD DEVICE_INFO;SN1\n"),
			wantCode: "MML_LINE_FORMAT_INVALID",
			lineNo:   1,
			rawLine:  "LST DEVICE_INFO;SN1;MOD DEVICE_INFO;SN1",
		},
		{
			name:     "duplicate serial number in one physical line",
			raw:      []byte("LST Device.DeviceInfo.SoftwareVersion;SN1,SN1\n"),
			wantCode: "MML_DEVICE_SN_DUPLICATE",
			lineNo:   1,
			rawLine:  "LST Device.DeviceInfo.SoftwareVersion;SN1,SN1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, issues := ParseScriptTXT(tt.raw)

			require.NotEmpty(t, issues)
			require.Equal(t, tt.wantCode, issues[0].Code)
			require.Equal(t, tt.lineNo, issues[0].LineNo)
			require.Equal(t, tt.rawLine, issues[0].RawLine)
			if got != nil {
				require.NotEmpty(t, got.NormalizedContent)
			}
		})
	}
}

func TestParseScriptTXT_IgnoresQuotedAndBracedDelimiters(t *testing.T) {
	got, issues := ParseScriptTXT([]byte(`MOD Device.DeviceInfo.Description="a;b,c",Device.DeviceInfo.Values={x;y,z};SN1` + "\n"))

	require.Empty(t, issues)
	require.Len(t, got.Lines, 1)
	require.Equal(t, "MOD", got.Lines[0].OperationType)
	require.Equal(t, "MOD PATH", got.Lines[0].CommandCode)
	require.Equal(t, "SN1", got.Lines[0].DeviceSN)
	require.Equal(t, map[string]string{"Device.DeviceInfo.Description": "a;b,c", "Device.DeviceInfo.Values": "{x;y,z}"}, got.Lines[0].Parameters)
}

func TestParseScriptTXT_ParsesImplicitStandardPathRows(t *testing.T) {
	got, issues := ParseScriptTXT([]byte(strings.Join([]string{
		"LST Device.IP.Interface.1.Enable;SN1",
		"MOD Device.IP.Interface.1.Enable=true,Device.IP.Interface.2.Enable=false;SN1",
		"ADD Device.IP.Interface.1.IPv4Address.:IPAddress=192.168.1.10,SubnetMask=255.255.255.0;SN1",
		"RMV Device.IP.Interface.1.IPv4Address.3.;SN1",
	}, "\n") + "\n"))

	require.Empty(t, issues)
	require.Len(t, got.Lines, 4)
	require.Equal(t, "LST PATH", got.Lines[0].CommandCode)
	require.Equal(t, "MOD PATH", got.Lines[1].CommandCode)
	require.Equal(t, map[string]string{
		"Device.IP.Interface.1.Enable": "true",
		"Device.IP.Interface.2.Enable": "false",
	}, got.Lines[1].Parameters)
	require.Equal(t, "ADD PATH", got.Lines[2].CommandCode)
	require.Equal(t, map[string]string{"IPAddress": "192.168.1.10", "SubnetMask": "255.255.255.0"}, got.Lines[2].Parameters)
	require.Equal(t, "RMV PATH", got.Lines[3].CommandCode)
}

func TestParseScriptTXT_ParsesExplicitPathPrefixForCompatibility(t *testing.T) {
	got, issues := ParseScriptTXT([]byte("LST PATH:Device.IP.Interface.1.Enable;SN1\n"))

	require.Empty(t, issues)
	require.Len(t, got.Lines, 1)
	require.Equal(t, "LST", got.Lines[0].OperationType)
	require.Equal(t, "LST PATH", got.Lines[0].CommandCode)
	require.Equal(t, []string{"Device.IP.Interface.1.Enable"}, got.Lines[0].ParamPaths)
}

func TestParseScriptTXT_ParsesPrivatePathPrefix(t *testing.T) {
	got, issues := ParseScriptTXT([]byte(strings.Join([]string{
		"LST PRIVATE:InternetGatewayDevice.DeviceInfo.SoftwareVersion;SN1",
		"MOD PRIVATE:InternetGatewayDevice.DeviceInfo.X_VENDOR_Label=alpha;SN1",
	}, "\n") + "\n"))

	require.Empty(t, issues)
	require.Len(t, got.Lines, 2)
	require.Equal(t, rawPathModePrivate, got.Lines[0].RawPathMode)
	require.Equal(t, "LST PATH", got.Lines[0].CommandCode)
	require.Equal(t, []string{"InternetGatewayDevice.DeviceInfo.SoftwareVersion"}, got.Lines[0].ParamPaths)
	require.Equal(t, rawPathModePrivate, got.Lines[1].RawPathMode)
	require.Equal(t, map[string]string{"InternetGatewayDevice.DeviceInfo.X_VENDOR_Label": "alpha"}, got.Lines[1].Parameters)
}

func TestParseScriptTXT_RejectsMoreThanMaxLines(t *testing.T) {
	raw := []byte(strings.Repeat("LST DEVICE_INFO;SN1\n", MaxScriptLines+1))

	got, issues := ParseScriptTXT(raw)

	require.NotNil(t, got)
	require.Len(t, got.Lines, MaxScriptLines)
	require.Len(t, issues, 1)
	require.Equal(t, "MML_FILE_TOO_LARGE", issues[0].Code)
	require.Equal(t, MaxScriptLines+1, issues[0].LineNo)
	require.Equal(t, "LST DEVICE_INFO;SN1", issues[0].RawLine)
}

func TestParseScriptTXT_RejectsMoreThanMaxDevices(t *testing.T) {
	var raw strings.Builder
	for index := 1; index <= MaxScriptDevices+1; index++ {
		raw.WriteString("LST DEVICE_INFO;SN" + strconv.Itoa(index) + "\n")
	}

	got, issues := ParseScriptTXT([]byte(raw.String()))

	require.NotNil(t, got)
	require.Len(t, got.Lines, MaxScriptDevices+1)
	require.Len(t, issues, 1)
	require.Equal(t, "MML_FILE_TOO_LARGE", issues[0].Code)
	require.Equal(t, MaxScriptDevices+1, issues[0].LineNo)
	require.Equal(t, "LST DEVICE_INFO;SN201", issues[0].RawLine)
}

func TestParseScriptTXT_BoundsMalformedLineIssues(t *testing.T) {
	raw := []byte(strings.Repeat("x\n", MaxScriptBytes/2))

	got, issues := ParseScriptTXT(raw)

	require.NotNil(t, got)
	require.Len(t, got.Lines, 0)
	require.Len(t, issues, MaxScriptIssues+1)
	require.Equal(t, "MML_FILE_TOO_LARGE", issues[MaxScriptIssues].Code)
	require.Equal(t, MaxScriptIssues+1, issues[MaxScriptIssues].LineNo)
	require.Equal(t, "x", issues[MaxScriptIssues].RawLine)
}

func TestParseScriptTXT_BoundsPhysicalLines(t *testing.T) {
	raw := []byte(strings.TrimSuffix(strings.Repeat("# comment\n", MaxScriptLines+1), "\n"))

	got, issues := ParseScriptTXT(raw)

	require.NotNil(t, got)
	require.Len(t, got.Lines, 0)
	require.Len(t, issues, 1)
	require.Equal(t, "MML_FILE_TOO_LARGE", issues[0].Code)
	require.Equal(t, MaxScriptLines+1, issues[0].LineNo)
	require.Equal(t, "# comment", issues[0].RawLine)
}

func TestParseScriptTXT_DoesNotCountTerminalLFSentinel(t *testing.T) {
	tests := []struct {
		name   string
		row    string
		assert func(t *testing.T, got *ParsedScript, issues []ScriptIssue)
	}{
		{
			name: "valid rows",
			row:  "LST DEVICE_INFO;SN1\n",
			assert: func(t *testing.T, got *ParsedScript, issues []ScriptIssue) {
				require.Empty(t, issues)
				require.Len(t, got.Lines, MaxScriptLines)
			},
		},
		{
			name: "comment rows",
			row:  "# comment\n",
			assert: func(t *testing.T, got *ParsedScript, issues []ScriptIssue) {
				require.Len(t, got.Lines, 0)
				require.Len(t, issues, 1)
				require.Equal(t, "MML_FILE_EMPTY", issues[0].Code)
			},
		},
		{
			name: "malformed rows",
			row:  "x\n",
			assert: func(t *testing.T, got *ParsedScript, issues []ScriptIssue) {
				require.Len(t, got.Lines, 0)
				require.Len(t, issues, MaxScriptIssues+1)
				require.Equal(t, "MML_FILE_TOO_LARGE", issues[MaxScriptIssues].Code)
				require.Equal(t, MaxScriptIssues+1, issues[MaxScriptIssues].LineNo)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, issues := ParseScriptTXT([]byte(strings.Repeat(tt.row, MaxScriptLines)))
			tt.assert(t, got, issues)
		})
	}
}
