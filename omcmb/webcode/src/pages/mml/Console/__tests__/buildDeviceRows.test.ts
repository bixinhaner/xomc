import { describe, expect, it } from 'vitest';
import {
  buildDeviceRows,
  buildRawExecutePayload,
  hasPlanRows,
  mapResultItemToRow,
  PARTIAL_PATH_FAILED_FALLBACK,
  PATH_FAILED_CELL,
  rawCommandName,
} from '../adapters';
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
  it('保留后端返回的自定义命令名，详情不回退到内部命令码', () => {
    const result = mapResultItemToRow(
      item({ success: true, commandCode: 'RAW LST', commandName: 'dxpTest' }),
      columns,
      true,
    );

    expect(result.commandName).toBe('dxpTest');
  });

  it('只有实际存在计划行时才显示 Plan Row 列', () => {
    expect(hasPlanRows([{ planLineNo: undefined }])).toBe(false);
    expect(hasPlanRows([{ planLineNo: 7 }])).toBe(true);
  });

  it('裸路径执行请求携带用户选择的命令名', () => {
    const payload = buildRawExecutePayload('LST', [{ path: SW, value: '' }], [SN], 'dxpTest', 'whole', 'dxpTest');
    expect(payload.command_name).toBe('dxpTest');
  });

  it('裸路径命名无语言上下文时使用操作码兜底，不硬编码中文操作词', () => {
    expect(rawCommandName('LST', [SW])).toBe('LST SoftwareVersion');
    expect(rawCommandName('LST', [SW], undefined, 'Query')).toBe('Query SoftwareVersion');
    expect(rawCommandName('LST', [SW, HW], undefined, 'Query', ' and 2 items')).toBe('Query SoftwareVersion and 2 items');
    expect(rawCommandName('LST', [SW, HW])).not.toMatch(/[\u4e00-\u9fff]/);
  });

  // 注：成功 path 的读回值由 parseMmlDeviceTaskResult(GPV) 解析（需 DOM，已在 BUG-3 真机验证）；
  // 本单测聚焦 buildDeviceRows 的「合并」新逻辑：分组 / 失败标记 / 行状态 / pathTasks。
  it('单设备多 path → 合并为一行，失败 path 标内部 sentinel、行状态 failed、pathTasks 逐 path，并展示真实失败原因', () => {
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
    expect(r.cells[HW]).toBe(PATH_FAILED_CELL);
    expect(r.status).toBe('failed');
    expect(r.faultCode).toBe('[Server] Invalid Parameter Names');
    expect(r.pathTasks).toHaveLength(2);
    expect(r.pathTasks?.map((p) => p.status)).toEqual(['success', 'failed']);
    expect(r.pathTasks?.[0].path).toBe(SW);
    expect(r.pathTasks?.[1].path).toBe(HW);
  });

  it('单设备多 path 的失败摘要优先显示设备返回的参数级错误', () => {
    const faultPath = 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.NeighborList.NRCell.2.CID';
    const rows = buildDeviceRows([
      item({ success: true, commandIndex: 0 }),
      item({
        success: false,
        commandIndex: 1,
        failReason: '[Client] Invalid arguments',
        result: {
          success: false,
          rawOutput: '',
          parsedData: {
            method: 'SetParameterValues',
            param_faults: [{
              parameter_name: faultPath,
              fault_code: 9007,
              fault_string: 'NR_NEIGH_CELL_IDENTITY:NotValidValue: Neigh cell info already exists, do not add it again.',
            }],
          },
          executionTime: 0,
          timestamp: '',
        },
      }),
    ], columns, false);

    expect(rows[0].status).toBe('failed');
    expect(rows[0].faultCode).toBe(
      `${faultPath}: 9007 NR_NEIGH_CELL_IDENTITY:NotValidValue: Neigh cell info already exists, do not add it again.`,
    );
  });

  it('全部 path 成功 → 行状态 success，无失败标记', () => {
    const items: DeviceTaskResultItem[] = [
      item({ success: true, commandIndex: 0, result: { success: true, rawOutput: '', parsedData: gpvEnvelope(SW, 'v1'), executionTime: 0, timestamp: '' } }),
      item({ success: true, commandIndex: 1, result: { success: true, rawOutput: '', parsedData: gpvEnvelope(HW, 'v2'), executionTime: 0, timestamp: '' } }),
    ];
    const rows = buildDeviceRows(items, columns, true);
    expect(rows).toHaveLength(1);
    expect(rows[0].status).toBe('success');
    expect(rows[0].cells[HW]).not.toBe(PATH_FAILED_CELL);
  });

  it('整体下发（每设备 1 条结果）→ 退化为每设备一行', () => {
    const items: DeviceTaskResultItem[] = [
      item({ success: true, deviceSn: 'A', result: { success: true, rawOutput: '', parsedData: gpvEnvelope(SW, 'x'), executionTime: 0, timestamp: '' } }),
      item({ success: true, deviceSn: 'B', result: { success: true, rawOutput: '', parsedData: gpvEnvelope(SW, 'y'), executionTime: 0, timestamp: '' } }),
    ];
    const rows = buildDeviceRows(items, columns, true);
    expect(rows.map((r) => r.deviceSn).sort()).toEqual(['A', 'B']);
  });

  it('ADD 成功但响应不带参数值时，用已下发值回填结果列', () => {
    const submitted = { [SW]: '12', [HW]: '123' };
    const rows = buildDeviceRows([item({ success: true })], columns, false, submitted);

    expect(rows[0].status).toBe('success');
    expect(rows[0].cells).toMatchObject(submitted);
  });

  it('ADD 失败时不把已下发值伪装成成功结果', () => {
    const rows = buildDeviceRows(
      [item({ success: false, failReason: 'SPV failed' })],
      columns,
      false,
      { [SW]: '12' },
    );

    expect(rows[0].status).toBe('failed');
    expect(rows[0].cells[SW]).not.toBe('12');
  });
});
