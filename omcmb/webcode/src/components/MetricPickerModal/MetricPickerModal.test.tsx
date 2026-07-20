/**
 * 制式联动锁定（T-0188）：MetricPickerModal 的 lockDeviceType 入参契约。
 *
 * 验证：
 *   - lockDeviceType=true 时不渲染「设备类型」下拉，且 useIndicatorList 始终按 initialDeviceType 取数（锁死制式）。
 *   - 不传/false 时仍渲染「设备类型」下拉（向后兼容 KPIQuery 可切换行为）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';

// 捕获 useIndicatorList 收到的 deviceType。
const { useIndicatorListSpy, indicatorListData, allIndicatorsData } = vi.hoisted(() => ({
  useIndicatorListSpy: vi.fn(),
  indicatorListData: { items: [] as unknown[], total: 0 },
  allIndicatorsData: { items: [] as unknown[], total: 0 },
}));

vi.mock('@core/hooks/api/useIndicatorsLibrary', () => {
  // 稳定引用：组件内有按 data 身份做的 render-phase 同步，mock 每次返回新对象会在多次
  // rerender 时把它放大成无限渲染（生产用真 React Query 数据稳定，不触发）。
  const STABLE = { data: indicatorListData, isLoading: false };
  const ALL_STABLE = { data: allIndicatorsData, isLoading: false };
  return {
    useIndicatorList: (deviceType: unknown, params: unknown) => {
      useIndicatorListSpy(deviceType, params);
      return STABLE;
    },
    useAllIndicators: () => ALL_STABLE,
  };
});

import MetricPickerModal from './MetricPickerModal';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';

// 组件内用 useIntl 取文案，测试渲染必须套 IntlProvider（真实 zh-CN 语料）。
function wrapIntl(node: React.ReactElement) {
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>{node}</App>
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
    indicatorListData.items = [];
    indicatorListData.total = 0;
    allIndicatorsData.items = [];
    allIndicatorsData.total = 0;
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
    indicatorListData.items = [];
    indicatorListData.total = 0;
    allIndicatorsData.items = [];
    allIndicatorsData.total = 0;
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

describe('MetricPickerModal 搜索状态', () => {
  beforeEach(() => {
    useIndicatorListSpy.mockClear();
    indicatorListData.items = [];
    indicatorListData.total = 0;
    allIndicatorsData.items = [];
    allIndicatorsData.total = 0;
  });

  it('关闭后重开会清空上次搜索词并回到第一页', () => {
    const { rerender } = render(
      wrapIntl(<MetricPickerModal open onClose={() => {}} onConfirm={() => {}} />),
    );
    const input = screen.getByPlaceholderText('按指标路径 / 中文名 搜索') as HTMLInputElement;

    fireEvent.change(input, { target: { value: 'availability' } });
    fireEvent.click(screen.getByRole('button', { name: /搜\s*索/ }));

    expect(useIndicatorListSpy.mock.calls.at(-1)?.[1]).toMatchObject({
      keyword: 'availability',
      page: 1,
    });

    rerender(
      wrapIntl(<MetricPickerModal open={false} onClose={() => {}} onConfirm={() => {}} />),
    );
    rerender(
      wrapIntl(<MetricPickerModal open onClose={() => {}} onConfirm={() => {}} />),
    );

    const reopenedInput = screen.getByPlaceholderText('按指标路径 / 中文名 搜索') as HTMLInputElement;
    expect(reopenedInput.value).toBe('');
    expect(useIndicatorListSpy.mock.calls.at(-1)?.[1]).toMatchObject({
      keyword: undefined,
      page: 1,
    });
  });
});

describe('MetricPickerModal 选择数量限制', () => {
  beforeEach(() => {
    useIndicatorListSpy.mockClear();
    indicatorListData.items = [];
    indicatorListData.total = 0;
    allIndicatorsData.items = [];
    allIndicatorsData.total = 0;
  });

  it('默认不限制选择数量，由复用页面自行决定上限', () => {
    const onConfirm = vi.fn();
    renderModal({ initialSelected: Array.from({ length: 51 }, (_, i) => `K-${i + 1}`), onConfirm });

    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).toHaveBeenCalledWith(
      Array.from({ length: 51 }, (_, i) => `K-${i + 1}`),
      expect.any(Object),
    );
  });

  it('已选指标正好 50 个时，点击确定可以提交', () => {
    const onConfirm = vi.fn();
    const selected = Array.from({ length: 50 }, (_, i) => `K-${i + 1}`);
    renderModal({ initialSelected: selected, maxSelected: 50, onConfirm });

    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).toHaveBeenCalledWith(selected, expect.any(Object));
  });

  it('已选指标超过 50 个时，点击确定不提交', () => {
    const onConfirm = vi.fn();
    renderModal({ initialSelected: Array.from({ length: 51 }, (_, i) => `K-${i + 1}`), maxSelected: 50, onConfirm });

    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).not.toHaveBeenCalled();
  });
});

describe('MetricPickerModal 批量输入指标 ID', () => {
  beforeEach(() => {
    useIndicatorListSpy.mockClear();
    indicatorListData.items = [];
    indicatorListData.total = 0;
    allIndicatorsData.items = [
      {
        id: 'C000060011',
        name: 'rrc_att',
        cnName: 'RRC请求次数',
        enName: 'RRC Attempts',
        isCounter: true,
        deviceType: 'ENB',
      },
      {
        id: 'K900010002',
        name: 'availability',
        cnName: '可用率',
        enName: 'Availability',
        isCounter: false,
        deviceType: 'ENB',
      },
    ];
    allIndicatorsData.total = 2;
  });

  it('默认不显示批量输入入口，由调用页显式开启', () => {
    renderModal();

    expect(screen.queryByRole('button', { name: /批量输入/ })).toBeNull();
  });

  it('只把全量真实指标库里存在的 ID 加入已选，并忽略不存在 ID', async () => {
    const onConfirm = vi.fn();
    renderModal({ initialSelected: ['EXISTING'], onConfirm, enableBatchInput: true });

    fireEvent.click(screen.getByRole('button', { name: /批量输入/ }));
    const input = await screen.findByPlaceholderText(/K000000001/);
    fireEvent.change(input, {
      target: { value: 'K900010002，UNKNOWN\nC000060011 K900010002' },
    });
    fireEvent.click(screen.getByRole('button', { name: /加入已选/ }));
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).toHaveBeenCalledWith(
      ['EXISTING', 'K900010002', 'C000060011'],
      expect.objectContaining({
        K900010002: '可用率',
        C000060011: 'RRC请求次数',
      }),
    );
  });

  it('全部 ID 不存在时不改变原已选', async () => {
    const onConfirm = vi.fn();
    renderModal({ initialSelected: ['EXISTING'], onConfirm, enableBatchInput: true });

    fireEvent.click(screen.getByRole('button', { name: /批量输入/ }));
    const input = await screen.findByPlaceholderText(/K000000001/);
    fireEvent.change(input, {
      target: { value: 'UNKNOWN-1 UNKNOWN-2' },
    });
    fireEvent.click(screen.getByRole('button', { name: /加入已选/ }));
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).toHaveBeenCalledWith(['EXISTING'], expect.any(Object));
  });

  it('批量输入同样受 maxSelected 上限限制', async () => {
    const onConfirm = vi.fn();
    renderModal({ initialSelected: ['EXISTING'], maxSelected: 2, onConfirm, enableBatchInput: true });

    fireEvent.click(screen.getByRole('button', { name: /批量输入/ }));
    const input = await screen.findByPlaceholderText(/K000000001/);
    fireEvent.change(input, {
      target: { value: 'K900010002 C000060011' },
    });
    fireEvent.click(screen.getByRole('button', { name: /加入已选/ }));
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    expect(onConfirm).toHaveBeenCalledWith(['EXISTING', 'K900010002'], expect.any(Object));
  });
});

describe('MetricPickerModal 指标级别列', () => {
  beforeEach(() => {
    useIndicatorListSpy.mockClear();
    indicatorListData.items = [
      {
        id: 'K-BOTH',
        name: 'rrc_sr',
        cnName: 'RRC连接成功率',
        enName: 'RRC Success Rate',
        indicatorLevel: 'both',
        isCounter: false,
        deviceType: 'ENB',
      },
    ];
    indicatorListData.total = 1;
    allIndicatorsData.items = [];
    allIndicatorsData.total = 0;
  });

  it('ENB/GSM 指标选择列表显示“指标级别”列并把 both 显示为“设备级 / PLMN级”', () => {
    renderModal({ initialDeviceType: 'ENB' });

    expect(screen.getAllByText('指标级别').length).toBeGreaterThan(0);
    expect(screen.getByText('设备级 / PLMN级')).toBeTruthy();
  });

  it('GNB 指标选择列表暂不显示“指标级别”列', () => {
    renderModal({ initialDeviceType: 'GNB' });

    expect(screen.queryByText('指标级别')).toBeNull();
  });
});
