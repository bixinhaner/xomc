import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { PM_QUERY_SELECTION_LIMIT } from '@/constants/pmQueryLimits';
import zhCN from '@core/i18n/zh-CN';
import { CreateAdhocTaskDrawer, type CreateAdhocPreset } from './CreateAdhocTaskDrawer';

const createMutateAsync = vi.fn();

vi.mock('@core/hooks/api/usePmAdhoc', () => ({
  useCreatePmAdhoc: () => ({ mutateAsync: createMutateAsync, isPending: false }),
}));

function renderDrawer(preset: CreateAdhocPreset) {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <CreateAdhocTaskDrawer open preset={preset} onClose={vi.fn()} />
      </App>
    </IntlProvider>,
  );
}

describe('CreateAdhocTaskDrawer selection limits', () => {
  beforeEach(() => {
    createMutateAsync.mockReset();
  });

  it('blocks preset creation when device SNs exceed the PM query selection limit', async () => {
    renderDrawer({
      name: '设备超限聚合',
      deviceSns: Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => `SN-${index + 1}`),
      metricPaths: ['K0001'],
      granularities: ['hourly'],
    });

    fireEvent.click(screen.getByRole('button', { name: /创\s*建/ }));

    await waitFor(() => {
      expect(createMutateAsync).not.toHaveBeenCalled();
    });
  });

  it('blocks preset creation when metric paths exceed the PM query selection limit', async () => {
    renderDrawer({
      name: '指标超限聚合',
      deviceSns: ['SN-1'],
      metricPaths: Array.from({ length: PM_QUERY_SELECTION_LIMIT + 1 }, (_, index) => `K${String(index + 1).padStart(4, '0')}`),
      granularities: ['hourly'],
    });

    fireEvent.click(screen.getByRole('button', { name: /创\s*建/ }));

    await waitFor(() => {
      expect(createMutateAsync).not.toHaveBeenCalled();
    });
  });

  it('未手动修改计划结束时间时不提交 plannedEndAt，由后端按创建时间默认', async () => {
    createMutateAsync.mockResolvedValue({ id: 'created-task' });
    renderDrawer({
      name: '默认计划结束任务',
      deviceSns: ['SN-1'],
      metricPaths: ['K0001'],
      granularities: ['hourly'],
    });

    fireEvent.click(screen.getByRole('button', { name: /创\s*建/ }));

    await waitFor(() => {
      expect(createMutateAsync).toHaveBeenCalled();
    });
    expect(createMutateAsync.mock.calls[0][0]).not.toHaveProperty('plannedEndAt');
  });
});
