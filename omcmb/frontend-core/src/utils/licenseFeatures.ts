import type { FeatureList, SystemLicense } from '../services/api/systemLicenseApi';

export const UPS_LICENSE_FEATURE_CODE = 'CODE_UPS';
export const UPS_LICENSE_FEATURE_ID = '81';

export type LicenseControlledDeviceStandard = 'UPS';

export interface DeviceStandardLicenseRule {
  standard: LicenseControlledDeviceStandard;
  label: string;
  featureCode: string;
  legacyFeatureID?: string;
  authorizationPath: string[];
  aliases: string[];
  productClassPrefixes?: string[];
}

export const DEVICE_STANDARD_LICENSE_RULES: Record<LicenseControlledDeviceStandard, DeviceStandardLicenseRule> = {
  UPS: {
    standard: 'UPS',
    label: 'UPS',
    featureCode: UPS_LICENSE_FEATURE_CODE,
    legacyFeatureID: UPS_LICENSE_FEATURE_ID,
    authorizationPath: ['UPS', 'Monitor'],
    aliases: ['UPS'],
    productClassPrefixes: ['UPS'],
  },
};

const DEVICE_STANDARD_LICENSE_RULE_LIST = Object.values(DEVICE_STANDARD_LICENSE_RULES);

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value
    .filter((item): item is string => typeof item === 'string')
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizeFeatureCode(value?: string | null): string {
  return (value ?? '').trim().toUpperCase();
}

export function normalizeDeviceStandardValue(value?: string | null): string {
  return (value ?? '').trim().toUpperCase();
}

function featureListValue(featureList: FeatureList | undefined, key: string): unknown {
  if (!featureList || typeof featureList !== 'object') return undefined;
  return (featureList as Record<string, unknown>)[key];
}

function hasAuthorizationPath(value: unknown, path: string[]): boolean {
  if (typeof value === 'string') {
    return value === 'All';
  }
  if (Array.isArray(value)) {
    if (path.length !== 1) return false;
    return value.some((item) => item === 'All' || item === path[0]);
  }
  if (!value || typeof value !== 'object' || path.length === 0) {
    return false;
  }
  const record = value as Record<string, unknown>;
  const child = record[path[0]];
  if (child === undefined) return false;
  return hasAuthorizationPath(child, path.slice(1));
}

function hasLicensedFeatureCode(featureList: FeatureList | undefined, featureCode: string): boolean {
  const code = normalizeFeatureCode(featureCode);
  if (!code) return false;

  if (asStringArray(featureListValue(featureList, 'legacy_feature_codes')).some((item) => normalizeFeatureCode(item) === code)) {
    return true;
  }

  return (featureList?.features ?? []).some((feature) => (
    normalizeFeatureCode(feature.featureCode) === code && feature.licensed !== false
  ));
}

function hasLegacyFeatureID(featureList: FeatureList | undefined, featureID: string): boolean {
  const target = featureID.trim();
  return Boolean(target) && asStringArray(featureListValue(featureList, 'legacy_feature_ids')).includes(target);
}

export interface DeviceStandardDetectOptions {
  /** ProductClass 规则里 UPS_* / UPSxxx 都归 UPS；普通制式字段仍按精确值判断。 */
  matchProductClassPrefix?: boolean;
}

export function getLicenseControlledDeviceStandard(
  value?: string | null,
  options: DeviceStandardDetectOptions = {},
): LicenseControlledDeviceStandard | undefined {
  const normalized = normalizeDeviceStandardValue(value);
  if (!normalized) return undefined;

  for (const rule of DEVICE_STANDARD_LICENSE_RULE_LIST) {
    if (rule.aliases.some((alias) => normalizeDeviceStandardValue(alias) === normalized)) {
      return rule.standard;
    }
    if (
      options.matchProductClassPrefix
      && (rule.productClassPrefixes ?? []).some((prefix) => normalized.startsWith(normalizeDeviceStandardValue(prefix)))
    ) {
      return rule.standard;
    }
  }
  return undefined;
}

function firstString(...values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value === 'string') {
      const trimmed = value.trim();
      if (trimmed) return trimmed;
    }
  }
  return undefined;
}

function basenameWithoutXml(value: unknown): string | undefined {
  const text = firstString(value);
  if (!text) return undefined;
  const base = text.split(/[\\/]/).pop() ?? text;
  return base.replace(/\.xml$/i, '');
}

