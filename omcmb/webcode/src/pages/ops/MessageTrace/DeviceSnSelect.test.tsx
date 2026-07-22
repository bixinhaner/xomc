import { render, screen } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { describe, expect, it, vi } from 'vitest';
import enUS from '@core/i18n/en-US';
import zhCN from '@core/i18n/zh-CN';
import { DeviceSnSelect } from './index';

vi.mock('@core/hooks/api/useDevices', () => ({
  useDeviceList: () => ({ data: { items: [] }, isFetching: false }),
}));

describe('DeviceSnSelect', () => {
  it('shows the Device SN search hint in English when the page locale is English', () => {
    render(
      <IntlProvider locale="en-US" defaultLocale="en-US" messages={enUS}>
        <DeviceSnSelect />
      </IntlProvider>,
    );

    expect(
      screen.getByText(
        'Enter an SN or site name to search for a device (full SN paste supported)',
      ),
    ).toBeInTheDocument();
  });

  it('defines all device-search hints for both supported locales', () => {
    expect(enUS).toMatchObject({
      'trace.field.deviceSn.noInput': 'Enter an SN or site name to search for a device',
      'trace.field.deviceSn.searching': 'Searching...',
      'trace.field.deviceSn.notFound': 'No matching devices',
    });
    expect(zhCN).toMatchObject({
      'trace.field.deviceSn.noInput': '输入 SN / 站点名搜索设备',
      'trace.field.deviceSn.searching': '搜索中...',
      'trace.field.deviceSn.notFound': '无匹配设备',
    });
  });
});
