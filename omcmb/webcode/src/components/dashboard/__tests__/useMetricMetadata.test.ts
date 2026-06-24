/**
 * resolveMetricMeta 单测。
 *
 * 重构后采「可控源优先」三层 fallback：
 *   1) KPI_CATALOG 命中（含旧 symbolic 桥接到主键）→ 直接走 dashboard.kpi.* i18n；
 *   2) 指标库元数据（按 canonicalized key 查）→ 名字/单位来自库；
 *   3) 都未命中 → 名字回退编号本身、无单位、换算 1。
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
  it('catalog 主键（K 编号）命中：走 i18n，即便库里同时有该编号也优先 i18n', () => {
    const meta = fakeMeta({
      K900010015: { name: 'KPI.PdcpUpOctDl', unit: '', isCounter: false },
    });
    const r = resolveMetricMeta('K900010015', meta, echoT);
    expect(r.name).toBe('dashboard.kpi.totalDataVolumeDl');
    expect(r.unit).toBe('unit.gb');
    expect(r.conversion).toBe(1);
  });

  it('旧 symbolic 别名（LTE）→ 桥接到主键、走 i18n', () => {
    const meta = fakeMeta({});
    const r = resolveMetricMeta('LTE_PDCP_VOLUME_DL', meta, echoT);
    expect(r.name).toBe('dashboard.kpi.totalDataVolumeDl');
    expect(r.unit).toBe('unit.gb');
  });

  it('旧 symbolic 别名（NR）→ 走 i18n', () => {
    const meta = fakeMeta({});
    const r = resolveMetricMeta('NR_PDCP_VOLUME_DL', meta, echoT);
    expect(r.name).toBe('dashboard.kpi.totalDataVolumeDl');
    expect(r.unit).toBe('unit.gb');
  });

  it('旧 symbolic 别名（GSM）→ 走 i18n', () => {
    const meta = fakeMeta({});
    const r = resolveMetricMeta('GSM_CALL_SETUP_SR', meta, echoT);
    expect(r.name).toBe('dashboard.kpi.callSetupSr');
    expect(r.unit).toBe('unit.percent');
  });

  it('alias label override：LTE_CELL_AVAILABLE 与 WIRELESS_SETUP_SR 同指主键 K900010006 但显示名不同', () => {
    // K900010006 catalog 默认 label 是 wirelessSetupSr（接入侧主语义）。
    // LTE_CELL_AVAILABLE 沿用老的「小区可用性」语义 → alias 表 override label。
    // WIRELESS_SETUP_SR 与主键默认一致 → 不 override，跟随。
    const meta = fakeMeta({});
    const cellAvail = resolveMetricMeta('LTE_CELL_AVAILABLE', meta, echoT);
    expect(cellAvail.name).toBe('dashboard.kpi.cellAvailable');
    const wirelessSr = resolveMetricMeta('WIRELESS_SETUP_SR', meta, echoT);
    expect(wirelessSr.name).toBe('dashboard.kpi.wirelessSetupSr');
  });

  it('无 K 编号 symbolic（LTE_PDCP_RATE_DL）作 catalog 主键 → 走 i18n', () => {
    const meta = fakeMeta({});
    const r = resolveMetricMeta('LTE_PDCP_RATE_DL', meta, echoT);
    expect(r.name).toBe('dashboard.kpi.throughputDl');
    expect(r.unit).toBe('unit.mbps');
  });

  it('catalog 未登记的编号 + 库命中：走库元数据', () => {
    const meta = fakeMeta({
      C000010012: { name: '小区下行PRB占用数', unit: 'number', isCounter: true },
    });
    const r = resolveMetricMeta('C000010012', meta, echoT);
    expect(r.name).toBe('小区下行PRB占用数');
    expect(r.unit).toBe('number');
    expect(r.conversion).toBe(1);
  });

  it('catalog 未登记的 symbolic 别名 + 库按 canonicalized key 命中：走库元数据', () => {
    // canonicalizeMetricKey 对未登记 key 原样返回，所以库可按 raw key 命中。
    const meta = fakeMeta({
      UNKNOWN_SYMBOLIC: { name: 'FROM_LIBRARY', unit: 'ms', isCounter: false },
    });
    const r = resolveMetricMeta('UNKNOWN_SYMBOLIC', meta, echoT);
    expect(r.name).toBe('FROM_LIBRARY');
    expect(r.unit).toBe('ms');
  });

  it('catalog 与库都未命中：名字回退编号本身、无单位、换算 1', () => {
    const meta = fakeMeta({});
    const r = resolveMetricMeta('C999999999', meta, echoT);
    expect(r.name).toBe('C999999999');
    expect(r.unit).toBe('');
    expect(r.conversion).toBe(1);
  });
});
