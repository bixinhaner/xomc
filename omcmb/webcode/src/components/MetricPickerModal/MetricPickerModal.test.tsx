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
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';

// 组件内用 useIntl 取文案，测试渲染必须套 IntlProvider（真实 zh-CN 语料）。
function wrapIntl(node: React.ReactElement) {
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      {node}
    </IntlProvider>
  );
}

function renderModal(props: Partial<React.ComponentProps<typeof MetricPickerModal>> = {}) {
  return render(
    wrapIntl(<MetricPickerModal open onClose={() => {}} onConfirm={() => {}} {...props} />),
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
      wrapIntl(<MetricPickerModal open onClose={() => {}} onConfirm={() => {}} lockDeviceType initialDeviceType="ENB" />),
    );
    expect(useIndicatorListSpy.mock.calls.at(-1)?.[0]).toBe('ENB');
    // 关闭（不卸载组件）→ 换制式 → 重开
    rerender(
      wrapIntl(<MetricPickerModal open={false} onClose={() => {}} onConfirm={() => {}} lockDeviceType initialDeviceType="GNB" />),
    );
    rerender(
      wrapIntl(<MetricPickerModal open onClose={() => {}} onConfirm={() => {}} lockDeviceType initialDeviceType="GNB" />),
    );
    expect(useIndicatorListSpy.mock.calls.at(-1)?.[0]).toBe('GNB');
  });
});

describe('MetricPickerModal 已选回显', () => {
  beforeEach(() => {
    useIndicatorListSpy.mockClear();
  });

  // 回归：与制式同款「组件常驻不卸载」问题——内部 selected 仅首挂载赋值一次。
  // 编辑不同模板时关闭后用新 initialSelected 重开，「已选」面板必须回显最新模板的指标，
  // 而非停留在上次打开弹窗时的残留选择。
  it('关闭后以新 initialSelected 重开，「已选」回显最新入参而非上次残留', () => {
    const { rerender } = render(
      wrapIntl(
        <MetricPickerModal open onClose={() => {}} onConfirm={() => {}} initialSelected={['C000080007']} />,
      ),
    );
    // 首开：回显模板 A 的指标
    expect(screen.getByText('C000080007')).toBeTruthy();

    // 关闭（不卸载组件）→ 换成模板 B 的指标 → 重开
    rerender(
      wrapIntl(
        <MetricPickerModal open={false} onClose={() => {}} onConfirm={() => {}} initialSelected={['C000030170']} />,
      ),
    );
    rerender(
      wrapIntl(
        <MetricPickerModal open onClose={() => {}} onConfirm={() => {}} initialSelected={['C000030170']} />,
      ),
    );
    // 重开：必须回显模板 B 的指标，且不残留模板 A 的指标
    expect(screen.getByText('C000030170')).toBeTruthy();
    expect(screen.queryByText('C000080007')).toBeNull();
  });
});
