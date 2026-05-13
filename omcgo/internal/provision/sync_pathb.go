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
func (s *SyncService) StartPathBSync(ctx context.Context, dev *model.Device, sourceID string) (bool, error) {
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

	if err := s.enqueueGPVPrefixes(ctx, dev, prefixes, sourceID); err != nil {
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

// ResolveTranslator 公共接口：返回 device 当前 ProductClass + FirmwareVersion 对应的 Translator。
//
// 供 P2-05 orchestrator 模板下发路径使用：模板 Parameters key 视为 standardPath，
// 通过 Translator.ToPrivate 翻译为设备私有路径再下发 SPV / GPV。
//
// 任一前置条件失败 → 返回 (nil, false)，调用方降级到旧栈。
func (s *SyncService) ResolveTranslator(ctx context.Context, dev *model.Device) (*parammodel.Translator, bool) {
	set, ok := s.resolveMappingSet(ctx, dev)
	if !ok {
		return nil, false
	}
	t := parammodel.NewTranslator(set, nil, s.logger)
	if t == nil {
		return nil, false
	}
	return t, true
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

// extractStorablePrefixes 从 ParamMapping 列表抽出可下发的 GPV path 列表。
//
// 算法（设计 §1.11 Path B）：
//   1. 过滤 is_storable=true && is_supported=true（T-0103 后者剔除固件不支持的 path）
//   2. basePrefix 处理每条 privatePath：
//      - 含 "{i}" 模板段 → 截到第一个 "{i}" 前的对象前缀（让 CPE 枚举实例）
//      - 叶子参数、末尾带点对象 → 原样
//   3. 去重排序输出
//
// 设计原则："只查 XML 字典里实际列出的 path"，不从叶子自动派生父对象前缀。
// 历史教训：
//   - basePrefix 曾把叶子 "....UeAccess.Enable" 截成 "....UeAccess."，
//     BAICELLS BaiBLQ_5.0.16.1_1229 固件不识别该对象节点 → 9005 Fault 整批 reject。
//   - 即便不截断，BLQ.xml 字典仍含若干 BAICELLS 固件不支持的叶子（AmbrLimitSwitch /
//     RunningStatus / X_COM_SCTP_CONFIG_MTU / UeAccess.Enable），T-0103 通过 XML
//     supported="false" + is_supported 列在此处过滤。
func extractStorablePrefixes(mappings []parammodel.ParamMapping) []string {
	seen := make(map[string]struct{}, len(mappings))
	for _, m := range mappings {
		if !m.IsStorable || !m.IsSupported {
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

// basePrefix 从一条 privatePath 提取 GPV 下发用的路径。
//
// 只对含 "{i}" 的模板路径做截断（截到第一个 "{i}" 前的对象前缀），其它形态原样返回。
// 不再从叶子参数自动派生父对象前缀，避免基站不识别人为截出的对象节点。
//
//   - "Dev.WiFi.SSID.{i}.Enabled" → "Dev.WiFi.SSID."（截到 "{i}" 前，CPE 枚举实例）
//   - "Dev.WiFi.SSID."             → "Dev.WiFi.SSID."（object 原样）
//   - "Dev.System.Mode"            → "Dev.System.Mode"（叶子原样）
//   - "Dev"                        → "Dev"（无 "."，原样，由基站判定）
//   - ""                           → ""
func basePrefix(privatePath string) string {
	if privatePath == "" {
		return ""
	}
	if idx := strings.Index(privatePath, "{i}"); idx > 0 {
		return privatePath[:idx]
	}
	return privatePath
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

