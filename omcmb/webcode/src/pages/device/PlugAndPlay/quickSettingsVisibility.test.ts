import { describe, expect, it } from 'vitest';
import {
  isIpsecParametersVisible,
  isPtpDetailsVisible,
} from './quickSettingsVisibility';

describe('plug-and-play quick-setting conditional visibility', () => {
  it('shows IPsec parameters only when the switch is explicitly on', () => {
    expect(isIpsecParametersVisible('0')).toBe(false);
    expect(isIpsecParametersVisible(false)).toBe(false);
    expect(isIpsecParametersVisible('1')).toBe(true);
    expect(isIpsecParametersVisible(true)).toBe(true);
    expect(isIpsecParametersVisible('')).toBe(false);
    expect(isIpsecParametersVisible(undefined)).toBe(false);
  });

  it('shows PTP details only in 1588 mode', () => {
    expect(isPtpDetailsVisible('1588_PPS')).toBe(true);
    expect(isPtpDetailsVisible('GPS_PPS')).toBe(false);
    expect(isPtpDetailsVisible('GPS_AND_PTP')).toBe(false);
    expect(isPtpDetailsVisible(undefined)).toBe(false);
  });
});
