/**
 * 制式联动锁定（T-0188）：MetricPickerModal 的 lockDeviceType 入参契约。
 *
 * 验证：
 *   - lockDeviceType=true 时不渲染「设备类型」下拉，且 useIndicatorList 始终按 initialDeviceType 取数（锁死制式）。
 *   - 不传/false 时仍渲染「设备类型」下拉（向后兼容 KPIQuery 可切换行为）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';

// 捕获 useIndicatorList 收到的 deviceType。
const useIndicatorListSpy = vi.fn();

vi.mock('@core/hooks/api/useIndicatorsLibrary', () => {
  // 稳定引用：组件内有按 data 身份做的 render-phase 同步，mock 每次返回新对象会在多次
  // rerender 时把它放大成无限渲染（生产用真 React Query 数据稳定，不触发）。
  const STABLE = { data: { items: [], total: 0 }, isLoading: false };
  return {
    useIndicatorList: (deviceType: unknown, params: unknown) => {
      useIndicatorListSpy(deviceType, params);
      return STABLE;
    },
  };
});

import MetricPickerModal from './MetricPickerModal';

function renderModal(props: Partial<React.ComponentProps<typeof MetricPickerModal>> = {}) {
  return render(
    <MetricPickerModal open onClose={() => {}} onConfirm={() => {}} {...props} />,
  );
}

describe('MetricPickerModal 制式锁定', () => {
  beforeEach(() => {
    useIndicatorListSpy.mockClear();
  });

  it('lockDeviceType=true 时隐藏「设备类型」下拉，按 initialDeviceType 锁死取数', () => {
    renderModal({ lockDeviceType: true, initialDeviceType: 'GNB' });
    expect(screen.queryByText('设备类型：')).toBeNull();
    const lastDeviceType = useIndicatorListSpy.mock.calls.at(-1)?.[0];
    expect(lastDeviceType).toBe('GNB');
  });

  it('不传 lockDeviceType 时仍渲染「设备类型」下拉（向后兼容可切换）', () => {
    renderModal({ initialDeviceType: 'ENB' });
    expect(screen.getByText('设备类型：')).toBeTruthy();
    const lastDeviceType = useIndicatorListSpy.mock.calls.at(-1)?.[0];
    expect(lastDeviceType).toBe('ENB');
  });

  // 回归：本组件在调用页常驻不卸载（destroyOnHidden 只销毁弹窗 DOM、不重挂载本组件），
  // 故必须在「关闭后用新制式重新打开」时按最新 initialDeviceType 取数，而非停留在首挂载制式。
  // 这是首版单测（每次全新挂载）验不出、被运行栈 playwright 抓出的盲区。
  it('锁定态：关闭后以新制式重开，取数按最新 initialDeviceType 切换（修组件常驻不重挂载）', () => {
    const { rerender } = render(
      <MetricPickerModal open onClose={() => {}} onConfirm={() => {}} lockDeviceType initialDeviceType="ENB" />,
    );
    expect(useIndicatorListSpy.mock.calls.at(-1)?.[0]).toBe('ENB');
    // 关闭（不卸载组件）→ 换制式 → 重开
    rerender(
      <MetricPickerModal open={false} onClose={() => {}} onConfirm={() => {}} lockDeviceType initialDeviceType="GNB" />,
    );
    rerender(
      <MetricPickerModal open onClose={() => {}} onConfirm={() => {}} lockDeviceType initialDeviceType="GNB" />,
    );
    expect(useIndicatorListSpy.mock.calls.at(-1)?.[0]).toBe('GNB');
  });
});
