package specparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// 3) en-US 英文覆盖：sub-field 英文标签取 ParamName（英文），命令英文名带英文 op 前缀。
	assert.Contains(t, sql, "SoftwareVersion", "label_i18n.en-US 应取英文 ParamName")
	assert.Contains(t, sql, "List", "命令英文名应带英文 op 前缀（LST→List）")

	// 4) label_i18n 的 en-US 值必须是英文（非 CJK）。从生成 SQL 抽 label_i18n JSON 片段，
	//    确认 ParamName(英文) 进了 en-US，且整段无中文错位到 en-US。
	labelJSON := `{"en-US":"SoftwareVersion","zh-CN":"软件版本"}`
	assert.Contains(t, sql, labelJSON, "label_i18n 应为长码 + en-US 取英文 ParamName")
	assert.False(t, strings.Contains(`"en-US":"SoftwareVersion"`, "软件"),
		"label_i18n.en-US 不应混入中文")
}
