/**
 * KPI-EXPORT 类型映射测试：BackendKpiExportTask → mapBackendKpiExportTask → KpiExportTask。
 * 覆盖完整字段、缺省/空值兜底（成功 + 边界路径）。
 */
import { describe, it, expect } from 'vitest';
import {
  mapBackendKpiExportTask,
  type BackendKpiExportTask,
} from '../kpiExport';

describe('mapBackendKpiExportTask', () => {
  it('完整字段逐项映射（snake_case → camelCase）', () => {
    const b: BackendKpiExportTask = {
      id: 'task-1',
      task_name: 'KPI导出_仪表盘_x',
      source_type: 'dashboard',
      params: { granularity: 'hourly', metric_paths: ['C1'] },
      format: 'csv',
      status: 'succeeded',
      row_count: 24,
      file_size: 4687,
      error: '',
      create_user: 'admin',
      created_at: '2026-06-04T10:00:00Z',
      started_at: '2026-06-04T10:00:02Z',
      finished_at: '2026-06-04T10:00:08Z',
    };
    const t = mapBackendKpiExportTask(b);
    expect(t.id).toBe('task-1');
    expect(t.taskName).toBe('KPI导出_仪表盘_x');
    expect(t.sourceType).toBe('dashboard');
    expect(t.params).toEqual({ granularity: 'hourly', metric_paths: ['C1'] });
    expect(t.status).toBe('succeeded');
    expect(t.rowCount).toBe(24);
    expect(t.fileSize).toBe(4687);
    expect(t.error).toBeUndefined(); // 空串归一为 undefined
    expect(t.createUser).toBe('admin');
    expect(t.startedAt).toBe('2026-06-04T10:00:02Z');
    expect(t.finishedAt).toBe('2026-06-04T10:00:08Z');
  });

  it('缺省/空值兜底：params 缺→{}，row/size 缺→0，未跑完无 started/finished', () => {
    const b = {
      id: 'task-2',
      task_name: 'pending-task',
      source_type: 'adhoc_result',
      format: 'csv',
      status: 'pending',
      create_user: 'op',
      created_at: '2026-06-04T11:00:00Z',
    } as BackendKpiExportTask;
    const t = mapBackendKpiExportTask(b);
    expect(t.sourceType).toBe('adhoc_result');
    expect(t.params).toEqual({});
    expect(t.rowCount).toBe(0);
    expect(t.fileSize).toBe(0);
    expect(t.startedAt).toBeUndefined();
    expect(t.finishedAt).toBeUndefined();
    expect(t.error).toBeUndefined();
  });

  it('失败任务保留 error', () => {
    const b = {
      id: 'task-3',
      task_name: 'failed-task',
      source_type: 'dashboard',
      format: 'csv',
      status: 'failed',
      row_count: 0,
      file_size: 0,
      error: 'aggregator: unknown granularity "bogus"',
      create_user: 'op',
      created_at: '2026-06-04T11:00:00Z',
    } as BackendKpiExportTask;
    const t = mapBackendKpiExportTask(b);
    expect(t.status).toBe('failed');
    expect(t.error).toBe('aggregator: unknown granularity "bogus"');
  });

  it('保留设备性能查看来源', () => {
    const b = {
      id: 'task-4',
      task_name: 'KPI导出_设备性能查看_x',
      source_type: 'device_view',
      format: 'csv',
      status: 'pending',
      row_count: 0,
      file_size: 0,
      create_user: 'op',
      created_at: '2026-06-04T11:00:00Z',
    } as BackendKpiExportTask;
    const t = mapBackendKpiExportTask(b);
    expect(t.sourceType).toBe('device_view');
    expect(t.taskName).not.toContain('仪表盘');
  });
});
