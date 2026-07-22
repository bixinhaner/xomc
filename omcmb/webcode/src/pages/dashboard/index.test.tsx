import { act, cleanup, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useUserStore } from '@core/store/userStore';
import type { DashboardSummary } from '@core/types/dashboard';
import {
  readDashboardCardSnapshot,
  resolveDashboardCardApiScope,
  writeDashboardCardSnapshot,
  type DashboardCardSnapshot,
} from '@core/utils/dashboardCardSnapshot';
import { runDashboardSummaryRefresh } from '@core/utils/dashboardRefresh';
import DashboardPage from './index';

interface SummaryHookResult {
  data: DashboardSummary | undefined;
  dataUpdatedAt: number;
  isLoading: boolean;
  isPending: boolean;
  isFetching: boolean;
  refetch: ReturnType<typeof vi.fn>;
}

const dashboardMocks = vi.hoisted(() => ({
  summaryResult: {
    data: undefined,
    dataUpdatedAt: 0,
    isLoading: true,
    isPending: true,
    isFetching: true,
    refetch: vi.fn(),
  } as SummaryHookResult,
  useDashboardSummary: vi.fn(),
  useDashboardRealtime: vi.fn(),
}));

vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>();
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  };
});

vi.mock('@core/hooks/api/useDashboard', () => ({
  useDashboardSummary: dashboardMocks.useDashboardSummary,
  useDeviceStatusByType: () => ({ data: {}, isLoading: false }),
}));

vi.mock('@core/hooks/api/useDashboardRealtime', () => ({
  useDashboardRealtime: dashboardMocks.useDashboardRealtime,
}));

vi.mock('@/components/KPICard', () => ({
  default: ({
    title,
    value,
    loading,
    hasComparison,
  }: {
    title: string;
    value: number | string;
    loading?: boolean;
    hasComparison?: boolean;
  }) => (
    <div
      data-testid={`kpi-${title}`}
      data-loading={String(Boolean(loading))}
      data-has-comparison={String(hasComparison)}
    >
      {String(value)}
    </div>
  ),
}));

vi.mock('@/components/Charts/BarChart', () => ({
  default: () => <div data-testid="bar-chart" />,
}));

vi.mock('@/components/common/EmptyState', () => ({
  default: () => <div data-testid="empty-state" />,
}));

vi.mock('./DashboardKPIModules', () => ({
  DashboardKPIModules: () => <div data-testid="dashboard-kpi-modules" />,
}));

vi.mock('@/components/dashboard/useTechnologyDictionary', () => ({
  useTechnologyDictionary: () => ({ options: [] }),
}));

vi.mock('@/components/Effects', () => ({
  TiltCard: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({ colorPrimary: '#1677ff' }),
}));

vi.mock('@/hooks/useScrollReveal', () => ({
  useScrollReveal: vi.fn(),
}));

vi.mock('@/components/MenuBootstrap/featureFlag', () => ({
  isDynamicMenuEnabled: () => false,
}));

vi.mock('@core/utils/routeAccess', () => ({
  isRouteAllowed: () => true,
}));

const user = {
  id: 'user-a',
  username: 'user-a',
  displayName: 'User A',
  email: 'user-a@example.com',
  role: 'admin' as const,
  status: 'active' as const,
  createTime: '2026-07-20T00:00:00Z',
};

const userB = {
  ...user,
  id: 'user-b',
  username: 'user-b',
  displayName: 'User B',
  email: 'user-b@example.com',
};

const previousSnapshot: DashboardCardSnapshot = {
  totalDevices: 10,
  onlineDevices: 6,
  activeAlarms: 2,
  activeUE: 3,
  updatedAt: 100,
};

const successfulSummary: DashboardSummary = {
  deviceCounts: {
    total: 20,
    online: 15,
    offline: 5,
    alarm: 2,
  },
  alarmCounts: {
    critical: 1,
    major: 1,
    minor: 1,
    warning: 1,
    total: 4,
  },
  kpiSummary: {
    UE_ACTIVE: 8.9,
  },
  kpiDeltas: {},
  taskSummary: {
    running: 0,
    pending: 0,
    success: 0,
    failed: 0,
  },
};

