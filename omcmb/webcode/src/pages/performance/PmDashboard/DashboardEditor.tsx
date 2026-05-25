/**
 * 旧路由 `/performance/pm-dashboard/:id` 的入口。
 *
 * G6-Gap-1 后新版用 `PerformanceLayout` 的左右栏 + `?dashboard=:id` query。
 * 本组件保留为向后兼容（旧链接 / 旧收藏），重定向到新路由。
 */

import { Navigate, useParams } from 'react-router-dom';

export default function DashboardEditor() {
  const { id } = useParams<{ id: string }>();
  if (!id) {
    return <Navigate to="/performance" replace />;
  }
  return <Navigate to={`/performance?dashboard=${id}`} replace />;
}
