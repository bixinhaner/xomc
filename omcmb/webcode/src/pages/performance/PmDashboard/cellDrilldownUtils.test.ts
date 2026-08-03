import { describe, it, expect } from 'vitest';
import type { MetricObject } from '@core/types/pmObject';
import {
  getEffectiveLdns,
  getEffectiveLdnsWithNrRecommendedDefault,
  getNrRecommendedDefaultSelectedObjectLdns,
  isRecommendedNrObjectLdn,
  type CellSelection,
} from './cellDrilldownUtils';

const LDN1 = 'Cellid=111172245,PLMN=46068';
const LDN2 = 'Cellid=111172246,PLMN=46068';
const LDN3 = 'Cellid=111172247,PLMN=46068';
const NR_GNB = 'Type=gNB,Mode=SA,gNBID=111';
const NR_GNB_TYPE_ONLY = 'Type=gNB';
const NR_GNB_CHANGED = 'Type=gNB,Mode=NSA,gNBID=222';
const NR_CELL_PLMN = 'Type=Cell,Mode=SA,gNBID=111,NrCGI=10,CUID=1,PLMNID=00101';
const NR_CELL_PLMN_CHANGED = 'Type=Cell,Mode=NSA,gNBID=222,NrCGI=20,DUID=2,PLMNID=46001';
const NR_CELL_PLMN_VALUE_WITH_COMMA = 'Type=Cell,Mode=SA,gNBID=111,NrCGI=10,PLMNID=460,01';
const NR_CELL_NO_PLMN = 'Type=Cell,Mode=SA,gNBID=111,NrCGI=10,CUID=1';
const NR_CELL_PLMN_NOT_ID = 'Type=Cell,Mode=SA,gNBID=111,NrCGI=10,PLMN=00101';
const NR_SLICE = 'Type=Cell,Mode=SA,gNBID=111,NrCGI=10,CUID=1,NSSAI=0/1/2';
const GSM_LDN = 'Uid=4002-1';

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

describe('getEffectiveLdns — #241 5G 默认推荐 object_ldn', () => {
  it('只按字段名和 Type 类别匹配，不绑定 gNBID/Mode/NrCGI/PLMNID 的具体值', () => {
    expect(isRecommendedNrObjectLdn(NR_GNB)).toBe(true);
    expect(isRecommendedNrObjectLdn(NR_GNB_TYPE_ONLY)).toBe(true);
    expect(isRecommendedNrObjectLdn(NR_GNB_CHANGED)).toBe(true);
    expect(isRecommendedNrObjectLdn(NR_CELL_PLMN)).toBe(true);
    expect(isRecommendedNrObjectLdn(NR_CELL_PLMN_CHANGED)).toBe(true);
  });

  it('5G 非设备级、非 Cell+PLMN 级 job 默认不勾选', () => {
    expect(isRecommendedNrObjectLdn(NR_CELL_NO_PLMN)).toBe(false);
    expect(isRecommendedNrObjectLdn(NR_SLICE)).toBe(false);
  });

  it('LTE/GSM 保持旧行为：默认全选不过滤', () => {
    const byDevice = {
      'LTE-SN': objs(LDN1, LDN2),
      'GSM-SN': objs(GSM_LDN),
    };
    expect(getNrRecommendedDefaultSelectedObjectLdns(byDevice['LTE-SN'])).toEqual([LDN1, LDN2]);
    expect(getNrRecommendedDefaultSelectedObjectLdns(byDevice['GSM-SN'])).toEqual([GSM_LDN]);
    expect(getEffectiveLdns({}, byDevice)).toEqual([]);
  });

  it('基础 getEffectiveLdns 保持旧语义：5G 未手动选择也不过滤', () => {
    const byDevice = {
      'NR-SN': objs(NR_GNB, NR_CELL_PLMN, NR_CELL_NO_PLMN, NR_SLICE),
    };
    expect(getEffectiveLdns({}, byDevice)).toEqual([]);
  });

  it('未手动选择时，5G 默认白名单只包含 gNB 和 Cell+PLMN', () => {
    const byDevice = {
      'NR-SN': objs(NR_GNB, NR_CELL_PLMN, NR_CELL_NO_PLMN, NR_SLICE),
    };
    expect(getNrRecommendedDefaultSelectedObjectLdns(byDevice['NR-SN'])).toEqual([NR_GNB, NR_CELL_PLMN]);
    expect(getEffectiveLdnsWithNrRecommendedDefault({}, byDevice)).toEqual([NR_GNB, NR_CELL_PLMN]);
  });

  it('PLMN 不会误判成 PLMNID，PLMNID 字段值含逗号也能按字段名识别', () => {
    expect(isRecommendedNrObjectLdn(NR_CELL_PLMN_NOT_ID)).toBe(false);
    expect(isRecommendedNrObjectLdn(NR_CELL_PLMN_VALUE_WITH_COMMA)).toBe(true);
  });

  it('用户手动全选 5G 对象后仍表示不过滤，可查全部 job', () => {
    const byDevice = {
      'NR-SN': objs(NR_GNB, NR_CELL_PLMN, NR_CELL_NO_PLMN),
    };
    const value: CellSelection = {
      'NR-SN': [NR_GNB, NR_CELL_PLMN, NR_CELL_NO_PLMN],
    };
    expect(getEffectiveLdnsWithNrRecommendedDefault(value, byDevice)).toEqual([]);
  });
});
