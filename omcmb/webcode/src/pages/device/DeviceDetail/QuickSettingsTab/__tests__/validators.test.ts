import { describe, expect, it } from 'vitest';
import {
  BM_RU_RF_SWITCH_PATH,
  formatEnumDisplayValue,
  formatLteBandwidthDisplay,
  LTE_BANDWIDTH_PATH,
  validateMmeIpPlmnLimit,
} from '../validators';

describe('LTE bandwidth display formatting', () => {
  it('uses the same LTE enum mapping helper as quick settings', () => {
    expect(formatLteBandwidthDisplay('100')).toBe(
      formatEnumDisplayValue('100', undefined, LTE_BANDWIDTH_PATH),
    );
    expect(formatLteBandwidthDisplay('100')).toBe('CELL_BW_100(20M)');
  });

  it('falls back to dash for empty detail values', () => {
    expect(formatLteBandwidthDisplay('')).toBe('-');
    expect(formatLteBandwidthDisplay(undefined)).toBe('-');
    expect(formatLteBandwidthDisplay(null)).toBe('-');
  });

  it('preserves unknown values', () => {
    expect(formatLteBandwidthDisplay('20MHz')).toBe('20MHz');
  });

  it('maps BM RU RF switch values to on/off labels', () => {
    expect(formatEnumDisplayValue('1', undefined, BM_RU_RF_SWITCH_PATH)).toBe('开');
    expect(formatEnumDisplayValue('0', undefined, BM_RU_RF_SWITCH_PATH)).toBe('关');
  });
});

describe('MME list limit validation', () => {
  it('allows MME rows up to the metadata limit', () => {
    const rows = Array.from({ length: 16 }, (_, idx) => ({
      mmeIp: `10.0.0.${idx + 1}`,
      plmn: '46000',
    }));
    expect(validateMmeIpPlmnLimit(rows, 16)).toBeNull();
  });

  it('rejects rows above the metadata limit', () => {
    const rows = Array.from({ length: 17 }, (_, idx) => ({
      mmeIp: `10.0.0.${idx + 1}`,
      plmn: '46000',
    }));
    expect(validateMmeIpPlmnLimit(rows, 16)).toBe('最多支持 16 个 MME');
  });

  it('does not count blank draft rows toward the limit', () => {
    const rows = [
      ...Array.from({ length: 16 }, (_, idx) => ({
        mmeIp: `10.0.0.${idx + 1}`,
        plmn: '46000',
      })),
      { mmeIp: '', plmn: '' },
    ];
    expect(validateMmeIpPlmnLimit(rows, 16)).toBeNull();
  });

  it('does not limit MME rows when metadata has no limit', () => {
    const rows = Array.from({ length: 32 }, (_, idx) => ({
      mmeIp: `10.0.0.${idx + 1}`,
      plmn: '46000',
    }));
    expect(validateMmeIpPlmnLimit(rows, undefined)).toBeNull();
  });
});
