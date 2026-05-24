package mml

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

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

// translateTaskPaths 实现 R-9.3：把 task.Commands 内 standardPath 翻译为 privatePath。
//
// PathTranslator 未注入时跳过（向后兼容）。
//
// 翻译输入：
//   - task.Commands[i]["param_refs"]      []MMLParamRef     LST/MOD/ADD 选中路径
//   - task.Commands[i]["parameters"]      map[string]any    MOD 值（key=path）
// 翻译输出：
//   - 在 param_refs[] 中追加 PrivatePath / Source 字段（供 fanouter 写入 device_task.params）
//   - 在 parameters 中把 key 替换为 privatePath
//   - 记录 translation_results 元数据（standard / private / source）供 TerminalPanel 回显
//
// 注意：当前实现按 task 维度做"一次翻译"（假设 device_sns 同 product_class，
// R-8.4 已保证）。如果未来出现同 product_class 不同 software_version，可由 fanouter
// 在 device 维度再翻译（D27 决策保留弹性）。
func (s *Service) translateTaskPaths(ctx context.Context, task *MMLTask) error {
	if s.pathTranslator == nil {
		return nil
	}
	if len(task.DeviceSNs) == 0 || s.deviceLookup == nil {
		return nil
	}

	// 取第一个设备的 product_class + software_version（R-8.4 已保证同 product_class）
	firstDev, err := s.deviceLookup.GetBySerialNumber(ctx, task.DeviceSNs[0])
	if err != nil {
		return fmt.Errorf("lookup device %s for translation: %w", task.DeviceSNs[0], err)
	}
	if firstDev == nil || firstDev.ProductClass == "" {
		// 已被 R-8.4 拦住的情况；这里只是双保险
		return nil
	}
	productClass := firstDev.ProductClass
	softwareVersion := firstDev.FirmwareVersion

	// 收集本次任务待翻译的所有 standardPath
	standardPaths := collectStandardPaths(task.Commands)
	if len(standardPaths) == 0 {
		return nil
	}

	outcome, err := s.pathTranslator.TranslateForDevice(ctx, productClass, softwareVersion, standardPaths)
	if err != nil {
		// T-0168: ErrProductClassUnresolved 已被适配器内部消化为 orphan_passthrough；
		// 保留 sentinel 检查作为旧适配器实现的向后兼容兜底。
		var orphanErr *ErrProductClassUnresolved
		if errors.As(err, &orphanErr) {
			return err
		}
		// Translator 内部错误（Registry IO / DI 失败）→ wrap
		return fmt.Errorf("translate paths for product_class %s: %w", productClass, err)
	}
	if outcome == nil {
		return fmt.Errorf("translate paths for product_class %s: nil outcome", productClass)
	}

	// T-0168: 写回任务级翻译元数据，由 PgTaskRepository.Create 持久化到 mml_tasks 4 列。
	task.ProductResolved = outcome.ProductResolved
	task.MatchedProductClass = outcome.ProductClass
	if outcome.ProductResolved && outcome.ProductID != uuid.Nil {
		id := outcome.ProductID
		task.MatchedProductID = &id
	} else {
		task.MatchedProductID = nil
	}
	task.PathTranslationSource = outcome.AggregateSource

	// 构造 standard → privatePath 映射
	trans := make(map[string]TranslatedPath, len(outcome.Paths))
	for _, r := range outcome.Paths {
		trans[r.Standard] = r
	}

	// 写回 task.Commands：在 commands[i] 增加 translation_results 元数据
	for i := range task.Commands {
		applyTranslationToCommandEntry(task.Commands[i], trans)
	}
	return nil
}

// collectStandardPaths 从 task.Commands 中收集所有需要翻译的 standardPath。
// param_refs[].Path 或 parameters 的 key 都是 standardPath 来源（互斥）。
func collectStandardPaths(commands []map[string]interface{}) []string {
	seen := make(map[string]struct{})
	for _, entry := range commands {
		// param_refs: []MMLParamRef 或 []map
		if refs, ok := entry["param_refs"].([]interface{}); ok {
			for _, r := range refs {
				if m, ok := r.(map[string]interface{}); ok {
					if p, ok := m["param_path"].(string); ok && p != "" {
						seen[p] = struct{}{}
					}
				}
			}
		}
		// parameters: map[string]any
		if params, ok := entry["parameters"].(map[string]interface{}); ok {
			for k := range params {
				if k != "" && k != "object_name" { // object_name 不参与翻译（ADD/RMV 父级路径）
					seen[k] = struct{}{}
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
//   - param_refs[].private_path / param_refs[].translation_source 追加
//   - parameters key 同步替换为 privatePath（仅当翻译命中且与 standard 不同）
//   - 整体 entry 追加 translation_results 数组，供 TerminalPanel 回显
func applyTranslationToCommandEntry(entry map[string]interface{}, trans map[string]TranslatedPath) {
	usedTranslations := make([]TranslatedPath, 0)

	// param_refs
	if refs, ok := entry["param_refs"].([]interface{}); ok {
		for _, r := range refs {
			m, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			p, _ := m["param_path"].(string)
			if t, hit := trans[p]; hit {
				m["private_path"] = t.Private
				m["translation_source"] = t.Source
				usedTranslations = append(usedTranslations, t)
			}
		}
	}

	// parameters：替换 key 为 privatePath
	if params, ok := entry["parameters"].(map[string]interface{}); ok {
		newParams := make(map[string]interface{}, len(params))
		for k, v := range params {
			if t, hit := trans[k]; hit && t.Private != k {
				newParams[t.Private] = v
				usedTranslations = append(usedTranslations, t)
			} else {
				newParams[k] = v
			}
		}
		entry["parameters"] = newParams
	}

	if len(usedTranslations) > 0 {
		entry["translation_results"] = usedTranslations
	}
}
