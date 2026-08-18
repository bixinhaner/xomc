export interface NrCarrierBandwidthOption {
  value: string;
  label: string;
}

/**
 * 5G NR carrier bandwidths supported by the device, indexed by SCS enum value:
 * 0 = 15kHz, 1 = 30kHz, 2 = 60kHz.
 */
export const NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS: Record<string, NrCarrierBandwidthOption[]> = {
  '0': [
    { value: '52', label: '10MHz(52RB)' },
    { value: '106', label: '20MHz(106RB)' },
    { value: '160', label: '30MHz(160RB)' },
    { value: '216', label: '40MHz(216RB)' },
    { value: '270', label: '50MHz(270RB)' },
  ],
  '1': [
    { value: '24', label: '10MHz(24RB)' },
    { value: '51', label: '20MHz(51RB)' },
    { value: '78', label: '30MHz(78RB)' },
    { value: '106', label: '40MHz(106RB)' },
    { value: '133', label: '50MHz(133RB)' },
    { value: '162', label: '60MHz(162RB)' },
    { value: '189', label: '70MHz(189RB)' },
    { value: '217', label: '80MHz(217RB)' },
    { value: '245', label: '90MHz(245RB)' },
    { value: '273', label: '100MHz(273RB)' },
  ],
  '2': [
    { value: '11', label: '10MHz(11RB)' },
    { value: '24', label: '20MHz(24RB)' },
    { value: '38', label: '30MHz(38RB)' },
    { value: '51', label: '40MHz(51RB)' },
    { value: '65', label: '50MHz(65RB)' },
    { value: '79', label: '60MHz(79RB)' },
    { value: '93', label: '70MHz(93RB)' },
    { value: '107', label: '80MHz(107RB)' },
    { value: '121', label: '90MHz(121RB)' },
    { value: '135', label: '100MHz(135RB)' },
  ],
};
