import { describe, expect, it } from 'vitest';
import { messages, SUPPORTED_LOCALES } from '../index';

const requiredKeys = [
  'geofence.title',
  'geofence.settings.title',
  'geofence.mode.off',
  'geofence.mode.observe',
  'geofence.mode.enforce',
  'geofence.ruleType.polygonAllowZone',
  'geofence.ruleType.baselineRadius',
  'geofence.status.draft',
  'geofence.status.enabled',
  'geofence.status.disabled',
  'geofence.status.archived',
  'geofence.action.create',
  'geofence.action.edit',
  'geofence.action.publish',
  'geofence.action.enable',
  'geofence.action.disable',
  'geofence.action.archive',
  'geofence.action.bindDevices',
  'geofence.lifecycle.previewTitle',
  'geofence.lifecycle.reasonRequired',
  'geofence.binding.previewTitle',
  'geofence.binding.batchTitle',
  'geofence.binding.move',
  'geofence.binding.moveWarning',
  'geofence.binding.noEligibleDevices',
  'geofence.job.status.running',
  'geofence.job.status.succeeded',
  'geofence.message.disableDoesNotRecover',
  'device.locationSourceMode',
  'device.locationSourceModeHint',
  'device.locationSourceTr069',
  'device.locationSourceExternal',
] as const;

describe('geofence i18n messages', () => {
  it.each(SUPPORTED_LOCALES)(
    'defines required user-visible messages for %s',
    (locale) => {
      for (const key of requiredKeys) {
        expect(
          messages[locale][key],
          `${locale} missing ${key}`,
        ).toEqual(expect.any(String));
        expect(messages[locale][key].trim()).not.toBe('');
      }
    },
  );

  it('keeps every geofence key aligned across locales', () => {
    const keysByLocale = SUPPORTED_LOCALES.map((locale) =>
      Object.keys(messages[locale])
        .filter((key) => key.startsWith('geofence.'))
        .sort(),
    );

    expect(keysByLocale[0]).toEqual(keysByLocale[1]);
  });
});
