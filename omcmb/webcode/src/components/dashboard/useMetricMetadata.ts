/**
 * useMetricMetadata - 首页 KPI 面板指标元数据取数（KPI-ALL-IND 阶段4）
 *
 * 背景：放开全部指标后，面板存的是指标编号（K/C 编号），名字/单位不能再靠前端写死的
 * 「精选小表」反查。本 hook 按制式从指标库（GET /api/v1/indicators）拉全库元数据，
 * 建「编号 → {名字, 单位}」映射供面板渲染。
 *
 * - 名字：按界面语言取中文名 / 英文名（缺则回退另一种 / 编号）。
 * - 单位：取指标库的原始单位串（如 %/ms/KByte/number），非 i18n key。
 * - 旧 symbolic 别名（存量配置）在库里查不到 → 调用方回退老的 getKPIConfigByKey。
 */

import { useMemo } from 'react';
import { useAllIndicators } from '@core/hooks/api/useIndicatorsLibrary';
import { useAppStore } from '@core/store/appStore';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import type { TechnologyType } from '@/pages/dashboard/kpi-config';
import { getKPIConfigByKey } from '@/pages/dashboard/kpi-config';

/** 制式 → 指标库设备类型（与配置页 KpiConfig 的 TECH_TO_DEVICE_TYPE 一致）。 */
const TECH_TO_DEVICE_TYPE: Record<TechnologyType, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};

export interface MetricMeta {
  /** 本地化显示名（中/英）。 */
  name: string;
  /** 原始单位串（指标库 unitId，可能为空）。 */
  unit: string;
  /** 是否为计数器（counter）；派生 KPI 为 false/undefined。 */
  isCounter: boolean;
}

/** 按界面语言取友好名（与 MetricPickerModal 同款回退顺序）。 */
function metricNameOf(r: IndicatorInfo, isEn: boolean): string {
  return isEn ? r.enName || r.cnName || r.id : r.cnName || r.enName || r.id;
}

export interface MetricMetadataResult {
  /** 按编号查指标元数据；库里无此编号（旧别名等）返回 undefined。 */
  getMeta: (code: string) => MetricMeta | undefined;
  /** 指标库是否加载中。 */
  isLoading: boolean;
}

/**
 * 按制式拉全库指标元数据，返回「编号 → 元数据」查询器。
 */
export function useMetricMetadata(technology: TechnologyType): MetricMetadataResult {
  const isEn = useAppStore((s) => s.locale) === 'en-US';
  const deviceType = TECH_TO_DEVICE_TYPE[technology];

  // 取整库（自动翻页）：后端 page_size 封顶 1000，单页拉不全 ~1408 条，useAllIndicators 循环翻页取全。
  const { data, isLoading } = useAllIndicators(deviceType);

  const metaByCode = useMemo(() => {
    const map = new Map<string, MetricMeta>();
    for (const it of data?.items ?? []) {
      map.set(it.id, {
        name: metricNameOf(it, isEn),
        unit: it.unit ?? '',
        isCounter: Boolean(it.isCounter),
      });
    }
    return map;
  }, [data, isEn]);

  const getMeta = useMemo(
    () => (code: string) => metaByCode.get(code),
    [metaByCode],
  );

  return { getMeta, isLoading };
}

/** 指标的展示元数据（名字 + 单位 + 数值换算系数）。 */
export interface ResolvedMetricMeta {
  /** 已本地化的显示名（可直接渲染，不再过 t()）。 */
  name: string;
  /** 已本地化的单位串（可直接渲染，不再过 t()）；空串表示无单位。 */
  unit: string;
  /** 数值换算系数（仅旧精选 key 有，库指标恒为 1）。 */
  conversion: number;
  /** 取值/格式化用的 i18n 单位 key（仅旧精选 key 有，库指标为空 → 走原始单位串）。 */
  unitI18nKey?: string;
}

/**
 * 解析一个指标的展示元数据：
 *   1) 优先指标库元数据（编号命中）→ 名字/单位来自库，换算系数 1、无 i18n 单位 key；
 *   2) 回退老 getKPIConfigByKey（存量 symbolic 别名）→ i18n label/unit + 换算系数；
 *   3) 都查不到 → 名字回退编号本身，无单位、换算 1。
 */
export function resolveMetricMeta(
  metricKey: string,
  meta: MetricMetadataResult,
  t: (key: string) => string,
): ResolvedMetricMeta {
  const libMeta = meta.getMeta(metricKey);
  if (libMeta) {
    return { name: libMeta.name, unit: libMeta.unit, conversion: 1 };
  }
  const legacy = getKPIConfigByKey(metricKey);
  if (legacy) {
    return {
      name: t(legacy.label),
      unit: legacy.unit ? t(legacy.unit) : '',
      conversion: legacy.unitConversion ?? 1,
      unitI18nKey: legacy.unit,
    };
  }
  return { name: metricKey, unit: '', conversion: 1 };
}
