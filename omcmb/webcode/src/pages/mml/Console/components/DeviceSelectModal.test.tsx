import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import type { Device } from '@core/types/device';
import type { useDeviceList } from '@core/hooks/api/useDevices';
import DeviceSelectModal from './DeviceSelectModal';

const fixtures = vi.hoisted(() => ({
  selectedSns: [] as string[],
  deviceListCalls: [] as Array<Record<string, unknown>>,
}));

function makeDevice(sn: string, productClass = 'FAP/BU1810'): Device {
  return {
    id: sn,
    sn,
    isOnline: true,
    productName: 'BM product',
    productClass,
    groupName: 'default',
  } as Device;
}

vi.mock('@core/hooks/api/useDevices', () => ({
  useDeviceList: (params: { snList?: string[] }) => {
    fixtures.deviceListCalls.push(params as Record<string, unknown>);
    const sns = params.snList ?? fixtures.selectedSns;
    return {
      data: { items: sns.map((sn) => makeDevice(sn)), total: sns.length },
      isFetching: false,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useDeviceList>;
  },
}));

vi.mock('@core/hooks/api/useProducts', () => ({
  useProductList: () => ({ data: { items: [{ id: 'product-1', name: 'BM product' }] } }),
}));

vi.mock('@core/hooks/api/useSystem', () => ({
  useDictionaryBatch: () => ({ data: { product_class: { sysDictionaryDetails: [] } } }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string, values?: Record<string, unknown>) =>
    values ? `${id}:${JSON.stringify(values)}` : id,
}));

describe('DeviceSelectModal', () => {
  it('keeps all selected devices after product scope validation and returns productClass', async () => {
    const selectedSns = Array.from({ length: 250 }, (_, index) => `sn-${index + 1}`);
    fixtures.selectedSns = selectedSns;
    fixtures.deviceListCalls = [];
    const onConfirm = vi.fn();

    render(
      <App>
        <DeviceSelectModal
          open
          value={selectedSns}
          onCancel={() => undefined}
          onConfirm={onConfirm}
        />
      </App>,
    );

    fireEvent.click(await screen.findByRole('button', { name: /mml\.consoleV2\.deviceSelect\.okText/ }));

    expect(onConfirm).toHaveBeenCalledWith(selectedSns, 'product-1', 'FAP/BU1810');
    expect(fixtures.deviceListCalls).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ snList: selectedSns, pageSize: selectedSns.length }),
      ]),
    );
  });
});
