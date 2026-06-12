/**
 * #236 规则名必填星标：AlarmRuleDrawer 的「规则名称」字段必填 + 红色星标。
 * AntD 仅在 Form.Item 含 required 规则时给标签加 ant-form-item-required（红色星标），
 * 故断言该 class 即等价验证了「规则名称必填」这一改动。
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';

vi.mock('@core/hooks/api/useDevices', () => ({
  useDeviceList: () => ({ data: { items: [], total: 0 }, isLoading: false }),
  useDeviceGroups: () => ({ data: [], isLoading: false }),
  useDevicesByIds: () => [],
}));
vi.mock('@core/hooks/api/useAlarmDefinitions', () => ({
  useAllAlarmDefinitions: () => ({ data: [], isLoading: false }),
}));

import AlarmRuleDrawer from './AlarmRuleDrawer';

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
});
