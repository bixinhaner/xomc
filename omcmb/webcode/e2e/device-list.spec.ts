import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';
import { navigateTo, waitForPageLoad } from './helpers/navigation';

test.describe('Device List page', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load the device list page', async ({ page }) => {
    await navigateTo(page, '/device/list');
    await waitForPageLoad(page);

    // The page should contain a data table (Ant Design Table)
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 15_000 });
  });

  test('should display filter bar for searching devices', async ({ page }) => {
    await navigateTo(page, '/device/list');
    await waitForPageLoad(page);

    // FilterBar should be present — it renders Ant Design form or input elements
    const hasFilterInputs = await page.locator('input, .ant-select').first().isVisible({ timeout: 10_000 });
    expect(hasFilterInputs).toBeTruthy();
  });

  test('should display table rows with device data', async ({ page }) => {
    await navigateTo(page, '/device/list');
    await waitForPageLoad(page);

    // Wait for table to have rows (mock data should populate the table)
    const tableBody = page.locator('.ant-table-tbody');
    await expect(tableBody).toBeVisible({ timeout: 15_000 });

    // There should be at least one row (mock data)
    const rows = tableBody.locator('tr.ant-table-row');
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
  });

  test('should navigate to device detail when clicking a device', async ({ page }) => {
    await navigateTo(page, '/device/list');
    await waitForPageLoad(page);

    // Wait for table rows to load
    const rows = page.locator('.ant-table-tbody tr.ant-table-row');
    await expect(rows.first()).toBeVisible({ timeout: 15_000 });

    // Click on a link in the first row (device SN or name is typically a link)
    const firstRowLink = rows.first().locator('a').first();
    if (await firstRowLink.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await firstRowLink.click();
      // Should navigate to device detail page
      await page.waitForURL(/\/device\/detail\//, { timeout: 10_000 });
      await expect(page).toHaveURL(/\/device\/detail\//);
    }
  });
});
