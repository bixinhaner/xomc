/**
 * 分类 Tab 自动选中决策（#127）。
 *
 * /transfer/center 的 categories = 后端 ufte_task_types 真实分类 + 前端追加的
 * 虚拟入口分类（MR 测量 / KPI 导出）。首屏 task-types 未返回时 categories 只含
 * 虚拟项，自动选中会落在 'mr_measurement'；真实分类到达后若用户没手动点过 Tab，
 * 必须回退到首个真实分类，否则默认 Tab 停在虚拟分类、执行视图卡片被隐藏。
 */

/** 前端虚拟分类（入口聚合 Tab，非后端 ufte_task_types 真实分类） */
export const VIRTUAL_CATEGORIES = new Set(['mr_measurement', 'kpi_export']);

interface CategoryLike {
  category: string;
}

/**
 * 计算分类 Tab 应自动切换到的值。
 *
 * 返回 null 表示保持现状；返回非 null 表示需要 setSelectedCategory(返回值)，
 * 且该选择是"自动兜底"（调用方应清除用户手选标记）。
 *
 * @param categories       当前分类列表（真实分类在前，虚拟分类固定在末位）
 * @param selectedCategory 当前选中的分类
 * @param userPicked       用户是否主动选过分类（点击 Tab / URL deep link）
 */
export function resolveAutoSelectedCategory(
  categories: readonly CategoryLike[],
  selectedCategory: string,
  userPicked: boolean,
): string | null {
  if (categories.length === 0) {
    return selectedCategory === '' ? null : '';
  }
  if (!categories.some((item) => item.category === selectedCategory)) {
    return categories[0].category;
  }
  // #127 竞态修复：当前选中是首屏自动落上的虚拟分类且用户没手动点过 Tab 时，
  // 真实分类一旦出现就回退到首个真实分类。
  if (!userPicked && VIRTUAL_CATEGORIES.has(selectedCategory)) {
    const firstReal = categories.find((item) => !VIRTUAL_CATEGORIES.has(item.category));
    if (firstReal) {
      return firstReal.category;
    }
  }
  return null;
}
