import type { Page } from '@playwright/test';

/**
 * Default mock-mode credentials.
 * In mock mode the login page accepts any username with password "admin123".
 */
export const MOCK_CREDENTIALS = {
  username: 'admin',
  password: 'admin123',
} as const;

export async function submitLoginForm(
  page: Page,
  credentials: { username: string; password: string } = MOCK_CREDENTIALS,
): Promise<void> {
  await page.getByPlaceholder(/username|user|用户名/i).first().fill(credentials.username);
  await page.getByPlaceholder(/password|密码/i).first().fill(credentials.password);
  await page.locator('button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 15_000 });
}

/**
 * Perform a login through the UI form.
 * After this helper returns the page should be on the dashboard (or the
 * redirect target passed via `redirectTo`).
 */
export async function login(
  page: Page,
  credentials: { username: string; password: string } = MOCK_CREDENTIALS,
): Promise<void> {
  await page.goto('/login');
  await submitLoginForm(page, credentials);
}

/**
 * Logout via the user dropdown in the header.
 */
export async function logout(page: Page): Promise<void> {
  // Open the user dropdown (click the user trigger area in the header)
  await page
    .locator('button[aria-haspopup="menu"]')
    .filter({ has: page.locator('.ant-avatar') })
    .click();

  // Click the logout menu item
  await page.getByRole('menuitem', { name: /logout|log out|exit|退出登录/i }).click();

  // Wait for redirect to login page
  await page.waitForURL('**/login', { timeout: 10_000 });
}
