package provision

// sync_pathb_expand.go - Path B 同步的 instance-level 展开。
//
// 背景: Path B 把 storable 前缀整批 GPV 入队让 CPE 一次返回某对象全部实例
// (省去 GPN)。但部分对象(典型: DeviceGSM.Bts. 254 实例 × 70 字段)单次响应
// ParameterValueStruct 数量达 17791 项,序列化进 NATS 事件 ~1.5MB,超过
// NATS 默认 max_payload=1MB → publish 直接失败,整条事件丢弃,订阅者
// (device-rpc-resp-sub / provision) 收不到 → device_parameters 一条不落 →
// 前端 BTS 选择器永远显示空。
//
// 治本方案: 对估算"单 prefix 响应字节数 > 阈值"的 object prefix,在入队前
// 展开成 instance-level prefix 列表(["DeviceGSM.Bts.1.", "DeviceGSM.Bts.2.",
// ..., "DeviceGSM.Bts.N."]),每个 instance prefix 单独入队 GPV task。CPE 收到
// instance prefix 后只返回该实例的 ~70 字段(~5KB),稳稳在 1MB 之内。
//
// hint 估算策略(优先级从高到低):
//  1. DB 已有 max instance 号(滚动学习,首次同步后越来越准)
//  2. hintFloor=256 (首次同步无 DB 历史的兜底; 覆盖 BSC 物理上限 256 BTS)
//  3. 硬上限 maxHintCap=512 (防 estimate 异常膨胀)
//
// hintFloor=256 的副作用过滤: expand 触发条件是 estBytes >= 600KB,即
// fields × 256 × 60B >= 600KB → fields >= 40。小对象(<40 字段)即使 hintFloor=256
// 也不会展开,不产生 SoapFault 9005 浪费。BSC `DeviceGSM.Bts.` (~50 字段) 自然
// 命中展开,首次同步即可工作。
//
// 不存在的 instance prefix CPE 返回 SoapFault 9005,ACS handler.tryRecoverGPVFault
// 自然容错,不阻塞同步。
//
// 弱化语义(Phase 1): 如果实例从 CPE 物理移除(如 BTS.5 下架),DB 残留 Bts.5 不会
// 被自动删除(因为 instance-level reconcile 范围被限定到当前实例内部)。前端会
// 多显示 1 个"幽灵实例",字段是上次同步的旧值。Phase 2 通过日级哨兵 GPN 同步
// + reconcile 多余实例修复。

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
)

// Path B instance expansion 常量。
const (
	// avgFieldBytes: ParameterValueStruct 在 NATS 事件 payload 内的平均 JSON 体积估算
	// (含 Name + Value + Type),实测 BTS 单字段 ~60B。
	avgFieldBytes = 60

	// expandThreshold: 单 object prefix 估算响应字节数超过此阈值即触发展开。
	// NATS 默认 max_payload=1MB,留 ~40% 给事件 envelope(headers + meta) 后实际事件
	// 上限 ~600KB。比这个再小没必要(标量批 50 项 × 60B = 3KB,远未到阈值)。
	expandThreshold = 600 * 1024

	// hintFloor: 首次同步无 DB 历史时,instance 展开数兜底值。
	//
	// 取值 256 来源:
	//  - BSC 物理上限 256 BTS,首次同步必须能覆盖;
	//  - 历史曾用 32,导致 BSC `DeviceGSM.Bts.` cold-start estBytes ≈ 96KB << 600KB
	//    阈值 → 不展开 → CPE 一次返回 ~1.5MB SOAP body → ACS publishRPCResponseEvent
	//    触发 NATS `maximum payload exceeded` → 事件丢失 → device_parameters 无 BTS
	//    实例参数 → 前端 BSC QuickSettings 临区/TRX 空白(死锁: DB 永远学不到 hint);
	//  - 副作用受 expandThreshold 联合过滤: 小对象(<40 字段)仍不展开,不会产生
	//    SoapFault 9005 浪费 round-trip。
	hintFloor = 256

	// maxHintCap: instance 展开数硬上限,防 DB 历史异常(如残留古老脏数据)导致估算
	// 膨胀到几千个 task。BSC 物理上限 256 BTS,留 2 倍裕量。
	maxHintCap = 512
)

// expandLargeObjectPrefixes 把响应规模可能撑爆 NATS 单事件上限的 object prefix
// 展开成 instance-level prefix 列表。
//
// 算法见包注释。入参 prefixes 是 extractStorablePrefixes() 的输出。
// 输出顺序: 未展开的 prefix 原序,展开的 prefix 用 [p1., p2., ..., pN.] 替换原 prefix。
func (s *SyncService) expandLargeObjectPrefixes(
	ctx context.Context,
	deviceID uuid.UUID,
	mappings []parammodel.ParamMapping,
	prefixes []string,
) []string {
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		out = append(out, s.maybeExpandSinglePrefix(ctx, deviceID, mappings, p)...)
	}
	return out
}

