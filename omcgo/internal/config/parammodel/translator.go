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
	set                               *MappingSet
	standardToPrivate                 map[string]*ParamMapping
	privateToStandard                 map[string]*ParamMapping
	standardTemplatesToPrivate        *pathTemplateIndex[*ParamMapping]
	privateTemplatesToStandard        *pathTemplateIndex[*ParamMapping]
	standardPartialToPrivate          map[string][]partialPrefixTranslation
	privatePartialToStandard          map[string][]partialPrefixTranslation
	standardPartialTemplatesToPrivate *pathTemplateIndex[[]partialPrefixTranslation]
	privatePartialTemplatesToStandard *pathTemplateIndex[[]partialPrefixTranslation]
	skippedCount                      int
	metrics                           *registryMetrics
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
		emptyMappings := map[string]*ParamMapping{}
		emptyPartials := map[string][]partialPrefixTranslation{}
		return &Translator{
			set:                               &MappingSet{},
			standardToPrivate:                 emptyMappings,
			privateToStandard:                 emptyMappings,
			standardTemplatesToPrivate:        buildPathTemplateIndex(emptyMappings),
			privateTemplatesToStandard:        buildPathTemplateIndex(emptyMappings),
			standardPartialToPrivate:          emptyPartials,
			privatePartialToStandard:          emptyPartials,
			standardPartialTemplatesToPrivate: buildPathTemplateIndex(emptyPartials),
			privatePartialTemplatesToStandard: buildPathTemplateIndex(emptyPartials),
			metrics:                           metrics,
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

	stpPartials := buildPartialPrefixIndex(stpPartialCandidates)
	ptsPartials := buildPartialPrefixIndex(ptsPartialCandidates)

	return &Translator{
		set:                               set,
		standardToPrivate:                 stp,
		privateToStandard:                 pts,
		standardTemplatesToPrivate:        buildPathTemplateIndex(stp),
		privateTemplatesToStandard:        buildPathTemplateIndex(pts),
		standardPartialToPrivate:          stpPartials,
		privatePartialToStandard:          ptsPartials,
		standardPartialTemplatesToPrivate: buildPathTemplateIndex(stpPartials),
		privatePartialTemplatesToStandard: buildPathTemplateIndex(ptsPartials),
		skippedCount:                      skipped,
		metrics:                           metrics,
	}
}

// ToPrivate 把 standardPath 翻译为 privatePath。Found=false → Translated=Original。
//
// ToPrivateCandidates 恰好返回一个结果时命中；多候选 partial prefix 保守返回 miss。
func (t *Translator) ToPrivate(standard string) TranslationResult {
	if result, ok := t.lookupPrivateExact(standard); ok {
		t.observeTranslation("to_private", true)
		return result
	}
	if candidates, instanceNumbers, ok := t.lookupPrivatePartial(standard); ok {
		if result, ok := singlePartialTranslationResult(standard, candidates, instanceNumbers); ok {
			t.observeTranslation("to_private", true)
			return result
		}
	}
	t.observeTranslation("to_private", false)
	return TranslationResult{Original: standard, Translated: standard, Found: false}
}

// ToPrivateCandidates 把 standardPath 翻译为所有合法 privatePath 候选。
//
// 精确映射和完整运行时叶子优先且只返回一个结果；partial object prefix 返回构造期
// 预计算、按目标路径排序并去重的全部候选。
func (t *Translator) ToPrivateCandidates(standard string) []TranslationResult {
	if result, ok := t.lookupPrivateExact(standard); ok {
		t.observeTranslation("to_private", true)
		return []TranslationResult{result}
	}
	if candidates, instanceNumbers, ok := t.lookupPrivatePartial(standard); ok {
		results := partialTranslationResults(standard, candidates, instanceNumbers)
		t.observeTranslation("to_private", len(results) > 0)
		return results
	}
	t.observeTranslation("to_private", false)
	return nil
}

