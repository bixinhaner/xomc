/**
 * pmDashboard 类型映射测试（#22 mapBackend 字段缺失 / 类型漂移路径）：
 *   - mapBackendDashboard：shared_with/layout/is_builtin 缺省兜底（[]/{panels:[]}/false）。
 *   - mapBackendPanel：granularities/config 缺省兜底（[]/{}）。
 *   - mapBackendPreferences：technology 缺省 → 'lte'，kpi_card_layout/shared_filters 缺省 → {}。
 *   - mapBackendAggregatedRow：全零 UUID group_id 规整 undefined；filled 占位行 metricValue → null；
 *     metric_type 缺省 → 'counter'；object_ldn null 透传。
 */
import { describe, it, expect } from 'vitest';
import {
  mapBackendDashboard,
  mapBackendPanel,
  mapBackendPreferences,
  mapBackendAggregatedRow,
  type BackendDashboard,
  type BackendPanel,
  type BackendUserPreferences,
  type BackendAggregatedRow,
} from '../pmDashboard';

describe('mapBackendDashboard — 字段缺省兜底', () => {
  it('完整字段逐项映射（snake → camel）', () => {
    const b: BackendDashboard = {
      id: 'd1',
      name: '5G 概览',
      description: 'NR SA',
      owner_id: 'u1',
      shared_with: ['u2', 'u3'],
      parent_dashboard_id: 'p0',
      technology: 'nr',
      layout: { panels: [{ i: 'pn1', x: 0, y: 0, w: 6, h: 4 }] },
      is_builtin: true,
      created_at: '2026-06-01T00:00:00Z',
      updated_at: '2026-06-02T00:00:00Z',
    };
    const d = mapBackendDashboard(b);
    expect(d.ownerId).toBe('u1');
    expect(d.sharedWith).toEqual(['u2', 'u3']);
    expect(d.parentDashboardId).toBe('p0');
    expect(d.technology).toBe('nr');
    expect(d.isBuiltin).toBe(true);
    expect(d.layout.panels).toHaveLength(1);
  });

  it('shared_with/layout/is_builtin 缺省 → []/{panels:[]}/false', () => {
    const b = {
      id: 'd2',
      name: 'x',
      owner_id: 'u1',
      technology: 'lte',
      created_at: 'c',
      updated_at: 'u',
    } as unknown as BackendDashboard;
    const d = mapBackendDashboard(b);
    expect(d.sharedWith).toEqual([]);
    expect(d.layout).toEqual({ panels: [] });
    expect(d.isBuiltin).toBe(false);
  });
});

describe('mapBackendPanel — granularities/config 缺省兜底', () => {
  it('granularities 缺省 → []，config 缺省 → {}', () => {
    const b = {
      id: 'pn1',
      dashboard_id: 'd1',
      panel_type: 'kpi_card',
      title: 'RRC 建立成功率',
      metric_paths: ['RRC_CONN_SETUP_SR'],
      dimension: 'device',
      time_range: { start_offset: '-1h' },
    } as unknown as BackendPanel;
    const p = mapBackendPanel(b);
    expect(p.panelType).toBe('kpi_card');
    expect(p.granularities).toEqual([]);
    expect(p.config).toEqual({});
    expect(p.dimension).toBe('device');
  });

  it('完整字段映射（含 compare_mode / adhoc_task_id）', () => {
    const b: BackendPanel = {
      id: 'pn2',
      dashboard_id: 'd1',
      panel_type: 'adhoc_result',
      title: '自定义聚合',
      metric_paths: [],
      granularities: ['hourly', 'daily'],
      dimension: 'device_group',
      device_group_ids: ['g1'],
      time_range: { absolute_start: '2026-06-01T00:00:00Z', absolute_end: '2026-06-02T00:00:00Z' },
      compare_mode: 'previous_window',
      adhoc_task_id: 'task-9',
      config: { topN: 10 },
    };
    const p = mapBackendPanel(b);
    expect(p.granularities).toEqual(['hourly', 'daily']);
    expect(p.compareMode).toBe('previous_window');
    expect(p.adhocTaskId).toBe('task-9');
    expect(p.config).toEqual({ topN: 10 });
  });
});

