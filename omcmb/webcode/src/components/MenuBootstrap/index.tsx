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

import { useEffect, type ReactNode } from 'react';
import { Spin } from 'antd';

import { useAppStore } from '@core/store/appStore';
import { useMenuStore } from '@core/store/menuStore';
import { useUserStore } from '@core/store/userStore';
import { useUserMenus } from '@core/hooks/api/useMenus';
import { useSysConfigsByCategory } from '@core/hooks/api/useSystem';

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

// useShowMenuIconBootstrap 把 sys_configs.system.show_menu_icon 同步到 appStore。
// 与 useUserMenus 同生命周期挂载 — 已登录且灰度启用时拉一次，NavMenu 自然消费。
// 用户在「菜单管理」改 Switch 后会调 batchUpdateSysConfigs；React Query 的
// invalidateQueries 会触发本 hook 重拉，从而把改动同步给当前 tab 的 appStore。
function useShowMenuIconBootstrap() {
  const setShowMenuIcon = useAppStore((s) => s.setShowMenuIcon);
  const { data } = useSysConfigsByCategory('system');
  useEffect(() => {
    if (!data) return;
    const item = data.find((c) => c.key === 'show_menu_icon');
    if (!item) return;
    // sys_configs.value 是 'true'/'false' 字符串（value_type='bool'）
    setShowMenuIcon(item.value === 'true');
  }, [data, setShowMenuIcon]);
}

function MenuBootstrapInner({ loaded, children }: { loaded: boolean; children: ReactNode }) {
  // useQuery 仅在已登录 + 灰度启用时挂载；否则不会无谓发请求。
  const { isError } = useUserMenus();
  useShowMenuIconBootstrap();

  // 阻塞条件改为 `!loaded` 单一判定（不再依赖 isLoading）。
  //
  // 历史 bug：之前是 `!loaded && isLoading`。当 React Query 命中前一个用户的缓存时
  // isLoading 立即为 false（同步返回），但 cache hit 不会触发 queryFn 的 setMenus
  // 写入，menuStore.loaded 始终为 false → 此分支不命中，fall through 到 children
  // → NavMenu 看到空 menuStore → 回退到 NAV_CONFIG 静态菜单 → 用户切换后看到"上个用户菜单残留"。
  // 修复后：只要 menuStore 还没加载完成就一律 spin，避免任何窗口期渲染脏 UI。
  if (!loaded && !isError) {
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
