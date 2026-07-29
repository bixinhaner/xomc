import { fireEvent, render, screen, within } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { describe, expect, it, vi } from 'vitest';

import zhCN from '@core/i18n/zh-CN';
import type { AdhocTask } from '@core/types/pmAdhoc';
import PmAdhocPage from './index';

const navigateSpy = vi.fn();
const setSearchParamsSpy = vi.fn();

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

const customTasks: AdhocTask[] = [
  makeTask({
    id: 'public-canceled',
    name: '公开已取消',
    creator: 'alice',
    visibility: 'public',
    status: 'canceled',
  }),
  makeTask({
    id: 'public-running',
    name: '公开运行中',
    creator: 'alice',
    visibility: 'public',
    status: 'running',
  }),
  makeTask({
    id: 'private-mine',
    name: '我的私有',
    creator: 'bob',
    visibility: 'private',
    status: 'running',
  }),
];

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
    options: [
      { label: 'eNB(LTE)', value: 'lte', sort: 1 },
      { label: 'gNB(NR)', value: 'nr', sort: 2 },
      { label: 'GSM', value: 'gsm', sort: 3 },
    ],
    deviceTypeOptions: [
      { label: 'eNB(LTE)', value: 'ENB', sort: 1, technology: 'lte' },
      { label: 'gNB(NR)', value: 'GNB', sort: 2, technology: 'nr' },
      { label: 'GSM', value: 'GSM', sort: 3, technology: 'gsm' },
    ],
    labelForTechnology: (tech?: string | null) => (tech ? tech.toUpperCase() : '—'),
    labelForRadioMode: (radioMode?: string | null) => (radioMode ? radioMode : '—'),
    isLoading: false,
  }),
}));

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  usePmAdhocList: (opts?: { isBuiltin?: boolean }) => ({
    data: opts?.isBuiltin ? [] : customTasks,
    isLoading: false,
  }),
  usePmAdhocDetail: () => ({ data: undefined }),
  useCreatePmAdhoc: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdatePmAdhoc: () => ({ mutateAsync: vi.fn(), isPending: false }),
  usePmAdhocRuns: () => ({ data: [], isLoading: false }),
  useCancelPmAdhoc: () => ({ mutateAsync: vi.fn() }),
  useDeletePmAdhoc: () => ({ mutateAsync: vi.fn() }),
  useResumePmAdhoc: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock('./AdhocResultPanel', () => ({
  AdhocResultPanel: () => <div data-testid="adhoc-result-panel" />,
}));

vi.mock('./BuiltinMetricEditModal', () => ({
  default: () => null,
}));

vi.mock('./SelectedMetricsTags', () => ({
  default: () => <span>K0001</span>,
}));

function renderPage() {
  render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <PmAdhocPage />
      </App>
    </IntlProvider>,
  );
}

function rowFor(name: string): HTMLElement {
  const cell = screen.getByText(name);
  const row = cell.closest('tr');
  if (!row) throw new Error(`row not found for ${name}`);
  return row;
}

describe('PmAdhoc visibility list actions', () => {
  it('shows visibility and applies public/private action rules in the custom task list', () => {
    renderPage();

    const publicCanceled = within(rowFor('公开已取消'));
    expect(publicCanceled.getByText('公开')).toBeInTheDocument();
    expect(publicCanceled.getByText('alice')).toBeInTheDocument();
    expect(publicCanceled.getByText('编辑')).toBeInTheDocument();
    expect(publicCanceled.getByText('启用')).toBeInTheDocument();
    expect(publicCanceled.getByText('删除')).toBeInTheDocument();
    expect(publicCanceled.queryByText('取消')).not.toBeInTheDocument();

    const publicRunning = within(rowFor('公开运行中'));
    expect(publicRunning.getByText('公开')).toBeInTheDocument();
    expect(publicRunning.getByText('编辑')).toBeInTheDocument();
    expect(publicRunning.queryByText('取消')).not.toBeInTheDocument();

    const privateMine = within(rowFor('我的私有'));
    expect(privateMine.getByText('私有')).toBeInTheDocument();
    expect(privateMine.getByText('bob')).toBeInTheDocument();
    expect(privateMine.getByText('编辑')).toBeInTheDocument();
    expect(privateMine.getByText('取消')).toBeInTheDocument();
  });
});
