import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 配置管理域冒烟（真实后端）：四个页面渲染不崩 + 关键骨架元素可见。
 *
 * 选择器依据：
 *   - src/pages/config/{ParamSync,BatchParamTemplate,BaselineManagement,ParamList}/index.tsx
 *   - i18n：nav.config.paramSync（参数同步/Param Sync）、nav.config.batchTemplate
 *     （批量参数模板/Batch Param Template）、nav.config.paramList（参数列表/Param List）、
 *     config.baseline（基线/Baseline）—— 默认 zh-CN，正则兼容 en-US。
 *   - 结构断言只用 antd 稳定 class（.ant-table/.ant-tree/.ant-form）；
 *     不断言具体业务数据（真实栈数据稀疏，空表也有表头）。
 *   - 侧边导航菜单含同名文案，页面内文案断言一律收窄到 <main> 并取 .first()。
 */
test.describe('配置管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/config/param-sync 参数同步：树面板 + 参数表格渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/param-sync');

    // 页面头部标题（TreeListPageLayout 主区 Typography.Text）
    await expect(
      page.locator('main').getByText(/参数同步|Param Sync/).first(),
    ).toBeVisible();
    // 左侧网元树
    await expect(page.locator('main .ant-tree').first()).toBeVisible();
    // 参数表格骨架（空表也有表头）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/config/batch-template 批量参数模板：标题 + 模板表格渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/batch-template');

    // ListPageLayout 标题（nav.config.batchTemplate）
    await expect(
      page.locator('main').getByText(/批量参数模板|Batch Param Template/).first(),
    ).toBeVisible();
    // 模板列表表格骨架
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/config/baseline 基线管理：基线树 + 参数对比表渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/baseline');

    // 左侧树面板标题（config.baseline：基线/Baseline）
    await expect(
      page.locator('main').getByText(/基线|Baseline/).first(),
    ).toBeVisible();
    // 基线树（baselines 加载完成后渲染；空数据时回落 mock 也有树）
    await expect(page.locator('main .ant-tree').first()).toBeVisible();
    // 参数对比表骨架
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/config/param-list 参数列表：标题 + 筛选栏 + 参数字典表渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/param-list');

    // ListPageLayout 标题（nav.config.paramList）
    await expect(
      page.locator('main').getByText(/参数列表|Param List/).first(),
    ).toBeVisible();
    // FilterBar（antd Form）
    await expect(page.locator('main .ant-form').first()).toBeVisible();
    // 参数字典表格骨架
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });
});
