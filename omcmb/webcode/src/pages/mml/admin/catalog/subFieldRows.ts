/**
 * subFieldRows — AddSubFieldsModal「添加 PATH」下拉交互的纯状态逻辑。
 *
 * 抽成纯函数便于单测（AntD Select 的 portal/虚拟列表在 jsdom 里难以稳定驱动，
 * 故只测可观察的状态变换：追加 / 删除 / 计算下拉排除集）。交互规则（2026-06）：
 *   1. 隐藏已选：下拉只展示未选项 → excludeIds = 已保存 ∪ 已入表
 *   2. 连续添加：每次选中即 appendRows 入表，Select 自身不留标签
 *   3. 单独删除：removeRow 逐行删（替代下拉一键清除 allowClear）
 */
import type { StandardParamView } from '@core/types/mmlAdmin';
import { deriveLabel } from './deriveLabel';

export interface PreviewRow {
  paramId: string;
  standardPath: string;
  description: string;
  /** autofill：path 末段 UPPER_SNAKE */
  mmlCode: string;
  /** 单值显示名（去多语言）；提交仍只传 standardPathIds，由后端派生。 */
  label: string;
  access: string;
  dataType: string;
}

/** 把 standard_path 末段转成 UPPER_SNAKE_CASE。与后端 derivePathLeafCode 同算法。 */
export function deriveMmlCode(path: string): string {
  const idx = path.lastIndexOf('.');
  const leaf = idx >= 0 ? path.slice(idx + 1) : path;
  let out = '';
  let prevLower = false;
  for (let i = 0; i < leaf.length; i++) {
    const ch = leaf[i];
    const isUpper = ch >= 'A' && ch <= 'Z';
    const isLower = ch >= 'a' && ch <= 'z';
    const isDigit = ch >= '0' && ch <= '9';
    if (i > 0 && prevLower && isUpper) out += '_';
    if (ch === '_' || ch === '-') {
      out += '_';
    } else if (isUpper || isDigit) {
      out += ch;
    } else if (isLower) {
      out += ch.toUpperCase();
    }
    prevLower = isLower;
  }
  return out;
}

/** 由 standard_param 记录派生一行预览（mml_code / label autofill）。 */
export function buildRow(sp: StandardParamView): PreviewRow {
  const mmlCode = deriveMmlCode(sp.standardPath);
  return {
    paramId: sp.id,
    standardPath: sp.standardPath,
    description: sp.description,
    mmlCode,
    label: sp.description || deriveLabel(sp.standardPath) || mmlCode,
    access: sp.access,
    dataType: sp.dataType,
  };
}

/**
 * 追加新选中的记录到预览表，保留已有行（含用户的 inline 编辑），跳过已存在的
 * paramId（规则 2：连续添加；下拉已排除已选，重复入参是防御性去重）。
 */
export function appendRows(prev: PreviewRow[], records: StandardParamView[]): PreviewRow[] {
  const have = new Set(prev.map((r) => r.paramId));
  const additions: PreviewRow[] = [];
  for (const sp of records) {
    if (have.has(sp.id)) continue;
    have.add(sp.id);
    additions.push(buildRow(sp));
  }
  return additions.length === 0 ? prev : [...prev, ...additions];
}

/** 按 paramId 移除一行（规则 3：单独删除按钮）。 */
export function removeRow(prev: PreviewRow[], paramId: string): PreviewRow[] {
  return prev.filter((r) => r.paramId !== paramId);
}

/** 下拉需排除的 id 集合 = 命令已绑定的 ∪ 本次会话已入表的（规则 1：隐藏已选）。 */
export function computeExcludeIds(existingPathIds: string[], rows: PreviewRow[]): string[] {
  return [...existingPathIds, ...rows.map((r) => r.paramId)];
}
