import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 报表域冒烟：登录后三个报表页面正常渲染（真实后端，不断言具体业务数据）。
 *
 * 选择器依据：
 *   - src/pages/report/LTEStandardReport/index.tsx：TreeListPageLayout（左树
 *     nav.report.lteStandard 标题 + report.category.* 分类树）+ DataTable
 *   - src/pages/report/StationReport/index.tsx：ListPageLayout（标题
 *     nav.report.station，title 与 subtitle 同文案需 .first()）+ FilterBar + DataTable
 *   - src/pages/report/PollStatistics/index.tsx：ListPageLayout（标题
 *     nav.report.pollStats）+ 刷新按钮 + FilterBar + DataTable
 * 文案来自 frontend-core/src/i18n/{zh-CN,en-US}/index.ts，正则兼容双语。
 * 断言统一收窄到 <main> 内，避免命中侧边栏同名导航项。
 */
test.describe('报表冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/report/lte-standard 渲染，分类树与报表列表骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/report/lte-standard');
    const main = page.locator('main');

    // 左侧树面板标题（nav.report.lteStandard：LTE标准报表 / LTE Standard Report）
    await expect(main.getByText(/LTE标准报表|LTE Standard Report/).first()).toBeVisible();

    // 报表分类树（report.category.kpi：KPI报表 / KPI Reports）
    await expect(main.locator('.ant-tree')).toBeVisible();
    await expect(main.getByText(/KPI报表|KPI Reports/).first()).toBeVisible();

    // 右侧报表记录表格骨架（空表也有表头）
    await expect(main.locator('.ant-table').first()).toBeVisible();
  });

  test('/report/station 渲染，标题、批量导出按钮与表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/report/station');
    const main = page.locator('main');

    // 页面标题（nav.report.station：站点报表 / Station Report；title+subtitle 同文案取 first）
    await expect(main.getByText(/站点报表|Station Report/).first()).toBeVisible();

    // 顶部批量导出按钮（common.batchExport：批量导出 / Batch Export）
    await expect(
      main.getByRole('button', { name: /批量导出|Batch Export/ }).first(),
    ).toBeVisible();

    // FilterBar 表单 + KPI 报表表格骨架
    await expect(main.locator('.ant-form').first()).toBeVisible();
    await expect(main.locator('.ant-table').first()).toBeVisible();
  });

  test('/report/poll-stats 渲染，标题、刷新按钮与表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/report/poll-stats');
    const main = page.locator('main');

    // 页面标题（nav.report.pollStats：轮询统计 / Poll Statistics）
    await expect(main.getByText(/轮询统计|Poll Statistics/).first()).toBeVisible();

    // 顶部刷新按钮（common.refresh：刷新 / Refresh）
    await expect(main.getByRole('button', { name: /刷新|Refresh/ }).first()).toBeVisible();

    // FilterBar 表单 + 轮询任务表格骨架
    await expect(main.locator('.ant-form').first()).toBeVisible();
    await expect(main.locator('.ant-table').first()).toBeVisible();
  });
});
