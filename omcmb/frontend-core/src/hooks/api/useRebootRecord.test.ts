import { describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  useQuery: vi.fn(),
}));

vi.mock('@tanstack/react-query', () => ({
  useQuery: mocks.useQuery,
}));

vi.mock('../../services/api/rebootRecordApi', () => ({
  rebootRecordApi: {
    list: vi.fn(),
    statByDevice: vi.fn(),
  },
}));

describe('useRebootRecordList', () => {
  it('passes through a page auto-refresh interval when configured', async () => {
    const { useRebootRecordList } = await import('./useRebootRecord');

    useRebootRecordList({ page: 1, pageSize: 20 }, { refetchInterval: 30_000 });

    expect(mocks.useQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: true,
        refetchInterval: 30_000,
      }),
    );
  });
});