describe('mapBackendPreferences — technology 缺省 → lte', () => {
  it('technology 缺省（undefined）回退 lte，layout/filters 缺省 → {}', () => {
    // mapper 用 `?? 'lte'`：只兜 null/undefined。technology 整个缺字段时走此路径。
    const b = {
      user_id: 'u1',
    } as unknown as BackendUserPreferences;
    const pref = mapBackendPreferences(b);
    expect(pref.userId).toBe('u1');
    expect(pref.technology).toBe('lte');
    expect(pref.kpiCardLayout).toEqual({});
    expect(pref.sharedFilters).toEqual({});
  });

  it('technology 为空串时按现行实现透传空串（?? 不兜空串，记录现状契约）', () => {
    // 类型漂移现状：后端理论不会返回空 technology，但 `??` 不拦空串。
    // 本断言锁定现行行为，若未来改为 `||` 兜底需同步更新。
    const b = { user_id: 'u1', technology: '' } as unknown as BackendUserPreferences;
    const pref = mapBackendPreferences(b);
    expect(pref.technology).toBe('');
  });

  it('完整字段透传', () => {
    const b: BackendUserPreferences = {
      user_id: 'u1',
      technology: 'nr',
      kpi_card_layout: { order: ['a', 'b'] },
      current_dashboard_id: 'd9',
      shared_filters: { region: '华东' },
    };
    const pref = mapBackendPreferences(b);
    expect(pref.technology).toBe('nr');
    expect(pref.currentDashboardId).toBe('d9');
    expect(pref.kpiCardLayout).toEqual({ order: ['a', 'b'] });
  });
});

describe('mapBackendAggregatedRow — 占位行 / 全零 UUID / 类型漂移', () => {
  function row(overrides: Record<string, unknown> = {}): BackendAggregatedRow {
    return {
      device_sn: 'SN001',
      device_group_id: '00000000-0000-0000-0000-000000000000',
      metric_path: 'C1',
      metric_type: 'counter',
      metric_value: 123,
      granularity: 'hourly',
      time: '2026-06-10T00:00:00Z',
      start_time: '2026-06-10T00:00:00Z',
      end_time: '2026-06-10T01:00:00Z',
      ingest_time: '2026-06-10T01:05:00Z',
      ...overrides,
    };
  }

  it('全零 UUID 的 device_group_id 规整为 undefined（device 维度查询）', () => {
    const r = mapBackendAggregatedRow(row());
    expect(r.deviceGroupId).toBeUndefined();
    expect(r.deviceSn).toBe('SN001');
    expect(r.metricValue).toBe(123);
  });

  it('真实 group_id 透传', () => {
    const r = mapBackendAggregatedRow(row({ device_group_id: 'real-group-id', device_sn: '' }));
    expect(r.deviceGroupId).toBe('real-group-id');
    expect(r.deviceSn).toBeUndefined(); // 空串 → undefined
  });

  it('filled=true 占位行：metricValue 强制 null（渲染 "-"，忽略无意义的 metric_value）', () => {
    const r = mapBackendAggregatedRow(row({ filled: true, metric_value: 0 }));
    expect(r.metricValue).toBeNull();
    expect(r.filled).toBe(true);
  });

  it('object_ldn null 透传；extra 缺省 → {}', () => {
    const r = mapBackendAggregatedRow(row({ object_ldn: null }));
    expect(r.objectLdn).toBeNull();
    expect(r.extra).toEqual({});
  });

  it('KPI 行 display_name 透传到 displayName（展示层用友好名）', () => {
    const r = mapBackendAggregatedRow(
      row({ metric_type: 'kpi', metric_path: 'K001', display_name: 'RRC 建立成功率' }),
    );
    expect(r.metricType).toBe('kpi');
    expect(r.displayName).toBe('RRC 建立成功率');
  });
});
