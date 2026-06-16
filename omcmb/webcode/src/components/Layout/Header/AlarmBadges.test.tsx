import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import AlarmBadges from './AlarmBadges';

const navigate = vi.fn();
const openTab = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    useNavigate: () => navigate,
  };
});

vi.mock('@core/store/alarmStore', () => ({
  useAlarmStore: (selector: (state: { counts: Record<string, number> }) => unknown) =>
    selector({
      counts: {
        critical: 17,
        major: 16,
        minor: 0,
        warning: 2,
      },
    }),
}));

vi.mock('@core/store/tabStore', () => ({
  useTabStore: (selector: (state: { openTab: typeof openTab }) => unknown) => selector({ openTab }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

describe('AlarmBadges', () => {
  beforeEach(() => {
    navigate.mockReset();
    openTab.mockReset();
  });

  it('点击重要告警徽章时按对应 severity 打开当前告警标签并导航', () => {
    render(<AlarmBadges />);

    fireEvent.click(screen.getByRole('button', { name: '▲16' }));

    expect(openTab).toHaveBeenCalledWith({
      key: 'alarm-current',
      label: 'nav.alarm.current',
      labelRaw: false,
      path: '/alarm/current?severity=major',
      closable: true,
    });
    expect(navigate).toHaveBeenCalledWith('/alarm/current?severity=major');
  });
});