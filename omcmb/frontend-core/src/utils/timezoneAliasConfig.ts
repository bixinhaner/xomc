import timezoneAliasConfigRaw from '../config/timezone-aliases.json?raw';

interface TimezoneAliasConfigItem {
  iana?: string;
  deviceValue?: string;
  displayValue?: string;
}

export interface TimezoneAliasOption {
  label: string;
  value: string;
}

function buildAliasToTimezoneMap(raw: string): Map<string, string> {
  const map = new Map<string, string>();
  const items = JSON.parse(raw) as TimezoneAliasConfigItem[];
  for (const item of items) {
    const timezone = String(item.displayValue || item.iana || '').trim();
    const alias = String(item.deviceValue || '').trim();
    if (!timezone || !alias || map.has(alias)) continue;
    map.set(alias, timezone);
  }
  return map;
}

const aliasToTimezoneMap = buildAliasToTimezoneMap(timezoneAliasConfigRaw);

function buildTimezoneAliasOptions(raw: string): TimezoneAliasOption[] {
  const items = JSON.parse(raw) as TimezoneAliasConfigItem[];
  const options: TimezoneAliasOption[] = [];
  const seen = new Set<string>();
  for (const item of items) {
    const label = String(item.displayValue || item.iana || '').trim();
    const value = String(item.deviceValue || '').trim();
    if (!label || !value || seen.has(value)) continue;
    seen.add(value);
    options.push({ label, value });
  }
  return options;
}

const timezoneAliasOptions = buildTimezoneAliasOptions(timezoneAliasConfigRaw);

export function mapTimezoneAliasToDisplay(value: string): string {
  const normalized = String(value ?? '').trim();
  if (!normalized) return '';
  return aliasToTimezoneMap.get(normalized) ?? normalized;
}

export function getTimezoneAliasOptions(): TimezoneAliasOption[] {
  return timezoneAliasOptions;
}
