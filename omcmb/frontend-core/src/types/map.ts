/**
 * GIS 地图相关类型定义
 * @module types/map
 */

/**
 * 设备状态枚举（地图显示用）
 * - onlineActive: 在线激活
 * - onlineInactive: 在线未激活（已注册/配置中）
 * - offline: 离线
 */
export type DeviceStatus = 'onlineActive' | 'onlineInactive' | 'offline';

/**
 * 原始设备状态（后端数据库中的状态）
 */
export type RawDeviceStatus =
  | 'discovered'
  | 'registered'
  | 'provisioning'
  | 'active'
  | 'maintenance'
  | 'offline'
  | 'decommissioned';

/**
 * 将原始状态转换为显示状态
 */
export function toDisplayStatus(status: RawDeviceStatus): DeviceStatus {
  switch (status) {
    case 'active':
      return 'onlineActive';
    case 'registered':
    case 'provisioning':
      return 'onlineInactive';
    default:
      return 'offline';
  }
}

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
  /** IP 地址 */
  ip_address?: string;
  /** MAC 地址 */
  mac?: string;
  /** PCI（物理小区标识） */
  pci?: string;
  /** 运维自定义设备名称 */
  device_name?: string;
  /** 当前接入 UE 数 */
  ueCount?: number;
  /** 最高告警级别 1=Critical,2=Major,3=Minor,4=Warning; null/undefined=无告警 */
  highestAlarmSeverity?: number | null;
  /** 最高级别的告警数量 */
  highestSeverityAlarmCount?: number;
}

/** 设备运行时参数解析出的天线扇区。 */
export interface AntennaSector {
  number: number;
  cellId?: string;
  antennaHeight?: number;
  mechanicalDowntilt?: number;
  electronicDowntilt?: string;
  verticalBeamwidth?: number;
  horizontalBeamwidth?: number;
  azimuth?: number;
  nearRadiusMeters?: number;
  farRadiusMeters?: number;
  fieldSources: Record<string, string>;
  directionAvailable: boolean;
  coverageAvailable: boolean;
  missingFields: string[];
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
  longitude: number | null;
  /** 纬度 (北纬为正) */
  latitude: number | null;
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
  /** IP 地址 */
  ip_address?: string;
  /** MAC 地址 */
  mac?: string;
  /** PCI（物理小区标识） */
  pci?: string;
  /** 运维自定义设备名称 */
  device_name?: string;
  /** 当前接入 UE 数 */
  ueCount?: number;
  /** 最高告警级别 */
  highestAlarmSeverity?: number | null;
  /** 最高级别的告警数量 */
  highestSeverityAlarmCount?: number;
}

/**
 * 设备聚合项
 */
export interface DeviceCluster {
  /** 聚合ID */
  id: string;
  /** 聚合中心点经度 */
  longitude: number | null;
  /** 聚合中心点纬度 */
  latitude: number | null;
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
  /** UE 数上限过滤：0 = 只返回 UE=0 的基站 */
  ueCountMax?: number;
}

/** 设备地理查询结果及其完整性信息。 */
export interface MapGeoResponse {
  items: DeviceGeo[];
  total: number;
  hasMore: boolean;
  complete: boolean;
  coordinateCount: number;
}

/**
 * 地图统计数据
 */
export interface MapStats {
  /** 总设备数 */
  total: number;
  /** 各状态数量 */
  statusCount: {
    onlineActive: number;
    onlineInactive: number;
    offline: number;
  };
  /** 总告警数 */
  alarmCount: number;
  /** 各类型数量 */
  typeCount?: Record<DeviceType, number>;
  /** 当前视图内设备数 */
  viewportCount?: number;
  /** 地图中心点（设备经纬度平均值） */
  center?: GeoCenter;
  /** UE 数为 0 的基站数 */
  ueZeroCount?: number;
}

/**
 * 地理中心点
 */
