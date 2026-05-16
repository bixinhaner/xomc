import type { Page } from '@playwright/test';

/**
 * Default mock-mode credentials.
 * In mock mode the login page accepts any username with password "admin123".
 */
export const MOCK_CREDENTIALS = {
  username: 'admin',
  password: 'admin123',
} as const;

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

  // Fill in the login form
  await page.getByPlaceholder(/username|user|用户名/i).first().fill(credentials.username);
  await page.getByPlaceholder(/password|密码/i).first().fill(credentials.password);

  // Submit
  await page.locator('button[type="submit"]').click();

  // Wait until we are redirected away from /login
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 15_000 });
}

/**
 * Logout via the user dropdown in the header.
 */
export async function logout(page: Page): Promise<void> {
  // Open the user dropdown (click the user trigger area in the header)
  await page.locator('.ant-dropdown-trigger').filter({ has: page.locator('.ant-avatar') }).click();

  // Click the logout menu item
  await page.getByText(/logout|log out|exit/i).click();

  // Wait for redirect to login page
  await page.waitForURL('**/login', { timeout: 10_000 });
}
