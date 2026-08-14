/**
 * #236 规则名必填星标：AlarmRuleDrawer 的「规则名称」字段必填 + 红色星标。
 * AntD 仅在 Form.Item 含 required 规则时给标签加 ant-form-item-required（红色星标），
 * 故断言该 class 即等价验证了「规则名称必填」这一改动。
 */
import { beforeEach, describe, it, expect, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';
import type { AlarmRule } from '@core/types/alarm';
import type { Device, DeviceListResponse } from '@core/types/device';

const deviceHooks = vi.hoisted(() => ({
  useDeviceList: vi.fn(() => ({ data: { items: [], total: 0 }, isLoading: false })),
  useDeviceGroups: vi.fn(() => ({ data: { groups: [] }, isLoading: false })),
  useDevicesByIds: vi.fn(),
}));

const emptyDeviceQueries: unknown[] = [];
const alarmDefinitionHooks = vi.hoisted(() => ({
  useAllAlarmDefinitions: vi.fn(() => ({ data: { items: [] }, isLoading: false })),
}));

vi.mock('@core/hooks/api/useDevices', () => ({
  useDeviceList: deviceHooks.useDeviceList,
  useDeviceGroups: deviceHooks.useDeviceGroups,
  useDevicesByIds: deviceHooks.useDevicesByIds,
}));
vi.mock('@core/hooks/api/useAlarmDefinitions', () => ({
  useAllAlarmDefinitions: alarmDefinitionHooks.useAllAlarmDefinitions,
}));

import AlarmRuleDrawer from './AlarmRuleDrawer';

const visibleDevice = {
  id: 'visible-device',
  sn: 'SN-visible-device',
  name: 'visible-device',
  networkType: 'lte',
  connStatus: 'online',
} as Device;

function makeDeviceListResponse(items: Device[], total = items.length): DeviceListResponse {
  return {
    items,
    total,
    page: 1,
    pageSize: 5,
    stats: {
      total,
      online_count: items.filter((device) => device.isOnline).length,
      offline_count: items.filter((device) => !device.isOnline).length,
      alarmed: 0,
    },
  };
}

function makeDevice(index: number): Device {
  return {
    id: `device-${index}`,
    sn: `SN-device-${index}`,
    name: `device-${index}`,
    networkType: 'lte',
    connStatus: 'online',
  } as Device;
}

function makeRule(
  selectedDevices: string[],
  ruleType = 'ignore',
  emailRecipients: string[] = [],
): AlarmRule {
  return {
    id: 'rule-1',
    ruleName: 'rule-1',
    ruleType,
    severity: 'warning',
    enabled: false,
    conditions: [
      { field: 'device_id', operator: 'contains', value: selectedDevices },
      { field: 'alarm_identifier', operator: 'contains', value: ['30000'] },
    ],
    actions: [],
    emailRecipients,
    createTime: '2026-06-18T00:00:00Z',
    updateTime: '2026-06-18T00:00:00Z',
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  deviceHooks.useDeviceList.mockReturnValue({ data: makeDeviceListResponse([]), isLoading: false });
  deviceHooks.useDeviceGroups.mockReturnValue({ data: { groups: [] }, isLoading: false });
  deviceHooks.useDevicesByIds.mockReturnValue(emptyDeviceQueries);
  alarmDefinitionHooks.useAllAlarmDefinitions.mockReturnValue({ data: { items: [] }, isLoading: false });
});

function renderDrawer(props: Partial<React.ComponentProps<typeof AlarmRuleDrawer>> = {}) {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <App>
        <AlarmRuleDrawer open mode="add" onClose={() => {}} onSubmit={async () => {}} {...props} />
      </App>
    </IntlProvider>,
  );
}

