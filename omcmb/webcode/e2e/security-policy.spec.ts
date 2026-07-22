import { test, expect, type Page, type Route } from '@playwright/test';

// LoginPage placeholder 当前是 i18n('login.usernameTip')='请输入用户名'，
// helpers/auth.ts 的 /username|user/i regex 与之不匹配（pre-existing 问题）。
// 这里 spec 内部直接用 placeholder 文本，spec 自洽。
const USERNAME_PLACEHOLDER = /请输入用户名|username|user/i;
const PASSWORD_PLACEHOLDER = /请输入密码|password/i;

async function localLogin(page: Page): Promise<void> {
  await page.goto('/login');
  await page.getByPlaceholder(USERNAME_PLACEHOLDER).first().fill('admin');
  await page.getByPlaceholder(PASSWORD_PLACEHOLDER).first().fill('admin123');
  await page.locator('button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 15_000 });
}

/**
 * P2-⑦ 屏幕锁定 + P2-⑧ 禁止浏览器记密码 浏览器层 E2E。
 *
 * 这两项纯 FE 行为，不通过后端响应验证，必须在浏览器里跑。后端
 * sys_configs 通过 page.route 拦截 mock，避免：
 *   - 测试依赖真 PG / SecurityPolicy 30s 缓存
 *   - 测试间相互污染（一个 spec 改配置影响下一个）
 *
 * 跑法：
 *   cd omcmb/webcode
 *   npx playwright test e2e/security-policy.spec.ts
 */

const PUBLIC_CONFIGS_URL = /\/admin\/public\/configs.*category=security/;

/**
 * mockPublicSecurityConfigs 拦截 /admin/public/configs?category=security 响应，
 * 注入定制的 4 个公开安全字段。剩余字段后端正常返。
 *
 * 注意：必须在 page.goto 之前调，否则 LoginPage 已经发起请求。
 */
async function mockPublicSecurityConfigs(
  page: Page,
  overrides: {
    userSessionExpirationMin?: number;
    isBrowserAutoRecordPass?: boolean;
    enabledFlag?: boolean;
    msg?: string;
  },
): Promise<void> {
  await page.route(PUBLIC_CONFIGS_URL, async (route: Route) => {
    const items: Array<{ key: string; value: string; value_type: string; is_public: boolean }> = [];
    if (overrides.userSessionExpirationMin !== undefined) {
      items.push({
        key: 'userSessionExpirationMin',
        value: String(overrides.userSessionExpirationMin),
        value_type: 'int',
        is_public: true,
      });
    }
    if (overrides.isBrowserAutoRecordPass !== undefined) {
      items.push({
        key: 'isBrowserAutoRecordPass',
        value: String(overrides.isBrowserAutoRecordPass),
        value_type: 'bool',
        is_public: true,
      });
    }
    if (overrides.enabledFlag !== undefined) {
      items.push({
        key: 'enabledFlag',
        value: String(overrides.enabledFlag),
        value_type: 'bool',
        is_public: true,
      });
    }
    if (overrides.msg !== undefined) {
      items.push({
        key: 'msg',
        value: overrides.msg,
        value_type: 'string',
        is_public: true,
      });
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      // 后端 envelope：{ret:1, msg:'查询成功', data:[...]}
      body: JSON.stringify({ ret: 1, msg: 'ok', data: items }),
    });
  });
}

// 同理 mock 登录后拉的 /admin/sysConfig?category=security（带认证）
async function mockSysConfigsByCategory(
  page: Page,
  overrides: { userSessionExpirationMin?: number },
): Promise<void> {
  await page.route(/\/admin\/sysConfig\b.*category=security/, async (route) => {
    const items: Array<{ key: string; value: string; value_type: string; is_public: boolean }> = [];
    if (overrides.userSessionExpirationMin !== undefined) {
      items.push({
        key: 'userSessionExpirationMin',
        value: String(overrides.userSessionExpirationMin),
        value_type: 'int',
        is_public: false,
      });
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ret: 1, msg: 'ok', data: items }),
    });
  });
}

// ============================================================================
// P2-⑧ Autocomplete switching
// ============================================================================

