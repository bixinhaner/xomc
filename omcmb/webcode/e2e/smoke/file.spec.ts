import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 文件传输与管理域冒烟（真实后端）：四个页面渲染不崩 + 关键骨架元素可见。
 *
 * 选择器依据：
 *   - /transfer/center  → src/pages/transfer/FileTransferCenter/index.tsx
 *       标题 ufte.page.taskCreate（任务创建 / Task Management）、
 *       执行视图卡片 ufte.card.executionView（执行视图 / Execution View）、
 *       分类 Tabs（后端 ufte_task_types 驱动 + 虚拟 MR/KPI Tab）、任务列表表格。
 *   - /file/user-files  → src/pages/file/UserFiles/index.tsx
 *       标题 nav.file.userFiles（用户文件 / User Files）、
 *       Tabs：用户文件 + 固件上传（nav.software.firmware）、DataTable。
 *   - /file/config-retrieval → src/pages/file/ConfigRetrieval/index.tsx
 *       TreeListPageLayout：左侧设备树（.ant-tree）+ 右侧已选网元/采集结果两张表。
 *   - /file/device-files → src/pages/file/DeviceFiles/index.tsx
 *       标题 nav.file.deviceFiles（设备文件 / Device Files）、
 *       Tabs：配置文件采集（nav.file.configRetrieval）+ 日志采集（nav.file.logRetrieval）。
 *
 * 真实栈数据稀疏：只断言容器/标题/表格骨架，不断言具体业务数据。
 */
test.describe('文件传输与管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/transfer/center 文件传输中心渲染，分类 Tabs 与执行视图可见', async ({ page }) => {
    await expectPageRenders(page, '/transfer/center');

    // 页面标题（ListPageLayout title 与卡片内 Title 均为"任务创建"，取 first）
    await expect(page.getByText(/任务创建|Task Management/).first()).toBeVisible();

    // 业务分类 Tabs：后端内置模板分类（5G升级/配置文件备份等）+ 前端虚拟 MR 测量/KPI 导出 Tab
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /MR 测量|MR Measurement/ }).first(),
    ).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /5G升级|5G Upgrade/ }).first(),
    ).toBeVisible();

    // 已知前端竞态（见 issues）：首屏 task-types 未返回前 selectedCategory 被置为
    // 虚拟 'mr_measurement'，数据到达后不回退 → 默认 Tab 落在「MR 测量」，执行视图
    // 卡片被隐藏。切到真实分类「配置文件备份」验证执行视图正常渲染。
    await page.locator('.ant-tabs-tab').filter({ hasText: /配置文件备份|Config Backup/ }).first().click();
    await expect(
      page.locator('.ant-card-head-title').filter({ hasText: /执行视图|Execution View/ }),
    ).toBeVisible();

    // 任务列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/file/user-files 用户文件渲染，双 Tab 与文件表格可见', async ({ page }) => {
    await expectPageRenders(page, '/file/user-files');

    // 页面标题（侧边菜单同名，取 main 内首个）
    await expect(
      page.locator('main').getByText(/用户文件|User Files/).first(),
    ).toBeVisible();

    // Tabs：用户文件 / 固件上传
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /固件上传|Firmware Upload/ }).first(),
    ).toBeVisible();

    // 用户文件列表表格（真实后端 /files 数据可能为空，空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/file/config-retrieval 配置文件采集渲染，设备树与结果表格可见', async ({ page }) => {
    await expectPageRenders(page, '/file/config-retrieval');

    // 左侧设备树（TreeListPageLayout tree 面板，含 按类型/按子网 等 Tab）
    await expect(page.locator('.ant-tree').first()).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /按类型/ }).first(),
    ).toBeVisible();

    // 右侧：已选网元统计行（table.total：合计 / Total）+ 网元/结果两张表
    await expect(page.locator('main').getByText(/合计|Total/).first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/file/device-files 设备文件渲染，配置/日志双 Tab 与表格可见', async ({ page }) => {
    await expectPageRenders(page, '/file/device-files');

    // 页面标题（侧边菜单同名，取 main 内首个）
    await expect(
      page.locator('main').getByText(/设备文件|Device Files/).first(),
    ).toBeVisible();

    // 一级 Tabs：配置文件采集 / 日志采集
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /配置文件采集|Config Retrieval/ }).first(),
    ).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /日志采集|Log Retrieval/ }).first(),
    ).toBeVisible();

    // 默认 Tab（配置文件采集）下的文件表格
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });
});
