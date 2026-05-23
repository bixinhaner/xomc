/**
 * GIS 地图主组件（OpenLayers 实现）
 * @module components/GISMap
 * @description 基于 OpenLayers 的设备地图组件，支持聚合显示、筛选、搜索
 */

import React, { useState, useEffect, useMemo, useCallback, forwardRef, useImperativeHandle } from 'react';
import { Spin, Alert } from 'antd';
import { LoadingOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { useIntl } from 'react-intl';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { GISMapProps, MapDevice, MapViewport, MapStats } from '@core/types/map';
import { MAP_CONFIG } from './constants';
import { useOLMap } from './useOLMap';
import MapPopup from './MapPopup';
import MapControls from './MapControls';
import MapStatsPanel from './MapStatsPanel';
import styles from './styles.module.css';

/**
 * GISMap 组件暴露的方法接口
 */
export interface GISMapRef {
  /** 高亮设备并飞行到指定位置 */
  highlightAndFlyTo: (device: MapDevice) => void;
  /** 飞行到指定坐标 */
  flyTo: (lng: number, lat: number, zoom?: number) => void;
  /** 获取当前视图状态 */
  getViewport: () => MapViewport | null;
}

/**
 * GIS 地图组件
 * 基于 OpenLayers 实现，支持设备标记、聚合、悬浮提示、统计面板
 */
const GISMap = forwardRef<GISMapRef, GISMapProps>(({
  devices = [],
  height = '100%',
  defaultCenter = MAP_CONFIG.defaultCenter,
  defaultZoom = MAP_CONFIG.defaultZoom,
  tileUrl,
  onDeviceClick,
  onViewportChange,
  onMapClick,
  showStats = true,
  showControls = true,
  className,
  style,
}, ref) => {
  const intl = useIntl();
  const token = useThemeToken();
  const [hoveredDevice, setHoveredDevice] = useState<MapDevice | null>(null);
  const [popupPosition, setPopupPosition] = useState<{ x: number; y: number } | null>(null);
  const [, setHighlightedId] = useState<string | null>(null);
  const [viewport, setViewport] = useState<MapViewport | null>(null);
  const [showMetadataTip, setShowMetadataTip] = useState(false);
  // 点击锁定的设备（优先显示，支持复制）
  const [clickedDevice, setClickedDevice] = useState<MapDevice | null>(null);
  const [clickedPosition, setClickedPosition] = useState<{ x: number; y: number } | null>(null);

  // 使用 OpenLayers Hook
  const {
    mapRef,
    mapInstanceRef,
    updateDevices,
    getViewport,
    flyTo,
    clearHighlight,
    isReady,
    updateSize,
    highlightAndSpiderfyIfNeeded,
    metadata,
    metadataLoading,
  } = useOLMap({
    center: defaultCenter,
    zoom: defaultZoom,
    tileUrl,
    onDeviceClick: (device, pixel) => {
      // 点击设备时锁定弹窗
      setClickedDevice(device);
      if (pixel) {
        setClickedPosition({ x: pixel.x, y: pixel.y });
      }
      onDeviceClick?.(device);
    },
    onDeviceHover: (device, pixel) => {
      // hover 只在未锁定时更新
      if (!clickedDevice) {
        setHoveredDevice(device);
        if (device && pixel) {
          setPopupPosition({ x: pixel.x, y: pixel.y });
        } else {
          setPopupPosition(null);
        }
      }
    },
    onViewportChange: (vp) => {
      setViewport(vp);
      onViewportChange?.(vp);
    },
    onMapClick: () => {
      // 点击地图空白时清除锁定
      setClickedDevice(null);
      setClickedPosition(null);
      onMapClick?.();
    },
  });

  // 更新设备数据
  useEffect(() => {
    if (isReady) {
      updateDevices(devices);
    }
  }, [isReady, devices, updateDevices]);

  // 元数据加载完成日志（用于调试）
  useEffect(() => {
    if (metadata) {
      console.log('[GISMap] Metadata loaded:', metadata.region, metadata.bounds);
      // 显示短暂的元数据加载提示
      setShowMetadataTip(true);
      const timer = setTimeout(() => setShowMetadataTip(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [metadata]);

  // 窗口大小变化时更新地图
  useEffect(() => {
    const handleResize = () => {
      updateSize();
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [updateSize]);

  // 缩放控制
  const handleZoomIn = useCallback(() => {
    const map = mapInstanceRef.current;
    if (map) {
      const view = map.getView();
      const currentZoom = view.getZoom() ?? defaultZoom;
      view.animate({
        zoom: currentZoom + 1,
        duration: 300,
      });
    }
  }, [mapInstanceRef, defaultZoom]);

  const handleZoomOut = useCallback(() => {
    const map = mapInstanceRef.current;
    if (map) {
      const view = map.getView();
      const currentZoom = view.getZoom() ?? defaultZoom;
      view.animate({
        zoom: currentZoom - 1,
        duration: 300,
      });
    }
  }, [mapInstanceRef, defaultZoom]);

  // 高亮设备（用于搜索定位）
  const highlightAndFlyTo = useCallback((device: MapDevice) => {
    // 使用新函数：自动展开聚合并显示脉冲效果
    highlightAndSpiderfyIfNeeded(device);
    setHighlightedId(device.id);

    // 5秒后取消高亮（延长时间以便用户查看）
    setTimeout(() => {
      clearHighlight();
      setHighlightedId(null);
    }, 5000);
  }, [highlightAndSpiderfyIfNeeded, clearHighlight]);

  // 暴露方法给父组件
  useImperativeHandle(ref, () => ({
    highlightAndFlyTo,
    flyTo,
    getViewport,
  }), [highlightAndFlyTo, flyTo, getViewport]);

  // 计算统计数据
  const stats = useMemo<MapStats>(() => {
    const statusCount = devices.reduce(
      (acc, d) => {
        acc[d.status] = (acc[d.status] || 0) + 1;
        return acc;
      },
      {} as Record<string, number>
    );

    const alarmCount = devices.reduce((sum, d) => sum + (d.alarmCount || 0), 0);

    return {
      total: devices.length,
      statusCount: statusCount as MapStats['statusCount'],
      alarmCount,
    };
  }, [devices]);

  // 容器样式
  const containerStyle: React.CSSProperties = {
    position: 'relative',
    width: '100%',
    height,
    background: token.colorBgLayout,
    overflow: 'visible', // 允许悬浮提示显示在容器外
    borderRadius: 8,
    ...style,
  };

  const mapContainerStyle: React.CSSProperties = {
    width: '100%',
    height: '100%',
    overflow: 'hidden', // 地图容器内部保持裁剪
    borderRadius: 8,
  };

  const loadingStyle: React.CSSProperties = {
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    zIndex: 1000,
  };

  // 加载提示文本
  const getLoadingText = () => {
    if (metadataLoading) {
      return intl.formatMessage({ id: 'map.loadingConfig' });
    }
    return intl.formatMessage({ id: 'map.initializing' });
  };

  // 元数据信息提示
  const metadataAlert = useMemo(() => {
    if (!metadata || !showMetadataTip) return null;
    return {
      message: intl.formatMessage(
        { id: 'map.loaded' },
        { name: metadata.name, region: metadata.region }
      ),
      description: intl.formatMessage(
        { id: 'map.metadata' },
        {
          min: metadata.zoom.min,
          max: metadata.zoom.max,
          lon: metadata.center.lon,
          lat: metadata.center.lat
        }
      ),
    };
  }, [metadata, showMetadataTip, intl]);

  return (
    <div
      style={containerStyle}
      className={`gis-map-container ${styles.gisMapContainer} ${className || ''}`}
    >
      {/* 地图容器 */}
      <div ref={mapRef} style={mapContainerStyle} />

      {/* 加载状态 */}
      {!isReady && (
        <div style={loadingStyle}>
          <Spin size="large" indicator={<LoadingOutlined spin />} />
          <div style={{ marginTop: 16, color: token.colorTextSecondary }}>
            {getLoadingText()}
          </div>
        </div>
      )}

      {/* 元数据加载提示 */}
      {isReady && metadataAlert && (
        <div style={{
          position: 'absolute',
          top: 16,
          left: '50%',
          transform: 'translateX(-50%)',
          zIndex: 1000,
          maxWidth: '80%',
        }}>
          <Alert
            message={metadataAlert.message}
            description={metadataAlert.description}
            type="info"
            icon={<InfoCircleOutlined />}
            showIcon
            closable
            onClose={() => setShowMetadataTip(false)}
            style={{ fontSize: 12 }}
          />
        </div>
      )}

      {/* 缩放控制 */}
      {showControls && isReady && (
        <MapControls
          onZoomIn={handleZoomIn}
          onZoomOut={handleZoomOut}
          zoomInDisabled={viewport?.zoom !== undefined && viewport.zoom >= MAP_CONFIG.maxZoom}
          zoomOutDisabled={viewport?.zoom !== undefined && viewport.zoom <= MAP_CONFIG.minZoom}
        />
      )}

      {/* 统计面板 */}
      {showStats && isReady && <MapStatsPanel stats={stats} visible />}

      {/* 悬浮提示：优先显示点击锁定的设备，否则显示 hover 的设备 */}
      {(clickedDevice || hoveredDevice) && (clickedPosition || popupPosition) && (
        <MapPopup
          device={clickedDevice ?? hoveredDevice!}
          visible
          position={clickedPosition ?? popupPosition!}
          onClose={() => {
            setClickedDevice(null);
            setClickedPosition(null);
            setHoveredDevice(null);
            setPopupPosition(null);
          }}
        />
      )}
    </div>
  );
});

// 导出组件和类型
export default GISMap;
export type { GISMapProps, MapDevice, MapViewport, MapStats } from '@core/types/map';

// 导出子组件（可选）
export { default as MapPopup } from './MapPopup';
export { default as MapControls } from './MapControls';
export { default as MapStatsPanel } from './MapStatsPanel';
export { default as GroupTree } from './GroupTree';
export { default as DeviceSearch } from './DeviceSearch';

// 导出 Hook
export { useOLMap } from './useOLMap';

// 导出常量和工具
export * from './constants';
export * from './styleUtils';
