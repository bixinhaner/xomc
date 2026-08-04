import { fireEvent, render, screen } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { describe, expect, it, vi } from 'vitest';

import { zhCN } from '@core/i18n';
import type { DashboardPeriodProgress } from '@core/types/dashboard';
import { LayoutKPIPanel } from '../LayoutKPIPanel';

vi.mock('../useMetricMetadata', () => ({
  useMetricMetadata: () => ({
    getMeta: (code: string) => ({ name: code, unit: '%', isCounter: false }),
    isLoading: false,
  }),
  resolveMetricMeta: (
    code: string,
    meta: { getMeta: (value: string) => { name: string; unit: string } },
  ) => ({
    name: meta.getMeta(code)?.name ?? code,
    unit: meta.getMeta(code)?.unit ?? '',
    conversion: 1,
  }),
}));

vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({ colorText: '#111' }),
}));

vi.mock('@/components/Charts/LineChart', () => ({
  default: () => <div data-testid="line-chart" />,
}));

const progress: DashboardPeriodProgress = {
  taskId: '0184dddd-0001-4000-8000-000000000001',
  taskVersionId: '11111111-1111-4111-8111-111111111111',
  granularity: 'daily',
  windowStart: '2026-07-31T00:00:00+08:00',
  windowEnd: '2026-08-01T00:00:00+08:00',
  entityKey: 'network',
  revision: 2,
  versionEffectiveFrom: '2026-07-31T06:00:00+08:00',
  versionEffectiveTo: null,
  receivedSlots: 4,
  expectedSlots: 24,
  versionExpectedSlots: 18,
  coverageRatio: 1 / 6,
  versionSliceComplete: false,
  periodComplete: false,
  state: 'partial',
};

function renderPanel(
  progressState: 'available' | 'unavailable',
  periodProgress: DashboardPeriodProgress[],
  granularity: 'daily' | 'weekly' = 'daily',
) {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <LayoutKPIPanel
        technology="lte"
        panel={{
          title: '测试指标',
          metrics: ['K900010006'],
          x: 0,
          y: 0,
          w: 6,
          h: 8,
          chartType: 'line',
        }}
        trendData={{
          K900010006: {
            current: [{ time: progress.windowStart, value: 98.5 }],
            compare: [],
            metadata: { kpi_name: 'K900010006', compare_type: 'last_week' },
          },
        }}
        isLoading={false}
        granularity={granularity}
        onGranularityChange={vi.fn()}
        bucketKeys={['2026-07-31']}
        periodProgress={periodProgress}
        progressState={progressState}
      />
    </IntlProvider>,
  );
}

describe('LayoutKPIPanel current period progress', () => {
  it('显示当前自然周期覆盖率', () => {
    renderPanel('available', [progress]);

    expect(screen.getByText('进行中 4/24（16.7%）')).toBeInTheDocument();
  });

  it('进度源失败时明确显示暂不可用', () => {
    renderPanel('unavailable', []);

    expect(screen.getByText('进行中状态暂不可用')).toBeInTheDocument();
  });

  it('区分版本片段完整和自然周期完整并显示完整版本有效区间', async () => {
    renderPanel('available', [{
      ...progress,
      versionEffectiveTo: '2026-07-31T18:00:00+08:00',
      receivedSlots: 18,
      versionExpectedSlots: 18,
      versionSliceComplete: true,
      periodComplete: false,
    }]);

    fireEvent.mouseEnter(screen.getByText('进行中 18/24（75.0%）'));

    expect(await screen.findByText(/有效区间 2026-07-31T06:00:00\+08:00 至 2026-07-31T18:00:00\+08:00/))
      .toBeInTheDocument();
    expect(screen.getByText('统计开始时间：2026-07-31T00:00:00+08:00')).toBeInTheDocument();
    expect(screen.getByText('统计结束时间：2026-08-01T00:00:00+08:00')).toBeInTheDocument();
    expect(screen.getByText('版本片段 18/18：完整')).toBeInTheDocument();
    expect(screen.getByText('自然周期 18/24：未完整')).toBeInTheDocument();
  });

  it('周视图优先展示自然周进度而不是 08:00 边界窗口', () => {
    const naturalWeek: DashboardPeriodProgress = {
      ...progress,
      granularity: 'weekly',
      windowStart: '2026-08-03T00:00:00+08:00',
      windowEnd: '2026-08-10T00:00:00+08:00',
      receivedSlots: 1,
      expectedSlots: 168,
      versionExpectedSlots: 151,
      coverageRatio: 1 / 168,
    };
    const boundaryWeek: DashboardPeriodProgress = {
      ...progress,
      granularity: 'weekly',
      windowStart: '2026-08-03T08:00:00+08:00',
      windowEnd: '2026-08-10T08:00:00+08:00',
      receivedSlots: 1,
      expectedSlots: 7,
      versionExpectedSlots: 7,
      coverageRatio: 1 / 7,
    };

    renderPanel('available', [naturalWeek, boundaryWeek], 'weekly');

    expect(screen.getByText('进行中 1/168（0.6%）')).toBeInTheDocument();
    expect(screen.queryByText('进行中 1/7（14.3%）')).not.toBeInTheDocument();
  });

  it('周视图不会把 08:00 边界窗口仅因 168 槽位误判为自然周', () => {
    const naturalWeek: DashboardPeriodProgress = {
      ...progress,
      granularity: 'weekly',
      windowStart: '2026-08-03T00:00:00+08:00',
      windowEnd: '2026-08-10T00:00:00+08:00',
      receivedSlots: 1,
      expectedSlots: 168,
      versionExpectedSlots: 151,
      coverageRatio: 1 / 168,
    };
    const boundaryWeek: DashboardPeriodProgress = {
      ...progress,
      granularity: 'weekly',
      windowStart: '2026-08-03T08:00:00+08:00',
      windowEnd: '2026-08-10T08:00:00+08:00',
      receivedSlots: 24,
      expectedSlots: 168,
      versionExpectedSlots: 7,
      coverageRatio: 24 / 168,
    };

    renderPanel('available', [naturalWeek, boundaryWeek], 'weekly');

    expect(screen.getByText('进行中 1/168（0.6%）')).toBeInTheDocument();
    expect(screen.queryByText('进行中 24/168（14.3%）')).not.toBeInTheDocument();
  });
});
