import dayjs from 'dayjs';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { zhCN } from '@core/i18n';
import { usePmPageStateStore } from '@core/store/pmPageStateStore';
import {
  buildSubmittedTaskDashboardQuery,
  buildTaskDashboardStateSnapshot,
  PM_DASHBOARD_PAGE_KEY,
} from './taskDashboardState';

const usePmAdhocResultsMock = vi.fn();

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  usePmAdhocDetail: () => ({
    data: {
      id: 'task-1',
      name: '内置-全网-LTE',
      mode: 'continuous',
      deviceSns: [],
      metricPaths: ['M1'],
      granularities: ['hourly'],
      windowStart: '2026-07-01T00:00:00Z',
      windowEnd: '2026-07-02T00:00:00Z',
      dimension: 'network',
      technology: 'lte',
      isBuiltin: true,
      expireDays: 60,
      visibility: 'public',
      status: 'succeeded',
      progress: 100,
      creator: 'system',
      createdAt: '2026-07-01T00:00:00Z',
      updatedAt: '2026-07-01T00:00:00Z',
    },
    isLoading: false,
    isError: false,
  }),
  usePmAdhocFilterOptions: () => ({ data: undefined }),
  usePmAdhocResults: (...args: unknown[]) => {
    usePmAdhocResultsMock(...args);
    return { data: { rows: [] }, isLoading: false };
  },
}));

vi.mock('@core/hooks/api/usePerformance', () => ({
  useIndicatorCandidates: () => ({ data: [] }),
}));

vi.mock('@core/hooks/api/useKpiExport', () => ({
  useCreateKpiExport: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('@core/hooks/api/useSystemTimezone', () => ({
  useSystemTimezoneValue: () => 'UTC',
}));

vi.mock('@core/hooks/api/useTechnologyDictionary', () => ({
  isKnownTechnology: () => true,
  technologyToDeviceType: () => 'enb',
  useTechnologyDictionary: () => ({ labelForTechnology: () => 'LTE' }),
}));

vi.mock('./DashboardFilterBar', () => ({
  default: () => <div data-testid="filter-bar" />,
}));

vi.mock('./ChartCard', () => ({
  default: () => <div data-testid="chart-card" />,
}));

import TaskDashboardPane from './TaskDashboardPane';

function renderPane() {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <TaskDashboardPane taskId="task-1" />
      </App>
    </IntlProvider>,
  );
}

function seedPageState(input: { submitted: boolean; savedAt: string }) {
  const filter = {
    range: [dayjs('2026-07-01T00:00:00Z'), dayjs('2026-07-02T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
    weekdays: [1, 2],
    hours: [8, 9],
    compare: false,
  };
  const submitted = input.submitted
    ? buildSubmittedTaskDashboardQuery(filter, { systemTimezone: 'UTC' })
    : null;
  usePmPageStateStore.setState({
    pages: {
      [PM_DASHBOARD_PAGE_KEY]: {
        ...buildTaskDashboardStateSnapshot({
          taskId: 'task-1',
          filter,
          dimSelected: [],
          rangeMode: { kind: 'absolute' },
          submitted,
        }),
        savedAt: input.savedAt,
      },
    },
  });
}

function resultTaskIds(): unknown[] {
  return usePmAdhocResultsMock.mock.calls.map((call) => call[0]);
}

describe('TaskDashboardPane page-state restore query behavior', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));
    usePmAdhocResultsMock.mockClear();
    usePmPageStateStore.setState({ pages: {} });
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('triggers results query when a submitted snapshot is restored outside the throttle window', () => {
    seedPageState({
      submitted: true,
      savedAt: '2026-07-30T11:59:40.000Z',
    });

    renderPane();

    expect(resultTaskIds()).toContain('task-1');
  });

  it('delays a recently restored submitted snapshot, then runs one results query after the throttle window', () => {
    seedPageState({
      submitted: true,
      savedAt: '2026-07-30T11:59:59.000Z',
    });

    renderPane();

    expect(resultTaskIds()).not.toContain('task-1');

    act(() => {
      vi.advanceTimersByTime(1_000);
    });

    expect(resultTaskIds()).toContain('task-1');
  });

  it('does not trigger results query when restored state has no submitted query', () => {
    seedPageState({
      submitted: false,
      savedAt: '2026-07-30T11:59:40.000Z',
    });

    renderPane();

    expect(resultTaskIds()).not.toContain('task-1');
  });

  it('reset clears page state and prevents automatic results query afterwards', () => {
    seedPageState({
      submitted: true,
      savedAt: '2026-07-30T11:59:40.000Z',
    });

    renderPane();
    expect(resultTaskIds()).toContain('task-1');
    usePmAdhocResultsMock.mockClear();

    act(() => {
      fireEvent.click(screen.getByText('重置'));
    });

    expect(usePmPageStateStore.getState().getPageState(PM_DASHBOARD_PAGE_KEY)).toBeNull();
    expect(resultTaskIds()).not.toContain('task-1');
  });
});
