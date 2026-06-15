/**
 * #363 (qa-614 c3)：自定义聚合任务 (粒度, 维度) 组合约束——三皮肤共享单点。
 *
 * 设备组(device_group)维度只物化了 hourly/daily/weekly/monthly 四档预聚合快表
 * (pm_group_metrics_*)，没有 15min 级设备组聚合源。后端 aggregator.SelectTable 对
 * 15min × device_group 硬拒 (ErrUnsupportedQuery)；后端创建/编辑守门也已前移该校验
 * (internal/pm/adhoc/handler.go unsupportedGranularityDimension)。
 *
 * 前端三皮肤向导复用本文件，避免各自硬编码不支持组合，并把约束在向导内做联动禁用 +
 * 提交前校验，不再向后端发起注定失败的请求。
 */
import type { AdhocDimension } from '../types/pmAdhoc';
import type { Granularity } from '../types/pmDashboard';

/** 不支持的 (粒度, 维度) 组合集中表。新增约束只在此处追加一行。 */
export const UNSUPPORTED_GRANULARITY_DIMENSIONS: ReadonlyArray<{
  granularity: Granularity;
  dimension: AdhocDimension;
}> = [{ granularity: '15min', dimension: 'device_group' }];

/**
 * 判断 (粒度, 维度) 组合是否被聚合器支持。
 * @returns true=支持（可建任务）；false=不支持（向导应禁用/拦截）。
 */
export function isGranularityDimensionSupported(
  granularity: string | undefined,
  dimension: AdhocDimension | undefined,
): boolean {
  if (!granularity || !dimension) return true;
  return !UNSUPPORTED_GRANULARITY_DIMENSIONS.some(
    (c) => c.granularity === granularity && c.dimension === dimension,
  );
}
