import { App } from 'antd';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AttentionBar from './AttentionBar';

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  refetchSummary: vi.fn(),
  refetchPage: vi.fn(),
  useAttentionSummary: vi.fn(),
  useAttentionPage: vi.fn(),
}));

vi.mock('react-router-dom', () => ({ useNavigate: () => mocks.navigate }));
vi.mock('@core/hooks/api/useAttention', () => ({
  useAttentionSummary: mocks.useAttentionSummary,
  useAttentionPage: mocks.useAttentionPage,
}));
vi.mock('@/hooks/useT', () => ({ useT: () => (key: string) => key }));
vi.mock('@core/utils/systemTime', () => ({ formatSystemTime: (value: string) => value }));

const item = (id: string, kind: 'active_alarm' | 'device_access_review', route: string) => ({
  id,
  kind,
  source: kind === 'active_alarm' ? 'alarm' : 'device_access',
  sourceId: id,
  title: `title-${id}`,
  summary: `summary-${id}`,
  severity: kind === 'active_alarm' ? 'critical' : undefined,
  priority: kind === 'active_alarm' ? 'critical' : 'normal',
  createdAt: '2026-08-24T09:00:00Z',
  detailRoute: route,
  allowedActions: kind === 'active_alarm'
    ? ['view_alarm' as const]
    : ['review_device_candidate' as const],
});

const abnormalities = [
  item('a1', 'active_alarm', '/alarm/current?alarmId=a1'),
  item('a2', 'active_alarm', '/alarm/current?alarmId=a2'),
  item('a3', 'active_alarm', '/alarm/current?alarmId=a3'),
];
const todos = [
  item('t1', 'device_access_review', '/device/access-control?tab=candidates&candidateId=t1'),
  item('t2', 'device_access_review', '/device/access-control?tab=candidates&candidateId=t2'),
  item('t3', 'device_access_review', '/device/access-control?tab=candidates&candidateId=t3'),
];

function summaryResult(overrides: Record<string, unknown> = {}) {
  return {
    data: {
      abnormalities: { status: 'ok', total: 3, items: abnormalities },
      todos: { status: 'ok', total: 3, items: todos },
      generatedAt: '2026-08-24T09:01:00Z',
    },
    isPending: false,
    isError: false,
    refetch: mocks.refetchSummary,
    ...overrides,
  };
}

describe('AttentionBar', () => {
  beforeEach(() => {
    mocks.navigate.mockReset();
    mocks.useAttentionSummary.mockReset().mockReturnValue(summaryResult());
    mocks.useAttentionPage.mockReset().mockImplementation((section: 'abnormalities' | 'todos', page: number, pageSize: number, enabled: boolean) => ({
      data: { status: 'ok', total: 41, items: section === 'todos' ? todos : abnormalities, page, pageSize },
      isPending: false,
      isError: false,
      isFetching: false,
      refetch: mocks.refetchPage,
      enabled,
    }));
  });

  it('renders only the highest-priority preview item and navigates without opening the drawer', async () => {
    const user = userEvent.setup();
    render(<App><AttentionBar /></App>);

    expect(screen.getByText('title-a1')).toBeInTheDocument();
    expect(screen.queryByText('title-a2')).not.toBeInTheDocument();
    expect(screen.queryByText('title-a3')).not.toBeInTheDocument();
    expect(screen.getByText('title-t1')).toBeInTheDocument();
    expect(screen.queryByText('title-t2')).not.toBeInTheDocument();
    expect(screen.queryByText('title-t3')).not.toBeInTheDocument();
    expect(screen.getByText('dashboard.attention.level.critical')).toBeInTheDocument();
    expect(screen.getByText('dashboard.attention.type.deviceReview')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /title-a1/ }));
    expect(mocks.navigate).toHaveBeenCalledWith('/alarm/current?alarmId=a1');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('opens the requested read-only drawer for only that section and requests page 2', async () => {
    const user = userEvent.setup();
    render(<App><AttentionBar /></App>);

    await user.click(screen.getAllByRole('button', { name: 'dashboard.attention.viewAll' })[1]);
    const drawer = await screen.findByRole('dialog');
    expect(within(drawer).getByText('dashboard.attention.todos')).toBeInTheDocument();
    expect(within(drawer).queryByRole('tab')).not.toBeInTheDocument();
    expect(within(drawer).queryByText('title-a1')).not.toBeInTheDocument();
    expect(within(drawer).getByText('title-t1')).toBeInTheDocument();
    expect(mocks.useAttentionPage).toHaveBeenLastCalledWith('todos', 1, 20, true);

    await user.click(within(drawer).getByTitle('2'));
    await waitFor(() => expect(mocks.useAttentionPage).toHaveBeenLastCalledWith('todos', 2, 20, true));
    expect(within(drawer).queryByText('common.approve')).not.toBeInTheDocument();
    expect(within(drawer).queryByText('common.reject')).not.toBeInTheDocument();
  });

  it('renders partial and empty section states without hiding the other section', () => {
    mocks.useAttentionSummary.mockReturnValue(summaryResult({
      data: {
        abnormalities: { status: 'partial', total: 0, items: [] },
        todos: { status: 'ok', total: 3, items: todos },
        generatedAt: '2026-08-24T09:01:00Z',
      },
    }));
    render(<App><AttentionBar /></App>);

    expect(screen.getByText('dashboard.attention.partial')).toBeInTheDocument();
    expect(screen.getByText('dashboard.attention.emptyAbnormalities')).toBeInTheDocument();
    expect(screen.getByText('title-t1')).toBeInTheDocument();
  });
});
