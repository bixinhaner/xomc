const LTE_DL_BANDWIDTH_SUFFIX = '.LTE.RAN.RF.DLBandwidth';
const LTE_UL_BANDWIDTH_SUFFIX = '.LTE.RAN.RF.ULBandwidth';

export function isLteDownlinkBandwidthPath(path: string): boolean {
  return path.endsWith(LTE_DL_BANDWIDTH_SUFFIX);
}

export function isLteUplinkBandwidthPath(path: string): boolean {
  return path.endsWith(LTE_UL_BANDWIDTH_SUFFIX);
}

export function lteBandwidthValuesMatch(downlink: unknown, uplink: unknown): boolean {
  return String(downlink ?? '').trim() === String(uplink ?? '').trim();
}
