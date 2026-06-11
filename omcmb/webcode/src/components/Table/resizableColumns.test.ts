import { describe, it, expect } from 'vitest';
import { applyColumnResize, MIN_RESIZE_WIDTH } from './resizableColumnsCore';

describe('applyColumnResize (#214 可拖拽列宽核心)', () => {
  it('设置指定列宽并四舍五入', () => {
    expect(applyColumnResize({}, 'standardPath', 260.6)).toEqual({ standardPath: 261 });
  });

  it('夹到最小列宽，拖不没', () => {
    expect(applyColumnResize({}, 'privatePath', 10)[`privatePath`]).toBe(MIN_RESIZE_WIDTH);
    expect(applyColumnResize({}, 'privatePath', -50, 80).privatePath).toBe(80);
  });

  it('不可变：返回新对象，保留其它列宽', () => {
    const prev = { a: 100, standardPath: 200 };
    const next = applyColumnResize(prev, 'standardPath', 320);
    expect(next).not.toBe(prev);
    expect(prev.standardPath).toBe(200); // 原对象不变
    expect(next).toEqual({ a: 100, standardPath: 320 });
  });

  it('自定义最小宽生效', () => {
    expect(applyColumnResize({}, 'x', 30, 120).x).toBe(120);
    expect(applyColumnResize({}, 'x', 300, 120).x).toBe(300);
  });
});
