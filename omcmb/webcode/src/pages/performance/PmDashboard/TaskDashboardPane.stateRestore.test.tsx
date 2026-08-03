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
let mockTaskGranularities = ['hourly'];
let mockTaskDetailReady = true;
let mockResultsData: Record<string, unknown> = { rows: [] };

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  usePmAdhocDetail: () => ({
    data: mockTaskDetailReady ? {
      id: 'task-1',
      name: '内置-全网-LTE',
      mode: 'continuous',
      deviceSns: [],
      metricPaths: ['M1'],
      granularities: mockTaskGranularities,
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
    } : undefined,
    isLoading: !mockTaskDetailReady,
    isError: false,
  }),
  usePmAdhocFilterOptions: () => ({ data: undefined }),
  usePmAdhocResults: (...args: unknown[]) => {
    usePmAdhocResultsMock(...args);
    return { data: mockResultsData, isLoading: false };
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
  isKnownTechnology: (value: unknown) => Boolean(value),
  technologyToDeviceType: () => 'enb',
  useTechnologyDictionary: () => ({ labelForTechnology: () => 'LTE' }),
}));

vi.mock('./DashboardFilterBar', () => ({
  default: () => <div data-testid="filter-bar" />,
}));

vi.mock('./ChartCard', () => ({
  default: ({ chart }: { chart: unknown }) => (
    <div data-testid="chart-card">{JSON.stringify(chart)}</div>
  ),
}));

import TaskDashboardPane from './TaskDashboardPane';

function paneElement() {
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <TaskDashboardPane taskId="task-1" />
      </App>
    </IntlProvider>
  );
}

function renderPane() {
  return render(paneElement());
}

function seedPageState(input: {
  submitted: boolean;
  savedAt: string;
  rangeMode?: { kind: 'absolute' } | { kind: 'relative'; durationMs: number };
}) {
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
          rangeMode: input.rangeMode ?? { kind: 'absolute' },
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

function resultParams(): Array<Record<string, unknown>> {
  return usePmAdhocResultsMock.mock.calls
    .filter((call) => call[0] === 'task-1')
    .map((call) => call[1] as Record<string, unknown>);
}

function seedDailySubmittedState() {
  const filter = {
    range: [dayjs('2026-08-01T00:00:00Z'), dayjs('2026-08-04T00:00:00Z')] as [dayjs.Dayjs, dayjs.Dayjs],
    weekdays: [0, 1, 2, 3, 4, 5, 6],
    hours: Array.from({ length: 24 }, (_, index) => index),
    compare: false,
  };
  const submitted = buildSubmittedTaskDashboardQuery(filter, {
    systemTimezone: 'UTC',
    granularity: 'daily',
  });
  usePmPageStateStore.setState({
    pages: {
      [PM_DASHBOARD_PAGE_KEY]: {
        ...buildTaskDashboardStateSnapshot({
          taskId: 'task-1',
          filter,
          dimSelected: [],
          activeGran: 'daily',
          rangeMode: { kind: 'absolute' },
          submitted,
        }),
        savedAt: '2026-07-30T11:59:40.000Z',
      },
    },
  });
}

describe('TaskDashboardPane page-state restore query behavior', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-30T12:00:00Z'));
    mockTaskGranularities = ['hourly'];
    mockTaskDetailReady = true;
    mockResultsData = { rows: [] };
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

  it.each([
    {
      granularity: 'hourly',
      expectedStart: '2026-07-29T12:00:00Z',
      expectedEnd: '2026-07-30T12:00:00Z',
    },
    {
      granularity: 'daily',
      expectedStart: '2026-07-23T00:00:00Z',
      expectedEnd: '2026-07-30T00:00:00Z',
    },
  ])(
    'hydrates legacy restored submitted query with default task granularity before first request: $granularity',
    ({ granularity, expectedStart, expectedEnd }) => {
      mockTaskGranularities = [granularity];
      const staleFilter = {
        range: [
          dayjs('2026-07-29T12:23:12Z'),
          dayjs('2026-07-30T12:23:12Z'),
        ] as [dayjs.Dayjs, dayjs.Dayjs],
        weekdays: [1, 2],
        hours: [8, 9],
        compare: false,
      };
      const submitted = buildSubmittedTaskDashboardQuery(staleFilter, { systemTimezone: 'UTC' });
      usePmPageStateStore.setState({
        pages: {
          [PM_DASHBOARD_PAGE_KEY]: {
            ...buildTaskDashboardStateSnapshot({
              taskId: 'task-1',
              filter: staleFilter,
              dimSelected: [],
              activeGran: undefined,
              rangeMode: { kind: 'relative', durationMs: 24 * 60 * 60 * 1000 },
              submitted,
            }),
            savedAt: '2026-07-30T11:59:40.000Z',
            view: { activeGran: null },
          },
        },
      });

      renderPane();

      const firstBusinessParams = resultParams()[0];
      expect(firstBusinessParams?.startTime).toBe(expectedStart);
      expect(firstBusinessParams?.endTime).toBe(expectedEnd);
      expect(String(firstBusinessParams?.startTime)).not.toContain(':23:12');
      expect(String(firstBusinessParams?.endTime)).not.toContain(':23:12');
    },
  );

  it('does not let the restore throttle timer send a legacy submitted query before delayed task granularity is known', async () => {
    mockTaskGranularities = ['hourly'];
    mockTaskDetailReady = false;
    const staleFilter = {
      range: [
        dayjs('2026-07-29T12:23:12Z'),
        dayjs('2026-07-30T12:23:12Z'),
      ] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1, 2],
      hours: [8, 9],
      compare: false,
    };
    const submitted = buildSubmittedTaskDashboardQuery(staleFilter, { systemTimezone: 'UTC' });
    usePmPageStateStore.setState({
      pages: {
        [PM_DASHBOARD_PAGE_KEY]: {
          ...buildTaskDashboardStateSnapshot({
            taskId: 'task-1',
            filter: staleFilter,
            dimSelected: [],
            activeGran: undefined,
            rangeMode: { kind: 'relative', durationMs: 24 * 60 * 60 * 1000 },
            submitted,
          }),
          savedAt: '2026-07-30T11:59:59.000Z',
          view: { activeGran: null },
        },
      },
    });

    const view = renderPane();
    expect(resultTaskIds()).not.toContain('task-1');

    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    expect(resultTaskIds()).not.toContain('task-1');

    mockTaskDetailReady = true;
    await act(async () => {
      view.rerender(paneElement());
    });
    await act(async () => {});

    expect(resultParams()[0]?.startTime).toBe('2026-07-29T12:00:00Z');
    expect(resultParams()[0]?.endTime).toBe('2026-07-30T12:00:00Z');
    expect(String(resultParams()[0]?.startTime)).not.toContain(':23:12');
    expect(String(resultParams()[0]?.endTime)).not.toContain(':23:12');
  });

  it('does not send a legacy absolute submitted query before delayed task granularity can realign it', async () => {
    mockTaskGranularities = ['hourly'];
    mockTaskDetailReady = false;
    const staleFilter = {
      range: [
        dayjs('2026-07-29T12:23:12Z'),
        dayjs('2026-07-30T12:23:12Z'),
      ] as [dayjs.Dayjs, dayjs.Dayjs],
      weekdays: [1, 2],
      hours: [8, 9],
      compare: false,
    };
    const submitted = buildSubmittedTaskDashboardQuery(staleFilter, { systemTimezone: 'UTC' });
    usePmPageStateStore.setState({
      pages: {
        [PM_DASHBOARD_PAGE_KEY]: {
          ...buildTaskDashboardStateSnapshot({
            taskId: 'task-1',
            filter: staleFilter,
            dimSelected: [],
            activeGran: undefined,
            rangeMode: { kind: 'absolute' },
            submitted,
          }),
          savedAt: '2026-07-30T11:59:59.000Z',
          view: { activeGran: null },
        },
      },
    });

    const view = renderPane();
    expect(resultTaskIds()).not.toContain('task-1');

    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    expect(resultTaskIds()).not.toContain('task-1');

    mockTaskDetailReady = true;
    await act(async () => {
      view.rerender(paneElement());
    });
    await act(async () => {});

    expect(resultParams()[0]?.startTime).toBe('2026-07-29T13:00:00Z');
    expect(resultParams()[0]?.endTime).toBe('2026-07-30T12:00:00Z');
    expect(String(resultParams()[0]?.startTime)).not.toContain(':23:12');
    expect(String(resultParams()[0]?.endTime)).not.toContain(':23:12');
  });
});