test.describe('P2-⑧ Browser password-save behavior on login page', () => {
  test('isBrowserAutoRecordPass=true → text input + CSS mask + autocomplete=off', async ({
    page,
  }) => {
    await mockPublicSecurityConfigs(page, { isBrowserAutoRecordPass: true });
    await page.goto('/login');

    // 等 hook 触发一次请求并应用值
    await page.waitForResponse(PUBLIC_CONFIGS_URL);

    const usernameInput = page.getByPlaceholder(USERNAME_PLACEHOLDER).first();
    const passwordInput = page.getByPlaceholder(PASSWORD_PLACEHOLDER).first();

    await expect(usernameInput).toHaveAttribute('autocomplete', 'off');
    await expect(passwordInput).toHaveAttribute('type', 'text');
    await expect(passwordInput).toHaveAttribute('autocomplete', 'off');
    await expect(passwordInput).toHaveCSS('-webkit-text-security', 'disc');
  });

  test('isBrowserAutoRecordPass=false → default autocomplete (username / current-password)', async ({
    page,
  }) => {
    await mockPublicSecurityConfigs(page, { isBrowserAutoRecordPass: false });
    await page.goto('/login');
    await page.waitForResponse(PUBLIC_CONFIGS_URL);

    const usernameInput = page.getByPlaceholder(USERNAME_PLACEHOLDER).first();
    const passwordInput = page.getByPlaceholder(PASSWORD_PLACEHOLDER).first();

    await expect(usernameInput).toHaveAttribute('autocomplete', 'username');
    await expect(passwordInput).toHaveAttribute('type', 'password');
    await expect(passwordInput).toHaveAttribute('autocomplete', 'current-password');
  });
});

// ============================================================================
// P2-⑦ Idle timer (auto-logout after N minutes of inactivity)
// ============================================================================

test.describe('P2-⑦ Idle screen lock after inactivity', () => {
  test.beforeEach(async ({ page }) => {
    // 登录态请求需要 mock — 登录 hook 内拉 useSysConfigsByCategory
    // 也需要返回 idleLockMinutes 值才能驱动 useIdleLogout
    await mockSysConfigsByCategory(page, { userSessionExpirationMin: 1 });
  });

  test('设 idleMinutes=1 + 跳时钟 65s → 显示解锁并恢复锁屏前页面', async ({ page }) => {
    // page.clock 1.45+ — install 后所有 setInterval/Date.now 走假时钟。
    // 必须在 navigation 之前 install。
    await page.clock.install();

    await localLogin(page); // 进 /dashboard
    await expect(page).toHaveURL(/\/dashboard/);
    await page.goto('/system/config?tab=security');
    await expect(page).toHaveURL(/\/system\/config\?tab=security/);

    // useIdleLogout 监听用户事件刷新 lastActivity；
    // 我们不触发任何事件，让 lastActivity 停在 login 完成时刻；
    // 跳时钟 65 秒 → 通过一轮 30s checkInterval 且 >= 1min idleMs，触发登出。
    await page.clock.fastForward('01:05'); // 65 秒（mm:ss）

    // 仍复用认证页路由，但界面必须明确是锁屏/解锁态，而不是普通登录态。
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 });
    await expect(page.getByRole('heading', { name: /屏幕已锁定|Screen Locked/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /解\s*锁|Unlock/i })).toBeVisible();

    await page.getByPlaceholder(PASSWORD_PLACEHOLDER).first().fill('admin123');
    await page.getByRole('button', { name: /解\s*锁|Unlock/i }).click();
    await expect(page).toHaveURL(/\/system\/config\?tab=security/);
  });

  test('设 idleMinutes=0 → idle hook 不挂载，时钟跳过也不登出', async ({ page }) => {
    // 复盖 beforeEach 的 mock：这次返 idleLockMinutes=0
    await page.unroute(/\/admin\/sysConfig\b.*category=security/);
    await mockSysConfigsByCategory(page, { userSessionExpirationMin: 0 });

    await page.clock.install();
    await localLogin(page);
    await expect(page).toHaveURL(/\/dashboard/);

    // 跳很长时间 — 不应自动登出
    await page.clock.fastForward('10:00');

    // 仍在 dashboard
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test('设 idleMinutes=1 + 跳时钟前模拟用户活动 → 不登出', async ({ page }) => {
    await page.clock.install();
    await localLogin(page);

    // 模拟 50s 后用户点了一下（刷活动时间戳）
    await page.clock.fastForward('00:50');
    await page.mouse.click(100, 100);

    // 再跳 50 秒（累计 100s，但活动时间戳已在 50s 时刷新，
    // 距上次活动只过去 50s < 60s idleMs）
    await page.clock.fastForward('00:50');

    // 不应被踢
    await expect(page).toHaveURL(/\/dashboard/);
  });
});
