import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 运维工具域冒烟：四个页面在真实后端下正常渲染。
 *
 * 选择器依据各页面源码 + i18n（默认 zh-CN，正则兼容 en-US）：
 *   - /ops/templates    src/pages/ops/Templates       标题 nav.ops.templates（运维模板 / Ops Templates）
 *                       + FilterBar(.ant-form) + DataTable(.ant-table) + ops.newTemplate 按钮
 *   - /ops/tasks        src/pages/ops/TaskManagement  标题硬编码中文「任务管理」（未 i18n）
 *                       + FilterBar + DataTable + 「新建任务」按钮
 *   - /ops/downloads    src/pages/ops/Downloads       占位页：Card + Empty（「运维下载子系统实施中」，硬编码中文）
 *   - /ops/message-trace src/pages/ops/MessageTrace   标题 trace.title（TR069 报文跟踪 / TR069 Message Trace）
 *                       + DataTable + trace.action.create 按钮（开始抓取 / Start Capture）
 *   - /ops/commands     src/pages/ops/CommandManagement  标题 nav.ops.commands（运维命令 / Ops Commands）
 *                       + FilterBar(.ant-form) + DataTable(.ant-table)（命令执行记录，仅查看 + 详情抽屉）
 *   - /ops/network-diagnosis src/pages/ops/NetworkDiagnosis  占位页：硬编码中文标题「网络诊断」
 *                       + Card + Empty（「网络诊断子系统实施中」）
 *   - /ops/aggregation-trigger src/pages/ops/AggregationTrigger  硬编码中文标题「PM 聚合手动触发」
 *                       + Card + Form + 「入队聚合任务」按钮
 *
 * 仅做"没崩 + 骨架在"级断言，不断言具体业务数据（真实栈数据稀疏，空表也有表头）。
 */
test.describe('运维工具冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/ops/templates 运维模板列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/ops/templates');

    // 页面标题（ListPageLayout → Typography.Title level=4 → h4）
    await expect(
      page.getByRole('heading', { name: /运维模板|Ops Templates/ }),
    ).toBeVisible();
    // 筛选栏表单 + 列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
    // 新建模板按钮（ops.newTemplate）
    await expect(page.getByRole('button', { name: /新建模板|New Template/ })).toBeVisible();
  });

  test('/ops/tasks 运维任务列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/ops/tasks');

    // 页面标题与新建按钮均为硬编码中文（页面尚未 i18n）
    await expect(page.getByRole('heading', { name: /任务管理/ })).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
    await expect(page.getByRole('button', { name: /新建任务/ })).toBeVisible();
  });

  test('/ops/downloads 运维下载占位页渲染', async ({ page }) => {
    await expectPageRenders(page, '/ops/downloads');

    // 占位页：标题 + Card 内 Empty（文案硬编码中文）
    // exact 避免与 Empty 内的「运维下载子系统实施中」h4 撞 strict mode
    await expect(page.getByRole('heading', { name: '运维下载', exact: true })).toBeVisible();
    await expect(page.locator('.ant-card').first()).toBeVisible();
    await expect(page.locator('.ant-empty').first()).toBeVisible();
    await expect(page.getByText(/运维下载子系统实施中/).first()).toBeVisible();
  });

  test('/ops/message-trace TR069 报文跟踪列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/ops/message-trace');

    // 页面标题（trace.title：zh 'TR069 报文跟踪' / en 'TR069 Message Trace'）
    await expect(
      page.getByRole('heading', { name: /TR069\s*(报文跟踪|Message Trace)/ }),
    ).toBeVisible();
    // 跟踪任务表格骨架 + 开始抓取按钮（trace.action.create）
    await expect(page.locator('.ant-table').first()).toBeVisible();
    await expect(
      page.getByRole('button', { name: /开始抓取|Start Capture/ }),
    ).toBeVisible();
  });

  test('/ops/commands 运维命令记录列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/ops/commands');

    // 页面标题（nav.ops.commands：zh '运维命令' / en 'Ops Commands'）
    await expect(
      page.getByRole('heading', { name: /运维命令|Ops Commands/ }),
    ).toBeVisible();
    // 筛选栏表单 + 命令记录表格骨架（空表也有表头）
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/ops/network-diagnosis 网络诊断占位页渲染', async ({ page }) => {
    await expectPageRenders(page, '/ops/network-diagnosis');

    // 占位页：标题硬编码中文「网络诊断」+ Card 内 Empty
    // exact 避免与 Empty 内的「网络诊断子系统实施中」h4 撞 strict mode
    await expect(page.getByRole('heading', { name: '网络诊断', exact: true })).toBeVisible();
    await expect(page.locator('.ant-card').first()).toBeVisible();
    await expect(page.locator('.ant-empty').first()).toBeVisible();
    await expect(page.getByText(/网络诊断子系统实施中/).first()).toBeVisible();
  });

  test('/ops/aggregation-trigger 聚合手动触发页渲染', async ({ page }) => {
    await expectPageRenders(page, '/ops/aggregation-trigger');

    // 标题硬编码中文「PM 聚合手动触发」+ Card 内表单 + 入队按钮
    await expect(
      page.getByRole('heading', { name: /PM 聚合手动触发/ }),
    ).toBeVisible();
    await expect(page.locator('.ant-card').first()).toBeVisible();
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.getByRole('button', { name: /入队聚合任务/ })).toBeVisible();
  });
});
