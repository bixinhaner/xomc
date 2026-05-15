import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';
import { navigateTo, waitForPageLoad } from './helpers/navigation';

async function openAlarmRules(page: Parameters<typeof test.beforeEach>[0] extends never ? never : import('@playwright/test').Page) {
  await navigateTo(page, '/alarm/rules');
  await waitForPageLoad(page);
  await expect(page.locator('.ant-table').first()).toBeVisible({ timeout: 15_000 });
}

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
    await openAlarmRules(page);

    // Should display a data table or configuration form
    const hasContent = await page.locator('.ant-table, .ant-form, .ant-card').first()
      .isVisible({ timeout: 15_000 })
      .catch(() => false);
    expect(hasContent).toBeTruthy();
  });

  test('should open add drawer and require selecting at least one alarm', async ({ page }) => {
    await openAlarmRules(page);

    await page.getByRole('button', { name: '新增' }).click();

    const drawer = page.locator('.ant-drawer-content').last();
    await expect(drawer).toBeVisible();

    await drawer.getByRole('button', { name: /确\s*认/ }).click();

    await expect(drawer.getByText('请至少选择一条告警')).toBeVisible();
  });

  test('should only allow editing disabled alarm rules', async ({ page }) => {
    await openAlarmRules(page);

    const enabledRow = page.locator('tr', { hasText: 'S1链路中断告警过滤' }).first();
    await expect(enabledRow.getByRole('button', { name: '编辑' })).toBeDisabled();

    const disabledRow = page.locator('tr', { hasText: '温度过高抑制规则' }).first();
    await disabledRow.getByRole('button', { name: '编辑' }).click();

    const drawer = page.locator('.ant-drawer-content').last();
    await expect(drawer).toBeVisible();
    await expect(drawer.getByPlaceholder(/规则名称/)).toHaveValue('温度过高抑制规则');
  });

  test('should create a new alarm rule', async ({ page }) => {
    await openAlarmRules(page);

    const ruleName = 'E2E告警规则新增用例';
    await page.getByRole('button', { name: '新增' }).click();

    const drawer = page.locator('.ant-drawer-content').last();
    await expect(drawer).toBeVisible();

    await drawer.getByPlaceholder(/规则名称/).fill(ruleName);

    const alarmTable = drawer.locator('.ant-table').last();
    await alarmTable.locator('tbody').getByRole('checkbox').first().check();

    await drawer.getByRole('button', { name: /确\s*认/ }).click();

    await expect(drawer).toBeHidden();
    await expect(page.getByText('共 24 条')).toBeVisible({ timeout: 15_000 });

    const pagination = page.locator('ul.ant-pagination').first();
    await pagination.getByText('2', { exact: true }).click();
    await expect(page.getByText(ruleName)).toBeVisible({ timeout: 15_000 });
  });
});
