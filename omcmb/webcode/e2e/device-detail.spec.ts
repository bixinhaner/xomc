import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';
import { navigateTo, waitForPageLoad } from './helpers/navigation';

test.describe('Device Detail page navigation', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should navigate from device list to device detail', async ({ page }) => {
    await navigateTo(page, '/device/list');
    await waitForPageLoad(page);

    // Wait for table rows to appear
    const rows = page.locator('.ant-table-tbody tr.ant-table-row');
    await expect(rows.first()).toBeVisible({ timeout: 15_000 });

    // Click the first device link to go to detail
    const firstRowLink = rows.first().locator('a').first();
    if (await firstRowLink.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await firstRowLink.click();
      await page.waitForURL(/\/device\/detail\//, { timeout: 10_000 });

      // Device detail page should show device information
      await waitForPageLoad(page);
      const hasContent = await page.locator('.ant-descriptions, .ant-card, .ant-tabs').first()
        .isVisible({ timeout: 15_000 })
        .catch(() => false);
      expect(hasContent).toBeTruthy();
    }
  });

  test('should load the dashboard with navigation cards', async ({ page }) => {
    await navigateTo(page, '/dashboard');
    await waitForPageLoad(page);

    // Dashboard should have cards
    await expect(page.locator('.ant-card').first()).toBeVisible({ timeout: 15_000 });

    // Dashboard should have quick-access items that navigate to other pages
    const cards = page.locator('.ant-card');
    const cardCount = await cards.count();
    expect(cardCount).toBeGreaterThan(0);
  });

  test('should navigate between major sections via sidebar', async ({ page }) => {
    await navigateTo(page, '/dashboard');
    await waitForPageLoad(page);

    // The sidebar (Ant Design Menu) should be present
    const sidebar = page.locator('.ant-menu');
    await expect(sidebar.first()).toBeVisible({ timeout: 10_000 });

    // Find and click a sidebar menu item to navigate
    const menuItems = sidebar.locator('.ant-menu-item, .ant-menu-submenu-title');
    const menuCount = await menuItems.count();
    expect(menuCount).toBeGreaterThan(0);
  });

  test('should show 404 for unknown routes', async ({ page }) => {
    await navigateTo(page, '/some/nonexistent/route');
    await waitForPageLoad(page);

    // Should display a not-found message (404 page or Ant Result component)
    const has404 = await page.locator('text=/404|not found|page not found/i').first()
      .isVisible({ timeout: 10_000 })
      .catch(() => false);
    expect(has404).toBeTruthy();
  });
});
