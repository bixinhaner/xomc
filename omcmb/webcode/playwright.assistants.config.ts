import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './tests/assistant-preview', testMatch: '*.spec.ts', timeout: 60000,
  expect: { timeout: 15000 }, workers: 1, retries: 0,
  reporter: [['list'], ['html', { outputFolder: 'assistant-browser-report', open: 'never' }]],
  use: { baseURL: 'http://127.0.0.1:3000', browserName: 'chromium', screenshot: 'only-on-failure', trace: 'retain-on-failure' },
  webServer: { command: 'npm run dev -- --host 127.0.0.1', url: 'http://127.0.0.1:3000/tests/assistant-preview/index.html', reuseExistingServer: !process.env.CI, timeout: 120000 },
});
