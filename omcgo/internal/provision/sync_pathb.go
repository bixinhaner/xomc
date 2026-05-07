package provision

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// WithParamRegistry 启用 T-0098 P2-04 dual-stack 模式（新栈 Path B）。
//
// enabled=false 或 paramReg/prodReg 任一为 nil → 等价于不调用本方法（沿用 datamodel
// 旧栈，零行为差异）。
func (s *SyncService) WithParamRegistry(paramReg *parammodel.Registry, prodReg *product.Registry, enabled bool) *SyncService {
	s.paramRegistry = paramReg
	s.productRegistry = prodReg
	s.paramRegistryEnabled = enabled && paramReg != nil && prodReg != nil
	return s
}

// PathBEnabled 返回是否对当前 device 走新栈 Path B。
//
// 调用方典型用法：handleAutoSync(ctx, dev, dm) 先 PathBEnabled(ctx, dev) 判断；
// 命中 → 走 StartPathBSync，否则走 StartTwoPhaseSync。
//
// 任何前置条件失败（flag off / 任一 registry nil / dev.ProductClass 空 / MatchProductClass
// 未命中 / GetByProduct 出错）→ 返回 false，让调用方降级到旧栈。
func (s *SyncService) PathBEnabled(ctx context.Context, dev *model.Device) bool {
	_, ok := s.resolveMappingSet(ctx, dev)
	return ok
}

// StartPathBSync 启动 T-0098 设计 §1.11 Path B 同步。
//
// 流程（彻底删去 GPN 阶段）：
//  1. paramRegistry.GetByProduct → MappingSet（discovered 优先，default 兜底）
//  2. 抽 is_storable=true 的 privatePath → 去重前缀（ParamMapping 列表已含全部参数）
//  3. 直接 enqueue GPV 对象前缀；CPE 自动返回所有实例
//
// 返回 (true, nil) 表示已切到 Path B；(false, nil) 表示无法走新栈，调用方应降级旧栈；
// (false, err) 表示新栈选中后执行出错（不再降级，由 engine 处理）。
func (s *SyncService) StartPathBSync(ctx context.Context, dev *model.Device) (bool, error) {
	set, ok := s.resolveMappingSet(ctx, dev)
	if !ok {
		return false, nil
	}

	prefixes := extractStorablePrefixes(set.Mappings)
	if len(prefixes) == 0 {
		s.logger.Info("path-b sync skipped: no storable prefixes",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("source", string(set.Source)),
		)
		// 仍标 syncing → completed，避免下游 stuck
		log, _ := s.discoveryRepo.GetByDeviceID(ctx, dev.ID)
		if log != nil {
			_ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryCompleted, "")
		}
		return true, nil
	}

	if err := s.enqueueGPVPrefixes(ctx, dev, prefixes); err != nil {
		return true, fmt.Errorf("enqueue path-b GPV: %w", err)
	}

	s.logger.Info("path-b sync started",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("source", string(set.Source)),
		zap.Int("prefixes", len(prefixes)),
		zap.Int("total_mappings", len(set.Mappings)),
	)
	return true, nil
}

// HandleSyncResultPathB 处理新栈 GPV 响应。
//
// 与旧 HandleSyncResult 的差异：
//   - 通过 Translator + MappingValidator 把设备返回的 privatePath 翻译为 standardPath
//   - is_storable=false 的条目直接丢弃（不写入参数仓库）
//   - 翻译未命中的条目降级为原 privatePath 写入（容错），加 WARN log
//
// 返回 nil 即视为成功；底层 BatchUpsert 错误向上传。
func (s *SyncService) HandleSyncResultPathB(ctx context.Context, dev *model.Device,
	paramValues []tr069.ParameterValueStruct) (bool, error) {
	if len(paramValues) == 0 {
		return false, nil
	}

	set, ok := s.resolveMappingSet(ctx, dev)
	if !ok {
		return false, nil
	}
	validator := parammodel.NewMappingValidator(set)
	if validator == nil {
		return false, nil
	}

	params := make([]model.DeviceParameter, 0, len(paramValues))
	now := time.Now()
	dropped := 0
	translated := 0
	untranslated := 0

	for _, pv := range paramValues {
		mapping := validator.LookupParam(pv.Name)
		if mapping == nil {
			// 未命中 mapping —— 容错写入原私有路径。CPE 可能返回 mapping 表未覆盖
			// 的厂商扩展参数，丢弃则丢数据；保留则保住可见性。
			s.logger.Warn("path-b sync: privatePath not in mapping; storing as-is",
				zap.String("device_sn", dev.SerialNumber),
				zap.String("private_path", pv.Name),
			)
			params = append(params, model.DeviceParameter{
				DeviceID:       dev.ID,
				ParameterPath:  pv.Name,
				ParameterValue: pv.Value,
				ParameterType:  inferParameterType(pv.Value, pv.Type),
				Writable:       false,
				LastUpdatedAt:  now,
			})
			untranslated++
			continue
		}
		if !mapping.IsStorable {
			// is_storable=false：GPV 子树会带回不可存条目，丢弃即可（设计 §1.11）
			dropped++
			continue
		}
		standardPath := instantiateStandardPath(pv.Name, mapping.PrivatePath, mapping.StandardPath)
		params = append(params, model.DeviceParameter{
			DeviceID:       dev.ID,
			ParameterPath:  standardPath,
			ParameterValue: pv.Value,
			ParameterType:  inferParameterType(pv.Value, pv.Type),
			Writable:       parammodel.IsAccessWritable(mapping.Access),
			LastUpdatedAt:  now,
		})
		translated++
	}

	if len(params) > 0 {
		if err := s.paramRepo.BatchUpsert(ctx, dev.ID, params); err != nil {
			return true, fmt.Errorf("path-b batch upsert: %w", err)
		}
	}

	s.logger.Debug("path-b sync result handled",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("source", string(set.Source)),
		zap.Int("input", len(paramValues)),
		zap.Int("translated", translated),
		zap.Int("untranslated", untranslated),
		zap.Int("dropped_non_storable", dropped),
	)
	return true, nil
}

