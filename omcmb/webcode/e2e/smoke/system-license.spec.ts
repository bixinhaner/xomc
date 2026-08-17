import { test, expect } from '@playwright/test';
import { smokeLogin } from './helpers';

/**
 * System License 页面冒烟（真实后端）——issue #319 回归。
 *
 * Basic Info 精简为单行 4 字段（License ID / License Type / 到期日期 / 上传时间）：
 *   - Issuer / Licensee：legacy TrueLicense claims 不产出，页面上恒为空，已移除；
 *   - 签发日期：厂商 TrueLicense 签发工具把 issued 写死为 2019-01-01（占位值
 *     而非真实签发时间，全量真实 license 语料均如此），已移除；
 *   - 状态 / 签名状态：与到期日期信息冗余，已移除。
 *
 * 断言直接取 Basic Info Descriptions 的标签清单做精确匹配（"状态"是通用词，
 * 页面级 getByText 断言会误伤其他卡片），i18n 默认 zh-CN，正则兼容 en-US。
 */
test.describe('System License 冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/license Basic Info 单行 4 字段，符合 issue #319 精简', async ({ page }) => {
    await page.goto('/license');
    const basicInfoCard = page.locator('.ant-card', { hasText: /Basic Info/ }).first();
    await expect(basicInfoCard).toBeVisible({ timeout: 15_000 });

    const labels = basicInfoCard.locator('.ant-descriptions-item-label');
    await expect(labels).toHaveCount(4);
    await expect(labels.nth(0)).toHaveText(/License ID/);
    await expect(labels.nth(1)).toHaveText(/License Type/);
    await expect(labels.nth(2)).toHaveText(/到期日期|Expiry Date/);
    await expect(labels.nth(3)).toHaveText(/上传时间|Uploaded At/);

    // 单行布局：4 个字段同一行（tr 只有一行）
    await expect(basicInfoCard.locator('.ant-descriptions-view tr')).toHaveCount(1);
  });
});
