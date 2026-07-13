/**
 * 首页 KPI 配置页编辑逻辑（issue #213 S3，纯逻辑）
 *
 * 职责：把配置页里「增删图 / 改标题 / 选指标 / 拖拽网格」这些编辑动作，
 * 在一份「工作态布局」（带前端临时 id 的 panel 数组）上做不可变更新，
 * 并提供「按制式过滤指标库」「网格回写坐标」「转存盘形状」等纯函数。
 *
 * 本模块不依赖 React / 网络 / antd / react-grid-layout，便于纯逻辑单测。
 */

import type { KPILayoutPanel } from '@core/types/dashboard';
import type { KPIDefinitionItem, KPITechDefinitions } from '@core/types/dashboard';
import type { TechnologyType } from './kpi-config';

/** 12 列网格半宽（两图一行）。 */
export const HALF_WIDTH = 6;
/** 12 列网格满宽（独占一行）。 */
export const FULL_WIDTH = 12;
/** 新增图默认高度（与 seed / 回退默认一致）。 */
export const DEFAULT_HEIGHT = 8;

/**
 * 工作态 panel：在持久化形状（KPILayoutPanel）上加一个前端临时 id，
 * 供 react-grid-layout 的 layout item key（i）与列表 React key 用。
 * 存盘时剥掉 id（见 toSavePanels）。
 */
export interface WorkingPanel extends KPILayoutPanel {
  /** 前端临时唯一 id（不持久化）。 */
  id: string;
}

let panelIdSeq = 0;
/** 生成前端临时 panel id（不持久化，仅用于 RGL/React key）。 */
export function nextPanelId(): string {
  panelIdSeq += 1;
  return `panel-${Date.now()}-${panelIdSeq}`;
}

/**
 * 把读到的全局布局 panels 转成工作态（补 id）。
 *
 * @param panels 读 S1 接口或回退默认得到的 panels
 * @returns 带临时 id 的工作态 panels
 */
export function toWorkingPanels(panels: KPILayoutPanel[]): WorkingPanel[] {
  return panels.map((p) => ({ ...p, metrics: [...p.metrics], id: nextPanelId() }));
}

/**
 * 把工作态 panels 转回存盘形状（剥掉前端临时 id）。
 *
 * @param panels 工作态 panels
 * @returns 持久化 panels（与后端一致：title/metrics/x,y/w,h/chartType）
 */
export function toSavePanels(panels: WorkingPanel[]): KPILayoutPanel[] {
  return panels.map(({ id: _id, ...rest }) => ({ ...rest, metrics: [...rest.metrics] }));
}

/**
 * 按制式取指标库定义条目（配置页指标选择器只列当前制式指标）。
 *
 * 指标库按制式分组（lte / nr / gsm），LTE tab 只返回 lte 组（4G/ENB 指标），
 * 不混入 5G/NR、2G/GSM——杜绝跨制式塞错。
 *
 * @param technologies 指标库按制式分组的定义
 * @param tech 当前制式
 * @returns 该制式的指标定义列表（无该制式则空）
 */
export function definitionsForTech(
  technologies: KPITechDefinitions[] | undefined,
  tech: TechnologyType,
): KPIDefinitionItem[] {
  if (!technologies) return [];
  const group = technologies.find((g) => g.tech === tech);
  return group?.items ?? [];
}

/** 指标选择器的一个选项。 */
export interface MetricOption {
  /** 指标 symbolic key（存进 panel.metrics）。 */
  value: string;
  /** 展示名（指标库中文名，缺则回退 key）。 */
  label: string;
  /** 单位（如 % / Mbps）。 */
  unit: string;
  /** 库内无对应 KPI（none 项）→ 置灰不可选。 */
  disabled: boolean;
}

/**
 * 把当前制式指标库条目转成选择器选项；不可用（available=false）的置灰。
 *
 * @param items 当前制式指标库定义
 * @returns 选择器选项（不可用项 disabled=true）
 */
export function toMetricOptions(items: KPIDefinitionItem[]): MetricOption[] {
  return items.map((it) => ({
    value: it.key,
    label: it.cn_name || it.key,
    unit: it.unit,
    disabled: !it.available,
  }));
}

/**
 * 新增一张空图：插入布局最前，默认半宽、标题空、无指标。
 *
 * 落位策略：新图放在左上角，旧图按当前 y/x 视觉顺序排列，再从左到右、
 * 从上到下依次紧凑排布。旧图宽高保持不变，只调整 x/y；存盘后首页也会
 * 按 panels 数组顺序将新图渲染在最前。
 *
 * @param panels 现有工作态 panels
 * @param title 默认标题（i18n key 或用户纯文本）
 * @returns 新图置首后的新数组（不可变）
 */
