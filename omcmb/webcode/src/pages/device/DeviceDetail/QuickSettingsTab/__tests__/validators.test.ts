import { describe, expect, it } from 'vitest';
import {
  BM_RU_RF_SWITCH_PATH,
  formatEnumDisplayValue,
  formatLteBandwidthDisplay,
  LTE_BANDWIDTH_PATH,
  normalizeEnumValue,
  resolveQuickSettingsParameterType,
  validateLteQOffsetValue,
  validateValue,
  validateMmeIp,
  validateMmeIpPlmnLimit,
  validateMmeIpPlmnRows,
  validatePlmn,
} from '../validators';

describe('LTE bandwidth display formatting', () => {
  it('uses compact LTE bandwidth labels', () => {
    expect(formatEnumDisplayValue('100', undefined, LTE_BANDWIDTH_PATH)).toBe('CELL_BW_100(20M)');
    expect(formatLteBandwidthDisplay('100')).toBe('20M');
    expect(formatLteBandwidthDisplay('20')).toBe('20M');
    expect(formatLteBandwidthDisplay('20.00')).toBe('20M');
    expect(formatLteBandwidthDisplay(20)).toBe('20M');
  });

  it('falls back to dash for empty detail values', () => {
    expect(formatLteBandwidthDisplay('')).toBe('-');
    expect(formatLteBandwidthDisplay(undefined)).toBe('-');
    expect(formatLteBandwidthDisplay(null)).toBe('-');
  });

  it('preserves unknown values', () => {
    expect(formatLteBandwidthDisplay('unknown')).toBe('unknown');
  });

  it('maps BM RU RF switch values to on/off labels', () => {
    expect(formatEnumDisplayValue('1', undefined, BM_RU_RF_SWITCH_PATH)).toBe('开');
    expect(formatEnumDisplayValue('0', undefined, BM_RU_RF_SWITCH_PATH)).toBe('关');
  });
});

describe('LTE QOffset validation', () => {
  it('accepts even values in the device-supported range', () => {
    expect(validateLteQOffsetValue('-24')).toBeNull();
    expect(validateLteQOffsetValue('0')).toBeNull();
    expect(validateLteQOffsetValue('24')).toBeNull();
  });

  it('rejects odd and out-of-range values before SPV', () => {
    expect(validateLteQOffsetValue('23')).toBe('QOffset 仅支持 -24 到 24 的偶数');
    expect(validateLteQOffsetValue('26')).toBe('QOffset 仅支持 -24 到 24 的偶数');
    expect(validateLteQOffsetValue('-25')).toBe('QOffset 仅支持 -24 到 24 的偶数');
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

describe('quick settings parameter type resolution', () => {
  it('lets quicksettings XML override stale raw string types for numeric fields', () => {
    const type = resolveQuickSettingsParameterType('unsignedInt', 'string', 'string');

    expect(type).toBe('unsignedInt');
    expect(validateValue('26', type, { minValue: 22, maxValue: 32 })).toBeNull();
  });

  it('normalizes schema type casing without losing canonical parameter types', () => {
    expect(resolveQuickSettingsParameterType(undefined, 'BOOLEAN')).toBe('boolean');
    expect(resolveQuickSettingsParameterType(undefined, 'UNSIGNEDINT')).toBe('unsignedInt');
    expect(resolveQuickSettingsParameterType(undefined, 'DATETIME')).toBe('dateTime');
    expect(resolveQuickSettingsParameterType(undefined, 'HEXBINARY')).toBe('hexBinary');
  });
});

describe('MME IP + PLMN validation', () => {
  it('accepts valid IPv4 and PLMN values', () => {
    expect(validateMmeIp('10.0.0.1')).toBeNull();
    expect(validateMmeIp(' 172.16.1.254 ')).toBeNull();
    expect(validatePlmn('46000')).toBeNull();
    expect(validatePlmn('460000')).toBeNull();
  });

  it('rejects malformed MME IP values', () => {
    expect(validateMmeIp('1.2.3')).toBe('MME IP 必须为合法 IPv4 地址');
    expect(validateMmeIp('999.1.1.1')).toBe('MME IP 必须为合法 IPv4 地址');
    expect(validateMmeIp('01.2.3.4')).toBe('MME IP 必须为合法 IPv4 地址');
  });

  it('rejects PLMN values outside 5-6 digits', () => {
    expect(validatePlmn('4600')).toBe('PLMN 必须为 5-6 位数字');
    expect(validatePlmn('4600000')).toBe('PLMN 必须为 5-6 位数字');
    expect(validatePlmn('46A00')).toBe('PLMN 必须为 5-6 位数字');
  });

  it('rejects partial MME rows before serialization', () => {
    expect(validateMmeIpPlmnRows([{ mmeIp: '10.0.0.1', plmn: '' }])).toBe('第 1 行 PLMN 不能为空');
    expect(validateMmeIpPlmnRows([{ mmeIp: '', plmn: '46000' }])).toBe('第 1 行 MME IP 不能为空');
  });

  it('reports row-specific MME IP and PLMN errors', () => {
    expect(validateMmeIpPlmnRows([
      { mmeIp: '10.0.0.1', plmn: '46000' },
      { mmeIp: '999.1.1.1', plmn: '46000' },
    ])).toBe('第 2 行 MME IP 必须为合法 IPv4 地址');
    expect(validateMmeIpPlmnRows([
      { mmeIp: '10.0.0.1', plmn: '46000' },
      { mmeIp: '10.0.0.2', plmn: '4600000' },
    ])).toBe('第 2 行 PLMN 必须为 5-6 位数字');
  });
});

describe('enum value normalization for dirty check (issue #317)', () => {
  const oneZeroOptions = [
    { value: '1', label: 'NTP Server' },
    { value: '0', label: 'NTP Client' },
  ];
  const trueFalseOptions = [
    { value: 'true', label: 'Enable' },
    { value: 'false', label: 'Disable' },
  ];

  it('maps device boolean serialization "true" onto 1/0 option space', () => {
    expect(normalizeEnumValue('true', oneZeroOptions)).toBe('1');
    expect(normalizeEnumValue('false', oneZeroOptions)).toBe('0');
  });

  it('keeps exact option values untouched in true/false option space (BM style)', () => {
    expect(normalizeEnumValue('true', trueFalseOptions)).toBe('true');
    expect(normalizeEnumValue('false', trueFalseOptions)).toBe('false');
  });

  it('returns raw value unchanged when no options are declared', () => {
    expect(normalizeEnumValue('true', undefined)).toBe('true');
    expect(normalizeEnumValue('fiber', [])).toBe('fiber');
  });

  it('matches option values case-insensitively and by label', () => {
    expect(normalizeEnumValue('TRUE', trueFalseOptions)).toBe('true');
    expect(normalizeEnumValue('Enable', trueFalseOptions)).toBe('true');
    expect(normalizeEnumValue('ntp server', oneZeroOptions)).toBe('1');
  });

  it('preserves unknown values instead of forcing a match', () => {
    expect(normalizeEnumValue('3', oneZeroOptions)).toBe('3');
  });
});
