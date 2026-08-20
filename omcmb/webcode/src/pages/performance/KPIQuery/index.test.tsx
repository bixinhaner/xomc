import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';
import { usePmPageStateStore } from '@core/store/pmPageStateStore';
import type { QueryTemplate } from '@core/types/pmQuery';
import KPIQuery from './index';
import { buildKpiQueryStateSnapshot, PM_KPI_QUERY_PAGE_KEY } from './kpiQueryState';

const refetchAggSpy = vi.fn();
const createTemplateSpy = vi.fn();
const updateTemplateSpy = vi.fn();
const createExportSpy = vi.fn();
const metricPickerRenderSpy = vi.hoisted(() => vi.fn());
const aggregatedQuerySpy = vi.hoisted(() => vi.fn());
const queryTemplateMockCache = vi.hoisted(() => new Map<string, unknown>());
const aggregatedQueryState = vi.hoisted(() => ({
  current: {
    data: [] as unknown[],
    total: 0,
    truncated: false,
  },
}));
const technologyDictionaryState = vi.hoisted(() => {
  const defaultDictionary = {
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
  };
  return { current: defaultDictionary, defaultDictionary };
});

const validTemplate: QueryTemplate = {
  id: 'tpl-valid',
  name: '正常模板',
  visibility: 'public',
  creatorId: 'user-1',
  description: '正常模板描述',
  payload: {
    deviceSns: ['SN-OK'],
    metricPaths: ['K-1'],
    granularity: '15min',
    timeRangePreset: 'last_1h',
    deviceType: 'ENB',
  },
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const overLimitTemplate: QueryTemplate = {
  id: 'tpl-over-limit',
  name: '老模板-超限设备',
  visibility: 'public',
  creatorId: 'user-1',
  description: '',
  payload: {
    deviceSns: Array.from({ length: 51 }, (_, i) => `SN-${i + 1}`),
    metricPaths: ['K-1'],
    granularity: '15min',
    timeRangePreset: 'last_1h',
    deviceType: 'ENB',
  },
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const noMetricTemplate: QueryTemplate = {
  id: 'tpl-no-metric',
  name: '无指标模板',
  visibility: 'private',
  creatorId: 'user-1',
  description: '',
  payload: {
    deviceSns: ['SN-OK'],
    metricPaths: [],
    granularity: '15min',
    timeRangePreset: 'last_1h',
    deviceType: 'ENB',
  },
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

vi.mock('@core/hooks/api/usePmQuery', () => ({
  useQueryTemplates: (params?: { visibility?: string; search?: string; page?: number; pageSize?: number }) => {
    const cacheKey = JSON.stringify(params ?? {});
    const cached = queryTemplateMockCache.get(cacheKey);
    if (cached) return cached;
    const allTemplates = [validTemplate, overLimitTemplate, noMetricTemplate];
    const filtered = allTemplates.filter((template) => (
      (!params?.visibility || template.visibility === params.visibility)
      && (!params?.search || template.name.includes(params.search))
    ));
    const page = params?.page ?? 1;
    const pageSize = params?.pageSize ?? 50;
    const response = {
      data: {
        items: filtered.slice((page - 1) * pageSize, page * pageSize),
        total: filtered.length,
      },
      isLoading: false,
      refetch: vi.fn(),
    };
    queryTemplateMockCache.set(cacheKey, response);
    return response;
  },
  useCreateQueryTemplate: () => ({ mutateAsync: createTemplateSpy, isPending: false }),
  useUpdateQueryTemplate: () => ({ mutateAsync: updateTemplateSpy, isPending: false }),
  useDeleteQueryTemplate: () => ({ mutateAsync: vi.fn() }),
  useAggregatedMetricsByDevices: (
    baseParams: Record<string, unknown>,
    deviceSns: string[],
    enabled: boolean,
  ) => {
    aggregatedQuerySpy(baseParams, deviceSns, enabled);
    return {
      data: aggregatedQueryState.current.data,
      total: aggregatedQueryState.current.total,
      truncated: aggregatedQueryState.current.truncated,
      isLoading: false,
      isFetching: false,
      errors: [],
      refetch: refetchAggSpy,
    };
  },
  useMetricObjectsByDevices: () => ({ byDevice: {} }),
}));

vi.mock('@core/store/userStore', () => ({
  useUserStore: (selector: (state: { currentUser: { id: string; isSuperAdmin: boolean } }) => unknown) =>
    selector({ currentUser: { id: 'user-1', isSuperAdmin: false } }),
}));

vi.mock('@core/hooks/api/useSystemTimezone', () => ({
  useSystemTimezoneValue: () => 'Asia/Shanghai',
}));

vi.mock('@core/hooks/api/useKpiExport', () => ({
  useCreateKpiExport: () => ({ mutate: createExportSpy, isPending: false }),
}));

vi.mock('@core/hooks/api/useTechnologyDictionary', () => ({
  useTechnologyDictionary: () => technologyDictionaryState.current,
  deviceTypeToTechnology: (deviceType: string) =>
    ({ ENB: 'lte', GNB: 'nr', GSM: 'gsm' })[deviceType] ?? 'lte',
}));

vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({
    colorBgContainer: '#fff',
    colorBgTextHover: '#f5f5f5',
    colorBorderSecondary: '#eee',
    colorPrimary: '#1677ff',
    colorTextSecondary: '#999',
    borderRadiusLG: 6,
  }),
}));

