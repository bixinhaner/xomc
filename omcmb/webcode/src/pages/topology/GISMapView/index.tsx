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
import { useState, useMemo, useRef, useEffect } from 'react';
import { useIntl } from 'react-intl';
import { Checkbox, Spin, Empty, Collapse, Input } from 'antd';
import { SearchOutlined, PlusOutlined, MinusOutlined, CaretDownOutlined } from '@ant-design/icons';
import GISMap from '@/components/GISMap';
import { MAP_CONFIG } from '@/components/GISMap/constants';
import type { GISMapRef } from '@/components/GISMap';
import type { MapDevice, DeviceGroupNode, DeviceGeo } from '@core/types/map';
import type { Domain } from '@core/types/topology';
import { useThemeToken } from '@/hooks/useThemeToken';
import MapStatsPanel from '@/components/GISMap/MapStatsPanel';
import {
  useDomainTree,
  useMapDevicesGeo,
  useMapStats,
} from '@core/hooks/api/useTopology';
import { useDeviceSearch } from '@core/hooks/useDeviceSearch';
import { topologyApi } from '@core/services/api/topologyApi';
import { SPACING, RADIUS, SHADOWS, COLORS, transitionString, DURATION, EASING } from './styles';
import './animations.css';

/**
 * 将 DeviceGeo 转换为 MapDevice
 */
function deviceGeoToMapDevice(device: DeviceGeo): MapDevice {
  return {
    id: device.id,
    lat: device.latitude,
    lng: device.longitude,
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
  };
}

/**
 * 将 Domain 树转换为 DeviceGroupNode 树
 */
