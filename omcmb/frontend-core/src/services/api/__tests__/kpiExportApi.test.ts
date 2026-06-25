/**
 * kpiExportApi 契约测试：
 *   - create 发 source_type/params（snake_case），task_name 仅非空才带。
 *   - listTasks/listFiles 打对端点 + 解析 envelope.items。
 *   - download 以 blob 请求同源端点，并优先使用 Content-Disposition 文件名。
 *   - retry 用原任务 source/params 重新 create（后端无独立重试端点）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

const { getMock, postMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  deleteMock: vi.fn(),
}));
vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, delete: deleteMock },
}));

import { kpiExportApi } from '../kpiExportApi';
import type { KpiExportTask } from '../../../types/kpiExport';

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  deleteMock.mockReset();
});

describe('kpiExportApi.create', () => {
  it('发 source_type + params，task_name 非空才带', async () => {
    postMock.mockResolvedValue({
      data: {
        id: 'x',
        task_name: 'n',
        source_type: 'dashboard',
        format: 'csv',
        status: 'pending',
        create_user: 'admin',
        created_at: '2026-06-04T00:00:00Z',
      },
    });
    await kpiExportApi.create({ sourceType: 'dashboard', params: { granularity: 'hourly' }, taskName: 'n' });
    const [url, payload] = postMock.mock.calls[0];
    expect(url).toBe('/pm/exports');
    expect(payload.source_type).toBe('dashboard');
    expect(payload.params).toEqual({ granularity: 'hourly' });
    expect(payload.task_name).toBe('n');
  });

  it('taskName 缺省时不带 task_name（后端自动生成）', async () => {
    postMock.mockResolvedValue({
      data: { id: 'x', task_name: 'auto', source_type: 'adhoc', format: 'csv', status: 'pending', create_user: 'a', created_at: 'c' },
    });
    await kpiExportApi.create({ sourceType: 'adhoc', params: {} });
    const [, payload] = postMock.mock.calls[0];
    expect('task_name' in payload).toBe(false);
    expect(payload.source_type).toBe('adhoc');
  });
});

describe('kpiExportApi.listTasks / listFiles', () => {
  it('listTasks 打 /pm/exports + 解析 items', async () => {
    getMock.mockResolvedValue({
      data: { items: [{ id: 'a', task_name: 't', source_type: 'dashboard', format: 'csv', status: 'running', create_user: 'u', created_at: 'c' }] },
    });
    const out = await kpiExportApi.listTasks({ status: 'running', limit: 500 });
    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/pm/exports');
    expect(opts.params.status).toBe('running');
    expect(opts.params.limit).toBe(500);
    expect(out).toHaveLength(1);
    expect(out[0].status).toBe('running');
  });

  it('listFiles 打 /pm/exports/files + source_type 过滤', async () => {
    getMock.mockResolvedValue({ data: { items: [] } });
    await kpiExportApi.listFiles({ sourceType: 'adhoc' });
    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/pm/exports/files');
    expect(opts.params.source_type).toBe('adhoc');
  });

  it('items 缺失时返空数组（无崩溃）', async () => {
    getMock.mockResolvedValue({ data: {} });
    const out = await kpiExportApi.listTasks();
    expect(out).toEqual([]);
  });
});

describe('kpiExportApi.download', () => {
  it('按 blob 下载并使用服务端文件名', async () => {
    getMock.mockResolvedValue({
      data: new Blob(['a,b\n1,2\n'], { type: 'text/csv' }),
      headers: { 'content-disposition': 'attachment; filename="server.csv"' },
    });
    const clickSpy = vi.fn();
    const realCreate = document.createElement.bind(document);
    let anchor: HTMLAnchorElement | undefined;
    const createSpy = vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = realCreate(tag) as HTMLAnchorElement;
      if (tag === 'a') {
        anchor = el;
        el.click = clickSpy;
      }
      return el;
    });
    const createObjectURL = vi.fn(() => 'blob:kpi-export');
    const revokeObjectURL = vi.fn();
    Object.defineProperty(window.URL, 'createObjectURL', { configurable: true, value: createObjectURL });
    Object.defineProperty(window.URL, 'revokeObjectURL', { configurable: true, value: revokeObjectURL });

    await kpiExportApi.download('task-1', 'my.csv');

    expect(getMock.mock.calls[0]).toEqual([
      '/pm/exports/task-1/download',
      { responseType: 'blob' },
    ]);
    expect(anchor?.download).toBe('server.csv');
    expect(clickSpy).toHaveBeenCalledOnce();
    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:kpi-export');
    createSpy.mockRestore();
  });

  it('无 Content-Disposition 时使用调用方文件名', async () => {
    getMock.mockResolvedValue({ data: new Blob(['x']), headers: {} });
    const clickSpy = vi.fn();
    const realCreate = document.createElement.bind(document);
    let anchor: HTMLAnchorElement | undefined;
    const createSpy = vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = realCreate(tag) as HTMLAnchorElement;
      if (tag === 'a') {
        anchor = el;
        el.click = clickSpy;
      }
      return el;
    });
    Object.defineProperty(window.URL, 'createObjectURL', { configurable: true, value: vi.fn(() => 'blob:fallback') });
    Object.defineProperty(window.URL, 'revokeObjectURL', { configurable: true, value: vi.fn() });

    await kpiExportApi.download('task-2', 'fallback.csv');

    expect(anchor?.download).toBe('fallback.csv');
    expect(clickSpy).toHaveBeenCalledOnce();
    createSpy.mockRestore();
  });
});

describe('kpiExportApi.retry', () => {
  it('用原任务 source/params 重新 create', async () => {
    postMock.mockResolvedValue({
      data: { id: 'new', task_name: 'n', source_type: 'dashboard', format: 'csv', status: 'pending', create_user: 'a', created_at: 'c' },
    });
    const failed: KpiExportTask = {
      id: 'old',
      taskName: 'n',
      sourceType: 'dashboard',
      params: { granularity: 'hourly' },
      format: 'csv',
      status: 'failed',
      rowCount: 0,
      fileSize: 0,
      createUser: 'a',
      createdAt: 'c',
    };
    const out = await kpiExportApi.retry(failed);
    const [url, payload] = postMock.mock.calls[0];
    expect(url).toBe('/pm/exports');
    expect(payload.source_type).toBe('dashboard');
    expect(payload.params).toEqual({ granularity: 'hourly' });
    expect(out.id).toBe('new');
  });
});