export function addPanel(panels: WorkingPanel[], title = ''): WorkingPanel[] {
  const added: WorkingPanel = {
    id: nextPanelId(),
    title,
    metrics: [],
    x: 0,
    y: 0,
    w: HALF_WIDTH,
    h: DEFAULT_HEIGHT,
    chartType: 'line',
  };

  const ordered = panels
    .slice()
    .sort((a, b) => a.y - b.y || a.x - b.x);

  return packPanels([added, ...ordered]);
}

/** 按数组顺序从左到右、从上到下紧凑排位，保留每张图的宽高。 */
function packPanels(panels: WorkingPanel[]): WorkingPanel[] {
  const skyline = Array<number>(FULL_WIDTH).fill(0);

  const place = (panel: WorkingPanel): WorkingPanel => {
    const width = Math.max(1, Math.min(FULL_WIDTH, panel.w));
    let bestX = 0;
    let bestY = Number.POSITIVE_INFINITY;

    for (let x = 0; x <= FULL_WIDTH - width; x += 1) {
      const y = Math.max(...skyline.slice(x, x + width));
      if (y < bestY) {
        bestX = x;
        bestY = y;
      }
    }

    const bottom = bestY + panel.h;
    for (let x = bestX; x < bestX + width; x += 1) {
      skyline[x] = bottom;
    }
    return { ...panel, x: bestX, y: bestY };
  };

  return panels.map(place);
}

/**
 * 删除一张图。
 *
 * @param panels 现有工作态 panels
 * @param id 要删的图 id
 * @returns 删除后的新数组（不可变）
 */
export function removePanel(panels: WorkingPanel[], id: string): WorkingPanel[] {
  const remaining = panels.filter((p) => p.id !== id);
  return remaining.length === panels.length ? panels : packPanels(remaining);
}

/**
 * 改一张图的标题。
 *
 * @param panels 现有工作态 panels
 * @param id 目标图 id
 * @param title 新标题
 * @returns 更新后的新数组（不可变）
 */
export function updatePanelTitle(
  panels: WorkingPanel[],
  id: string,
  title: string,
): WorkingPanel[] {
  return panels.map((p) => (p.id === id ? { ...p, title } : p));
}

/**
 * 改一张图选中的指标（多选 = 一图多指标）。
 *
 * @param panels 现有工作态 panels
 * @param id 目标图 id
 * @param metrics 新的指标 symbolic key 列表
 * @returns 更新后的新数组（不可变）
 */
export function updatePanelMetrics(
  panels: WorkingPanel[],
  id: string,
  metrics: string[],
): WorkingPanel[] {
  return panels.map((p) => (p.id === id ? { ...p, metrics: [...metrics] } : p));
}

/** react-grid-layout 回传的单个 layout item（仅取用到的字段）。 */
export interface GridLayoutItem {
  i: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

/**
 * 拖拽 / 拉伸后，把 react-grid-layout 回传的坐标写回工作态 panels（按 id 对齐）。
 *
 * 只更新 x/y/w/h，标题与指标不动；若 layout 与当前 panels 的 id 集合不一致，
 * 说明这是增删卡片前的过期回调，整次忽略，避免旧坐标覆盖刚完成的重新排位。
 *
 * @param panels 现有工作态 panels
 * @param layout RGL 回传的 layout items
 * @returns 写回坐标后的新数组（不可变）
 */
export function applyGridLayout(
  panels: WorkingPanel[],
  layout: GridLayoutItem[],
): WorkingPanel[] {
  const panelIds = new Set(panels.map((p) => p.id));
  const layoutIds = new Set(layout.map((item) => item.i));
  if (
    panelIds.size !== layoutIds.size
    || [...panelIds].some((id) => !layoutIds.has(id))
  ) {
    return panels;
  }

  const byId = new Map(layout.map((l) => [l.i, l]));
  return panels.map((p) => {
    const g = byId.get(p.id);
    if (!g) return p;
    return { ...p, x: g.x, y: g.y, w: g.w, h: g.h };
  }).sort((a, b) => a.y - b.y || a.x - b.x);
}

/**
 * 由工作态 panels 生成 react-grid-layout 的 layout 数组（每图一项，i=id）。
 *
 * @param panels 工作态 panels
 * @returns RGL layout items
 */
export function toGridLayout(panels: WorkingPanel[]): GridLayoutItem[] {
  return panels.map((p) => ({ i: p.id, x: p.x, y: p.y, w: p.w, h: p.h }));
}
