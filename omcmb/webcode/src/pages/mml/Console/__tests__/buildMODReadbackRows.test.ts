import { describe, expect, it } from 'vitest';
import { buildMODReadbackRows, mapTaskToRecord } from '../adapters';
import type { DeviceTaskResultItem, MMLTask } from '@core/types/mml';

// #196：MOD 自动回读复合（SetParameterValues 下发 + GetParameterValues 回读）→ 每设备一行。
// 用真实 task 3f637b26 的形态：URL 改写 + 回读同 path。

const SN = '1202000240194DP0026';
const URL_PATH = 'Device.FAP.PerfMgmt.Config.1.URL';
const NEW_URL = 'http://172.19.1.143:8080/smallcell/FileUploadService';

const spvEnvelope = {
  method: 'SetParameterValuesResponse',
  raw_response:
    '<cwmp:SetParameterValuesResponse><Status>0</Status></cwmp:SetParameterValuesResponse>',
};
// 注：raw_response 用合法（无未声明命名空间前缀）的 XML，否则 DOMParser 报 parsererror 返回 []。
// 解析器按 local-name 匹配 ParameterValueStruct/Name/Value，无前缀即可。
const gpvEnvelope = (path: string, value: string) => ({
  method: 'GetParameterValuesResponse',
  raw_response: `<GetParameterValuesResponse><ParameterList><ParameterValueStruct><Name>${path}</Name><Value>${value}</Value></ParameterValueStruct></ParameterList></GetParameterValuesResponse>`,
});

const item = (
  over: Partial<DeviceTaskResultItem> & { success: boolean; parsedData?: Record<string, unknown> },
): DeviceTaskResultItem => ({
  deviceSn: SN,
  deviceTaskId: over.deviceTaskId,
  commandIndex: over.commandIndex,
  result: {
    success: over.success,
    rawOutput: (over.result?.rawOutput as string) ?? '',
    parsedData: over.parsedData,
    executionTime: 0,
    timestamp: '',
  },
  startedAt: over.startedAt,
  finishedAt: over.finishedAt,
  failReason: over.failReason,
});

