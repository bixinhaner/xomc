import { describe, it, expect } from 'vitest';
import { buildBatchTaskTypeMap, batchActionHasDetail } from './deviceBatchTask';

// 用恒等 t 让 map 值即 i18n key，便于断言键集与映射关系。
const t = (k: string) => k;

describe('buildBatchTaskTypeMap (#179 抓包入口移除)', () => {
  it('保留 reboot / log-collect / alarm-sync 三个批量操作映射', () => {
    const map = buildBatchTaskTypeMap(t);
    expect(map['batch-reboot']).toBe('common.batchReboot');
    expect(map['batch-log-collect']).toBe('device.action.logCollect');
    expect(map['batch-alarm-sync']).toBe('device.action.alarmSync');
  });

  it('不再识别 batch-tr069-collect（抓包按钮已移除）', () => {
    const map = buildBatchTaskTypeMap(t);
    expect('batch-tr069-collect' in map).toBe(false);
    expect(Object.keys(map)).toHaveLength(3);
  });
});

describe('batchActionHasDetail', () => {
  it('仅日志采集带详情', () => {
    expect(batchActionHasDetail('batch-log-collect')).toBe(true);
  });

  it('重启 / 告警同步 / 已移除的抓包均无详情', () => {
    expect(batchActionHasDetail('batch-reboot')).toBe(false);
    expect(batchActionHasDetail('batch-alarm-sync')).toBe(false);
    expect(batchActionHasDetail('batch-tr069-collect')).toBe(false);
    expect(batchActionHasDetail(undefined)).toBe(false);
  });
});
