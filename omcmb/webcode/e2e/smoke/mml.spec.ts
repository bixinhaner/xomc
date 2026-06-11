import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * MML 控制台域冒烟：七个页面在真实后端下正常渲染。
 *
 * 路由依据 src/router/routes.tsx（mml/console 是侧边栏主入口 nav.mml.console，
 * mml/console-v2 为重设计版独立路由，mml/commands 在导航中隐藏但路由仍在）：
 *   - /mml/console      src/pages/mml/Console      三栏布局（设备树｜命令树｜操作面板）
 *                       + StepBar(.ant-steps，mml.console.stepBar.step1-4)
 *                       + DeviceTree 头部 nav.device.list（设备列表）与 mml.console.batchInput（批量输入）
 *   - /mml/console-v2   src/pages/mml/ConsoleV2    顶部 SelectionBar（硬编码中文：
 *                       ① 选择设备 / ② 选择命令 / ③ 配置参数 / 执行）
 *   - /mml/commands     src/pages/mml/CommandTree  TreeListPageLayout：
 *                       左侧分类树(.ant-tree，根节点 mml.category.all)
 *                       + 右侧命令表(.ant-table) + 头部 nav.mml.commands（命令树）
 *   - /mml/task-records src/pages/mml/TaskRecord   ListPageLayout 标题 nav.mml.taskRecord（任务记录）
 *                       + FilterBar(.ant-form，mml.taskName 任务名称) + DataTable(.ant-table)
 *   - /mml/script       src/pages/mml/ScriptTask   ListPageLayout 标题 nav.mml.script（脚本任务，
 *                       读 mml_scripts）+ 新增脚本按钮(mml.newScript) + DataTable(.ant-table)
 *   - /mml/admin/catalog src/pages/mml/admin/catalog 命令目录后台（withAdminRole 守卫，
 *                       admin 凭据可入，被拦则跳 /403）；Card 布局：左 LeftNavTree(.ant-tree)
 *                       + 顶部「新增分组」按钮(mml.admin.catalog.groups.add)
 *                       + 搜索框(mml.admin.catalog.groups.searchPlaceholder)
 *   - /mml/private-command src/pages/mml/PrivateCommand 私有命令（withSuspense，无角色守卫）
 *                       ListPageLayout 标题 mml.privateCommand.pageTitle（私有命令）
 *                       + 新增私有命令按钮(mml.console.addPrivateTemplate) + DataTable(.ant-table)
 *
 * 仅做"没崩 + 骨架在"级断言，不断言具体业务数据（真实栈数据稀疏，空表也有表头）。
 * 文案默认 zh-CN，正则兼容 en-US。
 */
