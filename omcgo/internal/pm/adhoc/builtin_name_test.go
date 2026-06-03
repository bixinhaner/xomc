package adhoc

import (
	"testing"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func TestLocalizeTaskName(t *testing.T) {
	cases := []struct {
		name       string
		loc        appcontext.Locale
		rawName    string
		isBuiltin  bool
		dim        Dimension
		technology string
		want       string
	}{
		// 内置任务 + 英文：按结构字段组装，覆盖全部 6 个维度映射。
		{"en builtin network lte", appcontext.LocaleEN, "内置-全网-LTE", true, DimensionNetwork, "lte", "Built-in-Network-LTE"},
		{"en builtin device_group nr", appcontext.LocaleEN, "内置-设备组-NR", true, DimensionDeviceGroup, "nr", "Built-in-Device Group-NR"},
		{"en builtin product gsm", appcontext.LocaleEN, "内置-产品-GSM", true, DimensionProduct, "gsm", "Built-in-Product-GSM"},
		{"en builtin band lte", appcontext.LocaleEN, "内置-频段-LTE", true, DimensionBand, "lte", "Built-in-Band-LTE"},
		{"en builtin device lte", appcontext.LocaleEN, "内置-设备-LTE", true, DimensionDevice, "lte", "Built-in-Device-LTE"},
		{"en builtin aggregate_group nr", appcontext.LocaleEN, "内置-临时组-NR", true, DimensionAggregateGroup, "nr", "Built-in-Aggregate Group-NR"},

		// 边界：制式为空 → 省略制式段。
		{"en builtin band no tech", appcontext.LocaleEN, "内置-频段", true, DimensionBand, "", "Built-in-Band"},
		// 边界：未知维度 → 兜底用原始枚举值，不组装出空段。
		{"en builtin unknown dim", appcontext.LocaleEN, "内置-x-LTE", true, Dimension("weird"), "lte", "Built-in-weird-LTE"},

		// 内置任务 + 中文（默认）：直接返回库里原值，不组装（中文路径零回归）。
		{"zh builtin uses raw name", appcontext.LocaleZH, "内置-频段-LTE", true, DimensionBand, "lte", "内置-频段-LTE"},
		// 缺省 locale 同中文路径。
		{"default locale uses raw name", appcontext.Locale(""), "内置-频段-LTE", true, DimensionBand, "lte", "内置-频段-LTE"},

		// 自建任务：任何 locale 都原样返回，不翻译。
		{"en custom task untouched", appcontext.LocaleEN, "验证-全网LTE-持续", false, DimensionNetwork, "lte", "验证-全网LTE-持续"},
		{"zh custom task untouched", appcontext.LocaleZH, "我的任务", false, DimensionBand, "lte", "我的任务"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := localizeTaskName(c.loc, c.rawName, c.isBuiltin, c.dim, c.technology)
			if got != c.want {
				t.Fatalf("localizeTaskName(%q, %q, builtin=%v, dim=%q, tech=%q) = %q, want %q",
					c.loc, c.rawName, c.isBuiltin, c.dim, c.technology, got, c.want)
			}
		})
	}
}
