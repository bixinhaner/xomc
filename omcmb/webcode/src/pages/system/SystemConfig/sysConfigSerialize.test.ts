import { describe, it, expect } from 'vitest';
import { encodeValue, buildBatchItems } from './sysConfigSerialize';
import type { SysConfigItem } from '@core/types/system';

describe('encodeValue', () => {
  it('整数 → int（离线阈值 enbTimeout/cpeTimeout 的契约）', () => {
    expect(encodeValue(100)).toEqual({ value: '100', valueType: 'int' });
    expect(encodeValue(600)).toEqual({ value: '600', valueType: 'int' });
  });

  it('小数 → float', () => {
    expect(encodeValue(1.5)).toEqual({ value: '1.5', valueType: 'float' });
  });

  it('布尔 → bool，对象 → json，字符串 → string，空 → string', () => {
    expect(encodeValue(true)).toEqual({ value: 'true', valueType: 'bool' });
    expect(encodeValue({ a: 1 })).toEqual({ value: '{"a":1}', valueType: 'json' });
    expect(encodeValue('x')).toEqual({ value: 'x', valueType: 'string' });
    expect(encodeValue(null)).toEqual({ value: '', valueType: 'string' });
    expect(encodeValue(undefined)).toEqual({ value: '', valueType: 'string' });
  });
});

describe('buildBatchItems (#203 设备离线阈值读写契约)', () => {
  it('设备页整数字段序列化为 value_type=int', () => {
    const items = buildBatchItems({ enbTimeout: 100, cpeTimeout: 600 }, []);
    const enb = items.find((i) => i.key === 'enbTimeout');
    const cpe = items.find((i) => i.key === 'cpeTimeout');
    expect(enb).toEqual({ key: 'enbTimeout', value: '100', value_type: 'int' });
    expect(cpe).toEqual({ key: 'cpeTimeout', value: '600', value_type: 'int' });
  });

  it('已存在 key 沿用 DB 中的 value_type（不被运行期推断覆盖）', () => {
    const existing: SysConfigItem[] = [
      { key: 'enbTimeout', value: '100', valueType: 'int' } as SysConfigItem,
    ];
    // 表单把数字渲染成字符串时也不应把已知 int 字段降级成 string
    const items = buildBatchItems({ enbTimeout: '120' }, existing);
    expect(items[0]).toEqual({ key: 'enbTimeout', value: '120', value_type: 'int' });
  });

  it('新增 key 用运行期推断的 value_type', () => {
    const items = buildBatchItems({ newFlag: true }, []);
    expect(items[0]).toEqual({ key: 'newFlag', value: 'true', value_type: 'bool' });
  });
});
