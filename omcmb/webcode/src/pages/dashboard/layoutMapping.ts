/**
 * 首页 KPI 折线图区布局映射工具（issue #213 S2，纯逻辑）
 *
 * 职责：把「全局布局（读 S1 接口或回退内置默认）」转成首页渲染与批量取数所需的中间结构。
 * - 内置默认布局：由原写死的 kpi-config.ts 派生，作为配置读不到 / 为空 / 出错时的回退值，保证首页永不空白。
 * - collectMetrics：汇总当前制式所有可见图要画的指标，去重后用于一次批量取数。
 * - layoutToRows：把 panels 按网格坐标（y 升序、同行 x 升序）排成行，供 Row/Col 渲染（首页只读不可拖）。
 *
 * 本模块不依赖 React / 网络，便于纯逻辑单测。
 */

import type { KPILayout, KPILayoutPanel } from '@core/types/dashboard';
import type { TechnologyType, PanelType } from './kpi-config';
import { getPanelsForTech, getPanelConfig } from './kpi-config';

/** 12 列网格半宽（两图一行）。 */
const HALF_WIDTH = 6;
/** 12 列网格满宽（独占一行）。 */
const FULL_WIDTH = 12;

/**
 * 内置默认布局：由原写死的 kpi-config.ts 派生（与后端 seed 等价）。
 *
 * 用作配置读不到 / 为空 / 出错时的回退；默认值即现状，本片上线首页视觉零变化。
 * 网格坐标按既有 getPanelLayout 的行结构折算（LTE 2×3、NR 1×2、GSM 第一行 2 个 + 第二行 1 个占满）。
 *
 * @param tech 制式
 * @returns 该制式的回退布局
 */
export function buildDefaultLayout(tech: TechnologyType): KPILayout {
  const panelTypes = getPanelsForTech(tech);

  // GSM 末图占满整行（与原 getPanelLayout 的「第二行 1 个占满」一致）。
  const gsmFullLastIndex = tech === 'gsm' ? panelTypes.length - 1 : -1;

  const panels: KPILayoutPanel[] = panelTypes.map((panelType: PanelType, index) => {
    const config = getPanelConfig(tech, panelType);
    const metrics = config?.indicators.map((ind) => ind.key) ?? [];
    const isFull = index === gsmFullLastIndex;
    const w = isFull ? FULL_WIDTH : HALF_WIDTH;
    // 半宽图两个一行：x 在 0/6 间交替，y 按行号递增；满宽图独占一行 x=0。
    const row = isFull ? Math.ceil(index / 2) : Math.floor(index / 2);
    const x = isFull ? 0 : (index % 2) * HALF_WIDTH;
    const y = row * 8;
    return {
      title: config?.title ?? '',
      metrics,
      x,
      y,
      w,
      h: 8,
      chartType: 'line',
    };
  });

  return {
    tech,
    panels,
    updatedAt: '',
  };
}

/**
 * 解析当前制式要渲染的布局：有全局配置（且非空）用之，否则回退内置默认。
 *
 * @param tech 当前制式
 * @param layout 读 S1 接口得到的全局布局（可能 undefined / panels 为空）
 * @returns 实际渲染用布局
 */
export function resolveLayout(
  tech: TechnologyType,
  layout: KPILayout | undefined,
): KPILayout {
  if (layout && layout.panels.length > 0) {
    return layout;
  }
  return buildDefaultLayout(tech);
}

/**
 * 汇总一套布局里所有图要画的指标，去重（同一指标被多张图引用只取一次）。
 *
 * 用于把整页所有图的取数合并成一次批量取数（避免 N 张图各发请求）。
 *
 * @param panels 布局的图列表
 * @returns 去重后的指标 symbolic key 列表（保持首次出现顺序）
 */
export function collectMetrics(panels: KPILayoutPanel[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const panel of panels) {
    for (const metric of panel.metrics) {
      if (!seen.has(metric)) {
        seen.add(metric);
        result.push(metric);
      }
    }
  }
  return result;
}

/** layoutToRows 返回的一行（含其在 24 栅格里的列宽，供 antd Col 用）。 */
export interface LayoutRow {
  /** 这一行的图（按 x 升序）。 */
  panels: KPILayoutPanel[];
}

/**
 * 把 panels 按网格坐标排成行：先按 y 升序分行（同 y 为一行），同行内按 x 升序。
 *
 * 首页只读不可拖，故只需把存的坐标稳定还原成行布局给 Row/Col 渲染。
 *
 * @param panels 布局的图列表
 * @returns 行列表（每行的图按 x 升序）
 */
export function layoutToRows(panels: KPILayoutPanel[]): LayoutRow[] {
  const byY = new Map<number, KPILayoutPanel[]>();
  for (const panel of panels) {
    const row = byY.get(panel.y) ?? [];
    row.push(panel);
    byY.set(panel.y, row);
  }
  return Array.from(byY.keys())
    .sort((a, b) => a - b)
    .map((y) => ({
      panels: (byY.get(y) ?? []).slice().sort((a, b) => a.x - b.x),
    }));
}
