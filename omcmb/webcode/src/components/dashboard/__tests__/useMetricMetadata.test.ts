/**
 * resolveMetricMeta 单测（KPI-ALL-IND 阶段4）。
 *
 * 覆盖三条解析路径：
 *   1) 指标库命中（编号）→ 名字/单位来自库、换算系数 1、无 i18n 单位 key；
 *   2) 库未命中 + 旧 symbolic 别名 → i18n label/unit + 换算系数 + i18n 单位 key；
 *   3) 都未命中 → 名字回退编号本身、无单位、换算 1。
 */

import { describe, it, expect } from 'vitest';
import { resolveMetricMeta } from '../useMetricMetadata';
import type { MetricMeta, MetricMetadataResult } from '../useMetricMetadata';

// 假元数据查询器：仅识别预置编号。
function fakeMeta(table: Record<string, MetricMeta>): MetricMetadataResult {
  return {
    getMeta: (code: string) => table[code],
    isLoading: false,
  };
}

// 测试用 t()：原样回显 i18n key（断言映射结果即可，不依赖真实语料）。
const echoT = (key: string) => key;

describe('resolveMetricMeta', () => {
  it('指标库命中：名字/单位取自库，换算 1、无 i18n 单位 key', () => {
    const meta = fakeMeta({
      C000010012: { name: '小区下行PRB占用数', unit: 'number', isCounter: true },
    });
    const r = resolveMetricMeta('C000010012', meta, echoT);
    expect(r.name).toBe('小区下行PRB占用数');
    expect(r.unit).toBe('number');
    expect(r.conversion).toBe(1);
    expect(r.unitI18nKey).toBeUndefined();
  });

  it('库未命中但是旧 symbolic 别名：回退老配置，名字/单位经 t() 映射 + i18n 单位 key', () => {
    const meta = fakeMeta({});
    const r = resolveMetricMeta('LTE_PDCP_VOLUME_DL', meta, echoT);
    // 老配置 label/unit 是 i18n key，echoT 原样回显
    expect(r.name).toBe('dashboard.kpi.totalDataVolumeDl');
    expect(r.unit).toBe('unit.gb');
    expect(r.unitI18nKey).toBe('unit.gb');
  });

  it('库与老配置都未命中：名字回退编号本身、无单位、换算 1', () => {
    const meta = fakeMeta({});
    const r = resolveMetricMeta('C999999999', meta, echoT);
    expect(r.name).toBe('C999999999');
    expect(r.unit).toBe('');
    expect(r.conversion).toBe(1);
    expect(r.unitI18nKey).toBeUndefined();
  });

  it('库优先于旧别名：同时存在时取库元数据', () => {
    const meta = fakeMeta({
      LTE_PDCP_VOLUME_DL: { name: 'FROM_LIBRARY', unit: 'GB', isCounter: false },
    });
    const r = resolveMetricMeta('LTE_PDCP_VOLUME_DL', meta, echoT);
    expect(r.name).toBe('FROM_LIBRARY');
    expect(r.unit).toBe('GB');
    expect(r.conversion).toBe(1);
  });
});
