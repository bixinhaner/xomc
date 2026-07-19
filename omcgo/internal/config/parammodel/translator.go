package parammodel

import (
	"sort"
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
	set                      *MappingSet
	standardToPrivate        map[string]*ParamMapping
	privateToStandard        map[string]*ParamMapping
	standardPartialToPrivate map[string][]partialPrefixTranslation
	privatePartialToStandard map[string][]partialPrefixTranslation
	skippedCount             int
	metrics                  *registryMetrics
}

type partialPrefixTranslation struct {
	translated string
	mapping    *ParamMapping
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
			set:                      &MappingSet{},
			standardToPrivate:        map[string]*ParamMapping{},
			privateToStandard:        map[string]*ParamMapping{},
			standardPartialToPrivate: map[string][]partialPrefixTranslation{},
			privatePartialToStandard: map[string][]partialPrefixTranslation{},
			metrics:                  metrics,
		}
	}

	stp := make(map[string]*ParamMapping, len(set.Mappings))
	pts := make(map[string]*ParamMapping, len(set.Mappings))
	stpPartialCandidates := make(map[string]map[string]*ParamMapping)
	ptsPartialCandidates := make(map[string]map[string]*ParamMapping)
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

		addPartialPrefixCandidates(stpPartialCandidates, m.StandardPath, m.PrivatePath, m)
		addPartialPrefixCandidates(ptsPartialCandidates, m.PrivatePath, m.StandardPath, m)
	}

	return &Translator{
		set:                      set,
		standardToPrivate:        stp,
		privateToStandard:        pts,
		standardPartialToPrivate: buildPartialPrefixIndex(stpPartialCandidates),
		privatePartialToStandard: buildPartialPrefixIndex(ptsPartialCandidates),
		skippedCount:             skipped,
		metrics:                  metrics,
	}
}

// ToPrivate 把 standardPath 翻译为 privatePath。Found=false → Translated=Original。
//
// ToPrivateCandidates 恰好返回一个结果时命中；多候选 partial prefix 保守返回 miss。
func (t *Translator) ToPrivate(standard string) TranslationResult {
	candidates := t.toPrivateCandidates(standard)
	if len(candidates) == 1 {
		t.observeTranslation("to_private", true)
		return candidates[0]
	}
	t.observeTranslation("to_private", false)
	return TranslationResult{Original: standard, Translated: standard, Found: false}
}

// ToPrivateCandidates 把 standardPath 翻译为所有合法 privatePath 候选。
//
// 精确映射和完整运行时叶子优先且只返回一个结果；partial object prefix 返回构造期
// 预计算、按目标路径排序并去重的全部候选。
func (t *Translator) ToPrivateCandidates(standard string) []TranslationResult {
	candidates := t.toPrivateCandidates(standard)
	t.observeTranslation("to_private", len(candidates) > 0)
	return candidates
}

// ToStandard 把 privatePath 翻译为 standardPath。Found=false → Translated=Original。
//
// ToStandardCandidates 恰好返回一个结果时命中；多候选 partial prefix 保守返回 miss。
func (t *Translator) ToStandard(private string) TranslationResult {
	candidates := t.toStandardCandidates(private)
	if len(candidates) == 1 {
		t.observeTranslation("to_standard", true)
		return candidates[0]
	}
	t.observeTranslation("to_standard", false)
	return TranslationResult{Original: private, Translated: private, Found: false}
}

// ToStandardCandidates 把 privatePath 翻译为所有合法 standardPath 候选。
//
// 规则与 ToPrivateCandidates 对称。
func (t *Translator) ToStandardCandidates(private string) []TranslationResult {
	candidates := t.toStandardCandidates(private)
	t.observeTranslation("to_standard", len(candidates) > 0)
	return candidates
}

func (t *Translator) toPrivateCandidates(standard string) []TranslationResult {
	if m, ok := t.standardToPrivate[standard]; ok {
		return singleTranslationResult(standard, m.PrivatePath, m)
	}
	norm := normalizeInstancePath(standard)
	if norm != standard {
		if m, ok := t.standardToPrivate[norm]; ok {
			if translated, ok := substituteInstanceNumbers(standard, m.PrivatePath); ok {
				return singleTranslationResult(standard, translated, m)
			}
		}
	}
	if candidates, ok := t.standardPartialToPrivate[standard]; ok {
		return partialTranslationResults(standard, candidates, false)
	}
	if norm != standard {
		if candidates, ok := t.standardPartialToPrivate[norm]; ok {
			return partialTranslationResults(standard, candidates, true)
		}
	}
	return nil
}