vi.mock('../PmDashboard/CellDrilldownSelector', () => ({
  default: () => <div data-testid="cell-drilldown-selector" />,
}));

vi.mock('./components/DevicePickerModal', () => ({
  default: (props: { open: boolean; onConfirm: (deviceSns: string[]) => void }) =>
    props.open ? (
      <button type="button" onClick={() => props.onConfirm(['SN-NEW'])}>
        模拟选择设备
      </button>
    ) : null,
}));

vi.mock('@/components/MetricPickerModal', () => ({
  default: (props: Record<string, unknown>) => {
    metricPickerRenderSpy(props);
    return null;
  },
}));

vi.mock('./components/PivotTable', () => ({
  default: (props: { rows: unknown[]; loading: boolean }) => (
    <div
      data-testid="pivot-table"
      data-loading={String(props.loading)}
      data-row-count={String(props.rows.length)}
    />
  ),
}));

vi.mock('./templateMetricResolver', () => ({
  resolveTemplateMetricPaths: async (_deviceType: string, paths: string[]) => ({
    paths,
    labels: paths.includes('K-1') ? { 'K-1': '小区可用率' } : {},
    ambiguous: [],
  }),
}));

function renderPage() {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <KPIQuery />
      </App>
    </IntlProvider>,
  );
}

function resetUpdateTemplateMock() {
  updateTemplateSpy.mockReset();
  updateTemplateSpy.mockImplementation(async ({ id, input }) => ({
    id,
    creatorId: 'user-1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-02T00:00:00Z',
    ...input,
  }));
}

async function selectOverLimitTemplate() {
  fireEvent.click(screen.getByText('老模板-超限设备'));
  await waitFor(() => {
    expect(screen.getByDisplayValue(/已选 51 个/)).toBeTruthy();
  });
}

async function findModalByTitle(title: string) {
  const titleNode = await screen.findByText(title);
  const modal = titleNode.closest('.ant-modal');
  expect(modal).toBeTruthy();
  return modal as HTMLElement;
}

async function findDrawerByTitle(title: string) {
  const titleNode = await screen.findByText(title);
  const drawer = titleNode.closest('.ant-drawer');
  expect(drawer).toBeTruthy();
  return drawer as HTMLElement;
}

function templateListItem(name: string) {
  const item = screen.getByText(name).closest('[data-template-id]');
  expect(item).toBeTruthy();
  return item as HTMLElement;
}

afterEach(() => {
  vi.useRealTimers();
});

