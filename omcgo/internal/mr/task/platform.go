package task

import (
	"context"
	"strings"
)

// 2026-05-26 重构：从"regex 匹配 productClass 字符串"改为"走 ProductRegistry
// 解析出 product → param_model.name → 校对允许列表"。原因：
//   - 原 regex 写死字符串模式，与产品中心实际录入脱节；
//     例如 QRTB 系列 → param_model=BLQ，但 productClass 不含 BLQ 字串，regex 漏；
//   - 用户希望 MR 支持范围与产品中心数据保持同源。
//
// 允许列表与 MR_Feature_Analysis.md §2 + 用户决策（2026-05-26）一致：
//   BLQ / MLQ / MLN / BM（替换原 IntelCR）。
//
// 扩展：未来加新平台只需把 param_models.name 加进 mrSupportedPlatforms。

// mrSupportedPlatforms 是允许跑 MR 的 param_model.name 集合。
// 与 internal/config/parammodel 加载的 param_models 表的 name 列对齐
// （如 data/param-mappings/BLQ.xml 中 paramModel="BLQ"）。
var mrSupportedPlatforms = map[string]struct{}{
	"BLQ": {},
	"MLQ": {},
	"MLN": {},
	"BM":  {},
}

// PlatformResolver 把 productClass 解析为 platform 名（即 param_model.name）。
// 由 cmd/app/provider/mr_adapters.go 实现（链路：ProductRegistry.MatchProductClass
// → product.ParamModelID → parammodel.GetParamModelByID → ParamModel.Name）。
//
// 单进程/单测可用 fake 实现。返回空字符串表示"未识别"，等价于 unsupport。
type PlatformResolver interface {
	ResolvePlatform(ctx context.Context, productClass string) (platformName string, err error)
}

// IsSupportedPlatform 判断 platform 名是否在 MR 允许列表内。
// 仅做 set 查找，**不**做 productClass 解析（解析是 PlatformResolver 的职责）。
func IsSupportedPlatform(name string) bool {
	if name == "" {
		return false
	}
	_, ok := mrSupportedPlatforms[strings.TrimSpace(name)]
	return ok
}

// SupportedPlatforms 返回允许列表（便于日志 / 调试输出）。
func SupportedPlatforms() []string {
	out := make([]string, 0, len(mrSupportedPlatforms))
	for k := range mrSupportedPlatforms {
		out = append(out, k)
	}
	return out
}
