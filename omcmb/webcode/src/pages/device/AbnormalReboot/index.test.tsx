import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import AbnormalReboot from './index';

const mocks = vi.hoisted(() => ({
  refetchList: vi.fn(),
  useRebootRecordList: vi.fn(),
  useRebootRecordStatByDevice: vi.fn(),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

vi.mock('@/components/Layout/ListPageLayout', () => ({
  default: ({ title, children }: { title: string; children: React.ReactNode }) => (
    <section>
      <h1>{title}</h1>
      {children}
    </section>
  ),
}));

vi.mock('@/components/FilterBar', () => ({
  default: () => <div data-testid="filter-bar" />,
}));

vi.mock('@/components/DataTable', () => ({
  default: () => <div data-testid="reboot-record-table" />,
}));

vi.mock('@core/hooks/api/useRebootRecord', () => ({
  useRebootRecordList: mocks.useRebootRecordList,
  useRebootRecordStatByDevice: mocks.useRebootRecordStatByDevice,
}));

vi.mock('@core/services/api/rebootRecordApi', () => ({
  rebootRecordApi: {
    list: vi.fn(),
  },
}));

describe('AbnormalReboot', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('exposes manual refresh and enables automatic list refresh', () => {
    mocks.useRebootRecordList.mockReturnValue({
      data: { items: [], total: 0 },
      isLoading: false,
      isFetching: false,
      refetch: mocks.refetchList,
    });
    mocks.useRebootRecordStatByDevice.mockReturnValue({
      data: { items: [] },
      isLoading: false,
    });

    render(<AbnormalReboot />);

    expect(mocks.useRebootRecordList).toHaveBeenCalledWith(
      { page: 1, pageSize: 20, rebootType: 'all' },
      expect.objectContaining({ refetchInterval: 30_000 }),
    );

    fireEvent.click(screen.getByRole('button', { name: /common.refresh/ }));

    expect(mocks.refetchList).toHaveBeenCalledTimes(1);
  });
});
