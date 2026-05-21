import { describe, it, expect } from 'vitest';
import type { InstanceRange } from '@core/types/mmlConsole';
import {
  validateInstanceLayer,
  findRangeByLayer,
  buildRangeHint,
} from '../instanceRangeValidation';

const fixedRange = (overrides: Partial<InstanceRange>): InstanceRange => ({
  layer: 1,
  rangeExpr: '1~3',
  rangeMin: 1,
  rangeMax: 3,
  dynamic: false,
  ...overrides,
});

describe('validateInstanceLayer — 空值与必填', () => {
  it('空值 + LST (非必填) → ok', () => {
    expect(validateInstanceLayer('', undefined, false)).toEqual({ ok: true });
  });

  it('纯空白 + 必填 → required 错误', () => {
    const r = validateInstanceLayer('   ', undefined, true);
    expect(r.ok).toBe(false);
    expect(r.errorKey).toBe('mml.console.instanceArity.errors.required');
  });

  it('空值 + 必填 + 有元数据 → required 错误（短路，不报范围）', () => {
    const r = validateInstanceLayer('', fixedRange({}), true);
    expect(r.ok).toBe(false);
    expect(r.errorKey).toBe('mml.console.instanceArity.errors.required');
  });
});

describe('validateInstanceLayer — 格式校验', () => {
  it('字母 → invalidFormat', () => {
    const r = validateInstanceLayer('abc', fixedRange({}), false);
    expect(r.ok).toBe(false);
    expect(r.errorKey).toBe('mml.console.instanceArity.errors.invalidFormat');
  });

  it('小数 → invalidFormat', () => {
    const r = validateInstanceLayer('1.5', fixedRange({}), false);
    expect(r.ok).toBe(false);
    expect(r.errorKey).toBe('mml.console.instanceArity.errors.invalidFormat');
  });

  it('负数 → invalidFormat', () => {
    const r = validateInstanceLayer('-1', fixedRange({}), false);
    expect(r.ok).toBe(false);
    expect(r.errorKey).toBe('mml.console.instanceArity.errors.invalidFormat');
  });

  it('前后空白 + 整数 → trim 后通过', () => {
    expect(validateInstanceLayer('  2  ', fixedRange({}), true).ok).toBe(true);
  });
});

describe('validateInstanceLayer — 范围校验', () => {
  it('在范围内 → ok', () => {
    expect(validateInstanceLayer('2', fixedRange({ rangeMin: 1, rangeMax: 3 }), true).ok).toBe(true);
  });

  it('刚好等于下界 → ok（闭区间）', () => {
    expect(validateInstanceLayer('1', fixedRange({ rangeMin: 1, rangeMax: 3 }), true).ok).toBe(true);
  });

  it('刚好等于上界 → ok（闭区间）', () => {
    expect(validateInstanceLayer('3', fixedRange({ rangeMin: 1, rangeMax: 3 }), true).ok).toBe(true);
  });

  it('小于下界 → tooSmall 带 min 占位符', () => {
    const r = validateInstanceLayer('0', fixedRange({ rangeMin: 1, rangeMax: 3 }), true);
    expect(r.ok).toBe(false);
    expect(r.errorKey).toBe('mml.console.instanceArity.errors.tooSmall');
    expect(r.errorValues).toEqual({ min: 1 });
  });

  it('大于上界 → tooLarge 带 max 占位符', () => {
    const r = validateInstanceLayer('4', fixedRange({ rangeMin: 1, rangeMax: 3 }), true);
    expect(r.ok).toBe(false);
    expect(r.errorKey).toBe('mml.console.instanceArity.errors.tooLarge');
    expect(r.errorValues).toEqual({ max: 3 });
  });

  it('range = undefined → 仅做格式校验，跳过范围', () => {
    expect(validateInstanceLayer('99999', undefined, true).ok).toBe(true);
  });

  it('dynamic=true → 跳过上界检查；下界仍校验', () => {
    const range = fixedRange({ rangeMin: 0, rangeMax: null, dynamic: true, nSource: 'X' });
    expect(validateInstanceLayer('99999', range, true).ok).toBe(true);
    // 下界仍生效（rangeMin=0 时所有非负整数都通过；改 rangeMin=2 验证）
    const range2 = fixedRange({ rangeMin: 2, rangeMax: null, dynamic: true, nSource: 'X' });
    const r = validateInstanceLayer('1', range2, true);
    expect(r.ok).toBe(false);
    expect(r.errorKey).toBe('mml.console.instanceArity.errors.tooSmall');
  });

  it('rangeMin=null + rangeMax 非空 → 只校验上界', () => {
    const range = fixedRange({ rangeMin: null, rangeMax: 5 });
    expect(validateInstanceLayer('0', range, true).ok).toBe(true);
    expect(validateInstanceLayer('6', range, true).ok).toBe(false);
  });
});

