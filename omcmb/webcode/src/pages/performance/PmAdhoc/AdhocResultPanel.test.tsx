import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { zhCN } from '@core/i18n';
import type { AdhocDimension, AdhocTask } from '@core/types/pmAdhoc';
import { AdhocResultPanel } from './AdhocResultPanel';

const mocks = vi.hoisted(() => ({
  createExportSpy: vi.fn(),
  task: undefined as AdhocTask | undefined,
}));

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echart" />,
}));

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  usePmAdhocDetail: () => ({
    data: mocks.task,
    isLoading: false,
    isError: false,
  }),
  usePmAdhocResults: () => ({
    data: { rows: [], total: 0 },
    isLoading: false,
  }),
}));

vi.mock('@core/hooks/api/useKpiExport', () => ({
  useCreateKpiExport: () => ({ mutate: mocks.createExportSpy, isPending: false }),
}));

vi.mock('@core/hooks/api/useSystemTimezone', () => ({
  useSystemTimezoneValue: () => 'Asia/Shanghai',
}));

function makeTask(dimension: AdhocDimension): AdhocTask {
  return {
    id: 'task-1',
    name: '摘要测试任务',
    mode: 'oneshot',
    deviceSns: ['SN-001', 'SN-002'],
    metricPaths: ['K0001', 'K0002'],
    granularities: ['hourly'],
    windowStart: '2026-07-20T00:00:00Z',
    windowEnd: '2026-07-20T01:00:00Z',
    dimension,
    technology: 'lte',
    isBuiltin: false,
    expireDays: 60,
    visibility: 'private',
    status: 'succeeded',
    progress: 100,
    creator: 'alice',
    createdAt: '2026-07-20T00:00:00Z',
    updatedAt: '2026-07-20T00:00:00Z',
  };
}

function renderPanel() {
  render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <AdhocResultPanel taskId="task-1" />
      </App>
    </IntlProvider>,
  );
}

describe('AdhocResultPanel result summary', () => {
  beforeEach(() => {
    mocks.createExportSpy.mockClear();
  });

  it.each<AdhocDimension>(['device', 'aggregate_group'])(
    'keeps device count for selected-device %s dimension tasks',
    (dimension) => {
      mocks.task = makeTask(dimension);

      renderPanel();

      expect(screen.getByText('2 设备 × 2 指标 × 1 粒度')).toBeInTheDocument();
    },
  );

  it.each<AdhocDimension>(['product', 'band', 'device_group', 'network'])(
    'hides device count for non-selected-device %s dimension tasks',
    (dimension) => {
      mocks.task = makeTask(dimension);

      renderPanel();

      expect(screen.getByText('2 指标 × 1 粒度')).toBeInTheDocument();
      expect(screen.queryByText('2 设备 × 2 指标 × 1 粒度')).not.toBeInTheDocument();
    },
  );
});
