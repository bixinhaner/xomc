import { describe, it, expect } from 'vitest';
import {
  adhocIncludesCell,
  adhocObjectHeaderKey,
  adhocObjectName,
  objectKeyOf,
} from './adhocObjectColumn';
import type { AdhocResultRow, AdhocDimension } from '@core/types/pmAdhoc';

const base: AdhocResultRow = {
  id: '',
  taskId: '',
  deviceOui: '',
  deviceSn: '',
  productId: '',
  productName: '',
  deviceGroupName: '',
  objectLdn: '',
  metricPath: 'C1',
  metricType: 'counter',
  metricValue: 1,
  granularity: 'hourly',
  time: '',
  startTime: '',
  endTime: '',
  displayName: 'C1',
};

// fakeIntl 按 id 返回对应文案，模拟真实 i18n 契约（network=全网，聚合组=聚合组·N 台）。
const fakeIntl = {
  formatMessage: (d: { id: string }, v?: { count?: number }) => {
    if (d.id === 'perf.adhoc.colObject.network') return '全网';
    if (d.id === 'perf.adhoc.aggregateGroupUnit') return `聚合组·${v?.count ?? 0} 台`;
    return d.id;
  },
} as never;

const NON_DEVICE: AdhocDimension[] = ['device_group', 'product', 'band', 'network', 'aggregate_group'];

describe('adhocIncludesCell', () => {
  it('仅 device 含小区列', () => {
    expect(adhocIncludesCell('device')).toBe(true);
    for (const d of NON_DEVICE) {
      expect(adhocIncludesCell(d)).toBe(false);
    }
  });
});

describe('adhocObjectHeaderKey', () => {
  it('各维度映射到对应首列表头 key', () => {
    expect(adhocObjectHeaderKey('device')).toBe('perf.adhoc.colDevice');
    expect(adhocObjectHeaderKey('device_group')).toBe('perf.adhoc.colObject.deviceGroup');
    expect(adhocObjectHeaderKey('product')).toBe('perf.adhoc.colObject.product');
    expect(adhocObjectHeaderKey('band')).toBe('perf.adhoc.colObject.band');
    expect(adhocObjectHeaderKey('network')).toBe('perf.adhoc.colObject.network');
    expect(adhocObjectHeaderKey('aggregate_group')).toBe('perf.adhoc.colObject.aggregateGroup');
  });
});

describe('adhocObjectName', () => {
  it('device → OUI/SN，无 OUI 时仅 SN', () => {
    expect(adhocObjectName({ ...base, deviceOui: 'O1', deviceSn: 'S1' }, 'device', [], fakeIntl)).toBe('O1/S1');
    expect(adhocObjectName({ ...base, deviceOui: '', deviceSn: 'S1' }, 'device', [], fakeIntl)).toBe('S1');
  });
  it('device_group → 组名优先，缺失回退 uuid 前 8（剥前缀）', () => {
    expect(
      adhocObjectName({ ...base, deviceGroupName: '华东A组', objectLdn: 'DeviceGroup=abcdef1234' }, 'device_group', [], fakeIntl),
    ).toBe('华东A组');
    expect(
      adhocObjectName({ ...base, deviceGroupName: '', objectLdn: 'DeviceGroup=abcdef1234' }, 'device_group', [], fakeIntl),
    ).toBe('abcdef12');
  });
  it('product → 产品名优先，缺失回退 id 前 8', () => {
    expect(
      adhocObjectName({ ...base, productName: 'NR-Pico', productId: '11112222-3333' }, 'product', [], fakeIntl),
    ).toBe('NR-Pico');
    expect(
      adhocObjectName({ ...base, productName: '', productId: '11112222-3333' }, 'product', [], fakeIntl),
    ).toBe('11112222');
  });
  it('band → 剥 Band= 前缀', () => {
    expect(adhocObjectName({ ...base, objectLdn: 'Band=42' }, 'band', [], fakeIntl)).toBe('42');
  });
  it('network → 全网', () => {
    expect(adhocObjectName(base, 'network', [], fakeIntl)).toBe('全网');
  });
  it('aggregate_group → 聚合组·N 台', () => {
    expect(adhocObjectName(base, 'aggregate_group', ['a', 'b', 'c'], fakeIntl)).toBe('聚合组·3 台');
  });
});

describe('objectKeyOf', () => {
  it('product 用 productId 区分（避免不同产品行键碰撞）', () => {
    const a = objectKeyOf({ ...base, productId: 'P1' }, 'product');
    const b = objectKeyOf({ ...base, productId: 'P2' }, 'product');
    expect(a).not.toBe(b);
  });
  it('device_group/band/aggregate_group 用 objectLdn 区分', () => {
    expect(objectKeyOf({ ...base, objectLdn: 'DeviceGroup=g1' }, 'device_group')).toBe('DeviceGroup=g1');
    expect(objectKeyOf({ ...base, objectLdn: 'Band=42' }, 'band')).toBe('Band=42');
    expect(objectKeyOf({ ...base, objectLdn: 'X=1' }, 'aggregate_group')).toBe('X=1');
  });
  it('network 恒定单键（全网汇成一组）', () => {
    expect(objectKeyOf({ ...base, deviceSn: 'S1' }, 'network')).toBe(
      objectKeyOf({ ...base, deviceSn: 'S2' }, 'network'),
    );
  });
  it('device 用 deviceSn', () => {
    expect(objectKeyOf({ ...base, deviceSn: 'S1' }, 'device')).toBe('S1');
  });
});
