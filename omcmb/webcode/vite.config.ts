import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@core': path.resolve(__dirname, '../frontend-core/src'),
    },
    // 强制 React/ReactDOM 在 npm workspaces 下也只有一份实例。
    // workspace root 与 webcode 子包同时声明 ^19.2.0 时，npm 可能解析到不同 patch
    // (19.2.4 vs 19.2.5)，导致 ReactSharedInternals 多副本 → useCallback 时
    // dispatcher 串味 → 全局白屏。dedupe 让 vite 在打包前先归一。
    dedupe: ['react', 'react-dom', 'scheduler', 'react-router-dom'],
  },
  server: {
    host: '0.0.0.0',
    port: 3000,
    proxy: {
      '/api': {
        // 后端代理目标（通过环境变量配置，冒烟测试等场景可指向其他端口）
        target: env.VITE_API_PROXY_TARGET || 'http://localhost:8081',
        changeOrigin: true,
      },
      // 离线地图元数据代理（必须在 /tiles 前面，与 /tiles 同源）
      '/tiles-metadata': {
        target: env.VITE_TILES_PROXY_TARGET || 'http://localhost:8081',
        changeOrigin: true,
        rewrite: (path) => path,
      },
      // 离线地图瓦片代理（通过环境变量配置目标服务器）
      '/tiles': {
        target: env.VITE_TILES_PROXY_TARGET || 'http://localhost:8081',
        changeOrigin: true,
        timeout: 30000,
        // 不重写路径，保持 /tiles/... 原样转发
        rewrite: (path) => path,
      },
    },
  },
  build: {
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        // 函数式 manualChunks：保证 react/react-dom/scheduler 一定打进同一个
        // vendor-react chunk，避免 React 多副本导致 ReactSharedInternals.H
        // dispatcher 串味（症状：useCallback dispatcher null → 全局白屏）。
        // 之前用对象式声明，rollup 在 antd 间接 import react-dom 时会复制到
        // vendor-antd / entry，造成多份 React 实例。
        manualChunks: (id) => {
          if (!id.includes('node_modules')) return undefined;
          if (
            id.includes('/react/') ||
            id.includes('/react-dom/') ||
            id.includes('/scheduler/') ||
            id.includes('/react-router/') ||
            id.includes('/react-router-dom/')
          ) {
            return 'vendor-react';
          }
          if (id.includes('/antd/') || id.includes('/@ant-design/')) {
            return 'vendor-antd';
          }
          if (id.includes('/echarts/') || id.includes('/echarts-for-react/')) {
            return 'vendor-echarts';
          }
          return undefined;
        },
      },
    },
  },
}
})
