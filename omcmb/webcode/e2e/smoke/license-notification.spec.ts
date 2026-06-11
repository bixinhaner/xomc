import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 许可与通知域冒烟（真实后端）：/license、/license/history、/notifications。
 *
 * 选择器依据：
 *   - src/pages/SystemLicense/index.tsx + History.tsx
 *     （后端 system_license 表为空时 GetCurrent 返回 biz_code 12113，
 *       页面应渲染"未配置"引导空态而非 ErrorBoundary 崩溃——这正是冒烟要断言的）
 *   - src/pages/notifications/index.tsx（ListPageLayout + Tabs：通知模板 / 发送历史，
 *     TemplateList 内部 DataTable 始终渲染 antd Table，空数据也有表头）
 *   - i18n 语料 frontend-core/src/i18n/{zh-CN,en-US}/index.ts 的
 *     systemLicense.* / notification.* 段，正则兼容双语
 */
test.describe('许可与通知冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/license 未配置 License 时优雅显示空态引导（不崩溃）', async ({ page }) => {
    await expectPageRenders(page, '/license');

    // 空态走 antd Result（icon + title + extra 按钮），不是 ErrorBoundary
    // zh: 系统未配置许可证 / en: No system license configured
    await expect(page.locator('.ant-result')).toBeVisible();
    await expect(
      page.getByText(/系统未配置许可证|No system license configured/).first(),
    ).toBeVisible();

    // 空态引导按钮：systemLicense.update（两个语种均为 "Update"）
    await expect(page.getByRole('button', { name: /Update/ }).first()).toBeVisible();

    // 不应出现"真错误"分支的 error Result（common.error）
    await expect(page.locator('.ant-result-error')).not.toBeVisible();
  });

  test('/license/history 渲染历史列表页（空历史时显示空态）', async ({ page }) => {
    await expectPageRenders(page, '/license/history');

    // 页面标题：systemLicense.history.title（License 历史 / License History）
    await expect(page.getByText(/License 历史|License History/).first()).toBeVisible();

    // 数据区：有历史时是 antd Table，空历史时是 Empty（暂无历史 license）。
    // 真实栈数据稀疏，两态都算渲染正常。
    await expect(
      page.locator('.ant-table').or(page.locator('.ant-empty')).first(),
    ).toBeVisible();
  });

  test('/notifications 渲染通知中心，模板/历史两个 Tab 与表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/notifications');

    // ListPageLayout 标题：notification.title（通知中心 / Notification Center）
    await expect(page.getByText(/通知中心|Notification Center/).first()).toBeVisible();

    // Tabs：通知模板（默认激活）+ 发送历史
    await expect(page.locator('.ant-tabs')).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /通知模板|Templates/ }),
    ).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /发送历史|History/ }),
    ).toBeVisible();

    // 默认 Tab（通知模板）内 DataTable 骨架：空数据也渲染 .ant-table 表头
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });
});
