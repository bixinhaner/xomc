import { describe, expect, it } from 'vitest';

import {
  parseGpvInstances,
  type GpvParameter,
} from '../parseGpvInstances';

describe('parseGpvInstances', () => {
  it('提取 Device.IP.Interface. 下的实例索引', () => {
    const parameters: GpvParameter[] = [
      { name: 'Device.IP.Interface.1.Enable', value: 'true' },
      { name: 'Device.IP.Interface.1.IPv4Address.1.IPAddress', value: '10.0.0.1' },
      { name: 'Device.IP.Interface.3.Enable', value: 'false' },
      { name: 'Device.IP.Interface.2.Enable', value: 'true' },
    ];
    expect(parseGpvInstances(parameters, 'Device.IP.Interface.')).toEqual([1, 2, 3]);
  });

  it('实例号去重 — 同一实例的多个 leaf 仅算一次', () => {
    const parameters: GpvParameter[] = [
      { name: 'Device.Foo.1.Bar', value: 'a' },
      { name: 'Device.Foo.1.Baz', value: 'b' },
      { name: 'Device.Foo.1.Qux', value: 'c' },
    ];
    expect(parseGpvInstances(parameters, 'Device.Foo.')).toEqual([1]);
  });

  it('忽略不匹配 targetObject 的参数', () => {
    const parameters: GpvParameter[] = [
      { name: 'Device.Foo.1.Bar', value: 'a' },
      { name: 'Device.Other.2.Baz', value: 'b' },
      { name: 'Device.Foo.5.Qux', value: 'c' },
    ];
    expect(parseGpvInstances(parameters, 'Device.Foo.')).toEqual([1, 5]);
  });

  it('忽略 targetObject 后非数字段（命名实例不在本期范围）', () => {
    const parameters: GpvParameter[] = [
      { name: 'Device.Foo.1.Bar', value: 'a' },
      { name: 'Device.Foo.WAN.Bar', value: 'b' },
    ];
    expect(parseGpvInstances(parameters, 'Device.Foo.')).toEqual([1]);
  });

  it('targetObject 必须以 "." 结尾，否则返 []', () => {
    const parameters: GpvParameter[] = [
      { name: 'Device.Foo.1.Bar', value: 'a' },
    ];
    expect(parseGpvInstances(parameters, 'Device.Foo')).toEqual([]);
  });

  it('空 parameters → []', () => {
    expect(parseGpvInstances([], 'Device.Foo.')).toEqual([]);
  });

  it('空 targetObject → []', () => {
    expect(parseGpvInstances([{ name: 'Device.Foo.1.Bar' }], '')).toEqual([]);
  });

  it('参数名缺失或非 string → 跳过不抛错', () => {
    const parameters = [
      { name: 'Device.Foo.1.Bar', value: 'a' },
      // @ts-expect-error 测试运行时防御
      { name: null, value: 'b' },
      { name: 'Device.Foo.2.Bar', value: 'c' },
    ] as GpvParameter[];
    expect(parseGpvInstances(parameters, 'Device.Foo.')).toEqual([1, 2]);
  });

  it('targetObject 中的正则元字符 (.) 正确转义，不误匹配', () => {
    // "Device.Foo." 的 "." 如果没转义会匹配任意字符
    // 这里用 "DeviceXFoo.1.Bar" 测试：未转义会误匹配
    const parameters: GpvParameter[] = [
      { name: 'DeviceXFoo.1.Bar', value: 'a' }, // 不应匹配
      { name: 'Device.Foo.2.Bar', value: 'b' }, // 应匹配
    ];
    expect(parseGpvInstances(parameters, 'Device.Foo.')).toEqual([2]);
  });

  it('实例号 0 算合法（TR-069 实例从 1 开始但代码侧不做硬约束）', () => {
    const parameters: GpvParameter[] = [
      { name: 'Device.Foo.0.Bar', value: 'a' },
      { name: 'Device.Foo.1.Bar', value: 'b' },
    ];
    expect(parseGpvInstances(parameters, 'Device.Foo.')).toEqual([0, 1]);
  });

  it('多位数实例号正确解析（如 12, 100）', () => {
    const parameters: GpvParameter[] = [
      { name: 'Device.Foo.12.Bar', value: 'a' },
      { name: 'Device.Foo.100.Bar', value: 'b' },
      { name: 'Device.Foo.3.Bar', value: 'c' },
    ];
    expect(parseGpvInstances(parameters, 'Device.Foo.')).toEqual([3, 12, 100]);
  });
});
