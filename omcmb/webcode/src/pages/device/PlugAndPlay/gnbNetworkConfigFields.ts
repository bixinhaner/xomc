export interface GnbNetworkConfigField {
  header: string;
  labelKey: string;
}

export const GNB_NETWORK_CONFIG_FIELDS: readonly GnbNetworkConfigField[] = [
  { header: 'Interface Name', labelKey: 'provision.nrQuick.interfaceName' },
  { header: 'Address Type', labelKey: 'provision.nrQuick.addressType' },
  { header: 'IP Address', labelKey: 'provision.ipAddress' },
  { header: 'Subnet Mask', labelKey: 'provision.subnetMask' },
  { header: 'Prefix Length', labelKey: 'provision.nrQuick.prefixLength' },
  { header: 'Gateway', labelKey: 'provision.gateway' },
  { header: 'Bear Type', labelKey: 'provision.nrQuick.bearType' },
  { header: 'Vlan Name', labelKey: 'provision.nrQuick.vlanName' },
  { header: 'Vlan ID', labelKey: 'provision.vlanId' },
] as const;
