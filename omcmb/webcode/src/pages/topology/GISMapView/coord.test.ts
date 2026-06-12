import { describe, it, expect } from 'vitest';
import { hasValidCoord } from './coord';

describe('hasValidCoord（GIS 搜索结果坐标有效性 · issue #192）', () => {
  // 成功路径：真实经纬度可定位
  it('真实经纬度返回 true', () => {
    expect(hasValidCoord(39.8615, 116.3654)).toBe(true);
    expect(hasValidCoord(25.92, 115.36)).toBe(true);
    // 负坐标（南半球 / 西半球）同样有效
    expect(hasValidCoord(-12.97, 28.65)).toBe(true);
  });

  // 失败路径：(0,0) 缺省坐标——落点外海空白网格，视为无法定位
  it('(0,0) 缺省坐标返回 false', () => {
    expect(hasValidCoord(0, 0)).toBe(false);
  });

  // 失败路径：null / undefined 未回填坐标
  it('null / undefined 坐标返回 false', () => {
    expect(hasValidCoord(null, null)).toBe(false);
    expect(hasValidCoord(undefined, undefined)).toBe(false);
    expect(hasValidCoord(39.8615, null)).toBe(false);
    expect(hasValidCoord(null, 116.3654)).toBe(false);
  });

  // 边界：仅一个轴为 0 仍可定位（如赤道或本初子午线上的真实站点）
  it('单轴为 0、另一轴非 0 视为有效', () => {
    expect(hasValidCoord(0, 116.3654)).toBe(true);
    expect(hasValidCoord(39.8615, 0)).toBe(true);
  });
});
