import { test, expect } from '@playwright/test';
import { smokeLogin } from './helpers';

/**
 * License 菜单可见性冒烟（真实后端）。
 *
 * 验收 bug 修复：source=admin 的登录用户走 GetByRole，role_menus 未分配 /license，
 * 导致「有 license 时 /license 菜单反而不见」（无 license 时 filterLicenseOnlyMenus
 * 强制可见，行为不一致，用户无法管理 license）。后端在 applyLicenseMenuGate 的
 * 有 license 分支补 ensureLicenseMenuVisible 后，菜单应始终可见。
 *
 * 前置：后端 system_license 存在 current 行（本栈默认已上传测试 license）。
 *
 * 选择器依据：
 *   - src/components/Layout/Sidebar/NavMenu.tsx（antd Menu，菜单项 .ant-menu-item）
 *   - src/pages/SystemLicense/index.tsx（有 license 时渲染 Title + Descriptions +
 *     DevicesSupportCards；空态才是 .ant-result）
 *   - i18n systemLicense.title（zh/en 均为 "License"）
 */
test.describe('License 菜单可见性冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('有 license 时侧边栏始终含 License 菜单项', async ({ page }) => {
    // 登录后落在 /dashboard，侧边栏已渲染
    await page.waitForURL(/\/dashboard/);

    // 侧边栏 role=complementary（aria-label 导航菜单）；License 菜单项 accessible name
    // 含 "License"（icon safety 的 alt + 菜单文案，systemLicense.title zh/en 均为 License）。
    const sidebar = page.getByRole('complementary');
    await expect(
      sidebar.getByRole('menuitem', { name: /License/ }),
    ).toBeVisible();
  });

  test('访问 /license 渲染 license 详情（有 license 时非空态）', async ({ page }) => {
    // 菜单可见性已由上一条用例覆盖；这里直接验证页面渲染：
    // 有 license 时显示 Title + Update 工具栏，不出现空态 Result（系统未配置许可证）。
    await page.goto('/license');
    await page.waitForURL(/\/license/);

    await expect(page.locator('main')).toBeVisible();
    await expect(page.locator('.ant-result')).not.toBeVisible();

    // 工具栏 Update 按钮（systemLicense.update，zh/en 均为 "Update"）。
    // antd Button + icon 的 accessible name = "图标名 文本"，故用子串匹配。
    await expect(
      page.getByRole('button', { name: /Update/ }).first(),
    ).toBeVisible();

    // Basic Info 表格：License ID 行（systemLicense 表 current 行的字段）。
    await expect(page.locator('main').getByText(/NO2022-03-14002|License ID/).first()).toBeVisible();
  });
});
