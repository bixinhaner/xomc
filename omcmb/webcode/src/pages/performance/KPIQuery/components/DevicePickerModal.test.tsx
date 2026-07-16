/**
 * 制式联动锁定（T-0188）：DevicePickerModal 的 technology 入参契约。
 *
 * 验证：
 *   - 传 technology 时，networkType 按制式透传到 useDeviceList（按制式过滤设备）。
 *   - 不传 technology 时，networkType 为 undefined（列全部设备，向后兼容 KPIQuery）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';

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
  // 组件内用 useIntl 取文案，测试渲染必须套 IntlProvider（真实 zh-CN 语料）。
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <DevicePickerModal open onClose={() => {}} onConfirm={() => {}} {...props} />
      </App>
    </IntlProvider>,
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

describe('DevicePickerModal 已选回显', () => {
  beforeEach(() => {
    useDeviceListSpy.mockClear();
  });

  // 回归：与制式同款「组件常驻不卸载」问题——内部 selected 仅首挂载赋值一次。
  // 编辑不同模板时关闭后用新 initialSelected 重开，「已选」面板必须回显最新模板的设备，
  // 而非停留在上次打开弹窗时的残留选择。
  it('关闭后以新 initialSelected 重开，「已选」回显最新入参而非上次残留', () => {
    const wrap = (node: React.ReactElement) => (
      <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
        <App>{node}</App>
      </IntlProvider>
    );
    const { rerender } = render(
      wrap(<DevicePickerModal open onClose={() => {}} onConfirm={() => {}} initialSelected={['SN-AAA']} />),
    );
    // 首开：回显模板 A 的设备
    expect(screen.getByText('SN-AAA')).toBeTruthy();

    // 关闭（不卸载组件）→ 换成模板 B 的设备 → 重开
    rerender(
      wrap(<DevicePickerModal open={false} onClose={() => {}} onConfirm={() => {}} initialSelected={['SN-BBB']} />),
    );
    rerender(
      wrap(<DevicePickerModal open onClose={() => {}} onConfirm={() => {}} initialSelected={['SN-BBB']} />),
    );
    // 重开：必须回显模板 B 的设备，且不残留模板 A 的设备
    expect(screen.getByText('SN-BBB')).toBeTruthy();
    expect(screen.queryByText('SN-AAA')).toBeNull();
  });
});

describe('DevicePickerModal 选择数量限制', () => {
  beforeEach(() => {
    useDeviceListSpy.mockClear();
  });

  it('默认不限制选择数量，由复用页面自行决定上限', () => {
    const onConfirm = vi.fn();
    renderModal({ initialSelected: Array.from({ length: 51 }, (_, i) => `SN-${i + 1}`), onConfirm });

    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).toHaveBeenCalledWith(Array.from({ length: 51 }, (_, i) => `SN-${i + 1}`));
  });

  it('已选设备正好 50 个时，点击确定可以提交', () => {
    const onConfirm = vi.fn();
    const selected = Array.from({ length: 50 }, (_, i) => `SN-${i + 1}`);
    renderModal({ initialSelected: selected, maxSelected: 50, onConfirm });

    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).toHaveBeenCalledWith(selected);
  });

  it('已选设备超过 50 个时，点击确定不提交', () => {
    const onConfirm = vi.fn();
    renderModal({ initialSelected: Array.from({ length: 51 }, (_, i) => `SN-${i + 1}`), maxSelected: 50, onConfirm });

    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).not.toHaveBeenCalled();
  });
});
