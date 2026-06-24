import { useMemo } from 'react';
import { useIntl } from 'react-intl';
import { useDictionary } from '@core/hooks/api/useSystem';
import type { TechnologyType } from '@/pages/dashboard/kpi-config';

// 前端 KPI 静态契约目前只识别这三类；字典里若出现其他 value 先过滤掉，
// 等单独 issue 放开 TechnologyType 联合类型 + 后端 CHECK 约束后再扩展。
const KNOWN_TECHS: ReadonlySet<TechnologyType> = new Set(['lte', 'nr', 'gsm']);

export interface TechnologyOption {
  value: TechnologyType;
  label: string;
  sort: number;
}

const pickLabel = (...candidates: Array<string | undefined | null>): string | undefined => {
  for (const c of candidates) {
    const s = (c ?? '').trim();
    if (s) return s;
  }
  return undefined;
};

/**
 * 首页 Segmented / KPI 配置 Tabs 的网络制式选项来源。
 * 纯字典驱动：字典里有几项就显示几项；空就空（让运维感知字典缺失）。
 */
export function useTechnologyDictionary(): {
  options: TechnologyOption[];
  isLoading: boolean;
} {
  const { data, isLoading } = useDictionary('network_type');
  const { locale } = useIntl();

  const options = useMemo<TechnologyOption[]>(() => {
    const details = data?.sysDictionaryDetails;
    if (!details?.length) return [];

    return details
      .filter((d) => d.status !== false)
      .filter((d) => KNOWN_TECHS.has(d.value as TechnologyType))
      .map<TechnologyOption>((d) => {
        const tech = d.value as TechnologyType;
        const label = pickLabel(d.labelI18n?.[locale], d.label) ?? tech.toUpperCase();
        return { value: tech, label, sort: d.sort ?? 0 };
      })
      .sort((a, b) => a.sort - b.sort);
  }, [data, locale]);

  return { options, isLoading };
}
