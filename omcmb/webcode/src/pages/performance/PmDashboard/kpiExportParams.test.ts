import { describe, expect, it } from 'vitest';
import {
  buildDashboardExportParams,
  buildAdhocExportParams,
  validateDashboardExportSelection,
  defaultExportTaskName,
  type DashboardExportSelection,
} from './kpiExportParams';

const fullSel: DashboardExportSelection = {
  technology: 'lte',
  deviceSns: ['SN1', 'SN2'],
  metricPaths: ['C000170043'],
  granularity: 'hourly',
  startTime: '2026-06-01T00:00:00.000Z',
  endTime: '2026-06-05T00:00:00.000Z',
};

describe('buildDashboardExportParams', () => {
  it('带上设备/指标/时间/粒度，维度固定 device、类型固定 counter、制式装成数组', () => {
    const p = buildDashboardExportParams(fullSel);
    expect(p).toMatchObject({
      granularity: 'hourly',
      dimension: 'device',
      device_sns: ['SN1', 'SN2'],
      metric_paths: ['C000170043'],
      metric_type: 'counter',
      technologies: ['lte'],
      start_time: '2026-06-01T00:00:00.000Z',
      end_time: '2026-06-05T00:00:00.000Z',
    });
    // 死判 T4-params-carried：设备/指标/时间/粒度均非空。
    expect((p.device_sns as string[]).length).toBeGreaterThan(0);
    expect((p.metric_paths as string[]).length).toBeGreaterThan(0);
    expect(p.granularity).toBeTruthy();
    expect(p.start_time).toBeTruthy();
    expect(p.end_time).toBeTruthy();
  });

  it('小区/PLMN 白名单本期 out-of-scope：永不发送 object_ldns 键（后端无对应过滤）', () => {
    expect(buildDashboardExportParams(fullSel)).not.toHaveProperty('object_ldns');
  });

  it('制式为空时 technologies 为空数组', () => {
    const p = buildDashboardExportParams({ ...fullSel, technology: '' });
    expect(p.technologies).toEqual([]);
  });
});

describe('buildAdhocExportParams', () => {
  it('只带 task_id 时不写时窗', () => {
    expect(buildAdhocExportParams({ taskId: 'task-1' })).toEqual({ task_id: 'task-1' });
  });

  it('带二次时窗时写入 start_time / end_time', () => {
    expect(
      buildAdhocExportParams({
        taskId: 'task-1',
        startTime: '2026-06-01T00:00:00.000Z',
        endTime: '2026-06-05T00:00:00.000Z',
      }),
    ).toEqual({
      task_id: 'task-1',
      start_time: '2026-06-01T00:00:00.000Z',
      end_time: '2026-06-05T00:00:00.000Z',
    });
  });
});

describe('validateDashboardExportSelection', () => {
  it('齐备时返回 null', () => {
    expect(validateDashboardExportSelection(fullSel)).toBeNull();
  });

  it('缺设备/指标/粒度/时间各返回对应语料键', () => {
    expect(validateDashboardExportSelection({ ...fullSel, deviceSns: [] })).toBe(
      'perf.dashboard.selectAtLeastOneDevice',
    );
    expect(validateDashboardExportSelection({ ...fullSel, metricPaths: [] })).toBe(
      'perf.dashboard.selectAtLeastOneMetric',
    );
    expect(validateDashboardExportSelection({ ...fullSel, granularity: '' })).toBe(
      'kpiExport.export.missingGranularity',
    );
    expect(validateDashboardExportSelection({ ...fullSel, endTime: '' })).toBe(
      'kpiExport.export.missingTimeRange',
    );
  });
});

describe('defaultExportTaskName', () => {
  it('按来源 + 时间戳生成可读名', () => {
    const d = new Date(2026, 5, 4, 21, 23, 8); // 2026-06-04 21:23:08 本地
    expect(defaultExportTaskName('dashboard', d)).toBe('KPI导出_仪表盘_20260604_212308');
    expect(defaultExportTaskName('adhoc', d)).toBe('KPI导出_任务结果_20260604_212308');
  });
});
