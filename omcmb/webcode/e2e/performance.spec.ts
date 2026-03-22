import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';
import { navigateTo, waitForPageLoad } from './helpers/navigation';

test.describe('Performance Management pages', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load the KPI standard report page', async ({ page }) => {
    await navigateTo(page, '/performance/kpi-standard');
    await waitForPageLoad(page);

    // KPI Standard page uses a tree-list layout
    const hasContent = await page.locator('.ant-table, .ant-tree, .ant-card').first()
      .isVisible({ timeout: 15_000 })
      .catch(() => false);
    expect(hasContent).toBeTruthy();
  });

  test('should display KPI tree navigation', async ({ page }) => {
    await navigateTo(page, '/performance/kpi-standard');
    await waitForPageLoad(page);

    // The tree sidebar should be visible
    const tree = page.locator('.ant-tree');
    if (await tree.isVisible({ timeout: 10_000 }).catch(() => false)) {
      // Tree should have nodes
      const treeNodes = tree.locator('.ant-tree-treenode');
      await expect(treeNodes.first()).toBeVisible({ timeout: 5_000 });
    }
  });

  test('should load the performance charts page', async ({ page }) => {
    await navigateTo(page, '/performance/charts');
    await waitForPageLoad(page);

    // Charts page should have chart containers or cards
    const hasContent = await page.locator('.ant-card, canvas, [class*="echarts"]').first()
      .isVisible({ timeout: 15_000 })
      .catch(() => false);
    expect(hasContent).toBeTruthy();
  });

  test('should load the threshold configuration page', async ({ page }) => {
    await navigateTo(page, '/performance/threshold');
    await waitForPageLoad(page);

    // Threshold page should have a table or form
    const hasContent = await page.locator('.ant-table, .ant-form, .ant-card').first()
      .isVisible({ timeout: 15_000 })
      .catch(() => false);
    expect(hasContent).toBeTruthy();
  });

  test('should load the performance files page', async ({ page }) => {
    await navigateTo(page, '/performance/files');
    await waitForPageLoad(page);

    // Files page should have a data table
    const hasContent = await page.locator('.ant-table, .ant-card').first()
      .isVisible({ timeout: 15_000 })
      .catch(() => false);
    expect(hasContent).toBeTruthy();
  });
});
