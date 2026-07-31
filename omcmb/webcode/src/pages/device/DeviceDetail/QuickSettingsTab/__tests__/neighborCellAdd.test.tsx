import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import MultiInstanceTable from '../MultiInstanceTable';

const mocks = vi.hoisted(() => ({
  updateParameters: vi.fn(),
  addObject: vi.fn(),
  deleteObject: vi.fn(),
}));

const objectPath =
  'Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.';

const schema = {
  parameters: [
    {
      path: `${objectPath}1.EUTRACarrierARFCN`,
      type: 'string',
      writable: true,
      currentValue: '38750',
    },
    {
      path: `${objectPath}1.PhyCellID`,
      type: 'string',
      writable: true,
      currentValue: '12',
    },
    {
      path: `${objectPath}1.QOffset`,
      type: 'string',
      writable: true,
      currentValue: '0',
    },
    {
      path: `${objectPath}1.CIO`,
      type: 'string',
      writable: true,
      currentValue: '0',
    },
    {
      path: `${objectPath}1.X_COM_TAC`,
      type: 'string',
      writable: true,
      currentValue: '1',
    },
    {
      path: `${objectPath}1.PLMNID`,
      type: 'string',
      writable: true,
      currentValue: '46000',
    },
  ],
  objects: [
    {
      path: objectPath,
      currentInstances: [1],
      canAdd: true,
      canDeleteAny: true,
    },
  ],
  total: 6,
};

vi.mock('@core/hooks/api/useDeviceParameters', () => ({
  useParameterSchema: () => ({
    data: schema,
    isLoading: false,
    refetch: vi.fn().mockResolvedValue({ data: schema }),
  }),
  useSearchParameters: () => ({ data: [] }),
  useUpdateParameters: () => ({
    isPending: false,
    mutateAsync: mocks.updateParameters,
  }),
  useAddObject: () => ({
    isPending: false,
    mutateAsync: mocks.addObject,
  }),
  useDeleteObject: () => ({
    isPending: false,
    mutateAsync: mocks.deleteObject,
  }),
}));

vi.mock('@core/hooks/api/useDevices', () => ({
  useSyncDeviceParams: () => ({
    isPending: false,
    mutateAsync: vi.fn(),
  }),
}));

vi.mock('@core/hooks/api/useDeviceTask', () => ({
  useDeviceTaskStatus: () => ({ data: undefined }),
}));

vi.mock('@core/services/api/deviceTaskApi', () => ({
  deviceTaskApi: { getTask: vi.fn() },
}));

vi.mock('@core/services/api/deviceParameterApi', () => ({
  deviceParameterApi: {
    invalidateParameterSchemaCache: vi.fn(),
    searchParameters: vi.fn(),
  },
}));

vi.mock('@core/services/api/configSyncApi', () => ({
  configSyncApi: { submit: vi.fn() },
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string, values?: Record<string, string | number>) =>
    values?.count === undefined ? id : `${id}:${values.count}`,
}));

const neighborGroup: QuickSettingsGroup = {
  id: 'enb-neighbor-cell',
  titleZh: '邻区列表',
  titleEn: 'Neighbor Cell List',
  multiInstance: true,
  maxInstances: 32,
  objectPath:
    'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.',
  params: [
    {
      name: 'EARFCN',
      titleZh: '频点',
      titleEn: 'EARFCN',
      leaf: 'EUTRACarrierARFCN',
      type: 'unsignedInt',
      minValue: 0,
      maxValue: 65535,
    },
    {
      name: 'PCI',
      titleZh: 'PCI',
      titleEn: 'PCI',
      leaf: 'PhyCellID',
      type: 'unsignedInt',
      minValue: 0,
      maxValue: 503,
    },
    {
      name: 'QOffset',
      titleZh: 'QOffset',
      titleEn: 'QOffset',
      leaf: 'QOffset',
      type: 'int',
      minValue: -24,
      maxValue: 24,
    },
    {
      name: 'CIO',
      titleZh: 'CIO',
      titleEn: 'CIO',
      leaf: 'CIO',
      type: 'int',
      minValue: -24,
      maxValue: 24,
    },
    {
      name: 'TAC',
      titleZh: 'TAC',
      titleEn: 'TAC',
      leaf: 'X_COM_TAC',
      type: 'unsignedInt',
      required: true,
      minValue: 0,
      maxValue: 65535,
    },
    {
      name: 'PLMN',
      titleZh: 'PLMN',
      titleEn: 'PLMN',
      leaf: 'PLMNID',
      type: 'string',
      minValue: 5,
      maxValue: 6,
    },
  ],
};

function renderTable() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  return render(
    <MultiInstanceTable
      deviceId="device-1"
      group={neighborGroup}
      instanceContext={{ networkType: 'lte', fapInstance: 1 }}
      locale="zh-CN"
    />,
    { wrapper },
  );
}

describe('LTE neighbor cell add', () => {
  beforeEach(() => {
    mocks.updateParameters.mockReset();
    mocks.addObject.mockReset();
    mocks.deleteObject.mockReset();
    useQuickSettingsFeedbackStore.setState({
      entries: {},
      drafts: {},
      draftRevisions: {},
    });
  });

  it('shows the vendor TAC leaf and requires a value', async () => {
    renderTable();

    fireEvent.click(screen.getByRole('button', { name: /common\.add/ }));
    const dialog = await screen.findByRole('dialog');

    expect(within(dialog).getByText('TAC')).toBeInTheDocument();
    fireEvent.click(
      within(dialog).getByRole('button', {
        name: 'device.multi.confirmAdd',
      }),
    );

    expect(
      await within(dialog).findByText('device.multi.packed.fieldRequired'),
    ).toBeInTheDocument();
  });

  it('uses quick-settings numeric constraints when schema type is stale', async () => {
    renderTable();

    fireEvent.click(screen.getByRole('button', { name: /common\.add/ }));
    const dialog = await screen.findByRole('dialog');
    const inputs = within(dialog).getAllByRole('textbox');
    fireEvent.change(inputs[0], { target: { value: '-1' } });
    fireEvent.change(inputs[1], { target: { value: '504' } });
    fireEvent.change(inputs[4], { target: { value: '1' } });
    fireEvent.click(
      within(dialog).getByRole('button', {
        name: 'device.multi.confirmAdd',
      }),
    );

    await waitFor(() => {
      expect(within(dialog).getByText('请输入非负整数')).toBeInTheDocument();
      expect(within(dialog).getByText('最大值为 503')).toBeInTheDocument();
    });
    expect(mocks.addObject).not.toHaveBeenCalled();
  });
});
