import { test, expect, type Page } from '@playwright/test';
import { login } from './helpers/auth';
import { navigateTo, waitForPageLoad } from './helpers/navigation';

/**
 * #268 告警库三个回归(against mock dev server):
 *   1. 手工新增行(无加载源)可下载 → 触发 <neType>-manual.xml 下载事件
 *   2. 二级页修改告警级别 → 返回一级后按严重性统计刷新
 *   3. 二级筛选(搜索词)→ 返回再进入 → 上一次筛选被清除
 */

async function openAlarmLibrary(page: Page) {
  await navigateTo(page, '/product/alarm-library');
  await waitForPageLoad(page);
  await expect(page.locator('.ant-table').first()).toBeVisible({ timeout: 15_000 });
}

// 一级表:序号|名称|加载源|告警总数|严重|主要|次要|警告|操作 → 严重=td[4],主要=td[5]
function severityCell(page: Page, rowText: string, index: number) {
  return page
    .locator('.ant-table-tbody tr')
    .filter({ hasText: rowText })
    .first()
    .locator('td')
    .nth(index);
}

test.describe('告警库 drill-down(#268)', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('#268-1 手工新增行的下载图标可用并触发下载', async ({ page }) => {
    await openAlarmLibrary(page);

    const manualRow = page
      .locator('.ant-table-tbody tr')
      .filter({ hasText: '手工新增' })
      .first();
    await expect(manualRow).toBeVisible();

    // 操作列(最后一格)第一个按钮 = 下载图标(#268 前 disabled={!loadedFrom} 置灰)
    const downloadBtn = manualRow.locator('td').last().locator('.ant-btn').first();
    await expect(downloadBtn).toBeEnabled();

    const dlPromise = page.waitForEvent('download');
    await downloadBtn.click();
    const download = await dlPromise;
    expect(download.suggestedFilename()).toBe('eNodeB-manual.xml');
  });

  test('#268-2 修改告警级别后返回一级,严重性统计刷新', async ({ page }) => {
    await openAlarmLibrary(page);

    // mock 文件行:eNodeB.xml,严重=1(101002 Critical)
    const fileRow = page
      .locator('.ant-table-tbody tr')
      .filter({ hasText: 'eNodeB.xml' })
      .first();
    await expect(severityCell(page, 'eNodeB.xml', 4)).toHaveText(/1/);

    // 下钻 → 编辑 101002 的严重级别 Critical → Major
    await fileRow.getByRole('button', { name: 'eNodeB' }).click();
    await expect(page.getByText('eNodeB 的告警定义', { exact: false })).toBeVisible();

    const detailRow = page
      .locator('.ant-table-tbody tr')
      .filter({ hasText: '101002' })
      .first();
    await detailRow.getByRole('button', { name: '编辑' }).click();

    const drawer = page.locator('.ant-drawer').filter({ hasText: '编辑告警定义' }).first();
    await expect(drawer).toBeVisible();
    await drawer.locator('.ant-select').first().click();
    await page.locator('.ant-select-item-option').filter({ hasText: '主要' }).first().click();
    await drawer.getByRole('button', { name: /保\s*存/ }).click();
    await expect(drawer).toBeHidden({ timeout: 10_000 });

    // 返回一级 → 严重列归零(—),主要列变 2(mock 有态更新 + ne-types 失效重取)
    await page.getByRole('button', { name: '返回' }).click();
    await expect(page.locator('.ant-table-tbody tr').filter({ hasText: 'eNodeB.xml' }).first()).toBeVisible();
    await expect(severityCell(page, 'eNodeB.xml', 4)).toHaveText('—', { timeout: 10_000 });
    await expect(severityCell(page, 'eNodeB.xml', 5)).toHaveText(/2/);
  });

  test('#268-3 返回再进入,上一次筛选被清除', async ({ page }) => {
    await openAlarmLibrary(page);

    const fileRow = page
      .locator('.ant-table-tbody tr')
      .filter({ hasText: 'eNodeB.xml' })
      .first();
    await fileRow.getByRole('button', { name: 'eNodeB' }).click();
    await expect(page.getByText('eNodeB 的告警定义', { exact: false })).toBeVisible();

    // 设置筛选:搜索 101001 → 明细只剩 1 行
    const search = page.getByPlaceholder('搜索 标识符 / 名称 / 描述');
    await search.fill('101001');
    await search.press('Enter');
    await expect(page.locator('.ant-table-tbody tr').filter({ hasText: '101001' })).toHaveCount(1, {
      timeout: 10_000,
    });

    // 返回 → 再次进入:搜索框应被清空,明细回到全量(3 行数据)
    await page.getByRole('button', { name: '返回' }).click();
    await expect(page.locator('.ant-table-tbody tr').filter({ hasText: 'eNodeB.xml' }).first()).toBeVisible();
    await fileRow.getByRole('button', { name: 'eNodeB' }).click();
    await expect(page.getByText('eNodeB 的告警定义', { exact: false })).toBeVisible();

    await expect(page.getByPlaceholder('搜索 标识符 / 名称 / 描述')).toHaveValue('');
    await expect(
      page.locator('.ant-table-tbody tr').filter({ hasText: '101002' }),
    ).toBeVisible({ timeout: 10_000 });
  });
});
