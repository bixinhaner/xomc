import { describe, it, expect } from 'vitest';
import { basePathOf, loginUrlFor } from '../appBase';

// 三皮肤 base 前缀：v1 部署在 '/'、v2 在 '/v2/'、v3 在 '/v3/'（vite base 固化进
// import.meta.env.BASE_URL）。硬跳转登录页必须带 base 前缀，否则 v2/v3 会被甩到
// v1 的 /login。这里测纯函数（读环境版 appBasePath/loginUrl 只是套一层 BASE_URL）。

describe('basePathOf', () => {
  it('v1 根 base "/" → 空前缀', () => {
    expect(basePathOf('/')).toBe('');
  });
  it('v2 base "/v2/" → "/v2"（去尾斜杠）', () => {
    expect(basePathOf('/v2/')).toBe('/v2');
  });
  it('v3 base "/v3/" → "/v3"', () => {
    expect(basePathOf('/v3/')).toBe('/v3');
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
  it('v2 → /v2/login（不丢皮肤上下文）', () => {
    expect(loginUrlFor('/v2/')).toBe('/v2/login');
  });
  it('v3 → /v3/login', () => {
    expect(loginUrlFor('/v3/')).toBe('/v3/login');
  });
  it('base 缺失兜底 /login（不报错）', () => {
    expect(loginUrlFor(undefined)).toBe('/login');
  });
});
