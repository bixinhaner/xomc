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
	raw := []byte("\xef\xbb\xbf# note\r\nLST DEVICE_INFO;SN1\r\nMOD DEVICE_INFO:USER_LABEL=A;SN2\r\nLST DEVICE_INFO;SN2\r\n")

	got, issues := ParseScriptTXT(raw)

	require.Empty(t, issues)
	require.Equal(t, "# note\nLST DEVICE_INFO;SN1\nMOD DEVICE_INFO:USER_LABEL=A;SN2\nLST DEVICE_INFO;SN2\n", got.NormalizedContent)
	require.Len(t, got.SHA256, 64)
	require.Equal(t, []int{1, 1, 2}, []int{got.Lines[0].Order, got.Lines[1].Order, got.Lines[2].Order})
	require.Equal(t, []int{2, 3, 4}, []int{got.Lines[0].LineNo, got.Lines[1].LineNo, got.Lines[2].LineNo})
	require.Equal(t, []string{"LST DEVICE_INFO", "MOD DEVICE_INFO", "LST DEVICE_INFO"}, []string{
		got.Lines[0].CommandCode, got.Lines[1].CommandCode, got.Lines[2].CommandCode,
	})
}

func TestParseScriptTXT_TemplateDocumentsSupportedSyntaxAndExamplesParse(t *testing.T) {
	require.Contains(t, scriptImportTemplateForParserTest, "支持操作")
	require.Contains(t, scriptImportTemplateForParserTest, "LST 查询")
	require.Contains(t, scriptImportTemplateForParserTest, "MOD 修改")
	require.Contains(t, scriptImportTemplateForParserTest, "ADD 新增")
	require.Contains(t, scriptImportTemplateForParserTest, "RMV 删除")
	require.Contains(t, scriptImportTemplateForParserTest, "DEL 删除兼容写法")
	require.Contains(t, scriptImportTemplateForParserTest, "操作 命令编码[:参数名=参数值")

	executableTemplate := strings.ReplaceAll(scriptImportTemplateForParserTest, "DEVICE_SN", "SN-TEMPLATE-1")
	got, issues := ParseScriptTXT([]byte(executableTemplate))

	require.Empty(t, issues)
	require.NotEmpty(t, got.Lines)
	require.Equal(t, []string{"LST", "MOD", "ADD", "RMV", "RMV", "MOD"}, []string{
		got.Lines[0].OperationType,
		got.Lines[1].OperationType,
		got.Lines[2].OperationType,
		got.Lines[3].OperationType,
		got.Lines[4].OperationType,
		got.Lines[5].OperationType,
	})
	require.Equal(t, "DEL ETHERNET_INTERFACE;SN-TEMPLATE-1", got.Lines[4].RawLine)
	require.Equal(t, map[string]string{"DESCRIPTION": "site,a;sector-b", "ALIAS": "{main,backup}"}, got.Lines[5].Parameters)
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
			name:     "multiple serial numbers",
			raw:      []byte("LST DEVICE_INFO;SN1,SN2\n"),
			wantCode: "MML_DEVICE_SN_MULTIPLE",
			lineNo:   1,
			rawLine:  "LST DEVICE_INFO;SN1,SN2",
		},
		{
			name:     "two commands in one physical line",
			raw:      []byte("LST DEVICE_INFO;SN1;MOD DEVICE_INFO;SN1\n"),
			wantCode: "MML_LINE_FORMAT_INVALID",
			lineNo:   1,
			rawLine:  "LST DEVICE_INFO;SN1;MOD DEVICE_INFO;SN1",
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
	got, issues := ParseScriptTXT([]byte(`MOD DEVICE_INFO:DESC="a;b,c",VALUES={x;y,z};SN1` + "\n"))

	require.Empty(t, issues)
	require.Len(t, got.Lines, 1)
	require.Equal(t, "MOD", got.Lines[0].OperationType)
	require.Equal(t, "MOD DEVICE_INFO", got.Lines[0].CommandCode)
	require.Equal(t, "SN1", got.Lines[0].DeviceSN)
	require.Equal(t, map[string]string{"DESC": "a;b,c", "VALUES": "{x;y,z}"}, got.Lines[0].Parameters)
}

func TestParseScriptTXT_DELIsRMVAlias(t *testing.T) {
	got, issues := ParseScriptTXT([]byte("DEL ETHERNET_INTERFACE;SN1\n"))

	require.Empty(t, issues)
	require.Len(t, got.Lines, 1)
	require.Equal(t, "RMV", got.Lines[0].OperationType)
	require.Equal(t, "RMV ETHERNET_INTERFACE", got.Lines[0].CommandCode)
	require.Equal(t, "DEL ETHERNET_INTERFACE;SN1", got.Lines[0].RawLine)
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