// ToStandard 把 privatePath 翻译为 standardPath。Found=false → Translated=Original。
//
// ToStandardCandidates 恰好返回一个结果时命中；多候选 partial prefix 保守返回 miss。
func (t *Translator) ToStandard(private string) TranslationResult {
	if result, ok := t.lookupStandardExact(private); ok {
		t.observeTranslation("to_standard", true)
		return result
	}
	if candidates, instanceNumbers, ok := t.lookupStandardPartial(private); ok {
		if result, ok := singlePartialTranslationResult(private, candidates, instanceNumbers); ok {
			t.observeTranslation("to_standard", true)
			return result
		}
	}
	t.observeTranslation("to_standard", false)
	return TranslationResult{Original: private, Translated: private, Found: false}
}

// ToStandardCandidates 把 privatePath 翻译为所有合法 standardPath 候选。
//
// 规则与 ToPrivateCandidates 对称。
func (t *Translator) ToStandardCandidates(private string) []TranslationResult {
	if result, ok := t.lookupStandardExact(private); ok {
		t.observeTranslation("to_standard", true)
		return []TranslationResult{result}
	}
	if candidates, instanceNumbers, ok := t.lookupStandardPartial(private); ok {
		results := partialTranslationResults(private, candidates, instanceNumbers)
		t.observeTranslation("to_standard", len(results) > 0)
		return results
	}
	t.observeTranslation("to_standard", false)
	return nil
}

func (t *Translator) lookupPrivateExact(standard string) (TranslationResult, bool) {
	if m, ok := t.standardToPrivate[standard]; ok {
		return translationResult(standard, m.PrivatePath, m), true
	}
	if m, instanceNumbers, ok := t.standardTemplatesToPrivate.Match(standard); ok {
		if translated, ok := substituteInstanceNumbers(instanceNumbers, m.PrivatePath); ok {
			return translationResult(standard, translated, m), true
		}
	}
	return TranslationResult{}, false
}

func (t *Translator) lookupPrivatePartial(
	standard string,
) ([]partialPrefixTranslation, []string, bool) {
	if candidates, ok := t.standardPartialToPrivate[standard]; ok {
		return candidates, nil, true
	}
	if candidates, instanceNumbers, ok := t.standardPartialTemplatesToPrivate.Match(standard); ok {
		return candidates, instanceNumbers, true
	}
	return nil, nil, false
}

func (t *Translator) lookupStandardExact(private string) (TranslationResult, bool) {
	if m, ok := t.privateToStandard[private]; ok {
		return translationResult(private, m.StandardPath, m), true
	}
	if m, instanceNumbers, ok := t.privateTemplatesToStandard.Match(private); ok {
		if translated, ok := substituteInstanceNumbers(instanceNumbers, m.StandardPath); ok {
			return translationResult(private, translated, m), true
		}
	}
	return TranslationResult{}, false
}

func (t *Translator) lookupStandardPartial(
	private string,
) ([]partialPrefixTranslation, []string, bool) {
	if candidates, ok := t.privatePartialToStandard[private]; ok {
		return candidates, nil, true
	}
	if candidates, instanceNumbers, ok := t.privatePartialTemplatesToStandard.Match(private); ok {
		return candidates, instanceNumbers, true
	}
	return nil, nil, false
}

func translationResult(original, translated string, mapping *ParamMapping) TranslationResult {
	return TranslationResult{
		Original:   original,
		Translated: translated,
		Found:      true,
		Mapping:    mapping,
	}
}

func singlePartialTranslationResult(
	original string,
	candidates []partialPrefixTranslation,
	instanceNumbers []string,
) (TranslationResult, bool) {
	var result TranslationResult
	found := false
	for _, candidate := range candidates {
		translated, ok := translatePartialCandidate(candidate.translated, instanceNumbers)
		if !ok {
			continue
		}
		if found {
			if translated != result.Translated {
				return TranslationResult{}, false
			}
			continue
		}
		result = translationResult(original, translated, candidate.mapping)
		found = true
	}
	return result, found
}

