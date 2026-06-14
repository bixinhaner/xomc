/**
 * instanceRangeValidation — R-4.1.1 多层 {i} 输入的范围校验纯函数。
 *
 * 方案：docs/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md §R-4.1.1
 *
 * 角色：InstanceArityInput 在渲染时按层调用 validateInstanceLayer，得到 ok/errorKey/errorValues
 * 三元组 → 驱动 Antd Input `status="error"` 红框 + Tooltip 错误文案。
 *
 * 为什么独立成文件：纯函数，无 React 依赖，单测干净；UI 文件只负责把校验结果映射为视觉。
 */

import type { InstanceRange } from '@core/types/mmlConsole';
import type { TranslateFn } from '@/hooks/useT';

/**
 * 校验单层 {i} 输入是否合规。
 *
 * 校验顺序（短路）：
 *   1. 空值 → 必填时 required 错误；非必填（LST）时 ok
 *   2. 格式 → 必须是十进制非负整数；含小数 / 字母 / 负号 → invalidFormat
 *   3. 下界 → range.rangeMin 非 null 且 value < rangeMin → tooSmall
 *   4. 上界 → !dynamic 且 range.rangeMax 非 null 且 value > rangeMax → tooLarge
 *
 * 元数据兜底：
 *   - range = undefined（v1 catalog / 当前层无 metadata）→ 仅做 1+2，跳过范围检查
 *   - dynamic = true（上限由 nSource 运行时决定）→ 跳过上界检查；下界仍校验
 *
 * 返回：errorKey 是 i18n key，errorValues 是 react-intl 占位符 — UI 侧 t() 调用合成最终文案。
 */
export interface InstanceLayerValidation {
  ok: boolean;
  errorKey?: string;
  errorValues?: Record<string, string | number>;
}

export function validateInstanceLayer(
  rawValue: string,
  range: InstanceRange | undefined,
  isRequired: boolean,
): InstanceLayerValidation {
  const value = rawValue.trim();

  if (value === '') {
    if (isRequired) {
      return { ok: false, errorKey: 'mml.console.instanceArity.errors.required' };
    }
    return { ok: true };
  }

  if (!/^[0-9]+$/.test(value)) {
    return { ok: false, errorKey: 'mml.console.instanceArity.errors.invalidFormat' };
  }

  if (!range) {
    return { ok: true };
  }

  const n = parseInt(value, 10);

  if (range.rangeMin !== null && n < range.rangeMin) {
    return {
      ok: false,
      errorKey: 'mml.console.instanceArity.errors.tooSmall',
      errorValues: { min: range.rangeMin },
    };
  }

  if (!range.dynamic && range.rangeMax !== null && n > range.rangeMax) {
    return {
      ok: false,
      errorKey: 'mml.console.instanceArity.errors.tooLarge',
      errorValues: { max: range.rangeMax },
    };
  }

  return { ok: true };
}

/**
 * 按 1-based layer 字段查找当前层的元数据。
 *
 * 不用数组下标的原因：spec 解析可能跳层（某层无规则）；按下标会错位。
 * 找不到返回 undefined → 校验降级为"必填+整数格式"基础检查。
 */
export function findRangeByLayer(
  meta: InstanceRange[] | undefined,
  layer: number,
): InstanceRange | undefined {
  if (!meta || meta.length === 0) return undefined;
  return meta.find((r) => r.layer === layer);
}

/**
 * 生成 Tooltip 的"范围提示"文案（始终展示，与 error 状态独立）。
 *
 * 用户视角：把 spec 原文 description / 范围表达式直接呈现给操作者，避免"凭空猜数字"。
 * 返回 undefined 时上层不渲染 Tooltip 内容。
 */
export function buildRangeHint(
  range: InstanceRange | undefined,
  t: TranslateFn,
): string | undefined {
  if (!range) return undefined;

  const lines: string[] = [];

  if (range.dynamic) {
    lines.push(
      t('mml.console.instanceArity.hint.rangeDynamic', {
        min: range.rangeMin ?? 0,
        source: range.nSource ?? '?',
      }),
    );
  } else if (range.rangeMin !== null && range.rangeMax !== null) {
    lines.push(
      t('mml.console.instanceArity.hint.range', {
        min: range.rangeMin,
        max: range.rangeMax,
      }),
    );
  } else if (range.rangeExpr) {
    lines.push(`${t('mml.console.instanceArity.hint.rangePrefix')} ${range.rangeExpr}`);
  }

  if (range.description) {
    lines.push(range.description);
  }

  return lines.length > 0 ? lines.join('\n') : undefined;
}
