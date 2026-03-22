import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';
import { navigateTo, waitForPageLoad } from './helpers/navigation';

test.describe('Alarm Management pages', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load the current alarms page', async ({ page }) => {
    await navigateTo(page, '/alarm/current');
    await waitForPageLoad(page);

    // Should display a data table for current alarms
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 15_000 });
  });

  test('should display alarm filter controls', async ({ page }) => {
    await navigateTo(page, '/alarm/current');
    await waitForPageLoad(page);

    // Filter bar should have inputs or select controls for alarm filtering
    const filterArea = page.locator('input, .ant-select, .ant-picker');
    await expect(filterArea.first()).toBeVisible({ timeout: 10_000 });
  });

  test('should load the historical alarms page', async ({ page }) => {
    await navigateTo(page, '/alarm/history');
    await waitForPageLoad(page);

    // Should display a data table for historical alarms
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 15_000 });
  });

  test('should load the alarm statistics page', async ({ page }) => {
    await navigateTo(page, '/alarm/statistics');
    await waitForPageLoad(page);

    // The statistics page should render charts or statistic cards
    const hasContent = await page.locator('.ant-card, .ant-statistic, canvas, [class*="echarts"]').first()
      .isVisible({ timeout: 15_000 })
      .catch(() => false);
    expect(hasContent).toBeTruthy();
  });

  test('should load the alarm rules page', async ({ page }) => {
    await navigateTo(page, '/alarm/rules');
    await waitForPageLoad(page);

    // Should display a data table or configuration form
    const hasContent = await page.locator('.ant-table, .ant-form, .ant-card').first()
      .isVisible({ timeout: 15_000 })
      .catch(() => false);
    expect(hasContent).toBeTruthy();
  });
});
