/**
 * GIS 地图视图页面
 * 完全按照 UI 原型图 GISMap_UI_Design_Main.svg 实现
 * 使用真实 API 接口获取数据
 */
import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import { Checkbox, Spin, Empty, message } from 'antd';
import { SearchOutlined, PlusOutlined, MinusOutlined } from '@ant-design/icons';
import GISMap, { MAP_CONFIG } from '@/components/GISMap';
import type { MapDevice, DeviceGroupNode, MapBounds, DeviceGeo, DeviceSearchResult } from '@/types/map';
import type { Domain } from '@/types/topology';
import { useThemeToken } from '@/hooks/useThemeToken';
import {
  useDomainTree,
  useMapDevicesGeo,
  useMapStats,
  useMapDeviceSearch,
} from '@/hooks/api/useTopology';

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

export default function GISMapView() {
  const token = useThemeToken();

  // ========== 状态管理 ==========

  // 设备组搜索
  const [groupSearchValue, setGroupSearchValue] = useState('');
  // 选中的设备组 ID 列表（支持多选）
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);
  // 展开的设备组 ID 列表
  const [expandedGroupIds, setExpandedGroupIds] = useState<string[]>([]);

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

  // 设备搜索
  const [deviceSearchValue, setDeviceSearchValue] = useState('');
  const [deviceSearchResults, setDeviceSearchResults] = useState<DeviceSearchResult[]>([]);
  const [deviceSearchExpanded, setDeviceSearchExpanded] = useState(false);

  // 地图组件引用
  const mapRef = useRef<{ highlightAndFlyTo: (device: MapDevice) => void }>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const searchContainerRef = useRef<HTMLDivElement>(null);

  // ========== API Hooks ==========

  // 获取设备组树
  const { data: domainTree, isLoading: isLoadingTree } = useDomainTree();

  // 获取设备地理数据
  const filterParams = useMemo(() => {
    const statusList: ('onlineActive' | 'onlineInactive' | 'offline')[] = [
      ...(statusFilter.onlineActive ? ['onlineActive' as const] : []),
      ...(statusFilter.onlineInactive ? ['onlineInactive' as const] : []),
      ...(statusFilter.offline ? ['offline' as const] : []),
    ];
    return {
      groupIds: selectedGroupIds.length > 0 ? selectedGroupIds : undefined,
      // 如果三个状态都被选中（默认情况），不传 status 参数让后端返回全部
      // 如果部分被选中，传对应的状态
      // 如果都没选中，传空数组表示不查询任何设备
      status: statusList.length === 3 ? undefined : statusList,
      enabled: true,
      pageSize: 10000, // 获取大量数据
    };
  }, [selectedGroupIds, statusFilter]);

  const { data: devicesGeoData, isLoading: isLoadingDevices } = useMapDevicesGeo(filterParams);

  // Debug: 输出 filterParams 和结果
  useEffect(() => {
    console.log('[GISMapView] filterParams:', filterParams);
    console.log('[GISMapView] devicesGeoData:', devicesGeoData);
  }, [filterParams, devicesGeoData]);

  // 获取地图统计数据
  const { data: mapStatsData, isLoading: isLoadingStats } = useMapStats({
    groupIds: selectedGroupIds.length > 0 ? selectedGroupIds : undefined,
  });

  // 设备搜索 hook
  const { data: searchResults, isLoading: isSearching } = useMapDeviceSearch(deviceSearchValue);

  // ========== 数据转换 ==========

  // 设备组树（转换格式）
  const groupTree: DeviceGroupNode[] = useMemo(() => {
    if (!domainTree?.length) return [];
    return domainTree.map(domainToGroupNode);
  }, [domainTree]);

  // 设备列表（转换为 MapDevice 格式）
  const mapDevices: MapDevice[] = useMemo(() => {
    if (!devicesGeoData?.items?.length) return [];
    return devicesGeoData.items.map(deviceGeoToMapDevice);
  }, [devicesGeoData]);

  // 统计数据
  const stats = useMemo(() => {
    if (!mapStatsData) {
      return {
        total: 0,
        onlineActive: 0,
        onlineInactive: 0,
        offline: 0,
        center: undefined,
      };
    }
    return {
      total: mapStatsData.total,
      onlineActive: mapStatsData.statusCount?.onlineActive ?? 0,
      onlineInactive: mapStatsData.statusCount?.onlineInactive ?? 0,
      offline: mapStatsData.statusCount?.offline ?? 0,
      center: mapStatsData.center,
    };
  }, [mapStatsData]);

  // ========== 搜索处理 ==========

  // 处理设备搜索
  const handleDeviceSearch = useCallback((value: string) => {
    setDeviceSearchValue(value);
    if (value.length >= 2) {
      setDeviceSearchExpanded(true);
    } else {
      setDeviceSearchResults([]);
      setDeviceSearchExpanded(false);
    }
  }, []);

  // 搜索结果更新（根据状态过滤）
  useEffect(() => {
    if (searchResults) {
      // 根据状态筛选过滤搜索结果
      const filtered = searchResults.filter((device) => {
        if (device.status === 'onlineActive' && !statusFilter.onlineActive) return false;
        if (device.status === 'onlineInactive' && !statusFilter.onlineInactive) return false;
        if (device.status === 'offline' && !statusFilter.offline) return false;
        return true;
      });
      setDeviceSearchResults(filtered);
    }
  }, [searchResults, statusFilter]);

  // ========== 设备组树处理 ==========

  // 获取所有子节点 ID
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
    }
  }, [groupSearchValue, filteredGroupTree]);

  // 初始化：选中并展开根节点
  useEffect(() => {
    if (groupTree.length > 0 && selectedGroupIds.length === 0) {
      // 默认选中所有根节点
      const rootIds = groupTree.map((node) => node.id);
      setSelectedGroupIds(rootIds);
      setExpandedGroupIds(rootIds);
    }
  }, [groupTree]);

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
    border: '2px solid rgb(217, 217, 217)',
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
    padding: '10px 16px',
    borderTop: '1px solid #F0F0F0',
    textAlign: 'center',
    background: '#FAFAFA',
    flexShrink: 0,
  };

  const searchResultItemStyle = (isHighlighted: boolean): React.CSSProperties => ({
    padding: '12px 16px',
    cursor: 'pointer',
    background: isHighlighted ? '#E6F7FF' : 'transparent',
    transition: 'background 0.15s',
  });

  // ========== 渲染 ==========

  const isLoading = isLoadingTree || isLoadingDevices || isLoadingStats;

  return (
    <div style={{ display: 'flex', width: '100%', height: '100%', background: '#F0F2F5' }}>
      {/* 左侧筛选面板 */}
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
          {isLoadingTree ? (
            <div style={{ padding: 20, textAlign: 'center' }}>
              <Spin size="small" />
            </div>
          ) : filteredGroupTree.length === 0 ? (
            <Empty description="暂无设备组" style={{ padding: 20 }} />
          ) : (
            filteredGroupTree.map((node) => renderGroupNode(node))
          )}
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
            <span style={{ fontSize: 14, color: '#262626' }}>在线激活</span>
            <span style={{ marginLeft: 'auto', fontSize: 14, color: '#52C41A' }}>
              {stats.onlineActive.toLocaleString()}
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
            <span style={{ fontSize: 14, color: '#262626' }}>在线未激活</span>
            <span style={{ marginLeft: 'auto', fontSize: 14, color: '#FAAD14' }}>
              {stats.onlineInactive.toLocaleString()}
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
            <span style={{ fontSize: 14, color: '#262626' }}>离线</span>
            <span style={{ marginLeft: 'auto', fontSize: 14, color: '#b60808' }}>
              {stats.offline.toLocaleString()}
            </span>
          </div>
        </div>

        {/* 分隔线 */}
        <div style={dividerStyle} />

        {/* 图例 */}
        <div style={sectionTitleStyle}>图例</div>
        <div style={{ padding: '0 20px' }}>
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
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>在线激活</span>
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
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>在线未激活</span>
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
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>离线设备</span>
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
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>设备聚合</span>
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
              <span style={{ marginLeft: 8, fontSize: 12, color: '#595959' }}>告警数量</span>
            </div>
          </div>
        </div>
      </div>

      {/* 地图区域 */}
      <div style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
        {/* 地图组件 */}
        {isLoading ? (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
            <Spin size="large" tip="加载设备数据..." />
          </div>
        ) : (
          <GISMap
            ref={mapRef}
            devices={mapDevices}
            height="100%"
            defaultCenter={stats.center ? [stats.center.lng, stats.center.lat] : [28.221, -14.607]}
            defaultZoom={6}
            showStats={false}
            showControls={false}
            tileUrl={MAP_CONFIG.osmTileUrl}
            onDeviceClick={(device) => {
              console.log('Device clicked:', device);
            }}
            onMapClick={() => {
              // 点击地图时收起搜索结果面板
              setDeviceSearchExpanded(false);
            }}
          />
        )}

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
                {isSearching ? (
                  <div style={{ padding: 24, textAlign: 'center' }}>
                    <Spin size="small" />
                  </div>
                ) : deviceSearchResults.length === 0 ? (
                  <div style={{ padding: 24, textAlign: 'center', color: '#8C8C8C' }}>
                    🔍
                    <br />
                    <span style={{ fontSize: 12 }}>未找到匹配的设备</span>
                    <br />
                    <span style={{ fontSize: 11, color: '#BFBFBF' }}>请尝试其他关键词</span>
                  </div>
                ) : (
                  <>
                    {/* 可滚动的结果列表 */}
                    <div style={searchResultsListStyle}>
                      {deviceSearchResults.map((result, index) => {
                        // onlineActive 和 onlineInactive 都算在线
                        const isOnline = result.status === 'onlineActive' || result.status === 'onlineInactive';
                        // 状态颜色：在线激活=绿色，在线未激活=黄色，离线=红色
                        const statusColor = result.status === 'onlineActive'
                          ? 'linear-gradient(180deg, #73D13D 0%, #52C41A 100%)'
                          : result.status === 'onlineInactive'
                            ? 'linear-gradient(180deg, #FFC53D 0%, #FAAD14 100%)'
                            : '#b60808';
                        const statusText = result.status === 'onlineActive'
                          ? '🟢 在线激活'
                          : result.status === 'onlineInactive'
                            ? '🟡 在线未激活'
                            : '🔴 离线';
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
                                  background: statusColor,
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
                        共找到 {deviceSearchResults.length} 个结果
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

        {/* 统计面板 */}
        <div style={statsPanelStyle}>
          <div style={{ fontSize: 14, fontWeight: 600, color: '#262626', marginBottom: 8 }}>
            设备统计
          </div>
          <div style={{ borderTop: '1px solid #F0F0F0', margin: '8px 0 16px' }} />

          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontSize: 12, color: '#8C8C8C' }}>总设备</span>
            <span style={{ fontSize: 22, fontWeight: 700, color: '#262626' }}>
              {stats.total.toLocaleString()}
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
                  background: 'linear-gradient(180deg, #73D13D 0%, #52C41A 100%)',
                  marginRight: 8,
                }}
              />
              <span style={{ fontSize: 12, color: '#595959' }}>在线激活</span>
            </div>
            <span style={{ fontSize: 14, fontWeight: 600, color: '#52C41A' }}>
              {stats.onlineActive.toLocaleString()}
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
                  background: 'linear-gradient(180deg, #FFC53D 0%, #FAAD14 100%)',
                  marginRight: 8,
                }}
              />
              <span style={{ fontSize: 12, color: '#595959' }}>在线未激活</span>
            </div>
            <span style={{ fontSize: 14, fontWeight: 600, color: '#FAAD14' }}>
              {stats.onlineInactive.toLocaleString()}
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
              {stats.offline.toLocaleString()}
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
          OMC GIS Map - Topology View v2.0 (Real API)
        </div>
      </div>
    </div>
  );
}
