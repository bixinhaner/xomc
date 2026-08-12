import { describe, expect, it } from 'vitest';
import type { ParameterSchemaItem } from '@core/types/deviceParameter';
import { buildInterfaceBindingOptions } from '../interfaceBindingOptions';

const parameter = (path: string, currentValue: string): ParameterSchemaItem => ({
  path,
  currentValue,
  type: 'string',
  writable: true,
});

describe('buildInterfaceBindingOptions', () => {
  it('combines physical and VLAN IPv4 instances into binding labels', () => {
    const options = buildInterfaceBindingOptions([
      parameter('Device.Ethernet.Interface.1.Name', 'BH1'),
      parameter('Device.Ethernet.Interface.1.IPv4Address.1.IPAddress', '172.19.9.217'),
      parameter('Device.Ethernet.Interface.1.IPv4Address.2.IPAddress', '192.168.12.217'),
      parameter('Device.Ethernet.Interface.1.IPv4Address.3.IPAddress', '172.19.3.157'),
      parameter('Device.Ethernet.Interface.1.VlanInterface.1.Name', 'vlan12'),
      parameter('Device.Ethernet.Interface.1.VlanInterface.1.IPv4Address.1.IPAddress', '10.0.12.1'),
    ]);

    expect(options).toEqual([
      { value: 'Device.Ethernet.Interface.1.IPv4Address.3.IPAddress', label: 'BH1.IPv4:3' },
      { value: 'Device.Ethernet.Interface.1.IPv4Address.2.IPAddress', label: 'BH1.IPv4:2' },
      { value: 'Device.Ethernet.Interface.1.IPv4Address.1.IPAddress', label: 'BH1.IPv4:1' },
      { value: 'Device.Ethernet.Interface.1.VlanInterface.1.IPv4Address.1.IPAddress', label: 'BH1.vlan12:1' },
    ]);
  });

  it('uses stable fallback names when the device omits a Name leaf', () => {
    expect(buildInterfaceBindingOptions([
      parameter('Device.Ethernet.Interface.2.VlanInterface.4.IPv4Address.2.IPAddress', '10.0.0.2'),
    ])).toEqual([
      {
        value: 'Device.Ethernet.Interface.2.VlanInterface.4.IPv4Address.2.IPAddress',
        label: 'Interface.2.VLAN.4:2',
      },
    ]);
  });

  it('only includes addresses belonging to the configured WAN interface', () => {
    expect(buildInterfaceBindingOptions([
      parameter('Device.Ethernet.Interface.1.Name', 'BH1'),
      parameter('Device.Ethernet.Interface.1.IPv4Address.1.IPAddress', '172.19.9.217'),
      parameter('Device.Ethernet.Interface.3.Name', 'ETH'),
      parameter('Device.Ethernet.Interface.3.IPv4Address.1.IPAddress', '192.168.1.1'),
    ], 'BH1')).toEqual([
      { value: 'Device.Ethernet.Interface.1.IPv4Address.1.IPAddress', label: 'BH1.IPv4:1' },
    ]);
  });
});
