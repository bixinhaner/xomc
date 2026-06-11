package indicator

import (
	"strings"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

// data_type.go — issue #67 §4：指标 dataType 受控码 + i18n 标签。
//
// 历史成因：data/indicator-library/*.xml 的 dataType 属性是中文枚举（整数/实数/浮点数）
// 与英文混用（Integer/number/整数n）。本文件把它收敛为受控码（int/real/float），
// Loader 落库前归一化，handler 输出再按 locale 映射为 i18n 标签。
// 受控码集合与 seed/000040（sys_dictionaries type='indicator_data_type'）镜像一致。

// 受控码常量（落库值 = perf_indicators_*.data_type）。
const (
	DataTypeInt   = "int"   // 整数 / Integer / 整数n
	DataTypeReal  = "real"  // 实数
	DataTypeFloat = "float" // 浮点数 / number
)

// dataTypeLabels 是受控码 → {zh-CN, en-US} 标签的进程内映射，
// 与 seed/000040 灌入的字典明细保持一致（前端 data-dictionary UI 走字典，
// 运行期 handler 输出走本映射，避免每请求查库）。
var dataTypeLabels = map[string]map[appcontext.Locale]string{
	DataTypeInt:   {appcontext.LocaleZH: "整数", appcontext.LocaleEN: "Integer"},
	DataTypeReal:  {appcontext.LocaleZH: "实数", appcontext.LocaleEN: "Real"},
	DataTypeFloat: {appcontext.LocaleZH: "浮点数", appcontext.LocaleEN: "Float"},
}

// normalizeDataType 把 XML / 历史枚举值归一化为受控码。
//   - 整数 / 整数n / Integer / int    → int
//   - 实数 / real                     → real
//   - 浮点数 / number / float / double → float
//
// 已是受控码则原样返回；空串返回空串（NULL 落库由调用方 nullIfEmpty 处理）；
// 未识别值原样返回（保守：不丢数据，便于巡检发现新枚举）。
func normalizeDataType(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "":
		return ""
	case "整数", "整数n", "integer", "int":
		return DataTypeInt
	case "实数", "real":
		return DataTypeReal
	case "浮点数", "number", "float", "double":
		return DataTypeFloat
	default:
		return raw
	}
}

// dataTypeLabel 按 locale 返回受控码对应的 i18n 标签。
// 未识别码（normalize 未覆盖的历史脏值）回退原码本身，保证不空白。
func dataTypeLabel(code string, loc appcontext.Locale) string {
	labels, ok := dataTypeLabels[code]
	if !ok {
		return code
	}
	if v, ok := labels[loc]; ok && v != "" {
		return v
	}
	// locale 缺对应语言 → 回退中文（系统主语言）。
	if v, ok := labels[appcontext.LocaleZH]; ok {
		return v
	}
	return code
}

// fillDataTypeLabel 给单条 PerfIndicator 按 locale 填充 DataTypeLabel（issue #67 §4）。
// DataType 为 nil/空 → 不填（前端无可显示标签时回退空）。
func fillDataTypeLabel(ind *PerfIndicator, loc appcontext.Locale) {
	if ind == nil || ind.DataType == nil || *ind.DataType == "" {
		return
	}
	label := dataTypeLabel(*ind.DataType, loc)
	ind.DataTypeLabel = &label
}

// fillDataTypeLabels 批量给列表项填充 DataTypeLabel。
func fillDataTypeLabels(items []IndicatorListItem, loc appcontext.Locale) {
	for i := range items {
		fillDataTypeLabel(&items[i].PerfIndicator, loc)
	}
}
