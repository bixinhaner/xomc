package mml

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
)

// ============================================================
// console_validate.go — R-8.4 / R-9.3 Service 入口校验与翻译
//
// 方案文档：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.4 §6.6
//
// 职责：
//   - R-8.4：device_sns → product_class 一致性校验，混类型直接拒绝
//   - R-9.3：把 task.Commands 内 standardPath 按 device.product_class 翻译为 privatePath
//
// 设计原则：
//   - 校验逻辑独立于 fanout 之外，便于单测
//   - DeviceLookup / PathTranslator nil 时整体跳过，向后兼容
// ============================================================

// ErrMixedProductClass 由 R-8.4 触发，混类型设备执行被拒绝。
// 错误体含 distinctClasses 列表，便于前端做精确提示。
type ErrMixedProductClass struct {
	Classes []string // 混入的所有 product_class（去重排序）
}

func (e *ErrMixedProductClass) Error() string {
	return fmt.Sprintf("mml: devices have mixed product_class (%s); refuse to execute (R-8.4)",
		strings.Join(e.Classes, ", "))
}

// Code 让 errors.Is / handler 错误码映射器拿到 ErrCodeDevicesMixedProductClass。
func (e *ErrMixedProductClass) Code() int { return global.ErrCodeDevicesMixedProductClass }

// ErrProductClassUnresolved 由 R-9.3 触发，孤儿设备 product_class 在 ProductRegistry 找不到。
type ErrProductClassUnresolved struct {
	SN           string
	ProductClass string
}

func (e *ErrProductClassUnresolved) Error() string {
	return fmt.Sprintf("mml: device %s product_class %q unresolved by ProductRegistry (R-9.3)",
		e.SN, e.ProductClass)
}

func (e *ErrProductClassUnresolved) Code() int { return global.ErrCodeProductClassUnresolved }

// ErrPathUnsupported (T-0170) — product 已识别但单条 standardPath 在该 paramModel
// 的 param_mappings 中无映射 → 不应静默 passthrough（CPE 必拒 9005），整 task 拒绝。
//
// 设计哲学：param_mappings 是 product 实际支持 path 的真值源；缺映射 = 不支持。
// 与 ErrProductClassUnresolved（整 product 未识别 → T-0168 orphan_passthrough）的区别：
// 本错误仅在 product 命中、Translator 加载成功后、单 path miss 时触发。
type ErrPathUnsupported struct {
	ProductClass string
	ParamModelID string
	Paths        []string // 不支持的 standardPath 清单
}

func (e *ErrPathUnsupported) Error() string {
	return fmt.Sprintf("mml: %d path(s) not in param_mappings for paramModel %s (product_class=%s): %v (T-0170)",
		len(e.Paths), e.ParamModelID, e.ProductClass, e.Paths)
}

func (e *ErrPathUnsupported) Code() int { return global.ErrCodeProductClassUnresolved } // 复用最贴近的错误码

// ErrNoValidDevices 由 R-8.4 / R-9.3 共用，目标设备列表为空或全部失效。
var ErrNoValidDevices = errors.New("mml: no valid devices to execute against (R-8.4)")

// validateDeviceProductClassUniform 实现 R-8.4：
//   - 按 SN 逐个查 devices.product_class（DeviceLookup.GetBySerialNumber）
//   - 若全部失效 → ErrNoValidDevices
//   - 若 distinct(product_class) > 1 → ErrMixedProductClass
//
// DeviceLookup 未注入时跳过校验（向后兼容；dev / 单测路径）。
// 性能：N≤10 设备时 N 次查询可接受；批量 SN 大场景下应优化为单次 IN-list 查询
// （在 device package 内增加 batch 方法，本期不做）。
func (s *Service) validateDeviceProductClassUniform(ctx context.Context, deviceSns []string) error {
	if s.deviceLookup == nil {
		return nil
	}
	if len(deviceSns) == 0 {
		return ErrNoValidDevices
	}

	classSet := make(map[string]struct{}, len(deviceSns))
	validCount := 0
	for _, sn := range deviceSns {
		dev, err := s.deviceLookup.GetBySerialNumber(ctx, sn)
		if err != nil {
			// 单设备查询失败：作为"无效设备"处理，不中断整体校验
			continue
		}
		if dev == nil || dev.ProductClass == "" {
			continue
		}
		classSet[dev.ProductClass] = struct{}{}
		validCount++
	}
	if validCount == 0 {
		return ErrNoValidDevices
	}
	if len(classSet) == 1 {
		return nil
	}

	classes := make([]string, 0, len(classSet))
	for pc := range classSet {
		classes = append(classes, pc)
	}
	sort.Strings(classes)
	return &ErrMixedProductClass{Classes: classes}
}

// pcTransCacheTTL 是 per-product_class 翻译结果的缓存有效期（用户决策：1 分钟）。
const pcTransCacheTTL = time.Minute

