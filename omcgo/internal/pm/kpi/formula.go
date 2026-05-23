package kpi

import (
	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
)

// Formula 是 KPI 算术公式的解析结果。
//
// 真正的解析 / 求值实现已迁移到 `pm/kpi/expr` 子包（拆包是为了打破
// pm/kpi ↔ pm/indicator 的循环依赖）；本类型作为兼容别名保留，
// 老调用方（engine.go / handler / formula_test.go）保持原 import 不变。
type Formula = expr.Formula

// ParseFormula parses a KPI formula expression into a Formula.
func ParseFormula(expression string) (*Formula, error) {
	return expr.Parse(expression)
}
