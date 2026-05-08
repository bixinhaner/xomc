package parammodel

import (
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// placeholderToken 是 TR-069 路径中的实例占位符——standardPath 与 privatePath
// 必须 token 出现次数相同（设计 §1.7：按位置填实例号）。
const placeholderToken = "{i}"

// Translator 提供 standardPath ↔ privatePath 的 O(1) 双向翻译（设计 §1.6）。
//
// 不可变。一次构造对应一个 MappingSet 快照；调用方应短时间内用完丢弃，
// 避免错过 Registry 的更新。Registry.Translator(ctx, productID, swVersion)
// 会自动从最新快照构造。
type Translator struct {
	set               *MappingSet
	standardToPrivate map[string]*ParamMapping
	privateToStandard map[string]*ParamMapping
	skippedCount      int
	metrics           *registryMetrics
}

// NewTranslator 从 MappingSet 构造 Translator。
//
// 构造期校验：standardPath 与 privatePath 的 `{i}` 出现次数必须相等（设计 §1.7）。
// 不等条目跳过，记 WARN 日志 + Prometheus 计数。
//
// 重复 standardPath / privatePath：后者覆盖前者（按设计假定一对一，不应发生；
// 若发生则记录 WARN 并按字典顺序最后一条胜出）。
func NewTranslator(set *MappingSet, metrics *registryMetrics, logger *zap.Logger) *Translator {
	if logger == nil {
		logger = zap.NewNop()
	}
	if set == nil {
		return &Translator{
			set:               &MappingSet{},
			standardToPrivate: map[string]*ParamMapping{},
			privateToStandard: map[string]*ParamMapping{},
			metrics:           metrics,
		}
	}

	stp := make(map[string]*ParamMapping, len(set.Mappings))
	pts := make(map[string]*ParamMapping, len(set.Mappings))
	skipped := 0
	paramModelLabel := paramModelLabel(set)

	for i := range set.Mappings {
		m := &set.Mappings[i]
		if !validatePlaceholders(m.StandardPath, m.PrivatePath) {
			logger.Warn("ParamTranslator placeholder mismatch",
				zap.String("standard_path", m.StandardPath),
				zap.String("private_path", m.PrivatePath),
				zap.String("param_model", paramModelLabel),
			)
			if metrics != nil {
				metrics.invalidPlaceholder(paramModelLabel)
			}
			skipped++
			continue
		}
		if existing, ok := stp[m.StandardPath]; ok && existing != m {
			logger.Warn("ParamTranslator duplicate standardPath",
				zap.String("standard_path", m.StandardPath),
				zap.String("param_model", paramModelLabel),
			)
		}
		stp[m.StandardPath] = m
		if existing, ok := pts[m.PrivatePath]; ok && existing != m {
			logger.Warn("ParamTranslator duplicate privatePath",
				zap.String("private_path", m.PrivatePath),
				zap.String("param_model", paramModelLabel),
			)
		}
		pts[m.PrivatePath] = m
	}

	return &Translator{
		set:               set,
		standardToPrivate: stp,
		privateToStandard: pts,
		skippedCount:      skipped,
		metrics:           metrics,
	}
}

// ToPrivate 把 standardPath 翻译为 privatePath。Found=false → Translated=Original。
func (t *Translator) ToPrivate(standard string) TranslationResult {
	if m, ok := t.standardToPrivate[standard]; ok {
		if t.metrics != nil {
			t.metrics.translateHit("to_private")
		}
		return TranslationResult{
			Original:   standard,
			Translated: m.PrivatePath,
			Found:      true,
			Mapping:    m,
		}
	}
	if t.metrics != nil {
		t.metrics.translateMiss("to_private")
	}
	return TranslationResult{Original: standard, Translated: standard, Found: false}
}

// ToStandard 把 privatePath 翻译为 standardPath。Found=false → Translated=Original。
func (t *Translator) ToStandard(private string) TranslationResult {
	if m, ok := t.privateToStandard[private]; ok {
		if t.metrics != nil {
			t.metrics.translateHit("to_standard")
		}
		return TranslationResult{
			Original:   private,
			Translated: m.StandardPath,
			Found:      true,
			Mapping:    m,
		}
	}
	if t.metrics != nil {
		t.metrics.translateMiss("to_standard")
	}
	return TranslationResult{Original: private, Translated: private, Found: false}
}

// Mappings 返回构造时 MappingSet 的原始顺序（包含被占位符校验跳过的条目；
// P2-04 sync.go 用此做去重前缀，需要全集；翻译则只看校验通过的双向 map）。
func (t *Translator) Mappings() []ParamMapping {
	return t.set.Mappings
}

// Source 返回当前快照的来源（discovered 或 default）。
func (t *Translator) Source() MappingSource { return t.set.Source }

// SkippedCount 返回因 `{i}` 校验跳过的条目数；可用于运维巡检。
func (t *Translator) SkippedCount() int { return t.skippedCount }

// validatePlaceholders 校验两侧 `{i}` 出现次数是否相等。
func validatePlaceholders(standard, private string) bool {
	return strings.Count(standard, placeholderToken) == strings.Count(private, placeholderToken)
}

// paramModelLabel 给 metric / log 取一个稳定可读的标签：
//   - default 来源：用 paramModelID 字符串
//   - discovered 来源：用 productID:swVersion
//   - 都为空：fallback "unknown"
func paramModelLabel(set *MappingSet) string {
	if set.Source == MappingSourceDiscovered {
		return set.ProductID.String() + ":" + set.SoftwareVersion
	}
	if set.ParamModelID != uuid.Nil {
		return set.ParamModelID.String()
	}
	return "unknown"
}
