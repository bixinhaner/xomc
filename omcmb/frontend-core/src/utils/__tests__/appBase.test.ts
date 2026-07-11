import { describe, it, expect } from 'vitest';
import { basePathOf, loginUrlFor } from '../appBase';

// V1 默认部署在 '/'，也支持自定义子路径；硬跳转登录页必须带 base 前缀。

describe('basePathOf', () => {
  it('v1 根 base "/" → 空前缀', () => {
    expect(basePathOf('/')).toBe('');
  });
  it('自定义 base "/omc/" → "/omc"（去尾斜杠）', () => {
    expect(basePathOf('/omc/')).toBe('/omc');
  });
  it('base 缺失（undefined/空串）兜底回退到根', () => {
    expect(basePathOf(undefined)).toBe('');
    expect(basePathOf('')).toBe('');
  });
});

describe('loginUrlFor', () => {
  it('v1 → /login', () => {
    expect(loginUrlFor('/')).toBe('/login');
  });
  it('自定义 base → 对应子路径登录页', () => {
    expect(loginUrlFor('/omc/')).toBe('/omc/login');
  });
  it('base 缺失兜底 /login（不报错）', () => {
    expect(loginUrlFor(undefined)).toBe('/login');
  });
});
