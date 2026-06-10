/**
 * deviceTaskApi 契约测试（#22 关键 API — task 域）：
 *   - getTask：打 /devices/tasks/:id，BackendDeviceTask → DeviceTask 映射（snake→camel，
 *     retry/max 缺省兜 0，可选 sent/completed/error 字段透传）。
 *   - 404/500 错误原样抛（useDeviceTaskStatus retry:false 依赖错误冒泡停止轮询）。
 *
 * 用 vi.mock 替换 http 客户端（参照 alarmApi.test.ts 既有模式）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }));
vi.mock('../../http', () => ({
  default: { get: getMock, post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

import { deviceTaskApi } from '../deviceTaskApi';

beforeEach(() => {
  getMock.mockReset();
});

describe('deviceTaskApi.getTask', () => {
  it('打 /devices/tasks/:id 且完整字段映射（snake → camel）', async () => {
    getMock.mockResolvedValue({
      data: {
        id: 't1',
        device_sn: 'SN001',
        method: 'SetParameterValues',
        status: 'completed',
        retry_count: 1,
        max_retries: 3,
        created_at: '2026-06-10T00:00:00Z',
        sent_at: '2026-06-10T00:00:01Z',
        completed_at: '2026-06-10T00:00:05Z',
        error_code: 0,
        error_message: '',
      },
    });
    const t = await deviceTaskApi.getTask('t1');
    expect(getMock.mock.calls[0][0]).toBe('/devices/tasks/t1');
    expect(t.id).toBe('t1');
    expect(t.deviceSn).toBe('SN001');
    expect(t.method).toBe('SetParameterValues');
    expect(t.status).toBe('completed');
    expect(t.retryCount).toBe(1);
    expect(t.maxRetries).toBe(3);
    expect(t.sentAt).toBe('2026-06-10T00:00:01Z');
    expect(t.completedAt).toBe('2026-06-10T00:00:05Z');
  });

  it('retry_count / max_retries 缺省兜 0；可选字段缺省为 undefined', async () => {
    getMock.mockResolvedValue({
      data: {
        id: 't2',
        device_sn: 'SN002',
        method: 'AddObject',
        status: 'pending',
        created_at: '2026-06-10T00:00:00Z',
      },
    });
    const t = await deviceTaskApi.getTask('t2');
    expect(t.retryCount).toBe(0);
    expect(t.maxRetries).toBe(0);
    expect(t.sentAt).toBeUndefined();
    expect(t.completedAt).toBeUndefined();
    expect(t.errorMessage).toBeUndefined();
  });

  it('失败任务保留 error_code / error_message', async () => {
    getMock.mockResolvedValue({
      data: {
        id: 't3',
        device_sn: 'SN003',
        method: 'SetParameterValues',
        status: 'failed',
        retry_count: 3,
        max_retries: 3,
        created_at: '2026-06-10T00:00:00Z',
        error_code: 9001,
        error_message: 'CPE rejected: invalid value',
      },
    });
    const t = await deviceTaskApi.getTask('t3');
    expect(t.status).toBe('failed');
    expect(t.errorCode).toBe(9001);
    expect(t.errorMessage).toBe('CPE rejected: invalid value');
  });

  it('404 错误原样抛（轮询 Hook 依赖错误冒泡停止）', async () => {
    getMock.mockRejectedValue({ response: { status: 404 } });
    await expect(deviceTaskApi.getTask('missing')).rejects.toEqual({ response: { status: 404 } });
  });

  it('500 错误原样抛', async () => {
    getMock.mockRejectedValue({ response: { status: 500 } });
    await expect(deviceTaskApi.getTask('t1')).rejects.toEqual({ response: { status: 500 } });
  });
});