describe('buildMODReadbackRows (#196 MOD 下发 + 回读 LST)', () => {
  it('MOD + 回读 GPV → 一行；pathTasks 含 MOD/LST 两行，操作类型 + 子任务 ID 各自归属', () => {
    const items: DeviceTaskResultItem[] = [
      item({
        success: true,
        commandIndex: 0,
        deviceTaskId: 'mod-task-id',
        result: { success: true, rawOutput: spvEnvelope.raw_response, parsedData: spvEnvelope, executionTime: 0, timestamp: '' },
        parsedData: spvEnvelope,
      }),
      item({
        success: true,
        commandIndex: 1,
        deviceTaskId: 'lst-task-id',
        result: { success: true, rawOutput: 'lst-raw', parsedData: gpvEnvelope(URL_PATH, NEW_URL), executionTime: 0, timestamp: '' },
        parsedData: gpvEnvelope(URL_PATH, NEW_URL),
      }),
    ];
    const rows = buildMODReadbackRows(items, { [URL_PATH]: NEW_URL });
    expect(rows).toHaveLength(1);
    const r = rows[0];
    // 回读值 = 下发值 → success
    expect(r.status).toBe('success');
    // 两个报文都保留：MOD 响应 raw + 回读 LST 响应 readbackRaw（详情页双页签）
    expect(r.raw).toBe(spvEnvelope.raw_response);
    expect(r.readbackRaw).toBe('lst-raw');

    const tasks = r.pathTasks ?? [];
    const mod = tasks.find((t) => t.opType === 'MOD');
    const lst = tasks.find((t) => t.opType === 'LST');
    expect(mod).toBeDefined();
    expect(lst).toBeDefined();
    // MOD 行：path = 下发 path，value = 下发值，子任务 ID = MOD device_task
    expect(mod?.path).toBe(URL_PATH);
    expect(mod?.value).toBe(NEW_URL);
    expect(mod?.subTaskId).toBe('mod-task-id');
    // LST 行：path = 回读响应 path（修复回读行 PATH 为空），子任务 ID = LST device_task
    expect(lst?.path).toBe(URL_PATH);
    expect(lst?.value).toBe(NEW_URL);
    expect(lst?.subTaskId).toBe('lst-task-id');
  });

  it('回读值与下发值不符 → status=mismatch', () => {
    const items: DeviceTaskResultItem[] = [
      item({
        success: true,
        deviceTaskId: 'mod',
        result: { success: true, rawOutput: '', parsedData: spvEnvelope, executionTime: 0, timestamp: '' },
        parsedData: spvEnvelope,
      }),
      item({
        success: true,
        deviceTaskId: 'lst',
        result: { success: true, rawOutput: '', parsedData: gpvEnvelope(URL_PATH, 'http://stale.example/old'), executionTime: 0, timestamp: '' },
        parsedData: gpvEnvelope(URL_PATH, 'http://stale.example/old'),
      }),
    ];
    const rows = buildMODReadbackRows(items, { [URL_PATH]: NEW_URL });
    expect(rows[0].status).toBe('mismatch');
  });

  it('boolean 下发 true 与基站回读 1 视为一致', () => {
    const path = 'Device.DeviceInfo.SignallingTrace.Enable';
    const items: DeviceTaskResultItem[] = [
      item({
        success: true,
        deviceTaskId: 'mod',
        result: { success: true, rawOutput: '', parsedData: spvEnvelope, executionTime: 0, timestamp: '' },
        parsedData: spvEnvelope,
      }),
      item({
        success: true,
        deviceTaskId: 'lst',
        result: { success: true, rawOutput: '', parsedData: gpvEnvelope(path, '1'), executionTime: 0, timestamp: '' },
        parsedData: gpvEnvelope(path, '1'),
      }),
    ];

    const rows = buildMODReadbackRows(items, { [path]: 'true' });
    expect(rows[0].status).toBe('success');
  });

  it('mapTaskToRecord：裸 MOD 下发值在 commands.parameters（非 param_values）→ setValues 正确、回读一致 = success', () => {
    // 复现 task 3f637b26：RAW MOD 的下发值存 parameters（path→value map），param_values 缺失。
    // 修复前 setValues[path]='' → MOD 行值空 + 误判 mismatch；修复后回退 parameters → 正确。
    const task: MMLTask = {
      id: 'task-1',
      taskName: 'MOD URL',
      deviceSns: [SN],
      commands: ['RAW MOD', 'RAW LST'],
      commandsDetail: [
        {
          commandCode: 'RAW MOD',
          operationType: 'MOD',
          paramPaths: [URL_PATH],
          // param_values 缺失（裸 MOD 走 parameters）
          parameters: { [URL_PATH]: NEW_URL },
        },
        {
          commandCode: 'RAW LST',
          operationType: 'LST',
          paramPaths: [URL_PATH],
        },
      ],
      status: 'completed',
      results: [
        item({
          success: true,
          deviceTaskId: 'mod',
          result: { success: true, rawOutput: '', parsedData: spvEnvelope, executionTime: 0, timestamp: '' },
          parsedData: spvEnvelope,
        }),
        item({
          success: true,
          deviceTaskId: 'lst',
          result: { success: true, rawOutput: '', parsedData: gpvEnvelope(URL_PATH, NEW_URL), executionTime: 0, timestamp: '' },
          parsedData: gpvEnvelope(URL_PATH, NEW_URL),
        }),
      ] as unknown as MMLTask['results'],
      createdAt: '',
      updatedAt: '',
      executeType: 'immediate',
    } as MMLTask;

    const rec = mapTaskToRecord(task);
    expect(rec.setValues?.[URL_PATH]).toBe(NEW_URL);
    expect(rec.rows).toHaveLength(1);
    // 回读 = 下发 → success（不再误判 mismatch/未生效）
    expect(rec.rows[0].status).toBe('success');
    const modRow = (rec.rows[0].pathTasks ?? []).find((t) => t.opType === 'MOD');
    expect(modRow?.value).toBe(NEW_URL); // MOD 行值不再为空
  });

  it('mapTaskToRecord：结构化 MOD 的 parameters 使用 param_code key 时按 path 恢复下发值', () => {
    const task: MMLTask = {
      id: 'task-structured-mod',
      taskName: 'MOD URL',
      deviceSns: [SN],
      commands: ['MOD DEVICE_INFO', 'MOD DEVICE_INFO'],
      commandsDetail: [
        {
          commandCode: 'MOD DEVICE_INFO',
          operationType: 'MOD',
          paramPaths: [URL_PATH],
          paramRefs: [{ paramCode: 'URL', tr069Path: URL_PATH }],
          parameters: { URL: NEW_URL },
        },
        { commandCode: 'LST DEVICE_INFO', operationType: 'LST', paramPaths: [URL_PATH] },
      ],
      status: 'completed',
      results: [
        item({ success: true, deviceTaskId: 'mod', parsedData: spvEnvelope }),
        item({ success: true, deviceTaskId: 'lst', parsedData: gpvEnvelope(URL_PATH, NEW_URL) }),
      ] as unknown as MMLTask['results'],
      createdAt: '',
      updatedAt: '',
      executeType: 'immediate',
    } as MMLTask;

    const record = mapTaskToRecord(task);
    expect(record.setValues?.[URL_PATH]).toBe(NEW_URL);
    expect(record.rows[0].status).toBe('success');
    expect(record.rows[0].commandCode).toBe('MOD DEVICE_INFO');
  });

  it('仅 MOD 无回读（GPV 缺失）→ status=unverified，仅 MOD 行', () => {
    const items: DeviceTaskResultItem[] = [
      item({
        success: true,
        deviceTaskId: 'mod',
        result: { success: true, rawOutput: '', parsedData: spvEnvelope, executionTime: 0, timestamp: '' },
        parsedData: spvEnvelope,
      }),
    ];
    const rows = buildMODReadbackRows(items, { [URL_PATH]: NEW_URL });
    expect(rows[0].status).toBe('unverified');
    expect((rows[0].pathTasks ?? []).every((t) => t.opType === 'MOD')).toBe(true);
  });
});
