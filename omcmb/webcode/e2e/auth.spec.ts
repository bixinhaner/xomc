import { test, expect } from '@playwright/test';
import { login, logout, submitLoginForm } from './helpers/auth';

test.describe('Authentication flow', () => {
  test('should display the login page with form fields', async ({ page }) => {
    await page.goto('/login');

    // The login page should show the title / branding
    await expect(page.locator('h1')).toBeVisible();

    // Username and password fields should be present
    await expect(page.getByPlaceholder(/username|user/i).first()).toBeVisible();
    await expect(page.getByPlaceholder(/password/i).first()).toBeVisible();

    // Submit button should be present
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should reject invalid credentials', async ({ page }) => {
    await page.goto('/login');

    await page.getByPlaceholder(/username|user/i).first().fill('admin');
    await page.getByPlaceholder(/password/i).first().fill('wrong-password');
    await page.locator('button[type="submit"]').click();

    // Should remain on the login page
    await expect(page).toHaveURL(/\/login/);

    // An error message should appear (Ant Design message component)
    await expect(page.locator('.ant-message')).toBeVisible({ timeout: 5_000 });
  });

  test('should login successfully with valid credentials', async ({ page }) => {
    await login(page);

    // Should redirect to dashboard
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test('should redirect unauthenticated users to login', async ({ page }) => {
    // Try to access a protected page without logging in
    await page.goto('/device/list');

    // Should be redirected to login
    await expect(page).toHaveURL(/\/login/);
  });

  test('should logout and redirect to login page', async ({ page }) => {
    // First login
    await login(page);
    await expect(page).toHaveURL(/\/dashboard/);

    // Then logout
    await logout(page);
    await expect(page).toHaveURL(/\/login/);
  });

  test('should open dashboard after logging in again from a manual logout', async ({ page }) => {
    await login(page);
    await page.goto('/device/list');
    await expect(page).toHaveURL(/\/device\/list/);

    await logout(page);
    await submitLoginForm(page);

    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.getByRole('tab', { name: /仪表板|Dashboard/i })).toBeVisible();
    await expect(page.getByText(/总设备数|Total Devices/).first()).toBeVisible();
    await expect(
      page.getByPlaceholder(/SN \/ 名称 \/ IP \/ MAC \/ PCI|SN.*Name.*IP.*MAC.*PCI/i),
    ).toHaveCount(0);
  });
});