// maybeExpandSinglePrefix 处理单个 prefix: 不超阈值返回 [prefix],超过则返回展开列表。
func (s *SyncService) maybeExpandSinglePrefix(
	ctx context.Context,
	deviceID uuid.UUID,
	mappings []parammodel.ParamMapping,
	p string,
) []string {
	// 仅 object prefix (尾点 ".") 可展开;标量参数无所谓
	if !strings.HasSuffix(p, ".") {
		return []string{p}
	}
	// 当前仅保留 BTS 类大对象的实例级展开保护。其它对象（如 Ethernet.Interface）
	// 优先整对象 GPV：让 CPE 直接返回真实实例集合，避免冷启动 hintFloor=256 导致
	// 1..256 的坏路径盲扫风暴。BTS 仍保留展开,因为其整对象响应已被真机验证会打爆
	// NATS max_payload。
	if !shouldExpandByInstance(p) {
		return []string{p}
	}
	mappingFields := countStorableFieldsUnderPrefix(mappings, p)
	hint := s.estimateMaxInstance(ctx, deviceID, p)

	// fields 估算: mapping 字段数往往低估(CPE 实际返回字段含大量 unmapped/storeAsIs),
	// 用 DB 历史 totalRows/maxInst 算实际"每实例平均字段数",取 mapping 与 DB 估算的最大值。
	// CountByPathPrefix 失败 / 0 → 退到 mapping 估算(首次同步 DB 空时如此)。
	fields := mappingFields
	if total, err := s.paramRepo.CountByPathPrefix(ctx, deviceID, p); err == nil && total > 0 {
		instApprox := hint
		if instApprox <= 0 {
			instApprox = 1
		}
		avgFromDB := total / instApprox
		if avgFromDB > fields {
			fields = avgFromDB
		}
	}
	if fields == 0 {
		// mapping 0 字段且 DB 0 行 → 无信息,不展开,任凭 CPE 自然返回
		return []string{p}
	}

	estBytes := avgFieldBytes * fields * hint
	if estBytes < expandThreshold {
		return []string{p}
	}
	if hint > maxHintCap {
		hint = maxHintCap
	}
	expanded := make([]string, 0, hint)
	for i := 1; i <= hint; i++ {
		expanded = append(expanded, p+strconv.Itoa(i)+".")
	}
	if s.logger != nil {
		s.logger.Info("path-b expand: object prefix expanded to instance level",
			zap.String("object_prefix", p),
			zap.Int("fields_per_instance", fields),
			zap.Int("max_instance_hint", hint),
			zap.Int("estimated_single_prefix_bytes", estBytes),
			zap.Int("expanded_count", len(expanded)),
		)
	}
	return expanded
}

func shouldExpandByInstance(prefix string) bool {
	switch prefix {
	case "DeviceGSM.Bts.":
		return true
	default:
		return false
	}
}

// estimateMaxInstance 估算某 object prefix 下 instance 展开数 hint。
//
// 通过窄接口 (instanceHintRepo) type-assertion 拿到 PG 实现的
// MaxInstanceNumberByPrefix。DB 有历史时严格使用 maxSeen；只有接口不支持 /
// 查询失败 / 返回 0 时才用 hintFloor 兜底。
//
// 这种"窄接口 + 可选"设计避免污染主 DeviceParameterRepository(所有 mock/stub 都要改),
// 仅生产 PG 实现需要支持。
func (s *SyncService) estimateMaxInstance(
	ctx context.Context,
	deviceID uuid.UUID,
	prefix string,
) int {
	type instanceHintRepo interface {
		MaxInstanceNumberByPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int, error)
	}
	r, ok := s.paramRepo.(instanceHintRepo)
	if !ok {
		return hintFloor
	}
	maxSeen, err := r.MaxInstanceNumberByPrefix(ctx, deviceID, prefix)
	if err != nil {
		if s.logger != nil {
			s.logger.Debug("path-b expand: MaxInstanceNumberByPrefix failed; using hintFloor",
				zap.String("prefix", prefix),
				zap.Error(err),
			)
		}
		return hintFloor
	}
	if maxSeen <= 0 {
		return hintFloor
	}
	if maxSeen < 1 {
		return 1
	}
	return maxSeen
}

// countStorableFieldsUnderPrefix 统计 mappings 中以 prefix+"{i}." 为路径祖先的
// storable && supported 字段数,反映 CPE 展开 prefix 时单实例会返回多少叶子参数。
//
// 例 prefix="DeviceGSM.Bts." → 匹配前缀 "DeviceGSM.Bts.{i}."
// 命中所有 "DeviceGSM.Bts.{i}.<XXX>" 形态的 mapping。
//
// 注: extractStorablePrefixes 对 Device.Services.FAPService.{i}. 做了 normalize
// 替换成 Device.Services.FAPService.1.,故传入 prefix 可能是 FAPService.1. 形态。
// 这里对 mapping.PrivatePath 也做同样 normalize 后再匹配,保持一致。
func countStorableFieldsUnderPrefix(mappings []parammodel.ParamMapping, prefix string) int {
	matchPrefix := prefix + "{i}."
	cnt := 0
	for _, m := range mappings {
		if !m.IsStorable || !m.IsSupported {
			continue
		}
		normalized := normalizeSingletonFAPServicePath(m.PrivatePath)
		if strings.HasPrefix(normalized, matchPrefix) {
			cnt++
		}
	}
	return cnt
}
