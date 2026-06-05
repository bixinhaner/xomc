/**
 * 任务 C — 自定义聚合结果表「按维度出列」纯函数。
 *
 * 维度 → 首列表头 i18n key / 首列对象名 / 行键 / 是否含「小区/PLMN」列。
 * 与后端 buildResultsQuery / 导出 adhocObjectLabel（任务 B）同口径：
 * 产品名 / 设备组名优先，缺失回退 ID 前 8 位；频段剥 `Band=` 前缀；
 * 全网恒「全网」；临时聚合组出「聚合组·N 台」。只有 device 维度含小区列。
 */
import type { IntlShape } from 'react-intl';
import type { AdhocResultRow, AdhocDimension } from '@core/types/pmAdhoc';

/** 首列列名的 i18n key（按维度）。device 复用既有 colDevice。 */
export function adhocObjectHeaderKey(dimension: AdhocDimension): string {
  switch (dimension) {
    case 'device_group':
      return 'perf.adhoc.colObject.deviceGroup';
    case 'product':
      return 'perf.adhoc.colObject.product';
    case 'band':
      return 'perf.adhoc.colObject.band';
    case 'network':
      return 'perf.adhoc.colObject.network';
    case 'aggregate_group':
      return 'perf.adhoc.colObject.aggregateGroup';
    default:
      return 'perf.adhoc.colDevice';
  }
}

/** 仅 device 维度结果表含「小区/PLMN」列。 */
export function adhocIncludesCell(dimension: AdhocDimension): boolean {
  return dimension === 'device';
}

/** 行键的对象标识部分：按维度取分组键，避免聚合维度行键碰撞。 */
export function objectKeyOf(r: AdhocResultRow, dimension: AdhocDimension): string {
  switch (dimension) {
    case 'network':
      return '__network__';
    case 'product':
      return r.productId || r.deviceSn || '__unknown__';
    case 'band':
    case 'device_group':
    case 'aggregate_group':
      return r.objectLdn || r.deviceSn || '__unknown__';
    default:
      return r.deviceSn || '__unknown__';
  }
}

/**
 * 首列对象名（裸名，列头已带维度词）。
 * 与后端 buildResultsQuery / 导出 adhocObjectLabel 同口径，名缺失回退 ID 前 8 位。
 */
export function adhocObjectName(
  r: AdhocResultRow,
  dimension: AdhocDimension,
  taskDeviceSns: string[],
  intl: IntlShape,
): string {
  switch (dimension) {
    case 'device_group':
      return r.deviceGroupName || stripPrefix(r.objectLdn, 'DeviceGroup=').slice(0, 8);
    case 'product':
      return r.productName || (r.productId ?? '').slice(0, 8);
    case 'band':
      return stripPrefix(r.objectLdn, 'Band=');
    case 'network':
      return intl.formatMessage({ id: 'perf.adhoc.colObject.network' }); // 「全网」
    case 'aggregate_group':
      return intl.formatMessage({ id: 'perf.adhoc.aggregateGroupUnit' }, { count: taskDeviceSns.length });
    default:
      return r.deviceOui ? `${r.deviceOui}/${r.deviceSn}` : r.deviceSn;
  }
}

function stripPrefix(s: string | undefined, prefix: string): string {
  const v = s ?? '';
  return v.startsWith(prefix) ? v.slice(prefix.length) : v;
}
