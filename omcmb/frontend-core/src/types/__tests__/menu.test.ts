import { describe, it, expect } from 'vitest';
import { resolveMenuLabel, mapBackendMenu, type Menu, type BackendMenu } from '../menu';

// resolveMenuLabel 的渲染优先级（T-0113 菜单多语言改造方案 §C）：
//   1. i18nKey 命中 messages → t(i18nKey)
//   2. nameI18n[locale]      → 当前语言译文
//   3. nameI18n['zh-CN']     → 中文兜底
//   4. name                  → 终极 fallback
describe('resolveMenuLabel', () => {
  const baseMenu: Pick<Menu, 'name'> = { name: '原始名' };

  it('i18nKey 命中 messages 时优先返回翻译值', () => {
    const menu = {
      ...baseMenu,
      i18nKey: 'nav.ops',
      nameI18n: { 'zh-CN': '运维管理', 'en-US': 'Operations' },
    };
    const messages = { 'nav.ops': '从 i18n 包来的运维' };
    expect(resolveMenuLabel(menu, 'zh-CN', messages)).toBe('从 i18n 包来的运维');
    expect(resolveMenuLabel(menu, 'en-US', messages)).toBe('从 i18n 包来的运维');
  });

  it('i18nKey 未命中 messages 时跳过，走 nameI18n', () => {
    const menu = {
      ...baseMenu,
      i18nKey: 'nav.never.exists',
      nameI18n: { 'zh-CN': '运维', 'en-US': 'Operations' },
    };
    expect(resolveMenuLabel(menu, 'en-US', { 'other.key': 'x' })).toBe('Operations');
  });

  it('未传 messages 时 i18nKey 路径自动跳过', () => {
    const menu = {
      ...baseMenu,
      i18nKey: 'nav.ops',
      nameI18n: { 'zh-CN': '运维', 'en-US': 'Operations' },
    };
    expect(resolveMenuLabel(menu, 'en-US')).toBe('Operations');
  });

  it('nameI18n 命中当前 locale 时返回译文', () => {
    const menu = { ...baseMenu, nameI18n: { 'zh-CN': '运维', 'en-US': 'Operations' } };
    expect(resolveMenuLabel(menu, 'zh-CN')).toBe('运维');
    expect(resolveMenuLabel(menu, 'en-US')).toBe('Operations');
  });

  it('当前 locale 缺译文时回退中文', () => {
    const menu = { ...baseMenu, nameI18n: { 'zh-CN': '运维' } };
    expect(resolveMenuLabel(menu, 'en-US')).toBe('运维');
  });

  it('nameI18n 完全缺失时回退 name 字段', () => {
    expect(resolveMenuLabel({ name: '原始名' }, 'en-US')).toBe('原始名');
  });

  it('messages key 非 string 时（react-intl 编译后形态）不命中 i18nKey', () => {
    const menu = {
      ...baseMenu,
      i18nKey: 'nav.ops',
      nameI18n: { 'zh-CN': '运维', 'en-US': 'Operations' },
    };
    // react-intl 编译后 messages 值可能是 MessageFormatElement[]，应当被 typeof === 'string' 过滤
    const messages = { 'nav.ops': [{ type: 0, value: 'X' }] as unknown as string };
    expect(resolveMenuLabel(menu, 'en-US', messages)).toBe('Operations');
  });
});

describe('mapBackendMenu i18n 透传', () => {
  it('后端 name_i18n / i18n_key 字段正确映射到前端 camelCase', () => {
    const b: BackendMenu = {
      id: 'm1',
      name: '运维',
      name_i18n: { 'zh-CN': '运维', 'en-US': 'Operations' },
      i18n_key: 'nav.ops',
      type: 'directory',
      permission_key: 'ops',
      sort_order: 1,
      show_status: 'show',
      status: 'normal',
    };
    const m = mapBackendMenu(b);
    expect(m.nameI18n).toEqual({ 'zh-CN': '运维', 'en-US': 'Operations' });
    expect(m.i18nKey).toBe('nav.ops');
  });

  it('后端 null / 空 i18n 字段映射为 undefined', () => {
    const b: BackendMenu = {
      id: 'm2',
      name: '原始',
      name_i18n: null,
      i18n_key: null,
      type: 'menu',
      permission_key: 'x',
      sort_order: 1,
      show_status: 'show',
      status: 'normal',
    };
    const m = mapBackendMenu(b);
    expect(m.nameI18n).toBeUndefined();
    expect(m.i18nKey).toBeUndefined();
  });

  it('递归子菜单也透传 i18n 字段', () => {
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
          i18n_key: 'nav.child',
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
    expect(m.children?.[0].i18nKey).toBe('nav.child');
  });
});
