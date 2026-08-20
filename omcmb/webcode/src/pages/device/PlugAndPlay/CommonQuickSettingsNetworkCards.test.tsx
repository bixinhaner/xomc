import { Form } from 'antd';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { describe, expect, it, vi } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import CommonQuickSettingsNetworkCards from './CommonQuickSettingsNetworkCards';

const mockGroups: QuickSettingsGroup[] = [
  {
    id: 'gnb-network-interface',
    titleZh: 'WAN(VLAN)/LAN',
    titleEn: 'WAN(VLAN)/LAN',
    multiInstance: true,
    maxInstances: 16,
    objectPath: 'Device.Ethernet.Interface.{i}.',
    params: [
      { name: 'InterfaceType', titleZh: '接口类型', titleEn: 'Interface Type', leaf: 'interfaceType' },
    ],
  },
  {
    id: 'gnb-interface-ipv4',
    titleZh: '接口直连 IPv4',
    titleEn: 'Direct Interface IPv4',
    multiInstance: true,
    parentSelector: 'gnb-network-interface',
    maxInstances: 8,
    objectPath: 'Device.Ethernet.Interface.{i}.IPv4Address.{i}.',
    params: [
      { name: 'AddressingType', titleZh: 'IP 类型', titleEn: 'IP Type', leaf: 'AddressingType' },
      { name: 'IPAddress', titleZh: 'IP 地址', titleEn: 'IP Address', leaf: 'IPAddress' },
      { name: 'SubnetMask', titleZh: '子网掩码', titleEn: 'Subnet Mask', leaf: 'SubnetMask' },
    ],
  },
  {
    id: 'gnb-interface-ipv6',
    titleZh: 'IPv6 地址',
    titleEn: 'IPv6 Address',
    multiInstance: true,
    parentSelector: 'gnb-network-interface',
    maxInstances: 8,
    objectPath: 'Device.Ethernet.Interface.{i}.IPv6Address.{i}.',
    params: [
      { name: 'Origin', titleZh: 'IP 类型', titleEn: 'IP Type', leaf: 'Origin' },
      { name: 'IPAddress', titleZh: 'IP 地址', titleEn: 'IP Address', leaf: 'IPAddress' },
    ],
  },
];

vi.mock('@core/services/api/quicksettingsApi', () => ({
  quicksettingsApi: {
    getGroupsByParamModel: vi.fn(async () => ({ paramModel: 'BaiBNQ', groups: mockGroups })),
  },
}));

function NetworkHarness({
  readOnly = false,
  onRequestEdit,
}: {
  readOnly?: boolean;
  onRequestEdit?: () => void;
}) {
  const [form] = Form.useForm();
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <Form
        form={form}
        initialValues={{
          sheetParameters: {
            INTERFACE: [{
              'IP 类型': 'Static',
              'IP 地址': '192.0.2.10',
              '子网掩码': '255.255.255.0',
            }],
          },
          networkObjectInstances: {
            'gnb-network-interface@root': [{}],
          },
          networkParameterValues: {
            'Device.Ethernet.Interface.1.IPv4Address.1.AddressingType': 'Static',
            'Device.Ethernet.Interface.1.IPv4Address.1.IPAddress': '192.0.2.10',
            'Device.Ethernet.Interface.1.IPv4Address.1.SubnetMask': '255.255.255.0',
            'Device.Ethernet.Interface.1.IPv6Address.1.Origin': 'DHCPv6',
            'Device.Ethernet.Interface.1.IPv6Address.1.IPAddress': '2001:db8::10',
          },
        }}
      >
        <CommonQuickSettingsNetworkCards paramModelName="BaiBNQ" readOnly={readOnly} onRequestEdit={onRequestEdit} />
      </Form>
    </IntlProvider>
  );
}

describe('CommonQuickSettingsNetworkCards', () => {
  it('keeps each multi-instance network parameter on its own TR path', async () => {
    const onRequestEdit = vi.fn();
    render(<NetworkHarness onRequestEdit={onRequestEdit} />);

    await screen.findByText('WAN(VLAN)/LAN 1');
    fireEvent.click(screen.getByText('接口直连 IPv4').closest('.ant-collapse-header')!);
    fireEvent.click(screen.getByText('IPv6 地址').closest('.ant-collapse-header')!);

    expect(screen.getByDisplayValue('192.0.2.10')).toBeInTheDocument();
    expect(screen.getByDisplayValue('2001:db8::10')).toBeInTheDocument();

    fireEvent.change(screen.getByDisplayValue('255.255.255.0'), {
      target: { value: '255.255.0.0' },
    });

    await waitFor(() => {
      expect(screen.getByDisplayValue('192.0.2.10')).toBeInTheDocument();
      expect(screen.getByDisplayValue('2001:db8::10')).toBeInTheDocument();
      expect(screen.getByDisplayValue('255.255.0.0')).toBeInTheDocument();
    });
    expect(onRequestEdit).toHaveBeenCalled();
  });

  it('renders saved child network instances from TR path values in read-only mode', async () => {
    render(<NetworkHarness readOnly />);

    await screen.findByText('WAN(VLAN)/LAN 1');
    fireEvent.click(screen.getByText('接口直连 IPv4').closest('.ant-collapse-header')!);
    fireEvent.click(screen.getByText('IPv6 地址').closest('.ant-collapse-header')!);

    await waitFor(() => {
      expect(screen.getByDisplayValue('192.0.2.10')).toBeInTheDocument();
      expect(screen.getByDisplayValue('255.255.255.0')).toBeInTheDocument();
      expect(screen.getByDisplayValue('2001:db8::10')).toBeInTheDocument();
    });
    expect(screen.getAllByText('1/8')).toHaveLength(2);
  });

  it('requests edit mode when a read-only network parameter receives focus', async () => {
    const onRequestEdit = vi.fn();
    const { container } = render(<NetworkHarness readOnly onRequestEdit={onRequestEdit} />);

    await screen.findByText('WAN(VLAN)/LAN 1');
    fireEvent.focus(container.querySelector('input')!);

    expect(onRequestEdit).toHaveBeenCalledTimes(1);
  });
});