test.describe('MML控制台冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/mml/console MML控制台（主入口）三栏布局渲染', async ({ page }) => {
    await expectPageRenders(page, '/mml/console');

    // 顶部步骤条（StepBar：选择设备 → 选择命令 → 配置参数 → 查看结果）
    await expect(page.locator('.ant-steps').first()).toBeVisible();
    await expect(page.getByText(/选择设备|Select device/).first()).toBeVisible();
    await expect(page.getByText(/配置参数|Configure/).first()).toBeVisible();

    // 左栏 DeviceTree 头部：设备列表标题 + 批量输入按钮
    await expect(page.getByText(/设备列表|Device List/).first()).toBeVisible();
    await expect(
      page.getByRole('button', { name: /批量输入|Batch Input/ }),
    ).toBeVisible();
  });

  test('/mml/console-v2 MML控制台V2 顶部选择条渲染', async ({ page }) => {
    await expectPageRenders(page, '/mml/console-v2');

    // SelectionBar 三个步骤按钮 + 执行按钮（组件内硬编码中文，无 en-US 分支）
    await expect(page.getByRole('button', { name: /① 选择设备/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /② 选择命令/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /③ 配置参数/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /执行/ }).first()).toBeVisible();
  });

  test('/mml/commands 命令树（分类树 + 命令列表）渲染', async ({ page }) => {
    await expectPageRenders(page, '/mml/commands');

    // 左侧分类树（根节点 mml.category.all：全部命令 / All Commands）
    await expect(page.locator('.ant-tree').first()).toBeVisible();
    await expect(page.getByText(/全部命令|All Commands/).first()).toBeVisible();

    // 右侧头部（未选分类时显示 nav.mml.commands）+ 命令表格骨架
    await expect(page.getByText(/命令树|Command Tree/).first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/mml/task-records 任务记录列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/mml/task-records');

    // 页面标题（ListPageLayout → Typography.Title → heading；与侧边栏菜单项区分）
    await expect(
      page.getByRole('heading', { name: /任务记录|Task Records/ }),
    ).toBeVisible();
    // 筛选栏表单（任务名称/类型/状态/结果）+ 任务表格骨架（空表也有表头）
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.getByText(/任务名称|Task Name/).first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/mml/script 脚本任务（脚本库列表）渲染', async ({ page }) => {
    await expectPageRenders(page, '/mml/script');

    // 页面标题 nav.mml.script（heading 角色，避开同名侧边栏菜单项）
    await expect(
      page.getByRole('heading', { name: /脚本任务|Script Task/ }),
    ).toBeVisible();
    // 头部"新增脚本"按钮（mml.newScript）+ 脚本表格骨架（空表也有表头）
    await expect(
      page.getByRole('button', { name: /新增脚本|New Script/ }),
    ).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/mml/admin/catalog 命令目录后台（admin 守卫）渲染', async ({ page }) => {
    // withAdminRole 守卫页：admin 凭据角色为 admin → 放行；若被拦则跳 /403。
    // <Navigate replace> 是 hydration 后的客户端跳转，goto 完成时 URL 可能还停在
    // 目标路径，故先等任一落点（命令目录骨架 .ant-tree 或 /403 结果页）稳定，
    // 再判定走哪条断言分支。
    await page.goto('/mml/admin/catalog');
    await page.waitForLoadState('domcontentloaded');
    await expect(async () => {
      const onForbidden = /\/403/.test(page.url());
      const treeVisible = await page
        .locator('.ant-tree')
        .first()
        .isVisible()
        .catch(() => false);
      expect(onForbidden || treeVisible).toBe(true);
    }).toPass({ timeout: 15_000 });

    if (/\/403/.test(page.url())) {
      // 守卫生效（admin 也被拦）：断言重定向本身即可，亦是合法冒烟结果。
      await expect(page).toHaveURL(/\/403/);
      return;
    }

    await expectPageRenders(page, '/mml/admin/catalog');

    // 顶部「新增分组」主按钮（mml.admin.catalog.groups.add）
    await expect(
      page.getByRole('button', { name: /新增分组|Add Group/ }),
    ).toBeVisible();
    // 搜索框 placeholder（mml.admin.catalog.groups.searchPlaceholder）
    await expect(
      page.getByPlaceholder(/搜索分组|Search groups/),
    ).toBeVisible();
    // 左侧分类导航树（LeftNavTree → antd Tree，加载完成后渲染骨架）
    await expect(page.locator('.ant-tree').first()).toBeVisible();
  });

  test('/mml/private-command 私有命令列表渲染', async ({ page }) => {
    // withSuspense（无角色守卫），admin 凭据直接可入。
    await expectPageRenders(page, '/mml/private-command');

    // 页面标题 mml.privateCommand.pageTitle（heading 角色，避开同名侧边栏菜单项）
    await expect(
      page.getByRole('heading', { name: /私有命令|Private Commands/ }),
    ).toBeVisible();
    // 右上角「新增私有命令」按钮（mml.console.addPrivateTemplate）
    await expect(
      page.getByRole('button', { name: /新增私有命令|Add Private Command/ }),
    ).toBeVisible();
    // 私有命令表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });
});
