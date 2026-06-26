/**
 * #669（取代旧 #363）：自定义聚合任务粒度约束——三皮肤共享单点。
 *
 * 历史：旧 #363 仅约束 `15min × device_group` 这一种 (粒度, 维度) 组合不支持，
 * 其他维度 + 15min 一律放行。但实际「continuous + 15min」存在"创建放行、调度器
 * 完成水位闸门永久拒绝放行"的半成品行为（pm_completion_watermarks 表 CHECK 约束
 * 不含 15min、产数链路无 (15min, device) 水位写入）。
 *
 * #669 决策：缩范围——自定义聚合任务最细粒度限定 hourly，15min 整组下线。
 * 15min 原始数据用户改走「指标查询 / 数据提取」即可。
 *
 * 后端 handler 已同步拒绝（internal/pm/adhoc/handler.go unsupportedGranularity）。
 * 前端三皮肤向导：
 *   1. 粒度选项数组直接删除 15min（彻底从下拉/按钮里消失）；
 *   2. 提交前用本约束做兜底校验，防止编辑模式遇旧任务 / 任何绕过路径。
 */
import type { AdhocDimension } from '../types/pmAdhoc';

/** 不再支持的粒度集（向后扩展场景：将来若再砍 5min 等只需追加此数组）。 */
export const UNSUPPORTED_GRANULARITIES: ReadonlyArray<string> = ['15min'];

/**
 * 判断 (粒度, 维度) 组合是否被自定义聚合任务支持。
 *
 * #669 后约束只判粒度（dimension 入参保留是为日后再有「某维度 × 某粒度」类
 * 组合约束时无需改签名再扩展用）。
 *
 * @returns true=支持（可建任务）；false=不支持（向导应禁用提交 / 兜底拦截）。
 */
export function isGranularityDimensionSupported(
  granularity: string | undefined,
  _dimension: AdhocDimension | undefined,
): boolean {
  if (!granularity) return true;
  return !UNSUPPORTED_GRANULARITIES.includes(granularity);
}
