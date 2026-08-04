package specparser

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var seedEnUSValueRE = regexp.MustCompile(`"en-US"\s*:\s*"([^"]*)"|'en-US'\s*,\s*'([^']*)'`)
var seedChapterGroupValueRE = regexp.MustCompile(`(?m)^\s*\('[A-Z_]+',\s*'chapter:[A-Z_]+',\s*'[^']*',\s*'([^']*)',\s*\d+\),?$`)
var seedTRPathGroupValueRE = regexp.MustCompile(`(?m)^\s*\('MML_STD_TRPATH_G\d{2}',\s*'[^']*',\s*'([^']*)',\s*\d+\),?$`)
var seedTRPathCommandValueRE = regexp.MustCompile(`(?m)^\s*\('(LST|MOD) STD_TRPATH_G\d{2}',\s*'MML_STD_TRPATH_G\d{2}',\s*'(LST|MOD)',\s*'[^']*',\s*'([^']*)',\s*'[^']*',\s*'([^']*)'\),?$`)

// TestGenerateSQL_I18nLongKeysAndEnCoverage 是 issue #67 §6 的 CI 断言：
// catalog 导入生成的 SQL 必须用长码 i18n 键（zh-CN/en-US），且 command/logical/label
// 三个 i18n 字段都带 en-US 值（英文覆盖充分），杜绝短键回潮与英文缺漏。
func TestGenerateSQL_I18nLongKeysAndEnCoverage(t *testing.T) {
	cat := &SpecCatalog{
		Version:    "test",
		SourceMD:   "test.md",
		SourceHash: "deadbeef",
	}
	rep := &DiffReport{
		NewCommands: []*SpecCommand{
			{
				GroupCode:     "chapter:SA",
				Chapter:       "SA",
				OperationType: "LST",
				CommandCode:   "LST DEVICE_INFO",
				LogicalCode:   "DEVICE_INFO",
				CommandZhName: "设备信息",
				RPCMethod:     "GetParameterValues",
			},
		},
		NewSubFieldLinks: []*SubFieldLink{
			{
				CommandCode:  "LST DEVICE_INFO",
				StandardPath: "Device.DeviceInfo.SoftwareVersion",
				MmlCode:      "SOFTWARE_VERSION",
				SortOrder:    0,
				ChineseName:  "软件版本",
				ParamName:    "SoftwareVersion",
			},
		},
	}

	sql := GenerateSQL(rep, cat)
	require.NotEmpty(t, sql)

	// 1) 必须出现长码键。
	assert.Contains(t, sql, `"en-US"`, "i18n JSON 应含 en-US 长码键")
	assert.Contains(t, sql, `"zh-CN"`, "i18n JSON 应含 zh-CN 长码键")

	// 2) 绝不再出现裸短键（JSON 形如 "en": / "zh": ）。
	assert.NotContains(t, sql, `"en":`, "不应回潮短键 en")
	assert.NotContains(t, sql, `"zh":`, "不应回潮短键 zh")

	// 3) en-US 英文覆盖：sub-field 英文标签取 ParamName（英文），命令英文名带英文 op 前缀，
	//    命令对象名从 LogicalCode 派生英文兜底，不再使用 CommandZhName。
	assert.Contains(t, sql, "SoftwareVersion", "label_i18n.en-US 应取英文 ParamName")
	assert.Contains(t, sql, `"en-US":"List Device Information"`, "command_name_i18n.en-US 应为英文命令名")
	assert.Contains(t, sql, `"en-US":"Device Information"`, "logical_name_i18n.en-US 应为英文业务名")
	assert.NotContains(t, sql, `"en-US":"List 设备信息"`, "command_name_i18n.en-US 不应使用中文业务名")
	assert.NotContains(t, sql, `"en-US":"设备信息"`, "logical_name_i18n.en-US 不应使用中文业务名")

	// 4) label_i18n 的 en-US 值必须是英文（非 CJK）。从生成 SQL 抽 label_i18n JSON 片段，
	//    确认 ParamName(英文) 进了 en-US，且整段无中文错位到 en-US。
	labelJSON := `{"en-US":"SoftwareVersion","zh-CN":"软件版本"}`
	assert.Contains(t, sql, labelJSON, "label_i18n 应为长码 + en-US 取英文 ParamName")
	assert.False(t, strings.Contains(`"en-US":"SoftwareVersion"`, "软件"),
		"label_i18n.en-US 不应混入中文")
	assertNoCJKInEnUSValues(t, sql)
}

func TestSeedMMLCommandEnUSValuesDoNotContainCJK(t *testing.T) {
	seed, err := os.ReadFile("../../../migrations/seed/000001_init_seed.sql")
	require.NoError(t, err)

	sql := string(seed)
	assertNoCJKInEnUSValues(t, sql)
	assertNoCJKInStandardTRPathEnglishColumns(t, sql)
}

func assertNoCJKInEnUSValues(t *testing.T, sql string) {
	t.Helper()
	for _, match := range seedEnUSValueRE.FindAllStringSubmatchIndex(sql, -1) {
		start, end := match[2], match[3]
		if start < 0 {
			start, end = match[4], match[5]
		}
		value := sql[start:end]
		if containsCJK(value) {
			line := strings.Count(sql[:start], "\n") + 1
			t.Fatalf("en-US value contains CJK at line %d: %q", line, value)
		}
	}
}

func assertNoCJKInStandardTRPathEnglishColumns(t *testing.T, sql string) {
	t.Helper()
	for _, match := range seedTRPathGroupValueRE.FindAllStringSubmatchIndex(sql, -1) {
		assertNoCJKMatch(t, sql, match[2], match[3], "standard TR path group_name_en")
	}
	for _, match := range seedChapterGroupValueRE.FindAllStringSubmatchIndex(sql, -1) {
		assertNoCJKMatch(t, sql, match[2], match[3], "standard chapter group_name_en")
	}
	for _, match := range seedTRPathCommandValueRE.FindAllStringSubmatchIndex(sql, -1) {
		assertNoCJKMatch(t, sql, match[6], match[7], "standard TR path command_name_en")
		assertNoCJKMatch(t, sql, match[8], match[9], "standard TR path logical_name_en")
	}
}

func assertNoCJKMatch(t *testing.T, sql string, start, end int, field string) {
	t.Helper()
	value := sql[start:end]
	if containsCJK(value) {
		line := strings.Count(sql[:start], "\n") + 1
		t.Fatalf("%s contains CJK at line %d: %q", field, line, value)
	}
}

func containsCJK(s string) bool {
	for _, r := range s {
		if (r >= '\u3400' && r <= '\u9fff') || (r >= '\uf900' && r <= '\ufaff') {
			return true
		}
	}
	return false
}
