import { describe, it, expect } from 'vitest';
import { baseProductClass, collapseImageProductClasses } from '../productClass';

// qa-614 #369：IMAGE 升级镜像"版本合一"——同族载波变体 /SC /DC /CA 收敛为共同基础标识。

describe('baseProductClass', () => {
  it('剥去 /SC /DC /CA 载波变体后缀', () => {
    expect(baseProductClass('FAP/MLN/SC')).toBe('FAP/MLN');
    expect(baseProductClass('FAP/MLN/DC')).toBe('FAP/MLN');
    expect(baseProductClass('FAP/MLN/CA')).toBe('FAP/MLN');
  });
  it('无载波变体后缀的原样返回', () => {
    expect(baseProductClass('FAP/MLN')).toBe('FAP/MLN');
    expect(baseProductClass('PM-B4860')).toBe('PM-B4860');
  });
  it('精确锚定，不误伤非后缀的 CA/SC/DC 片段', () => {
    expect(baseProductClass('FAP/MLN/CA-cert')).toBe('FAP/MLN/CA-cert');
    expect(baseProductClass('SCALER/X')).toBe('SCALER/X');
  });
});

describe('collapseImageProductClasses', () => {
  it('同族载波变体收敛为单一共同标识、保持顺序去重', () => {
    expect(
      collapseImageProductClasses(['FAP/MLN/SC', 'FAP/MLN/DC', 'FAP/MLN/CA'])
    ).toEqual(['FAP/MLN']);
  });
  it('混合多产品族各自收敛、保留首次出现顺序', () => {
    expect(
      collapseImageProductClasses([
        'FAP/MLN/SC',
        'PM-B4860',
        'FAP/MLN/CA',
        'PM-B4860',
      ])
    ).toEqual(['FAP/MLN', 'PM-B4860']);
  });
  it('过滤空值', () => {
    expect(collapseImageProductClasses(['', 'FAP/MLN/SC', ''])).toEqual([
      'FAP/MLN',
    ]);
  });
});
