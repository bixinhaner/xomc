import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';
import { login } from '../helpers/auth';
import { navigateTo, waitForPageLoad } from '../helpers/navigation';

/**
 * 冒烟测试共享工具（打真实后端）。
 *
 * 凭据默认 admin/admin123（与本地真实栈一致），可用环境变量
 * E2E_USERNAME / E2E_PASSWORD 覆盖。
 */
export const SMOKE_CREDENTIALS = {
  username: process.env.E2E_USERNAME || 'admin',
  password: process.env.E2E_PASSWORD || 'admin123',
};

/**
 * 用冒烟凭据走 UI 表单登录（复用 ../helpers/auth 的 login）。
 * 真实后端登录（RSA 加密 + getMe）比 mock 慢，调用方注意预留超时。
 */
export async function smokeLogin(page: Page): Promise<void> {
  await login(page, SMOKE_CREDENTIALS);
}

/**
 * 断言指定路径的页面能正常渲染（最低限度的"没崩"检查）：
 *   1. 导航 + 等待 Suspense/Spin 加载完成；
 *   2. 无 ErrorBoundary 崩溃特征（"页面发生错误/Page Error" 文案、.ant-result-500）；
 *   3. 页面主体容器（布局 <main>）可见。
 */
export async function expectPageRenders(page: Page, path: string): Promise<void> {
  await navigateTo(page, path);
  await waitForPageLoad(page);

  // 未被 PrivateRoute 弹回登录页
  await expect(page).not.toHaveURL(/\/login/);

  // ErrorBoundary fallback 文案（zh-CN: 页面发生错误 / en-US: Page Error）
  await expect(
    page.getByText(/页面发生错误|Page Error|something went wrong|页面出错/i),
  ).not.toBeVisible();
  // 路由级 500 错误页
  await expect(page.locator('.ant-result-500')).not.toBeVisible();

  // 布局主体容器已渲染（Layout 用 <main> 包裹页面内容）
  await expect(page.locator('main')).toBeVisible();
}
