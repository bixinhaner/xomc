import type { MMLParamRef } from '@core/types/mml';

type TFn = (id: string, values?: Record<string, string | number>) => string;

interface ValueConstraint {
  type?: string;
  raw?: string;
  pattern?: string;
  min?: number;
  max?: number;
  min_length?: number;
  max_length?: number;
  options?: Array<{ label: string; value: string | number }>;
  values?: Array<string | number>;
  desc?: string;
  description?: string;
}

/**
 * 把后端 value_constraint.raw 这种简写形态人性化展开。
 * 例：
 *   "strint-[001-999]"   → "数字字符串，范围 001-999"
 *   "unsignedInt[0:256]" → "整数，范围 0-256"
 *   其它原样返回，由调用方决定是否再降级。
 */
function humanizeRaw(raw: string, t: TFn): string | null {
  const trimmed = raw.trim();

  let m = trimmed.match(/^strint-?\[(.+)\]$/i);
  if (m) return t('mml.console.hint.numericStringRange', { range: m[1] });

  m = trimmed.match(/^unsignedInt\[(\d+)\s*[:\-]\s*(\d+)\]$/i);
  if (m) return t('mml.console.hint.integerRange', { min: m[1], max: m[2] });

  m = trimmed.match(/^int\[(-?\d+)\s*[:\-]\s*(-?\d+)\]$/i);
  if (m) return t('mml.console.hint.integerRange', { min: m[1], max: m[2] });

  return null;
}

/**
 * 基于 MMLParamRef 的 valueConstraint / jsRegex / defaultValue 综合生成 placeholder。
 * 优先级：
 *   1. valueConstraint.raw（最权威的人类可读串，由 TR069 文档语义直出）
 *   2. valueConstraint.desc / description（自由文本说明）
 *   3. 按 valueType 拼装 min/max / min_length/max_length 范围
 *   4. jsRegex（兜底，规约形式给开发参考）
 *   5. 通用 fallback
 *
 * defaultValue 不为空时，在末尾追加 "（默认: X）" 提示。
 */
export function buildParamPlaceholder(ref: MMLParamRef, t: TFn): string {
  const c: ValueConstraint = (ref.valueConstraint ?? {}) as ValueConstraint;
  let main = '';

  if (typeof c.raw === 'string' && c.raw.trim()) {
    main = humanizeRaw(c.raw, t) ?? c.raw;
  } else if (typeof c.desc === 'string' && c.desc.trim()) {
    main = c.desc;
  } else if (typeof c.description === 'string' && c.description.trim()) {
    main = c.description;
  } else if (ref.valueType === 'number' || c.type === 'unsignedInt' || c.type === 'int') {
    if (typeof c.min === 'number' && typeof c.max === 'number') {
      main = t('mml.console.hint.integerRange', { min: c.min, max: c.max });
    } else if (typeof c.min === 'number') {
      main = t('mml.console.hint.integerMin', { min: c.min });
    } else if (typeof c.max === 'number') {
      main = t('mml.console.hint.integerMax', { max: c.max });
    } else {
      main = t('mml.console.inputNumber');
    }
  } else if (ref.valueType === 'string') {
    if (typeof c.min_length === 'number' && typeof c.max_length === 'number') {
      main = c.min_length === c.max_length
        ? t('mml.console.hint.stringFixedLength', { length: c.min_length })
        : t('mml.console.hint.stringLengthRange', { min: c.min_length, max: c.max_length });
    } else if (typeof c.max_length === 'number') {
      main = t('mml.console.hint.stringMaxLength', { max: c.max_length });
    } else if (typeof c.pattern === 'string' && c.pattern) {
      main = c.pattern;
    } else if (ref.jsRegex) {
      main = stripRegexDelimiters(ref.jsRegex);
    } else {
      main = t('mml.console.inputValue');
    }
  } else if (ref.valueType === 'enum') {
    main = t('common.pleaseSelect');
  } else {
    main = t('mml.console.inputValue');
  }

  if (ref.defaultValue) {
    return `${main}${t('mml.console.hint.defaultSuffix', { value: ref.defaultValue })}`;
  }
  return main;
}

// '/^[0-9]{3}$/' → '^[0-9]{3}$'，用于在 placeholder 里展示更干净的正则。
function stripRegexDelimiters(raw: string): string {
  const trimmed = raw.trim();
  if (trimmed.startsWith('/') && trimmed.endsWith('/') && trimmed.length > 2) {
    return trimmed.slice(1, -1);
  }
  return trimmed;
}
