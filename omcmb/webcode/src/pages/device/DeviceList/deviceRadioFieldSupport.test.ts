import { describe, expect, it } from 'vitest';
import { formatDeviceRadioField } from './deviceRadioFieldSupport';

describe('formatDeviceRadioField', () => {
  it('BSC/BTS 的 LTE/NR 专属字段显示为不适用', () => {
    const bsc = { networkType: 'GSM', productClass: 'FAP/PGSM' };
    const bts = { networkType: 'GSM', productClass: 'FAP/BTS' };

    for (const field of ['pci', 'tac', 'band', 'dlEarfcn', 'ulEarfcn'] as const) {
      expect(formatDeviceRadioField(bsc, field, '')).toBe('-');
      expect(formatDeviceRadioField(bts, field, '')).toBe('-');
    }

    expect(formatDeviceRadioField(bsc, 'txPower', '')).toBe('-');
    expect(formatDeviceRadioField(bsc, 'pci', '101')).toBe('-');
  });

  it('LTE/NR 支持字段未上报时保持空白以暴露数据缺失', () => {
    const lte = { networkType: 'eNB', productClass: 'FAP/BLQ' };
    const nr = { networkType: 'gNB', productClass: 'FAP/BSC7040' };

    expect(formatDeviceRadioField(lte, 'pci', '')).toBe('');
    expect(formatDeviceRadioField(nr, 'tac', undefined)).toBe('');
  });

  it('保留有效值和数值零', () => {
    const lte = { networkType: 'eNB', productClass: 'FAP/BLQ' };

    expect(formatDeviceRadioField(lte, 'pci', '101')).toBe('101');
    expect(formatDeviceRadioField(lte, 'txPower', 0)).toBe('0');
  });

  it('BTS 发射功率有产品模型支持，缺失时不能伪装成不适用', () => {
    const bts = { networkType: 'GSM', productClass: 'FAP/BTS' };

    expect(formatDeviceRadioField(bts, 'txPower', '')).toBe('');
  });

  it('未知产品保持缺失语义，不贸然显示不适用', () => {
    const unknown = { networkType: 'future-radio', productClass: 'UNKNOWN' };

    expect(formatDeviceRadioField(unknown, 'pci', '')).toBe('');
  });
});
