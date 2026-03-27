/**
 * GIS 地图视图页面
 * 完全按照 UI 原型图 GISMap_UI_Design_Main.svg 实现
 */
import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import { Checkbox } from 'antd';
import { SearchOutlined, PlusOutlined, MinusOutlined } from '@ant-design/icons';
import GISMap from '@/components/GISMap';
import type { MapDevice, DeviceGroupNode } from '@/types/map';
import type { Site } from '@/types/topology';
import { useSites } from '@/hooks/api/useTopology';
import { useThemeToken } from '@/hooks/useThemeToken';

// 导入大量 Mock 数据（约 3720 条设备）
import { mockMapDevices, mockStats } from '@/components/GISMap/mockDeviceData';

// 是否使用大量 Mock 数据进行测试
const USE_LARGE_MOCK_DATA = true;

// 扩展 Site 类型，增加激活状态
interface ExtendedSite extends Site {
  activated: boolean; // true: 已激活, false: 未激活
}

// ============ Mock Data Generator ============

// 城市坐标数据
const CITY_COORDS: Record<string, { lng: number; lat: number; groupId: string }> = {
  // 北京
  'bj-cy': { lng: 116.46, lat: 39.92, groupId: 'bj-cy' },
  'bj-hd': { lng: 116.31, lat: 39.98, groupId: 'bj-hd' },
  'bj-dc': { lng: 116.41, lat: 39.93, groupId: 'bj-dc' },
  'bj-ft': { lng: 116.28, lat: 39.85, groupId: 'bj-ft' },
  // 上海
  'sh-pd': { lng: 121.60, lat: 31.21, groupId: 'sh-pd' },
  'sh-ja': { lng: 121.45, lat: 31.23, groupId: 'sh-ja' },
  'sh-xh': { lng: 121.48, lat: 31.19, groupId: 'sh-xh' },
  'sh-mh': { lng: 121.38, lat: 31.12, groupId: 'sh-mh' },
  // 广州
  'gz-th': { lng: 113.33, lat: 23.12, groupId: 'gz-th' },
  'gz-by': { lng: 113.27, lat: 23.18, groupId: 'gz-by' },
  'gz-hz': { lng: 113.35, lat: 23.09, groupId: 'gz-hz' },
  // 深圳
  'sz-ns': { lng: 113.93, lat: 22.53, groupId: 'sz-ns' },
  'sz-ft': { lng: 114.05, lat: 22.55, groupId: 'sz-ft' },
  'sz-lh': { lng: 114.12, lat: 22.58, groupId: 'sz-lh' },
  // 成都
  'cd-jn': { lng: 104.08, lat: 30.66, groupId: 'cd-jn' },
  'cd-wh': { lng: 104.02, lat: 30.69, groupId: 'cd-wh' },
  // 武汉
  'wh-hk': { lng: 114.28, lat: 30.58, groupId: 'wh-hk' },
  'wh-wc': { lng: 114.35, lat: 30.55, groupId: 'wh-wc' },
  // 杭州
  'hz-xh': { lng: 120.12, lat: 30.28, groupId: 'hz-xh' },
  'hz-jg': { lng: 120.18, lat: 30.25, groupId: 'hz-jg' },
  // 南京
  'nj-xw': { lng: 118.78, lat: 32.06, groupId: 'nj-xw' },
  'nj-gl': { lng: 118.82, lat: 32.02, groupId: 'nj-gl' },
  // 西安
  'xa-ys': { lng: 108.95, lat: 34.27, groupId: 'xa-ys' },
  'xa-bl': { lng: 108.88, lat: 34.30, groupId: 'xa-bl' },
  // 重庆
  'cq-yb': { lng: 106.55, lat: 29.56, groupId: 'cq-yb' },
  'cq-jlp': { lng: 106.48, lat: 29.52, groupId: 'cq-jlp' },
  // 天津
  'tj-hp': { lng: 117.22, lat: 39.12, groupId: 'tj-hp' },
  'tj-nk': { lng: 117.15, lat: 39.10, groupId: 'tj-nk' },
  // 苏州
  'szh-gs': { lng: 120.62, lat: 31.32, groupId: 'szh-gs' },
  'szh-sz': { lng: 120.58, lat: 31.28, groupId: 'szh-sz' },
};

