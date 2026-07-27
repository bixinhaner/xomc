import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import zhCN from '@core/i18n/zh-CN';
import type { QueryTemplate } from '@core/types/pmQuery';

import QueryTemplateDetailModal from './QueryTemplateDetailModal';

vi.mock('@core/utils/saveBlob', () => ({
  saveBlob: vi.fn(),
}));

import { saveBlob } from '@core/utils/saveBlob';

vi.mock('@core/hooks/api/useIndicatorsLibrary', () => ({
  useAllIndicators: () => ({
    data: {
      items: [
        {
          id: 'KPI-36',
          indicatorLevel: 'device',
        },
      ],
    },
  }),
}));

const template: QueryTemplate = {
  id: 'template-36',
  name: '小时模板',
  visibility: 'private',
  creatorId: 'user-1',
  description: '完整详情',
  payload: {
    deviceSns: ['SN-36'],
    metricPaths: ['KPI-36'],
    granularity: 'hourly',
    timeRangePreset: 'last_7d',
    deviceType: 'ENB',
  },
  createdAt: '2026-07-11T00:00:00Z',
  updatedAt: '2026-07-11T01:00:00Z',
};

describe('QueryTemplateDetailModal', () => {
  beforeEach(() => {
    vi.mocked(saveBlob).mockClear();
  });

  function renderModal(
    targetTemplate: QueryTemplate = template,
    metricLabels = { 'KPI-36': '小区可用率' },
  ) {
    return render(
      <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
        <App>
          <QueryTemplateDetailModal
            open
            template={targetTemplate}
            metricLabels={metricLabels}
            onClose={vi.fn()}
          />
        </App>
      </IntlProvider>,
    );
  }

  it('shows the complete saved query configuration', () => {
    renderModal();

    expect(screen.getByText('小时模板')).toBeInTheDocument();
    expect(screen.getByText(/私有/)).toBeInTheDocument();
    expect(screen.getByText('小时')).toBeInTheDocument();
    expect(screen.getByText('SN-36')).toBeInTheDocument();
    expect(screen.getByText('小区可用率（设备级）')).toBeInTheDocument();
    expect(screen.getByText('近 7 天')).toBeInTheDocument();
  });

  it('exports selected metric IDs from the template detail modal', () => {
    renderModal();

    fireEvent.click(screen.getByRole('button', { name: /导出指标 ID/ }));

    expect(saveBlob).toHaveBeenCalledWith(
      'KPI-36',
      '小时模板_metrics.txt',
      'text/plain;charset=utf-8',
    );
  });

  it('does not export when the template has no selected metrics', async () => {
    renderModal({
      ...template,
      payload: {
        ...template.payload,
        metricPaths: [],
      },
    });

    fireEvent.click(screen.getByRole('button', { name: /导出指标 ID/ }));

    expect(saveBlob).not.toHaveBeenCalled();
    expect(await screen.findByText('该模板没有已选指标')).toBeInTheDocument();
  });
});
