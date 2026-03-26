/**
 * GIS 地图相关类型定义
 * @module types/map
 */

/**
 * 设备状态枚举
 */
export type DeviceStatus = 'online' | 'offline';

/**
 * 设备类型枚举
 */
export type DeviceType = 'macro' | 'small' | 'pico' | 'femto' | 'rru';

/**
 * 地图设备标记数据
 */
export interface MapDevice {
  /** 设备ID */
  id: string;
  /** 纬度 */
  lat: number;
  /** 经度 */
  lng: number;
  /** 设备名称 */
  name: string;
  /** 设备状态 */
  status: DeviceStatus;
  /** 序列号 */
  sn: string;
  /** 设备组ID */
  groupId?: string;
  /** 设备组名称 */
  groupName?: string;
  /** 告警数量 */
  alarmCount?: number;
  /** 详细地址 */
  address?: string;
  /** 设备类型 */
  type?: DeviceType;
}

/**
 * 设备地理信息（后端返回格式）
 */
export interface DeviceGeo {
  /** 设备ID */
  id: string;
  /** 设备名称 */
  name: string;
  /** 设备序列号 */
  sn: string;
  /** 经度 (东经为正) */
  longitude: number;
  /** 纬度 (北纬为正) */
  latitude: number;
  /** 设备状态 */
  status: DeviceStatus;
  /** 设备类型 */
  type?: DeviceType;
  /** 所属设备组ID */
  groupId: string;
  /** 设备组名称 */
  groupName?: string;
  /** 详细地址 */
  address?: string;
  /** 告警数量 */
  alarmCount?: number;
}

/**
 * 设备聚合项
 */
export interface DeviceCluster {
  /** 聚合ID */
  id: string;
  /** 聚合中心点经度 */
  longitude: number;
  /** 聚合中心点纬度 */
  latitude: number;
  /** 聚合内设备数量 */
  count: number;
  /** 各状态数量统计 */
  statusCount: Record<DeviceStatus, number>;
  /** 聚合内告警总数 */
  alarmCount: number;
  /** 聚合范围（网格） */
  bounds?: MapBounds;
}

/**
 * 地图边界
 */
export interface MapBounds {
  minLng: number;
  maxLng: number;
  minLat: number;
  maxLat: number;
}

/**
 * 地图视图状态
 */
export interface MapViewport {
  /** 中心经度 */
  centerLng: number;
  /** 中心纬度 */
  centerLat: number;
  /** 缩放级别 */
  zoom: number;
  /** 边界范围 */
  bounds: MapBounds;
}

/**
 * 地图筛选参数
 */
export interface MapFilterParams {
  /** 设备组ID列表（筛选这些组及其子组下所有设备，支持多选） */
  groupIds?: string[];
  /** 状态筛选 */
  status?: DeviceStatus[];
  /** 类型筛选 */
  type?: DeviceType[];
  /** 搜索关键词（名称/序列号） */
  keyword?: string;
  /** 视图边界 "minLng,maxLng,minLat,maxLat" */
  bounds?: string;
  /** 页码 */
  page?: number;
  /** 每页条数 */
  pageSize?: number;
  /** 是否启用查询 */
  enabled?: boolean;
}

/**
 * 地图统计数据
 */
export interface MapStats {
  /** 总设备数 */
  total: number;
  /** 各状态数量 */
  statusCount: Record<DeviceStatus, number>;
  /** 总告警数 */
  alarmCount: number;
  /** 各类型数量 */
  typeCount?: Record<DeviceType, number>;
  /** 当前视图内设备数 */
  viewportCount?: number;
}

/**
 * 搜索结果项
 */
export interface DeviceSearchResult {
  /** 设备ID */
  id: string;
  /** 设备名称 */
  name: string;
  /** 序列号 */
  sn: string;
  /** 设备状态 */
  status: DeviceStatus;
  /** 经度 */
  longitude: number;
  /** 纬度 */
  latitude: number;
  /** 设备组名称 */
  groupName?: string;
}

/**
 * 地图配置
 */