// 设备组树数据
const MOCK_GROUP_TREE: DeviceGroupNode[] = [
  {
    id: 'china',
    name: 'China group',
    parentId: null,
    level: 1,
    children: [
      {
        id: 'bj',
        name: 'Beijing group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'bj-cy', name: 'Chaoyang group', parentId: 'bj', level: 3, isLeaf: true },
          { id: 'bj-hd', name: 'Haidian group', parentId: 'bj', level: 3, isLeaf: true },
          { id: 'bj-dc', name: 'Dongcheng group', parentId: 'bj', level: 3, isLeaf: true },
          { id: 'bj-ft', name: 'Fengtai group', parentId: 'bj', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'sh',
        name: 'Shanghai group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'sh-pd', name: 'Pudong group', parentId: 'sh', level: 3, isLeaf: true },
          { id: 'sh-ja', name: 'Jingan group', parentId: 'sh', level: 3, isLeaf: true },
          { id: 'sh-xh', name: 'Xuhui group', parentId: 'sh', level: 3, isLeaf: true },
          { id: 'sh-mh', name: 'Minhang group', parentId: 'sh', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'gd',
        name: 'Guangdong group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'gz-th', name: 'Guangzhou Tianhe', parentId: 'gd', level: 3, isLeaf: true },
          { id: 'gz-by', name: 'Guangzhou Baiyun', parentId: 'gd', level: 3, isLeaf: true },
          { id: 'sz-ns', name: 'Shenzhen Nanshan', parentId: 'gd', level: 3, isLeaf: true },
          { id: 'sz-ft', name: 'Shenzhen Futian', parentId: 'gd', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'cd',
        name: 'Chengdu group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'cd-jn', name: 'Jinniu group', parentId: 'cd', level: 3, isLeaf: true },
          { id: 'cd-wh', name: 'Wuhou group', parentId: 'cd', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'wh',
        name: 'Wuhan group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'wh-hk', name: 'Hankou group', parentId: 'wh', level: 3, isLeaf: true },
          { id: 'wh-wc', name: 'Wuchang group', parentId: 'wh', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'hz',
        name: 'Zhejiang group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'hz-xh', name: 'Hangzhou Xihu', parentId: 'hz', level: 3, isLeaf: true },
          { id: 'hz-jg', name: 'Hangzhou Jianggan', parentId: 'hz', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'nj',
        name: 'Nanjing group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'nj-xw', name: 'Xuanwu group', parentId: 'nj', level: 3, isLeaf: true },
          { id: 'nj-gl', name: 'Gulou group', parentId: 'nj', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'xa',
        name: "Xi'an group",
        parentId: 'china',
        level: 2,
        children: [
          { id: 'xa-ys', name: 'Yanta group', parentId: 'xa', level: 3, isLeaf: true },
          { id: 'xa-bl', name: 'Beilin group', parentId: 'xa', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'cq',
        name: 'Chongqing group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'cq-yb', name: 'Yubei group', parentId: 'cq', level: 3, isLeaf: true },
          { id: 'cq-jlp', name: 'Jiangbei group', parentId: 'cq', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'tj',
        name: 'Tianjin group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'tj-hp', name: 'Heping group', parentId: 'tj', level: 3, isLeaf: true },
          { id: 'tj-nk', name: 'Nankai group', parentId: 'tj', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'szh',
        name: 'Suzhou group',
        parentId: 'china',
        level: 2,
        children: [
          { id: 'szh-gs', name: 'Gusu group', parentId: 'szh', level: 3, isLeaf: true },
          { id: 'szh-sz', name: 'Suzhou group', parentId: 'szh', level: 3, isLeaf: true },
        ],
      },
    ],
  },
];

// 组ID到组名路径映射
const GROUP_PATH_MAP: Record<string, string> = {};
MOCK_GROUP_TREE[0].children?.forEach((province) => {
  const provinceName = province.name;
  province.children?.forEach((city) => {
    GROUP_PATH_MAP[city.id] = `China / ${provinceName.replace(' group', '')} / ${city.name}`;
  });
});

// 生成 Mock 设备数据
function generateMockDevices(): ExtendedSite[] {
  const devices: ExtendedSite[] = [];
  const cityIds = Object.keys(CITY_COORDS);
  let deviceIndex = 0;

  cityIds.forEach((cityId) => {
    const cityCoord = CITY_COORDS[cityId];
    // 每个城市 3-8 个设备
    const deviceCount = 3 + Math.floor(Math.random() * 6);

    for (let i = 0; i < deviceCount; i++) {
      deviceIndex++;
      const isOnline = Math.random() > 0.08; // 92% 在线率
      const isActivated = Math.random() > 0.1; // 90% 激活率

      // 在城市坐标附近随机偏移
      const lngOffset = (Math.random() - 0.5) * 0.15;
      const latOffset = (Math.random() - 0.5) * 0.1;

      devices.push({
        id: String(deviceIndex).padStart(3, '0'),
        name: `Device ${String(deviceIndex).padStart(3, '0')} @ ${cityId.toUpperCase()}`,
        domainId: cityId,
        address: `Address ${deviceIndex}, District ${cityId.toUpperCase()}`,
        longitude: cityCoord.lng + lngOffset,
        latitude: cityCoord.lat + latOffset,
        deviceCount: 1,
        status: isOnline ? 'active' : 'inactive',
        activated: isActivated,
      });
    }
  });

  return devices;
}

// 生成 Mock 数据（只生成一次）
const MOCK_SITES: ExtendedSite[] = generateMockDevices();

// 将 mockMapDevices 转换为 ExtendedSite 格式（用于大量数据测试）
const LARGE_MOCK_SITES: ExtendedSite[] = mockMapDevices.map((device) => ({
  id: device.id,
  name: device.name,
  domainId: device.groupName?.split('/')[0] || 'LUSAKA',
  address: device.address || '',
  longitude: device.lng,
  latitude: device.lat,
  deviceCount: 1,
  status: device.status === 'online' ? 'active' : 'inactive',
  activated: true,
}));

// 大量 Mock 数据的设备组树
const LARGE_MOCK_GROUP_TREE: DeviceGroupNode[] = [
  {
    id: 'zambia',
    name: 'Zambia Network',
    parentId: null,
    level: 1,
    children: [
      {
        id: 'LUSAKA',
        name: 'Lusaka Region',
        parentId: 'zambia',
        level: 2,
        children: [
          { id: 'LUSAKA/LTE700', name: 'LTE700', parentId: 'LUSAKA', level: 3, isLeaf: true },
          { id: 'LUSAKA/NR2600', name: 'NR2600', parentId: 'LUSAKA', level: 3, isLeaf: true },
        ],
      },
      {
        id: 'COPPERBELT',
        name: 'Copperbelt Region',
        parentId: 'zambia',
        level: 2,
        isLeaf: true,
      },
      {
        id: 'CENTRAL',
        name: 'Central Region',
        parentId: 'zambia',
        level: 2,
        isLeaf: true,
      },
    ],
  },
];

// 大量 Mock 数据的组路径映射
const LARGE_GROUP_PATH_MAP: Record<string, string> = {
  'LUSAKA': 'Zambia / Lusaka',
  'LUSAKA/LTE700': 'Zambia / Lusaka / LTE700',
  'LUSAKA/NR2600': 'Zambia / Lusaka / NR2600',
  'COPPERBELT': 'Zambia / Copperbelt',
  'CENTRAL': 'Zambia / Central',
};

export default function GISMapView() {
  const token = useThemeToken();

  // 状态
  const [groupSearchValue, setGroupSearchValue] = useState('');
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>(['zambia']);
  const [expandedGroupIds, setExpandedGroupIds] = useState<string[]>(['zambia', 'LUSAKA']);
  // 扩展状态筛选：在线/离线 + 激活/未激活
  const [statusFilter, setStatusFilter] = useState<{
    online: boolean;
    offline: boolean;
    activated: boolean;
    deactivated: boolean;
  }>({
    online: true,
    offline: true,
    activated: true,
    deactivated: true,
  });
  const [deviceSearchValue, setDeviceSearchValue] = useState('');
  const [deviceSearchResults, setDeviceSearchResults] = useState<ExtendedSite[]>([]);
  const [deviceSearchExpanded, setDeviceSearchExpanded] = useState(false);
  // 高亮设备ID（用于搜索定位）
  const [highlightedDeviceId, setHighlightedDeviceId] = useState<string | null>(null);

  const searchInputRef = useRef<HTMLInputElement>(null);
  const searchContainerRef = useRef<HTMLDivElement>(null);

  // 地图组件引用，用于调用定位方法
  const mapRef = useRef<{ highlightAndFlyTo: (device: MapDevice) => void }>(null);

  const { data: sitesData } = useSites({});

  // 根据开关选择使用大量 Mock 数据还是普通 Mock 数据
  const sites = USE_LARGE_MOCK_DATA
    ? LARGE_MOCK_SITES
    : ((sitesData ?? MOCK_SITES) as ExtendedSite[]);

  // 选择对应的设备组树
  const groupTree = USE_LARGE_MOCK_DATA ? LARGE_MOCK_GROUP_TREE : MOCK_GROUP_TREE;
  const groupPathMap = USE_LARGE_MOCK_DATA ? LARGE_GROUP_PATH_MAP : GROUP_PATH_MAP;

  // ========== 动态统计计算 ==========

  // 全部设备统计（包含激活/未激活）
  const allStats = useMemo(() => {
    const total = sites.length;
    const online = sites.filter((s) => s.status === 'active').length;
    const offline = total - online;
    const activated = sites.filter((s) => s.activated).length;
    const deactivated = total - activated;
    return { total, online, offline, activated, deactivated };
  }, [sites]);

  // 过滤设备
  const filteredSites = useMemo(() => {
    return sites.filter((site) => {
      // 在线/离线状态过滤
      const isOnline = site.status === 'active';
      if (isOnline && !statusFilter.online) return false;
      if (!isOnline && !statusFilter.offline) return false;

      // 激活/未激活状态过滤
      if (site.activated && !statusFilter.activated) return false;
      if (!site.activated && !statusFilter.deactivated) return false;

      // 设备组过滤
      if (selectedGroupIds.length > 0) {
        const siteDomain = site.domainId;
        // 检查是否匹配任何选中的组（包括子组）
        const matchesGroup = selectedGroupIds.some((gid) => {
          if (gid === 'china' || gid === 'zambia') return true; // 根节点匹配所有
          return siteDomain === gid || siteDomain.startsWith(gid);
        });
        if (!matchesGroup) return false;
      }

      return true;
    });
  }, [sites, statusFilter, selectedGroupIds]);

  // 过滤后设备统计（包含激活/未激活）
  const filteredStats = useMemo(() => {
    const total = filteredSites.length;
    const online = filteredSites.filter((s) => s.status === 'active').length;
    const offline = total - online;
    const activated = filteredSites.filter((s) => s.activated).length;
    const deactivated = total - activated;
    return { total, online, offline, activated, deactivated };
  }, [filteredSites]);

  // 转换为地图设备
  const mapDevices: MapDevice[] = useMemo(
    () =>
      filteredSites.map((site) => ({
        id: site.id,
        lat: site.latitude,
        lng: site.longitude,
        name: site.name,
        status: site.status === 'active' ? 'online' : 'offline',
        sn: `SN2024${site.id}`,
        groupName: groupPathMap[site.domainId] || 'Zambia',
        address: site.address,
        alarmCount: site.status === 'active' ? Math.floor(Math.random() * 8) : 0,
      })),
    [filteredSites, groupPathMap]
  );

  // 设备搜索
  const handleDeviceSearch = useCallback(
    (value: string) => {
      setDeviceSearchValue(value);
      if (value.length >= 2) {
        const results = sites.filter(
          (s) =>
            s.name.toLowerCase().includes(value.toLowerCase()) ||
            s.id.toLowerCase().includes(value.toLowerCase())
        );
        setDeviceSearchResults(results);
        setDeviceSearchExpanded(true);
      } else {
        setDeviceSearchResults([]);
        setDeviceSearchExpanded(false);
      }
    },
    [sites]
  );

  // 获取所有子节点ID
  const getAllDescendantIds = useCallback((node: DeviceGroupNode): string[] => {
    const ids = [node.id];
    if (node.children) {
      node.children.forEach((child) => {
        ids.push(...getAllDescendantIds(child));
      });
    }
    return ids;
  }, []);

  // 过滤设备组树（根据搜索值）
  const filteredGroupTree = useMemo(() => {
    if (!groupSearchValue.trim()) {
      return groupTree;
    }

    const searchLower = groupSearchValue.toLowerCase();

    // 递归过滤树节点，返回匹配的节点及其父节点路径
    const filterNode = (node: DeviceGroupNode, parentMatch = false): DeviceGroupNode | null => {
      const nameMatch = node.name.toLowerCase().includes(searchLower);
      const idMatch = node.id.toLowerCase().includes(searchLower);
      const selfMatch = nameMatch || idMatch;

      // 处理子节点
      const filteredChildren: DeviceGroupNode[] = [];
      if (node.children) {
        node.children.forEach((child) => {
          const filteredChild = filterNode(child, selfMatch || parentMatch);
          if (filteredChild) {
            filteredChildren.push(filteredChild);
          }
        });
      }

      // 如果自己匹配，或者有匹配的子节点，则保留
      if (selfMatch || filteredChildren.length > 0) {
        return {
          ...node,
          children: filteredChildren.length > 0 ? filteredChildren : node.children,
        };
      }

      return null;
    };

    // 过滤整棵树
    const result: DeviceGroupNode[] = [];
    groupTree.forEach((node) => {
      const filtered = filterNode(node);
      if (filtered) {
        result.push(filtered);
      }
    });

    return result;
  }, [groupSearchValue]);

  // 当搜索值变化时，自动展开匹配的节点
  useEffect(() => {
    if (groupSearchValue.trim() && filteredGroupTree.length > 0) {
      // 收集所有需要展开的节点ID
      const collectExpandIds = (nodes: DeviceGroupNode[]): string[] => {
        const ids: string[] = [];
        nodes.forEach((node) => {
          if (node.children && node.children.length > 0) {
            ids.push(node.id);
            ids.push(...collectExpandIds(node.children));
          }
        });
        return ids;
      };
      const expandIds = collectExpandIds(filteredGroupTree);
      setExpandedGroupIds((prev) => [...new Set([...prev, ...expandIds])]);
    }
  }, [groupSearchValue, filteredGroupTree]);

  // 渲染设备组树节点
  const renderGroupNode = (node: DeviceGroupNode, depth: number = 0): React.ReactNode => {
    const isLeaf = node.isLeaf || !node.children || node.children.length === 0;
    const isExpanded = expandedGroupIds.includes(node.id);
    const isSelected = selectedGroupIds.includes(node.id);
    const paddingLeft = depth * 12 + 16;

    return (
      <div key={node.id}>
        {/* 节点行 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            padding: '8px 12px',
            paddingLeft,
            cursor: 'pointer',
            background: isSelected ? '#E6F7FF' : 'transparent',
            borderRadius: 6,
            margin: '2px 8px',
            transition: 'background 0.2s',
          }}
          onClick={() => {
            if (!isLeaf) {
              setExpandedGroupIds((prev) =>
                prev.includes(node.id) ? prev.filter((id) => id !== node.id) : [...prev, node.id]
              );
            }
          }}
        >
          {/* 复选框 */}
          <Checkbox
            checked={selectedGroupIds.includes(node.id)}
            onChange={(e) => {
              e.stopPropagation();
              const ids = getAllDescendantIds(node);
              if (e.target.checked) {
                setSelectedGroupIds((prev) => [...new Set([...prev, ...ids])]);
              } else {
                setSelectedGroupIds((prev) => prev.filter((id) => !ids.includes(id)));
              }
            }}
            style={{ marginRight: 8 }}
          />

          {/* 展开/收起图标 (仅非叶子节点) */}
          {!isLeaf && (
            <div
              onClick={(e) => {
                e.stopPropagation();
                setExpandedGroupIds((prev) =>
                  prev.includes(node.id) ? prev.filter((id) => id !== node.id) : [...prev, node.id]
                );
              }}
              style={{
                width: 14,
                height: 14,
                borderRadius: 2,
                border: `1.5px solid ${isExpanded ? token.colorPrimary : '#BFBFBF'}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                marginRight: 8,
                cursor: 'pointer',
                flexShrink: 0,
              }}
            >
              {isExpanded ? (
                <MinusOutlined style={{ fontSize: 8, color: token.colorPrimary }} />
              ) : (
                <PlusOutlined style={{ fontSize: 8, color: '#BFBFBF' }} />
              )}
            </div>
          )}

          {/* 节点名称 */}
          <span
            style={{
              fontSize: node.level === 1 ? 14 : 13,
              color: isSelected ? token.colorPrimary : node.level === 1 ? '#262626' : '#595959',
              fontWeight: isSelected || node.level === 1 ? 500 : 400,
            }}
          >
            {node.name}
          </span>
        </div>

        {/* 子节点 */}
        {isExpanded && !isLeaf && node.children && (
          <div>{node.children.map((child) => renderGroupNode(child, depth + 1))}</div>
        )}
      </div>
    );
  };

  // 样式定义
  const leftPanelStyle: React.CSSProperties = {
    width: 280,
    height: '100%',
    background: 'linear-gradient(180deg, #FAFBFC 0%, #F5F7FA 100%)',
    borderRight: '1px solid #E8E8E8',
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
  };

  const searchBoxStyle: React.CSSProperties = {
    margin: '16px 16px 0',
    padding: '10px 12px',
    background: '#FFF',
    border: '1px solid #D9D9D9',
    borderRadius: 8,
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  };

  const treeContainerStyle: React.CSSProperties = {
    flex: '0 0 auto',
    maxHeight: 280,
    overflow: 'auto',
    padding: '4px 0',
  };

  const dividerStyle: React.CSSProperties = {
    margin: '0 20px',
    borderTop: '1px solid #E8E8E8',
  };

  const sectionTitleStyle: React.CSSProperties = {
    padding: '12px 20px 6px',
    fontSize: 12,
    fontWeight: 500,
    color: '#8C8C8C',
  };

  const statsPanelStyle: React.CSSProperties = {
    position: 'absolute',
    right: 24,
    bottom: 24,
    width: 260,
    background: '#FFF',
    borderRadius: 12,
    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
    border: '1px solid #E8E8E8',
    padding: '16px 20px',
    zIndex: 10,
  };

  const zoomControlsStyle: React.CSSProperties = {
    position: 'absolute',
    right: 24,
    top: 100,
    width: 44,
    background: '#FFF',
    borderRadius: 12,
    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
    border: '1px solid #E8E8E8',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: '6px 0',
    zIndex: 10,
  };

  const zoomButtonStyle: React.CSSProperties = {
    width: 32,
    height: 32,
    borderRadius: '50%',
    background: '#F5F5F5',
    border: 'none',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    margin: '6px 0',
    transition: 'background 0.2s',
  };

  const deviceSearchStyle: React.CSSProperties = {
    position: 'absolute',
    left: 20,
    top: 20,
    width: 320,
    zIndex: 500,
  };

  const searchBoxOuterStyle: React.CSSProperties = {
    background: '#FFF',
    borderRadius: 12,
    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
    border: `2px solid ${token.colorPrimary}`,
    overflow: 'hidden',
  };

  const searchInputContainerStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    padding: '0 16px',
    height: 36,
    gap: 12,
  };

  const searchResultsStyle: React.CSSProperties = {
    maxHeight: 220,
    overflow: 'auto',
    borderTop: '1px solid #E8E8E8',
  };

  const searchResultItemStyle = (isHighlighted: boolean): React.CSSProperties => ({
    padding: '12px 16px',
    cursor: 'pointer',
    background: isHighlighted ? '#E6F7FF' : 'transparent',
    transition: 'background 0.15s',
  });

  return (
    <div style={{ display: 'flex', width: '100%', height: '100%', background: '#F0F2F5' }}>
      {/* ============ 左侧筛选面板 ============ */}
      <div style={leftPanelStyle}>
        {/* Header */}
        <div
          style={{
            height: 64,
            background: '#FFF',
            display: 'flex',
            alignItems: 'center',
            padding: '0 20px',
            borderBottom: '1px solid #E8E8E8',
          }}
        >
          <span style={{ fontSize: 18, fontWeight: 600, color: '#262626' }}>筛选</span>
        </div>

        {/* 搜索设备组 */}
        <div style={searchBoxStyle}>
          <SearchOutlined style={{ color: '#BFBFBF' }} />
          <input
            type="text"
            placeholder="搜索设备组"
            value={groupSearchValue}
            onChange={(e) => setGroupSearchValue(e.target.value)}
            style={{
              flex: 1,
              border: 'none',
              outline: 'none',
              fontSize: 13,
              background: 'transparent',
            }}
          />
        </div>

        {/* 设备组树 */}
        <div style={sectionTitleStyle}>设备组</div>
        <div style={treeContainerStyle}>
          {filteredGroupTree.map((node) => renderGroupNode(node))}
        </div>

        {/* 已选择汇总 */}
        {selectedGroupIds.length > 0 && (
          <div
            style={{
              margin: '8px 16px',
              padding: '10px 12px',
              background: '#E6F7FF',
              borderRadius: 8,
            }}
          >
            <span style={{ fontSize: 13, color: token.colorPrimary }}>
              已选择 {selectedGroupIds.length} 个设备组
            </span>
          </div>
        )}

        {/* 分隔线 */}
        <div style={dividerStyle} />

        {/* 设备状态筛选 */}
        <div style={sectionTitleStyle}>设备状态</div>
        <div style={{ padding: '0 20px' }}>
          {/* 在线 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              padding: '10px 0',
            }}
          >
            <Checkbox
              checked={statusFilter.online}
              onChange={(e) => setStatusFilter((prev) => ({ ...prev, online: e.target.checked }))}
            />
            <div
              style={{
                width: 14,
                height: 14,
                borderRadius: '50%',
                background: 'linear-gradient(180deg, #73D13D 0%, #52C41A 100%)',
                margin: '0 8px 0 12px',
              }}
            />
            <span style={{ fontSize: 14, color: '#262626' }}>在线</span>
            <span style={{ marginLeft: 'auto', fontSize: 14, color: '#8C8C8C' }}>
              {allStats.online.toLocaleString()}
            </span>
          </div>

          {/* 离线 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              padding: '10px 0',
            }}
          >
            <Checkbox
              checked={statusFilter.offline}
              onChange={(e) => setStatusFilter((prev) => ({ ...prev, offline: e.target.checked }))}
            />
            <div
              style={{
                width: 14,
                height: 14,
                borderRadius: '50%',
                background: '#b60808',
                margin: '0 8px 0 12px',
              }}
            />
            <span style={{ fontSize: 14, color: '#262626' }}>离线</span>
            <span style={{ marginLeft: 'auto', fontSize: 14, color: '#b60808' }}>
              {allStats.offline.toLocaleString()}
            </span>
          </div>

          {/* 激活 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              padding: '10px 0',
            }}
          >
            <Checkbox
              checked={statusFilter.activated}
              onChange={(e) => setStatusFilter((prev) => ({ ...prev, activated: e.target.checked }))}
            />
            <div
              style={{
                width: 14,
                height: 14,
                borderRadius: '50%',
                background: 'linear-gradient(180deg, #69B1FF 0%, #1677FF 100%)',
                margin: '0 8px 0 12px',
              }}
            />
            <span style={{ fontSize: 14, color: '#262626' }}>激活</span>
            <span style={{ marginLeft: 'auto', fontSize: 14, color: '#1677FF' }}>
              {allStats.activated.toLocaleString()}
            </span>
          </div>

          {/* 未激活 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              padding: '10px 0',
            }}
          >
            <Checkbox
              checked={statusFilter.deactivated}
              onChange={(e) => setStatusFilter((prev) => ({ ...prev, deactivated: e.target.checked }))}
            />
            <div
              style={{
                width: 14,
                height: 14,
                borderRadius: '50%',
                background: '#FAAD14',
                margin: '0 8px 0 12px',
              }}
            />
            <span style={{ fontSize: 14, color: '#262626' }}>未激活</span>
            <span style={{ marginLeft: 'auto', fontSize: 14, color: '#FAAD14' }}>
              {allStats.deactivated.toLocaleString()}
            </span>
          </div>
        </div>

        {/* 分隔线 */}
        <div style={dividerStyle} />

        {/* 图例 */}
        <div style={sectionTitleStyle}>图例</div>
        <div style={{ padding: '0 20px' }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '12px 24px' }}>
            {/* 在线设备 */}
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  background: 'linear-gradient(180deg, #73D13D 0%, #52C41A 100%)',
                  border: '2px solid #FFF',
                  boxShadow: '0 0 0 1px #E8E8E8',
                }}
              />
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>在线设备</span>
            </div>

            {/* 离线设备 */}
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  background: '#b60808',
                  border: '2px solid #FFF',
                  boxShadow: '0 0 0 1px #E8E8E8',
                }}
              />
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>离线设备</span>
            </div>

            {/* 设备聚合 */}
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 24,
                  height: 24,
                  borderRadius: '50%',
                  background: 'linear-gradient(180deg, #40A9FF 0%, #1890FF 100%)',
                  border: '2px solid #FFF',
                  boxShadow: '0 0 0 1px #E8E8E8',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <span style={{ fontSize: 9, fontWeight: 700, color: '#FFF' }}>N</span>
              </div>
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>设备聚合</span>
            </div>

            {/* 告警数量 */}
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 18,
                  height: 18,
                  borderRadius: '50%',
                  background: 'linear-gradient(180deg, #FF7875 0%, #F5222D 100%)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <span style={{ fontSize: 9, fontWeight: 700, color: '#FFF' }}>3</span>
              </div>
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>告警数量</span>
            </div>
          </div>
        </div>
      </div>

      {/* ============ 地图区域 ============ */}
      <div style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
        {/* 地图组件 */}
        <GISMap
          ref={mapRef}
          devices={mapDevices}
          height="100%"
          defaultCenter={USE_LARGE_MOCK_DATA ? [28.3, -15.4] : [116.4, 39.9]}
          defaultZoom={USE_LARGE_MOCK_DATA ? 11 : 5}
          showStats={false}
          showControls={false}
          onDeviceClick={(device) => {
            console.log('Device clicked:', device);
          }}
        />

        {/* ============ 设备搜索 ============ */}
        <div ref={searchContainerRef} style={deviceSearchStyle}>
          <div style={searchBoxOuterStyle}>
            {/* 搜索输入框 */}
            <div style={searchInputContainerStyle}>
              {/* 搜索图标 */}
              <div style={{ position: 'relative', width: 16, height: 16 }}>
                <div
                  style={{
                    width: 10,
                    height: 10,
                    border: `2px solid ${token.colorPrimary}`,
                    borderRadius: '50%',
                  }}
                />
                <div
                  style={{
                    position: 'absolute',
                    right: 2,
                    bottom: 4,
                    width: 6,
                    height: 2,
                    background: token.colorPrimary,
                    transform: 'rotate(45deg)',
                  }}
                />
              </div>

              {/* 输入框 */}
              <input
                ref={searchInputRef as any}
                type="text"
                value={deviceSearchValue}
                onChange={(e) => handleDeviceSearch(e.target.value)}
                onFocus={() => {
                  if (deviceSearchValue.length >= 2 && deviceSearchResults.length > 0) {
                    setDeviceSearchExpanded(true);
                  }
                }}
                placeholder="搜索设备名称或序列号..."
                style={{
                  flex: 1,
                  border: 'none',
                  outline: 'none',
                  fontSize: 13,
                  color: '#262626',
                  background: 'transparent',
                }}
              />

              {/* 清除按钮 */}
              {deviceSearchValue && (
                <div
                  onClick={() => {
                    handleDeviceSearch('');
                  }}
                  style={{
                    width: 24,
                    height: 20,
                    background: '#F5F5F5',
                    borderRadius: 4,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: 'pointer',
                  }}
                >
                  <span style={{ fontSize: 14, color: '#8C8C8C' }}>×</span>
                </div>
              )}

              {/* 展开箭头 */}
              <div
                style={{
                  fontSize: 12,
                  color: token.colorPrimary,
                  transform: deviceSearchExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
                  transition: 'transform 0.2s',
                }}
              >
                ▼
              </div>
            </div>

            {/* 搜索结果列表 */}
            {deviceSearchExpanded && (
              <div style={searchResultsStyle}>
                {deviceSearchResults.length === 0 ? (
                  <div style={{ padding: 24, textAlign: 'center', color: '#8C8C8C' }}>
                    🔍
                    <br />
                    <span style={{ fontSize: 12 }}>未找到匹配的设备</span>
                    <br />
                    <span style={{ fontSize: 11, color: '#BFBFBF' }}>请尝试其他关键词</span>
                  </div>
                ) : (
                  <>
                    {deviceSearchResults.map((result, index) => {
                      const isOnline = result.status === 'active';
                      return (
                        <div
                          key={result.id}
                          style={searchResultItemStyle(index === 0)}
                          onClick={() => {
                            // 关闭搜索结果面板
                            setDeviceSearchExpanded(false);
                            // 定位并高亮设备
                            const mapDevice: MapDevice = {
                              id: result.id,
                              lat: result.latitude,
                              lng: result.longitude,
                              name: result.name,
                              status: isOnline ? 'online' : 'offline',
                              sn: `SN2024${result.id}`,
                              groupName: groupPathMap[result.domainId] || 'Zambia',
                              address: result.address,
                            };
                            mapRef.current?.highlightAndFlyTo(mapDevice);
                          }}
                        >
                          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                            <div
                              style={{
                                width: 12,
                                height: 12,
                                borderRadius: '50%',
                                background: isOnline
                                  ? 'linear-gradient(180deg, #73D13D 0%, #52C41A 100%)'
                                  : '#b60808',
                              }}
                            />
                            <span style={{ fontSize: 13, fontWeight: 500, color: '#262626' }}>
                              {result.name}
                            </span>
                          </div>
                          <div
                            style={{
                              display: 'flex',
                              justifyContent: 'space-between',
                              alignItems: 'center',
                              marginTop: 4,
                              paddingLeft: 20,
                            }}
                          >
                            <span style={{ fontSize: 11, color: '#8C8C8C' }}>
                              SN: SN2024{result.id}
                            </span>
                            <span style={{ fontSize: 11, color: isOnline ? '#52C41A' : '#b60808' }}>
                              {isOnline ? '🟢 在线' : '🔴 离线'}
                            </span>
                          </div>
                        </div>
                      );
                    })}

                    {/* 结果计数 */}
                    <div
                      style={{
                        padding: '10px 16px',
                        borderTop: '1px solid #F0F0F0',
                        textAlign: 'center',
                      }}
                    >
                      <span style={{ fontSize: 11, color: '#8C8C8C' }}>
                        共找到 {deviceSearchResults.length} 个结果
                      </span>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        </div>

        {/* ============ 缩放控制 ============ */}
        <div style={zoomControlsStyle}>
          <button
            style={zoomButtonStyle}
            onMouseEnter={(e) => (e.currentTarget.style.background = '#E8E8E8')}
            onMouseLeave={(e) => (e.currentTarget.style.background = '#F5F5F5')}
            onClick={() => console.log('Zoom in')}
          >
            <span style={{ fontSize: 16, fontWeight: 600, color: '#262626' }}>+</span>
          </button>
          <div style={{ width: 24, height: 1, background: '#F0F0F0' }} />
          <button
            style={zoomButtonStyle}
            onMouseEnter={(e) => (e.currentTarget.style.background = '#E8E8E8')}
            onMouseLeave={(e) => (e.currentTarget.style.background = '#F5F5F5')}
            onClick={() => console.log('Zoom out')}
          >
            <span style={{ fontSize: 16, fontWeight: 600, color: '#262626' }}>−</span>
          </button>
        </div>

        {/* ============ 统计面板 (使用过滤后的数据) ============ */}
        <div style={statsPanelStyle}>
          <div style={{ fontSize: 14, fontWeight: 600, color: '#262626', marginBottom: 8 }}>
            设备统计
          </div>
          <div style={{ borderTop: '1px solid #F0F0F0', margin: '8px 0 16px' }} />

          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontSize: 12, color: '#8C8C8C' }}>总设备</span>
            <span style={{ fontSize: 22, fontWeight: 700, color: '#262626' }}>
              {filteredStats.total.toLocaleString()}
            </span>
          </div>

          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginTop: 12,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 12,
                  height: 12,
                  borderRadius: '50%',
                  background: '#52C41A',
                  marginRight: 8,
                }}
              />
              <span style={{ fontSize: 12, color: '#595959' }}>在线</span>
            </div>
            <span style={{ fontSize: 14, fontWeight: 600, color: '#52C41A' }}>
              {filteredStats.online.toLocaleString()}
            </span>
          </div>

          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginTop: 12,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 12,
                  height: 12,
                  borderRadius: '50%',
                  background: '#b60808',
                  marginRight: 8,
                }}
              />
              <span style={{ fontSize: 12, color: '#595959' }}>离线</span>
            </div>
            <span style={{ fontSize: 14, fontWeight: 600, color: '#8C8C8C' }}>
              {filteredStats.offline.toLocaleString()}
            </span>
          </div>

          <div style={{ borderTop: '1px solid #F0F0F0', margin: '12px 0' }} />

          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 12,
                  height: 12,
                  borderRadius: '50%',
                  background: '#1677FF',
                  marginRight: 8,
                }}
              />
              <span style={{ fontSize: 12, color: '#595959' }}>激活</span>
            </div>
            <span style={{ fontSize: 14, fontWeight: 600, color: '#1677FF' }}>
              {filteredStats.activated.toLocaleString()}
            </span>
          </div>

          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginTop: 12,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 12,
                  height: 12,
                  borderRadius: '50%',
                  background: '#FAAD14',
                  marginRight: 8,
                }}
              />
              <span style={{ fontSize: 12, color: '#595959' }}>未激活</span>
            </div>
            <span style={{ fontSize: 14, fontWeight: 600, color: '#FAAD14' }}>
              {filteredStats.deactivated.toLocaleString()}
            </span>
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            position: 'absolute',
            bottom: 8,
            left: '50%',
            transform: 'translateX(-50%)',
            fontSize: 11,
            color: '#BFBFBF',
          }}
        >
          OMC GIS Map - Topology View v1.5
        </div>
      </div>
    </div>
  );
}
