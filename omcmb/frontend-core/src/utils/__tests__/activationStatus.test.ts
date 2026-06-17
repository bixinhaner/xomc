import { describe, it, expect } from 'vitest';
import { activationStatusOf } from '../activationStatus';

describe('activationStatusOf — 三皮肤共享的"激活状态"判定单一来源', () => {
  it('严格 === "1" → active', () => {
    expect(activationStatusOf('1')).toBe('active');
  });

  it('严格 === "0" → inactive', () => {
    expect(activationStatusOf('0')).toBe('inactive');
  });

  it('空/null/undefined/"unknown" → null（UI 应渲染占位符 "-" 或 "—"）', () => {
    expect(activationStatusOf(undefined)).toBeNull();
    expect(activationStatusOf(null)).toBeNull();
    expect(activationStatusOf('')).toBeNull();
    expect(activationStatusOf('unknown')).toBeNull();
  });

  it('其它非 "1" 字符串一律按未激活展示（避免详情页回显 raw 字符串与列表脱节）', () => {
    // 后端理论上不会返回这些值,但若出现也不应让它们以 raw 形态出现在 UI 上
    expect(activationStatusOf('2')).toBe('inactive');
    expect(activationStatusOf('active')).toBe('inactive');
    expect(activationStatusOf('inactive')).toBe('inactive');
    expect(activationStatusOf('foo')).toBe('inactive');
  });
});