export interface MapConfig {
  /** 默认中心点 [lng, lat] */
  defaultCenter: [number, number];
  /** 默认缩放级别 */
  defaultZoom: number;
  /** 最小缩放级别 */
  minZoom: number;
  /** 最大缩放级别 */
  maxZoom: number;
  /** 瓦片服务地址 */
  tileUrl?: string;
}

/**
 * 设备组树节点
 */
export interface DeviceGroupNode {
  /** 设备组ID */
  id: string;
  /** 设备组名称 */
  name: string;
  /** 父设备组ID */
  parentId: string | null;
  /** 层级 (1-5) */
  level: number;
  /** 子设备组 */
  children?: DeviceGroupNode[];
  /** 设备数量（含子组） */
  deviceCount?: number;
  /** 是否为叶子节点 */
  isLeaf?: boolean;
}

/**
 * GISMap 主组件 Props
 */
export interface GISMapProps {
  /** 设备数据列表 */
  devices?: MapDevice[];
  /** 地图高度 */
  height?: string | number;
  /** 默认中心点 [lng, lat] */
  defaultCenter?: [number, number];
  /** 默认缩放级别 */
  defaultZoom?: number;
  /** 设备点击回调 */
  onDeviceClick?: (device: MapDevice) => void;
  /** 视图变化回调 */
  onViewportChange?: (viewport: MapViewport) => void;
  /** 是否显示统计面板 */
  showStats?: boolean;
  /** 是否显示控制按钮 */
  showControls?: boolean;
  /** 自定义样式 */
  className?: string;
  /** 自定义样式对象 */
  style?: React.CSSProperties;
}

/**
 * GroupTree 组件 Props
 */
export interface GroupTreeProps {
  /** 设备组树数据 */
  data?: DeviceGroupNode[];
  /** 选中的设备组ID列表（支持多选） */
  selectedGroupIds?: string[];
  /** 选中变化回调 */
  onSelect: (groupIds: string[]) => void;
  /** 是否显示搜索 */
  showSearch?: boolean;
  /** 加载状态 */
  loading?: boolean;
}

/**
 * DeviceSearch 组件 Props
 */
export interface DeviceSearchProps {
  /** 搜索关键词 */
  value?: string;
  /** 关键词变化回调 */
  onChange: (keyword: string) => void;
  /** 搜索结果 */
  results?: DeviceSearchResult[];
  /** 结果项点击回调 */
  onResultClick: (result: DeviceSearchResult) => void;
  /** 加载状态 */
  loading?: boolean;
  /** 是否展开结果列表 */
  expanded?: boolean;
}

/**
 * MapPopup 组件 Props
 */
export interface MapPopupProps {
  /** 设备数据 */
  device: MapDevice | null;
  /** 弹窗位置 [x, y] */
  position?: [number, number];
  /** 是否可见 */
  visible?: boolean;
  /** 关闭回调 */
  onClose?: () => void;
}

/**
 * MapStatsPanel 组件 Props
 */
export interface MapStatsPanelProps {
  /** 统计数据 */
  stats?: MapStats;
  /** 是否可见 */
  visible?: boolean;
}

// ============ Backend 类型定义（snake_case） ============

/**
 * 后端设备地理信息
 */
export interface BackendDeviceGeo {
  id: string;
  name: string;
  sn: string;
  longitude: number;
  latitude: number;
  status: string;
  type?: string;
  group_id: string;
  group_name?: string;
  address?: string;
  alarm_count?: number;
}

/**
 * 后端设备聚合
 */
export interface BackendDeviceCluster {
  id: string;
  longitude: number;
  latitude: number;
  count: number;
  status_count: Record<string, number>;
  alarm_count: number;
  bounds?: {
    min_lng: number;
    max_lng: number;
    min_lat: number;
    max_lat: number;
  };
}

/**
 * 后端地图统计
 */
export interface BackendMapStats {
  total: number;
  status_count: Record<string, number>;
  alarm_count: number;
  type_count?: Record<string, number>;
  viewport_count?: number;
}

/**
 * 后端搜索结果
 */
export interface BackendSearchResult {
  id: string;
  name: string;
  sn: string;
  status: string;
  longitude: number;
  latitude: number;
  group_name?: string;
}
