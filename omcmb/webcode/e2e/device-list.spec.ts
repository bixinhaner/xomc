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

  test('日志收集下载列即使被旧列设置显式显示也不再出现', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('omc_col_vis_device-list-table', '[]');
      localStorage.setItem('omc_col_order_device-list-table', JSON.stringify(['latestLog']));
    });
    await navigateTo(page, '/device/list');
    await waitForPageLoad(page);

    await expect(page.locator('.ant-table').first()).toBeVisible();
    await expect(
      page.getByRole('columnheader', { name: 'SN', exact: true }).first(),
    ).toBeVisible();
    await expect(
      page.getByRole('columnheader', { name: /运行日志|Runtime Log/ }),
    ).toHaveCount(0);
  });

  test('日志收集触发成功后停留在设备列表并提示任务管理入口', async ({ page }) => {
    await navigateTo(page, '/device/list');
    await waitForPageLoad(page);

    const rows = page.locator('.ant-table-tbody tr.ant-table-row');
    await expect(rows.first()).toBeVisible({ timeout: 15_000 });
    let selectedOnlineDevice = false;
    for (let index = 0; index < await rows.count(); index += 1) {
      const checkbox = rows.nth(index).locator('.ant-checkbox-input');
      if (!(await checkbox.isDisabled())) {
        await checkbox.check({ force: true });
        selectedOnlineDevice = true;
        break;
      }
    }
    expect(selectedOnlineDevice).toBe(true);

    const collectButton = page.getByRole('button', { name: /日志收集|Log Collect/ });
    await expect(collectButton).toBeEnabled();
    await collectButton.click();

    const confirm = page.locator('.ant-modal-confirm');
    await expect(confirm).toBeVisible();
    await confirm.getByRole('button', { name: /确\s*认|Confirm/ }).click();

    await expect(
      page.getByText(
        /日志收集已触发.*文件传输.*任务管理|Log collection triggered.*File Transfer.*Task Management/,
      ),
    ).toBeVisible();
    await expect(page).toHaveURL(/\/device\/list(?:\?|$)/);
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
