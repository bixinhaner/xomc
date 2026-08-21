package provision

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// enabled=false 或 paramReg/prodReg 任一为 nil → 等价于不调用本方法（沿用 datamodel
// 旧栈，零行为差异）。
func (s *SyncService) WithParamRegistry(paramReg *parammodel.Registry, prodReg *product.Registry, enabled bool) *SyncService {
	s.paramRegistry = paramReg
	s.productRegistry = prodReg
	s.paramRegistryEnabled = enabled && paramReg != nil && prodReg != nil
	return s
}

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
// T-0123: opts 支持 WithReason("device_online"/"periodic"/"firmware_changed"/"manual")
// 通过 Redis 临时映射传递到 HandleSyncResultPathB 完成时打差异日志。
//
// 返回 (true, nil) 表示已切到 Path B；(false, nil) 表示无法走新栈，调用方应降级旧栈；
// (false, err) 表示新栈选中后执行出错（不再降级，由 engine 处理）。
func (s *SyncService) StartPathBSync(ctx context.Context, dev *model.Device, sourceID string, opts ...PathBOption) (bool, int, error) {
	var pbOpts pathBOptions
	for _, opt := range opts {
		opt(&pbOpts)
	}
	if s.durableStarter == nil {
		return true, 0, fmt.Errorf("durable parameter sync starter unavailable")
	}
	handled, taskCount, err := s.durableStarter.StartDurableSync(ctx, dev, sourceID, pbOpts.reason, pbOpts.parameterPaths)
	if err != nil {
		return true, taskCount, err
	}
	if !handled {
		return true, taskCount, fmt.Errorf("durable parameter sync did not handle request")
	}
	return true, taskCount, nil
}

