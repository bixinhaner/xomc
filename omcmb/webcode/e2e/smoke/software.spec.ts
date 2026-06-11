import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 软件管理域冒烟（真实后端）：菜单已下线（routes.tsx §「Software Management」），
 * 但 /software/* 路由保留可经直接 URL 访问。只验证页面渲染不崩 + 关键骨架元素。
 *
 * 因菜单已移除，页面标题不会与侧边栏菜单项撞名，可直接断言标题文本。
 *
 * 选择器依据（标题取自 ListPageLayout title，i18n 现查 frontend-core/src/i18n）：
 *   - /software/version       → VersionQuery/index.tsx（ListPageLayout title=nav.software.version
 *                               版本查询/Version Query；FilterBar + DataTable）
 *   - /software/upgrade-plan  → UpgradePlan/index.tsx（title=nav.software.versionUpgrade
 *                               版本升级/Version Upgrade；headerExtra 工具栏 + FilterBar + DataTable）
 *   - /software/activation    → ActivationPlan/index.tsx（title=nav.software.activation
 *                               激活计划/Activation Plan；extra 新增 + FilterBar + DataTable）
 *   - /software/firmware      → FirmwareUpload/index.tsx（title=software.firmware.title
 *                               升级文件管理/Upgrade File Management；DataTable + Upload Dragger）
 *   - /software/rollback      → VersionRollback/index.tsx（title=nav.software.versionRollback
 *                               版本回退/Version Rollback；headerExtra 工具栏 + FilterBar + DataTable）
 *
 * 真实栈数据稀疏：不断言任何具体业务数据，列表空表也有表头（.ant-table 可见即可）。
 */
test.describe('软件管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/software/version 版本查询：标题 + 筛选栏 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/software/version');

    const main = page.locator('main');
    await expect(main.getByText(/版本查询|Version Query/).first()).toBeVisible();

    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/software/upgrade-plan 版本升级：标题 + 筛选栏 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/software/upgrade-plan');

    const main = page.locator('main');
    await expect(main.getByText(/版本升级|Version Upgrade/).first()).toBeVisible();

    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/software/activation 激活计划：标题 + 新增按钮 + 筛选栏 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/software/activation');

    const main = page.locator('main');
    await expect(main.getByText(/激活计划|Activation Plan/).first()).toBeVisible();

    // extra 新增按钮
    await expect(main.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();

    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/software/firmware 升级文件管理：标题 + 文件类型切换 + 导入按钮 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/software/firmware');

    const main = page.locator('main');
    await expect(main.getByText(/升级文件管理|Upgrade File Management/).first()).toBeVisible();

    // 文件类型 Radio.Group（IMAGE / PATCH，固定文案）
    await expect(main.getByText('IMAGE').first()).toBeVisible();

    // 导入文件按钮（点击才展开 Dragger 抽屉，故只断言入口按钮存在）
    await expect(main.getByRole('button', { name: /导入文件|Import File/ }).first()).toBeVisible();

    // 固件版本表格骨架（真实后端数据可空，空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/software/rollback 版本回退：标题 + 筛选栏 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/software/rollback');

    const main = page.locator('main');
    await expect(main.getByText(/版本回退|Version Rollback/).first()).toBeVisible();

    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });
});
