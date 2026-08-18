import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import FixedScalarSettingsTable from '../FixedScalarSettingsTable';

const mocks = vi.hoisted(() => ({
  taskStatus: 'sent',
  updateParameters: vi.fn(),
  refetchSchema: vi.fn(),
  invalidateParameterSchemaCache: vi.fn(),
}));

const schema = {
  parameters: [{
    path: 'Device.DeviceInfo.STATICROUTE1_NETADDR',
    type: 'STRING',
    writable: true,
    currentValue: '10.0.0.0',
  }],
  objects: [],
  total: 1,
};

vi.mock('@core/hooks/api/useDeviceParameters', () => ({
  useParameterSchema: () => ({
    data: schema,
    refetch: mocks.refetchSchema,
  }),
  useUpdateParameters: () => ({
    isPending: false,
    mutateAsync: mocks.updateParameters,
  }),
}));

vi.mock('@core/hooks/api/useDeviceTask', () => ({
  useDeviceTaskStatus: (taskId?: string) => ({
    data: taskId ? { id: taskId, status: mocks.taskStatus, result: {} } : undefined,
  }),
}));

vi.mock('@core/services/api/deviceParameterApi', () => ({
  deviceParameterApi: {
    invalidateParameterSchemaCache: mocks.invalidateParameterSchemaCache,
  },
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

const groups: QuickSettingsGroup[] = [{
  id: 'device-static-route-1',
  titleZh: '静态路由 1',
  titleEn: 'Static Route 1',
  multiInstance: false,
  params: [{
    name: 'DestinationNetwork',
    titleZh: '目的网络',
    titleEn: 'Destination Network',
    standardPath: 'Device.DeviceInfo.STATICROUTE1_NETADDR',
  }],
}];

describe('FixedScalarSettingsTable terminal refresh', () => {
  beforeEach(() => {
    mocks.taskStatus = 'sent';
    mocks.updateParameters.mockReset();
    mocks.updateParameters.mockResolvedValue({ taskId: 'task-1' });
    mocks.refetchSchema.mockReset();
    mocks.refetchSchema.mockResolvedValue({ data: schema });
    mocks.invalidateParameterSchemaCache.mockReset();
  });

  it.each(['completed', 'failed'])('refreshes the list when the device task becomes %s', async (terminalStatus) => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const view = render(
      <FixedScalarSettingsTable
        deviceId="device-1"
        groups={groups}
        locale="zh-CN"
        kind="static-route"
      />,
      { wrapper },
    );

    fireEvent.click(screen.getByRole('button', { name: '编辑第 1 行' }));
    fireEvent.change(screen.getByLabelText('目的网络'), { target: { value: '10.1.0.0' } });
    fireEvent.click(screen.getByRole('button', { name: /提\s*交/ }));
    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(1));

    invalidateQueries.mockClear();
    mocks.invalidateParameterSchemaCache.mockClear();
    mocks.taskStatus = terminalStatus;
    view.rerender(
      <FixedScalarSettingsTable
        deviceId="device-1"
        groups={groups}
        locale="zh-CN"
        kind="static-route"
      />,
    );

    await waitFor(() => {
      expect(mocks.invalidateParameterSchemaCache).toHaveBeenCalledWith('device-1');
      expect(invalidateQueries).toHaveBeenCalledWith(expect.objectContaining({
        queryKey: ['devices', 'parameters', 'device-1'],
      }));
    });
  });
});
