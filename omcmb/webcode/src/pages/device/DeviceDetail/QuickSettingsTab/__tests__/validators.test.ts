import { describe, expect, it } from 'vitest';
import {
  BM_RU_RF_SWITCH_PATH,
  formatEnumDisplayValue,
  formatLteBandwidthDisplay,
  LTE_BANDWIDTH_PATH,
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