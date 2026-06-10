import { test, expect } from '@playwright/test';
import { SMOKE_CREDENTIALS, smokeLogin } from './helpers';

/**
 * 认证域冒烟：真实后端登录成功 / 失败两条路径。
 */
test.describe('认证冒烟（真实后端）', { tag: '@smoke' }, () => {
  test('正确凭据登录成功并跳离 /login', async ({ page }) => {
    await smokeLogin(page);

    // login() 已等待跳离 /login；再确认落在 dashboard
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test('错误密码停留在 /login 并出现错误提示', async ({ page }) => {
    await page.goto('/login');

    await page.getByPlaceholder(/username|user|用户名/i).first().fill(SMOKE_CREDENTIALS.username);
    await page.getByPlaceholder(/password|密码/i).first().fill('definitely-wrong-password');
    await page.locator('button[type="submit"]').click();

    // 错误提示（antd message toast 或表单错误）
    await expect(
      page.locator('.ant-message').or(page.locator('.ant-form-item-explain-error')).first(),
    ).toBeVisible();

    // 仍停留在登录页
    await expect(page).toHaveURL(/\/login/);
  });
});
