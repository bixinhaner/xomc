import { describe, expect, it } from 'vitest';
import { messages, SUPPORTED_LOCALES } from '../index';

const requiredKeys = [
  'device.control.markerAria',
  'device.control.tooltipHint',
  'device.control.tabTitle',
  'device.control.viewInDetail',
  'device.control.quickTitle',
  'device.control.detailTitle',
  'device.control.latestResult',
  'device.control.sourceName',
  'device.control.phase.deactivating',
  'device.control.phase.verifying',
  'device.control.phase.deactivated',
  'device.control.phase.failed',
  'device.control.phase.recovering',
  'device.control.phase.recovery_failed',
  'device.control.phaseShort.deactivated',
  'device.control.reason',
  'device.control.reason.confirmedExit',
  'device.control.reason.confirmedEnter',
  'device.control.action.deactivate',
  'device.control.action.activate',
  'device.control.status.verified',
  'device.control.evaluationEvidence',
  'device.control.technicalDetails',
  'device.control.parameter.ipsec',
  'device.control.parameter.rf',
  'device.control.noHistory',
  'device.control.loadFailed',
] as const;

describe('device control i18n messages', () => {
  it.each(SUPPORTED_LOCALES)('defines required user-visible messages for %s', (locale) => {
    for (const key of requiredKeys) {
      expect(messages[locale][key], `${locale} missing ${key}`).toEqual(expect.any(String));
      expect(messages[locale][key].trim()).not.toBe('');
    }
  });

  it('keeps every device control key aligned across locales', () => {
    const keysByLocale = SUPPORTED_LOCALES.map((locale) =>
      Object.keys(messages[locale])
        .filter((key) => key.startsWith('device.control.'))
        .sort(),
    );

    expect(keysByLocale[0]).toEqual(keysByLocale[1]);
  });

  it('provides native English copy for the primary workflow', () => {
    expect(messages['en-US']['device.control.tabTitle']).toBe('Action History');
    expect(messages['en-US']['device.control.reason.confirmedExit']).toContain('left the allowed area');
    expect(messages['en-US']['device.control.tooltipHint']).toBe('Click to view the reason and action history');
  });
});