describe('KPIQuery 顶部 tab 现场保持', () => {
  beforeEach(() => {
    queryTemplateMockCache.clear();
    usePmPageStateStore.setState({ pages: {} });
    sessionStorage.clear();
    refetchAggSpy.mockClear();
    aggregatedQuerySpy.mockClear();
    aggregatedQueryState.current = { data: [], total: 0, truncated: false };
    createTemplateSpy.mockReset();
    resetUpdateTemplateMock();
    createExportSpy.mockReset();
    technologyDictionaryState.current = technologyDictionaryState.defaultDictionary;
  });

  it('切回已查询页面时按已提交条件和分页重新启用聚合查询', async () => {
    vi.useFakeTimers();
    usePmPageStateStore.setState({
      pages: {
        [PM_KPI_QUERY_PAGE_KEY]: {
          ...buildKpiQueryStateSnapshot({
            payload: {
              deviceType: 'ENB',
              deviceSns: ['SN-EDITED'],
              metricPaths: ['K-EDITED'],
              granularity: 'daily',
              timeRangePreset: 'last_7d',
            },
            customRange: null,
            timeRangeDirty: true,
            cellSel: { 'SN-EDITED': ['Cell=9'] },
            submitted: {
              payload: {
                deviceType: 'ENB',
                deviceSns: ['SN-SUBMITTED'],
                metricPaths: ['K-SUBMITTED'],
                granularity: 'hourly',
                timeRangePreset: 'last_1h',
              },
              range: {
                start: '2026-07-01T00:00:00+08:00',
                end: '2026-07-01T01:00:00+08:00',
              },
              cellSel: {},
            },
            pivotPage: 3,
            pivotPageSize: 50,
            templateTab: 'private',
            activeTemplateId: 'tpl-valid',
            sidebarCollapsed: false,
            resultsMaximized: false,
          }),
          savedAt: '2026-07-29T00:00:00.000Z',
        },
      },
    });

    renderPage();

    expect(aggregatedQuerySpy.mock.calls.some(([, , enabled]) => enabled === true)).toBe(false);
    expect(screen.getByTestId('pivot-table')).toHaveAttribute('data-loading', 'true');
    expect(screen.getByTestId('pivot-table')).toHaveAttribute('data-row-count', '0');

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    expect(aggregatedQuerySpy.mock.calls.some(([params, deviceSns, enabled]) =>
      enabled === true &&
      deviceSns[0] === 'SN-SUBMITTED' &&
      (params.metricPaths as string[] | undefined)?.[0] === 'K-SUBMITTED' &&
      params.offset === 100,
    )).toBe(true);
  });

  it('只恢复未提交条件时不会自动启用聚合查询', async () => {
    usePmPageStateStore.setState({
      pages: {
        [PM_KPI_QUERY_PAGE_KEY]: {
          ...buildKpiQueryStateSnapshot({
            payload: {
              deviceType: 'ENB',
              deviceSns: ['SN-EDITED'],
              metricPaths: ['K-EDITED'],
              granularity: 'daily',
              timeRangePreset: 'last_7d',
            },
            customRange: null,
            timeRangeDirty: true,
            cellSel: {},
            submitted: null,
            pivotPage: 2,
            pivotPageSize: 100,
            templateTab: 'public',
            sidebarCollapsed: false,
            resultsMaximized: false,
          }),
          savedAt: '2026-07-29T00:00:00.000Z',
        },
      },
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByDisplayValue('已选 1 个：SN-EDITED')).toBeTruthy();
    });
    expect(aggregatedQuerySpy.mock.calls.some(([, , enabled]) => enabled === true)).toBe(false);
  });

  it('点击重置会清理条件、分页和保存状态', async () => {
    usePmPageStateStore.setState({
      pages: {
        [PM_KPI_QUERY_PAGE_KEY]: {
          ...buildKpiQueryStateSnapshot({
            payload: {
              deviceType: 'ENB',
              deviceSns: ['SN-EDITED'],
              metricPaths: ['K-EDITED'],
              granularity: 'daily',
              timeRangePreset: 'last_7d',
            },
            customRange: null,
            timeRangeDirty: true,
            cellSel: {},
            submitted: {
              payload: {
                deviceType: 'ENB',
                deviceSns: ['SN-SUBMITTED'],
                metricPaths: ['K-SUBMITTED'],
                granularity: 'hourly',
                timeRangePreset: 'last_1h',
              },
              range: {
                start: '2026-07-01T00:00:00+08:00',
                end: '2026-07-01T01:00:00+08:00',
              },
              cellSel: {},
            },
            pivotPage: 3,
            pivotPageSize: 100,
            templateTab: 'private',
            sidebarCollapsed: true,
            resultsMaximized: false,
          }),
          savedAt: '2026-07-29T00:00:00.000Z',
        },
      },
    });

    renderPage();
    fireEvent.click(screen.getByRole('button', { name: /重\s*置/ }));

    await waitFor(() => {
      expect(usePmPageStateStore.getState().getPageState(PM_KPI_QUERY_PAGE_KEY)).toBeNull();
    });
    expect(screen.getByPlaceholderText('点击右侧按钮选择设备')).toHaveValue('');
    expect(screen.getByPlaceholderText('点击右侧按钮选择指标')).toHaveValue('');
  });

  it('切回已最大化页面时恢复查询结果最大化状态', () => {
    usePmPageStateStore.setState({
      pages: {
        [PM_KPI_QUERY_PAGE_KEY]: {
          ...buildKpiQueryStateSnapshot({
            payload: {
              deviceType: 'ENB',
              deviceSns: ['SN-EDITED'],
              metricPaths: ['K-EDITED'],
              granularity: '15min',
              timeRangePreset: 'last_1h',
            },
            customRange: null,
            timeRangeDirty: false,
            cellSel: {},
            submitted: null,
            pivotPage: 1,
            pivotPageSize: 50,
            templateTab: 'public',
            sidebarCollapsed: false,
            resultsMaximized: true,
          }),
          savedAt: '2026-07-29T00:00:00.000Z',
        },
      },
    });

    renderPage();

    expect(screen.queryByText('查询条件')).toBeNull();
    expect(screen.getByTestId('pivot-table')).toBeTruthy();
    expect(screen.getByRole('button', { name: '还原查询结果' })).toBeTruthy();
  });

  it('查询结果支持最大化和还原，不卸载结果表格和截断提示', () => {
    aggregatedQueryState.current = {
      data: Array.from({ length: 50 }, (_, index) => ({ time: `2026-01-01T00:${index}:00Z` })),
      total: 120,
      truncated: true,
    };
    renderPage();

    expect(screen.getByText('查询条件')).toBeTruthy();
    expect(screen.getByTestId('pivot-table')).toBeTruthy();
    expect(screen.getByText('结果已截断：仅显示最新 50 行，请缩小时间范围、减少指标数或使用导出')).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: '最大化查询结果' }));

    expect(screen.queryByText('查询条件')).toBeNull();
    expect(screen.getByTestId('pivot-table')).toBeTruthy();
    expect(screen.getByText('结果已截断：仅显示最新 50 行，请缩小时间范围、减少指标数或使用导出')).toBeTruthy();
    expect(screen.getByRole('button', { name: '还原查询结果' })).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: '还原查询结果' }));

    expect(screen.getByText('查询条件')).toBeTruthy();
    expect(screen.getByTestId('pivot-table')).toBeTruthy();
    expect(screen.getByRole('button', { name: '最大化查询结果' })).toBeTruthy();
  });
});

