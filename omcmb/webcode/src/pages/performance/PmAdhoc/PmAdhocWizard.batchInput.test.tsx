import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import zhCN from '@core/i18n/zh-CN';
import type { AdhocTask } from '@core/types/pmAdhoc';
import PmAdhocWizard from './PmAdhocWizard';

const navigateSpy = vi.fn();
const createMutateAsync = vi.fn();
let routeParams: { id?: string } = {};
let adhocDetail: AdhocTask | undefined;
const useIndicatorCandidatesSpy = vi.fn();

let indicatorCandidates = [
  { id: 'K0001', name: 'availability', cnName: '可用率', enName: 'Availability', isCounter: false },
  { id: 'C0001', name: 'rrc_att', cnName: 'RRC请求次数', enName: 'RRC Attempts', isCounter: true },
];

vi.mock('react-router-dom', () => ({
  useNavigate: () => navigateSpy,
  useParams: () => routeParams,
}));

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  useCreatePmAdhoc: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdatePmAdhoc: () => ({ mutateAsync: vi.fn(), isPending: false }),
  usePmAdhocDetail: () => ({ data: adhocDetail }),
}));

vi.mock('@core/hooks/api/useDevices', () => ({
  useDeviceList: () => ({ data: { items: [] }, isLoading: false }),
}));

vi.mock('@core/hooks/api/usePmQuery', () => ({
  useMetricObjectsByDevices: () => ({ byDevice: {} }),
}));

vi.mock('@core/hooks/api/usePerformance', () => ({
  useIndicatorCandidates: (deviceType: unknown, opts: unknown) => {
    useIndicatorCandidatesSpy(deviceType, opts);
    return {
      data: indicatorCandidates,
      isLoading: false,
    };
  },
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
    labelForTechnology: (tech?: string | null) =>
      ({ lte: 'eNB(LTE)', nr: 'gNB(NR)', gsm: 'GSM' })[tech ?? ''] ?? (tech ? tech.toUpperCase() : '—'),
    labelForRadioMode: (radioMode?: string | null) => (radioMode ? radioMode : '—'),
    isLoading: false,
  }),
  technologyToDeviceType: (tech: string) => ({ lte: 'ENB', nr: 'GNB', gsm: 'GSM' })[tech],
}));

vi.mock('../PmDashboard/CellDrilldownSelector', () => ({
  default: () => <div data-testid="cell-drilldown-selector" />,
}));

function renderWizard() {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <PmAdhocWizard />
      </App>
    </IntlProvider>,
  );
}

describe('PmAdhocWizard batch metric input', () => {
  beforeEach(() => {
    navigateSpy.mockReset();
    createMutateAsync.mockReset();
    useIndicatorCandidatesSpy.mockClear();
    routeParams = {};
    adhocDetail = undefined;
    indicatorCandidates = [
      { id: 'K0001', name: 'availability', cnName: '可用率', enName: 'Availability', isCounter: false },
      { id: 'C0001', name: 'rrc_att', cnName: 'RRC请求次数', enName: 'RRC Attempts', isCounter: true },
    ];
  });

  it('batch-adds valid metric IDs in step 3 and leaves missing IDs unselected', () => {
    renderWizard();

    fireEvent.change(screen.getByPlaceholderText('如：小时级 RRC 成功率'), {
      target: { value: '批量指标任务' },
    });
    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));
    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));

    fireEvent.click(screen.getByRole('button', { name: /批量输入指标 ID/ }));
    fireEvent.change(screen.getByPlaceholderText(/K000000001/), {
      target: { value: 'K0001，C0001 C404 K0001' },
    });
    fireEvent.click(screen.getByRole('button', { name: /加入已选/ }));

    expect(screen.getByText('已选 2 个指标')).toBeInTheDocument();
    expect(useIndicatorCandidatesSpy).toHaveBeenCalledWith(
      'ENB',
      expect.objectContaining({ includeCounters: true, enabledOnly: true }),
    );
  });

  it('caps batch-added metrics at the PM query selection limit', () => {
    indicatorCandidates = Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => {
      const id = `K${String(index + 1).padStart(4, '0')}`;
      return { id, name: `metric_${index + 1}`, cnName: `指标${index + 1}`, enName: `Metric ${index + 1}`, isCounter: false };
    });
    renderWizard();

    fireEvent.change(screen.getByPlaceholderText('如：小时级 RRC 成功率'), {
      target: { value: '批量指标任务' },
    });
    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));
    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));

    fireEvent.click(screen.getByRole('button', { name: /批量输入指标 ID/ }));
    fireEvent.change(screen.getByPlaceholderText(/K000000001/), {
      target: { value: indicatorCandidates.map((item) => item.id).join('\n') },
    });
    fireEvent.click(screen.getByRole('button', { name: /加入已选/ }));

    expect(screen.getByText(`已选 ${PM_QUERY_SELECTION_LIMIT} 个指标`)).toBeInTheDocument();
  });

  it('新建时未手动修改计划结束时间则交给后端按创建时间默认', async () => {
    createMutateAsync.mockResolvedValue({ id: 'created-task' });
    renderWizard();

    fireEvent.change(screen.getByPlaceholderText('如：小时级 RRC 成功率'), {
      target: { value: '默认计划结束任务' },
    });
    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));
    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));

    fireEvent.click(screen.getByRole('button', { name: /批量输入指标 ID/ }));
    fireEvent.change(screen.getByPlaceholderText(/K000000001/), {
      target: { value: 'K0001' },
    });
    fireEvent.click(screen.getByRole('button', { name: /加入已选/ }));

    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));
    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));
    fireEvent.click(screen.getByRole('button', { name: /提\s*交/ }));

    await waitFor(() => {
      expect(createMutateAsync).toHaveBeenCalled();
    });
    expect(createMutateAsync.mock.calls[0][0]).not.toHaveProperty('plannedEndAt');
  });

  it('does not allow an edited custom-device task with more than the PM query selection limit to leave the scope step', async () => {
    routeParams = { id: 'adhoc-over-limit' };
    adhocDetail = {
      id: 'adhoc-over-limit',
      name: '设备超限任务',
      mode: 'oneshot',
      deviceSns: Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => `SN-${index + 1}`),
      metricPaths: ['K0001'],
      granularities: ['hourly'],
      windowStart: '2026-07-20T00:00:00Z',
      windowEnd: '2026-07-20T01:00:00Z',
      dimension: 'aggregate_group',
      technology: 'lte',
      isBuiltin: false,
      expireDays: 60,
      visibility: 'private',
      status: 'scheduled',
      progress: 0,
      creator: 'tester',
      createdAt: '2026-07-20T00:00:00Z',
      updatedAt: '2026-07-20T00:00:00Z',
    };
    renderWizard();

    await waitFor(() => {
      expect(screen.getByDisplayValue('设备超限任务')).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: /下一步/ }));

    expect(screen.getByText(`自选设备（eNB(LTE)，已选 ${PM_QUERY_SELECTION_LIMIT + 1}）*`)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /下一步/ })).toBeDisabled();
  });
});
