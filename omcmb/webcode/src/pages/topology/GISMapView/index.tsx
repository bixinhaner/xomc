/**
 * GIS 地图视图页面
 * 完全按照 UI 原型图 GISMap_UI_Design_Main.svg 实现
 * 使用真实 API 接口获取数据
 *
 * UI/UX 优化：
 * - 统一的设计 Token 和间距系统
 * - 分层阴影系统
 * - 流畅的动画和交互反馈
 * - 可访问性增强
 */
import { useState, useMemo, useRef, useEffect, useCallback } from 'react';
import { useIntl } from 'react-intl';
import { useNavigate } from 'react-router-dom';
import { useTabStore } from '@core/store/tabStore';
import { Button, Checkbox, Spin, Empty, Collapse, Input, Tooltip, message } from 'antd';
import {
  CaretDownOutlined,
  MinusOutlined,
  PlusOutlined,
  SearchOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import GISMap from '@/components/GISMap';
import { MAP_CONFIG } from '@/components/GISMap/constants';
import { useMapConfig } from '@/components/GISMap/useMapConfig';
import { calculateCenterFromDevices, parseEnvCenter, resolveMapInitialView } from '@/utils/mapValidation';
import type { GISMapRef } from '@/components/GISMap';
import type { AntennaSector, MapDevice, DeviceGroupNode, DeviceGeo, MapViewport } from '@core/types/map';
import type {
  GeofenceMapDefinition,
  GeofencePolygonGeometry,
} from '@core/types/geofence';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useResponsive } from '@/hooks/useResponsive';
import { usePermission } from '@core/hooks/usePermission';
// import { useMapDeviceCache } from '@/hooks/useMapDeviceCache'; // 暂未使用
import {
  useDomainTree,
  useDeviceAntennaSectors,
  useMapDevicesGeo,
  useMapStats,
} from '@core/hooks/api/useTopology';
import { useDeviceSearch } from '@core/hooks/useDeviceSearch';
import { useAntennaSectorEditor } from '@core/hooks/useAntennaSectorEditor';
import {
  useGeofenceAvailability,
  useGeofenceMap,
} from '@core/hooks/api/useGeofence';
import { topologyApi } from '@core/services/api/topologyApi';
import { SPACING, RADIUS, SHADOWS, COLORS, transitionString, DURATION, EASING } from './styles';
import { hasValidCoord } from './coord';
import {
  buildGroupDisplayNameById,
  domainToGroupNode,
  filterGroupTreeBySearch,
  getGroupNodeDisplayName,
  normalizeGroupLocale,
} from './groupDisplay';
import { buildGeofenceMapQuery } from './geofenceViewModel';
import GeofenceBindingsDrawer from './GeofenceBindingsDrawer';
import GeofenceEditorDrawer from './GeofenceEditorDrawer';
import GeofencePanel from './GeofencePanel';
import GeofenceSettingsDrawer from './GeofenceSettingsDrawer';
import './animations.css';

// 环境变量在运行期不变，解析一次即可，避免每次 useMemo 重跑并重复打日志
const ENV_CENTER = parseEnvCenter();
const EMPTY_ANTENNA_SECTORS: AntennaSector[] = [];
const EMPTY_GEOFENCES: GeofenceMapDefinition[] = [];
const GEOFENCE_VIEW_PERMISSION = 'topology:gis-map:geofence:view';
const GEOFENCE_MANAGE_PERMISSION = 'topology:gis-map:geofence:manage';

/**
 * 将 DeviceGeo 转换为 MapDevice
 * 注意：调用前需确保 latitude/longitude 不为 null
 */
function deviceGeoToMapDevice(device: DeviceGeo): MapDevice {
  return {
    id: device.id,
    // 非空断言：调用前已过滤 null 值（见 mapDevices 的 filter）
    lat: device.latitude!,
    lng: device.longitude!,
    name: device.name,
    status: device.status,
    sn: device.sn,
    groupName: device.groupName,
    address: device.address,
    alarmCount: device.alarmCount,
    type: device.type,
    groupId: device.groupId,
    // 新增字段：IP/MAC/PCI/设备名称
    ip_address: device.ip_address,
    mac: device.mac,
    pci: device.pci,
    device_name: device.device_name,
    ueCount: device.ueCount,
    highestAlarmSeverity: device.highestAlarmSeverity,
    highestSeverityAlarmCount: device.highestSeverityAlarmCount,
  };
}

/**
 * 递归获取节点及其所有子孙节点的 ID
 */
function getAllDescendantIds(node: DeviceGroupNode): string[] {
  const ids: string[] = [node.id];
  if (node.children) {
    for (const child of node.children) {
      ids.push(...getAllDescendantIds(child));
    }
  }
  return ids;
}

// 图标绘制组件（双箭头图标）
const CollapseIcon = ({ direction }: { direction: 'left' | 'right' }) => (
  <svg width={14} height={14} viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
    {direction === 'left' ? (
      <>
        <path d="M8.5 3.5L5 7L8.5 10.5" stroke="#262626" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        <path d="M5 3.5L1.5 7L5 10.5" stroke="#262626" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
      </>
    ) : (
      <>
        <path d="M5.5 3.5L9 7L5.5 10.5" stroke="#262626" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        <path d="M9 3.5L12.5 7L9 10.5" stroke="#262626" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
      </>
    )}
  </svg>
);

