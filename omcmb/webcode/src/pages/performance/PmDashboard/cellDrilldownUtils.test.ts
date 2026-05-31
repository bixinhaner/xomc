import { describe, it, expect } from 'vitest';
import type { MetricObject } from '@core/types/pmObject';
import { getEffectiveLdns, type CellSelection } from './cellDrilldownUtils';

const LDN1 = 'Cellid=111172245,PLMN=46068';
const LDN2 = 'Cellid=111172246,PLMN=46068';
const LDN3 = 'Cellid=111172247,PLMN=46068';

function objs(...ldns: string[]): MetricObject[] {
  return ldns.map((objectLdn) => ({ objectLdn }));
}

describe('getEffectiveLdns — T-0193 下钻白名单汇总', () => {
  it('全部设备缺席（未动过）→ 空数组（不过滤）', () => {
    const value: CellSelection = {};
    const byDevice = { 'SN-A': objs(LDN1, LDN2) };
    expect(getEffectiveLdns(value, byDevice)).toEqual([]);
  });

  it('某设备全选（选中==全集）→ 不贡献过滤项', () => {
    const value: CellSelection = { 'SN-A': [LDN1, LDN2] };
    const byDevice = { 'SN-A': objs(LDN1, LDN2) };
    expect(getEffectiveLdns(value, byDevice)).toEqual([]);
  });

  it('某设备子集 → 只贡献选中项', () => {
    const value: CellSelection = { 'SN-A': [LDN1] };
    const byDevice = { 'SN-A': objs(LDN1, LDN2) };
    expect(getEffectiveLdns(value, byDevice)).toEqual([LDN1]);
  });

  it('多设备混合：一台子集一台全选 → 仅子集设备贡献', () => {
    const value: CellSelection = { 'SN-A': [LDN1], 'SN-B': [LDN3] /* 全选 */ };
    const byDevice = {
      'SN-A': objs(LDN1, LDN2),
      'SN-B': objs(LDN3), // 选中 LDN3 == 全集 → 全选不过滤
    };
    expect(getEffectiveLdns(value, byDevice)).toEqual([LDN1]);
  });

  it('跨设备相同 object_ldn 去重', () => {
    const value: CellSelection = { 'SN-A': [LDN1], 'SN-B': [LDN1] };
    const byDevice = {
      'SN-A': objs(LDN1, LDN2),
      'SN-B': objs(LDN1, LDN3),
    };
    expect(getEffectiveLdns(value, byDevice)).toEqual([LDN1]);
  });

  it('空选（用户取消全部）→ 视为全选不过滤（避免空集导致出图全空）', () => {
    const value: CellSelection = { 'SN-A': [] };
    const byDevice = { 'SN-A': objs(LDN1, LDN2) };
    expect(getEffectiveLdns(value, byDevice)).toEqual([]);
  });
});