describe('DashboardPage card snapshot integration', () => {
  beforeEach(() => {
    window.localStorage.clear();
    dashboardMocks.summaryResult = {
      data: undefined,
      dataUpdatedAt: 0,
      isLoading: true,
      isPending: true,
      isFetching: true,
      refetch: vi.fn(),
    };
    dashboardMocks.useDashboardSummary.mockImplementation(
      () => dashboardMocks.summaryResult
    );
    useUserStore.setState({
      currentUser: user,
      isAuthenticated: true,
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('requests refetch errors to be thrown to the refresh handler', async () => {
    const error = new Error('summary failed');
    const refetch = vi.fn().mockRejectedValue(error);

    await expect(runDashboardSummaryRefresh(refetch)).rejects.toThrow(
      'summary failed',
    );
    expect(refetch).toHaveBeenCalledWith({ throwOnError: true });
  });

  it('loads the current user snapshot while Summary is pending', () => {
    const apiScope = resolveDashboardCardApiScope(
      '/api/v1',
      undefined,
      window.location.origin,
    );
    writeDashboardCardSnapshot(
      window.localStorage,
      apiScope,
      user.id,
      previousSnapshot,
    );

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>
    );

    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveTextContent('10');
    expect(screen.getByTestId('kpi-dashboard.onlineDevices')).toHaveTextContent('6');
    expect(screen.getByTestId('kpi-dashboard.activeAlarmsEvents')).toHaveTextContent('2');
    expect(screen.getByTestId('kpi-dashboard.activeUE')).toHaveTextContent('3');
    expect(screen.getByText(/dashboard\.lastUpdate/)).toHaveTextContent(
      'dashboard.daysAgo',
    );
  });

  it('keeps the latest successful Summary in memory if Query data disappears', async () => {
    const apiScope = resolveDashboardCardApiScope(
      '/api/v1',
      undefined,
      window.location.origin,
    );
    writeDashboardCardSnapshot(
      window.localStorage,
      apiScope,
      user.id,
      previousSnapshot,
    );

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const view = render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>
    );

    dashboardMocks.summaryResult = {
      data: successfulSummary,
      dataUpdatedAt: 200,
      isLoading: false,
      isPending: false,
      isFetching: false,
      refetch: vi.fn(),
    };
    view.rerender(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveTextContent('20');
    });

    dashboardMocks.summaryResult = {
      data: undefined,
      dataUpdatedAt: 0,
      isLoading: true,
      isPending: true,
      isFetching: true,
      refetch: vi.fn(),
    };
    view.rerender(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>
    );

    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveTextContent('20');
    expect(screen.getByTestId('kpi-dashboard.onlineDevices')).toHaveTextContent('15');
    expect(screen.getByTestId('kpi-dashboard.activeAlarmsEvents')).toHaveTextContent('4');
    expect(screen.getByTestId('kpi-dashboard.activeUE')).toHaveTextContent('8');
  });

  it('keeps first-load skeletons while an offline query is pending but paused', () => {
    dashboardMocks.summaryResult = {
      data: undefined,
      dataUpdatedAt: 0,
      isLoading: false,
      isPending: true,
      isFetching: false,
      refetch: vi.fn(),
    };

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>
    );

    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveAttribute(
      'data-loading',
      'true',
    );
    expect(screen.getByText(/dashboard\.lastUpdate/)).toHaveTextContent('--');
  });

  it('marks period-comparison cards unavailable when Summary has no valid baselines', () => {
    dashboardMocks.summaryResult = {
      data: successfulSummary,
      dataUpdatedAt: 200,
      isLoading: false,
      isPending: false,
      isFetching: false,
      refetch: vi.fn(),
    };

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>,
    );

    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveAttribute(
      'data-has-comparison',
      'false',
    );
    expect(screen.getByTestId('kpi-dashboard.activeAlarmsEvents')).toHaveAttribute(
      'data-has-comparison',
      'false',
    );
    expect(screen.getByTestId('kpi-dashboard.activeUE')).toHaveAttribute(
      'data-has-comparison',
      'false',
    );
  });

  it('does not display another user snapshot on the same API', () => {
    const apiScope = resolveDashboardCardApiScope(
      '/api/v1',
      undefined,
      window.location.origin,
    );
    writeDashboardCardSnapshot(
      window.localStorage,
      apiScope,
      'user-b',
      previousSnapshot,
    );

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>
    );

    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveTextContent('0');
    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveAttribute(
      'data-loading',
      'true',
    );
  });

  it('does not reuse or persist user A Summary after switching directly to user B', async () => {
    const apiScope = resolveDashboardCardApiScope(
      '/api/v1',
      undefined,
      window.location.origin,
    );
    const userASummaryResult: SummaryHookResult = {
      data: successfulSummary,
      dataUpdatedAt: 200,
      isLoading: false,
      isPending: false,
      isFetching: false,
      refetch: vi.fn(),
    };
    const userBPendingResult: SummaryHookResult = {
      data: undefined,
      dataUpdatedAt: 0,
      isLoading: true,
      isPending: true,
      isFetching: true,
      refetch: vi.fn(),
    };
    dashboardMocks.useDashboardSummary.mockImplementation(
      (_apiScope: string, userScope: string | undefined) =>
        userScope === user.id ? userASummaryResult : userBPendingResult,
    );
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveTextContent('20');
      expect(
        readDashboardCardSnapshot(window.localStorage, apiScope, user.id),
      ).toBeDefined();
    });

    act(() => {
      useUserStore.setState({
        currentUser: userB,
        isAuthenticated: true,
      });
    });

    await waitFor(() => {
      expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveTextContent('0');
      expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveAttribute(
        'data-loading',
        'true',
      );
    });
    expect(
      readDashboardCardSnapshot(window.localStorage, apiScope, userB.id),
    ).toBeUndefined();
    expect(dashboardMocks.useDashboardSummary).toHaveBeenLastCalledWith(
      apiScope,
      userB.id,
    );
  });
});
