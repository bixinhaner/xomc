import type { ParameterType, ParameterConstraints } from '@core/types/deviceParameter';

export interface QuickSettingsInstanceContext {
  networkType: string;
  fapInstance: number;
  cellInstance?: number;
}

/**
 * 校验单个参数输入值。返回错误信息字符串或 null（表示通过）。
 *
 * 复制自 ParameterTreeTab/ParameterEditModal.tsx 的 validateValue（保持行为一致，避免双源漂移）。
 */
export function validateValue(
  value: string,
  parameterType: ParameterType,
  constraints?: ParameterConstraints,
): string | null {
  if (!value && parameterType !== 'string') {
    return '请输入值';
  }

  if (parameterType === 'int') {
    const num = Number(value);
    if (!Number.isInteger(num)) return '请输入整数';
    if (constraints?.minValue !== undefined && num < constraints.minValue) {
      return `最小值为 ${constraints.minValue}`;
    }
    if (constraints?.maxValue !== undefined && num > constraints.maxValue) {
      return `最大值为 ${constraints.maxValue}`;
    }
  }

  if (parameterType === 'unsignedInt') {
    const num = Number(value);
    if (!Number.isInteger(num) || num < 0) return '请输入非负整数';
    if (constraints?.minValue !== undefined && num < constraints.minValue) {
      return `最小值为 ${constraints.minValue}`;
    }
    if (constraints?.maxValue !== undefined && num > constraints.maxValue) {
      return `最大值为 ${constraints.maxValue}`;
    }
  }

  if (parameterType === 'string' && constraints) {
    // 后端 ParamMapping.MinValue/MaxValue 字段对 string 类型语义为"长度"（XML 同字段名
    // 复用，按 parameterType 解释）。前端兼容显式 maxLength/minLength 优先，未给时退回
    // 用 minValue/maxValue 当 length 边界。
    const maxLen = constraints.maxLength ?? constraints.maxValue;
    const minLen = constraints.minLength ?? constraints.minValue;
    if (maxLen !== undefined && value.length > maxLen) {
      return `最大长度为 ${maxLen}`;
    }
    if (minLen !== undefined && value.length < minLen) {
      return `最小长度为 ${minLen}`;
    }
    if (constraints.pattern) {
      try {
        const re = new RegExp(constraints.pattern);
        if (!re.test(value)) return `不匹配模式: ${constraints.pattern}`;
      } catch {
        // ignore invalid regex
      }
    }
  }

  if (constraints?.enumValues && constraints.enumValues.length > 0) {
    if (!constraints.enumValues.includes(value)) {
      return `允许的值: ${constraints.enumValues.join(', ')}`;
    }
  }

  return null;
}

interface ApplyInstanceContextOptions {
  preserveTrailingInstance?: boolean;
}

/**
 * 根据当前网络制式把快速设置路径中的实例占位符解析成具体路径。
 *
 * - LTE: 仅替换最外层 FAPService.{i}
 * - NR: 先解析 FAPService / CellConfig 两层实例，再把更深层未区分的列表实例保守落到 1
 *
 * preserveTrailingInstance 用于多实例 objectPath，保留末尾那层 {i}. 给表格行实例继续拼接。
 */
export function applyInstanceContext(
  path: string,
  context: QuickSettingsInstanceContext,
  options?: ApplyInstanceContextOptions,
): string {
  const preserveTrailingInstance = options?.preserveTrailingInstance && path.endsWith('{i}.');
  const trailingPlaceholder = '__QS_TRAILING_INSTANCE__';

  let resolved = preserveTrailingInstance
    ? `${path.slice(0, -4)}${trailingPlaceholder}.`
    : path;

  if (context.networkType === 'nr') {
    const cellInstance = context.cellInstance ?? 1;
    resolved = resolved.replace('FAPService.{i}', `FAPService.${context.fapInstance}`);
    resolved = resolved.replace('CellConfig.{i}', `CellConfig.${cellInstance}`);
    resolved = resolved.replace(/\{i\}/g, '1');
  } else {
    resolved = resolved.replace('{i}', String(context.fapInstance));
  }

  if (preserveTrailingInstance) {
    resolved = resolved.replace(trailingPlaceholder, '{i}');
  }

  return resolved;
}
