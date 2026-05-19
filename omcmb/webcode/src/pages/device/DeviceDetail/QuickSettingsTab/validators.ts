import type { ParameterType, ParameterConstraints } from '@core/types/deviceParameter';

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

/**
 * 把 group.params 的 standardPath 中 {i} 占位符按外层 FAPService 实例号替换。
 *
 * - ENB 路径含一层 {i}（FAPService.{i}）→ 用 fapInstance 替换
 * - GNB 路径已写死 FAPService.1，内层 CellConfig.{i} 暂保持原样（v1 不渲染 NR 小区下拉）
 *
 * 多实例分组的内层 {i} 由表格行驱动，调用方在拼接 objectPath 时按 row index 处理，不走本函数。
 */
export function applyFapInstance(path: string, fapInstance: number): string {
  // 仅替换第一个 {i}（最外层 FAPService）。GNB 路径无 FAPService.{i}（已是 FAPService.1），不受影响。
  return path.replace('{i}', String(fapInstance));
}
