/**
 * T-0193 自选设备下钻到小区/PLMN — 小区对象类型 + object_ldn 友好名共享工具。
 *
 * 设计：~/Documents/notes/PM功能设计/pm-cell-plmn-drilldown-design.md
 *
 * 一台设备的性能数据按「设备 + 小区 + PLMN」分行，每行唯一标识里含一个
 * `object_ldn` 字符串，形如 `Cellid=111172245,PLMN=46068`。本模块：
 *   - MetricObject：列小区接口返回的一项（原始 object_ldn + 拆好的 cellId/plmn）。
 *   - formatObjectLdn：把 object_ldn 渲染成可读友好名「小区111172245 · PLMN46068」，
 *     供下钻选择器与设备列表分线共用（避免各写一份）。
 */

/** 列小区接口返回的一项（前端视角，已 camelCase）。 */
export interface MetricObject {
  /** 完整 object_ldn 字符串，白名单/过滤用它（如 `Cellid=111172245,PLMN=46068`）。 */
  objectLdn: string;
  /** 拆解出的小区 ID（友好名用），缺段时 undefined。 */
  cellId?: string;
  /** 拆解出的 PLMN（友好名用），缺段时 undefined。 */
  plmn?: string;
}

/** 后端 wire 形态（http.ts 自动 camelCase，这里仅作类型声明备查）。 */
export interface BackendMetricObject {
  object_ldn: string;
  cell_id?: string;
  plmn?: string;
}

/** 从 object_ldn 字符串拆出小区 ID（`Cellid=数字`，大小写不敏感）。缺则 undefined。 */
export function parseCellId(objectLdn: string | null | undefined): string | undefined {
  if (!objectLdn) return undefined;
  const m = /Cellid=([^,]+)/i.exec(objectLdn);
  const v = m?.[1]?.trim();
  return v ? v : undefined;
}

/** 从 object_ldn 字符串拆出 PLMN（`PLMN=数字`，大小写不敏感）。缺则 undefined。 */
export function parsePlmn(objectLdn: string | null | undefined): string | undefined {
  if (!objectLdn) return undefined;
  const m = /PLMN=([^,]+)/i.exec(objectLdn);
  const v = m?.[1]?.trim();
  return v ? v : undefined;
}

/**
 * object_ldn → 可读友好名「小区111172245 · PLMN46068」。
 *   - 两段都有：`小区{cell} · PLMN{plmn}`
 *   - 只有小区：`小区{cell}`
 *   - 只有 PLMN：`PLMN{plmn}`
 *   - 两段都缺（空串 / 无法识别）：回退原始 object_ldn 字符串（非空时）或空串。
 *
 * 纯展示用，不做 i18n（小区/PLMN 是网管通用术语，前缀固定）。
 */
export function formatObjectLdn(objectLdn: string | null | undefined): string {
  const cell = parseCellId(objectLdn);
  const plmn = parsePlmn(objectLdn);
  const parts: string[] = [];
  if (cell) parts.push(`小区${cell}`);
  if (plmn) parts.push(`PLMN${plmn}`);
  if (parts.length > 0) return parts.join(' · ');
  // 无法拆解 → 回退原串（保留可见信息），空则空串。
  return (objectLdn ?? '').trim();
}

/** 设备尾号（SN 末 6 位，不足则全量）— 多设备分线时线名前缀用。 */
export function deviceSnTail(sn: string | null | undefined, tailLen = 6): string {
  const s = (sn ?? '').trim();
  if (!s) return '';
  return s.length > tailLen ? s.slice(-tailLen) : s;
}

/**
 * 设备列表分线的系列名：「设备尾号 · 小区X · PLMNY」。
 *   - 有 object_ldn：`{尾号} · {友好名}`
 *   - 无 object_ldn（缺小区信息，兜底单线）：仅 `{尾号}`（退化为按设备）。
 *   - 连尾号都缺：回退友好名或固定占位由调用方处理。
 */
export function buildDeviceSeriesName(
  sn: string | null | undefined,
  objectLdn: string | null | undefined,
): string {
  const tail = deviceSnTail(sn);
  const friendly = formatObjectLdn(objectLdn);
  if (tail && friendly) return `${tail} · ${friendly}`;
  if (tail) return tail;
  return friendly;
}
