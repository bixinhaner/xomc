import { describe, it, expect } from 'vitest';
import { resolveNetworkTypeLabel, normalizeNetworkTypeFilter } from '../networkType';
import type { NetworkTypeDictDetail } from '../networkType';

// 模拟 network_type 字典（与 seed 000001 一致：value=lte/nr，label 带空格）
const DICT: NetworkTypeDictDetail[] = [
  { label: 'eNB (LTE)', value: 'lte' },
  { label: 'gNB (NR)', value: 'nr' },
];

describe('resolveNetworkTypeLabel', () => {
  it('把设备字段 eNB 映射为字典 label eNB (LTE)', () => {
    expect(resolveNetworkTypeLabel('eNB', DICT)).toBe('eNB (LTE)');
  });

  it('把设备字段 gNB 映射为字典 label gNB (NR)', () => {
    expect(resolveNetworkTypeLabel('gNB', DICT)).toBe('gNB (NR)');
  });

  it('设备列表与回收站传同一字段值得到同一文案（同源一致）', () => {
    expect(resolveNetworkTypeLabel('eNB', DICT)).toBe(resolveNetworkTypeLabel('eNB', DICT));
    expect(resolveNetworkTypeLabel('gNB', DICT)).toBe(resolveNetworkTypeLabel('gNB', DICT));
  });

  it('字段值已是字典 value（lte/nr）时也能命中', () => {
    expect(resolveNetworkTypeLabel('lte', DICT)).toBe('eNB (LTE)');
    expect(resolveNetworkTypeLabel('nr', DICT)).toBe('gNB (NR)');
  });

  it('空字段返回占位符 -', () => {
    expect(resolveNetworkTypeLabel('', DICT)).toBe('-');
    expect(resolveNetworkTypeLabel(undefined, DICT)).toBe('-');
    expect(resolveNetworkTypeLabel(null, DICT)).toBe('-');
  });

  it('字典缺失时回退原始字段值（不显示空白）', () => {
    expect(resolveNetworkTypeLabel('eNB', undefined)).toBe('eNB');
    expect(resolveNetworkTypeLabel('gNB', [])).toBe('gNB');
  });

  it('字典里无匹配项时回退原始字段值', () => {
    expect(resolveNetworkTypeLabel('CPE', DICT)).toBe('CPE');
    expect(resolveNetworkTypeLabel('GSM', DICT)).toBe('GSM');
  });
});

describe('normalizeNetworkTypeFilter (#443)', () => {
  it('把制式码 lte/nr/gsm 归一为 device.networkType 存储值 eNB/gNB/GSM', () => {
    expect(normalizeNetworkTypeFilter('lte')).toBe('eNB');
    expect(normalizeNetworkTypeFilter('nr')).toBe('gNB');
    expect(normalizeNetworkTypeFilter('gsm')).toBe('GSM');
  });

  it('入参已是基站类型码 eNB/gNB/GSM 时原样保留（兼容老链路）', () => {
    expect(normalizeNetworkTypeFilter('eNB')).toBe('eNB');
    expect(normalizeNetworkTypeFilter('gNB')).toBe('gNB');
    expect(normalizeNetworkTypeFilter('GSM')).toBe('GSM');
  });

  it('归一结果与 device.networkType 实际取值一致（lte 命中 eNB 设备）', () => {
    // mock/真实设备的 networkType 字段恒为 eNB/gNB/GSM；归一后能等值命中。
    const deviceNetworkType = 'eNB';
    expect(normalizeNetworkTypeFilter('lte')).toBe(deviceNetworkType);
    expect(normalizeNetworkTypeFilter('lte') === deviceNetworkType).toBe(true);
  });

  it('未知值回退原值（不误吞，便于排查）', () => {
    expect(normalizeNetworkTypeFilter('CPE')).toBe('CPE');
    expect(normalizeNetworkTypeFilter('')).toBe('');
  });
});
