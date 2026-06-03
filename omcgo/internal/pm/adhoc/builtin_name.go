package adhoc

import (
	"strings"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

// dimensionEN 把维度枚举映射为英文显示词。覆盖全部 6 个枚举；未知维度兜底用原始枚举值。
var dimensionEN = map[Dimension]string{
	DimensionNetwork:        "Network",
	DimensionDeviceGroup:    "Device Group",
	DimensionProduct:        "Product",
	DimensionBand:           "Band",
	DimensionDevice:         "Device",
	DimensionAggregateGroup: "Aggregate Group",
}

// localizeTaskName 按 locale 返回任务显示名。
//
//   - 自建任务（!isBuiltin）：原样返回库里 rawName，不翻译。
//   - 内置任务 + 中文（默认）：直接返回库里 rawName（中文路径零回归，不走组装）。
//   - 内置任务 + 英文：按结构字段组装 "Built-in-<维度英文>-<制式大写>"，
//     如 band+lte → "Built-in-Band-LTE"。制式为空则省略制式段（"Built-in-<维度>"）。
func localizeTaskName(loc appcontext.Locale, rawName string, isBuiltin bool, dim Dimension, technology string) string {
	if !isBuiltin || loc != appcontext.LocaleEN {
		return rawName
	}
	dimWord, ok := dimensionEN[dim]
	if !ok {
		// 未知维度兜底：用原始枚举值，避免组装出空段。
		dimWord = string(dim)
	}
	name := "Built-in-" + dimWord
	if technology != "" {
		name += "-" + strings.ToUpper(technology)
	}
	return name
}
