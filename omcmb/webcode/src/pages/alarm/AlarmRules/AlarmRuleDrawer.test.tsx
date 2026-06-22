/**
 * #236 规则名必填星标：AlarmRuleDrawer 的「规则名称」字段必填 + 红色星标。
 * AntD 仅在 Form.Item 含 required 规则时给标签加 ant-form-item-required（红色星标），
 * 故断言该 class 即等价验证了「规则名称必填」这一改动。
 */
import { beforeEach, describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';
import type { AlarmRule } from '@core/types/alarm';
import type { Device } from '@core/types/device';

const deviceHooks = vi.hoisted(() => ({
  useDeviceList: vi.fn(() => ({ data: { items: [], total: 0 }, isLoading: false })),
  useDeviceGroups: vi.fn(() => ({ data: { groups: [] }, isLoading: false })),
  useDevicesByIds: vi.fn(() => []),
}));

vi.mock('@core/hooks/api/useDevices', () => ({
  useDeviceList: deviceHooks.useDeviceList,
  useDeviceGroups: deviceHooks.useDeviceGroups,
  useDevicesByIds: deviceHooks.useDevicesByIds,
}));
vi.mock('@core/hooks/api/useAlarmDefinitions', () => ({
  useAllAlarmDefinitions: () => ({ data: [], isLoading: false }),
}));

import AlarmRuleDrawer from './AlarmRuleDrawer';

const visibleDevice = {
  id: 'visible-device',
  sn: 'SN-visible-device',
  name: 'visible-device',
  networkType: 'lte',
  connStatus: 'online',
} as Device;

function makeDevice(index: number): Device {
  return {
    id: `device-${index}`,
    sn: `SN-device-${index}`,
    name: `device-${index}`,
    networkType: 'lte',
    connStatus: 'online',
  } as Device;
}

function makeRule(selectedDevices: string[]): AlarmRule {
  return {
    id: 'rule-1',
    ruleName: 'rule-1',
    ruleType: 'ignore',
    severity: 'warning',
    enabled: false,
    conditions: [
      { field: 'device_id', operator: 'contains', value: selectedDevices },
      { field: 'alarm_identifier', operator: 'contains', value: ['30000'] },
    ],
    actions: [],
    createTime: '2026-06-18T00:00:00Z',
    updateTime: '2026-06-18T00:00:00Z',
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  deviceHooks.useDeviceList.mockReturnValue({ data: { items: [], total: 0 }, isLoading: false });
  deviceHooks.useDeviceGroups.mockReturnValue({ data: { groups: [] }, isLoading: false });
  deviceHooks.useDevicesByIds.mockReturnValue([]);
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

  it('编辑规则时只补查当前设备列表缺失的已选设备', () => {
    deviceHooks.useDeviceList.mockReturnValue({
      data: { items: [visibleDevice], total: 1 },
      isLoading: false,
    });

    renderDrawer({
      mode: 'edit',
      rule: makeRule(['visible-device', 'missing-device']),
    });

    expect(deviceHooks.useDevicesByIds).toHaveBeenLastCalledWith(['missing-device']);
  });

  it('已选设备保持原列表顺序，不会被移动到第一页', () => {
    deviceHooks.useDeviceList.mockReturnValue({
      data: { items: [1, 2, 3, 4, 5, 6].map(makeDevice), total: 6 },
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
});
