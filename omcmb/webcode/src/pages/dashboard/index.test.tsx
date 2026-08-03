import {
  act,
  cleanup,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useUserStore } from '@core/store/userStore';
import type { DashboardSummary } from '@core/types/dashboard';
import type { DeviceListStats } from '@core/types/device';
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

interface DeviceStatsHookResult {
  data: DeviceListStats | undefined;
  dataUpdatedAt: number;
  isPending: boolean;
  isFetching: boolean;
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
  deviceStatsResult: {
    data: undefined,
    dataUpdatedAt: 0,
    isPending: true,
    isFetching: true,
  } as DeviceStatsHookResult,
  useDashboardSummary: vi.fn(),
  useDashboardDeviceStats: vi.fn(),
}));

const navigationMocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  openTab: vi.fn(),
}));

vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>();
  return {
    ...actual,
    useNavigate: () => navigationMocks.navigate,
  };
});

vi.mock('@core/store/tabStore', () => ({
  useTabStore: (
    selector: (state: { openTab: typeof navigationMocks.openTab }) => unknown,
  ) => selector({ openTab: navigationMocks.openTab }),
}));

vi.mock('@core/hooks/api/useDashboard', () => ({
  useDashboardSummary: dashboardMocks.useDashboardSummary,
  useDashboardDeviceStats: dashboardMocks.useDashboardDeviceStats,
  useDeviceStatusByType: () => ({ data: {}, isLoading: false }),
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

vi.mock('@core/hooks/api/useTechnologyDictionary', () => ({
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
	pmSlotHealth: [{
	  slotEnd: '2026-08-03T15:00:00Z',
	  technology: 'lte',
	  carrier: 'cmcc',
	  expectedDevices: 20_000,
	  receivedDevices: 19_600,
	  coverageRatio: 0.98,
	  status: 'complete',
	  evaluatedAt: '2026-08-03T15:12:00Z',
	}, {
	  slotEnd: '2026-08-03T15:00:00Z',
	  technology: 'lte',
	  carrier: 'cucc',
	  expectedDevices: 1_000,
	  receivedDevices: 980,
	  coverageRatio: 0.98,
	  status: 'complete',
	  evaluatedAt: '2026-08-03T15:12:00Z',
	}],
};

describe('DashboardPage card snapshot integration', () => {
  beforeEach(() => {
    window.localStorage.clear();
    navigationMocks.navigate.mockReset();
    navigationMocks.openTab.mockReset();
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
    dashboardMocks.deviceStatsResult = {
      data: undefined,
      dataUpdatedAt: 0,
      isPending: true,
      isFetching: true,
    };
    dashboardMocks.useDashboardDeviceStats.mockImplementation(
      () => dashboardMocks.deviceStatsResult
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

  it('does not render the temporarily hidden profile and quick-access cards', () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>,
    );

    expect(screen.queryByText('User A')).not.toBeInTheDocument();
    expect(screen.queryByText('dashboard.quickAccess')).not.toBeInTheDocument();
  });

	it('shows the latest persisted PM slot coverage for the selected technology', () => {
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

	  expect(screen.getByText('20580 / 21000 · 98.0%')).toBeInTheDocument();
	  expect(screen.getByText('dashboard.pmSlotCoverage')).toBeInTheDocument();
	});

  it('uses device-list stats for device cards without changing other Summary cards', () => {
    dashboardMocks.summaryResult = {
      data: successfulSummary,
      dataUpdatedAt: 200,
      isLoading: false,
      isPending: false,
      isFetching: false,
      refetch: vi.fn(),
    };
    dashboardMocks.deviceStatsResult = {
      data: {
        total: 20_002,
        online_count: 19_572,
        offline_count: 430,
        alarmed: 0,
      },
      dataUpdatedAt: 300,
      isPending: false,
      isFetching: false,
    };

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>,
    );

    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveTextContent('20002');
    expect(screen.getByTestId('kpi-dashboard.onlineDevices')).toHaveTextContent('19572');
    expect(screen.getByTestId('kpi-dashboard.activeAlarmsEvents')).toHaveTextContent('4');
    expect(screen.getByTestId('kpi-dashboard.activeUE')).toHaveTextContent('8');
    expect(dashboardMocks.useDashboardDeviceStats).toHaveBeenCalledWith(
      resolveDashboardCardApiScope('/api/v1', undefined, window.location.origin),
      user.id,
    );
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

  it('shows current device stats while Summary is still pending under load', () => {
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
    dashboardMocks.deviceStatsResult = {
      data: {
        total: 20_002,
        online_count: 19_572,
        offline_count: 430,
        alarmed: 0,
      },
      dataUpdatedAt: 300,
      isPending: false,
      isFetching: false,
    };

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DashboardPage />
      </QueryClientProvider>,
    );

    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveTextContent('20002');
    expect(screen.getByTestId('kpi-dashboard.totalDevices')).toHaveAttribute('data-loading', 'false');
    expect(screen.getByTestId('kpi-dashboard.onlineDevices')).toHaveTextContent('19572');
    expect(screen.getByTestId('kpi-dashboard.activeAlarmsEvents')).toHaveTextContent('2');
    expect(screen.getByTestId('kpi-dashboard.activeUE')).toHaveTextContent('3');
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
