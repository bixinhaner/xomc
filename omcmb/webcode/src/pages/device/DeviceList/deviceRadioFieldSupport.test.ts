import { describe, expect, it } from 'vitest';
import { formatDeviceRadioField, formatDeviceRFStatus, supportsOwnRF } from './deviceRadioFieldSupport';

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

  it('GSM band 透出带宽样式值时，改为根据 ARFCN 推导真实频段', () => {
    const gsm = { networkType: 'GSM', productClass: 'UNKNOWN', arfcn: '62' };

    expect(formatDeviceRadioField(gsm, 'band', 'GSM200K')).toBe('GSM900');
    expect(formatDeviceRadioField(gsm, 'band', '200')).toBe('GSM900');
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

describe('device RF support', () => {
  it('BSC 不拥有设备级 RF，历史 RF 快照也显示为不适用', () => {
    const bsc = { networkType: 'GSM', productClass: 'FAP/PGSM' };

    expect(supportsOwnRF(bsc)).toBe(false);
    expect(formatDeviceRFStatus(bsc, 'on,on')).toBe('-');
  });

  it('BTS 和 LTE 基站保留自身 RF 状态', () => {
    const bts = { networkType: 'GSM', productClass: 'FAP/BTS' };
    const lte = { networkType: 'eNB', productClass: 'FAP/MLN/DC' };

    expect(supportsOwnRF(bts)).toBe(true);
    expect(formatDeviceRFStatus(bts, 'on')).toBe('on');
    expect(formatDeviceRFStatus(lte, 'on,off')).toBe('on,off');
  });
});
