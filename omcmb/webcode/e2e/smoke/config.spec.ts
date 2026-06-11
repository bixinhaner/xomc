import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 配置管理域冒烟（真实后端）：九个页面渲染不崩 + 关键骨架元素可见。
 *
 * 选择器依据：
 *   - src/pages/config/{ParamSync,BatchParamTemplate,BaselineManagement,ParamList,
 *     LiveParamConfig,NeighborParams,AutoProvisioning,InteropTesting,CellManagement}/index.tsx
 *   - i18n：nav.config.paramSync（参数同步/Param Sync）、nav.config.batchTemplate
 *     （批量参数模板/Batch Param Template）、nav.config.paramList（参数列表/Param List）、
 *     config.baseline（基线/Baseline）、nav.config.liveParam（实时参数配置/Live Param Config）、
 *     nav.config.neighbor（邻区参数/Neighbor Params）、nav.config.cell（小区管理/Cell Management）
 *     —— 默认 zh-CN，正则兼容 en-US；AutoProvisioning/InteropTesting 标题为硬编码英文字面量。
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

  test('/config/live-param 实时参数：标题 + 筛选栏 + 参数表格渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/live-param');

    // ListPageLayout 标题（nav.config.liveParam）；与侧栏同名菜单收窄到 main 取 .first()
    await expect(
      page.locator('main').getByText(/实时参数配置|Live Param Config/).first(),
    ).toBeVisible();
    // FilterBar（antd Form）
    await expect(page.locator('main .ant-form').first()).toBeVisible();
    // 实时参数表格骨架（空表也有表头）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/config/neighbor 邻区参数：标题 + 筛选栏 + 邻区表格渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/neighbor');

    // ListPageLayout 标题（nav.config.neighbor）；侧栏同名菜单收窄 main 取 .first()
    await expect(
      page.locator('main').getByText(/邻区参数|Neighbor Params/).first(),
    ).toBeVisible();
    // FilterBar（antd Form）
    await expect(page.locator('main .ant-form').first()).toBeVisible();
    // 邻区关系表格骨架
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/config/auto-provision 自动开站：标题 + 任务表格渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/auto-provision');

    // ListPageLayout 标题（硬编码英文字面量 "Auto-Provisioning"，无对应 i18n key）
    await expect(
      page.locator('main').getByText('Auto-Provisioning').first(),
    ).toBeVisible();
    // 开站任务表格骨架（antd Table，空数据也有表头）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/config/interop-test 互操作测试：标题 + 用例库卡片表格渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/interop-test');

    // ListPageLayout 标题（硬编码英文字面量 "Interop Testing"）
    await expect(
      page.locator('main').getByText('Interop Testing').first(),
    ).toBeVisible();
    // "Run Conformance Tests" 卡片内的 antd Form（inline 表单）
    await expect(page.locator('main .ant-form').first()).toBeVisible();
    // 用例库表格骨架（Test Case Library，空数据也有表头）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/config/cell 小区管理：标题 + 筛选栏 + 小区表格渲染', async ({ page }) => {
    await expectPageRenders(page, '/config/cell');

    // ListPageLayout 标题（nav.config.cell）；侧栏同名菜单收窄 main 取 .first()
    await expect(
      page.locator('main').getByText(/小区管理|Cell Management/).first(),
    ).toBeVisible();
    // FilterBar（antd Form）
    await expect(page.locator('main .ant-form').first()).toBeVisible();
    // 小区列表表格骨架
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });
});
