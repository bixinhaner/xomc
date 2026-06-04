import { describe, it, expect } from 'vitest';
import { mapBackendAdhocTask, type BackendAdhocTask } from '../pmAdhoc';

// 一份合法的后端任务基底；各用例只覆盖差异字段。
function baseBackendTask(): BackendAdhocTask {
  return {
    id: 'task-1',
    name: '产品维度任务',
    mode: 'oneshot',
    device_sns: [],
    metric_paths: ['C1'],
    granularities: ['15min'],
    window_start: '2026-06-01T00:00:00Z',
    window_end: '2026-06-02T00:00:00Z',
    dimension: 'product',
    technology: 'lte',
    status: 'succeeded',
    progress: 100,
    creator: 'admin',
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
  };
}

describe('mapBackendAdhocTask — device_sns 健壮性（产品/频段/全网维度回传 null）', () => {
  it('device_sns 为 null 时映射成空数组（不破坏前端 string[] 契约）', () => {
    // 产品/频段/全网维度任务后端列存 JSON null → 响应 device_sns 为 null。
    const b = { ...baseBackendTask(), device_sns: null } as unknown as BackendAdhocTask;
    const t = mapBackendAdhocTask(b);
    expect(t.deviceSns).toEqual([]);
    // 详情页 / 结果面板都对 deviceSns 取 .length / .map，必须可安全调用。
    expect(() => t.deviceSns.length).not.toThrow();
  });

  it('device_sns 字段缺失时映射成空数组', () => {
    const b = baseBackendTask();
    delete (b as Partial<BackendAdhocTask>).device_sns;
    const t = mapBackendAdhocTask(b);
    expect(t.deviceSns).toEqual([]);
  });

  it('device_sns 为正常数组时原样保留（设备维度不受影响）', () => {
    const b = { ...baseBackendTask(), device_sns: ['SN1', 'SN2'], dimension: 'device' };
    const t = mapBackendAdhocTask(b);
    expect(t.deviceSns).toEqual(['SN1', 'SN2']);
  });
});
