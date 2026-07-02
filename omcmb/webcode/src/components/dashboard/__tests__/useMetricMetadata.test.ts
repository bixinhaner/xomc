/**
 * resolveMetricMeta 单测。
 *
 * 当前逻辑：直接查指标库元数据（getMeta），命中则返回库的名字/单位，否则 fallback 到编号本身 + 空单位。
 * 不再依赖 KPI_CATALOG 硬编码或 canonicalizeMetricKey 桥接。
 */

import { describe, it, expect } from 'vitest';
import { resolveMetricMeta } from '../useMetricMetadata';
import type { MetricMeta, MetricMetadataResult } from '../useMetricMetadata';

function fakeMeta(table: Record<string, MetricMeta>): MetricMetadataResult {
  return {
    getMeta: (code: string) => table[code],
    isLoading: false,
  };
}

const echoT = (key: string) => key;

describe('resolveMetricMeta', () => {
  it('K 编号命中指标库：返回库的名字和单位', () => {
    const meta = fakeMeta({
      K900010015: { name: '下行数据业务流量', unit: 'MByte', isCounter: false },
    });
    const r = resolveMetricMeta('K900010015', meta, echoT);
    expect(r.name).toBe('下行数据业务流量');
    expect(r.unit).toBe('MByte');
    expect(r.conversion).toBe(1);
  });

  it('百分比指标：单位 % 原样返回', () => {
    const meta = fakeMeta({
      K900010006: { name: '无线初始连接成功率', unit: '%', isCounter: false },
    });
    const r = resolveMetricMeta('K900010006', meta, echoT);
    expect(r.unit).toBe('%');
  });

  it('NR K 编号命中指标库', () => {
    const meta = fakeMeta({
      KGNB0511: { name: 'PDCP下行业务字节数', unit: 'MByte', isCounter: false },
    });
    const r = resolveMetricMeta('KGNB0511', meta, echoT);
    expect(r.name).toBe('PDCP下行业务字节数');
    expect(r.unit).toBe('MByte');
  });

  it('编号不在指标库：名字回退编号本身、无单位、换算 1', () => {
    const meta = fakeMeta({});
    const r = resolveMetricMeta('C999999999', meta, echoT);
    expect(r.name).toBe('C999999999');
    expect(r.unit).toBe('');
    expect(r.conversion).toBe(1);
  });

  it('指标库加载中（isLoading=true）时编号未命中：走 fallback', () => {
    const meta: MetricMetadataResult = { getMeta: () => undefined, isLoading: true };
    const r = resolveMetricMeta('K900010015', meta, echoT);
    expect(r.name).toBe('K900010015');
    expect(r.unit).toBe('');
  });
});

