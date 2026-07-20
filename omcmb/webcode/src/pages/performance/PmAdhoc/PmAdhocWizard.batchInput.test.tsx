import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import zhCN from '@core/i18n/zh-CN';
import PmAdhocWizard from './PmAdhocWizard';

const navigateSpy = vi.fn();

vi.mock('react-router-dom', () => ({
  useNavigate: () => navigateSpy,
  useParams: () => ({}),
}));

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  useCreatePmAdhoc: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdatePmAdhoc: () => ({ mutateAsync: vi.fn(), isPending: false }),
  usePmAdhocDetail: () => ({ data: undefined }),
}));

vi.mock('@core/hooks/api/useDevices', () => ({
  useDeviceList: () => ({ data: { items: [] }, isLoading: false }),
}));

vi.mock('@core/hooks/api/usePmQuery', () => ({
  useMetricObjectsByDevices: () => ({ byDevice: {} }),
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
  });
});