// resolveMappingSet 在 dual-stack 启用时尝试解析 device 对应的 MappingSet。
//
// 任何前置条件失败 → 返回 (nil, false)。
func (s *SyncService) resolveMappingSet(ctx context.Context, dev *model.Device) (*parammodel.MappingSet, bool) {
	if !s.paramRegistryEnabled || s.paramRegistry == nil || s.productRegistry == nil {
		return nil, false
	}
	if dev == nil || dev.ProductClass == "" {
		return nil, false
	}
	match, err := s.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil {
		return nil, false
	}
	set, err := s.paramRegistry.GetByProduct(ctx, match.Product.ID, dev.FirmwareVersion)
	if err != nil || set == nil {
		return nil, false
	}
	return set, true
}

// ── 纯函数辅助（便于单测） ─────────────────────────────────────────────

// extractStorablePrefixes 从 ParamMapping 列表抽出 is_storable=true 的去重对象前缀。
//
// 算法：
//   1. 过滤 is_storable=true
//   2. 把每条 privatePath 的"叶子路径"提取为基础对象前缀（去末段 + 加 "."）
//      - 如果 privatePath 以 "." 结尾（object 类型），原样使用
//      - 否则去掉最后一个 "." 之后的部分（最后一段是叶子参数名）
//   3. 把含 "{i}" 占位符的段替换为模板基础前缀（截断到第一个 "{i}" 之前一段含 "."）
//   4. 去重排序输出
//
// 设计 §1.11 Path B：枚举对象前缀做 GPV，CPE 自动展开所有实例号。
func extractStorablePrefixes(mappings []parammodel.ParamMapping) []string {
	seen := make(map[string]struct{}, len(mappings))
	for _, m := range mappings {
		if !m.IsStorable {
			continue
		}
		prefix := basePrefix(m.PrivatePath)
		if prefix == "" {
			continue
		}
		seen[prefix] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	// 排序便于结果稳定（go map 遍历无序）
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i] > out[j] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// basePrefix 从一条 privatePath 提取去重用的对象前缀（含末尾 "."）。
//
//   - "Dev.WiFi.SSID.{i}.Enabled" → "Dev.WiFi.SSID."（截到第一个 {i} 前一段）
//   - "Dev.WiFi.SSID."             → "Dev.WiFi.SSID."（object 原样）
//   - "Dev.System.Mode"            → "Dev.System."
//   - "Dev"                        → ""（无 "."，无意义）
//   - ""                           → ""
func basePrefix(privatePath string) string {
	if privatePath == "" {
		return ""
	}
	// 截到第一个 "{i}" 前面一段含 "."
	if idx := strings.Index(privatePath, "{i}"); idx > 0 {
		return privatePath[:idx]
	}
	if strings.HasSuffix(privatePath, ".") {
		return privatePath
	}
	if idx := strings.LastIndex(privatePath, "."); idx >= 0 {
		return privatePath[:idx+1]
	}
	return ""
}

// instantiateStandardPath 把 template standardPath 中的 {i} 占位符按 actualPrivate 中
// 对应位置的实例号填充。
//
//   actualPrivate    = "Dev.WiFi.SSID.7.Enabled"
//   templatePrivate  = "Dev.WiFi.SSID.{i}.Enabled"
//   templateStandard = "Device.WiFi.SSID.{i}.Enable"
//   返回             = "Device.WiFi.SSID.7.Enable"
//
// 段数不一致或位置不匹配 → 直接返回 templateStandard（容错）。
func instantiateStandardPath(actualPrivate, templatePrivate, templateStandard string) string {
	if !strings.Contains(templateStandard, "{i}") {
		return templateStandard
	}
	actualParts := strings.Split(actualPrivate, ".")
	templateParts := strings.Split(templatePrivate, ".")
	if len(actualParts) != len(templateParts) {
		return templateStandard
	}
	out := templateStandard
	for i := range templateParts {
		if templateParts[i] == "{i}" && i < len(actualParts) {
			out = strings.Replace(out, "{i}", actualParts[i], 1)
		}
	}
	return out
}

