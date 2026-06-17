import { describe, expect, it } from 'vitest';
import { buildDeviceRows } from '../adapters';
import type { ResultColumn } from '../types';
import type { DeviceTaskResultItem } from '@core/types/mml';

const SN = '1202000240194DP0026';
const SW = 'Device.DeviceInfo.SoftwareVersion';
const HW = 'Device.DeviceInfo.AdditionalHardwareVersion';

const columns: ResultColumn[] = [
  { key: 'c0', label: 'SoftwareVersion', path: SW },
  { key: 'c1', label: 'AdditionalHardwareVersion', path: HW },
];

const gpvEnvelope = (path: string, value: string) => ({
  method: 'GetParameterValuesResponse',
  raw_response: `<cwmp:GetParameterValuesResponse><ParameterList><ParameterValueStruct><Name>${path}</Name><Value>${value}</Value></ParameterValueStruct></ParameterList></cwmp:GetParameterValuesResponse>`,
});

const item = (over: Partial<DeviceTaskResultItem> & { success: boolean }): DeviceTaskResultItem => ({
  deviceSn: SN,
  result: {
    success: over.success,
    rawOutput: '',
    parsedData: over.result?.parsedData,
    executionTime: 0,
    timestamp: '',
  },
  ...over,
});

describe('buildDeviceRows (逐 PATH 合并)', () => {
  // 注：成功 path 的读回值由 parseMmlDeviceTaskResult(GPV) 解析（需 DOM，已在 BUG-3 真机验证）；
  // 本单测聚焦 buildDeviceRows 的「合并」新逻辑：分组 / 失败标记 / 行状态 / pathTasks。
  it('单设备多 path → 合并为一行，失败 path 标「✗ 失败」、行状态 failed、pathTasks 逐 path', () => {
    const items: DeviceTaskResultItem[] = [
      item({
        success: true,
        commandIndex: 0,
        result: { success: true, rawOutput: '', parsedData: gpvEnvelope(SW, 'BaiBLQ_5.0.16.1_1229'), executionTime: 0, timestamp: '' },
      }),
      item({ success: false, commandIndex: 1, failReason: '[Server] Invalid Parameter Names' }),
    ];
    const rows = buildDeviceRows(items, columns, true);
    expect(rows).toHaveLength(1);
    const r = rows[0];
    expect(r.cells[HW]).toBe('✗ 失败');
    expect(r.status).toBe('failed');
    expect(r.faultCode).toBe('部分 path 失败');
    expect(r.pathTasks).toHaveLength(2);
    expect(r.pathTasks?.map((p) => p.status)).toEqual(['success', 'failed']);
    expect(r.pathTasks?.[0].path).toBe(SW);
    expect(r.pathTasks?.[1].path).toBe(HW);
  });

  it('全部 path 成功 → 行状态 success，无失败标记', () => {
    const items: DeviceTaskResultItem[] = [
      item({ success: true, commandIndex: 0, result: { success: true, rawOutput: '', parsedData: gpvEnvelope(SW, 'v1'), executionTime: 0, timestamp: '' } }),
      item({ success: true, commandIndex: 1, result: { success: true, rawOutput: '', parsedData: gpvEnvelope(HW, 'v2'), executionTime: 0, timestamp: '' } }),
    ];
    const rows = buildDeviceRows(items, columns, true);
    expect(rows).toHaveLength(1);
    expect(rows[0].status).toBe('success');
    expect(rows[0].cells[HW]).not.toBe('✗ 失败');
  });

  it('整体下发（每设备 1 条结果）→ 退化为每设备一行', () => {
    const items: DeviceTaskResultItem[] = [
      item({ success: true, deviceSn: 'A', result: { success: true, rawOutput: '', parsedData: gpvEnvelope(SW, 'x'), executionTime: 0, timestamp: '' } }),
      item({ success: true, deviceSn: 'B', result: { success: true, rawOutput: '', parsedData: gpvEnvelope(SW, 'y'), executionTime: 0, timestamp: '' } }),
    ];
    const rows = buildDeviceRows(items, columns, true);
    expect(rows.map((r) => r.deviceSn).sort()).toEqual(['A', 'B']);
  });
});
