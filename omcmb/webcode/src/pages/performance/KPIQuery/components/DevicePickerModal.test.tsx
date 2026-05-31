/**
 * 制式联动锁定（T-0188）：DevicePickerModal 的 technology 入参契约。
 *
 * 验证：
 *   - 传 technology 时，networkType 按制式透传到 useDeviceList（按制式过滤设备）。
 *   - 不传 technology 时，networkType 为 undefined（列全部设备，向后兼容 KPIQuery）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import { App } from 'antd';

// 捕获 useDeviceList 收到的 params。
const useDeviceListSpy = vi.fn();

vi.mock('@core/hooks/api/useDevices', () => ({
  useDeviceList: (params: unknown, options: unknown) => {
    useDeviceListSpy(params, options);
    return { data: { items: [], total: 0 }, isLoading: false };
  },
}));

import DevicePickerModal from './DevicePickerModal';

function renderModal(props: Partial<React.ComponentProps<typeof DevicePickerModal>> = {}) {
  return render(
    <App>
      <DevicePickerModal open onClose={() => {}} onConfirm={() => {}} {...props} />
    </App>,
  );
}

describe('DevicePickerModal 制式联动', () => {
  beforeEach(() => {
    useDeviceListSpy.mockClear();
  });

  it('传 technology 时按制式过滤（networkType 透传 useDeviceList）', () => {
    renderModal({ technology: 'nr' });
    const lastParams = useDeviceListSpy.mock.calls.at(-1)?.[0] as { networkType?: string };
    expect(lastParams.networkType).toBe('nr');
  });

  it('不传 technology 时 networkType 为 undefined（向后兼容，列全部设备）', () => {
    renderModal();
    const lastParams = useDeviceListSpy.mock.calls.at(-1)?.[0] as { networkType?: string };
    expect(lastParams.networkType).toBeUndefined();
  });
});
