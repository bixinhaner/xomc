import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import MultiInstanceTable, { composeLteEci } from '../MultiInstanceTable';

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
      path: `${objectPath}1.CID`,
      type: 'unsignedInt',
      writable: true,
      currentValue: '2561',
    },
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
    {
      path: `${objectPath}1.NeighCellEnbType`,
      type: 'int',
      writable: true,
      currentValue: '1',
      constraints: {
        enumValues: ['1', '0'],
        enumLabels: ['Home', 'Macro'],
      },
    },
    {
      path: `${objectPath}1.X2Flag`,
      type: 'string',
      writable: true,
      currentValue: '0',
      constraints: {
        enumValues: ['1', '0'],
        enumLabels: ['Manual', 'SON'],
      },
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
  useT: () => (id: string, values?: Record<string, string | number>) => {
    const labels: Record<string, string> = {
      'device.multi.neighborEnbId': 'eNB ID',
      'device.multi.neighborCellId': 'Cell ID',
      'device.multi.col.frequency': 'EARFCN',
    };
    if (labels[id]) return labels[id];
    return values?.count === undefined ? id : `${id}:${values.count}`;
  },
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
      name: 'CellID',
      titleZh: '小区ID',
      titleEn: 'Cell ID',
      leaf: 'CID',
      type: 'unsignedInt',
      required: true,
      minValue: 0,
      maxValue: 268435455,
    },
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
    {
      name: 'NeighborCellEnbType',
      titleZh: '邻区基站类型',
      titleEn: 'eNodeB Type',
      leaf: 'NeighCellEnbType',
      type: 'int',
      defaultValue: '1',
      enumOptions: [
        { value: '1', label: 'Home' },
        { value: '0', label: 'Macro' },
      ],
    },
    {
      name: 'X2Flag',
      titleZh: 'X2 标记',
      titleEn: 'X2 Flag',
      leaf: 'X2Flag',
      type: 'string',
      defaultValue: '0',
      enumOptions: [
        { value: '0', label: 'SON' },
        { value: '1', label: 'Manual' },
      ],
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

  it('shows the complete BLQ add form and composes eNB ID plus Cell ID into ECI', async () => {
    renderTable();

    fireEvent.click(screen.getByRole('button', { name: /common\.add/ }));
    const dialog = await screen.findByRole('dialog');

    for (const label of ['eNB ID', 'Cell ID', 'EARFCN', 'PCI', 'QOffset', 'CIO', 'TAC', 'eNodeB Type', 'X2 Flag']) {
      expect(within(dialog).getByText(label)).toBeInTheDocument();
    }
    expect(composeLteEci('1048575', '255')).toBe('268435455');

    fireEvent.change(within(dialog).getByRole('textbox', { name: 'eNB ID' }), { target: { value: '1048575' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Cell ID' }), { target: { value: '255' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'EARFCN' }), { target: { value: '39751' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'PCI' }), { target: { value: '10' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'TAC' }), { target: { value: '1' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'device.multi.confirmAdd' }));

    expect(await screen.findByText('268435455')).toBeInTheDocument();
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
      (await within(dialog).findAllByText('device.multi.packed.fieldRequired')).length,
    ).toBeGreaterThanOrEqual(1);
  });

  it('uses quick-settings numeric constraints when schema type is stale', async () => {
    renderTable();

    fireEvent.click(screen.getByRole('button', { name: /common\.add/ }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'eNB ID' }), { target: { value: '10' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Cell ID' }), { target: { value: '1' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'EARFCN' }), { target: { value: '-1' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'PCI' }), { target: { value: '504' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'TAC' }), { target: { value: '1' } });
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
