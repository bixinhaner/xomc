import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Header from './index';

const mocks = vi.hoisted(() => ({
  onStorageProtectionClick: vi.fn(),
  setAlarmCounts: vi.fn(),
  setMobileOverlayOpen: vi.fn(),
  syncStale: vi.fn(),
  toggleLocale: vi.fn(),
  toggleSidebar: vi.fn(),
  toggleTheme: vi.fn(),
}));

vi.mock('@core/store/appStore', () => ({
  useAppStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({
      deviceType: 'all',
      theme: 'classic',
      locale: 'zh-CN',
      sidebarCollapsed: false,
      setMobileOverlayOpen: mocks.setMobileOverlayOpen,
      toggleLocale: mocks.toggleLocale,
      toggleSidebar: mocks.toggleSidebar,
      toggleTheme: mocks.toggleTheme,
      omcName: undefined,
    }),
}));

vi.mock('@core/hooks/api/useOmcName', () => ({
  resolveOmcName: (name: string | undefined, fallback: string) => name || fallback,
  usePublicOmcName: () => ({ omcName: undefined }),
}));

vi.mock('@core/store/alarmStore', () => ({
  useAlarmStore: (selector: (state: { setCounts: typeof mocks.setAlarmCounts }) => unknown) =>
    selector({ setCounts: mocks.setAlarmCounts }),
}));

vi.mock('@core/hooks/api/useNotificationCenter', () => ({
  useNotificationUnreadCount: () => ({ data: 0 }),
  useSyncStaleNotifications: () => ({ mutate: mocks.syncStale }),
}));

vi.mock('@core/hooks/api/useAlarms', () => ({
  useAlarmCount: () => ({ data: undefined }),
}));

vi.mock('@/hooks/useResponsive', () => ({
  useResponsive: () => ({ isMobile: false }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

vi.mock('@/components/NotificationCenter', () => ({
  default: () => <div />,
}));

vi.mock('./AlarmBadges', () => ({
  default: () => <div data-testid="alarm-badges" />,
}));

vi.mock('./TimezoneSelector', () => ({
  default: () => <div data-testid="timezone-selector" />,
}));

vi.mock('./UserDropdown', () => ({
  default: () => <div data-testid="user-dropdown" />,
}));

describe('Header storage protection badge', () => {
  beforeEach(() => {
    mocks.onStorageProtectionClick.mockReset();
  });

  it('显示收起后的存储阻断红色入口并可重新展开', () => {
    render(
      <Header
        storageProtectionBlocked
        onStorageProtectionClick={mocks.onStorageProtectionClick}
      />,
    );

    const badge = screen.getByRole('button', {
      name: 'system.storageProtection.globalBlock.expand',
    });
    expect(badge).toHaveTextContent('system.storageProtection.globalBlock.badge');

    fireEvent.click(badge);
    expect(mocks.onStorageProtectionClick).toHaveBeenCalledOnce();
  });
});
