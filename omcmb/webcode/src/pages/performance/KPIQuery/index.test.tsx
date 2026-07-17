import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';
import type { QueryTemplate } from '@core/types/pmQuery';
import KPIQuery from './index';

const refetchAggSpy = vi.fn();
const createTemplateSpy = vi.fn();
const createExportSpy = vi.fn();

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

vi.mock('@core/hooks/api/usePmQuery', () => ({
  useQueryTemplates: () => ({
    data: { items: [validTemplate, overLimitTemplate], total: 2 },
    isLoading: false,
    refetch: vi.fn(),
  }),
  useCreateQueryTemplate: () => ({ mutateAsync: createTemplateSpy, isPending: false }),
  useUpdateQueryTemplate: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteQueryTemplate: () => ({ mutateAsync: vi.fn() }),
  useAggregatedMetricsByDevices: () => ({
    data: [],
    total: 0,
    truncated: false,
    isLoading: false,
    isFetching: false,
    errors: [],
    refetch: refetchAggSpy,
  }),
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
  default: () => null,
}));

vi.mock('@/components/MetricPickerModal', () => ({
  default: () => null,
}));

vi.mock('./components/PivotTable', () => ({
  default: () => <div data-testid="pivot-table" />,
}));

vi.mock('./templateMetricResolver', () => ({
  resolveTemplateMetricPaths: async (_deviceType: string, paths: string[]) => ({
    paths,
    labels: {},
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

describe('KPIQuery 模板数量限制', () => {
  beforeEach(() => {
    refetchAggSpy.mockClear();
    createTemplateSpy.mockReset();
    createExportSpy.mockReset();
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

  it('已有查询快照时，当前老模板超 50 个设备也不能发起导出', async () => {
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

    expect(createExportSpy).not.toHaveBeenCalled();
  });
});

describe('KPIQuery 模板弹窗初始值', () => {
  beforeEach(() => {
    refetchAggSpy.mockClear();
    createTemplateSpy.mockReset();
    createExportSpy.mockReset();
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
    expect(within(dialog).queryByDisplayValue('已选 1 个：K-1')).toBeNull();
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
    fireEvent.click(await screen.findByText('近 1 小时'));
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
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择指标')).toHaveValue('已选 1 个：K-1');
    expect(within(dialog).getByText('近 1 小时')).toBeTruthy();
  });

  it('编辑模板弹窗仍回填待编辑模板数据', async () => {
    renderPage();

    fireEvent.click(screen.getAllByRole('button', { name: '编辑' })[0]);

    const dialog = await findModalByTitle('编辑查询模板');
    expect(within(dialog).getByDisplayValue('正常模板')).toBeTruthy();
    expect(within(dialog).getByDisplayValue('正常模板描述')).toBeTruthy();
    expect(within(dialog).getByLabelText('公共（所有人可见）')).toBeChecked();
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择设备')).toHaveValue('已选 1 个：SN-OK');
    expect(within(dialog).getByPlaceholderText('点击右侧按钮选择指标')).toHaveValue('已选 1 个：K-1');
    expect(within(dialog).getByText('近 1 小时')).toBeTruthy();
  });
});
