import { describe, it, expect } from 'vitest';
import { chapterSortKey, stripOpSuffix } from '../commandTreeUtils';

// CommandTree.tsx 的两个纯字符串 helper：
//   - chapterSortKey：与后端 group_tree_repository.go::chapterSortKey 对偶
//     （SA<SB<...<SR<空字符串排末位）
//   - stripOpSuffix：去掉 backend displayName 末尾 `(OP CODE)` 部分，让 OP Tag
//     + object name 视觉不重复
//
// CommandTree 主组件的渲染交互由 Playwright E2E 覆盖（依赖 antd Tree 真实 DOM）；
// 此处只测纯逻辑，保证 chapter sort + OP 前缀的正确性。

describe('chapterSortKey', () => {
  it('non-empty chapter returns 原值（保留字典序天然 SA < SB < SR）', () => {
    expect(chapterSortKey('SA')).toBe('SA');
    expect(chapterSortKey('SB')).toBe('SB');
    expect(chapterSortKey('SR')).toBe('SR');
  });

  it('empty / undefined → sentinel "~~~~~" 排末位', () => {
    expect(chapterSortKey('')).toBe('~~~~~');
    expect(chapterSortKey(undefined)).toBe('~~~~~');
  });

  it('字典序天然按 SA → SB → ... → SR 顺序（Unicode codepoint 比较）', () => {
    // localeCompare 在不同 locale 下对 punctuation 的处理不同；CommandTree.tsx
    // 实际用 `<` 操作符做字符串比较（Unicode codepoint），保证 sentinel "~~~~~"
    // (U+007E) > 大写字母 (U+0041~U+005A) 排末位。
    const chapters = ['SC', 'SA', 'SR', undefined, 'SB', ''];
    const sorted = [...chapters].sort((a, b) => {
      const ac = chapterSortKey(a);
      const bc = chapterSortKey(b);
      if (ac < bc) return -1;
      if (ac > bc) return 1;
      return 0;
    });
    // SA/SB/SC/SR 在前（按字典序），undefined 和 '' 都映射为 sentinel 排末位（顺序稳定）
    expect(sorted.slice(0, 4)).toEqual(['SA', 'SB', 'SC', 'SR']);
    expect(sorted.slice(4).sort()).toEqual(['', undefined]); // 末位 2 项是 ''/undefined 任意序
  });
});

describe('stripOpSuffix', () => {
  it('去掉历史格式 "设备信息(LST DEVICE_INFO)" 末尾括号', () => {
    expect(stripOpSuffix('设备信息(LST DEVICE_INFO)')).toBe('设备信息');
    expect(stripOpSuffix('设备版本升级(MOD SW_UPGRADE)')).toBe('设备版本升级');
  });

  it('无括号时原样返回', () => {
    expect(stripOpSuffix('设备信息')).toBe('设备信息');
    expect(stripOpSuffix('DeviceInfo')).toBe('DeviceInfo');
  });

  it('多个括号 — 只剥最后一段', () => {
    expect(stripOpSuffix('Foo(bar)(LST X)')).toBe('Foo(bar)');
  });

  it('右括号不闭合的异常字符串 — 原样返回（不破坏）', () => {
    expect(stripOpSuffix('Foo(LST')).toBe('Foo(LST');
  });

  it('空字符串', () => {
    expect(stripOpSuffix('')).toBe('');
  });
});
