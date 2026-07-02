/**
 * useMetricMetadata - 首页 KPI 面板指标元数据取数（KPI-ALL-IND 阶段4）
 *
 * 改造目标（PR#XXX）：统一从后端指标库获取，与 PM 性能仪表板和 KPI 视图配置对齐。
 * 不再依赖前端硬编码的 kpi-config.ts。
 *
 * 背景：放开全部指标后，面板存的是指标编号（K/C 编号），名字/单位需要从指标库拉取。
 * 本 hook 按制式从指标库（GET /api/v1/indicators）拉全库元数据，
 * 建「编号 → {名字, 单位}」映射供面板渲染。
 *
 * - 名字：按界面语言取中文名 / 英文名（缺则回退另一种 / 编号）。
 * - 单位：取指标库的原始单位串（如 %/Mbps/ms），无需 i18n 转换。
 */

import { useMemo } from 'react';
import { useAllIndicators } from '@core/hooks/api/useIndicatorsLibrary';
import { useAppStore } from '@core/store/appStore';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import type { TechnologyType } from '@/pages/dashboard/kpi-config';

/** 制式 → 指标库设备类型（与配置页 KpiConfig 的 TECH_TO_DEVICE_TYPE 一致）。 */
const TECH_TO_DEVICE_TYPE: Record<TechnologyType, DeviceType> = {
  lte: 'ENB',
  nr: 'GNB',
  gsm: 'GSM',
};

export interface MetricMeta {
  /** 本地化显示名（中/英），来自指标库的 cnName/enName。 */
  name: string;
  /** 原始单位串（指标库 unit 字段，如 "%"/"Mbps"/"ms"），无需 i18n 转换。 */
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

  return useMemo(() => ({ getMeta, isLoading }), [getMeta, isLoading]);
}

/** 指标的展示元数据（已本地化的名字 + 单位）。 */
export interface ResolvedMetricMeta {
  /** 已本地化的显示名（来自指标库 cnName/enName，可直接渲染）。 */
  name: string;
  /** 原始单位串（来自指标库 unit，可直接渲染）；空串表示无单位。 */
  unit: string;
  /** 数值换算系数（恒为 1，库路径不做前端换算）。 */
  conversion: number;
}

/**
 * 解析一个指标的展示元数据。统一从后端指标库获取，与 PM 性能仪表板和 KPI 视图配置对齐。
 *
 *   1) 指标库（primary）→ canonicalize 后按 K/C 编号查库，名字/单位直接用库值，无需 t()。
 *   2) 查不到 → 名字回退 metricKey 本身，无单位，换算 1。
 */
export function resolveMetricMeta(
  metricKey: string,
  meta: MetricMetadataResult,
  _t: (key: string) => string,
): ResolvedMetricMeta {
  // canonicalize：DB 迁移前的存量 symbolic name 桥接到 K/C 编号。
  const libMeta = meta.getMeta(metricKey);
  if (libMeta) {
    return {
      name: libMeta.name,   // cnName/enName，已按 locale 选取，无需 t()
      unit: libMeta.unit,   // 原始单位字符串（%/Mbps/ms），无需 i18n 转换
      conversion: 1,
    };
  }
  return { name: metricKey, unit: '', conversion: 1 };
}
