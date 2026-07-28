import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import zhCN from '@core/i18n/zh-CN';
import type { AdhocTask } from '@core/types/pmAdhoc';
import BuiltinMetricEditModal from './BuiltinMetricEditModal';

const updateMutateAsync = vi.fn();
const useIndicatorCandidatesSpy = vi.fn();

let indicatorCandidates = [
  { id: 'K0001', name: 'availability', cnName: '可用率', enName: 'Availability', isCounter: false },
  { id: 'C0001', name: 'rrc_att', cnName: 'RRC请求次数', enName: 'RRC Attempts', isCounter: true },
];

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  useUpdatePmAdhoc: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
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
  isKnownTechnology: (value: unknown) => value === 'lte' || value === 'nr' || value === 'gsm',
  technologyToDeviceType: (tech: string) => ({ lte: 'ENB', nr: 'GNB', gsm: 'GSM' })[tech],
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
}));

const task: AdhocTask = {
  id: 'builtin-1',
  name: '内置-全网-LTE',
  mode: 'oneshot',
  deviceSns: [],
  metricPaths: ['K0001'],
  granularities: ['hourly'],
  windowStart: '2026-07-20T00:00:00Z',
  windowEnd: '2026-07-20T01:00:00Z',
  dimension: 'network',
  technology: 'lte',
  isBuiltin: true,
  expireDays: 60,
  status: 'scheduled',
  progress: 0,
  creator: 'system',
  createdAt: '2026-07-20T00:00:00Z',
  updatedAt: '2026-07-20T00:00:00Z',
};

function renderModal(targetTask: AdhocTask = task) {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <BuiltinMetricEditModal open task={targetTask} onClose={vi.fn()} />
      </App>
    </IntlProvider>,
  );
}

describe('BuiltinMetricEditModal batch metric input', () => {
  beforeEach(() => {
    updateMutateAsync.mockReset();
    useIndicatorCandidatesSpy.mockClear();
    indicatorCandidates = [
      { id: 'K0001', name: 'availability', cnName: '可用率', enName: 'Availability', isCounter: false },
      { id: 'C0001', name: 'rrc_att', cnName: 'RRC请求次数', enName: 'RRC Attempts', isCounter: true },
    ];
  });

  it('adds valid metric IDs from batch input and ignores missing IDs before saving', async () => {
    renderModal();
    expect(screen.getByText('编辑指标：内置-全网-eNB(LTE)')).toBeInTheDocument();
    expect(useIndicatorCandidatesSpy).toHaveBeenCalledWith(
      'ENB',
      expect.objectContaining({ includeCounters: true, enabledOnly: true }),
    );

    fireEvent.click(screen.getByRole('button', { name: /批量输入指标 ID/ }));
    fireEvent.change(screen.getByPlaceholderText(/K000000001/), {
      target: { value: 'C0001, C404, C0001' },
    });
    fireEvent.click(screen.getByRole('button', { name: /加入已选/ }));
    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => {
      expect(updateMutateAsync).toHaveBeenCalledWith({
        id: 'builtin-1',
        input: { metricPaths: ['K0001', 'C0001'] },
      });
    });
  });

  it('caps batch-added metrics at the PM query selection limit before saving', async () => {
    indicatorCandidates = Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => {
      const id = `K${String(index + 1).padStart(4, '0')}`;
      return { id, name: `metric_${index + 1}`, cnName: `指标${index + 1}`, enName: `Metric ${index + 1}`, isCounter: false };
    });
    renderModal({ ...task, metricPaths: [] });

    fireEvent.click(screen.getByRole('button', { name: /批量输入指标 ID/ }));
    fireEvent.change(screen.getByPlaceholderText(/K000000001/), {
      target: { value: indicatorCandidates.map((item) => item.id).join('\n') },
    });
    fireEvent.click(screen.getByRole('button', { name: /加入已选/ }));
    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => {
      expect(updateMutateAsync).toHaveBeenCalledWith({
        id: 'builtin-1',
        input: { metricPaths: indicatorCandidates.slice(0, PM_QUERY_SELECTION_LIMIT).map((item) => item.id) },
      });
    });
  });

  it('blocks saving when an existing task already has more than the PM query selection limit', () => {
    const overLimitMetricPaths = Array.from(
      { length: PM_QUERY_SELECTION_LIMIT + 1 },
      (_, index) => `K${String(index + 1).padStart(4, '0')}`,
    );
    renderModal({ ...task, metricPaths: overLimitMetricPaths });

    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

    expect(updateMutateAsync).not.toHaveBeenCalled();
  });
});
