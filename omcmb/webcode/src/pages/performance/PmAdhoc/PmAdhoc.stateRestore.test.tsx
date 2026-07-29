import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import zhCN from '@core/i18n/zh-CN';
import { usePmPageStateStore } from '@core/store/pmPageStateStore';
import type { AdhocTask } from '@core/types/pmAdhoc';
import PmAdhocPage from './index';
import {
  buildPmAdhocListStateSnapshot,
  PM_ADHOC_PAGE_KEY,
} from './pmAdhocListState';

const navigateSpy = vi.fn();
const setSearchParamsSpy = vi.fn();
const usePmAdhocListSpy = vi.fn();
const resultPanelSpy = vi.fn();

function makeTask(partial: Partial<AdhocTask> & Pick<AdhocTask, 'id' | 'name' | 'creator' | 'visibility' | 'status'>): AdhocTask {
  return {
    mode: 'oneshot',
    deviceSns: [],
    metricPaths: ['K0001'],
    granularities: ['hourly'],
    windowStart: '2026-07-20T00:00:00Z',
    windowEnd: '2026-07-20T01:00:00Z',
    dimension: 'network',
    technology: 'lte',
    isBuiltin: false,
    expireDays: 60,
    progress: partial.status === 'running' ? 50 : 0,
    createdAt: '2026-07-20T00:00:00Z',
    updatedAt: '2026-07-20T00:00:00Z',
    ...partial,
  };
}

let builtinTasks: AdhocTask[] = [];
let customTasks: AdhocTask[] = [];

vi.mock('react-router-dom', () => ({
  useNavigate: () => navigateSpy,
  useSearchParams: () => [new URLSearchParams(), setSearchParamsSpy],
}));

vi.mock('@core/store/userStore', () => ({
  useUserStore: (selector: (state: { currentUser: { username: string; isSuperAdmin: boolean } }) => unknown) =>
    selector({ currentUser: { username: 'bob', isSuperAdmin: false } }),
}));

vi.mock('@core/hooks/api/useAdhocProgress', () => ({
  useAdhocProgressStream: () => new Map(),
}));

vi.mock('@core/hooks/api/useTechnologyDictionary', () => ({
  useTechnologyDictionary: () => ({
    options: [],
    deviceTypeOptions: [],
    labelForTechnology: (tech?: string | null) => (tech ? tech.toUpperCase() : '—'),
    labelForRadioMode: (radioMode?: string | null) => (radioMode ? radioMode : '—'),
    isLoading: false,
  }),
}));

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  usePmAdhocList: (opts?: { isBuiltin?: boolean }) => {
    usePmAdhocListSpy(opts);
    return {
      data: opts?.isBuiltin ? builtinTasks : customTasks,
      isLoading: false,
    };
  },
  usePmAdhocDetail: () => ({ data: undefined }),
  useCreatePmAdhoc: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdatePmAdhoc: () => ({ mutateAsync: vi.fn(), isPending: false }),
  usePmAdhocRuns: () => ({ data: [], isLoading: false }),
  useCancelPmAdhoc: () => ({ mutateAsync: vi.fn() }),
  useDeletePmAdhoc: () => ({ mutateAsync: vi.fn() }),
  useResumePmAdhoc: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock('./AdhocResultPanel', () => ({
  AdhocResultPanel: ({ taskId }: { taskId: string }) => {
    resultPanelSpy(taskId);
    return <div data-testid="adhoc-result-panel">{taskId}</div>;
  },
}));

vi.mock('./BuiltinMetricEditModal', () => ({
  default: () => null,
}));

vi.mock('./SelectedMetricsTags', () => ({
  default: () => <span>K0001</span>,
}));

function renderPage() {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <PmAdhocPage />
      </App>
    </IntlProvider>,
  );
}

function seedSavedState() {
  usePmPageStateStore.setState({
    pages: {
      [PM_ADHOC_PAGE_KEY]: {
        ...buildPmAdhocListStateSnapshot({
          activeListArea: 'custom',
          builtinPagination: { current: 1, pageSize: 10 },
          customPagination: { current: 2, pageSize: 10 },
          detailTaskId: 'custom-12',
        }),
        savedAt: '2026-07-29T00:00:00.000Z',
      },
    },
  });
}

describe('PmAdhoc list state restore', () => {
  beforeEach(() => {
    sessionStorage.clear();
    usePmPageStateStore.setState({ pages: {} });
    navigateSpy.mockReset();
    setSearchParamsSpy.mockReset();
    usePmAdhocListSpy.mockClear();
    resultPanelSpy.mockClear();
    builtinTasks = [
      makeTask({
        id: 'builtin-1',
        name: '内置任务 1',
        creator: 'system',
        visibility: 'public',
        status: 'succeeded',
        isBuiltin: true,
      }),
    ];
    customTasks = Array.from({ length: 12 }, (_, index) =>
      makeTask({
        id: `custom-${index + 1}`,
        name: `自建任务 ${index + 1}`,
        creator: 'bob',
        visibility: 'private',
        status: 'succeeded',
      }),
    );
  });

  it('restores custom list page and opens detail from the freshly fetched list', async () => {
    seedSavedState();

    renderPage();

    expect(usePmAdhocListSpy).toHaveBeenCalledWith(expect.objectContaining({
      isBuiltin: true,
      refetchOnMount: 'always',
      staleTime: 0,
    }));
    expect(usePmAdhocListSpy).toHaveBeenCalledWith(expect.objectContaining({
      isBuiltin: false,
      refetchOnMount: 'always',
      staleTime: 0,
    }));
    expect(screen.queryByText('自建任务 1')).not.toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '自建聚合任务' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByText('自建任务 11')).toBeInTheDocument();
    expect(screen.getByText('自建任务 12')).toBeInTheDocument();

    await screen.findByText('任务详情：自建任务 12');
    expect(resultPanelSpy).toHaveBeenCalledWith('custom-12');
  });

  it('saves detail drawer state when viewing a task and clears it when the drawer closes', async () => {
    renderPage();

    fireEvent.click(screen.getByRole('tab', { name: '自建聚合任务' }));
    const customRow = screen.getByText('自建任务 1').closest('tr');
    expect(customRow).toBeTruthy();
    fireEvent.click(within(customRow as HTMLElement).getByText('查看详情'));

    await waitFor(() => {
      expect(usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY)?.view?.detailTaskId).toBe('custom-1');
    });

    const closeButton = screen.getByLabelText('Close');
    fireEvent.click(closeButton);

    await waitFor(() => {
      expect(usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY)?.view?.detailTaskId).toBeNull();
    });
  });

  it('reset clears saved list state and returns pagination/detail to defaults', async () => {
    seedSavedState();

    renderPage();
    expect(screen.getByText('自建任务 11')).toBeInTheDocument();
    expect(usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY)).not.toBeNull();

    fireEvent.click(screen.getByRole('button', { name: /重置/ }));

    await waitFor(() => {
      expect(usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY)).toBeNull();
    });
    expect(screen.getByText('自建任务 1')).toBeInTheDocument();
    expect(screen.queryByText('任务详情：自建任务 12')).not.toBeInTheDocument();
  });

  it('does not skip the first tab save after resetting from default state', async () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: /重置/ }));
    await waitFor(() => {
      expect(usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY)).toBeNull();
    });

    fireEvent.click(screen.getByRole('tab', { name: '自建聚合任务' }));

    await waitFor(() => {
      expect(usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY)?.view?.activeListArea).toBe('custom');
    });
  });
});
