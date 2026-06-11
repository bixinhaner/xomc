import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 系统管理域冒烟（真实后端）：7 个页面渲染不崩 + 关键骨架元素可见。
 *
 * 选择器依据：
 *   - /system/operation-log → src/pages/system/OperationLog/index.tsx
 *       （re-export src/pages/log/OperationLog/index.tsx）四类日志 Tabs
 *       （log.operationLog 操作日志 / Operation Log 等 4 个 tab）+ 导出按钮
 *       （common.export 导出 / Export）+ .ant-table（空表也有表头）
 *   - /system/api-management → src/pages/system/ApiManagement/index.tsx
 *       FilterBar + 工具栏按钮（同步 / Sync、批量删除 / Batch Delete、新增 / Add）
 *       + .ant-table；注意 api.name 文案是「简介 / Summary」非「名称」
 *   - /system/users  → src/pages/system/UserManagement/index.tsx
 *       无页面标题（2026-06-03 决策移除），断言 FilterBar 标签
 *       user.userName（用户名称 / Username）+ 新增按钮 + .ant-table
 *   - /system/roles  → src/pages/system/RolePermission/index.tsx
 *       FilterBar 标签 role.roleName（角色名称 / Role Name）+ .ant-table
 *   - /system/config → src/pages/system/SystemConfig/index.tsx
 *       Tabs（system.config.basic 基本设置 / Basic Settings 等 9 个 tab）
 *   - /system/menus  → src/pages/system/MenuManagement/index.tsx
 *       列头硬编码中文「菜单名称（中文）」+ 新增按钮 + .ant-table
 *   - /system/data-dictionary → src/pages/system/DataDictionary/index.tsx
 *       左右双栏：dictionary.listTitle（字典列表 / Dictionary List）
 *       + dictionary.detailTitle（字典详细内容 / Dictionary Details）；
 *       未选中字典时右侧是 Empty，不断言 .ant-table
 *
 * 不断言具体业务数据（真实栈数据稀疏）；列表页 .ant-table 空表也有表头。
 */
test.describe('系统管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/system/operation-log 操作日志渲染，日志 Tabs + 导出按钮 + 表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/system/operation-log');

    // 四类日志 Tabs 容器
    await expect(page.locator('.ant-tabs').first()).toBeVisible({ timeout: 30_000 });
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /操作日志|Operation Log/ }).first(),
    ).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /安全日志|Security Log/ }).first(),
    ).toBeVisible();
    // 导出按钮（common.export 导出 / Export）
    await expect(page.getByRole('button', { name: /导出|Export/ }).first()).toBeVisible();
    // 日志列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible({ timeout: 30_000 });
  });

  test('/system/api-management API 管理渲染，工具栏按钮 + 表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/system/api-management');

    // 工具栏按钮（同步 / Sync、批量删除 / Batch Delete、新增 / Add）
    await expect(page.getByRole('button', { name: /同步|Sync/ }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /批量删除|Batch Delete/ }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();
    // API 端点列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible({ timeout: 30_000 });
  });

  test('/system/users 用户管理渲染，筛选条 + 表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/system/users');

    // FilterBar：用户名称筛选输入框（FilterBar 只渲染 placeholder 不渲染可见 label，
    // 且用户表列头是「用户昵称/用户账号」，故用 placeholder 断言）
    await expect(page.getByPlaceholder(/用户名称|Username/).first()).toBeVisible();
    // 顶部操作按钮（新增 / Add）
    await expect(page.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();
    // 用户列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible({ timeout: 30_000 });
  });

  test('/system/roles 角色权限渲染，筛选条 + 表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/system/roles');

    // 角色名称（FilterBar 标签 + 表格列头都会命中，取 first）
    await expect(page.getByText(/角色名称|Role Name/).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible({ timeout: 30_000 });
  });

  test('/system/config 系统配置渲染，设置 Tabs 可见', async ({ page }) => {
    await expectPageRenders(page, '/system/config');

    // 配置页是 Tabs 容器（9 个设置 tab）
    await expect(page.locator('.ant-tabs').first()).toBeVisible({ timeout: 30_000 });
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /基本设置|Basic Settings/ }).first(),
    ).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /安全设置|Security Settings/ }).first(),
    ).toBeVisible();
  });

  test('/system/menus 菜单管理渲染，菜单表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/system/menus');

    await expect(page.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();
    // 菜单树表格（GET /admin/menus/tree；列头「菜单名称（中文）」为硬编码中文）
    await expect(page.locator('.ant-table').first()).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('.ant-table').getByText(/菜单名称/).first()).toBeVisible();
  });

  test('/system/data-dictionary 数据字典渲染，左右双栏标题可见', async ({ page }) => {
    await expectPageRenders(page, '/system/data-dictionary');

    // 左栏：字典列表标题 + 新增字典按钮
    await expect(page.getByText(/字典列表|Dictionary List/).first()).toBeVisible();
    await expect(
      page.getByRole('button', { name: /新增字典|Add Dictionary/ }).first(),
    ).toBeVisible();
    // 右栏：字典详细内容标题（未选中字典时表格区域是 Empty，不断言表格）
    await expect(page.getByText(/字典详细内容|Dictionary Details/).first()).toBeVisible();
  });
});