export default function GISMapView() {
  const token = useThemeToken();
  const { isMobile } = useResponsive();
  const intl = useIntl();
  const canViewGeofence = usePermission(GEOFENCE_VIEW_PERMISSION);
  const canManageGeofence = usePermission(GEOFENCE_MANAGE_PERMISSION);

  // ========== 状态管理 ==========

  // 设备组搜索
  const [groupSearchValue, setGroupSearchValue] = useState('');
  // 选中的设备组 ID 列表（支持多选）
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);
  // 展开的设备组 ID 列表
  const [expandedGroupIds, setExpandedGroupIds] = useState<string[]>([]);
  // 是否已完成初始化（用于控制 API 请求时机）
  const [isInitialized, setIsInitialized] = useState(false);
  // 首次设备查询完成前不使用地图 bounds，避免初始视口反向限制设备发现。
  const [isInitialGeoQueryComplete, setIsInitialGeoQueryComplete] = useState(false);
  // 地图视口状态（用于动态加载设备）
  const [mapViewport, setMapViewport] = useState<MapViewport | null>(null);
  // 用户操作地图后，后台设备刷新不得再次覆盖用户视图。
  const [mapViewControlState, setMapViewControlState] = useState<'automatic' | 'user-controlled'>('automatic');
  // 搜索结果设备（用于独立显示在地图上）
  const [searchResultDevice, setSearchResultDevice] = useState<MapDevice | null>(null);
  const [selectedDevice, setSelectedDevice] = useState<MapDevice | null>(null);
  const [geofenceToolEnabled, setGeofenceToolEnabled] =
    useState(false);
  const [geofencePanelVisible, setGeofencePanelVisible] =
    useState(true);
  const [selectedGeofence, setSelectedGeofence] =
    useState<GeofenceMapDefinition | null>(null);
  const [geofenceSettingsOpen, setGeofenceSettingsOpen] =
    useState(false);
  const [geofenceEditorOpen, setGeofenceEditorOpen] =
    useState(false);
  const [geofenceDrawing, setGeofenceDrawing] = useState(false);
  const [geofenceEditorItem, setGeofenceEditorItem] =
    useState<GeofenceMapDefinition>();
  const [drawnGeofenceGeometry, setDrawnGeofenceGeometry] =
    useState<GeofencePolygonGeometry>();
  const [geofenceBindingsItem, setGeofenceBindingsItem] =
    useState<GeofenceMapDefinition>();
  const [geofenceBindingDeviceSNs, setGeofenceBindingDeviceSNs] =
    useState<string[]>();

  // 状态筛选：在线激活/在线未激活/离线
  const [statusFilter, setStatusFilter] = useState<{
    onlineActive: boolean;
    onlineInactive: boolean;
    offline: boolean;
  }>({
    onlineActive: true,
    onlineInactive: true,
    offline: true,
  });

  // 左侧筛选面板折叠状态（默认设备组展开，设备状态展开）
  const [filterPanelActiveKeys, setFilterPanelActiveKeys] = useState<string[]>([
    'deviceStatus',   // 设备状态
    'deviceGroup',    // 设备组（新增：默认展开）
  ]);

  // 侧边栏折叠状态（默认收起）
  const [sidebarCollapsed, setSidebarCollapsed] = useState(true);

  // ========== 显示/隐藏控制配置 ==========
  // 图例模块显示配置（默认隐藏）
  const SHOW_LEGEND = false;

  // 地图组件引用
  const mapRef = useRef<GISMapRef>(null);
  const navigate = useNavigate();
  const openTab = useTabStore((s) => s.openTab);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const searchContainerRef = useRef<HTMLDivElement>(null);

  // 测距模式状态
  const [isMeasuring, setIsMeasuring] = useState(false);

  // ========== 缓存机制 ==========

  // 设备数据 LRU 缓存 Hook（保留但暂未集成）
  //
  // 设计意图：按视口边界（bounds）缓存设备数据，支持：
  // - LRU 淘汰策略（最多 10 个视口）
  // - 5 分钟自动过期
  // - 跨组件会话共享缓存
  //
  // 当前状态：React Query 已提供完善的缓存机制（staleTime: 5分钟）
  // 其基于完整 queryKey（bounds + groupIds + status + pageSize）的缓存
  // 已能满足当前性能需求，LRU 缓存优势有限
  //
  // 未来场景：如需跨会话共享或更细粒度的 bounds 级别缓存时再集成
  //
  // LRU 缓存 Hook（暂未启用，代码注释保留）
  // const _deviceCache = useMapDeviceCache({
  //   maxSize: 10,
  //   ttl: 5 * 60 * 1000, // 5 分钟过期
  // });

  // 用于防抖的定时器
  const viewportChangeTimerRef = useRef<NodeJS.Timeout | null>(null);

  // ========== API Hooks ==========

  // 获取设备组树
  const { data: domainTree, isLoading: isLoadingTree } = useDomainTree();

  // ========== 分级显示策略 ==========

  /**
   * 根据缩放级别获取数据加载策略
   *
   * @param zoom - 地图缩放级别
   * @returns 加载策略配置
   */
  const getLoadStrategy = useCallback((zoom: number) => {
    if (zoom < 8) {
      // 低缩放级别：显示区域级聚合，数据量小
      return {
        aggregate: 'region',
        pageSize: 500,
        description: intl.formatMessage({ id: 'gis.strategy.region' }),
      };
    } else if (zoom < 12) {
      // 中缩放级别：显示站点级数据
      return {
        aggregate: 'site',
        pageSize: 1000,
        description: intl.formatMessage({ id: 'gis.strategy.site' }),
      };
    } else if (zoom < 15) {
      // 高缩放级别：显示详细设备，启用聚合
      return {
        aggregate: 'device',
        pageSize: 2000,
        description: intl.formatMessage({ id: 'gis.strategy.deviceAggregated' }),
      };
    } else {
      // 超高缩放级别：显示详细设备，后端按 bounds + 前端 VIEWPORT_CULLING 收收收口
      // 不再用 5000/10000 这种极端值，避免拖动时单发请求后端几秒
      return {
        aggregate: 'none',
        pageSize: 2000,
        description: intl.formatMessage({ id: 'gis.strategy.deviceDetailed' }),
      };
    }
  }, [intl]);

  // ========== 数据转换 ==========

  // 设备组树（转换格式）- 必须在 allGroupIds 之前定义
  const groupTree: DeviceGroupNode[] = useMemo(() => {
    if (!domainTree?.length) return [];
    return domainTree.map(domainToGroupNode);
  }, [domainTree]);

  const groupLocale = normalizeGroupLocale(intl.locale);

  const groupDisplayNameById = useMemo(
    () => buildGroupDisplayNameById(groupTree, groupLocale),
    [groupTree, groupLocale],
  );

  const localizeMapDevice = useCallback((device: MapDevice): MapDevice => {
    if (!device.groupId) return device;
    return {
      ...device,
      groupName: groupDisplayNameById.get(device.groupId) ?? device.groupName,
    };
  }, [groupDisplayNameById]);

  // 计算所有组的 ID 集合（用于判断是否选中了"全部"）
  const allGroupIds = useMemo(() => {
    return new Set(groupTree.flatMap((node) => getAllDescendantIds(node)));
  }, [groupTree]);

  // 获取设备地理数据（支持视口动态加载和分级显示）
  const filterParams = useMemo(() => {
    const statusList: ('onlineActive' | 'onlineInactive' | 'offline')[] = [
      ...(statusFilter.onlineActive ? ['onlineActive' as const] : []),
      ...(statusFilter.onlineInactive ? ['onlineInactive' as const] : []),
      ...(statusFilter.offline ? ['offline' as const] : []),
    ];

    // 判断是否选中了所有组（严格全等：避免 length 巧合一致但成员不同）
    // 全选时传 undefined 让后端返回所有设备（包括未分组的）
    const selectedSet = new Set(selectedGroupIds);
    const isAllSelected = allGroupIds.size > 0 &&
      [...allGroupIds].every((id) => selectedSet.has(id));

    // 根据视口范围生成 bounds 参数
    let boundsParam: string | undefined;
    if (isInitialGeoQueryComplete && mapViewport?.bounds) {
      const { minLng, maxLng, minLat, maxLat } = mapViewport.bounds;
      boundsParam = `${minLng},${maxLng},${minLat},${maxLat}`;
    }

    // 根据缩放级别获取加载策略
    const currentZoom = mapViewport?.zoom ?? MAP_CONFIG.defaultZoom;
    const strategy = getLoadStrategy(currentZoom);

    return {
      // 选中所有组时传 undefined（返回所有设备，包括未分组的）
      // 只选中部分组时传具体的 groupIds
      groupIds: isAllSelected ? undefined : (selectedGroupIds.length > 0 ? selectedGroupIds : undefined),
      // 如果三个状态都被选中（默认情况），不传 status 参数让后端返回全部
      // 如果部分被选中，传对应的状态
      // 如果都没选中，传空数组表示不查询任何设备
      status: statusList.length === 3 ? undefined : statusList,
      // 视口边界参数（用于动态加载可见区域设备）
      bounds: boundsParam,
      // 只有初始化完成后才启用请求，避免在 selectedGroupIds 为空时发送请求
      enabled: isInitialized,
      // 根据缩放级别调整页面大小
      pageSize: strategy.pageSize,
    };
  }, [
    selectedGroupIds,
    statusFilter,
    isInitialized,
    isInitialGeoQueryComplete,
    allGroupIds,
    mapViewport,
    getLoadStrategy,
  ]);

  const { data: devicesGeoData, isSuccess: isDevicesGeoQuerySuccessful } = useMapDevicesGeo(filterParams);
  const geofenceAvailabilityQuery = useGeofenceAvailability();
  const geofenceFeatureVisible =
    canViewGeofence && geofenceAvailabilityQuery.data?.enabled === true;
  const geofenceToolActive = geofenceFeatureVisible && geofenceToolEnabled;

  useEffect(() => {
    if (isDevicesGeoQuerySuccessful) {
      setIsInitialGeoQueryComplete(true);
    }
  }, [isDevicesGeoQuerySuccessful]);
  const geofenceMapQuery = useMemo(
    () =>
      buildGeofenceMapQuery(
        geofenceToolActive,
        mapViewport,
      ),
    [geofenceToolActive, mapViewport],
  );
  const { data: geofenceMapData } = useGeofenceMap(
    geofenceMapQuery.filter,
    { enabled: geofenceMapQuery.enabled },
  );
  const visibleGeofences = geofenceToolActive
    ? (geofenceMapData?.items ?? EMPTY_GEOFENCES)
    : EMPTY_GEOFENCES;
  const {
    data: antennaSectors = EMPTY_ANTENNA_SECTORS,
  } = useDeviceAntennaSectors(selectedDevice?.id);
  const {
    previewSectors,
    updatePreview,
    discardPreview,
    saveSector,
    isSaving: antennaSaving,
  } = useAntennaSectorEditor(selectedDevice?.id, antennaSectors);

  // 加载地图元数据（离线瓦片配置）
  const mapConfigData = useMapConfig();

  // 获取地图统计数据（与 useMapDevicesGeo 共用同一套 group/status 过滤口径，
  // 避免顶部统计与地图设备不一致；不传 bounds，统计始终反映过滤维度的全量）
  const { data: mapStatsData } = useMapStats({
    groupIds: filterParams.groupIds,
    status: filterParams.status,
  });

  // 设备搜索 hook（支持防抖、50 值限制和自动展开）
  const {
    keyword: searchKeyword,
    handleChange: handleSearchInputChange,
    handleClear: _handleSearchClear,
    results: searchResults,
    isLoading: isSearching,
    expanded: deviceSearchExpanded,
    setExpanded: setDeviceSearchExpanded,
  } = useDeviceSearch((kw, signal) => topologyApi.searchDevices(kw, signal), {
    minLength: 2,
    maxValues: 50,
    debounce: 300,
    autoExpand: true,
  });

  // 清除搜索时同时清除地图上的搜索结果设备
  const handleSearchClear = useCallback(() => {
    _handleSearchClear();
    setSearchResultDevice(null); // 同时清除地图上的高亮设备
  }, [_handleSearchClear]);

  // ========== 数据转换 ==========

  // 设备列表（转换为 MapDevice 格式）
  const mapDevices: MapDevice[] = useMemo(() => {
    if (!devicesGeoData?.items?.length) return [];
    return devicesGeoData.items
      .filter(device => device.latitude != null && device.longitude != null)
      .map(deviceGeoToMapDevice)
      .map(localizeMapDevice);
  }, [devicesGeoData, localizeMapDevice]);

  const localizedSearchResultDevice = useMemo(
    () => searchResultDevice ? localizeMapDevice(searchResultDevice) : null,
    [searchResultDevice, localizeMapDevice],
  );

  const localizedSelectedDevice = useMemo(
    () => selectedDevice ? localizeMapDevice(selectedDevice) : null,
    [selectedDevice, localizeMapDevice],
  );

  // 统计数据（匹配 MapStats 类型）
  const stats = useMemo(() => {
    if (!mapStatsData) {
      return {
        total: 0,
        statusCount: {
          onlineActive: 0,
          onlineInactive: 0,
          offline: 0,
        },
        alarmCount: 0,
        center: undefined,
      };
    }
    return {
      total: mapStatsData.total,
      statusCount: {
        onlineActive: mapStatsData.statusCount?.onlineActive ?? 0,
        onlineInactive: mapStatsData.statusCount?.onlineInactive ?? 0,
        offline: mapStatsData.statusCount?.offline ?? 0,
      },
      alarmCount: mapStatsData.alarmCount ?? 0,
      center: mapStatsData.center,
    };
  }, [mapStatsData]);

  // 使用稳定引用避免频繁重新计算
  const deviceItems = devicesGeoData?.items;

  /**
   * 计算初始地图中心点和缩放级别（4层决策逻辑）
   * 
   * 优先级：
   * 1. tiles.json 元数据中心点（如果离线瓦片可用）
   * 2. 设备数据计算的中心点（如果元数据不可用但设备存在）
   * 3. 环境变量配置的中心点（支持多地区部署）
   * 4. 代码默认值（全球默认）
   */
  const { initialCenter, initialZoom } = useMemo(() => {
    const resolvedView = resolveMapInitialView({
      metadata: mapConfigData.metadata,
      metadataAvailable:
        mapConfigData.status === 'success' &&
        !mapConfigData.isUsingDefault &&
        mapConfigData.tilesAvailable === true,
      devices: devicesGeoData?.complete ? (deviceItems ?? []) : [],
      envCenter: ENV_CENTER,
      defaultCenter: MAP_CONFIG.defaultCenter,
      defaultZoom: MAP_CONFIG.defaultZoom,
    });

    return {
      initialCenter: resolvedView.center,
      initialZoom: resolvedView.zoom,
    };
  }, [
    mapConfigData.metadata,
    mapConfigData.isUsingDefault,
    mapConfigData.status,
    mapConfigData.tilesAvailable,
    deviceItems,
  ]);

  // 首屏兜底：若 4 层中心点策略落点（通常是 metadata.center）与实际设备分布不在同一区域，
  // 用户首屏会看不到任何设备点。这里在地图就绪、设备数据到货后做一次性 fit：
  // - 若当前视口已经包含至少一个设备 → 标记完成，不动；
  // - 若一个都没有 → flyTo 到设备 bounds 中心。
  // 仅触发一次（hasAutoFittedRef 守卫），后续用户拖动/缩放不再被覆盖。
  const hasAutoFittedRef = useRef(false);
  useEffect(() => {
    if (hasAutoFittedRef.current) return;
    if (mapViewControlState === 'user-controlled') return;
    if (!devicesGeoData?.complete || !mapDevices.length) return;

    const map = mapRef.current;
    if (!map) return;

    const vp = map.getViewport();
    if (!vp?.bounds) return;

    const { bounds } = vp;
    const hasAnyInView = mapDevices.some(
      (d) =>
        d.lng >= bounds.minLng &&
        d.lng <= bounds.maxLng &&
        d.lat >= bounds.minLat &&
        d.lat <= bounds.maxLat,
    );

    if (hasAnyInView) {
      hasAutoFittedRef.current = true;
      return;
    }

    const fitTarget = calculateCenterFromDevices(
      mapDevices.map((d) => ({ longitude: d.lng, latitude: d.lat })),
    );
    map.flyTo(fitTarget.center[0], fitTarget.center[1], fitTarget.zoom, {
      progressive: false,
    });
    hasAutoFittedRef.current = true;
  }, [devicesGeoData?.complete, mapDevices, mapViewControlState]);

  // ========== 搜索处理 ==========

  // 搜索结果过滤（根据状态过滤）
  const filteredSearchResults = useMemo(() => {
    if (!searchResults) return [];
    return searchResults.filter((device) => {
      if (device.status === 'onlineActive' && !statusFilter.onlineActive) return false;
      if (device.status === 'onlineInactive' && !statusFilter.onlineInactive) return false;
      if (device.status === 'offline' && !statusFilter.offline) return false;
      return true;
    });
  }, [searchResults, statusFilter]);

  // ========== 设备组树处理 ==========

  // 过滤设备组树（根据搜索值）
  const filteredGroupTree = useMemo(
    () => filterGroupTreeBySearch(groupTree, groupSearchValue, groupLocale),
    [groupSearchValue, groupTree, groupLocale],
  );

  // 自动展开匹配的节点
  /* eslint-disable react-hooks/set-state-in-effect -- 搜索时自动展开是预期的副作用 */
  useEffect(() => {
    if (groupSearchValue.trim() && filteredGroupTree.length > 0) {
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
    } else if (!groupSearchValue.trim() && groupTree.length > 0) {
      // 搜索清空时，重置展开状态为根节点
      const rootIds = groupTree.map((node) => node.id);
      setExpandedGroupIds(rootIds);
    }
  }, [groupSearchValue, filteredGroupTree, groupTree]);

  // 初始化：选中并展开所有节点（包括子节点）
  /* eslint-disable react-hooks/set-state-in-effect -- 初始化时设置状态是预期的副作用 */
  useEffect(() => {
    if (groupTree.length > 0 && selectedGroupIds.length === 0) {
      // 默认选中所有节点（包括子节点）
      const allIds = groupTree.flatMap((node) => getAllDescendantIds(node));
      setSelectedGroupIds(allIds);
      // 默认展开根节点
      const rootIds = groupTree.map((node) => node.id);
      setExpandedGroupIds(rootIds);
      // 标记初始化完成，允许 API 请求
      setIsInitialized(true);
    }
  }, [groupTree, selectedGroupIds.length]);

  // ESC 退出测距模式
  useEffect(() => {
    if (!isMeasuring) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        mapRef.current?.stopMeasure();
        setIsMeasuring(false);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [isMeasuring]);

  // 渲染设备组树节点
  const renderGroupNode = (node: DeviceGroupNode, depth: number = 0): React.ReactNode => {
    const isLeaf = node.isLeaf || !node.children || node.children.length === 0;
    const isExpanded = expandedGroupIds.includes(node.id);
    const isSelected = selectedGroupIds.includes(node.id);
    const paddingLeft = depth * 12 + 16;

    return (
      <div key={node.id}>
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

          <span
            style={{
              fontSize: node.level === 1 ? 14 : 13,
              color: isSelected ? token.colorPrimary : node.level === 1 ? '#262626' : '#595959',
              fontWeight: isSelected || node.level === 1 ? 500 : 400,
            }}
          >
            {getGroupNodeDisplayName(node, groupLocale)}
          </span>
        </div>

        {isExpanded && !isLeaf && node.children && (
          <div>{node.children.map((child) => renderGroupNode(child, depth + 1))}</div>
        )}
      </div>
    );
  };

  // ========== 样式定义 ==========

  // 侧边栏容器样式（包含折叠按钮）
  const sidebarContainerStyle: React.CSSProperties = {
    position: 'relative',
    height: '100%',
    display: 'flex',
    alignItems: 'flex-start',
    transition: 'width 0.3s ease',
  };

  const leftPanelStyle: React.CSSProperties = {
    width: sidebarCollapsed ? 0 : 280,
    height: '100%',
    background: 'linear-gradient(180deg, #FAFBFC 0%, #F5F7FA 100%)',
    borderRight: sidebarCollapsed ? 'none' : '1px solid #E8E8E8',
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
    opacity: sidebarCollapsed ? 0 : 1,
    transition: 'width 0.3s ease, opacity 0.3s ease, border-right 0.3s ease',
  };

  // 左侧面板中间可滚动区域
  const leftPanelMiddleStyle: React.CSSProperties = {
    flex: 1,
    overflowY: 'auto',
    overflowX: 'hidden',
  };

  const searchBoxStyle: React.CSSProperties = {
    margin: `${SPACING.md}px ${SPACING.md}px ${SPACING.md}px`,
    padding: `${SPACING.md}px ${SPACING.lg}px`,
    background: '#FFF',
    border: '1px solid #E8E8E8',
    borderRadius: RADIUS.md,
    display: 'flex',
    alignItems: 'center',
    gap: SPACING.md,
  };

  // ========== Collapse 组件样式 ==========

  // Collapse.Panel 头部样式
  const collapseHeaderStyle: React.CSSProperties = {
    fontSize: 13,
    fontWeight: 600,
    color: '#595959',
    transition: transitionString(['color'], 'fast'),
  };

  // 折叠面板之间的间距
  const collapseItemStyle: React.CSSProperties = {
    marginBottom: SPACING.md,
    borderRadius: RADIUS.md,
    overflow: 'hidden',
    transition: transitionString(['background', 'box-shadow'], 'fast'),
    background: '#FFF',
    border: '1px solid #E8E8E8',
  };

  // 设备组面板特殊样式（避免被搜索框遮挡）
  const deviceGroupCollapseStyle: React.CSSProperties = {
    ...collapseItemStyle,
    marginTop: 0,
  };

  // 自定义折叠图标（红色向下箭头，与图片中的设计一致）
  const customExpandIcon = (panelProps: { isActive?: boolean }) => (
    <CaretDownOutlined
      rotate={panelProps.isActive ? 0 : 180}
      style={{
        color: '#FF4D4F',      // 红色箭头
        fontSize: 12,
        transition: `transform ${DURATION.normal}ms ${EASING.out}`,
      }}
    />
  );

  const zoomControlsStyle: React.CSSProperties = {
    position: 'absolute',
    right: selectedDevice && !isMobile ? 364 : 24,
    top: 100,
    width: 44,
    background: '#FFF',
    borderRadius: RADIUS.lg,
    boxShadow: SHADOWS.medium,
    border: `1px solid ${COLORS.neutral[200]}`,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: `${SPACING.sm}px 0`,
    zIndex: 10,
    transition: transitionString(['box-shadow', 'transform'], 'fast'),
  };

  // ========== 侧边栏折叠按钮设计 ==========

  // 侧边栏折叠按钮样式（右上角圆形按钮）
  const collapseButtonStyle: React.CSSProperties = {
    position: 'absolute',
    left: sidebarCollapsed ? 12 : 288,
    top: 16,
    width: 32,
    height: 32,
    background: '#FFF',
    borderRadius: '50%',
    boxShadow: SHADOWS.medium,
    border: `1px solid ${COLORS.neutral[200]}`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    cursor: 'pointer',
    zIndex: 600,
    transition: 'left 0.3s ease, background 0.2s, transform 0.2s',
  };

  const collapseButtonHoverStyle: React.CSSProperties = {
    background: COLORS.neutral[100],
    transform: 'scale(1.1)',
  };

  const zoomButtonStyle: React.CSSProperties = {
    width: 32,
    height: 32,
    borderRadius: '50%',
    background: COLORS.neutral[100],
    border: 'none',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    margin: `${SPACING.xs}px 0`,
    transition: transitionString(['background', 'transform'], 'fast'),
  };

  // 地图设备搜索框样式（固定位置，不随侧边栏状态变化）
  const deviceSearchStyle: React.CSSProperties = {
    position: 'absolute',
    left: 60,
    top: 12,
    width: 320,
    zIndex: 500,
  };

  const searchBoxOuterStyle: React.CSSProperties = {
    background: '#FFF',
    borderRadius: RADIUS.lg,
    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
    border: `2px solid ${deviceSearchExpanded ? token.colorPrimary : '#D9D9D9'}`,
    overflow: 'hidden',
    transition: 'border-color 0.2s',
  };

  const searchInputContainerStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    padding: `0 ${SPACING.xl}px`,
    height: 36,
    gap: SPACING.md,
  };

  const searchResultsStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    maxHeight: 280,
    borderTop: '1px solid #E8E8E8',
  };

  const searchResultsListStyle: React.CSSProperties = {
    flex: 1,
    overflow: 'auto',
    maxHeight: 240,
  };

  const searchResultsFooterStyle: React.CSSProperties = {
    padding: `${SPACING.md}px ${SPACING.xl}px`,
    borderTop: '1px solid #F0F0F0',
    textAlign: 'center',
    background: '#FAFAFA',
    flexShrink: 0,
  };

  const searchResultItemStyle = (isHighlighted: boolean): React.CSSProperties => ({
    padding: `${SPACING.md}px ${SPACING.xl}px`,
    cursor: 'pointer',
    background: isHighlighted ? '#E6F7FF' : 'transparent',
    transition: 'background 0.15s',
  });

  // ========== 渲染 ==========

  // 折叠按钮 hover 状态管理
  const [collapseButtonHovered, setCollapseButtonHovered] = useState(false);

  return (
    <div style={{ display: 'flex', width: '100%', height: '100%', background: '#F0F2F5' }}>
      {/* 左侧筛选面板容器 */}
      <div style={sidebarContainerStyle}>
        {/* 折叠/展开按钮 */}
        <Tooltip
          title={sidebarCollapsed ? intl.formatMessage({ id: 'common.expand' }) : intl.formatMessage({ id: 'common.collapse' })}
          placement="right"
        >
          <div
            style={{
              ...collapseButtonStyle,
              ...(collapseButtonHovered ? collapseButtonHoverStyle : {}),
            }}
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            onMouseEnter={() => setCollapseButtonHovered(true)}
            onMouseLeave={() => setCollapseButtonHovered(false)}
          >
            {sidebarCollapsed ? (
              <CollapseIcon direction="right" />
            ) : (
              <CollapseIcon direction="left" />
            )}
          </div>
        </Tooltip>

        {/* 左侧筛选面板 */}
        <div style={leftPanelStyle}>

        {/* 中间可滚动区域 */}
        <div style={leftPanelMiddleStyle}>
          {/* 折叠面板：设备状态 */}
          <Collapse
            activeKey={filterPanelActiveKeys.includes('deviceStatus') ? ['deviceStatus'] : []}
            onChange={(keys) => {
              const newKeys = keys.length > 0
                ? [...filterPanelActiveKeys.filter(k => k !== 'deviceStatus'), 'deviceStatus']
                : filterPanelActiveKeys.filter(k => k !== 'deviceStatus');
              setFilterPanelActiveKeys(newKeys as string[]);
            }}
            expandIcon={customExpandIcon}
            bordered={false}
            style={{ margin: `0 ${SPACING.md}px ${SPACING.md}px`, padding: 0 }}
            className="gismap-filter-collapse"
            items={[
            {
              key: 'deviceStatus',
              label: <span style={collapseHeaderStyle}>{intl.formatMessage({ id: 'gis.filter.deviceStatus' })}</span>,
              style: collapseItemStyle,
              children: (
                <div style={{ padding: '0 4px' }}>
              {/* 在线激活 */}
              <div style={{ display: 'flex', alignItems: 'center', padding: '10px 0' }}>
                <Checkbox
                  checked={statusFilter.onlineActive}
                  onChange={(e) => setStatusFilter((prev) => ({ ...prev, onlineActive: e.target.checked }))}
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
                <span style={{ fontSize: 14, color: 'var(--color-neutral-800)' }}>{intl.formatMessage({ id: 'gis.status.onlineActive' })}</span>
                <span style={{ marginLeft: 'auto', fontSize: 14, color: '#52C41A' }}>
                  {stats.statusCount.onlineActive.toLocaleString()}
                </span>
              </div>

              {/* 在线未激活 */}
              <div style={{ display: 'flex', alignItems: 'center', padding: '10px 0' }}>
                <Checkbox
                  checked={statusFilter.onlineInactive}
                  onChange={(e) => setStatusFilter((prev) => ({ ...prev, onlineInactive: e.target.checked }))}
                />
                <div
                  style={{
                    width: 14,
                    height: 14,
                    borderRadius: '50%',
                    background: 'linear-gradient(180deg, #FFC53D 0%, #FAAD14 100%)',
                    margin: '0 8px 0 12px',
                  }}
                />
                <span style={{ fontSize: 14, color: 'var(--color-neutral-800)' }}>{intl.formatMessage({ id: 'gis.status.onlineInactive' })}</span>
                <span style={{ marginLeft: 'auto', fontSize: 14, color: '#FAAD14' }}>
                  {stats.statusCount.onlineInactive.toLocaleString()}
                </span>
              </div>

              {/* 离线 */}
              <div style={{ display: 'flex', alignItems: 'center', padding: '10px 0' }}>
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
                <span style={{ fontSize: 14, color: 'var(--color-neutral-800)' }}>{intl.formatMessage({ id: 'gis.status.offline' })}</span>
                <span style={{ marginLeft: 'auto', fontSize: 14, color: '#b60808' }}>
                  {stats.statusCount.offline.toLocaleString()}
                </span>
              </div>
            </div>
              ),
            },
          ]}
        />

        {/* 搜索设备组 */}
        <div style={searchBoxStyle}>
          <Input
            placeholder={intl.formatMessage({ id: 'gis.search.deviceGroupPlaceholder' })}
            value={groupSearchValue}
            onChange={(e) => setGroupSearchValue(e.target.value)}
            prefix={<SearchOutlined style={{ color: '#8C8C8C' }} />}
            allowClear
            style={{
              border: 'none',
              padding: 0,
              background: 'transparent',
              color: '#262626',
            }}
            variant="borderless"
            className="device-group-search-input"
          />
        </div>

        {/* 折叠面板：设备组 */}
        <Collapse
          activeKey={filterPanelActiveKeys.includes('deviceGroup') ? ['deviceGroup'] : []}
          onChange={(keys) => {
            const newKeys = keys.length > 0
              ? [...filterPanelActiveKeys.filter(k => k !== 'deviceGroup'), 'deviceGroup']
              : filterPanelActiveKeys.filter(k => k !== 'deviceGroup');
            setFilterPanelActiveKeys(newKeys as string[]);
          }}
          expandIcon={customExpandIcon}
          bordered={false}
          style={{ margin: `0 ${SPACING.md}px ${SPACING.md}px`, padding: 0 }}
          className="gismap-filter-collapse"
          items={[
            {
              key: 'deviceGroup',
              label: <span style={collapseHeaderStyle}>{intl.formatMessage({ id: 'gis.filter.deviceGroup' })}</span>,
              style: deviceGroupCollapseStyle,
              children: (
                <div style={{ display: 'flex', flexDirection: 'column', maxHeight: '50vh' }}>
                  {/* 可滚动的设备组列表 */}
                  <div style={{ flex: 1, overflowY: 'auto', overflowX: 'hidden', padding: `${SPACING.xs}px 0` }}>
                    {isLoadingTree ? (
                      <div style={{ padding: 20, textAlign: 'center' }}>
                        <Spin size="small" />
                      </div>
                    ) : filteredGroupTree.length === 0 ? (
                      <Empty description={intl.formatMessage({ id: 'gis.search.noDeviceGroup' })} style={{ padding: 20 }} />
                    ) : (
                      filteredGroupTree.map((node) => renderGroupNode(node))
                    )}
                  </div>
                  {/* 固定在底部的汇总 */}
                  {selectedGroupIds.length > 0 && (
                    <div style={{ padding: '28px 12px 0', borderTop: '1px solid #E8E8E8', background: '#FFF', flexShrink: 0 }}>
                      <span style={{ fontSize: 13, color: '#595959' }}>
                        {intl.formatMessage({ id: 'gis.search.selectedGroups' }, { count: selectedGroupIds.length })}
                      </span>
                    </div>
                  )}
                </div>
              ),
            },
            // 图例 - 可通过 SHOW_LEGEND 控制，默认隐藏
            ...(SHOW_LEGEND ? [{
              key: 'legend',
              label: <span style={collapseHeaderStyle}>{intl.formatMessage({ id: 'gis.filter.legend' })}</span>,
              style: collapseItemStyle,
              children: (
                <div style={{ padding: '0 4px' }}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '12px 24px' }}>
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
                <span style={{ marginLeft: 8, fontSize: 12, color: 'var(--color-neutral-600)' }}>{intl.formatMessage({ id: 'gis.status.onlineActive' })}</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <div
                  style={{
                    width: 20,
                    height: 20,
                    borderRadius: '50%',
                    background: 'linear-gradient(180deg, #FFC53D 0%, #FAAD14 100%)',
                    border: '2px solid #FFF',
                    boxShadow: '0 0 0 1px #E8E8E8',
                  }}
                />
                <span style={{ marginLeft: 8, fontSize: 12, color: 'var(--color-neutral-600)' }}>{intl.formatMessage({ id: 'gis.status.onlineInactive' })}</span>
              </div>
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
                <span style={{ marginLeft: 8, fontSize: 12, color: 'var(--color-neutral-600)' }}>{intl.formatMessage({ id: 'gis.legend.offlineDevice' })}</span>
              </div>
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
                <span style={{ marginLeft: 8, fontSize: 12, color: 'var(--color-neutral-600)' }}>{intl.formatMessage({ id: 'gis.legend.deviceCluster' })}</span>
              </div>
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
                <span style={{ marginLeft: 8, fontSize: 12, color: 'var(--color-neutral-600)' }}>{intl.formatMessage({ id: 'gis.legend.alarmCount' })}</span>
              </div>
              </div>
            </div>
              ),
            }] : []),
          ]}
        />
        </div>
      </div>
      </div>

      {/* 地图区域 */}
      <div style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
        <GISMap
          ref={mapRef}
          devices={mapDevices}
          geofences={visibleGeofences}
          selectedGeofenceId={
            selectedGeofence?.definition.id
          }
          onGeofenceClick={(item) => {
            setSelectedDevice(null);
            setSelectedGeofence(item);
          }}
          onGeofenceDrawComplete={(geometry) => {
            setDrawnGeofenceGeometry(geometry);
            setGeofenceDrawing(false);
            setGeofenceEditorOpen(true);
          }}
          searchResultDevice={localizedSearchResultDevice}
          selectedDevice={localizedSelectedDevice}
          antennaSectors={previewSectors}
          height="100%"
          defaultCenter={initialCenter}
          defaultZoom={initialZoom}
          showStats={false}
          showControls={false}
          tileUrl={mapConfigData.status === 'success' && !mapConfigData.isUsingDefault ? MAP_CONFIG.tileUrl : undefined}
          onDeviceClick={(device) => {
            setSelectedGeofence(null);
            setSelectedDevice(device);
          }}
          onAlarmClick={(sn) => {
              const path = `/alarm/current?deviceSN=${encodeURIComponent(sn)}`;
              openTab({ key: 'alarm/current', label: intl.formatMessage({ id: 'nav.alarm.current' }), path, closable: true, labelRaw: true });
              navigate(path);
            }}
          onMapClick={() => {
			setSelectedDevice(null);
            setSelectedGeofence(null);
            // 点击地图时收起搜索结果面板
            setDeviceSearchExpanded(false);
            // 不清除搜索结果设备，保留高亮显示
            // 只有用户重新搜索或手动清除时才移除
            // setSearchResultDevice(null);
          }}
          onAntennaPreviewChange={updatePreview}
          onAntennaCancel={discardPreview}
          onAntennaSave={saveSector}
          antennaSaving={antennaSaving}
          onViewportChange={(viewport) => {
            // useOLMap 已过滤程序化飞行事件，此处收到的 viewport 代表用户视图变化。
            setMapViewControlState('user-controlled');

            // 视口变化防抖（默认 300ms，可通过 MAP_CONFIG.viewportDebounce 调整）
            // 需要这里防抖是因为 useOLMap 内部的 moveend 只做了 100ms 偏轻的合并，
            // 拖动过程中仍会频繁调出；再叠一层防抖避免拖动期间堆 setState。
            if (viewportChangeTimerRef.current) {
              clearTimeout(viewportChangeTimerRef.current);
            }

            viewportChangeTimerRef.current = setTimeout(() => {
              // 更新视口状态，触发设备数据重新请求
              // 旧请求会被 React Query 自动 abort（queryFn 已打通 signal）
              setMapViewport(viewport);

              // TODO(P2): 预加载周边区域，待后端支持批量 bounds 查询时实施
            }, MAP_CONFIG.viewportDebounce);
          }}
        />

        {geofenceFeatureVisible && (
        <div className="geofence-map-controls">
          <Button
            className={`geofence-map-tool-button${geofenceToolEnabled ? ' is-active' : ''}`}
            aria-pressed={geofenceToolEnabled}
            aria-label={intl.formatMessage({
              id: 'geofence.map.enableTool',
            })}
            icon={<SafetyCertificateOutlined aria-hidden="true" />}
            onClick={() => {
              const nextEnabled = !geofenceToolEnabled;
              if (!nextEnabled) {
                mapRef.current?.stopGeofenceDraw();
                setSelectedGeofence(null);
                setGeofenceSettingsOpen(false);
                setGeofenceEditorOpen(false);
                setGeofenceDrawing(false);
                setGeofenceEditorItem(undefined);
                setDrawnGeofenceGeometry(undefined);
                setGeofenceBindingsItem(undefined);
                setGeofenceBindingDeviceSNs(undefined);
                setGeofencePanelVisible(true);
              } else if (isMeasuring) {
                mapRef.current?.stopMeasure();
                setIsMeasuring(false);
              }
              if (nextEnabled) setGeofencePanelVisible(true);
              setGeofenceToolEnabled(nextEnabled);
            }}
          >
            {intl.formatMessage({
              id: 'geofence.map.enableTool',
            })}
          </Button>

          {geofenceToolEnabled && canManageGeofence && (
            <Button
              aria-label={intl.formatMessage({
                id: 'geofence.settings.title',
              })}
              icon={<SettingOutlined aria-hidden="true" />}
              onClick={() => setGeofenceSettingsOpen(true)}
            >
              {intl.formatMessage({
                id: 'geofence.action.settings',
              })}
            </Button>
          )}
        </div>
        )}

        {geofenceToolEnabled &&
          !geofencePanelVisible &&
          !geofenceDrawing &&
          !geofenceEditorOpen && (
            <Button
              className="geofence-panel-return"
              icon={<UnorderedListOutlined aria-hidden="true" />}
              onClick={() => setGeofencePanelVisible(true)}
            >
              {intl.formatMessage({ id: 'geofence.map.showList' })}
            </Button>
          )}

        {geofenceDrawing && (
          <div className="geofence-drawing-status">
            <span>
              {intl.formatMessage({
                id: geofenceEditorItem
                  ? 'geofence.message.redrawing'
                  : 'geofence.message.drawing',
              })}
            </span>
            <Button
              size="small"
              onClick={() => {
                mapRef.current?.stopGeofenceDraw();
                setGeofenceDrawing(false);
                setDrawnGeofenceGeometry(undefined);
                if (geofenceEditorItem) {
                  setGeofenceEditorOpen(true);
                }
              }}
            >
              {intl.formatMessage({ id: 'geofence.action.cancelDrawing' })}
            </Button>
          </div>
        )}

        <GeofencePanel
          canManage={canManageGeofence}
          open={
            geofenceToolEnabled &&
            geofencePanelVisible &&
            !geofenceDrawing &&
            !geofenceEditorOpen
          }
          selectedId={selectedGeofence?.definition.id}
          onCreate={() => {
            setGeofenceEditorItem(undefined);
            setDrawnGeofenceGeometry(undefined);
            setGeofenceDrawing(true);
            mapRef.current?.startGeofencePolygonDraw();
          }}
          onEdit={(item) => {
            setGeofenceEditorItem(item);
            setDrawnGeofenceGeometry(undefined);
            setGeofenceEditorOpen(true);
          }}
          onBind={(item, deviceSNs) => {
            setGeofenceBindingsItem(item);
            setGeofenceBindingDeviceSNs(deviceSNs);
          }}
          onLocate={(item) => {
            const bounds = item.currentVersion?.boundingBox;
            if (!bounds) return;
            setSelectedGeofence(item);
            setGeofencePanelVisible(false);
            mapRef.current?.fitBounds({
              minLng: bounds.minLongitude,
              maxLng: bounds.maxLongitude,
              minLat: bounds.minLatitude,
              maxLat: bounds.maxLatitude,
            });
          }}
        />

        {geofenceSettingsOpen && canManageGeofence && (
          <GeofenceSettingsDrawer
            open
            onClose={() => setGeofenceSettingsOpen(false)}
          />
        )}

        {geofenceEditorOpen && canManageGeofence && (
          <GeofenceEditorDrawer
            open
            item={geofenceEditorItem}
            drawnGeometry={drawnGeofenceGeometry}
            onStartDraw={() => {
              setDrawnGeofenceGeometry(undefined);
              setGeofenceEditorOpen(false);
              setGeofenceDrawing(true);
              mapRef.current?.startGeofencePolygonDraw();
            }}
            onStopDraw={() => mapRef.current?.stopGeofenceDraw()}
            onClose={() => {
              setGeofenceEditorOpen(false);
              setGeofenceEditorItem(undefined);
              setDrawnGeofenceGeometry(undefined);
              setGeofenceDrawing(false);
            }}
          />
        )}

        {canManageGeofence && (
          <GeofenceBindingsDrawer
            open={Boolean(geofenceBindingsItem)}
            item={geofenceBindingsItem}
            initialDeviceSNs={geofenceBindingDeviceSNs}
            onClose={() => {
              setGeofenceBindingsItem(undefined);
              setGeofenceBindingDeviceSNs(undefined);
            }}
          />
        )}

        {/* 设备搜索 */}
        <div ref={searchContainerRef} style={deviceSearchStyle} onClick={(e) => e.stopPropagation()}>
          <div style={searchBoxOuterStyle}>
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
                ref={searchInputRef}
                type="text"
                value={searchKeyword}
                onChange={(e) => handleSearchInputChange(e.target.value)}
                placeholder={intl.formatMessage({ id: 'gis.search.placeholder' })}
                style={{
                  flex: 1,
                  border: 'none',
                  outline: 'none',
                  fontSize: 13,
                  color: 'var(--color-neutral-800)',
                  background: 'transparent',
                }}
              />

              {/* 清除按钮 */}
              {searchKeyword && (
                <div
                  onClick={handleSearchClear}
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
                onClick={() => setDeviceSearchExpanded(!deviceSearchExpanded)}
                style={{
                  fontSize: 12,
                  color: token.colorPrimary,
                  transform: deviceSearchExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
                  transition: 'transform 0.2s',
                  cursor: 'pointer',
                }}
              >
                ▼
              </div>
            </div>

            {/* 搜索结果列表 */}
            {deviceSearchExpanded && (
              <div style={searchResultsStyle}>
                {isSearching ? (
                  <div style={{ padding: 24, textAlign: 'center' }}>
                    <Spin size="small" />
                  </div>
                ) : filteredSearchResults.length === 0 ? (
                  <div style={{ padding: 24, textAlign: 'center', color: '#8C8C8C' }}>
                    🔍
                    <br />
                    <span style={{ fontSize: 12 }}>{intl.formatMessage({ id: 'gis.search.notFound' })}</span>
                    <br />
                    <span style={{ fontSize: 11, color: '#BFBFBF' }}>{intl.formatMessage({ id: 'gis.search.tryOther' })}</span>
                  </div>
                ) : (
                  <>
                    {/* 可滚动的结果列表 */}
                    <div style={searchResultsListStyle}>
                      {filteredSearchResults.map((result, index) => {
                        // 状态颜色：在线激活=绿色，在线未激活=黄色，离线=红色
                        const statusColor = result.status === 'onlineActive'
                          ? 'linear-gradient(180deg, #73D13D 0%, #52C41A 100%)'
                          : result.status === 'onlineInactive'
                            ? 'linear-gradient(180deg, #FFC53D 0%, #FAAD14 100%)'
                            : '#b60808';
                        const statusText = result.status === 'onlineActive'
                          ? `🟢 ${intl.formatMessage({ id: 'gis.status.onlineActive' })}`
                          : result.status === 'onlineInactive'
                            ? `🟡 ${intl.formatMessage({ id: 'gis.status.onlineInactive' })}`
                            : `🔴 ${intl.formatMessage({ id: 'gis.status.offline' })}`;
                        const statusTextColor = result.status === 'onlineActive'
                          ? '#52C41A'
                          : result.status === 'onlineInactive'
                            ? '#FAAD14'
                            : '#b60808';
                        // 无有效坐标（null 或 (0,0)）的设备无法在地图定位，做视觉标记 + 点击给提示
                        const coordValid = hasValidCoord(result.latitude, result.longitude);
                        return (
                          <div
                            key={result.id}
                            style={{
                              ...searchResultItemStyle(index === 0),
                              ...(coordValid ? {} : { opacity: 0.6 }),
                            }}
                            onClick={() => {
                              // 无有效坐标：给出明确反馈而非静默无反应或飞到 (0,0) 海面（issue #192）
                              if (!coordValid) {
                                void message.warning(intl.formatMessage({ id: 'gis.search.noCoordToast' }));
                                return;
                              }
                              setDeviceSearchExpanded(false);
                              const mapDevice: MapDevice = {
                                id: result.id,
                                lat: result.latitude!,
                                lng: result.longitude!,
                                name: result.name,
                                status: result.status,
                                sn: result.sn,
                                groupId: result.groupId,
                                groupName: result.groupName,
                                address: '',
                                alarmCount: result.alarmCount ?? 0,
                                type: undefined,
                                ip_address: result.ip_address,
                                mac: result.mac,
                                pci: result.pci,
                                device_name: result.device_name,
                                ueCount: result.ueCount,
                                highestAlarmSeverity: result.highestAlarmSeverity,
                                highestSeverityAlarmCount: result.highestSeverityAlarmCount,
                              };
                              const localizedMapDevice = localizeMapDevice(mapDevice);
                              // 设置搜索结果设备，让地图组件独立显示
                              setSearchResultDevice(localizedMapDevice);
                              setSelectedDevice(localizedMapDevice);
                            }}
                          >
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              <div
                                style={{
                                  width: 12,
                                  height: 12,
                                  borderRadius: '50%',
                                  background: statusColor,
                                }}
                              />
                              <span style={{ fontSize: 13, fontWeight: 500, color: 'var(--color-neutral-800)' }}>
                                {result.name}
                              </span>
                              {!coordValid && (
                                <Tooltip title={intl.formatMessage({ id: 'gis.search.noCoordToast' })}>
                                  <span
                                    style={{
                                      fontSize: 10,
                                      lineHeight: '16px',
                                      padding: '0 6px',
                                      borderRadius: 4,
                                      color: '#FA8C16',
                                      background: 'rgba(250, 140, 22, 0.12)',
                                      border: '1px solid rgba(250, 140, 22, 0.4)',
                                    }}
                                  >
                                    {intl.formatMessage({ id: 'gis.search.noCoordTag' })}
                                  </span>
                                </Tooltip>
                              )}
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
                                SN: {result.sn}
                              </span>
                              <span style={{ fontSize: 11, color: statusTextColor }}>
                                {statusText}
                              </span>
                            </div>
                          </div>
                        );
                      })}
                    </div>

                    {/* 固定在底部的结果统计 */}
                    <div style={searchResultsFooterStyle}>
                      <span style={{ fontSize: 11, color: '#8C8C8C' }}>
                        {intl.formatMessage({ id: 'gis.search.resultsCount' }, { count: filteredSearchResults.length })}
                      </span>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        </div>

        {/* 测距工具按钮（搜索框右侧） */}
        <Tooltip
          title={isMeasuring ? '点击退出测距（或按 ESC）' : '测距'}
          placement="bottom"
          mouseEnterDelay={0.3}
        >
          <div
            style={{
              position: 'absolute',
              left: 392,
              top: 12,
              zIndex: 500,
              width: 40,
              height: 40,
              background: isMeasuring ? '#1677ff' : '#FFF',
              borderRadius: 10,
              boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
              border: isMeasuring ? '2px solid #1677ff' : '2px solid #D9D9D9',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              transition: 'all 0.2s',
              color: isMeasuring ? '#fff' : '#595959',
            }}
            onMouseEnter={(e) => {
              if (!isMeasuring) {
                e.currentTarget.style.boxShadow = '0 4px 12px rgba(0,0,0,0.15)';
                e.currentTarget.style.background = '#F5F5F5';
              }
            }}
            onMouseLeave={(e) => {
              if (!isMeasuring) {
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(0,0,0,0.08)';
                e.currentTarget.style.background = '#FFF';
              }
            }}
            onClick={() => {
              if (isMeasuring) {
                mapRef.current?.stopMeasure();
                setIsMeasuring(false);
              } else {
                mapRef.current?.startMeasure();
                setIsMeasuring(true);
              }
            }}
          >
            {/* 尺子图标（Material Design straighten，有刻度线） */}
            <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
              <path d="M21 6H3c-1.1 0-2 .9-2 2v8c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 10H3V8h2v4h2V8h2v4h2V8h2v4h2V8h2v4h2V8h2v8z"/>
            </svg>
          </div>
        </Tooltip>

        {/* 缩放控制 */}
        <div style={zoomControlsStyle}>
          <button
            style={zoomButtonStyle}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = COLORS.neutral[200];
              e.currentTarget.style.transform = 'scale(1.1)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = COLORS.neutral[100];
              e.currentTarget.style.transform = 'scale(1)';
            }}
            onClick={() => {
              const view = mapRef.current?.getViewport();
              const currentZoom = view?.zoom ?? MAP_CONFIG.defaultZoom;
              mapRef.current?.flyTo(
                view?.centerLng ?? MAP_CONFIG.defaultCenter[0],
                view?.centerLat ?? MAP_CONFIG.defaultCenter[1],
                Math.min(currentZoom + 1, 18)
              );
            }}
          >
            <span style={{ fontSize: 16, fontWeight: 600, color: COLORS.neutral[800] }}>+</span>
          </button>
          <div style={{ width: 24, height: 1, backgroundColor: COLORS.neutral[100] }} />
          <button
            style={zoomButtonStyle}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = COLORS.neutral[200];
              e.currentTarget.style.transform = 'scale(1.1)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = COLORS.neutral[100];
              e.currentTarget.style.transform = 'scale(1)';
            }}
            onClick={() => {
              const view = mapRef.current?.getViewport();
              const currentZoom = view?.zoom ?? MAP_CONFIG.defaultZoom;
              mapRef.current?.flyTo(
                view?.centerLng ?? MAP_CONFIG.defaultCenter[0],
                view?.centerLat ?? MAP_CONFIG.defaultCenter[1],
                Math.max(currentZoom - 1, 3)
              );
            }}
          >
            <span style={{ fontSize: 16, fontWeight: 600, color: COLORS.neutral[800] }}>−</span>
          </button>
        </div>

      </div>
    </div>
  );
}