// pcTransCacheEntry 缓存某 (product_class, sw, paths) 的成功翻译结果。
type pcTransCacheEntry struct {
	outcome *TranslationOutcome
	expiry  time.Time
}

// translateForProductCached 按 (product_class, sw, paths) 翻译并缓存（TTL 1 分钟）。
// 同 product_class（同 sw、同 path 集）在 TTL 内复用，避免逐设备重复翻译。仅缓存成功结果；
// 失败（orphan / unsupported / IO）不缓存，交由调用方按 uniform/mixed 策略处理。
func (s *Service) translateForProductCached(
	ctx context.Context, productClass, sw string, paths []string,
) (*TranslationOutcome, error) {
	key := productClass + "|" + sw + "|" + strings.Join(paths, ",")
	now := time.Now()

	s.pcTransCacheMu.Lock()
	if e, ok := s.pcTransCache[key]; ok && now.Before(e.expiry) {
		s.pcTransCacheMu.Unlock()
		return e.outcome, nil
	}
	s.pcTransCacheMu.Unlock()

	outcome, err := s.pathTranslator.TranslateForDevice(ctx, productClass, sw, paths)
	if err != nil {
		return nil, err
	}
	if outcome == nil {
		return nil, fmt.Errorf("translate paths for product_class %s: nil outcome", productClass)
	}

	s.pcTransCacheMu.Lock()
	if s.pcTransCache == nil {
		s.pcTransCache = make(map[string]pcTransCacheEntry)
	}
	s.pcTransCache[key] = pcTransCacheEntry{outcome: outcome, expiry: now.Add(pcTransCacheTTL)}
	s.pcTransCacheMu.Unlock()
	return outcome, nil
}

// translateTaskPaths 实现 per-device（按 product_class 去重）的 standardPath → privatePath 预翻译。
//
// 解除原 R-8.4「混类型直接拒绝」：device_sns 可含不同 product_class。对每个 distinct product_class
// 翻译一次（translateForProductCached 缓存复用），用于：① 任务级翻译元数据 / 命令 translation_results
// 回显；② 单一 product_class 时保留 T-0170 即时 422（orphan / unsupported）预检。
//
// 注意：device_tasks 入队存 standardPath，真正逐设备翻译在 ACS 出队时按各自 product_class 完成
// （含 fallback）。因此混类型下，本预翻译失败的 product_class 不整体 422，交由 ACS 逐设备兜底。
func (s *Service) translateTaskPaths(ctx context.Context, task *MMLTask) error {
	if s.pathTranslator == nil || len(task.DeviceSNs) == 0 || s.deviceLookup == nil {
		return nil
	}
	standardPaths := collectStandardPaths(task.Commands)
	if len(standardPaths) == 0 {
		return nil
	}

	// 解析所有设备的 distinct (product_class, sw)；保持首次出现顺序。
	type pcKey struct{ pc, sw string }
	seen := make(map[pcKey]struct{})
	classes := make([]pcKey, 0, 2)
	for _, sn := range task.DeviceSNs {
		dev, err := s.deviceLookup.GetBySerialNumber(ctx, sn)
		if err != nil || dev == nil || dev.ProductClass == "" {
			continue
		}
		k := pcKey{dev.ProductClass, dev.FirmwareVersion}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			classes = append(classes, k)
		}
	}
	if len(classes) == 0 {
		return nil
	}
	mixed := len(classes) > 1

	var firstOutcome *TranslationOutcome
	for _, c := range classes {
		outcome, err := s.translateForProductCached(ctx, c.pc, c.sw, standardPaths)
		if err != nil {
			if mixed {
				// 混类型：单个 product_class 预翻译失败（orphan / unsupported / IO）不整体 422，
				// 交由 ACS 出队逐设备翻译 + fallback。
				s.logger.Warn("mixed product_class: skip pre-flight translate for one class",
					zap.String("product_class", c.pc), zap.Error(err))
				continue
			}
			// 单一 product_class：保留原 T-0170 即时 422（orphan / unsupported 透传给 handler）。
			var orphanErr *ErrProductClassUnresolved
			var unsupportedErr *ErrPathUnsupported
			if errors.As(err, &orphanErr) || errors.As(err, &unsupportedErr) {
				return err
			}
			return fmt.Errorf("translate paths for product_class %s: %w", c.pc, err)
		}
		if firstOutcome == nil {
			firstOutcome = outcome
		}
	}
	if firstOutcome == nil {
		// 混类型且所有 product_class 预翻译都失败：不阻塞，交 ACS 逐设备兜底。
		return nil
	}

	// 任务级翻译元数据（混类型以首个成功 product_class 为代表；逐设备实际翻译见 device_tasks / ACS）。
	task.ProductResolved = firstOutcome.ProductResolved
	task.MatchedProductClass = firstOutcome.ProductClass
	if firstOutcome.ProductResolved && firstOutcome.ProductID != uuid.Nil {
		id := firstOutcome.ProductID
		task.MatchedProductID = &id
	} else {
		task.MatchedProductID = nil
	}
	task.PathTranslationSource = firstOutcome.AggregateSource

	trans := make(map[string]TranslatedPath, len(firstOutcome.Paths))
	for _, r := range firstOutcome.Paths {
		trans[r.Standard] = r
	}
	for i := range task.Commands {
		applyTranslationToCommandEntry(task.Commands[i], trans)
	}
	return nil
}