describe('AlarmRuleDrawer 规则名必填 (#236)', () => {
  it('「规则名称」字段渲染必填星标', () => {
    renderDrawer();
    const label = screen.getByText('规则名称');
    expect(label.tagName.toLowerCase()).toBe('label');
    expect(label.className).toContain('ant-form-item-required');
  });

  it('「规则名称」输入框存在且 maxLength 限制为 100', () => {
    renderDrawer();
    const input = screen.getByPlaceholderText('请输入规则名称');
    expect(input).toHaveAttribute('maxlength', '100');
  });

  it('设备列表按表格分页参数请求后端，避免只加载前 100 条', () => {
    renderDrawer();

    expect(deviceHooks.useDeviceList).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 5,
    });
  });

  it('编辑规则时不会把当前页缺失的已选设备插入设备表', () => {
    deviceHooks.useDeviceList.mockReturnValue({
      data: makeDeviceListResponse([visibleDevice]),
      isLoading: false,
    });

    renderDrawer({
      mode: 'edit',
      rule: makeRule(['visible-device', 'missing-device']),
    });

    expect(screen.getByText('SN-visible-device')).toBeInTheDocument();
    expect(screen.queryByText('SN-missing-device')).not.toBeInTheDocument();
  });

  it('已选设备保持原列表顺序，不会被移动到第一页', () => {
    deviceHooks.useDeviceList.mockReturnValue({
      data: makeDeviceListResponse([1, 2, 3, 4, 5].map(makeDevice), 6),
      isLoading: false,
    });

    renderDrawer({
      mode: 'edit',
      rule: makeRule(['device-6']),
    });

    expect(screen.getByText('SN-device-1')).toBeInTheDocument();
    expect(screen.getByText('SN-device-5')).toBeInTheDocument();
    expect(screen.queryByText('SN-device-6')).not.toBeInTheDocument();
  });

  it('设备列表展示总数、批量勾选入口和已选设备按钮', () => {
    deviceHooks.useDeviceList.mockReturnValue({
      data: makeDeviceListResponse([1, 2, 3].map(makeDevice), 12),
      isLoading: false,
    });

    renderDrawer();

    expect(screen.getByText('批量勾选当前页')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '已选 0 台' })).toBeInTheDocument();
    expect(screen.getByText('共 12 台')).toBeInTheDocument();
  }, 10000);

  it('已选设备弹窗显示当前页外的已选设备明细', () => {
    deviceHooks.useDeviceList.mockReturnValue({
      data: makeDeviceListResponse([visibleDevice], 2),
      isLoading: false,
    });
    deviceHooks.useDevicesByIds.mockImplementation((ids: string[]) => ids.map((id) => ({
      data: id === 'missing-device'
        ? {
            id: 'missing-device',
            sn: 'SN-missing-device',
            name: 'missing-device',
            networkType: 'nr',
            connStatus: 'online',
          } as Device
        : undefined,
      isLoading: false,
    })));

    renderDrawer({
      mode: 'edit',
      rule: makeRule(['visible-device', 'missing-device']),
    });

    fireEvent.click(screen.getByRole('button', { name: '已选 2 台' }));

    expect(screen.getByText('SN-missing-device')).toBeInTheDocument();
    expect(screen.getAllByText('已选 2 台')).toHaveLength(2);
    expect(screen.getAllByText('共 2 台')).not.toHaveLength(0);
  }, 12000);

  it('邮件通知规则显示并提交回填的收件人', async () => {
    const onSubmit = vi.fn(async () => {});
    renderDrawer({
      mode: 'edit',
      rule: makeRule([], 'notify_email', ['noc@example.com']),
      onSubmit,
    });

    expect(screen.getByText('邮件收件人')).toBeInTheDocument();
    expect(screen.getByText(/最多 50 个/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({
      emailRecipients: ['noc@example.com'],
    })));
  });

  it('编辑规则时回填并提交生效时间范围', async () => {
    const onSubmit = vi.fn(async () => {});
    const rule = makeRule([]);
    rule.effectiveStart = '2026-08-12T01:00:00.000Z';
    rule.effectiveEnd = '2026-08-12T02:00:00.000Z';
    renderDrawer({ mode: 'edit', rule, onSubmit });

    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({
      timeRange: ['2026-08-12T01:00:00.000Z', '2026-08-12T02:00:00.000Z'],
    })));
  });

  it('已选告警摘要使用主题容器并支持一键清空', () => {
    alarmDefinitionHooks.useAllAlarmDefinitions.mockReturnValue({
      data: {
        items: [{
          identifier: '30000',
          cnProbableCause: '本地验证告警',
          severityCode: 31001,
          eventType: 'communication',
        }],
      },
      isLoading: false,
    });

    renderDrawer({ mode: 'edit', rule: makeRule([]) });

    const summary = screen.getByTestId('selected-alarm-summary');
    expect(summary).toBeInTheDocument();
    expect(summary).toHaveTextContent('30000');
    fireEvent.click(screen.getByRole('button', { name: '清空' }));
    expect(screen.queryByTestId('selected-alarm-summary')).not.toBeInTheDocument();
  });
});
