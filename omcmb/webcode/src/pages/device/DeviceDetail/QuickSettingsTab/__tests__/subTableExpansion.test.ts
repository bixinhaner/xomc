import { describe, expect, it } from 'vitest';
import { rowsWithNestedInstances } from '../subTableExpansion';

describe('rowsWithNestedInstances', () => {
  it('expands only VLAN rows that contain IPv4 or IPv6 child instances', () => {
    const paths = [
      'Device.Ethernet.Interface.1.VlanInterface.1.Name',
      'Device.Ethernet.Interface.1.VlanInterface.1.IPv4Address.1.IPAddress',
      'Device.Ethernet.Interface.1.VlanInterface.2.Name',
      'Device.Ethernet.Interface.1.VlanInterface.3.IPv6Address.2.IPAddress',
    ];

    expect(rowsWithNestedInstances(
      paths,
      'Device.Ethernet.Interface.1.VlanInterface.',
      [1, 2, 3],
      ['IPv4Address', 'IPv6Address'],
    )).toEqual([1, 3]);
  });
});
