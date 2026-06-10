import { describe, it, expect } from 'vitest';
import { resolveAutoSelectedCategory, VIRTUAL_CATEGORIES } from './categorySelection';

// 分类列表小工厂：真实分类在前、虚拟分类固定末位（与 index.tsx categories useMemo 一致）
const cat = (category: string) => ({ category });
const VIRTUAL_ONLY = [cat('mr_measurement'), cat('kpi_export')];
const FULL = [cat('gnb_upgrade'), cat('config_backup'), cat('mr_measurement'), cat('kpi_export')];

describe('resolveAutoSelectedCategory — #127 默认 Tab 竞态', () => {
  const cases: Array<{
    name: string;
    categories: Array<{ category: string }>;
    selected: string;
    userPicked: boolean;
    want: string | null;
  }> = [
    {
      name: '首屏 task-types 未返回：空选中 → 自动落到首个分类（虚拟）',
      categories: VIRTUAL_ONLY,
      selected: '',
      userPicked: false,
      want: 'mr_measurement',
    },
    {
      name: '只有虚拟分类且已选虚拟：无真实分类可回退 → 保持现状',
      categories: VIRTUAL_ONLY,
      selected: 'mr_measurement',
      userPicked: false,
      want: null,
    },
    {
      name: '核心竞态：真实分类到达 + 选中仍是自动落上的虚拟分类 → 回退首个真实分类',
      categories: FULL,
      selected: 'mr_measurement',
      userPicked: false,
      want: 'gnb_upgrade',
    },
    {
      name: 'kpi_export 同为虚拟分类，自动选中时同样回退',
      categories: FULL,
      selected: 'kpi_export',
      userPicked: false,
      want: 'gnb_upgrade',
    },
    {
      name: '用户手动点选了虚拟分类 → 不覆盖用户选择',
      categories: FULL,
      selected: 'mr_measurement',
      userPicked: true,
      want: null,
    },
    {
      name: '选中真实分类（无论是否手选）→ 保持现状',
      categories: FULL,
      selected: 'config_backup',
      userPicked: false,
      want: null,
    },
    {
      name: '选中分类已不在列表（deep link 失效 / 分类被移除）→ 兜底首个分类',
      categories: FULL,
      selected: 'station_log',
      userPicked: true,
      want: 'gnb_upgrade',
    },
    {
      name: '分类列表为空且已选空 → 保持现状',
      categories: [],
      selected: '',
      userPicked: false,
      want: null,
    },
    {
      name: '分类列表为空但有残留选中 → 清空',
      categories: [],
      selected: 'gnb_upgrade',
      userPicked: false,
      want: '',
    },
  ];

  it.each(cases)('$name', ({ categories, selected, userPicked, want }) => {
    expect(resolveAutoSelectedCategory(categories, selected, userPicked)).toBe(want);
  });

  it('两步竞态串联：自动选虚拟 → 数据到达 → 回退后稳定在真实分类', () => {
    // 第 1 步：首屏只有虚拟分类，'' → mr_measurement
    const step1 = resolveAutoSelectedCategory(VIRTUAL_ONLY, '', false);
    expect(step1).toBe('mr_measurement');
    // 第 2 步：真实分类到达，mr_measurement（非手选）→ gnb_upgrade
    const step2 = resolveAutoSelectedCategory(FULL, step1!, false);
    expect(step2).toBe('gnb_upgrade');
    // 第 3 步：回退后收敛，不再变更（防 effect 抖动/死循环）
    expect(resolveAutoSelectedCategory(FULL, step2!, false)).toBeNull();
    expect(VIRTUAL_CATEGORIES.has(step2!)).toBe(false);
  });
});
