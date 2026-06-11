import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright 冒烟测试配置 —— 打「真实后端」的按业务域 smoke spec。
 *
 * 与 playwright.config.ts（纯 Mock 深度用例）的区别：
 *   - testDir 独立为 ./e2e/smoke，spec 统一打 @smoke tag；
 *   - webServer 用 `vite --mode test` 起 :3010（VITE_USE_MOCK=false，真实 API），
 *     /api 经 vite 代理转发到 VITE_API_PROXY_TARGET 指定的后端；
 *   - 真实后端响应较慢，test/expect 超时放宽到 45s/15s。
 *
 * 用法（在 omcmb/webcode/ 下）：
 *   npm run test:smoke                                        # 后端默认 :8081
 *   SMOKE_API_TARGET=http://localhost:18091 npm run test:smoke # 指定后端
 *   E2E_USERNAME=xxx E2E_PASSWORD=yyy 可覆盖登录凭据（默认 admin/admin123）
 */
export default defineConfig({
  testDir: './e2e/smoke',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { open: 'never' }],
    ['list'],
  ],
  timeout: 45_000,
  expect: {
    timeout: 15_000,
  },
  use: {
    baseURL: 'http://localhost:3010',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  /* 起一个连真实后端的 Vite dev server（:3010，避免与 :3000 mock 栈冲突） */
  webServer: {
    command: 'npx vite --mode test --port 3010',
    url: 'http://localhost:3010',
    reuseExistingServer: true,
    timeout: 90_000,
    env: {
      VITE_USE_MOCK: 'false',
      VITE_API_PROXY_TARGET: process.env.SMOKE_API_TARGET || 'http://localhost:8081',
    },
  },
});
