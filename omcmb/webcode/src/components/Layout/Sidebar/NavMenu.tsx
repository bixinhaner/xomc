import { useMemo } from 'react';
import { Menu as AntMenu } from 'antd';
import type { MenuProps } from 'antd';
import { useIntl } from 'react-intl';
import {
  AppstoreOutlined,
  AppstoreAddOutlined,
  DashboardOutlined,
  ClusterOutlined,
  AlertOutlined,
  SettingOutlined,
  LineChartOutlined,
  CodeOutlined,
  GlobalOutlined,
  SaveOutlined,
  CloudUploadOutlined,
  FolderOutlined,
  FileTextOutlined,
  ToolOutlined,
  BarChartOutlined,
  RadarChartOutlined,
  SafetyOutlined,
  GatewayOutlined,
  DeploymentUnitOutlined,
  WifiOutlined,
  ThunderboltOutlined,
  ApartmentOutlined,
  CloudServerOutlined,
  AimOutlined,
  ExperimentOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { useTabStore } from '@core/store/tabStore';
import { useAppStore } from '@core/store/appStore';
import { useMenuStore } from '@core/store/menuStore';
import { useUserStore } from '@core/store/userStore';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';
import {
  extractLicenseErrorCode,
  SystemLicenseErrorCodes,
} from '@core/services/api/systemLicenseApi';
import { isSystemLicensePath } from '@core/utils/systemLicenseAccess';
import type { Menu as DynamicMenu } from '@core/types/menu';
import { resolveMenuLabel } from '@core/types/menu';
import { useResponsive } from '@/hooks/useResponsive';
import { useT } from '@/hooks/useT';
import { resolveIcon } from '@/components/IconPicker/icons';
import { isDynamicMenuEnabled } from '@/components/MenuBootstrap/featureFlag';
import { NAV_CONFIG } from './navConfig';
import type { NavGroup, NavChild } from './navConfig';

type MenuItem = Required<MenuProps>['items'][number];

// antd Menu 的 key 在整棵树内必须唯一。NAV_CONFIG 中允许目录与其唯一子项
// 共享业务 key（例如 dashboard、license），渲染时为目录加命名空间，避免
// React/antd 报 duplicated key，同时不改变路由项的 key 和选中态映射。
function staticGroupMenuKey(groupKey: string): string {
  return `group:${groupKey}`;
}

// 静态模式（VITE_DYNAMIC_MENU=false）兜底图标映射，仅给 NAV_CONFIG iconName 用。
// 动态模式优先调 resolveIcon（IconPicker 116 个白名单）。
// AppstoreAddOutlined 来自 T-0098-P4-02 产品中心目录（origin/main）。
const STATIC_ICON_MAP: Record<string, React.ReactNode> = {
  DashboardOutlined: <DashboardOutlined />,
  ClusterOutlined: <ClusterOutlined />,
  AlertOutlined: <AlertOutlined />,
  SettingOutlined: <SettingOutlined />,
  LineChartOutlined: <LineChartOutlined />,
  CodeOutlined: <CodeOutlined />,
  GlobalOutlined: <GlobalOutlined />,
  SaveOutlined: <SaveOutlined />,
  CloudUploadOutlined: <CloudUploadOutlined />,
  FolderOutlined: <FolderOutlined />,
  FileTextOutlined: <FileTextOutlined />,
  ToolOutlined: <ToolOutlined />,
  BarChartOutlined: <BarChartOutlined />,
  RadarChartOutlined: <RadarChartOutlined />,
  SafetyOutlined: <SafetyOutlined />,
  AppstoreOutlined: <AppstoreOutlined />,
  AppstoreAddOutlined: <AppstoreAddOutlined />,
  GatewayOutlined: <GatewayOutlined />,
  DeploymentUnitOutlined: <DeploymentUnitOutlined />,
  WifiOutlined: <WifiOutlined />,
  ThunderboltOutlined: <ThunderboltOutlined />,
  ApartmentOutlined: <ApartmentOutlined />,
  CloudServerOutlined: <CloudServerOutlined />,
  AimOutlined: <AimOutlined />,
  ExperimentOutlined: <ExperimentOutlined />,
};

// ---------------------------------------------------------------------------
// 静态分支（保留兜底）：消费 NAV_CONFIG（已按 super_admin 过滤）
// ---------------------------------------------------------------------------

function buildStaticMenuItems(groups: NavGroup[], t: (id: string) => string): MenuItem[] {
  if (!groups) return [];
  return groups.map((group) => {
    const children = group.children || [];
    // 单子节点也保持为可展开子菜单（不扁平化），与动态菜单行为一致。
    return {
      type: 'submenu' as const,
      key: staticGroupMenuKey(group.key),
      icon: STATIC_ICON_MAP[group.iconName],
      label: t(group.label),
      children: children.map(
        (child: NavChild): MenuItem => ({
          type: 'item' as const,
          key: child.key,
          label: t(child.label),
        }),
      ),
    };
  });
}

function buildStaticKeyToChild(groups: NavGroup[]): Map<string, NavChild> {
  const map = new Map<string, NavChild>();
  if (!groups) return map;
  for (const group of groups) {
    const children = group.children || [];
    for (const child of children) {
      map.set(child.key, child);
    }
  }
  return map;
}

function buildStaticPathToKey(groups: NavGroup[]): Map<string, string> {
  const map = new Map<string, string>();
  if (!groups) return map;
  for (const group of groups) {
    const children = group.children || [];
    for (const child of children) {
      map.set(child.path, child.key);
    }
  }
  return map;
}

// ---------------------------------------------------------------------------
// 动态分支：消费 menuStore
// ---------------------------------------------------------------------------

interface DynamicLeaf {
  key: string;
  label: string;
  path: string;
}

function isVisible(menu: DynamicMenu): boolean {
  return menu.status === 'normal' && menu.showStatus !== 'hide';
}

function keepSystemLicenseMenu(menu: DynamicMenu): boolean {
  return isSystemLicensePath(menu.routePath ?? '')
    || (menu.children ?? []).some(keepSystemLicenseMenu);
}

function filterSystemLicenseMenus(menus: DynamicMenu[]): DynamicMenu[] {
  return menus
    .filter(keepSystemLicenseMenu)
    .map((menu) => ({
      ...menu,
      children: menu.children ? filterSystemLicenseMenus(menu.children) : menu.children,
    }));
}

function renderIcon(name: string | undefined): React.ReactNode {
  const Component = resolveIcon(name);
  return Component ? <Component /> : <AppstoreOutlined />;
}

/**
 * MenuLabelResolver 把 menu 解析为当前 locale 下的显示文案。
 * 用 type alias 避免在所有 helper 签名里重复 `(m) => string` 的 verbose 形式。
 */
type MenuLabelResolver = (menu: DynamicMenu) => string;

/**
 * 把后端菜单树转 antd Menu items。规则：
 *  - 仅渲染 type='directory'|'menu'（按钮跳过）
 *  - 目录即使只有一个子节点也保持为可展开子菜单（不扁平化），保证子菜单可见
 *  - 隐藏 status!=active 或 showStatus='hide' 的节点
 *  - label 走 resolveMenuLabel：nameI18n[locale] > nameI18n['zh-CN'] > name
 *  - icon 仅当 showIcon=true 时渲染（sys_configs.system.show_menu_icon 全局开关）
 *
 * 注：动态模式下，super_admin 过滤由后端 GetUserMenuTreeByRole / GetAllActive
 * 在 service 层完成（user.source='builtIn' 旁路），前端无需再过滤。
 */
function buildDynamicMenuItems(
  menus: DynamicMenu[],
  label: MenuLabelResolver,
  showIcon: boolean,
): MenuItem[] {
  const icon = (m: DynamicMenu) => (showIcon ? renderIcon(m.icon) : undefined);
  return menus
    .filter(isVisible)
    .filter((m) => m.type !== 'button')
    .map((m) => {
      const visibleChildren = (m.children ?? [])
        .filter(isVisible)
        .filter((c) => c.type !== 'button');

      // 叶子菜单
      if (m.type === 'menu' || visibleChildren.length === 0) {
        return {
          type: 'item' as const,
          key: m.routePath || m.id,
          icon: icon(m),
          label: label(m),
        };
      }

      // 目录：哪怕只有一个子节点，也保持为可展开的子菜单并显示该子节点
      // （不再扁平化成父级直跳——否则「运维管理」这种单子目录会直接跳到
      //  「TR069 报文跟踪」页且子菜单不显示）。
      return {
        type: 'submenu' as const,
        key: m.id,
        icon: icon(m),
        label: label(m),
        children: buildDynamicMenuItems(visibleChildren, label, showIcon),
      };
    });
}

/** 扁平索引：menu key → DynamicLeaf（path 跳转用）。leaf.label 同样走 label resolver。 */
function buildDynamicKeyToLeaf(menus: DynamicMenu[], label: MenuLabelResolver): Map<string, DynamicLeaf> {
  const map = new Map<string, DynamicLeaf>();
  const walk = (list: DynamicMenu[]) => {
    if (!list) return;
    for (const m of list.filter(isVisible)) {
      if (m.type === 'menu' && m.routePath) {
        map.set(m.routePath, { key: m.routePath, label: label(m), path: m.routePath });
      }
      // 子节点（含单子目录的唯一子节点）由递归 walk 统一映射，无需扁平化特例。
      if (m.children?.length) walk(m.children);
    }
  };
  if (!menus) return map;
  walk(menus);
  return map;
}

/** 找到包含当前 path 的顶级目录 ID，作为 antd Menu defaultOpenKeys。 */
function findDynamicTopOpenKey(menus: DynamicMenu[], pathname: string): string[] {
  if (!menus) return [];
  const matchInSubtree = (list: DynamicMenu[]): boolean => {
    for (const m of list) {
      if (m.routePath === pathname) return true;
      if (m.children?.length && matchInSubtree(m.children)) return true;
    }
    return false;
  };
  for (const top of menus) {
    if (top.children?.length && matchInSubtree(top.children)) {
      return [top.id];
    }
  }
  return [];
}

/** path → key（用于 selectedKeys）；动态模式 key === path，故映射 1:1。 */
function buildDynamicPathToKey(menus: DynamicMenu[]): Map<string, string> {
  const map = new Map<string, string>();
  const walk = (list: DynamicMenu[]) => {
    if (!list) return;
    for (const m of list.filter(isVisible)) {
      if (m.routePath) map.set(m.routePath, m.routePath);
      if (m.children?.length) walk(m.children);
    }
  };
  if (!menus) return map;
  walk(menus);
  return map;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function NavMenu({
  collapsed,
  position = 'left',
}: {
  collapsed?: boolean;
  position?: 'left' | 'right' | 'top';
}) {
  const navigate = useNavigate();
  const location = useLocation();
  const openTab = useTabStore((s) => s.openTab);
  const appTheme = useAppStore((s) => s.theme);
  const setMobileOverlayOpen = useAppStore((s) => s.setMobileOverlayOpen);
  const { isMobile } = useResponsive();
  const t = useT();
  const intl = useIntl();
  const currentUser = useUserStore((s) => s.currentUser);
  const isSuperAdmin = useUserStore((s) => s.currentUser?.isSuperAdmin === true);
  const isAdmin = currentUser?.role === 'admin' || isSuperAdmin;
  // sys_configs.system.show_menu_icon → appStore.showMenuIcon（MenuBootstrap 启动期同步）。
  // 关掉后整个动态菜单不渲染图标，运维在「菜单管理」页顶部 Switch 改即时生效。
  const showMenuIcon = useAppStore((s) => s.showMenuIcon);

  const dynamicMenus = useMenuStore((s) => s.menus);
  const { data: currentLicense, isLoading: isLicenseLoading, error: licenseError } = useSystemLicense();
  // licenseOnlyMode：无 license（404/12113）OR license 已过期（is_expired），
  // 与 PrivateRoute 保持同口径（issue #310）。
  const licenseOnlyMode = !isLicenseLoading
    && (currentLicense?.isExpired === true
      || (!currentLicense
        && extractLicenseErrorCode(licenseError) === SystemLicenseErrorCodes.NotConfigured));
  const visibleDynamicMenus = useMemo(
    () => (licenseOnlyMode ? filterSystemLicenseMenus(dynamicMenus || []) : dynamicMenus || []),
    [dynamicMenus, licenseOnlyMode],
  );

  // 动态菜单 label 解析器：依赖 intl.locale，切换语言后 antd Menu 立即重渲染
  // （同时影响 openTab 标题）。译文存 DB（menus.name_i18n），不再依赖前端 i18n 包。
  const labelResolver = useMemo<MenuLabelResolver>(
    () => (m: DynamicMenu) => resolveMenuLabel(m, intl.locale),
    [intl.locale],
  );

  // 灰度判定：仅依赖 env flag。动态模式启用后永远走动态分支，**任何时候**不回退到
  // NAV_CONFIG 静态菜单。
  //
  // 历史回归：
  //   - v1：useDynamic = enabled && loaded && menus.length > 0
  //         → 角色无菜单绑定 / 切换用户瞬态时回退静态 → 闪现 NAV_CONFIG（10 项）
  //   - v2：useDynamic = enabled && loaded
  //         → 仍有窗口期：clearAuth 把 loaded 置 false，但 NavMenu 重渲染发生在
  //           navigate('/login') 完成之前，使用 menuLoaded=false → 还是回退静态 → 闪现
  //   - v3（本版）：useDynamic = enabled
  //         → 切换用户时 menuStore 暂时空 → buildDynamicMenuItems([]) 渲染空侧边栏；
  //           loading 期间由 MenuBootstrap 全屏 spin 兜底；不会再有静态闪现
  const useDynamic = isDynamicMenuEnabled();

  // T-0098-P4-02：静态分支 NAV_CONFIG 按 super_admin 过滤（产品中心仅超管可见）。
  // 动态分支由后端 service 层完成同等过滤，无需前端二次处理。
  const filteredNav = useMemo(
    () => (NAV_CONFIG || [])
      .filter((group) => !licenseOnlyMode || group.key === 'license')
      .filter((g) => !g.requireSuperAdmin || isSuperAdmin)
      .map((group) => ({
        ...group,
        children: (group.children || []).filter((child) => !child.requireAdmin || isAdmin),
      }))
      .filter((group) => (group.children || []).length > 0),
    [isAdmin, isSuperAdmin, licenseOnlyMode],
  );

  const menuItems = useMemo(
    () =>
      useDynamic
        ? buildDynamicMenuItems(visibleDynamicMenus, labelResolver, showMenuIcon)
        : buildStaticMenuItems(filteredNav || [], t),
    [useDynamic, visibleDynamicMenus, labelResolver, showMenuIcon, filteredNav, t],
  );

  const dynamicKeyToLeaf = useMemo(
    () =>
      useDynamic
        ? buildDynamicKeyToLeaf(visibleDynamicMenus, labelResolver)
        : new Map<string, DynamicLeaf>(),
    [useDynamic, visibleDynamicMenus, labelResolver],
  );
  const dynamicPathToKey = useMemo(
    () => (useDynamic ? buildDynamicPathToKey(visibleDynamicMenus) : new Map<string, string>()),
    [useDynamic, visibleDynamicMenus],
  );
  const staticKeyToChild = useMemo(
    () => (useDynamic ? new Map<string, NavChild>() : buildStaticKeyToChild(filteredNav || [])),
    [useDynamic, filteredNav],
  );
  const staticPathToKey = useMemo(
    () => (useDynamic ? new Map<string, string>() : buildStaticPathToKey(filteredNav || [])),
    [useDynamic, filteredNav],
  );

  const selectedKeys = useMemo(() => {
    const map = useDynamic ? dynamicPathToKey : staticPathToKey;
    const key = map.get(location.pathname);
    return key ? [key] : [];
  }, [useDynamic, dynamicPathToKey, staticPathToKey, location.pathname]);

  const defaultOpenKeys = useMemo(() => {
    if (useDynamic) {
      return findDynamicTopOpenKey(visibleDynamicMenus, location.pathname);
    }
    const selectedKey = staticPathToKey.get(location.pathname);
    if (!selectedKey) return [];
    const group = (filteredNav || []).find((g) => (g.children || [])?.some((c) => c.key === selectedKey));
    return group ? [staticGroupMenuKey(group.key)] : [];
  }, [useDynamic, visibleDynamicMenus, staticPathToKey, filteredNav, location.pathname]);

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (useDynamic) {
      const leaf = dynamicKeyToLeaf.get(key);
      if (!leaf) return;
      openTab({ key: leaf.key, label: leaf.label, path: leaf.path, closable: true, labelRaw: true });
      void navigate(leaf.path);
    } else {
      const child = staticKeyToChild.get(key);
      if (!child) return;
      openTab({ key: child.key, label: child.label, path: child.path, closable: true, labelRaw: false });
      void navigate(child.path);
    }
    if (isMobile) setMobileOverlayOpen(false);
  };

  const isHorizontal = position === 'top';

  return (
    <AntMenu
      mode={isHorizontal ? 'horizontal' : 'inline'}
      theme={appTheme === 'fresh' ? 'light' : 'dark'}
      inlineIndent={24}
      inlineCollapsed={isHorizontal ? undefined : collapsed}
      items={menuItems}
      selectedKeys={selectedKeys}
      defaultOpenKeys={isHorizontal ? undefined : defaultOpenKeys}
      onClick={handleMenuClick}
      style={
        isHorizontal
          ? { border: 'none', flex: 1, height: '100%' }
          : { border: 'none', flex: 1, overflowY: 'auto', overflowX: 'hidden' }
      }
    />
  );
}
