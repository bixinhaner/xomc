import type { DeviceType } from '../types/indicatorLibrary';

export type IndicatorLevelTranslate = (id: string) => string;

export function formatIndicatorLevel(level: string | null | undefined, t: IndicatorLevelTranslate): string {
  switch (level) {
    case 'device':
      return t('perf.query.indicatorLevelDevice');
    case 'plmn':
      return t('perf.query.indicatorLevelPlmn');
    case 'both':
      return t('perf.query.indicatorLevelBoth');
    default:
      return '-';
  }
}

export function shouldShowIndicatorLevel(deviceType: DeviceType | string | null | undefined): boolean {
  return deviceType === 'ENB' || deviceType === 'GSM';
}
