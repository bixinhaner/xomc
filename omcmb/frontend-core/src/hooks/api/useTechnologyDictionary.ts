import { useCallback, useMemo } from 'react';
import { useIntl } from 'react-intl';
import type { DeviceType } from '../../types/indicatorLibrary';
import type { TechnologyType } from '../../types/technology';
import { useDictionary } from './useSystem';

// 前端 KPI 静态契约目前只识别这三类；字典里若出现其他 value 先过滤掉，
// 等单独 issue 放开 TechnologyType 联合类型 + 后端 CHECK 约束后再扩展。
const KNOWN_TECHS: ReadonlySet<TechnologyType> = new Set(['lte', 'nr', 'gsm']);
const TECH_TO_DEVICE_TYPE: Record<TechnologyType, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};
const DEVICE_TYPE_TO_TECH: Record<DeviceType, TechnologyType> = {
  ENB: 'lte',
  GNB: 'nr',
  GSM: 'gsm',
};
const RADIO_MODE_TO_TECH: Record<string, TechnologyType> = {
  eNB: 'lte',
  ENB: 'lte',
  gNB: 'nr',
  GNB: 'nr',
  GSM: 'gsm',
};

export interface TechnologyOption {
  value: TechnologyType;
  label: string;
  sort: number;
}

export interface TechnologyDeviceTypeOption {
  value: DeviceType;
  label: string;
  sort: number;
  technology: TechnologyType;
}

const pickLabel = (...candidates: Array<string | undefined | null>): string | undefined => {
  for (const c of candidates) {
    const s = (c ?? '').trim();
    if (s) return s;
  }
  return undefined;
};

export function isKnownTechnology(value: unknown): value is TechnologyType {
  return typeof value === 'string' && KNOWN_TECHS.has(value as TechnologyType);
}

export function technologyToDeviceType(technology: TechnologyType): DeviceType {
  return TECH_TO_DEVICE_TYPE[technology];
}

export function deviceTypeToTechnology(deviceType: DeviceType): TechnologyType {
  return DEVICE_TYPE_TO_TECH[deviceType];
}

export function radioModeToTechnology(radioMode: string): TechnologyType | undefined {
  return RADIO_MODE_TO_TECH[radioMode];
}

export function fallbackTechnologyLabel(technology?: string | null): string {
  const value = (technology ?? '').trim();
  return value ? value.toUpperCase() : '—';
}

export function useTechnologyDictionary(): {
  options: TechnologyOption[];
  deviceTypeOptions: TechnologyDeviceTypeOption[];
  labelForTechnology: (technology?: string | null) => string;
  labelForRadioMode: (radioMode?: string | null) => string;
  isLoading: boolean;
} {
  const { data, isLoading } = useDictionary('network_type');
  const { locale } = useIntl();

  const knownDetails = useMemo(() => {
    const details = data?.sysDictionaryDetails;
    return (details ?? []).filter((d) => isKnownTechnology(d.value));
  }, [data]);

  const labelMap = useMemo(() => {
    const map = new Map<TechnologyType, string>();
    knownDetails.forEach((d) => {
      const tech = d.value as TechnologyType;
      map.set(tech, pickLabel(d.labelI18n?.[locale], d.label) ?? fallbackTechnologyLabel(tech));
    });
    return map;
  }, [knownDetails, locale]);

  const labelForTechnology = useCallback(
    (technology?: string | null) => {
      if (isKnownTechnology(technology)) {
        return labelMap.get(technology) ?? fallbackTechnologyLabel(technology);
      }
      return fallbackTechnologyLabel(technology);
    },
    [labelMap],
  );

  const labelForRadioMode = useCallback(
    (radioMode?: string | null) => {
      const value = (radioMode ?? '').trim();
      if (!value) return '—';
      const technology = radioModeToTechnology(value);
      return technology ? labelForTechnology(technology) : value;
    },
    [labelForTechnology],
  );

  const options = useMemo<TechnologyOption[]>(() => {
    if (!knownDetails.length) return [];

    return knownDetails
      .filter((d) => d.status !== false)
      .map<TechnologyOption>((d) => {
        const tech = d.value as TechnologyType;
        const label = labelMap.get(tech) ?? fallbackTechnologyLabel(tech);
        return { value: tech, label, sort: d.sort ?? 0 };
      })
      .sort((a, b) => a.sort - b.sort);
  }, [knownDetails, labelMap]);

  const deviceTypeOptions = useMemo<TechnologyDeviceTypeOption[]>(
    () =>
      options.map((o) => ({
        value: technologyToDeviceType(o.value),
        label: o.label,
        sort: o.sort,
        technology: o.value,
      })),
    [options],
  );

  return { options, deviceTypeOptions, labelForTechnology, labelForRadioMode, isLoading };
}
