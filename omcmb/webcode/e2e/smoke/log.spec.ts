import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 日志管理域冒烟：真实后端下四个日志页面能正常渲染。
 *
 * 选择器依据 src/pages/log/{DeviceLog,OperationLog,SystemLog,LogConfig}/index.tsx
 * + i18n（默认 zh-CN，正则兼容 en-US）：
 *   - /log/device    → 标题 log.deviceLog（设备上报日志 / Device Log Upload）+ 新建任务按钮 + 任务表格
 *   - /log/operation → 四个日志 Tab（log.operationLog 等）+ 日志表格（真实后端数据可能为空，只断言骨架）
 *   - /log/system    → 标题 nav.log.system（系统日志 / System Log）+ 日志表格
 *   - /log/config    → 标题 nav.log.config（日志配置 / Log Config）+ 配置表格 + 存储占用 Progress
 *
 * 注意：侧边栏菜单含同名文案（nav.log.*），标题断言一律收窄到 main 内的 heading，
 * 避免 strict mode 多匹配。
 */
test.describe('日志管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/log/device 设备上报日志渲染，标题/新建任务按钮/任务表格可见', async ({ page }) => {
    await expectPageRenders(page, '/log/device');

    // 页面标题（ListPageLayout 的 h4，收窄到 main 避开侧边栏同名菜单）
    await expect(
      page.locator('main').getByRole('heading', { name: /设备上报日志|Device Log Upload/ }),
    ).toBeVisible();

    // 新建任务按钮
    await expect(
      page.locator('main').getByRole('button', { name: /新建任务|New Task/ }),
    ).toBeVisible();

    // 任务列表表格骨架（空表也有表头）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/log/operation 操作日志渲染，四个日志 Tab 与表格可见', async ({ page }) => {
    await expectPageRenders(page, '/log/operation');

    // 四类日志 Tab（操作/安全/系统/北向接口）
    await expect(page.locator('.ant-tabs').first()).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /操作日志|Operation Log/ }),
    ).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /安全日志|Security Log/ }),
    ).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /北向接口日志|Northbound Log/ }),
    ).toBeVisible();

    // 日志列表表格骨架（真实后端数据可能稀疏，仅断言表格容器）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/log/system 系统日志渲染，标题与日志表格可见', async ({ page }) => {
    await expectPageRenders(page, '/log/system');

    // 页面标题（title 与 subtitle 同文案，用 heading role 收窄到 h4）
    await expect(
      page.locator('main').getByRole('heading', { name: /系统日志|System Log/ }).first(),
    ).toBeVisible();

    // 日志列表表格骨架
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/log/config 日志配置渲染，标题/配置表格/存储进度条可见', async ({ page }) => {
    await expectPageRenders(page, '/log/config');

    // 页面标题
    await expect(
      page.locator('main').getByRole('heading', { name: /日志配置|Log Config/ }).first(),
    ).toBeVisible();

    // 配置表格（静态初始数据，必有行）
    await expect(page.locator('main .ant-table').first()).toBeVisible();

    // 每行的存储占用 Progress 条（静态数据渲染）
    await expect(page.locator('main .ant-progress').first()).toBeVisible();
  });
});