function domainToGroupNode(domain: Domain): DeviceGroupNode {
  return {
    id: domain.id,
    name: domain.name,
    parentId: domain.parentId ?? null,
    level: domain.level,
    children: domain.children?.map(domainToGroupNode),
    deviceCount: domain.deviceCount,
    isLeaf: !domain.children?.length,
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

export default function GISMapView() {
  const token = useThemeToken();
  const intl = useIntl();

  // ========== 状态管理 ==========

  // 设备组搜索
  const [groupSearchValue, setGroupSearchValue] = useState('');
  // 选中的设备组 ID 列表（支持多选）
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);
  // 展开的设备组 ID 列表
  const [expandedGroupIds, setExpandedGroupIds] = useState<string[]>([]);
  // 是否已完成初始化（用于控制 API 请求时机）
  const [isInitialized, setIsInitialized] = useState(false);

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

  // 左侧筛选面板折叠状态（默认全部展开，但图例默认隐藏）
  const [filterPanelActiveKeys, setFilterPanelActiveKeys] = useState<string[]>([
    'deviceGroup',    // 设备组
    'deviceStatus',   // 设备状态
  ]);

  // ========== 显示/隐藏控制配置 ==========
  // 筛选Header显示配置（默认隐藏）
  const SHOW_FILTER_HEADER = false;
  // 图例模块显示配置（默认隐藏）
  const SHOW_LEGEND = false;

  // 地图组件引用
  const mapRef = useRef<GISMapRef>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const searchContainerRef = useRef<HTMLDivElement>(null);

  // ========== API Hooks ==========

  // 获取设备组树
  const { data: domainTree, isLoading: isLoadingTree } = useDomainTree();

  // ========== 数据转换 ==========

  // 设备组树（转换格式）- 必须在 allGroupIds 之前定义
  const groupTree: DeviceGroupNode[] = useMemo(() => {
    if (!domainTree?.length) return [];
    return domainTree.map(domainToGroupNode);
  }, [domainTree]);

  // 计算所有组的 ID 集合（用于判断是否选中了"全部"）
  const allGroupIds = useMemo(() => {
    return new Set(groupTree.flatMap((node) => getAllDescendantIds(node)));
  }, [groupTree]);

  // 获取设备地理数据
  const filterParams = useMemo(() => {
    const statusList: ('onlineActive' | 'onlineInactive' | 'offline')[] = [
      ...(statusFilter.onlineActive ? ['onlineActive' as const] : []),
      ...(statusFilter.onlineInactive ? ['onlineInactive' as const] : []),
      ...(statusFilter.offline ? ['offline' as const] : []),
    ];

    // 判断是否选中了所有组
    // 简化逻辑：只要选中的数量等于所有组的数量，就认为选中了所有组
    // 传 undefined 让后端返回所有设备（包括未分组的）
    const isAllSelected = selectedGroupIds.length > 0 &&
      selectedGroupIds.length === allGroupIds.size;

    return {
      // 选中所有组时传 undefined（返回所有设备，包括未分组的）
      // 只选中部分组时传具体的 groupIds
      groupIds: isAllSelected ? undefined : (selectedGroupIds.length > 0 ? selectedGroupIds : undefined),
      // 如果三个状态都被选中（默认情况），不传 status 参数让后端返回全部
      // 如果部分被选中，传对应的状态
      // 如果都没选中，传空数组表示不查询任何设备
      status: statusList.length === 3 ? undefined : statusList,
      // 只有初始化完成后才启用请求，避免在 selectedGroupIds 为空时发送请求
      enabled: isInitialized,
      pageSize: 100, // 获取大量数据
    };
  }, [selectedGroupIds, statusFilter, isInitialized, allGroupIds]);

  const { data: devicesGeoData, isLoading: isLoadingDevices } = useMapDevicesGeo(filterParams);

  // 获取地图统计数据
  const { data: mapStatsData, isLoading: isLoadingStats } = useMapStats({
    groupIds: selectedGroupIds.length > 0 ? selectedGroupIds : undefined,
  });

  // 设备搜索 hook（支持防抖、50 值限制和自动展开）
  const {
    keyword: searchKeyword,
    handleChange: handleSearchInputChange,
    handleClear: handleSearchClear,
    results: searchResults,
    isLoading: isSearching,
    expanded: deviceSearchExpanded,
    setExpanded: setDeviceSearchExpanded,
  } = useDeviceSearch((kw) => topologyApi.searchDevices(kw), {
    minLength: 2,
    maxValues: 50,
    debounce: 300,
    autoExpand: true,
  });

  // ========== 数据转换 ==========

  // 设备列表（转换为 MapDevice 格式）
  const mapDevices: MapDevice[] = useMemo(() => {
    if (!devicesGeoData?.items?.length) return [];
    return devicesGeoData.items.map(deviceGeoToMapDevice);
  }, [devicesGeoData]);

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
  const filteredGroupTree = useMemo(() => {
    if (!groupSearchValue.trim()) {
      return groupTree;
    }

    const searchLower = groupSearchValue.toLowerCase();

    const filterNode = (node: DeviceGroupNode, parentMatch = false): DeviceGroupNode | null => {
      const nameMatch = node.name.toLowerCase().includes(searchLower);
      const idMatch = node.id.toLowerCase().includes(searchLower);
      const selfMatch = nameMatch || idMatch;

      const filteredChildren: DeviceGroupNode[] = [];
      if (node.children) {
        node.children.forEach((child) => {
          const filteredChild = filterNode(child, selfMatch || parentMatch);
          if (filteredChild) {
            filteredChildren.push(filteredChild);
          }
        });
      }

      if (selfMatch || filteredChildren.length > 0) {
        return {
          ...node,
          children: filteredChildren.length > 0 ? filteredChildren : node.children,
        };
      }

      return null;
    };

    const result: DeviceGroupNode[] = [];
    groupTree.forEach((node) => {
      const filtered = filterNode(node);
      if (filtered) {
        result.push(filtered);
      }
    });

    return result;
  }, [groupSearchValue, groupTree]);

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
            {node.name}
          </span>
        </div>

        {isExpanded && !isLeaf && node.children && (
          <div>{node.children.map((child) => renderGroupNode(child, depth + 1))}</div>
        )}
      </div>
    );
  };

  // ========== 样式定义 ==========

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
    margin: `${SPACING.md}px ${SPACING.md}px 0`,
    padding: `${SPACING.md}px ${SPACING.lg}px`,
    background: '#FFF',
    border: '1px solid #E8E8E8',
    borderRadius: RADIUS.md,
    display: 'flex',
    alignItems: 'center',
    gap: SPACING.md,
  };

  const treeContainerStyle: React.CSSProperties = {
    flex: '0 0 auto',
    maxHeight: 280,
    overflow: 'auto',
    padding: `${SPACING.xs}px 0`,
  };

  // ========== Collapse 组件样式 ==========

  // Collapse 容器样式
  const collapseContainerStyle: React.CSSProperties = {
    background: 'transparent',
    border: 'none',
    flex: 1,
    overflow: 'auto',
    padding: `0 ${SPACING.md}px`,
  };

  // Collapse.Panel 头部样式
  const collapseHeaderStyle: React.CSSProperties = {
    fontSize: 13,
    fontWeight: 600,
    color: '#595959',
    padding: `${SPACING.md}px 0`,
    transition: transitionString(['color'], 'fast'),
  };

  // 折叠面板之间的间距
  const collapseItemStyle: React.CSSProperties = {
    marginBottom: SPACING.md,
    borderRadius: RADIUS.md,
    overflow: 'hidden',
    transition: transitionString(['background', 'box-shadow'], 'fast'),
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
    right: 24,
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

  const deviceSearchStyle: React.CSSProperties = {
    position: 'absolute',
    left: 20,
    top: 20,
    width: 320,
    zIndex: 500,
  };

  const searchBoxOuterStyle: React.CSSProperties = {
    background: '#FFF',
    borderRadius: RADIUS.lg,
    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
    border: '2px solid rgb(217, 217, 217)',
    overflow: 'hidden',
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

  return (
    <div style={{ display: 'flex', width: '100%', height: '100%', background: '#F0F2F5' }}>
      {/* 左侧筛选面板 */}
      <div style={leftPanelStyle}>
        {/* Header - 可通过 SHOW_FILTER_HEADER 控制 */}
        {SHOW_FILTER_HEADER && (
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
            <span style={{ fontSize: 18, fontWeight: 600, color: 'var(--color-neutral-800)' }}>
              {intl.formatMessage({ id: 'common.filter' })}
            </span>
          </div>
        )}

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

        {/* 折叠面板：设备状态（调换位置到第一）、设备组（调换位置到第二）、图例（可选显示） */}
        <Collapse
          activeKey={filterPanelActiveKeys}
          onChange={(keys) => setFilterPanelActiveKeys(keys as string[])}
          expandIcon={customExpandIcon}
          bordered={false}
          style={collapseContainerStyle}
          className="gismap-filter-collapse"
          items={[
            // Panel 1: 设备状态（原第2位，现调至第1位）
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
            // Panel 2: 设备组（原第1位，现调至第2位）
            {
              key: 'deviceGroup',
              label: <span style={collapseHeaderStyle}>{intl.formatMessage({ id: 'gis.filter.deviceGroup' })}</span>,
              style: collapseItemStyle,
              children: (
                <>
                  {/* 设备组树 */}
                  <div style={treeContainerStyle}>
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

                  {/* 已选择汇总 */}
                  {selectedGroupIds.length > 0 && (
                    <div
                      style={{
                        margin: '12px 8px 8px', // 与树节点保持一致的左右边距
                        padding: '10px 12px',
                        background: '#E6F7FF', // 蓝色背景
                        borderRadius: 4,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      <span style={{ fontSize: 13, color: '#595959' }}>
                        {intl.formatMessage({ id: 'gis.search.selectedGroups' }, { count: selectedGroupIds.length })}
                      </span>
                    </div>
                  )}
                </>
              ),
            },
            // Panel 3: 图例 - 可通过 SHOW_LEGEND 控制，默认隐藏
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

      {/* 地图区域 */}
      <div style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
        <GISMap
          ref={mapRef}
          devices={mapDevices}
          height="100%"
          defaultCenter={[26, -13] as [number, number]}
          defaultZoom={MAP_CONFIG.defaultZoom}
          showStats={false}
          showControls={false}
          tileUrl={MAP_CONFIG.tileUrl}
          onDeviceClick={undefined}
          onMapClick={() => {
            // 点击地图时收起搜索结果面板
            setDeviceSearchExpanded(false);
          }}
        />

        {/* 设备搜索 */}
        <div ref={searchContainerRef} style={deviceSearchStyle}>
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
                        return (
                          <div
                            key={result.id}
                            style={searchResultItemStyle(index === 0)}
                            onClick={() => {
                              setDeviceSearchExpanded(false);
                              const mapDevice: MapDevice = {
                                id: result.id,
                                lat: result.latitude,
                                lng: result.longitude,
                                name: result.name,
                                status: result.status,
                                sn: result.sn,
                                groupName: result.groupName,
                                address: '',
                                alarmCount: 0,
                                type: undefined,
                              };
                              mapRef.current?.highlightAndFlyToWithCard(mapDevice);
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
              const currentZoom = view?.zoom ?? 6;
              mapRef.current?.flyTo(
                view?.centerLng ?? 28.221,
                view?.centerLat ?? -14.607,
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
              const currentZoom = view?.zoom ?? 6;
              mapRef.current?.flyTo(
                view?.centerLng ?? 28.221,
                view?.centerLat ?? -14.607,
                Math.max(currentZoom - 1, 3)
              );
            }}
          >
            <span style={{ fontSize: 16, fontWeight: 600, color: COLORS.neutral[800] }}>−</span>
          </button>
        </div>

        {/* 统计面板 */}
        <MapStatsPanel stats={stats} visible={false} />

        {/* Footer */}
        <div
          style={{
            position: 'absolute',
            bottom: 8,
            left: '50%',
            transform: 'translateX(-50%)',
            fontSize: 11,
            color: '#BFBFBF',
            width: 'fit-content',
          }}
        >
          OMC GIS Map - Topology View v2.0 (Real API)
        </div>
      </div>
    </div>
  );
}
