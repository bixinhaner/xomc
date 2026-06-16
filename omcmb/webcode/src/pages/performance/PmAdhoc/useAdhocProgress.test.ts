/**
 * issue #399：PM adhoc 进度 SSE reducer 单测。
 * 验证 progress / completed 两类事件应用到 live map 的纯逻辑（成功 + 失败/边界路径）。
 */
import { describe, it, expect } from 'vitest';
import {
  applyProgressEvent,
  applyCompletedEvent,
  type AdhocLiveProgress,
} from '@core/hooks/api/useAdhocProgress';
import type { AdhocProgressEvent, AdhocCompletedEvent } from '@core/types/pmAdhoc';

function progress(taskId: string, p: number, rows: number): AdhocProgressEvent {
  return { task_id: taskId, progress: p, granularity: '15m', rows };
}

describe('#399 applyProgressEvent', () => {
  it('progress 事件写入指定 task 的 progress/rows（空 map → 新建条目）', () => {
    const next = applyProgressEvent(new Map(), progress('t1', 42, 100));
    expect(next.get('t1')).toEqual<AdhocLiveProgress>({ progress: 42, rows: 100 });
  });

  it('后到的 progress 事件覆盖前值（进度推进）', () => {
    let m: ReadonlyMap<string, AdhocLiveProgress> = new Map();
    m = applyProgressEvent(m, progress('t1', 30, 50));
    m = applyProgressEvent(m, progress('t1', 70, 120));
    expect(m.get('t1')).toEqual<AdhocLiveProgress>({ progress: 70, rows: 120 });
  });

  it('多任务互不干扰（按 task_id 分别落）', () => {
    let m: ReadonlyMap<string, AdhocLiveProgress> = new Map();
    m = applyProgressEvent(m, progress('t1', 10, 5));
    m = applyProgressEvent(m, progress('t2', 90, 900));
    expect(m.get('t1')?.progress).toBe(10);
    expect(m.get('t2')?.progress).toBe(90);
  });

  it('已 done 的任务不被迟到的 progress 事件回退（终态优先）', () => {
    let m: ReadonlyMap<string, AdhocLiveProgress> = new Map();
    m = applyCompletedEvent(m, {
      task_id: 't1',
      status: 'succeeded',
      rows_total: 200,
    });
    m = applyProgressEvent(m, progress('t1', 55, 80)); // 迟到事件
    expect(m.get('t1')?.progress).toBe(100);
    expect(m.get('t1')?.done).toBe(true);
  });

  it('返回新 map，不就地修改入参（不可变）', () => {
    const prev = new Map<string, AdhocLiveProgress>();
    const next = applyProgressEvent(prev, progress('t1', 5, 1));
    expect(next).not.toBe(prev);
    expect(prev.size).toBe(0);
  });
});

describe('#399 applyCompletedEvent', () => {
  it('completed(succeeded) 落终态：progress=100、status、rowsTotal、done', () => {
    const ev: AdhocCompletedEvent = {
      task_id: 't1',
      status: 'succeeded',
      rows_total: 360,
    };
    const next = applyCompletedEvent(new Map(), ev);
    expect(next.get('t1')).toEqual<AdhocLiveProgress>({
      progress: 100,
      status: 'succeeded',
      rowsTotal: 360,
      error: undefined,
      done: true,
    });
  });

  it('completed(failed) 携带 error（失败路径）', () => {
    const ev: AdhocCompletedEvent = {
      task_id: 't1',
      status: 'failed',
      rows_total: 0,
      error: 'aggregate timeout',
    };
    const next = applyCompletedEvent(new Map(), ev);
    expect(next.get('t1')?.status).toBe('failed');
    expect(next.get('t1')?.error).toBe('aggregate timeout');
    expect(next.get('t1')?.done).toBe(true);
  });

  it('completed 在已有 progress 条目上落终态（保留 rows，progress 拉满）', () => {
    let m: ReadonlyMap<string, AdhocLiveProgress> = new Map();
    m = applyProgressEvent(m, progress('t1', 80, 288));
    m = applyCompletedEvent(m, { task_id: 't1', status: 'succeeded', rows_total: 360 });
    expect(m.get('t1')?.progress).toBe(100);
    expect(m.get('t1')?.rows).toBe(288); // progress 阶段的 rows 保留
    expect(m.get('t1')?.rowsTotal).toBe(360);
  });
});
