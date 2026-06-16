/**
 * pmDashboard 聚合行映射测试（#22 mapBackend 字段缺失 / 类型漂移路径）：
 *   - mapBackendAggregatedRow：全零 UUID group_id 规整 undefined；filled 占位行 metricValue → null；
 *     metric_type 缺省 → 'counter'；object_ldn null 透传；display_name → displayName。
 *
 * 注：旧拖拽仪表盘 Dashboard / Panel / 用户偏好 mapper 已随后端模块下线（ISSUE-403），
 *     对应测试一并移除，仅保留指标查询命脉聚合行映射。
 */
import { describe, it, expect } from 'vitest';
import {
  mapBackendAggregatedRow,
  type BackendAggregatedRow,
} from '../pmDashboard';

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
