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
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@core': resolve(__dirname, '../frontend-core/src'),
    },
  },
})