describe('findRangeByLayer', () => {
  const meta: InstanceRange[] = [
    fixedRange({ layer: 1, rangeExpr: '1~3' }),
    fixedRange({ layer: 3, rangeExpr: '0~N', dynamic: true, rangeMax: null }),
  ];

  it('元数据为 undefined → undefined', () => {
    expect(findRangeByLayer(undefined, 1)).toBeUndefined();
  });

  it('元数据为空数组 → undefined', () => {
    expect(findRangeByLayer([], 1)).toBeUndefined();
  });

  it('按 layer 字段精确查找（不按下标）', () => {
    expect(findRangeByLayer(meta, 1)?.rangeExpr).toBe('1~3');
    expect(findRangeByLayer(meta, 3)?.dynamic).toBe(true);
  });

  it('跳层（layer=2 在 metadata 中缺失）→ undefined', () => {
    expect(findRangeByLayer(meta, 2)).toBeUndefined();
  });
});

describe('buildRangeHint', () => {
  // 简化版 t：直接 echo key + JSON values（与组件测试 mock 同形态）
  const t = (key: string, values?: Record<string, unknown>) =>
    values ? `${key}|${JSON.stringify(values)}` : key;

  it('元数据 undefined → undefined', () => {
    expect(buildRangeHint(undefined, t)).toBeUndefined();
  });

  it('静态范围 → 单行 range key 带 min/max', () => {
    const hint = buildRangeHint(fixedRange({ rangeMin: 1, rangeMax: 3 }), t);
    expect(hint).toContain('mml.console.instanceArity.hint.range');
    expect(hint).toContain('"min":1');
    expect(hint).toContain('"max":3');
  });

  it('dynamic=true → rangeDynamic key 带 source', () => {
    const hint = buildRangeHint(
      fixedRange({ rangeMin: 0, rangeMax: null, dynamic: true, nSource: 'PLMNListNumberOfEntries' }),
      t,
    );
    expect(hint).toContain('mml.console.instanceArity.hint.rangeDynamic');
    expect(hint).toContain('PLMNListNumberOfEntries');
  });

  it('description 存在 → 拼到第二行', () => {
    const hint = buildRangeHint(
      fixedRange({ rangeMin: 1, rangeMax: 3, description: '载波实例（i=1:载波1）' }),
      t,
    );
    expect(hint).toContain('载波实例（i=1:载波1）');
    // 多行用 \n 分隔
    expect(hint?.split('\n').length).toBeGreaterThanOrEqual(2);
  });

  it('只有 rangeExpr（无 min/max 解析结果）→ rangePrefix fallback', () => {
    const hint = buildRangeHint(
      fixedRange({ rangeMin: null, rangeMax: null, rangeExpr: '0|1|2' }),
      t,
    );
    expect(hint).toContain('0|1|2');
    expect(hint).toContain('mml.console.instanceArity.hint.rangePrefix');
  });
});
