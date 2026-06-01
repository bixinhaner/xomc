import { Spin } from 'antd';

/**
 * 路由懒加载 Suspense fallback。
 * 从 routes.tsx 拆出（react-refresh/only-export-components：
 * routes.tsx 导出路由配置而非组件，组件单独成文件）。
 */
export const PageLoader = () => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', minHeight: 300 }}>
    <Spin size="large" />
  </div>
);
