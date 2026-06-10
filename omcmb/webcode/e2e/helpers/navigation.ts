import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';

/**
 * Navigate to a page within the authenticated shell and wait for it to load.
 * Assumes the user is already logged in.
 */
export async function navigateTo(page: Page, path: string): Promise<void> {
  await page.goto(path);
  // Wait for the main content area to become visible (layout has rendered)
  await page.waitForLoadState('domcontentloaded');
}

/**
 * Assert the current URL path matches the expected value.
 */
export async function expectPath(page: Page, expectedPath: string): Promise<void> {
  await expect(page).toHaveURL(new RegExp(`${expectedPath}$`));
}

/**
 * Wait for a page to finish its lazy-load Suspense boundary.
 * This waits for the Ant Design Spin component to disappear.
 */
export async function waitForPageLoad(page: Page, timeout = 15_000): Promise<void> {
  // Wait for any Ant Design Spin loaders to disappear.
  // Note: a page may show several spinners at once (e.g. dashboard KPI panels),
  // so use first()/toHaveCount(0) instead of strict-mode single-element waits.
  const spinner = page.locator('.ant-spin-spinning');
  if (await spinner.first().isVisible({ timeout: 2_000 }).catch(() => false)) {
    await expect(spinner).toHaveCount(0, { timeout });
  }
}