describe('KPIQuery 模板数量限制', () => {
  beforeEach(() => {
    queryTemplateMockCache.clear();
    usePmPageStateStore.setState({ pages: {} });
    sessionStorage.clear();
    refetchAggSpy.mockClear();
    aggregatedQuerySpy.mockClear();
    createTemplateSpy.mockReset();
    resetUpdateTemplateMock();
    createExportSpy.mockReset();
    technologyDictionaryState.current = technologyDictionaryState.defaultDictionary;
  });

  it('老模板仍可展示，但超 50 个设备时点击查询不会执行聚合查询', async () => {
    renderPage();
    await selectOverLimitTemplate();

    fireEvent.click(screen.getByRole('button', { name: /查询$/ }));

    expect(refetchAggSpy).not.toHaveBeenCalled();
  });

  it('老模板仍可展示，但超 50 个设备时保存模板不会调用创建接口', async () => {
    renderPage();
    await selectOverLimitTemplate();

    fireEvent.click(screen.getByRole('button', { name: /存为模板/ }));
    fireEvent.change(screen.getByPlaceholderText('例如：eNB 基础 KPI'), {
      target: { value: '复制超限模板' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(createTemplateSpy).not.toHaveBeenCalled();
    });
  });

  it('未形成查询快照时，当前老模板超 50 个设备仍可点击导出并被提示拦截', async () => {
    renderPage();
    await selectOverLimitTemplate();

    const exportButton = screen.getByRole('button', { name: /导出 CSV/ });
    expect(exportButton).not.toBeDisabled();

    fireEvent.click(exportButton);

    expect(createExportSpy).not.toHaveBeenCalled();
  });

  it('已有查询快照时，当前老模板超 50 个设备不影响按已提交条件导出', async () => {
    renderPage();

    fireEvent.click(screen.getByText('正常模板'));
    await waitFor(() => {
      expect(screen.getByDisplayValue('已选 1 个：SN-OK')).toBeTruthy();
    });
    fireEvent.click(screen.getByRole('button', { name: /查询$/ }));

    fireEvent.click(screen.getByText('老模板-超限设备'));
    await waitFor(() => {
      expect(screen.getByDisplayValue(/已选 51 个/)).toBeTruthy();
    });
    fireEvent.click(screen.getByRole('button', { name: /导出 CSV/ }));

    expect(createExportSpy).toHaveBeenCalledTimes(1);
    expect(createExportSpy.mock.calls[0][0].params.device_sns).toEqual(['SN-OK']);
    expect(createExportSpy.mock.calls[0][0].params.metric_paths).toEqual(['K-1']);
  });

  it('字典禁用 ENB 时，主查询和模板弹窗切到首个可用设备类型', async () => {
    technologyDictionaryState.current = {
      ...technologyDictionaryState.defaultDictionary,
      options: [
        { label: 'gNB(NR)', value: 'nr', sort: 2 },
        { label: 'GSM', value: 'gsm', sort: 3 },
      ],
      deviceTypeOptions: [
        { label: 'gNB(NR)', value: 'GNB', sort: 2, technology: 'nr' },
        { label: 'GSM', value: 'GSM', sort: 3, technology: 'gsm' },
      ],
    };

    renderPage();

    await waitFor(() => {
      expect(screen.getAllByText('gNB(NR)').length).toBeGreaterThan(0);
    });

    fireEvent.click(screen.getByRole('button', { name: '新建查询模板' }));
    const dialog = await findModalByTitle('新建查询模板');
    expect(within(dialog).getAllByText('gNB(NR)').length).toBeGreaterThan(0);
  });
});

describe('KPIQuery 模板弹窗初始值', () => {
  beforeEach(() => {
    queryTemplateMockCache.clear();
    usePmPageStateStore.setState({ pages: {} });
    sessionStorage.clear();
    refetchAggSpy.mockClear();
    aggregatedQuerySpy.mockClear();
    createTemplateSpy.mockReset();
    resetUpdateTemplateMock();
    createExportSpy.mockReset();
    metricPickerRenderSpy.mockClear();
    technologyDictionaryState.current = technologyDictionaryState.defaultDictionary;
  });

  it('侧栏新建模板每次打开都使用干净初始值', async () => {
    renderPage();

    fireEvent.click(screen.getByText('正常模板'));
    await waitFor(() => {
      expect(screen.getByDisplayValue('已选 1 个：SN-OK')).toBeTruthy();
    });

    fireEvent.click(screen.getByRole('button', { name: '新建查询模板' }));

    const dialog = await findModalByTitle('新建查询模板');
    expect(within(dialog).getByPlaceholderText('例如：eNB 基础 KPI')).toHaveValue('');
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择设备')).toHaveValue('');
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择指标')).toHaveValue('');
    expect(within(dialog).getByText('近 3 小时')).toBeTruthy();
    expect(within(dialog).queryByDisplayValue('已选 1 个：SN-OK')).toBeNull();
    expect(within(dialog).queryByDisplayValue('已选 1 个：K-1 小区可用率')).toBeNull();
  });

  it('侧栏新建模板再次打开不会残留上一次输入', async () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: '新建查询模板' }));

    const firstDialog = await findModalByTitle('新建查询模板');
    fireEvent.change(within(firstDialog).getByPlaceholderText('例如：eNB 基础 KPI'), {
      target: { value: '上一次创建的模板' },
    });
    const firstDescription = firstDialog.querySelector('textarea');
    expect(firstDescription).toBeTruthy();
    fireEvent.change(firstDescription as HTMLTextAreaElement, {
      target: { value: '上一次创建的描述' },
    });
    fireEvent.mouseDown(within(firstDialog).getByText('近 3 小时'));
    fireEvent.click((await screen.findAllByText('近 1 小时')).at(-1) as HTMLElement);
    fireEvent.click(within(firstDialog).getByRole('button', { name: /取\s*消/ }));

    fireEvent.click(screen.getByRole('button', { name: '新建查询模板' }));

    const secondDialog = await findModalByTitle('新建查询模板');
    const secondDescription = secondDialog.querySelector('textarea');
    expect(secondDescription).toBeTruthy();
    expect(within(secondDialog).getByPlaceholderText('例如：eNB 基础 KPI')).toHaveValue('');
    expect(secondDescription).toHaveValue('');
    expect(within(secondDialog).getAllByText('近 3 小时').length).toBeGreaterThan(0);
    expect(within(secondDialog).queryByDisplayValue('上一次创建的模板')).toBeNull();
    expect(within(secondDialog).queryByDisplayValue('上一次创建的描述')).toBeNull();
  });

  it('查询区存为模板仍使用当前查询条件作为初始值', async () => {
    renderPage();

    fireEvent.click(screen.getByText('正常模板'));
    await waitFor(() => {
      expect(screen.getByDisplayValue('已选 1 个：SN-OK')).toBeTruthy();
    });

    fireEvent.click(screen.getByRole('button', { name: /存为模板/ }));

    const dialog = await findModalByTitle('新建查询模板');
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择设备')).toHaveValue('已选 1 个：SN-OK');
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择指标')).toHaveValue('已选 1 个：K-1 小区可用率');
    expect(within(dialog).getByText('近 1 小时')).toBeTruthy();
  });

  it('新建查询模板不再混入定时报表配置', async () => {
    renderPage();

    expect(screen.getByRole('button', { name: '定时报表' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '新建查询模板' }));
    const dialog = await findModalByTitle('新建查询模板');

    expect(within(dialog).queryByText('KPI 定时报表')).toBeNull();
    expect(within(dialog).queryByRole('switch')).toBeNull();
    expect(within(dialog).getByText('查询配置')).toBeTruthy();
  }, 15_000);

  it('模板总量超过当前页时显示接口真实总数', () => {
    const pageItems = Array.from({ length: 2 }, (_, index) => ({
      ...validTemplate,
      id: `tpl-page-${index + 1}`,
      name: `分页模板-${index + 1}`,
    }));
    queryTemplateMockCache.set(
      JSON.stringify({ visibility: 'public', page: 1, pageSize: 20 }),
      {
        data: { items: pageItems, total: 25 },
        isLoading: false,
        refetch: vi.fn(),
      },
    );

    renderPage();

    expect(screen.getByRole('tab', { name: /公共 \(25\)/ })).toBeTruthy();
    expect(screen.getAllByRole('button', { name: '模板详情' })).toHaveLength(2);
  }, 15_000);

  it('选择模板后从查询操作栏独立配置并保存 KPI 定时报表', async () => {
    renderPage();

    const item = templateListItem('正常模板');
    expect(within(item).queryByRole('button', { name: '定时报表' })).toBeNull();
    fireEvent.click(item);
    await waitFor(() => {
      expect(screen.getByRole('button', { name: '定时报表' })).toBeEnabled();
    });
    fireEvent.click(screen.getByRole('button', { name: '定时报表' }));

    const drawer = await findDrawerByTitle('KPI 定时报表');
    expect(drawer).toHaveTextContent('正常模板');
    expect(within(drawer).getByText('设备 1 台')).toBeTruthy();
    expect(within(drawer).getByText('指标 1 个')).toBeTruthy();
    fireEvent.click(within(drawer).getByRole('switch', { name: '启用定时报表' }));
    fireEvent.change(within(drawer).getByPlaceholderText('多个邮箱用分号、逗号或换行分隔'), {
      target: { value: 'ops@example.com' },
    });
    fireEvent.click(within(drawer).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(updateTemplateSpy).toHaveBeenCalledWith(expect.objectContaining({
        id: 'tpl-valid',
        input: expect.objectContaining({
          payload: expect.objectContaining({
            regularReport: {
              enabled: true,
              sendTime: '08:00',
              periods: ['daily'],
              emailEnabled: true,
              recipients: ['ops@example.com'],
            },
          }),
        }),
      }));
    });
  }, 15_000);

  it('没有指标的模板不能启用定时报表', async () => {
    renderPage();

    fireEvent.click(screen.getByRole('tab', { name: /私有/ }));
    const item = templateListItem('无指标模板');
    fireEvent.click(item);
    await waitFor(() => {
      expect(screen.getByRole('button', { name: '定时报表' })).toBeEnabled();
    });
    fireEvent.click(screen.getByRole('button', { name: '定时报表' }));
    const drawer = await findDrawerByTitle('KPI 定时报表');
    fireEvent.click(within(drawer).getByRole('switch', { name: '启用定时报表' }));
    fireEvent.change(within(drawer).getByPlaceholderText('多个邮箱用分号、逗号或换行分隔'), {
      target: { value: 'ops@example.com' },
    });
    fireEvent.click(within(drawer).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(updateTemplateSpy).not.toHaveBeenCalled();
      expect(within(drawer).getByText('指标 0 个')).toBeTruthy();
    });
  }, 15_000);

  it('编辑模板弹窗仍回填待编辑模板数据', async () => {
    renderPage();

    fireEvent.click(screen.getAllByRole('button', { name: '编辑' })[0]);

    const dialog = await findModalByTitle('编辑查询模板');
    expect(within(dialog).getByDisplayValue('正常模板')).toBeTruthy();
    expect(within(dialog).getByDisplayValue('正常模板描述')).toBeTruthy();
    expect(within(dialog).getByLabelText('公共（所有人可见）')).toBeChecked();
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择设备')).toHaveValue('已选 1 个：SN-OK');
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择指标')).toHaveValue('已选 1 个：K-1 小区可用率');
    expect(within(dialog).getByText('近 1 小时')).toBeTruthy();
  }, 10000);

  it('指标摘要显示 ID 和名称，并把 metricLabels 传给指标选择弹窗', async () => {
    renderPage();

    fireEvent.click(screen.getByText('正常模板'));
    await waitFor(() => {
      expect(screen.getByDisplayValue('已选 1 个：K-1 小区可用率')).toBeTruthy();
    });

    await waitFor(() => {
      expect(metricPickerRenderSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          initialLabels: expect.objectContaining({ 'K-1': '小区可用率' }),
        }),
      );
    });
  });

  it('保存模板和导出 payload 仍只携带指标 ID', async () => {
    renderPage();

    fireEvent.click(screen.getByText('正常模板'));
    await waitFor(() => {
      expect(screen.getByDisplayValue('已选 1 个：K-1 小区可用率')).toBeTruthy();
    });

    fireEvent.click(screen.getByRole('button', { name: /存为模板/ }));
    const dialog = await findModalByTitle('新建查询模板');
    fireEvent.change(within(dialog).getByPlaceholderText('例如：eNB 基础 KPI'), {
      target: { value: '复制正常模板' },
    });
    fireEvent.click(within(dialog).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(createTemplateSpy).toHaveBeenCalled();
    });
    const createInput = createTemplateSpy.mock.calls[0]?.[0];
    expect(createInput.payload.metricPaths).toEqual(['K-1']);
    expect(JSON.stringify(createInput.payload)).not.toContain('小区可用率');

    fireEvent.click(screen.getByRole('button', { name: /查询$/ }));

    const exportButton = screen.getByRole('button', { name: /导出 CSV/ });
    await waitFor(() => {
      expect(exportButton).not.toBeDisabled();
    });
    fireEvent.click(exportButton);

    await waitFor(() => {
      expect(createExportSpy).toHaveBeenCalled();
    });
    const exportInput = createExportSpy.mock.calls[0]?.[0];
    expect(exportInput.params.metric_paths).toEqual(['K-1']);
    expect(JSON.stringify(exportInput.params)).not.toContain('小区可用率');
  });
});
