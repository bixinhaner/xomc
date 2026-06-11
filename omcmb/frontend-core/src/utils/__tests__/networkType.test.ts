import { describe, it, expect } from 'vitest';
import { resolveNetworkTypeLabel } from '../networkType';
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
