// MenuBootstrap — 菜单动态加载 P1 启动屏障。
//
// 在已登录的私有路由树外层包一层：
//   - VITE_DYNAMIC_MENU=false：直通；保留 NAV_CONFIG 静态行为
//   - VITE_DYNAMIC_MENU=true：触发 useUserMenus；首次拉取且无 persist cache 时阻塞首屏
//
// 阻塞策略：首次冷启动（loaded=false 且查询 isLoading）→ FullScreenSpin
// 命中 persist 缓存（loaded=true）→ 直通；后台 stale-while-revalidate 更新菜单。
//
// 设计依据：docs/prd/system/menu-dynamic-loading.md §4.3.2 (App 启动流程改造)。

import type { ReactNode } from 'react';
import { Spin } from 'antd';

import { useMenuStore } from '@core/store/menuStore';
import { useUserStore } from '@core/store/userStore';
import { useUserMenus } from '@core/hooks/api/useMenus';

import { isDynamicMenuEnabled } from './featureFlag';

interface MenuBootstrapProps {
  children: ReactNode;
}

export default function MenuBootstrap({ children }: MenuBootstrapProps) {
  const isAuthenticated = useUserStore((s) => s.isAuthenticated);
  const loaded = useMenuStore((s) => s.loaded);

  // 灰度未开启 → 完全透传，保留 NAV_CONFIG 行为。
  if (!isDynamicMenuEnabled()) {
    return <>{children}</>;
  }

  // 未登录 → PrivateRoute 自身会跳 /login，不在此抢占。
  if (!isAuthenticated) {
    return <>{children}</>;
  }

  return <MenuBootstrapInner loaded={loaded}>{children}</MenuBootstrapInner>;
}

function MenuBootstrapInner({ loaded, children }: { loaded: boolean; children: ReactNode }) {
  // useQuery 仅在已登录 + 灰度启用时挂载；否则不会无谓发请求。
  const { isLoading, isError } = useUserMenus();

  if (!loaded && isLoading) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: '100vw',
          height: '100vh',
        }}
      >
        <Spin size="large" tip="加载菜单..." />
      </div>
    );
  }

  if (!loaded && isError) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: '100vw',
          height: '100vh',
          flexDirection: 'column',
          gap: 12,
        }}
      >
        <div>菜单加载失败，请刷新或重新登录。</div>
      </div>
    );
  }

  return <>{children}</>;
}
