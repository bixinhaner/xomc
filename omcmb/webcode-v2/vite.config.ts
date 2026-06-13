import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  // 三套皮肤同一 nginx 下按子路径分发：v1=/ · v2=/v2/ · v3=/v3/。
  // base 决定产物 asset 前缀与 import.meta.env.BASE_URL（路由 basename 据此取值）。
  base: '/v2/',
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@core': path.resolve(__dirname, '../frontend-core/src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 3002,
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
  define: {
    __APP_VERSION__: JSON.stringify('2.0.0'),
  },
})
