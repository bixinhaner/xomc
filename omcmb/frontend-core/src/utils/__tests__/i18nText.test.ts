import { describe, it, expect } from 'vitest';
import { getI18nText, getRecordI18n } from '../i18nText';

describe('getI18nText', () => {
  it('returns long-key match', () => {
    expect(getI18nText({ 'en-US': 'Hi', 'zh-CN': '你好' }, 'en-US')).toBe('Hi');
    expect(getI18nText({ 'en-US': 'Hi', 'zh-CN': '你好' }, 'zh-CN')).toBe('你好');
  });

  it('falls back from long to short key for same locale', () => {
    expect(getI18nText({ en: 'Hi', zh: '你好' }, 'en-US')).toBe('Hi');
    expect(getI18nText({ en: 'Hi', zh: '你好' }, 'zh-CN')).toBe('你好');
  });

  it('falls back to zh-CN / zh when target locale missing', () => {
    expect(getI18nText({ 'zh-CN': '你好' }, 'en-US')).toBe('你好');
    expect(getI18nText({ zh: '你好' }, 'en-US')).toBe('你好');
  });

  it('falls back to legacy when i18n is empty', () => {
    expect(getI18nText({}, 'en-US', 'legacy value')).toBe('legacy value');
    expect(getI18nText(null, 'en-US', 'legacy value')).toBe('legacy value');
    expect(getI18nText(undefined, 'en-US', 'legacy value')).toBe('legacy value');
  });

  it('returns empty string when nothing is available', () => {
    expect(getI18nText(null, 'en-US')).toBe('');
    expect(getI18nText({}, 'zh-CN')).toBe('');
  });
});

describe('getRecordI18n', () => {
  it('reads name_i18n then falls back to name', () => {
    const record = { name: '默认设备组', name_i18n: { 'en-US': 'Default Group', 'zh-CN': '默认设备组' } };
    expect(getRecordI18n(record, 'name', 'en-US')).toBe('Default Group');
    expect(getRecordI18n(record, 'name', 'zh-CN')).toBe('默认设备组');
  });

  it('falls back to legacy when i18n missing', () => {
    const record = { name: '默认设备组' };
    expect(getRecordI18n(record, 'name', 'en-US')).toBe('默认设备组');
  });

  it('handles null/undefined record', () => {
    expect(getRecordI18n(null, 'name', 'en-US')).toBe('');
    expect(getRecordI18n(undefined, 'name', 'en-US')).toBe('');
  });
});
