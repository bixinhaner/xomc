import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@core': path.resolve(__dirname, '../frontend-core/src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 3003,
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
  define: {
    __APP_VERSION__: JSON.stringify('3.0.0-starforge'),
  },
})
