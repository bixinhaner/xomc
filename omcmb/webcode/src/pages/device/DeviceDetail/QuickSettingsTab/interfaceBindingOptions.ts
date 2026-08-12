import type { ParameterSchemaItem } from '@core/types/deviceParameter';

export interface InterfaceBindingOption {
  value: string;
  label: string;
}

const INTERFACE_NAME_PATTERN = /^Device\.Ethernet\.Interface\.(\d+)\.Name$/;
const DIRECT_IPV4_PATTERN = /^Device\.Ethernet\.Interface\.(\d+)\.IPv4Address\.(\d+)\.IPAddress$/;
const VLAN_NAME_PATTERN = /^Device\.Ethernet\.Interface\.(\d+)\.VlanInterface\.(\d+)\.Name$/;
const VLAN_IPV4_PATTERN = /^Device\.Ethernet\.Interface\.(\d+)\.VlanInterface\.(\d+)\.IPv4Address\.(\d+)\.IPAddress$/;

/** Build the device-defined interface binding choices while keeping the TR-181 path as the submitted value. */
export function buildInterfaceBindingOptions(
  parameters: ParameterSchemaItem[],
  targetInterfaceName?: string,
): InterfaceBindingOption[] {
  const interfaceNames = new Map<string, string>();
  const vlanNames = new Map<string, string>();

  for (const item of parameters) {
    const interfaceMatch = INTERFACE_NAME_PATTERN.exec(item.path);
    if (interfaceMatch) {
      interfaceNames.set(interfaceMatch[1], String(item.currentValue ?? ''));
      continue;
    }
    const vlanMatch = VLAN_NAME_PATTERN.exec(item.path);
    if (vlanMatch) {
      vlanNames.set(`${vlanMatch[1]}.${vlanMatch[2]}`, String(item.currentValue ?? ''));
    }
  }

  const options: Array<InterfaceBindingOption & { sortKey: number[] }> = [];
  for (const item of parameters) {
    const directMatch = DIRECT_IPV4_PATTERN.exec(item.path);
    if (directMatch) {
      const [, interfaceId, addressId] = directMatch;
      const interfaceName = interfaceNames.get(interfaceId) || `Interface.${interfaceId}`;
      if (targetInterfaceName && interfaceName !== targetInterfaceName) continue;
      options.push({
        value: item.path,
        label: `${interfaceName}.IPv4:${addressId}`,
        sortKey: [Number(interfaceId), 0, -Number(addressId)],
      });
      continue;
    }

    const vlanMatch = VLAN_IPV4_PATTERN.exec(item.path);
    if (vlanMatch) {
      const [, interfaceId, vlanId, addressId] = vlanMatch;
      const interfaceName = interfaceNames.get(interfaceId) || `Interface.${interfaceId}`;
      if (targetInterfaceName && interfaceName !== targetInterfaceName) continue;
      const vlanName = vlanNames.get(`${interfaceId}.${vlanId}`) || `VLAN.${vlanId}`;
      options.push({
        value: item.path,
        label: `${interfaceName}.${vlanName}:${addressId}`,
        sortKey: [Number(interfaceId), 1, Number(vlanId), -Number(addressId)],
      });
    }
  }

  return options
    .sort((left, right) => {
      const length = Math.max(left.sortKey.length, right.sortKey.length);
      for (let index = 0; index < length; index += 1) {
        const delta = (left.sortKey[index] ?? 0) - (right.sortKey[index] ?? 0);
        if (delta !== 0) return delta;
      }
      return left.value.localeCompare(right.value);
    })
    .map(({ value, label }) => ({ value, label }));
}
