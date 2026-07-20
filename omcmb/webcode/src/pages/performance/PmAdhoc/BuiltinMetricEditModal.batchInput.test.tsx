import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import zhCN from '@core/i18n/zh-CN';
import type { AdhocTask } from '@core/types/pmAdhoc';
import BuiltinMetricEditModal from './BuiltinMetricEditModal';

const updateMutateAsync = vi.fn();

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  useUpdatePmAdhoc: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}));

vi.mock('@core/hooks/api/usePerformance', () => ({
  useIndicatorCandidates: () => ({
    data: [
      { id: 'K0001', name: 'availability', cnName: '可用率', enName: 'Availability', isCounter: false },
      { id: 'C0001', name: 'rrc_att', cnName: 'RRC请求次数', enName: 'RRC Attempts', isCounter: true },
    ],
    isLoading: false,
  }),
}));

const task: AdhocTask = {
  id: 'builtin-1',
  name: '内置聚合任务',
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

function renderModal() {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <BuiltinMetricEditModal open task={task} onClose={vi.fn()} />
      </App>
    </IntlProvider>,
  );
}

describe('BuiltinMetricEditModal batch metric input', () => {
  beforeEach(() => {
    updateMutateAsync.mockReset();
  });

  it('adds valid metric IDs from batch input and ignores missing IDs before saving', async () => {
    renderModal();

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
});
