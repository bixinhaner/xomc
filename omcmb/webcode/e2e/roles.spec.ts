import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';
import { navigateTo, waitForPageLoad } from './helpers/navigation';

// PRD docs/prd/system/roles.md v0.6 — 角色页面关键 UI：列表 + 创建抽屉中三个权限 Tab。
// Mock 模式下断言面板结构，不依赖真实后端响应内容。

test.describe('System / Roles management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load the roles list page with table', async ({ page }) => {
    await navigateTo(page, '/system/roles');
    await waitForPageLoad(page);

    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 15_000 });
  });

  test('should display rows from mock data', async ({ page }) => {
    await navigateTo(page, '/system/roles');
    await waitForPageLoad(page);

    const tableBody = page.locator('.ant-table-tbody');
    await expect(tableBody).toBeVisible({ timeout: 15_000 });

    const rows = tableBody.locator('tr.ant-table-row');
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
  });

  test('should open create drawer with role name + permission tabs (menu / api / resource)', async ({
    page,
  }) => {
    await navigateTo(page, '/system/roles');
    await waitForPageLoad(page);

    // Primary add button (page header extra)
    await page.locator('button.ant-btn-primary').first().click();

    const drawer = page.locator('.ant-drawer').first();
    await expect(drawer).toBeVisible({ timeout: 5_000 });

    // Role name input should be present
    await expect(drawer.locator('input').first()).toBeVisible();

    // PRD §6 三个权限 Tab：菜单权限 / API 权限 / 资源权限（运营商语言上的体现）
    const tabs = drawer.locator('.ant-tabs-tab');
    await expect(tabs).toHaveCount(3, { timeout: 5_000 });
  });
});