func partialTranslationResults(
	original string,
	candidates []partialPrefixTranslation,
	instanceNumbers []string,
) []TranslationResult {
	results := make([]TranslationResult, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		translated, ok := translatePartialCandidate(candidate.translated, instanceNumbers)
		if !ok {
			continue
		}
		if _, ok := seen[translated]; ok {
			continue
		}
		seen[translated] = struct{}{}
		results = append(results, translationResult(original, translated, candidate.mapping))
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Translated < results[j].Translated
	})
	return results
}

func translatePartialCandidate(template string, instanceNumbers []string) (string, bool) {
	if instanceNumbers == nil {
		return template, true
	}
	return substituteInstanceNumbers(instanceNumbers, template)
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

type pathTemplateIndex[V any] struct {
	root pathTemplateNode[V]
}

type pathTemplateNode[V any] struct {
	static      map[string]*pathTemplateNode[V]
	placeholder *pathTemplateNode[V]
	value       V
	hasValue    bool
}

func buildPathTemplateIndex[V any](entries map[string]V) *pathTemplateIndex[V] {
	index := &pathTemplateIndex[V]{}
	for template, value := range entries {
		parts := strings.Split(template, ".")
		hasPlaceholder := false
		for _, part := range parts {
			if part == placeholderToken {
				hasPlaceholder = true
				break
			}
		}
		if !hasPlaceholder {
			continue
		}

		node := &index.root
		for _, part := range parts {
			if part == placeholderToken {
				if node.placeholder == nil {
					node.placeholder = &pathTemplateNode[V]{}
				}
				node = node.placeholder
				continue
			}
			if node.static == nil {
				node.static = make(map[string]*pathTemplateNode[V])
			}
			child := node.static[part]
			if child == nil {
				child = &pathTemplateNode[V]{}
				node.static[part] = child
			}
			node = child
		}
		node.value = value
		node.hasValue = true
	}
	return index
}

// Match 按模板段匹配运行时路径。静态段优先于 `{i}`，因此模板中的固定数字
// （如 Profile.1.）不会被误当作实例号；只有经过 `{i}` 边的纯数字段才被捕获。
func (index *pathTemplateIndex[V]) Match(path string) (V, []string, bool) {
	var zero V
	if index == nil {
		return zero, nil, false
	}
	return matchPathTemplateNode(&index.root, strings.Split(path, "."), 0, make([]string, 0, 4))
}

func matchPathTemplateNode[V any](
	node *pathTemplateNode[V],
	parts []string,
	position int,
	instanceNumbers []string,
) (V, []string, bool) {
	if position == len(parts) {
		if node.hasValue {
			return node.value, instanceNumbers, true
		}
		var zero V
		return zero, nil, false
	}

	part := parts[position]
	if child := node.static[part]; child != nil {
		if value, matchedNumbers, ok := matchPathTemplateNode(
			child,
			parts,
			position+1,
			instanceNumbers,
		); ok {
			return value, matchedNumbers, true
		}
	}
	if node.placeholder != nil && isAllDigits(part) {
		if value, matchedNumbers, ok := matchPathTemplateNode(
			node.placeholder,
			parts,
			position+1,
			append(instanceNumbers, part),
		); ok {
			return value, matchedNumbers, true
		}
	}

	var zero V
	return zero, nil, false
}

// substituteInstanceNumbers 把已按源模板 `{i}` 位置捕获的运行时实例号，按顺序
// 回填到目标模板的 `{i}` 槽。目标模板中的固定数字段保持原值。
//
// 返回 ok=false 当：
//   - 目标模板为空
//   - 捕获实例号数量 != 目标模板的 {i} 槽数
func substituteInstanceNumbers(instanceNumbers []string, dstTemplate string) (string, bool) {
	if dstTemplate == "" {
		return "", false
	}
	parts := strings.Split(dstTemplate, ".")
	consumed := 0
	for i, seg := range parts {
		if seg == placeholderToken {
			if consumed >= len(instanceNumbers) {
				return "", false
			}
			parts[i] = instanceNumbers[consumed]
			consumed++
		}
	}
	if consumed != len(instanceNumbers) {
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
