package mml

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sprint B Q-V3-4 决议：脚本带参 parser fail-fast。
// 任一行语法错误整脚本拒绝，返回 *ScriptParseError 含行号 + 原因。

func TestParseScriptContent_BareCommandCodes(t *testing.T) {
	content := `
# 注释
LST_DEVICE_DEVICEINFO
// JS 风格注释
LST_DEVICE_FAULTMGMT_CURRENTALARM;
`
	lines, err := ParseScriptContent(content)
	require.NoError(t, err)
	require.Len(t, lines, 2)

	assert.Equal(t, "LST_DEVICE_DEVICEINFO", lines[0].CommandCode)
	assert.Empty(t, lines[0].Parameters)
	assert.Equal(t, 3, lines[0].LineNumber, "首条非注释 1-based 行号")

	assert.Equal(t, "LST_DEVICE_FAULTMGMT_CURRENTALARM", lines[1].CommandCode)
	assert.Equal(t, 5, lines[1].LineNumber, "; 后注释行号还是原始行")
}

func TestParseScriptContent_WithParameters(t *testing.T) {
	content := `MOD_DEVICE_DEVICEINFO_ANTENNAINFO Azimuth=180 Downtilt=5`
	lines, err := ParseScriptContent(content)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Equal(t, "MOD_DEVICE_DEVICEINFO_ANTENNAINFO", lines[0].CommandCode)
	assert.Equal(t, map[string]string{"Azimuth": "180", "Downtilt": "5"}, lines[0].Parameters)
}

func TestParseScriptContent_FailFast_InvalidCommandCode(t *testing.T) {
	content := `
LST_OK
mod_lowercase_invalid
LST_VALID2`
	_, err := ParseScriptContent(content)
	require.Error(t, err, "fail-fast：不解析后续行")
	var parseErr *ScriptParseError
	require.ErrorAs(t, err, &parseErr)
	assert.Equal(t, 3, parseErr.LineNumber, "错误行号准确")
	assert.Contains(t, parseErr.Reason, "invalid command_code")
}

func TestParseScriptContent_FailFast_MissingEquals(t *testing.T) {
	content := `MOD_FOO param_without_value other=ok`
	_, err := ParseScriptContent(content)
	require.Error(t, err)
	var parseErr *ScriptParseError
	require.ErrorAs(t, err, &parseErr)
	assert.Equal(t, 1, parseErr.LineNumber)
	assert.Contains(t, parseErr.Reason, "missing '='")
}

func TestParseScriptContent_FailFast_EmptyKey(t *testing.T) {
	content := `MOD_FOO =badvalue`
	_, err := ParseScriptContent(content)
	require.Error(t, err)
	var parseErr *ScriptParseError
	require.ErrorAs(t, err, &parseErr)
	assert.Contains(t, parseErr.Reason, "empty key")
}

func TestParseScriptContent_FailFast_DuplicateKey(t *testing.T) {
	content := `MOD_FOO Azimuth=180 Azimuth=90`
	_, err := ParseScriptContent(content)
	require.Error(t, err)
	var parseErr *ScriptParseError
	require.ErrorAs(t, err, &parseErr)
	assert.Contains(t, parseErr.Reason, "duplicate parameter key")
}

func TestParseScriptContent_AllCommentsAndBlank(t *testing.T) {
	content := `
# 全注释脚本
// 啥也不干
`
	lines, err := ParseScriptContent(content)
	require.NoError(t, err)
	assert.Empty(t, lines, "纯注释/空行 → 解析结果为空但不报错")
}

func TestParseScriptContent_TrailingSemicolonAndInlineComment(t *testing.T) {
	content := `LST_FOO; # 这是行尾注释
MOD_BAR k=v;`
	lines, err := ParseScriptContent(content)
	require.NoError(t, err)
	require.Len(t, lines, 2)
	assert.Equal(t, "LST_FOO", lines[0].CommandCode)
	assert.Equal(t, "MOD_BAR", lines[1].CommandCode)
	assert.Equal(t, "v", lines[1].Parameters["k"])
}

func TestScriptParseError_Format(t *testing.T) {
	e := &ScriptParseError{LineNumber: 7, Raw: "bad line", Reason: "test reason"}
	assert.Contains(t, e.Error(), "line 7")
	assert.Contains(t, e.Error(), "test reason")
	assert.Contains(t, e.Error(), `"bad line"`)
}

// splitScriptLines 兼容 fallback：fail 时返回 nil（老 caller 无错误返回路径）
func TestSplitScriptLines_BackwardCompat(t *testing.T) {
	good := `LST_FOO
MOD_BAR`
	codes := splitScriptLines(good)
	assert.Equal(t, []string{"LST_FOO", "MOD_BAR"}, codes)

	// 坏脚本 → 返 nil（不抛错；fail-fast 由新接口 ParseScriptContent 承担）
	bad := `lowercase_invalid`
	assert.Nil(t, splitScriptLines(bad))
}

// 验证 ScriptParseError 是 errors.As 友好（FE 可识别）
func TestScriptParseError_ErrorsAs(t *testing.T) {
	_, err := ParseScriptContent("Invalid_lowercase")
	require.Error(t, err)
	var pe *ScriptParseError
	assert.True(t, errors.As(err, &pe))
	assert.NotNil(t, pe)
}
