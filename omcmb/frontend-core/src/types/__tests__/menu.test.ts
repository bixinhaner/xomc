import { describe, it, expect } from 'vitest';
import { resolveMenuLabel, mapBackendMenu, type BackendMenu } from '../menu';

// resolveMenuLabel 的 3 级 fallback（方案 C — DB JSONB 主路径）：
//   1. nameI18n[locale]   → 当前语言译文
//   2. nameI18n['zh-CN']  → 中文兜底
//   3. name               → 终极 fallback
describe('resolveMenuLabel', () => {
  it('nameI18n 命中当前 locale 时返回译文', () => {
    const menu = { name: '原始名', nameI18n: { 'zh-CN': '运维', 'en-US': 'Operations' } };
    expect(resolveMenuLabel(menu, 'zh-CN')).toBe('运维');
    expect(resolveMenuLabel(menu, 'en-US')).toBe('Operations');
  });

  it('当前 locale 缺译文时回退中文', () => {
    const menu = { name: '原始名', nameI18n: { 'zh-CN': '运维' } };
    expect(resolveMenuLabel(menu, 'en-US')).toBe('运维');
  });

  it('nameI18n 完全缺失时回退 name 字段', () => {
    expect(resolveMenuLabel({ name: '原始名' }, 'en-US')).toBe('原始名');
  });

  it('nameI18n 为空对象时回退 name', () => {
    expect(resolveMenuLabel({ name: '原始名', nameI18n: {} }, 'en-US')).toBe('原始名');
  });

  it('当前 locale 译文是空字符串时回退中文（避免渲染空 label）', () => {
    const menu = { name: '原始名', nameI18n: { 'zh-CN': '运维', 'en-US': '' } };
    expect(resolveMenuLabel(menu, 'en-US')).toBe('运维');
  });
});

describe('mapBackendMenu i18n 透传', () => {
  it('后端 name_i18n 字段正确映射到前端 camelCase', () => {
    const b: BackendMenu = {
      id: 'm1',
      name: '运维',
      name_i18n: { 'zh-CN': '运维', 'en-US': 'Operations' },
      type: 'directory',
      permission_key: 'ops',
      sort_order: 1,
      show_status: 'show',
      status: 'normal',
    };
    const m = mapBackendMenu(b);
    expect(m.nameI18n).toEqual({ 'zh-CN': '运维', 'en-US': 'Operations' });
  });

  it('后端 null name_i18n 映射为 undefined', () => {
    const b: BackendMenu = {
      id: 'm2',
      name: '原始',
      name_i18n: null,
      type: 'menu',
      permission_key: 'x',
      sort_order: 1,
      show_status: 'show',
      status: 'normal',
    };
    const m = mapBackendMenu(b);
    expect(m.nameI18n).toBeUndefined();
  });

  it('递归子菜单也透传 nameI18n 字段', () => {
    const b: BackendMenu = {
      id: 'p',
      name: '父',
      name_i18n: { 'en-US': 'Parent' },
      type: 'directory',
      permission_key: 'p',
      sort_order: 1,
      show_status: 'show',
      status: 'normal',
      children: [
        {
          id: 'c',
          name: '子',
          name_i18n: { 'en-US': 'Child' },
          type: 'menu',
          permission_key: 'c',
          sort_order: 1,
          show_status: 'show',
          status: 'normal',
        },
      ],
    };
    const m = mapBackendMenu(b);
    expect(m.children?.[0].nameI18n).toEqual({ 'en-US': 'Child' });
  });
});