export interface GeoCenter {
  /** 纬度 */
  lat: number;
  /** 经度 */
  lng: number;
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
  longitude: number | null;
  /** 纬度 */
  latitude: number | null;
  /** 设备组ID */
  groupId?: string;
  /** 设备组名称 */
  groupName?: string;
  /** IP 地址 */
  ip_address?: string;
  /** MAC 地址 */
  mac?: string;
  /** PCI（物理小区标识） */
  pci?: string;
  /** 运维自定义设备名称 */
  device_name?: string;
  /** 当前接入 UE 数 */
  ueCount?: number;
  /** 最高告警级别 */
  highestAlarmSeverity?: number | null;
  /** 最高级别的告警数量 */
  highestSeverityAlarmCount?: number;
  /** 当前活跃告警数 */
  alarmCount?: number;
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
  /** 设备组多语言名称 */
  nameI18n?: Record<string, string>;
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
  /** 搜索结果设备（独立显示，不受主设备列表限制） */
  searchResultDevice?: MapDevice | null;
  /** 当前选中的设备，用于展示其天线扇区。 */
  selectedDevice?: MapDevice | null;
  /** 当前选中设备的运行时天线扇区。 */
  antennaSectors?: AntennaSector[];
  /** 天线编辑值变更时更新地图中的临时覆盖预览。 */
  onAntennaPreviewChange?: (sectorNumber: number, field: 'azimuth' | 'mechanicalDowntilt', value: number | null) => void;
  /** 放弃编辑值并恢复地图中的原始覆盖范围。 */
  onAntennaCancel?: () => void;
  /** 提交设备天线参数设置任务。 */
  onAntennaSave?: (sectorNumber: number) => Promise<unknown>;
  /** 天线参数设置任务是否正在提交。 */
  antennaSaving?: boolean;
  /** 地图高度 */
  height?: string | number;
  /** 默认中心点 [lng, lat] */
  defaultCenter?: [number, number];
  /** 默认缩放级别 */
  defaultZoom?: number;
  /** 瓦片服务地址（如 OSM 瓦片 URL） */
  tileUrl?: string;
  /** 设备点击回调 */
  onDeviceClick?: (device: MapDevice) => void;
  /** 地图点击回调（点击任意位置时触发，包括设备和空白区域） */
  onMapClick?: () => void;
  /**
   * 大聚合点点击回调（用于弹出设备列表）。
   * TODO(P2): 当前未接入，bindMapEvents 路由逻辑待实现后再启用。
   * 参考：docs/design/gis-map-optimization-plan-20260629.md §3.3.2
   */
  onClusterShowList?: (devices: MapDevice[], pixel: { x: number; y: number }) => void;
  /** 视图变化回调 */
  onViewportChange?: (viewport: MapViewport) => void;
  /** 是否显示统计面板 */
  showStats?: boolean;
  /** 是否显示控制按钮 */
  showControls?: boolean;
  /** 是否显示元数据加载提示（默认 true，仪表板小地图可设为 false） */
  showMetadataTip?: boolean;
  /** 自定义样式 */
  className?: string;
  /** 自定义样式对象 */
  style?: React.CSSProperties;
  /** GIS 卡片告警区域点击回调（G-09）：跳转到该设备的当前告警页 */
  onAlarmClick?: (sn: string) => void;
}

/**
 * GISMap 组件暴露的方法接口（通过 ref）
 */
export interface GISMapRef {
  /** 高亮设备并飞行到指定位置（以最大放大程度显示） */
  highlightAndFlyTo: (device: MapDevice) => void;
  /**
   * 高亮设备并飞行到指定位置，同时显示该设备的卡片
   * 使用场景：搜索定位时，用户希望看到目标设备的详细信息
   */
  highlightAndFlyToWithCard: (device: MapDevice, options?: {
    /** 是否自动关闭旧卡片，默认 true */
    autoCloseOldCard?: boolean;
    /** 动画模式（默认 progressive） */
    animationMode?: 'progressive' | 'smooth' | 'fast' | 'direct';
  }) => void;
  /** 飞行到指定坐标 */
  flyTo: (lng: number, lat: number, zoom?: number, options?: {
    progressive?: boolean;
    maxZoom?: number;
    onComplete?: () => void;
  }) => void;
  /** 获取当前视图状态 */
  getViewport: () => MapViewport | null;
  /** 关闭当前锁定的卡片 */
  closeClickedCard: () => void;
  /**
   * 动态调整瓦片并发上限（0 = 暂停队列，正常值为 3）
   * 搜索时调低，为 API 请求让出连接；搜索完成后恢复
   */
  setTileConcurrency: (n: number) => void;
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
  longitude: number | null;
  latitude: number | null;
  status: string;
  type?: string;
  group_id: string;
  group_name?: string;
  address?: string;
  alarm_count?: number;
  /** IP 地址 */
  ip_address?: string;
  /** MAC 地址 */
  mac?: string;
  /** PCI（物理小区标识） */
  pci?: string;
  /** 运维自定义设备名称 */
  device_name?: string;  /** 当前接入 UE 数 */
  ue_count?: number;
  /** 最高告警级别 */
  highest_alarm_severity?: number | null;
  /** 最高级别的告警数量 */
  highest_severity_alarm_count?: number;
}

/**
 * 后端设备聚合
 */
export interface BackendDeviceCluster {
  id: string;
  longitude: number | null;
  latitude: number | null;
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
  status_count: {
    onlineActive: number;
    onlineInactive: number;
    offline: number;
  };
  alarm_count: number;
  type_count?: Record<string, number>;
  viewport_count?: number;
  /** 地图中心点（设备经纬度平均值） */
  center?: {
    lat: number;
    lng: number;
  };
  /** UE 数为 0 的基站数 */
  ue_zero_count?: number;
}

/**
 * 后端搜索结果
 */
export interface BackendSearchResult {
  id: string;
  name: string;
  sn: string;
  status: string;
  longitude: number | null;
  latitude: number | null;
  group_id?: string;
  group_name?: string;
  /** IP 地址 */
  ip_address?: string;
  /** MAC 地址 */
  mac?: string;
  /** PCI（物理小区标识） */
  pci?: string;
  /** 运维自定义设备名称 */
  device_name?: string;
  /** 当前接入 UE 数 */
  ue_count?: number;
  /** 最高告警级别 */
  highest_alarm_severity?: number | null;
  /** 最高级别的告警数量 */
  highest_severity_alarm_count?: number;
  /** 当前活跃告警数 */
  alarm_count?: number;
}
