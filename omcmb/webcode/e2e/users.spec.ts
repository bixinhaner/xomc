import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';
import { navigateTo, waitForPageLoad } from './helpers/navigation';

// PRD docs/prd/system/users.md v1.1 — 前端关键场景：列表、筛选、添加 / 编辑面板字段。
// Mock 模式（npm run dev:mock）下不依赖真实后端，断言聚焦 UI 结构与字段存在性。

test.describe('System / Users management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load the users list page with table and filter bar', async ({ page }) => {
    await navigateTo(page, '/system/users');
    await waitForPageLoad(page);

    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 15_000 });

    // FilterBar should be present (search/select inputs)
    const hasFilterInputs = await page
      .locator('input, .ant-select')
      .first()
      .isVisible({ timeout: 10_000 });
    expect(hasFilterInputs).toBeTruthy();
  });

  test('should display table rows from mock data', async ({ page }) => {
    await navigateTo(page, '/system/users');
    await waitForPageLoad(page);

    const tableBody = page.locator('.ant-table-tbody');
    await expect(tableBody).toBeVisible({ timeout: 15_000 });

    const rows = tableBody.locator('tr.ant-table-row');
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
  });

  test('should open "添加用户" drawer with PRD v1.1 fields when clicking primary add button', async ({
    page,
  }) => {
    await navigateTo(page, '/system/users');
    await waitForPageLoad(page);

    // Primary "添加" button (icon + label) — sits in the page header `extra` slot.
    await page.locator('button.ant-btn-primary').first().click();

    // Drawer titled "添加用户" should open
    const drawer = page.locator('.ant-drawer').filter({ hasText: '添加用户' });
    await expect(drawer).toBeVisible({ timeout: 5_000 });

    // PRD v1.1 字段：用户账号 / 用户昵称 / 角色 / 邮箱 / 备注
    await expect(drawer.locator('input[placeholder="用户账号"]')).toBeVisible();
    await expect(drawer.locator('input[placeholder="留空则与用户账号相同"]')).toBeVisible();
    await expect(drawer.locator('input[placeholder*="备注"]')).toBeVisible();
  });

  test('should not show legacy "导入用户" mode toggle (PRD v1.1 §5.1)', async ({ page }) => {
    await navigateTo(page, '/system/users');
    await waitForPageLoad(page);

    await page.locator('button.ant-btn-primary').first().click();

    // PRD v1.1 移除了"导入用户"模式 — 抽屉里不应出现该字样
    const drawer = page.locator('.ant-drawer').filter({ hasText: '添加用户' });
    await expect(drawer).toBeVisible({ timeout: 5_000 });
    await expect(drawer.getByText('导入用户')).toHaveCount(0);
  });
});
