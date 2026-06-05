import { describe, expect, it } from 'vitest';
import {
  buildDashboardExportParams,
  buildAdhocExportParams,
  validateDashboardExportSelection,
  defaultExportTaskName,
  kpiQueryToDashboardSelection,
  type DashboardExportSelection,
} from '../kpiExportParams';
import type { QueryTemplatePayload } from '../../types/pmQuery';

const fullSel: DashboardExportSelection = {
  technology: 'lte',
  deviceSns: ['SN1', 'SN2'],
  metricPaths: ['C000170043'],
  granularity: 'hourly',
  startTime: '2026-06-01T00:00:00.000Z',
  endTime: '2026-06-05T00:00:00.000Z',
};

describe('buildDashboardExportParams', () => {
  it('带上设备/指标/时间/粒度，维度固定 device、制式装成数组', () => {
    const p = buildDashboardExportParams(fullSel);
    expect(p).toMatchObject({
      granularity: 'hourly',
      dimension: 'device',
      device_sns: ['SN1', 'SN2'],
      metric_paths: ['C000170043'],
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

  it('A3：不发送 metric_type（与出图同口径，counter/kpi 都导）', () => {
    expect(buildDashboardExportParams(fullSel)).not.toHaveProperty('metric_type');
  });

  it('A1：未下钻（objectLdns 缺席/空）时不发送 object_ldns 键', () => {
    expect(buildDashboardExportParams(fullSel)).not.toHaveProperty('object_ldns');
    expect(buildDashboardExportParams({ ...fullSel, objectLdns: [] })).not.toHaveProperty(
      'object_ldns',
    );
  });

  it('A1：下钻定格小区/PLMN 时回填 object_ldns 白名单', () => {
    const p = buildDashboardExportParams({
      ...fullSel,
      objectLdns: ['Cellid=1,PLMN=00101', 'Cellid=1,PLMN=46068'],
    });
    expect(p.object_ldns).toEqual(['Cellid=1,PLMN=00101', 'Cellid=1,PLMN=46068']);
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

describe('kpiQueryToDashboardSelection', () => {
  const basePayload: QueryTemplatePayload = {
    deviceSns: ['SN1', 'SN2'],
    metricPaths: ['K900010043', 'C000170043'],
    granularity: 'daily',
    timeRangePreset: 'custom',
    deviceType: 'ENB',
  };
  const range = { start: '2026-06-01T00:00:00.000Z', end: '2026-06-05T00:00:00.000Z' };

  it('设备类型映射制式：ENB→lte / GNB→nr / GSM→gsm，缺省回退 lte', () => {
    expect(kpiQueryToDashboardSelection(basePayload, range).technology).toBe('lte');
    expect(
      kpiQueryToDashboardSelection({ ...basePayload, deviceType: 'GNB' }, range).technology,
    ).toBe('nr');
    expect(
      kpiQueryToDashboardSelection({ ...basePayload, deviceType: 'GSM' }, range).technology,
    ).toBe('gsm');
    expect(
      kpiQueryToDashboardSelection({ ...basePayload, deviceType: undefined }, range).technology,
    ).toBe('lte');
  });

  it('设备/指标/粒度/起止时间正确透传', () => {
    const sel = kpiQueryToDashboardSelection(basePayload, range);
    expect(sel).toMatchObject({
      deviceSns: ['SN1', 'SN2'],
      metricPaths: ['K900010043', 'C000170043'],
      granularity: 'daily',
      startTime: range.start,
      endTime: range.end,
    });
  });

  it('经 buildDashboardExportParams 产出 device 维度参数，不含 object_ldns / metric_type', () => {
    const p = buildDashboardExportParams(kpiQueryToDashboardSelection(basePayload, range));
    expect(p).toMatchObject({
      dimension: 'device',
      device_sns: ['SN1', 'SN2'],
      metric_paths: ['K900010043', 'C000170043'],
      technologies: ['lte'],
      granularity: 'daily',
      start_time: range.start,
      end_time: range.end,
    });
    expect(p).not.toHaveProperty('object_ldns');
    expect(p).not.toHaveProperty('metric_type');
  });
});
