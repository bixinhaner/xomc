/**
 * deviceTask 类型测试（#22 task 域）：
 *   isDeviceTaskTerminal —— useDeviceTaskStatus 轮询停机判定的纯函数。
 *   终态（completed/failed/expired/cancelled）→ true；进行态（pending/sent）+ undefined → false。
 *   该判定决定 React Query refetchInterval 是否返回 false（停轮询），错判会导致死循环轮询或过早停轮。
 */
import { describe, it, expect } from 'vitest';
import {
  isDeviceTaskTerminal,
  DEVICE_TASK_TERMINAL_STATUSES,
  type DeviceTaskStatus,
} from '../deviceTask';

describe('isDeviceTaskTerminal', () => {
  it('终态返 true（停止轮询）', () => {
    for (const s of ['completed', 'failed', 'expired', 'cancelled'] as DeviceTaskStatus[]) {
      expect(isDeviceTaskTerminal(s)).toBe(true);
    }
  });

  it('进行态返 false（继续轮询）', () => {
    expect(isDeviceTaskTerminal('pending')).toBe(false);
    expect(isDeviceTaskTerminal('sent')).toBe(false);
  });

  it('undefined（尚无数据）返 false（首次仍轮询）', () => {
    expect(isDeviceTaskTerminal(undefined)).toBe(false);
  });

  it('终态常量集合恰好包含 4 个终态', () => {
    expect([...DEVICE_TASK_TERMINAL_STATUSES].sort()).toEqual(
      ['cancelled', 'completed', 'expired', 'failed'],
    );
  });
});
