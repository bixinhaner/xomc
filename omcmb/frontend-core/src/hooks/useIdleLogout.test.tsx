import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useIdleLogout } from './useIdleLogout';

const mocks = vi.hoisted(() => ({
  lock: vi.fn(),
  logout: vi.fn(),
}));

vi.mock('../store/userStore', () => ({
  useUserStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({ lock: mocks.lock, logout: mocks.logout }),
}));

describe('useIdleLogout', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-16T00:00:00Z'));
    window.history.replaceState({}, '', '/system/config?tab=security');
    mocks.lock.mockReset();
    mocks.logout.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('空闲超时时锁定会话而不是执行主动退出', () => {
    renderHook(() => useIdleLogout(1));

    act(() => {
      vi.advanceTimersByTime(60_000);
    });

    expect(mocks.lock).toHaveBeenCalledOnce();
    expect(mocks.lock).toHaveBeenCalledWith('/system/config?tab=security');
    expect(mocks.logout).not.toHaveBeenCalled();
  });
});
