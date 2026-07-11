import { describe, it, expect } from 'vitest';
import { resolveOmcName } from './useOmcName';

// resolveOmcName 是品牌标题回退逻辑的核心：配置值优先、空/未配置回退默认名。
// 顶部与登录页依赖它保证“空值回退默认名而非空白”。
describe('resolveOmcName', () => {
  const fallback = 'OMC 统一网管系统';

  it('配置非空时返回配置值（去首尾空白）', () => {
    expect(resolveOmcName('我的网管', fallback)).toBe('我的网管');
    expect(resolveOmcName('  我的网管  ', fallback)).toBe('我的网管');
  });

  it('配置为 undefined 时回退默认名', () => {
    expect(resolveOmcName(undefined, fallback)).toBe(fallback);
  });

  it('配置为空串 / 纯空白时回退默认名（不渲染空白标题）', () => {
    expect(resolveOmcName('', fallback)).toBe(fallback);
    expect(resolveOmcName('   ', fallback)).toBe(fallback);
  });

  it('各皮肤可传各自默认名', () => {
    expect(resolveOmcName(undefined, 'OMC · v2')).toBe('OMC · v2');
    expect(resolveOmcName(undefined, 'STARFORGE')).toBe('STARFORGE');
    expect(resolveOmcName('品牌X', 'STARFORGE')).toBe('品牌X');
  });
});
