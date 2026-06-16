/**
 * #443 设备类型 → 网络制式码 映射契约。
 *
 * deviceTypeToNetworkTech 是「设备类型唯一来源」的共享落点：指标查询页（v1/v3）用它把外层
 * 设备类型(ENB/GNB/GSM) 映射为设备清单/弹窗的制式码(lte/nr/gsm)。务必与设备类型小写
 * （deviceTypeToTech，给指标库 ?tech= 用）区分，二者不可混用。
 */

import { describe, it, expect } from 'vitest';
import { deviceTypeToNetworkTech, deviceTypeToTech } from './indicatorLibrary';

describe('deviceTypeToNetworkTech（#443 唯一来源映射）', () => {
  it('eNB → lte / gNB → nr / GSM → gsm', () => {
    expect(deviceTypeToNetworkTech('ENB')).toBe('lte');
    expect(deviceTypeToNetworkTech('GNB')).toBe('nr');
    expect(deviceTypeToNetworkTech('GSM')).toBe('gsm');
  });

  it('与 deviceTypeToTech（设备类型小写）刻意不同：ENB→lte≠enb，GNB→nr≠gnb', () => {
    // 失败路径意图：若有人误用 deviceTypeToTech 当制式码传给设备清单，会得到 enb/gnb，
    // 与后端 networkType 期望的 lte/nr 不符；本断言固化两函数语义差异，防回归。
    expect(deviceTypeToNetworkTech('ENB')).not.toBe(deviceTypeToTech('ENB'));
    expect(deviceTypeToNetworkTech('GNB')).not.toBe(deviceTypeToTech('GNB'));
    // GSM 两者恰好同为 'gsm'，不应视为等价语义，仅巧合。
    expect(deviceTypeToNetworkTech('GSM')).toBe('gsm');
    expect(deviceTypeToTech('GSM')).toBe('gsm');
  });
});
