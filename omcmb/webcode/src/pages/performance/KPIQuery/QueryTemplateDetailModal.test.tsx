import { render, screen } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { describe, expect, it, vi } from 'vitest';

import zhCN from '@core/i18n/zh-CN';
import type { QueryTemplate } from '@core/types/pmQuery';

import QueryTemplateDetailModal from './QueryTemplateDetailModal';

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
  it('shows the complete saved query configuration', () => {
    render(
      <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
        <QueryTemplateDetailModal
          open
          template={template}
          metricLabels={{ 'KPI-36': '小区可用率' }}
          onClose={vi.fn()}
        />
      </IntlProvider>,
    );

    expect(screen.getByText('小时模板')).toBeInTheDocument();
    expect(screen.getByText(/私有/)).toBeInTheDocument();
    expect(screen.getByText('小时')).toBeInTheDocument();
    expect(screen.getByText('SN-36')).toBeInTheDocument();
    expect(screen.getByText('小区可用率（设备级）')).toBeInTheDocument();
    expect(screen.getByText('近 7 天')).toBeInTheDocument();
  });
});