// collectStandardPaths 从 task.Commands 中收集所有需要翻译的 standardPath。
//
// 真值源：param_refs[].Tr069Path（standardPath，由 buildLST/MODParamRefs 从 cmd.Params 拷贝）。
// parameters 的 key 是 MMLCode（leaf 段，per buildStatementCommandEntry MOD 分支 stmt.Values
// 在 StructuredToStatement 已 sf.MMLCode keyed），不是 standardPath，禁用为翻译来源。
//
// 类型容忍：buildStatementCommandEntries 内存中 param_refs 是 []MMLParamRef；JSON 反序列化
// 路径（极少触发）为 []interface{}。两种 shape 都遍历，字段名读 Tr069Path / "tr069_path"
// (MMLParamRef json tag)。
func collectStandardPaths(commands []map[string]interface{}) []string {
	seen := make(map[string]struct{})
	addPath := func(p string) {
		if p != "" {
			seen[normalizeNewInstancePlaceholder(p)] = struct{}{}
		}
	}
	for _, entry := range commands {
		switch refs := entry["param_refs"].(type) {
		case []MMLParamRef:
			for _, ref := range refs {
				addPath(ref.Tr069Path)
			}
		case []interface{}:
			for _, r := range refs {
				if m, ok := r.(map[string]interface{}); ok {
					if p, ok := m["tr069_path"].(string); ok {
						addPath(p)
					}
				}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// applyTranslationToCommandEntry 把翻译结果写回单条 command entry。
//   - param_refs[].PrivatePath / TranslationSource 字段（in-memory）追加
//   - parameters key 不替换（key 是 MMLCode，非 standardPath；SOAP 层用 param_refs 查
//     MMLCode→Tr069/PrivatePath，无需修改 parameters）
//   - 整体 entry 追加 translation_results 数组，供 TerminalPanel 回显
//
// 类型容忍：与 collectStandardPaths 对称，支持 []MMLParamRef 和 []interface{} 两种形态。
func applyTranslationToCommandEntry(entry map[string]interface{}, trans map[string]TranslatedPath) {
	usedTranslations := make([]TranslatedPath, 0)

	switch refs := entry["param_refs"].(type) {
	case []MMLParamRef:
		// 写回到具体 struct slice — 用索引写避免 range 拷贝
		updated := make([]MMLParamRef, len(refs))
		copy(updated, refs)
		for i := range updated {
			if t, hit := translationForRuntimePath(trans, updated[i].Tr069Path); hit {
				updated[i].PrivatePath = t.Private
				updated[i].TranslationSource = t.Source
				usedTranslations = append(usedTranslations, t)
			}
		}
		entry["param_refs"] = updated
	case []interface{}:
		for _, r := range refs {
			m, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			p, _ := m["tr069_path"].(string)
			if t, hit := translationForRuntimePath(trans, p); hit {
				m["private_path"] = t.Private
				m["translation_source"] = t.Source
				usedTranslations = append(usedTranslations, t)
			}
		}
	}

	if len(usedTranslations) > 0 {
		entry["translation_results"] = usedTranslations
	}
}

// ADD 的复合 AddObject→SPV 流程用 {NEW} 表示尚未由设备返回的新实例号。
// Translator 的模板匹配只接受具体数字实例，不接受“外层已具体 + 内层 {i}”的
// 混合运行时路径。因此预翻译时用一个不会真正下发的数字哨兵值匹配
// param_mappings，写回 private path 时再恢复 {NEW}，供 Sequencer 替换为
// AddObjectResponse.instance_number。
const newInstanceTranslationSentinel = "2147483647"

func normalizeNewInstancePlaceholder(path string) string {
	return strings.ReplaceAll(path, "{NEW}", newInstanceTranslationSentinel)
}

func translationForRuntimePath(trans map[string]TranslatedPath, runtimePath string) (TranslatedPath, bool) {
	lookupPath := normalizeNewInstancePlaceholder(runtimePath)
	t, ok := trans[lookupPath]
	if !ok {
		return TranslatedPath{}, false
	}
	t.Standard = runtimePath
	if strings.Contains(runtimePath, "{NEW}") {
		t.Private = strings.ReplaceAll(t.Private, newInstanceTranslationSentinel, "{NEW}")
	}
	return t, true
}
