import { useMemo } from 'react';
import { Menu } from 'antd';
import type { MenuProps } from 'antd';
import {
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
  AppstoreOutlined,
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
import { useTabStore } from '@/store/tabStore';
import { useAppStore } from '@/store/appStore';
import { useResponsive } from '@/hooks/useResponsive';
import { useT } from '@/hooks/useT';
import { NAV_CONFIG } from './navConfig';
import type { NavGroup, NavChild } from './navConfig';

type MenuItem = Required<MenuProps>['items'][number];

const ICON_MAP: Record<string, React.ReactNode> = {
  DashboardOutlined:    <DashboardOutlined />,
  ClusterOutlined:      <ClusterOutlined />,
  AlertOutlined:        <AlertOutlined />,
  SettingOutlined:      <SettingOutlined />,
  LineChartOutlined:    <LineChartOutlined />,
  CodeOutlined:         <CodeOutlined />,
  GlobalOutlined:       <GlobalOutlined />,
  SaveOutlined:         <SaveOutlined />,
  CloudUploadOutlined:  <CloudUploadOutlined />,
  FolderOutlined:       <FolderOutlined />,
  FileTextOutlined:     <FileTextOutlined />,
  ToolOutlined:         <ToolOutlined />,
  BarChartOutlined:     <BarChartOutlined />,
  RadarChartOutlined:   <RadarChartOutlined />,
  SafetyOutlined:         <SafetyOutlined />,
  AppstoreOutlined:       <AppstoreOutlined />,
  GatewayOutlined:        <GatewayOutlined />,
  DeploymentUnitOutlined: <DeploymentUnitOutlined />,
  WifiOutlined:           <WifiOutlined />,
  ThunderboltOutlined:    <ThunderboltOutlined />,
  ApartmentOutlined:      <ApartmentOutlined />,
  CloudServerOutlined:    <CloudServerOutlined />,
  AimOutlined:            <AimOutlined />,
  ExperimentOutlined:     <ExperimentOutlined />,
};

function buildMenuItems(groups: NavGroup[], t: (id: string) => string): MenuItem[] {
  return groups.map((group) => {
    // Groups with a single child that is the "main" page get rendered as a direct item
    if (group.children.length === 1) {
      const child = group.children[0];
      return {
        key: child.key,
        icon: ICON_MAP[group.iconName],
        label: t(group.label),
      } as MenuItem;
    }

    return {
      key: group.key,
      icon: ICON_MAP[group.iconName],
      label: t(group.label),
      children: group.children.map(
        (child: NavChild): MenuItem => ({
          key: child.key,
          label: t(child.label),
        }),
      ),
    } as MenuItem;
  });
}

// Flat map: menu item key → NavChild (for looking up path on click)
function buildKeyToChild(groups: NavGroup[]): Map<string, NavChild> {
  const map = new Map<string, NavChild>();
  for (const group of groups) {
    for (const child of group.children) {
      map.set(child.key, child);
    }
    // Also map single-child groups directly to their child
    if (group.children.length === 1) {
      map.set(group.key, group.children[0]);
    }
  }
  return map;
}

// Flat map: path → menu key (for selected keys from URL)
function buildPathToKey(groups: NavGroup[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const group of groups) {
    for (const child of group.children) {
      map.set(child.path, child.key);
    }
  }
  return map;
}

export default function NavMenu({ collapsed, position = 'left' }: { collapsed?: boolean; position?: 'left' | 'right' | 'top' }) {
  const navigate = useNavigate();
  const location = useLocation();
  const openTab = useTabStore((s) => s.openTab);
  const appTheme = useAppStore((s) => s.theme);
  const setMobileOverlayOpen = useAppStore((s) => s.setMobileOverlayOpen);
  const { isMobile } = useResponsive();
  const t = useT();

  const menuItems = useMemo(() => buildMenuItems(NAV_CONFIG, t), [t]);
  const keyToChild = useMemo(() => buildKeyToChild(NAV_CONFIG), []);
  const pathToKey = useMemo(() => buildPathToKey(NAV_CONFIG), []);

  // Derive selected keys from current pathname
  const selectedKeys = useMemo(() => {
    const key = pathToKey.get(location.pathname);
    return key ? [key] : [];
  }, [location.pathname, pathToKey]);

  // Derive open (expanded) submenu keys from current pathname
  const defaultOpenKeys = useMemo(() => {
    const selectedKey = pathToKey.get(location.pathname);
    if (!selectedKey) return [];
    const group = NAV_CONFIG.find((g) => g.children.some((c) => c.key === selectedKey));
    return group ? [group.key] : [];
  }, [location.pathname, pathToKey]);

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    const child = keyToChild.get(key);
    if (!child) return;
    openTab({ key: child.key, label: child.label, path: child.path, closable: true });
    void navigate(child.path);
    // Close mobile drawer after navigation
    if (isMobile) setMobileOverlayOpen(false);
  };

  const isHorizontal = position === 'top';

  return (
    <Menu
      mode={isHorizontal ? 'horizontal' : 'inline'}
      theme={appTheme === 'fresh' ? 'light' : 'dark'}
      inlineCollapsed={isHorizontal ? undefined : collapsed}
      items={menuItems}
      selectedKeys={selectedKeys}
      defaultOpenKeys={isHorizontal ? undefined : defaultOpenKeys}
      onClick={handleMenuClick}
      style={isHorizontal
        ? { border: 'none', flex: 1, height: '100%' }
        : { border: 'none', flex: 1, overflowY: 'auto', overflowX: 'hidden' }
      }
    />
  );
}
