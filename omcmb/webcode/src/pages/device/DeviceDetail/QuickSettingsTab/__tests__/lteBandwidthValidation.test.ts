import { describe, expect, it } from 'vitest';
import {
  isLteDownlinkBandwidthPath,
  isLteUplinkBandwidthPath,
  lteBandwidthValuesMatch,
} from '../lteBandwidthValidation';

describe('LTE bandwidth validation', () => {
  it('recognizes the generic and resolved DL/UL paths', () => {
    expect(isLteDownlinkBandwidthPath(
      'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth',
    )).toBe(true);
    expect(isLteUplinkBandwidthPath(
      'Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.ULBandwidth',
    )).toBe(true);
  });

  it('requires independently entered downlink and uplink values to match', () => {
    expect(lteBandwidthValuesMatch('50', 50)).toBe(true);
    expect(lteBandwidthValuesMatch('25', '50')).toBe(false);
    expect(lteBandwidthValuesMatch('50', '')).toBe(false);
  });
});
