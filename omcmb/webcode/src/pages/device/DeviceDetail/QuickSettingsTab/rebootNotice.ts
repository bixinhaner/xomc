import type { TranslateFn } from '@/hooks/useT';

export function quickSettingsRebootNotice(t: TranslateFn, target?: number): string {
  if (target === 1) return t('device.quickSettings.rebootRequired', { services: t('device.quickSettings.rebootCore') });
  if (target === 2) return t('device.quickSettings.rebootRequired', { services: t('device.quickSettings.rebootWeb') });
  if (target === 3) return t('device.quickSettings.rebootRequired', { services: t('device.quickSettings.rebootCoreWeb') });
  return t('device.quickSettings.rebootRequiredGeneric');
}
