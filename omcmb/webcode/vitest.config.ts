import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    css: true,
    include: [
      'src/**/*.{test,spec}.{ts,tsx}',
      '../frontend-core/src/**/*.{test,spec}.{ts,tsx}',
    ],
    env: {
      VITE_USE_MOCK: 'false',
      VITE_API_BASE_URL: '/api/v1',
    },
    coverage: {
      provider: 'v8',
      // 核心业务层覆盖：utils / hooks / providers / router / theme。
      // pages/ 与 components/ 大量是 antd pro UI shell，集成测试覆盖更合理（W2.C.3 范围之外）。
      include: [
        'src/utils/**/*.{ts,tsx}',
        'src/hooks/**/*.{ts,tsx}',
        'src/providers/**/*.{ts,tsx}',
        'src/router/**/*.{ts,tsx}',
        'src/theme/**/*.{ts,tsx}',
      ],
      exclude: [
        'src/**/*.{test,spec}.{ts,tsx}',
        'src/test/**',
        'src/main.tsx',
        'src/env.d.ts',
        'src/**/*.d.ts',
      ],
    },
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@core': resolve(__dirname, '../frontend-core/src'),
      // 让 frontend-core 测试也能解析依赖（webcode 是顶层 node_modules）
      zustand: resolve(__dirname, './node_modules/zustand'),
      axios: resolve(__dirname, './node_modules/axios'),
    },
  },
})