func (t *Translator) toStandardCandidates(private string) []TranslationResult {
	if m, ok := t.privateToStandard[private]; ok {
		return singleTranslationResult(private, m.StandardPath, m)
	}
	norm := normalizeInstancePath(private)
	if norm != private {
		if m, ok := t.privateToStandard[norm]; ok {
			if translated, ok := substituteInstanceNumbers(private, m.StandardPath); ok {
				return singleTranslationResult(private, translated, m)
			}
		}
	}
	if candidates, ok := t.privatePartialToStandard[private]; ok {
		return partialTranslationResults(private, candidates, false)
	}
	if norm != private {
		if candidates, ok := t.privatePartialToStandard[norm]; ok {
			return partialTranslationResults(private, candidates, true)
		}
	}
	return nil
}

func singleTranslationResult(original, translated string, mapping *ParamMapping) []TranslationResult {
	return []TranslationResult{{
		Original:   original,
		Translated: translated,
		Found:      true,
		Mapping:    mapping,
	}}
}

func partialTranslationResults(
	original string,
	candidates []partialPrefixTranslation,
	substitute bool,
) []TranslationResult {
	results := make([]TranslationResult, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		translated := candidate.translated
		if substitute {
			var ok bool
			translated, ok = substituteInstanceNumbers(original, translated)
			if !ok {
				continue
			}
		}
		if _, ok := seen[translated]; ok {
			continue
		}
		seen[translated] = struct{}{}
		results = append(results, TranslationResult{
			Original:   original,
			Translated: translated,
			Found:      true,
			Mapping:    candidate.mapping,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Translated < results[j].Translated
	})
	return results
}

func (t *Translator) observeTranslation(direction string, hit bool) {
	if t.metrics == nil {
		return
	}
	if hit {
		t.metrics.translateHit(direction)
		return
	}
	t.metrics.translateMiss(direction)
}

func addPartialPrefixCandidates(
	candidates map[string]map[string]*ParamMapping,
	sourceTemplate string,
	destinationTemplate string,
	mapping *ParamMapping,
) {
	sourcePrefixes := prefixesBeforePlaceholders(sourceTemplate)
	destinationPrefixes := prefixesBeforePlaceholders(destinationTemplate)
	if len(sourcePrefixes) != len(destinationPrefixes) {
		return
	}
	for i, source := range sourcePrefixes {
		destination := destinationPrefixes[i]
		if source == "" || destination == "" {
			continue
		}
		destinations := candidates[source]
		if destinations == nil {
			destinations = make(map[string]*ParamMapping)
			candidates[source] = destinations
		}
		if _, exists := destinations[destination]; !exists {
			destinations[destination] = mapping
		}
	}
}

func prefixesBeforePlaceholders(path string) []string {
	parts := strings.Split(path, ".")
	prefixes := make([]string, 0, strings.Count(path, placeholderToken))
	for i, part := range parts {
		if part != placeholderToken {
			continue
		}
		prefix := strings.Join(parts[:i], ".")
		if prefix != "" {
			prefix += "."
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

func buildPartialPrefixIndex(
	candidates map[string]map[string]*ParamMapping,
) map[string][]partialPrefixTranslation {
	index := make(map[string][]partialPrefixTranslation, len(candidates))
	for source, destinations := range candidates {
		translations := make([]partialPrefixTranslation, 0, len(destinations))
		for destination, mapping := range destinations {
			translations = append(translations, partialPrefixTranslation{
				translated: destination,
				mapping:    mapping,
			})
		}
		sort.Slice(translations, func(i, j int) bool {
			return translations[i].translated < translations[j].translated
		})
		index[source] = translations
	}
	return index
}

// substituteInstanceNumbers 把 srcWithNums 里各全数字段（运行时实例号）按顺序回填到
// dstTemplate 的 `{i}` 槽。两侧占位符计数已被 validatePlaceholders 保证相等——本
// 兜底只是把同序数字复制到对侧。
//
// 返回 ok=false 当：
//   - 任一入参为空
//   - src 抽出的数字段数 != dst 的 {i} 槽数（validator 已剔出 mismatch 条目，
//     这里是双保险，命中此分支应记为 translator 数据异常）
func substituteInstanceNumbers(srcWithNums, dstTemplate string) (string, bool) {
	if srcWithNums == "" || dstTemplate == "" {
		return "", false
	}
	nums := make([]string, 0, 4)
	for _, seg := range strings.Split(srcWithNums, ".") {
		if isAllDigits(seg) {
			nums = append(nums, seg)
		}
	}
	parts := strings.Split(dstTemplate, ".")
	consumed := 0
	for i, seg := range parts {
		if seg == placeholderToken {
			if consumed >= len(nums) {
				return "", false
			}
			parts[i] = nums[consumed]
			consumed++
		}
	}
	if consumed != len(nums) {
		return "", false
	}
	return strings.Join(parts, "."), true
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
