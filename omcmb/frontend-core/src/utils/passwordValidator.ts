/**
 * 密码复杂度校验工具
 *
 * 规则与后端 password_policy.go 保持一致：
 * RequireMix=true 时，密码需包含 4 类字符（数字/小写/大写/特殊字符）中的至少 3 种。
 *
 * @file frontend-core/src/utils/passwordValidator.ts
 */

/** 特殊字符集（与后端 SymbolChars 常量一致） */
const SYMBOL_CHARS = '.!@#$%^&*?';

/**
 * 统计密码中包含的字符类别数
 * @returns 包含的类别数（0-4）
 */
export function countPasswordClasses(password: string): number {
  if (!password) return 0;

  let classes = 0;

  // 数字
  if (/\d/.test(password)) classes++;

  // 小写字母
  if (/[a-z]/.test(password)) classes++;

  // 大写字母
  if (/[A-Z]/.test(password)) classes++;

  // 特殊字符
  const hasSymbol = password.split('').some((ch) => SYMBOL_CHARS.includes(ch));
  if (hasSymbol) classes++;

  return classes;
}

/**
 * 校验密码复杂度是否满足 "至少 3 种字符类型" 要求
 * @returns true 表示满足要求
 */
export function validatePasswordComplexity(password: string): boolean {
  return countPasswordClasses(password) >= 3;
}

/**
 * 获取密码中缺失的字符类别名称列表（用于提示用户）
 * @returns 缺失的类别名称数组
 */
export function getMissingPasswordClasses(password: string): string[] {
  if (!password) return ['digit', 'lowercase', 'uppercase', 'symbol'];

  const missing: string[] = [];

  if (!/\d/.test(password)) missing.push('digit');
  if (!/[a-z]/.test(password)) missing.push('lowercase');
  if (!/[A-Z]/.test(password)) missing.push('uppercase');

  const hasSymbol = password.split('').some((ch) => SYMBOL_CHARS.includes(ch));
  if (!hasSymbol) missing.push('symbol');

  return missing;
}

/**
 * Ant Design Form 规则：密码复杂度校验器
 *
 * 用法：
 * ```tsx
 * import { createPasswordComplexityRule } from '@core/utils/passwordValidator';
 *
 * <Form.Item
 *   name="password"
 *   rules={[
 *     { required: true, message: t('user.pleaseInputPassword') },
 *     createPasswordComplexityRule(t('system.security.passwordComplexityRequirement')),
 *   ]}
 * >
 *   <Input.Password />
 * </Form.Item>
 * ```
 */
export function createPasswordComplexityRule(
  message: string,
): { validator: (_: unknown, value: string) => Promise<void> } {
  return {
    validator: (_: unknown, value: string) => {
      // 空值由 required 规则处理，这里放行
      if (!value) return Promise.resolve();

      if (validatePasswordComplexity(value)) {
        return Promise.resolve();
      }

      return Promise.reject(new Error(message));
    },
  };
}

/** 密码长度默认值（与后端 defaultPwdMinLength/defaultPwdMaxLength 保持一致） */
export const DEFAULT_PASSWORD_MIN_LENGTH = 8;
export const DEFAULT_PASSWORD_MAX_LENGTH = 32;

/**
 * 校验密码长度是否在有效范围内
 * @param password 待校验密码
 * @param minLength 最小长度（默认 8）
 * @param maxLength 最大长度（默认 32）
 * @returns true 表示长度合法
 */
export function validatePasswordLength(
  password: string,
  minLength = DEFAULT_PASSWORD_MIN_LENGTH,
  maxLength = DEFAULT_PASSWORD_MAX_LENGTH,
): boolean {
  if (!password) return false;
  return password.length >= minLength && password.length <= maxLength;
}

/**
 * Ant Design Form 规则：密码长度校验器（支持动态长度配置）
 *
 * @param message 错误提示文案
 * @param minLength 最小长度（默认 8）
 * @param maxLength 最大长度（默认 32）
 */
export function createPasswordLengthRule(
  message: string,
  minLength = DEFAULT_PASSWORD_MIN_LENGTH,
  maxLength = DEFAULT_PASSWORD_MAX_LENGTH,
): { validator: (_: unknown, value: string) => Promise<void> } {
  return {
    validator: (_: unknown, value: string) => {
      // 空值由 required 规则处理，这里放行
      if (!value) return Promise.resolve();

      if (validatePasswordLength(value, minLength, maxLength)) {
        return Promise.resolve();
      }

      return Promise.reject(new Error(message));
    },
  };
}
