import { describe, expect, it } from 'vitest';
import type { QuickSettingsParam } from '@core/types/quicksettings';
import {
  resolveBscBtsAddParameterType,
  serializeBscBtsAddDeviceValue,
} from '../BscBtsAddModal';
import {
  normalizeBscCodecSupportValue,
  parseQuickSettingsMultiCheckboxValue,
  resolveQuickSettingsParameterType,
  serializeBscCodecSupportValue,
} from '../validators';

describe('BSC BTS add parameter serialization', () => {
  it('serializes CodecSupport selections with the device set encoding', () => {
    const param: QuickSettingsParam = {
      name: 'CodecSupport',
      titleZh: '编解码支持',
      titleEn: 'CodecSupport',
      type: 'multiCheckbox',
      checkboxOptions: ['fr', 'hr', 'efr', 'amr'],
    };

    expect(parseQuickSettingsMultiCheckboxValue('fr,hr')).toEqual(['fr', 'hr']);
    expect(parseQuickSettingsMultiCheckboxValue('fr-hr')).toEqual(['fr', 'hr']);
    expect(serializeBscBtsAddDeviceValue(param, 'fr,hr')).toBe('hr');
    expect(serializeBscBtsAddDeviceValue(param, 'fr-hr')).toBe('hr');
    expect(resolveBscBtsAddParameterType(param)).toBe('string');
    expect(resolveQuickSettingsParameterType('multiCheckbox')).toBe('string');
  });

  it('keeps fr mandatory while allowing other CodecSupport combinations', () => {
    const param: QuickSettingsParam = {
      name: 'CodecSupport',
      titleZh: '编解码支持',
      titleEn: 'CodecSupport',
      standardPath: 'DeviceGSM.Bts.{i}.CodecSupport',
      type: 'multiCheckbox',
      checkboxOptions: ['fr', 'hr', 'efr', 'amr'],
    };

    expect(normalizeBscCodecSupportValue('hr')).toEqual(['fr', 'hr']);
    expect(normalizeBscCodecSupportValue(['amr', 'efr'])).toEqual(['fr', 'efr', 'amr']);
    expect(serializeBscCodecSupportValue(['fr', 'hr', 'amr'])).toBe('hr-amr');
    expect(serializeBscBtsAddDeviceValue(param, '')).toBe('');
    expect(serializeBscBtsAddDeviceValue(param, 'hr')).toBe('hr');
    expect(serializeBscBtsAddDeviceValue(param, 'fr-hr-efr-amr')).toBe('hr-efr-amr');
  });

  it('normalizes handover display labels to the device enum value before SPV', () => {
    const param: QuickSettingsParam = {
      name: 'handover',
      titleZh: '切换开关',
      titleEn: 'handover',
      type: 'enum',
      enumOptions: [
        { value: '1', label: 'Allow' },
        { value: '0', label: 'Forbid' },
      ],
    };

    expect(serializeBscBtsAddDeviceValue(param, 'Allow')).toBe('1');
    expect(serializeBscBtsAddDeviceValue(param, 'Forbid')).toBe('0');
    expect(serializeBscBtsAddDeviceValue(param, '1')).toBe('1');
  });
});
