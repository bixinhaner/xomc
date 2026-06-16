/**
 * 首页 KPI 配置「保存前」校验（issue #408，三皮肤共享层）
 *
 * 现象：配置页保存一张「空标题 + 0 指标」的图，前端无校验拦截，直接 PUT
 * 写进布局成垃圾项。本模块提供纯校验函数：每张图必须「标题非空 + 指标数 ≥ 1」，
 * 否则返回不合规清单（含图序号 + 标题）供各皮肤拦截保存并弹明确提示。
 *
 * 本模块不依赖 React / 网络 / antd / 任何皮肤组件，便于纯逻辑单测，三皮肤共用。
 */

/** 校验所需的最小图形状（v1/v2 的工作态 panel 都满足）。 */
export interface KpiPanelForValidation {
  /** 图标题（可能是 i18n key 或管理员输入的纯文本）。 */
  title: string;
  /** 该图选中的指标列表。 */
  metrics: string[];
}

/** 单张不合规图的明细。 */
export interface KpiPanelViolation {
  /** 图在数组中的序号（0 基）。 */
  index: number;
  /** 图标题（原样，空标题为空串）。 */
  title: string;
  /** 标题是否为空（去空白后）。 */
  titleEmpty: boolean;
  /** 是否一个指标都没选。 */
  noMetrics: boolean;
}

/** 校验结果。 */
export interface KpiPanelValidationResult {
  /** 是否全部合规（无违规图）。 */
  valid: boolean;
  /** 不合规图清单（合规则为空数组）。 */
  violations: KpiPanelViolation[];
}

/**
 * 校验一组图：每张图必须「标题非空（去首尾空白后） + 至少 1 个指标」。
 *
 * @param panels 待保存的图列表
 * @returns 校验结果；valid=false 时 violations 列出每张不合规图的序号/标题/原因
 */
export function validateKpiPanels(
  panels: KpiPanelForValidation[],
): KpiPanelValidationResult {
  const violations: KpiPanelViolation[] = [];
  panels.forEach((panel, index) => {
    const titleEmpty = panel.title.trim().length === 0;
    const noMetrics = panel.metrics.length === 0;
    if (titleEmpty || noMetrics) {
      violations.push({ index, title: panel.title, titleEmpty, noMetrics });
    }
  });
  return { valid: violations.length === 0, violations };
}

/**
 * 把不合规清单拼成一句给用户看的中文提示（指出哪张图、为什么不合规）。
 *
 * 序号对用户用 1 基（第 1 张 / 第 2 张）。各皮肤拿这句话喂自己的提示组件
 * （v1 antd message / v2 内联告警 / v3 对应组件）。
 *
 * @param violations validateKpiPanels 返回的违规清单
 * @returns 中文提示文案（无违规则返回空串）
 */
export function formatKpiPanelViolations(
  violations: KpiPanelViolation[],
): string {
  if (violations.length === 0) return '';
  const parts = violations.map((v) => {
    const reasons: string[] = [];
    if (v.titleEmpty) reasons.push('标题为空');
    if (v.noMetrics) reasons.push('未选指标');
    const label = v.titleEmpty ? '' : `「${v.title}」`;
    return `第 ${v.index + 1} 张图${label}（${reasons.join('、')}）`;
  });
  return `以下图表不合规，无法保存：${parts.join('；')}。请补全标题并至少选择 1 个指标。`;
}
