import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 配置备份域冒烟（真实后端）：六个路由能渲染且关键骨架可见。
 *
 * 选择器依据：
 *   - /backup/tasks            src/pages/backup/BackupTasks（ListPageLayout 标题 nav.backup.tasks
 *                              + Radio.Group 任务/设备列表页签 + FilterBar + DataTable）
 *   - /backup/schedule         src/pages/backup/BackupSchedule（标题 backup.scheduleListTitle
 *                              + 新建调度按钮 backup.newSchedule + DataTable）
 *   - /backup/policy           src/pages/backup/BackupPolicy（标题 nav.backup.policy + Card 内
 *                              Form + Collapse，分组标题 backup.policy.retention 等）
 *   - /backup/config-snapshots src/pages/backup/ConfigSnapshotLibrary（无页面标题；筛选 Card
 *                              transfer.fileLib.filter.* + 导入配置按钮 + DataTable）
 *   - /backup/ftp              src/pages/backup/FTPConfig（ListPageLayout 标题 nav.backup.ftp
 *                              + 新增按钮 common.add + DataTable）
 *   - /backup/restore          src/pages/backup/RestoreData（ListPageLayout 标题 backup.restore.title
 *                              + 创建恢复按钮 backup.restore.create + 刷新按钮 + DataTable）
 * 文案均取自 frontend-core/src/i18n/{zh-CN,en-US}，正则兼容双语；不断言具体业务数据
 * （真实栈数据稀疏，空表也有表头，断言 .ant-table 可见即可）。
 * 标题/按钮断言统一收窄到 <main>，避免命中侧边栏同名菜单项。
 */
test.describe('配置备份冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/backup/tasks 备份任务页渲染，页签与表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/backup/tasks');
    const main = page.locator('main');

    // 页面标题（nav.backup.tasks：备份任务 / Backup Tasks）
    await expect(
      main.locator('h4.ant-typography').filter({ hasText: /备份任务|Backup Tasks/ }),
    ).toBeVisible();

    // 任务/设备两级页签（backup.taskList / backup.deviceList）
    await expect(main.getByText(/任务列表|Task List/).first()).toBeVisible();
    await expect(main.getByText(/设备列表|Device List/).first()).toBeVisible();

    // 列表骨架（空表也有表头）
    await expect(main.locator('.ant-table').first()).toBeVisible();
  });

  test('/backup/schedule 备份调度页渲染，新建调度入口与表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/backup/schedule');
    const main = page.locator('main');

    // 页面标题（backup.scheduleListTitle：备份调度任务 / Backup Schedules）
    await expect(
      main.locator('h4.ant-typography').filter({ hasText: /备份调度任务|Backup Schedules/ }),
    ).toBeVisible();

    // 新建调度按钮（backup.newSchedule）
    await expect(
      main.getByRole('button', { name: /新建调度|New Schedule/ }),
    ).toBeVisible();

    // 调度列表骨架
    await expect(main.locator('.ant-table').first()).toBeVisible();
  });

  test('/backup/policy 备份策略页渲染，策略表单与分组面板可见', async ({ page }) => {
    await expectPageRenders(page, '/backup/policy');
    const main = page.locator('main');

    // 页面标题（nav.backup.policy：备份策略 / Backup Policy）
    await expect(
      main.locator('h4.ant-typography').filter({ hasText: /备份策略|Backup Policy/ }),
    ).toBeVisible();

    // 策略表单（Card 内 Form + Collapse 分组）
    await expect(main.locator('.ant-form').first()).toBeVisible();
    await expect(
      main.locator('.ant-collapse-header').filter({ hasText: /保留策略|Retention Policy/ }),
    ).toBeVisible();
  });

  test('/backup/config-snapshots 配置快照库渲染，筛选区与表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/backup/config-snapshots');
    const main = page.locator('main');

    // 筛选区输入框（transfer.fileLib.filter.sn：设备 SN / Device SN）
    await expect(main.getByPlaceholder(/设备 SN|Device SN/).first()).toBeVisible();

    // 导入配置入口（transfer.fileLib.action.importConfig）
    await expect(
      main.getByRole('button', { name: /导入配置|Import Config/ }),
    ).toBeVisible();

    // 快照列表骨架
    await expect(main.locator('.ant-table').first()).toBeVisible();
  });

  test('/backup/ftp FTP 配置页渲染，新增入口与表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/backup/ftp');
    const main = page.locator('main');

    // 页面标题（nav.backup.ftp：FTP配置 / FTP Config）
    await expect(
      main.locator('h4.ant-typography').filter({ hasText: /FTP\s*配置|FTP Config/ }),
    ).toBeVisible();

    // 新增 FTP 配置按钮（common.add：新增 / Add）
    await expect(
      main.getByRole('button', { name: /新增|Add/ }).first(),
    ).toBeVisible();

    // FTP 配置列表骨架（空表也有表头）
    await expect(main.locator('.ant-table').first()).toBeVisible();
  });

  test('/backup/restore 配置恢复页渲染，创建恢复入口与表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/backup/restore');
    const main = page.locator('main');

    // 页面标题（backup.restore.title：数据恢复 / Data Restore）
    await expect(
      main.locator('h4.ant-typography').filter({ hasText: /数据恢复|Data Restore/ }),
    ).toBeVisible();

    // 创建恢复按钮（backup.restore.create：创建恢复 / Create Restore）
    await expect(
      main.getByRole('button', { name: /创建恢复|Create Restore/ }),
    ).toBeVisible();

    // 恢复任务列表骨架（空表也有表头）
    await expect(main.locator('.ant-table').first()).toBeVisible();
  });
});
