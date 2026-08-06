import { describe, expect, it, vi } from 'vitest';
import type { DeviceTaskResultItem } from '@core/types/mml';
import {
  MML_TASK_RESULT_EXPORT_PAGE_SIZE,
  buildMmlTaskResultsCsv,
  fetchAllMmlTaskResults,
} from '../taskResultCsv';
import { taskResultCommandText } from '../taskResultCommand';

const labels: Record<string, string> = {
  'table.index': '序号',
  'mml.resultDeviceCode': '设备SN',
  'mml.deviceName': '设备名称',
  'mml.scriptLineNoColumn': '脚本行号',
  'mml.planOrder': '计划顺序',
  'mml.resultCommand': '执行命令',
  'mml.status': '状态',
  'mml.result': '结果',
  'mml.failReason': '失败原因',
  'mml.taskResult.parsed.title': '解析结果',
  'mml.requestMessage': '请求报文',
  'mml.responseMessage': '响应报文',
  'mml.startTime': '开始时间',
  'mml.endTime': '结束时间',
  'mml.completedStatus': '已完成',
  'status.success': '成功',
  'mml.taskResult.parsed.notParsable': '无法解析',
};

function t(key: string, values?: Record<string, string | number>) {
  if (key === 'mml.scriptLineNo' && values) return `第 ${values.line} 行`;
  return labels[key] ?? key;
}

function row(overrides: Partial<DeviceTaskResultItem> = {}): DeviceTaskResultItem {
  return {
    deviceTaskId: 'device-task-1',
    deviceSn: 'SN001',
    deviceName: '基站A',
    commandCode: 'LST Device.DeviceInfo.SoftwareVersion',
    status: 'completed',
    result: {
      success: true,
      rawOutput: 'OK',
      parsedData: null,
      executionTime: 10,
      timestamp: '2026-07-13T00:00:00Z',
    },
    ...overrides,
  };
}

describe('task result CSV export', () => {
  it('escapes CSV cells, preserves multiline text, and guards Excel formulas', () => {
    const csv = buildMmlTaskResultsCsv([
      row({
        deviceSn: 'SN,001',
        deviceName: '基站 "A"',
        planLineNo: 21,
        planOrder: 2,
        mmlScript: '=HYPERLINK("http://bad")',
        request: { method: 'SetParameterValues', rawRequest: '<xml attr="1">x</xml>' },
        result: {
          success: true,
          rawOutput: 'line1\nline2',
          parsedData: null,
          executionTime: 10,
          timestamp: '2026-07-13T00:00:00Z',
        },
      }),
    ], t);

    expect(csv.split('\r\n')[0]).toBe('序号,设备SN,设备名称,脚本行号,计划顺序,执行命令,状态,结果,失败原因,解析结果,请求报文,响应报文,开始时间,结束时间');
    expect(csv).toContain('"SN,001"');
    expect(csv).toContain('"基站 ""A"""');
    expect(csv).toContain('"\'=HYPERLINK(""http://bad"")"');
    expect(csv).toContain('"<xml attr=""1"">x</xml>"');
    expect(csv).toContain('"line1\nline2"');
  });

  it('preserves backend timezone wall-clock values in exported timestamps', () => {
    const csv = buildMmlTaskResultsCsv([
      row({
        startedAt: '2026-07-13T10:00:00+09:00',
        finishedAt: '2026-07-13T11:30:00+09:00',
      }),
    ], t);

    expect(csv).toContain('2026-07-13 10:00:00');
    expect(csv).toContain('2026-07-13 11:30:00');
  });

  it('shows the generated SetParameterValues phase of ADD-with-params as MOD', () => {
    const command = taskResultCommandText(row({
      mmlScript: 'ADD Device.FAP.Ipsec.:TUNNEL_ENABLE=false,TUNNEL_GATEWAY=192.0.2.2;SN001',
      planRawLine: 'ADD Device.FAP.Ipsec.:TUNNEL_ENABLE=false,TUNNEL_GATEWAY=192.0.2.2;SN001',
      operationType: 'MOD',
      request: {
        method: 'SetParameterValues',
        payload: {
          values: [
            { name: 'Device.FAP.Ipsec.2.TUNNEL_ENABLE', value: 'false', type: 'xsd:string' },
            { name: 'Device.FAP.Ipsec.2.TUNNEL_GATEWAY', value: '192.0.2.2', type: 'xsd:string' },
          ],
        },
      },
    }));

    expect(command).toBe('MOD Device.FAP.Ipsec.2.:TUNNEL_ENABLE=false,TUNNEL_GATEWAY=192.0.2.2');
  });

  it('shows MOD when the stored request payload uses a parameter map', () => {
    const command = taskResultCommandText(row({
      commandCode: 'MOD DEVICE_INFO',
      operationType: 'MOD',
      request: {
        method: 'SetParameterValues',
        payload: { values: { 'Device.DeviceInfo.SoftwareVersion': '1.2.3' } },
      },
    }));

    expect(command).toBe('MOD Device.DeviceInfo.:SoftwareVersion=1.2.3');
  });

  it('shows MOD when the stored request payload uses the legacy parameters field', () => {
    const command = taskResultCommandText(row({
      commandCode: 'MOD DEVICE_INFO',
      operationType: 'MOD',
      request: {
        method: 'SetParameterValues',
        payload: { parameters: { 'Device.DeviceInfo.SoftwareVersion': '1.2.3' } },
      },
    }));

    expect(command).toBe('MOD Device.DeviceInfo.:SoftwareVersion=1.2.3');
  });

  it('fetches every result page with the backend-safe page size', async () => {
    const fetchPage = vi.fn()
      .mockResolvedValueOnce({
        total: 201,
        page: 1,
        pageSize: MML_TASK_RESULT_EXPORT_PAGE_SIZE,
        items: Array.from({ length: 100 }, (_, index) => row({ deviceTaskId: `p1-${index}` })),
      })
      .mockResolvedValueOnce({
        total: 201,
        page: 2,
        pageSize: MML_TASK_RESULT_EXPORT_PAGE_SIZE,
        items: Array.from({ length: 100 }, (_, index) => row({ deviceTaskId: `p2-${index}` })),
      })
      .mockResolvedValueOnce({
        total: 201,
        page: 3,
        pageSize: MML_TASK_RESULT_EXPORT_PAGE_SIZE,
        items: [row({ deviceTaskId: 'p3-0' })],
      });

    const rows = await fetchAllMmlTaskResults('task-1', fetchPage);

    expect(rows).toHaveLength(201);
    expect(fetchPage).toHaveBeenNthCalledWith(1, 'task-1', 1, MML_TASK_RESULT_EXPORT_PAGE_SIZE);
    expect(fetchPage).toHaveBeenNthCalledWith(2, 'task-1', 2, MML_TASK_RESULT_EXPORT_PAGE_SIZE);
    expect(fetchPage).toHaveBeenNthCalledWith(3, 'task-1', 3, MML_TASK_RESULT_EXPORT_PAGE_SIZE);
  });
});