describe('TaskDashboardPane chart result source', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-08-04T12:00:00Z'));
    mockTaskGranularities = ['daily'];
    mockTaskDetailReady = true;
    mockResultsData = { rows: [] };
    usePmAdhocResultsMock.mockClear();
    usePmPageStateStore.setState({ pages: {} });
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('does not plot daily progressRows when formal rows are empty', () => {
    seedDailySubmittedState();
    mockResultsData = {
      rows: [],
      progressRows: [{
        id: 'progress-1',
        taskId: 'task-1',
        deviceOui: '',
        deviceSn: 'AGGREGATED',
        metricPath: 'M1',
        metricType: 'counter',
        metricValue: 99,
        granularity: 'daily',
        time: '2026-08-03T08:00:00+08:00',
        startTime: '2026-08-03T08:00:00+08:00',
        endTime: '2026-08-04T08:00:00+08:00',
        partial: true,
        periodComplete: false,
      }],
    };

    renderPane();

    const chart = screen.getByTestId('chart-card').textContent ?? '';
    expect(chart).toContain('"values":["-","-","-"]');
    expect(chart).not.toContain('2026-08-03T08:00:00+08:00');
    expect(chart).not.toContain('"values":[99]');
  });

  it('plots formal daily rows normally', () => {
    seedDailySubmittedState();
    mockResultsData = {
      rows: [{
        id: 'row-1',
        taskId: 'task-1',
        deviceOui: '',
        deviceSn: 'AGGREGATED',
        metricPath: 'M1',
        metricType: 'counter',
        metricValue: 42,
        granularity: 'daily',
        time: '2026-08-02T00:00:00+08:00',
        startTime: '2026-08-02T00:00:00+08:00',
        endTime: '2026-08-03T00:00:00+08:00',
        partial: false,
        periodComplete: true,
      }],
      progressRows: [],
    };

    renderPane();

    const chart = screen.getByTestId('chart-card').textContent ?? '';
    expect(chart).toContain('2026-08-02T00:00:00+08:00');
    expect(chart).toContain('2026-08-03T00:00:00+08:00');
    expect(chart).toContain('"values":[42,"-","-"]');
  });
});