export interface DeviceScopedLicenseLike {
  name?: unknown;
  tech?: unknown;
  indicatorPlatform?: unknown;
  alarmNeType?: unknown;
  paramModelName?: unknown;
  neType?: unknown;
  deviceType?: unknown;
  networkType?: unknown;
  productType?: unknown;
  productName?: unknown;
  productClass?: unknown;
  loadedFrom?: unknown;
  patterns?: readonly unknown[];
  products?: readonly unknown[];
}

export function getLicenseControlledDeviceStandardFromScope(
  item: DeviceScopedLicenseLike,
): LicenseControlledDeviceStandard | undefined {
  const exactValues = [
    item.name,
    item.tech,
    item.indicatorPlatform,
    item.alarmNeType,
    item.paramModelName,
    item.neType,
    item.deviceType,
    item.networkType,
    item.productType,
    item.productName,
    basenameWithoutXml(item.loadedFrom),
  ];

  for (const value of exactValues) {
    const standard = getLicenseControlledDeviceStandard(firstString(value));
    if (standard) return standard;
  }

  const productClassStandard = getLicenseControlledDeviceStandard(firstString(item.productClass), {
    matchProductClassPrefix: true,
  });
  if (productClassStandard) return productClassStandard;

  for (const pattern of item.patterns ?? []) {
    const standard = getLicenseControlledDeviceStandard(firstString(pattern), {
      matchProductClassPrefix: true,
    });
    if (standard) return standard;
  }

  for (const product of item.products ?? []) {
    const standard = getLicenseControlledDeviceStandard(firstString(product));
    if (standard) return standard;
  }

  return undefined;
}

export function isDeviceStandardLicensed(
  license: SystemLicense | null | undefined,
  standard: LicenseControlledDeviceStandard,
): boolean {
  const rule = DEVICE_STANDARD_LICENSE_RULES[standard];
  if (!rule || !license || license.isExpired) return false;
  const featureList = license.featureList;
  if (hasLicensedFeatureCode(featureList, rule.featureCode)) return true;
  if (rule.legacyFeatureID && hasLegacyFeatureID(featureList, rule.legacyFeatureID)) return true;

  const rootAuthorizationPath = rule.authorizationPath.slice(0, 1);
  const authorizationTree = featureListValue(featureList, 'authorization_tree');
  return hasAuthorizationPath(authorizationTree, rule.authorizationPath)
    || hasAuthorizationPath(authorizationTree, rootAuthorizationPath);
}

export function isDeviceStandardVisibleByLicense(
  license: SystemLicense | null | undefined,
  isLicenseLoading: boolean,
  standard: LicenseControlledDeviceStandard,
): boolean {
  return isLicenseLoading || isDeviceStandardLicensed(license, standard);
}

export function isDeviceStandardValueVisibleByLicense(
  value: string | null | undefined,
  license: SystemLicense | null | undefined,
  isLicenseLoading: boolean,
  options: DeviceStandardDetectOptions = {},
): boolean {
  const standard = getLicenseControlledDeviceStandard(value, options);
  return !standard || isDeviceStandardVisibleByLicense(license, isLicenseLoading, standard);
}

export function isDeviceScopeVisibleByLicense(
  item: DeviceScopedLicenseLike,
  license: SystemLicense | null | undefined,
  isLicenseLoading: boolean,
): boolean {
  const standard = getLicenseControlledDeviceStandardFromScope(item);
  return !standard || isDeviceStandardVisibleByLicense(license, isLicenseLoading, standard);
}

export function filterDeviceScopedItemsByLicense<T extends DeviceScopedLicenseLike>(
  items: ReadonlyArray<T>,
  license: SystemLicense | null | undefined,
  isLicenseLoading: boolean,
): T[] {
  return items.filter((item) => isDeviceScopeVisibleByLicense(item, license, isLicenseLoading));
}

export interface DeviceStandardOptionLike {
  label?: unknown;
  value?: unknown;
}

export function filterDeviceStandardOptionsByLicense<T extends DeviceStandardOptionLike>(
  options: ReadonlyArray<T>,
  license: SystemLicense | null | undefined,
  isLicenseLoading: boolean,
): T[] {
  return options.filter((option) => (
    isDeviceStandardValueVisibleByLicense(firstString(option.value), license, isLicenseLoading)
    && isDeviceStandardValueVisibleByLicense(firstString(option.label), license, isLicenseLoading)
  ));
}

export function isUPSLicensed(license?: SystemLicense | null): boolean {
  return isDeviceStandardLicensed(license, 'UPS');
}