// HandleSyncResultPathB 处理新栈 GPV 响应。
//
// 与旧 HandleSyncResult 的差异：
//   - 通过 Translator + MappingValidator 把设备返回的 privatePath 翻译为 standardPath
//   - is_storable=false 的条目直接丢弃（不写入参数仓库）
//   - 翻译未命中的条目降级为原 privatePath 写入（容错），加 WARN log
//
// T-0127: BatchUpsert 完成后对比"DB 已有 standardPath"与"本次落地 standardPath"，
// 把差集（之前上报过、本次未上报）写应用日志便于运维追溯参数漂移。
//
// line 52 (2026-05-20): 全量同步对象级差集删除 — 基于本次响应自身推导"对象 prefix"
// （path 中最深的数字段之前的部分），仅对多实例对象做差集对账：本次响应已返回该对象的
// 部分实例时，把 DB 中同 prefix 但本次未返回的其他实例物理删除。
//
//   - 仅对至少出现过一次"实例号段"的 path 推导 prefix（叶子 path 不参与 reconcile）
//   - 完全没返回任何实例的对象（整个对象被删）→ 推导不出 prefix → DB 残留（保守，避免误删）
//   - 不依赖外部 requestedPrefixes，避免按 batch 拆分时根级前缀爆炸的误删风险
//
// 返回 nil 即视为成功；底层 BatchUpsert 错误向上传。
func (s *SyncService) HandleSyncResultPathB(ctx context.Context, dev *model.Device,
	paramValues []tr069.ParameterValueStruct, triggerCommandKey string) (bool, error) {
	fullSyncTrigger := isFullSyncTrigger(triggerCommandKey)
	finalizeSync := true
	var matchedProduct *product.Product
	if fullSyncTrigger {
		finalizeSync = s.shouldFinalizePathBSync(ctx, dev, triggerCommandKey)
		matchedProduct, _ = s.resolveMatchedProduct(ctx, dev)
	}

	if len(paramValues) == 0 {
		if !fullSyncTrigger {
			return false, nil
		}
		if finalizeSync {
			s.finalizePathBSyncAndLog(dev, fullSyncTrigger)
		}
		return true, nil
	}

	set, ok := s.resolveMappingSet(ctx, dev)
	if !ok {
		return false, nil
	}
	validator := parammodel.NewMappingValidator(set)
	if validator == nil {
		return false, nil
	}

	// T-0127: 同步前 snapshot 现有 standardPath 集合 B（供差集计算）。
	prevPaths := s.snapshotStandardPaths(ctx, dev.ID)
	if fullSyncTrigger && matchedProduct != nil {
		prevPaths = s.filterReadUnsupportedPathSet(ctx, matchedProduct.ID, prevPaths)
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

	// T-0127: 差异日志 — 把"之前上报、本次未上报"的 standardPath 差集写应用日志。
	// 仅 full-sync 的 sync-gpv 收尾才记录。MML LST、SPV 后置 GPV、北向调试 GPV
	// 都是局部读取，响应集合不代表设备全量参数，不能触发 param_sync_missing。
	if fullSyncTrigger && finalizeSync {
		s.logPathBSyncDiff(ctx, dev, prevPaths, params)
	}

	// line 52 (2026-05-20): 全量同步对象级差集删除 — 基于本次响应数据自动推导对象 prefix，
	// 在该 prefix 范围内对账（DB 有但 CPE 未返回 → 物理删除）。设计原则：
	//   - CPE 应答 SOAP Fault / 超时 → 上游不调本函数，不会触发删除
	//   - CPE 应答 Success 但 paramValues 空 → 函数顶部早返回，不删
	//   - 仅对响应中"含实例号"的多实例对象 path 推导 prefix，叶子 path 不参与（reconcile 语义不适用）
	//   - 完全没返回任何实例 → 推导不出 prefix → DB 残留（保守，避免空响应误删）
	//
	// 2026-05-21 防御补丁：reconcile 仅在显式"全量同步"上下文执行 — 即入队方
	// command_key 以 "sync-gpv-" 开头(SyncService.StartSync/enqueueGPVPrefixes 走的
	// manual/periodic/online-trigger Path B 路径)。其他 GPV(如 auto-gpv-after-spv-*、
	// auto-gpv-after-addobject-*) 只拉局部 path，CPE 响应集合不代表 prefix 全集合，
	// 套用 reconcile 会把同 prefix 下的兄弟实例误判为"missing"全删。
	if len(prevPaths) > 0 && isFullSyncTrigger(triggerCommandKey) {
		s.reconcileDeletedPaths(ctx, dev, prevPaths, params)
	}

	// T-0124: 回写 last_param_sync_at（统一口径，不区分触发源）。
	// 仅在 BatchUpsert 成功（含本次未变化时 len(params)=0 也算成功）后回写；
	// GPV 部分失败 / 超时 / 取消时不回写，让下一轮周期或上线重试自动覆盖。
	if finalizeSync {
		s.finalizePathBSyncAndLog(dev, fullSyncTrigger)
	}

	return true, nil
}

const pathBSyncPendingBatchesTTL = 10 * time.Minute

func pathBSyncPendingBatchesKey(deviceID uuid.UUID) string {
	return fmt.Sprintf("provision:pathb:pending:%s", deviceID.String())
}

func (s *SyncService) recordPathBSyncPendingBatches(ctx context.Context, deviceID uuid.UUID, totalBatches int) {
	if s.redisClient == nil || deviceID == uuid.Nil || totalBatches <= 0 {
		return
	}
	if err := s.redisClient.Set(ctx, pathBSyncPendingBatchesKey(deviceID), totalBatches, pathBSyncPendingBatchesTTL).Err(); err != nil {
		s.logger.Warn("record path-b pending batches failed",
			zap.String("device_id", deviceID.String()),
			zap.Int("total_batches", totalBatches),
			zap.Error(err))
	}
}

func (s *SyncService) shouldFinalizePathBSync(ctx context.Context, dev *model.Device, triggerCommandKey string) bool {
	if !isFullSyncTrigger(triggerCommandKey) || dev == nil {
		return true
	}
	if s.pathBSyncTaskReader != nil {
		hasOpen, err := s.pathBSyncTaskReader.HasIncompleteSyncGPVTasksByDevice(ctx, dev.SerialNumber)
		if err != nil {
			s.logger.Warn("query incomplete path-b tasks failed",
				zap.String("device_id", dev.ID.String()),
				zap.String("device_sn", dev.SerialNumber),
				zap.Error(err))
		} else {
			return !hasOpen
		}
	}
	deviceID := dev.ID
	if s.redisClient == nil || deviceID == uuid.Nil {
		return true
	}
	remaining, err := s.redisClient.Decr(ctx, pathBSyncPendingBatchesKey(deviceID)).Result()
	if err != nil {
		s.logger.Warn("decrement path-b pending batches failed",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
		return true
	}
	if remaining > 0 {
		return false
	}
	if err := s.redisClient.Del(ctx, pathBSyncPendingBatchesKey(deviceID)).Err(); err != nil {
		s.logger.Warn("delete path-b pending batches key failed",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
	}
	return true
}

func (s *SyncService) finalizePathBSync(ctx context.Context, dev *model.Device) error {
	if s.redisClient != nil && dev != nil && dev.ID != uuid.Nil {
		if err := s.redisClient.Del(ctx, pathBSyncPendingBatchesKey(dev.ID)).Err(); err != nil {
			s.logger.Warn("delete path-b pending batches key failed during finalize",
				zap.String("device_id", dev.ID.String()),
				zap.Error(err))
		}
	}
	if s.deviceInfoRefresher != nil && !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(dev.ProductClass)), "UPS") {
		if _, err := s.deviceInfoRefresher.SyncFromParameters(ctx, dev.ID, dev.Carrier, dev.Technology, dev.ProductClass); err != nil {
			s.logger.Warn("refresh device_info from parameters failed (non-fatal)",
				zap.String("device_id", dev.ID.String()),
				zap.String("device_sn", dev.SerialNumber),
				zap.Error(err))
		}
	}
	if s.paramSyncWriter != nil {
		if err := s.paramSyncWriter.UpdateLastParamSyncAt(ctx, dev.ID, time.Now()); err != nil {
			s.logger.Warn("update last_param_sync_at failed",
				zap.String("device_id", dev.ID.String()),
				zap.Error(err))
		} else {
			s.logger.Debug("last_param_sync_at written",
				zap.String("device_id", dev.ID.String()))
		}
	}

	// Issue #758: 设备名称同步钩子 — 检测 LMT 名称与网管名称是否一致，按配置方向同步
	if s.deviceNameSyncHook != nil {
		if err := s.deviceNameSyncHook.Execute(ctx, dev); err != nil {
			s.logger.Warn("device name sync hook failed (non-fatal)",
				zap.String("device_id", dev.ID.String()),
				zap.String("device_sn", dev.SerialNumber),
				zap.Error(err))
		}
	}

	return nil
}

func (s *SyncService) finalizePathBSyncAndLog(dev *model.Device, fullSyncTrigger bool) {
	if err := s.finalizePathBSync(context.Background(), dev); err != nil {
		s.logger.Warn("finalize path-b sync failed",
			zap.String("device_id", dev.ID.String()),
			zap.String("device_sn", dev.SerialNumber),
			zap.Bool("full_sync_trigger", fullSyncTrigger),
			zap.Error(err))
	}
}

// snapshotStandardPaths 拉取 device_parameters 中本设备的 standardPath 集合（T-0127 差异日志用）。
//
// 失败时返回 nil（差异日志降级跳过，不阻塞主流程）。
func (s *SyncService) snapshotStandardPaths(ctx context.Context, deviceID uuid.UUID) map[string]struct{} {
	existing, err := s.paramRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		s.logger.Debug("snapshot standardPaths failed (diff log skipped)",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
		return nil
	}
	if len(existing) == 0 {
		return nil
	}
	paths := make(map[string]struct{}, len(existing))
	for _, p := range existing {
		paths[p.ParameterPath] = struct{}{}
	}
	return paths
}

// logPathBSyncDiff 比对 BatchUpsert 前后的 standardPath 集合差集并打应用日志（T-0127）。
//
// 集合 B = prevPaths（同步前 snapshot）；集合 A = 本次 Upsert 的 standardPath。
// 差集 = B − A（之前上报、本次未上报）。
//
// 跳过场景：
//   - prevPaths 为空（首次同步 / B 为空）— 避免噪声
//   - missing 为空（所有旧路径都重新上报）— 无漂移
//
// reason 通过 Redis 临时映射 provision:syncreason:{deviceID} 读取（StartPathBSync 入队时写入，TTL 10min）；
// 读取失败或 key 不存在则 reason = "unknown"。
//
// 异常小子集（current < 0.5 × prev）日志级别提升到 Warn 提示可能 GPV 不完整。
func (s *SyncService) logPathBSyncDiff(ctx context.Context, dev *model.Device,
	prevPaths map[string]struct{}, params []model.DeviceParameter) {
	if len(prevPaths) == 0 {
		return
	}
	currentPaths := make(map[string]struct{}, len(params))
	for _, p := range params {
		currentPaths[p.ParameterPath] = struct{}{}
	}
	missing := make([]string, 0)
	for path := range prevPaths {
		if _, ok := currentPaths[path]; !ok {
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		return
	}

	// 截断采样到前 20 条
	sample := missing
	if len(sample) > 20 {
		sample = sample[:20]
	}

	// 读 reason（best-effort，失败降级为 "unknown"）
	reason := "unknown"
	if s.redisClient != nil {
		key := fmt.Sprintf("provision:syncreason:%s", dev.ID.String())
		if val, err := s.redisClient.Get(ctx, key).Result(); err == nil && val != "" {
			reason = val
		}
	}

	prevTotal := len(prevPaths)
	currentTotal := len(currentPaths)
	missingCount := len(missing)

	// 异常小子集 → Warn；常规 → Info
	fields := []zap.Field{
		zap.String("device_id", dev.ID.String()),
		zap.String("device_sn", dev.SerialNumber),
		zap.String("reason", reason),
		zap.Int("prev_total", prevTotal),
		zap.Int("current_total", currentTotal),
		zap.Int("missing_count", missingCount),
		zap.Strings("missing_paths_sample", sample),
	}
	if currentTotal < prevTotal/2 {
		s.logger.Warn("param_sync_missing", fields...)
	} else {
		s.logger.Info("param_sync_missing", fields...)
	}
}

// reconcileDeletedPaths 在 GPV 全量同步成功后，基于本次响应数据自动推导"对象 prefix"，
// 在该 prefix 范围内对账：把 "DB 有但 CPE 未返回" 的 standardPath 物理删除（line 52 全量同步语义）。
//
// 对象 prefix 推导（deriveObjectPrefixesFromParams）：取 path 中最深"数字段"之前的部分，
// 例 "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.Pci"
// → "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."
// 这天然把范围限定为"本次响应实际覆盖到的多实例对象"，避免跨 batch 误删。
//
// 设计原则：
//   - 仅对响应中"含实例号"的多实例对象 path 推导 prefix；叶子 path（如 Device.System.Mode）
//     不参与 reconcile（reconcile 语义不适用：单值参数没有"实例被删"概念）
//   - 完全没返回任何实例的对象 → 推导不出 prefix → DB 残留（保守，避免空响应误删）
//   - 每条 missing 是 leaf 级 standardPath，调 DeleteByPathPrefix(path) 精确删 1 行
//
// T-NATS-PAYLOAD: 配合 sync_pathb_expand.go 的 instance 展开,reconcile 推导改用
// deriveReconcilePrefixes(batch-aware): 单实例 batch 推导 instance-level prefix
// (DeviceGSM.Bts.5.) 而非 object-level prefix (DeviceGSM.Bts.),避免 expand 后
// 每个 instance batch 误删兄弟实例。
//
// 错误降级：DeleteByPathPrefix 失败仅打 Warn，不阻断主流程（下次同步还有机会修复）。
func (s *SyncService) reconcileDeletedPaths(ctx context.Context, dev *model.Device,
	prevPaths map[string]struct{}, params []model.DeviceParameter) {
	objectPrefixes := deriveReconcilePrefixes(params)
	if len(objectPrefixes) == 0 {
		return
	}

	currentInScope := make(map[string]struct{})
	for _, p := range params {
		if pathInAnyPrefix(p.ParameterPath, objectPrefixes) {
			currentInScope[p.ParameterPath] = struct{}{}
		}
	}

	missing := make([]string, 0)
	for path := range prevPaths {
		if !pathInAnyPrefix(path, objectPrefixes) {
			continue
		}
		if _, ok := currentInScope[path]; ok {
			continue
		}
		missing = append(missing, path)
	}
	if len(missing) == 0 {
		return
	}

	sample := missing
	if len(sample) > 20 {
		sample = sample[:20]
	}
	s.logger.Info("path-b reconcile: deleting paths missing from CPE response",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("device_id", dev.ID.String()),
		zap.Int("missing_count", len(missing)),
		zap.Int("object_prefixes_count", len(objectPrefixes)),
		zap.Strings("missing_sample", sample),
	)

	deleted := int64(0)
	for _, path := range missing {
		n, err := s.paramRepo.DeleteByPathPrefix(ctx, dev.ID, path)
		if err != nil {
			s.logger.Warn("path-b reconcile: delete failed",
				zap.String("device_sn", dev.SerialNumber),
				zap.String("path", path),
				zap.Error(err))
			continue
		}
		deleted += n
	}
	s.logger.Info("path-b reconcile: deletion done",
		zap.String("device_sn", dev.SerialNumber),
		zap.Int("missing_count", len(missing)),
		zap.Int64("deleted_rows", deleted),
	)
}

// deriveObjectPrefixesFromParams 从响应 path 集合推导"对象 prefix"列表。
//
// 算法：每条 path 取最深"数字段"之前的部分（含尾点）。例：
//   - "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.Pci"
//     → "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."
//     （取最深的数字段 "2" 之前的部分）
//   - "Device.System.Mode" → ""（无数字段，叶子参数不参与 reconcile）
//
// 输出去重。
func deriveObjectPrefixesFromParams(params []model.DeviceParameter) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, p := range params {
		prefix := nearestObjectPrefix(p.ParameterPath)
		if prefix == "" {
			continue
		}
		if _, ok := seen[prefix]; !ok {
			seen[prefix] = struct{}{}
			out = append(out, prefix)
		}
	}
	return out
}

// minPrefixSegments 是 nearestObjectPrefix 返回 prefix 的最小段数门槛。
//
// 实测教训 (2026-05-20)：BLQ 真机有 "Device.DeviceInfo.2.UE_Count" 这种"第二段就出现数字"
// 的路径，推导出 prefix "Device.DeviceInfo."（2 段），覆盖整个 DeviceInfo 子树（含数百
// 兄弟叶子），跨 batch 时误删大量行。门槛 >= 4 段（如 "Device.Services.FAPService.X."），
// 锁定真正的"深层多实例对象"，浅级实例（根节点直接子层）不参与 reconcile。
const minPrefixSegments = 4

// nearestObjectPrefix 把 path 截到最深"数字段"之前的部分（含尾点）。
//   - "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.2.Pci"
//     → "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."（深 prefix，OK）
//   - "Device.DeviceInfo.2.UE_Count" → ""（数字段位置太浅 i<4，弃之防误删）
//   - "Device.System.Mode" → ""（无数字段）
//   - "Device.X.Y.Z.1." → "Device.X.Y.Z."（path 自身末尾是数字段，仍按位置判断）
func nearestObjectPrefix(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(strings.TrimSuffix(path, "."), ".")
	for i := len(parts) - 1; i >= 0; i-- {
		if isPositiveInteger(parts[i]) {
			if i < minPrefixSegments {
				return ""
			}
			return strings.Join(parts[:i], ".") + "."
		}
	}
	return ""
}

// splitObjectAndInstancePrefix 把 path 同时拆出 (objectPrefix, instancePrefix)。
//
// 算法同 nearestObjectPrefix(取最深数字段),并额外返回"含数字段+尾点"的 instance prefix:
//   - "DeviceGSM.Bts.5.CellId"             → ("DeviceGSM.Bts.", "DeviceGSM.Bts.5.")
//   - "DeviceGSM.Bts.5.Trx.1.Rf"           → ("DeviceGSM.Bts.5.Trx.", "DeviceGSM.Bts.5.Trx.1.")
//     (最深数字段是 Trx 下的 "1",不是 Bts.5)
//   - "Device.System.Mode"                 → ("", "")
//   - 数字段位置 < minPrefixSegments(4)     → ("", "")
//
// 第二个返回值用于"单实例 batch reconcile 范围精化",见 deriveReconcilePrefixes。
func splitObjectAndInstancePrefix(path string) (string, string) {
	if path == "" {
		return "", ""
	}
	parts := strings.Split(strings.TrimSuffix(path, "."), ".")
	for i := len(parts) - 1; i >= 0; i-- {
		if isPositiveInteger(parts[i]) {
			if i < minPrefixSegments {
				return "", ""
			}
			objPrefix := strings.Join(parts[:i], ".") + "."
			instPrefix := strings.Join(parts[:i+1], ".") + "."
			return objPrefix, instPrefix
		}
	}
	return "", ""
}

// deriveReconcilePrefixes 智能推导 reconcile 范围 prefix 列表(batch-aware)。
//
// 配合 sync_pathb_expand.go 的 instance 展开:
//   - object batch (CPE 一次返多 instance,如 LTECell.{2,3,5} 都有 path):
//     输出 object-level prefix "...LTECell.",reconcile 范围覆盖整对象(可删该对象未上报的实例)
//   - instance batch (来自 expand,batch 内只有 1 个 instance,如全部都属 Bts.5):
//     输出 instance-level prefix "DeviceGSM.Bts.5.",reconcile 范围限定到该实例内部
//     (不会越界删兄弟 Bts.1..4 / 6..N 的字段)
//
// 算法:按 (objectPrefix → set{instancePrefix}) 分组,每组:
//
//	|set| == 1 → 输出 instancePrefix (单实例,精化)
//	|set| > 1  → 输出 objectPrefix   (多实例,沿用原逻辑)
//
// 弱化语义提示: instance batch 模式下 instance 整体被 CPE 删除(没有任何字段返回)
// 是"幽灵实例"残留场景,本算法无法感知;由 Phase 2 哨兵 GPN 同步处理。
func deriveReconcilePrefixes(params []model.DeviceParameter) []string {
	type instSet = map[string]struct{}
	groups := make(map[string]instSet)
	for _, p := range params {
		obj, inst := splitObjectAndInstancePrefix(p.ParameterPath)
		if obj == "" || inst == "" {
			continue
		}
		if _, ok := groups[obj]; !ok {
			groups[obj] = make(instSet)
		}
		groups[obj][inst] = struct{}{}
	}
	out := make([]string, 0, len(groups))
	for obj, set := range groups {
		if len(set) == 1 {
			for inst := range set {
				out = append(out, inst)
			}
			continue
		}
		out = append(out, obj)
	}
	return out
}

// isFullSyncTrigger 判定本次 GPV 响应是否来自"全量 Path B 同步"上下文。
// sync-gpv-partial-* 仍由 Path B 翻译和 ACS Fault 自愈处理，但响应只覆盖显式
// 请求范围，不能参与全量缺失日志或对象实例删除对账。
const partialSyncGPVCommandKeyPrefix = "sync-gpv-partial-"

func isFullSyncTrigger(commandKey string) bool {
	return strings.HasPrefix(commandKey, "sync-gpv-") &&
		!strings.HasPrefix(commandKey, partialSyncGPVCommandKeyPrefix)
}

func isPositiveInteger(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// pathInAnyPrefix 判断 path 是否落在任一 prefix 范围内（HasPrefix 匹配）。
func pathInAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
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

func (s *SyncService) resolveMatchedProduct(ctx context.Context, dev *model.Device) (*product.Product, bool) {
	if !s.paramRegistryEnabled || s.productRegistry == nil {
		return nil, false
	}
	if dev == nil || dev.ProductClass == "" {
		return nil, false
	}
	match, err := s.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil {
		return nil, false
	}
	return match.Product, true
}

// resolveMappingSet 在 dual-stack 启用时尝试解析 device 对应的 MappingSet。
//
// 任何前置条件失败 → 返回 (nil, false)。
func (s *SyncService) resolveMappingSet(ctx context.Context, dev *model.Device) (*parammodel.MappingSet, bool) {
	if !s.paramRegistryEnabled || s.paramRegistry == nil {
		return nil, false
	}
	matchedProduct, ok := s.resolveMatchedProduct(ctx, dev)
	if !ok {
		return nil, false
	}
	set, err := s.paramRegistry.GetByProduct(ctx, matchedProduct.ID, dev.FirmwareVersion)
	if err != nil || set == nil {
		return nil, false
	}
	return set, true
}

func (s *SyncService) filterReadUnsupportedMappings(ctx context.Context, productID uuid.UUID, mappings []parammodel.ParamMapping) []parammodel.ParamMapping {
	blocked := s.readUnsupportedPathSet(ctx, productID)
	if len(blocked) == 0 || len(mappings) == 0 {
		return mappings
	}
	filtered := make([]parammodel.ParamMapping, 0, len(mappings))
	skipped := 0
	for _, mapping := range mappings {
		if _, ok := blocked[mapping.StandardPath]; ok {
			skipped++
			continue
		}
		filtered = append(filtered, mapping)
	}
	if skipped > 0 {
		s.logger.Info("path-b sync filtered read-unsupported mappings",
			zap.String("product_id", productID.String()),
			zap.Int("filtered_mappings", skipped),
			zap.Int("remaining_mappings", len(filtered)))
	}
	return filtered
}

func (s *SyncService) filterReadUnsupportedPathSet(ctx context.Context, productID uuid.UUID, paths map[string]struct{}) map[string]struct{} {
	blocked := s.readUnsupportedPathSet(ctx, productID)
	if len(blocked) == 0 || len(paths) == 0 {
		return paths
	}
	filtered := make(map[string]struct{}, len(paths))
	for path := range paths {
		if _, ok := blocked[path]; ok {
			continue
		}
		filtered[path] = struct{}{}
	}
	return filtered
}

func (s *SyncService) readUnsupportedPathSet(ctx context.Context, productID uuid.UUID) map[string]struct{} {
	if s.unsupportedPathRepo == nil || productID == uuid.Nil {
		return nil
	}
	unsupported, err := s.unsupportedPathRepo.ListByProduct(ctx, productID)
	if err != nil {
		s.logger.Warn("load product unsupported paths failed",
			zap.String("product_id", productID.String()),
			zap.Error(err))
		return nil
	}
	blocked := make(map[string]struct{}, len(unsupported))
	for _, item := range unsupported {
		if !item.ReadUnsupported || item.Path == "" {
			continue
		}
		blocked[item.Path] = struct{}{}
	}
	if len(blocked) == 0 {
		return nil
	}
	return blocked
}

// ── 纯函数辅助（便于单测） ─────────────────────────────────────────────

// extractStorablePrefixes 从 ParamMapping 列表抽出可下发的 GPV path 列表。
//
// 算法（设计 §1.11 Path B）：
//  1. 过滤 is_storable=true && is_supported=true（T-0103 后者剔除固件不支持的 path）
//  2. basePrefix 处理每条 privatePath：
//     - 含 "{i}" 模板段 → 截到第一个 "{i}" 前的对象前缀（让 CPE 枚举实例）
//     - 叶子参数、末尾带点对象 → 原样
//  3. 去重排序输出
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
		prefix := basePrefix(normalizeSingletonFAPServicePath(m.PrivatePath))
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

// PathBStorablePrefixes exposes the established Path B selection algorithm to
// the durable parameter-sync scheduler. Keeping one implementation prevents
// the legacy and durable schedulers from drifting on template normalization.
func PathBStorablePrefixes(mappings []parammodel.ParamMapping) []string {
	return extractStorablePrefixes(mappings)
}

func extractStorablePrefixesForStandardPaths(mappings []parammodel.ParamMapping, standardPaths []string) []string {
	seen := make(map[string]struct{}, len(mappings))
	for _, m := range mappings {
		if !m.IsStorable || !m.IsSupported {
			continue
		}
		prefix, ok := scopedPrefixForStandardPaths(m.PrivatePath, m.StandardPath, standardPaths)
		if !ok {
			continue
		}
		if prefix == "" {
			continue
		}
		seen[prefix] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i] > out[j] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// PathBStorablePrefixesForStandardPaths exposes the established scoped Path B
// selection algorithm for partial/readback durable syncs.
func PathBStorablePrefixesForStandardPaths(mappings []parammodel.ParamMapping, standardPaths []string) []string {
	return extractStorablePrefixesForStandardPaths(mappings, standardPaths)
}

func scopedPrefixForStandardPaths(privatePath, standardPath string, targets []string) (string, bool) {
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if !standardPathMatches(standardPath, target) {
			continue
		}
		return scopedPrivatePrefixForStandardTarget(privatePath, standardPath, target), true
	}
	return "", false
}

func scopedPrivatePrefixForStandardTarget(privatePath, standardPath, target string) string {
	privatePath = normalizeSingletonFAPServicePath(privatePath)
	if target == "" {
		return ""
	}
	if strings.HasSuffix(target, ".") {
		return basePrefix(instantiatePrivateObjectPrefix(privatePath, standardPath, target))
	}
	resolved := instantiatePrivatePathFromStandardTarget(privatePath, standardPath, target)
	if strings.Contains(resolved, "{i}") {
		return basePrefix(resolved)
	}
	return resolved
}

func instantiatePrivateObjectPrefix(privatePath, standardPath, targetPrefix string) string {
	privateParts := strings.Split(strings.TrimSuffix(privatePath, "."), ".")
	standardParts := strings.Split(strings.TrimSuffix(standardPath, "."), ".")
	targetParts := strings.Split(strings.TrimSuffix(targetPrefix, "."), ".")
	if len(targetParts) > len(standardParts) {
		return privatePath
	}
	instances := make([]string, 0, 2)
	for i := range targetParts {
		if standardParts[i] == "{i}" && targetParts[i] != "{i}" {
			instances = append(instances, targetParts[i])
		}
	}
	end := correspondingPathTemplateEnd(standardParts, privateParts, len(targetParts)-1)
	if end < 0 {
		return privatePath
	}
	out := append([]string(nil), privateParts[:end+1]...)
	instanceIndex := 0
	for i := range out {
		if out[i] == "{i}" && instanceIndex < len(instances) {
			out[i] = instances[instanceIndex]
			instanceIndex++
		}
	}
	return strings.Join(out, ".") + "."
}

func correspondingPathTemplateEnd(from, to []string, fromEnd int) int {
	if fromEnd < 0 || fromEnd >= len(from) {
		return -1
	}
	if from[fromEnd] == "{i}" {
		ordinal := 0
		for i := 0; i <= fromEnd; i++ {
			if from[i] == "{i}" {
				ordinal++
			}
		}
		seen := 0
		for i, part := range to {
			if part == "{i}" {
				seen++
				if seen == ordinal {
					return i
				}
			}
		}
		return -1
	}
	for i := len(to) - 1; i >= 0; i-- {
		if to[i] == from[fromEnd] {
			return i
		}
	}
	return -1
}

func instantiatePrivatePathFromStandardTarget(privatePath, standardPath, target string) string {
	privateParts := strings.Split(privatePath, ".")
	standardParts := strings.Split(standardPath, ".")
	targetParts := strings.Split(target, ".")
	if len(privateParts) != len(standardParts) || len(standardParts) != len(targetParts) {
		return privatePath
	}
	out := append([]string(nil), privateParts...)
	for i := range standardParts {
		if standardParts[i] == "{i}" && targetParts[i] != "{i}" {
			out[i] = targetParts[i]
		}
	}
	return strings.Join(out, ".")
}

func standardPathMatches(template, target string) bool {
	if template == "" || target == "" {
		return false
	}
	if template == target {
		return true
	}
	if strings.HasSuffix(target, ".") {
		return standardPathPrefixMatches(template, target)
	}
	templateParts := strings.Split(strings.TrimSuffix(template, "."), ".")
	targetParts := strings.Split(strings.TrimSuffix(target, "."), ".")
	if len(templateParts) != len(targetParts) {
		return false
	}
	for i := range templateParts {
		if templateParts[i] == "{i}" || targetParts[i] == "{i}" {
			continue
		}
		if templateParts[i] != targetParts[i] {
			return false
		}
	}
	return true
}

func standardPathPrefixMatches(template, targetPrefix string) bool {
	templateParts := strings.Split(strings.TrimSuffix(template, "."), ".")
	targetParts := strings.Split(strings.TrimSuffix(targetPrefix, "."), ".")
	if len(targetParts) > len(templateParts) {
		return false
	}
	for i := range targetParts {
		if templateParts[i] == "{i}" || targetParts[i] == "{i}" {
			continue
		}
		if templateParts[i] != targetParts[i] {
			return false
		}
	}
	return true
}

// normalizeSingletonFAPServicePath replaces the leading FAPService instance
// placeholder with a concrete singleton instance before basePrefix() runs.
//
// Without this normalization, paths such as:
//   - Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.
//   - Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID
//
// would both be truncated at the first {i} and collapse to the over-broad
// prefix "Device.Services.FAPService.", which lets unsupported NR children fault
// otherwise valid LTE fetches.
func normalizeSingletonFAPServicePath(privatePath string) string {
	const templ = "Device.Services.FAPService.{i}."
	const inst = "Device.Services.FAPService.1."
	if !strings.HasPrefix(privatePath, templ) {
		return privatePath
	}
	return strings.Replace(privatePath, templ, inst, 1)
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
//	actualPrivate    = "Dev.WiFi.SSID.7.Enabled"
//	templatePrivate  = "Dev.WiFi.SSID.{i}.Enabled"
//	templateStandard = "Device.WiFi.SSID.{i}.Enable"
//	返回             = "Device.WiFi.SSID.7.Enable"
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
