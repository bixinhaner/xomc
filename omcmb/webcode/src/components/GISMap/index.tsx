/**
 * GIS 地图主组件（OpenLayers 实现）
 * @module components/GISMap
 * @description 基于 OpenLayers 的设备地图组件，支持聚合显示、筛选、搜索
 */

import React, { useState, useEffect, useMemo, useCallback, forwardRef, useImperativeHandle, useRef } from 'react';
import { Spin, Alert } from 'antd';
import { LoadingOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { useIntl } from 'react-intl';
import { fromLonLat } from 'ol/proj';
import { offset as offsetCoordinate } from 'ol/sphere';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { GISMapProps, MapDevice, MapViewport, MapStats, MapBounds } from '@core/types/map';
import { MAP_CONFIG, ANIMATION_CONFIG } from './constants';
import { useOLMap } from './useOLMap';
import { useGeofenceLayer } from './useGeofenceLayer';
import MapPopup from './MapPopup';
import MapControls from './MapControls';
import MapStatsPanel from './MapStatsPanel';
import styles from './styles.module.css';
import { resolveAntennaSectorRenderMode } from './antennaSectorRender';

/**
 * 高亮并显示卡片的配置选项
 */
interface HighlightWithCardOptions {
  /** 是否自动关闭旧卡片，默认 true */
  autoCloseOldCard?: boolean;
  /**
   * 动画策略，默认 'progressive'
   * - 'progressive': 渐进式缩放（国家→省→市→区→街道），视觉效果最佳
   * - 'smooth': 直接平滑动画到目标位置
   * - 'direct': 直接跳转，无动画
   * - 'fast': 快速单阶段动画
   */
  animationMode?: 'progressive' | 'smooth' | 'direct' | 'fast';
}

/**
 * GISMap 组件暴露的方法接口
 */
interface GISMapRef {
  /** 高亮设备并飞行到指定位置 */
  highlightAndFlyTo: (device: MapDevice) => void;
  /**
   * 高亮设备并飞行到指定位置，同时显示该设备的卡片
   *
   * 使用场景：搜索定位时，用户希望看到目标设备的详细信息
   * - 清除旧的高亮和卡片
   * - 飞行到目标设备
   * - 显示该设备的卡片（方便复制信息）
   *
   * @example
   * mapRef.current?.highlightAndFlyToWithCard(searchResult);
   */
  highlightAndFlyToWithCard: (device: MapDevice, options?: HighlightWithCardOptions) => void;
  /** 飞行到指定坐标 */
  flyTo: (lng: number, lat: number, zoom?: number, options?: {
    /** 是否使用渐进式缩放动画（默认根据配置决定） */
    progressive?: boolean;
    /** 瓦片服务最大zoom（用于限制动画范围） */
    maxZoom?: number;
    /** 动画完成回调 */
    onComplete?: () => void;
  }) => void;
  /** 获取当前视图状态 */
  getViewport: () => MapViewport | null;
  /** 自适应显示指定包围盒（OL view.fit，自动计算 zoom，带 padding） */
  fitBounds: (bounds: MapBounds, options?: { duration?: number }) => void;
  /** 清除所有设备数据（筛选条件变化时调用，彻底重置） */
  clearDevices: () => void;
  /** 关闭当前锁定的卡片 */
  closeClickedCard: () => void;
  /**
   * 动态调整瓦片并发上限（0 = 暂停队列，正常值为 3）
   * 搜索时调低，为 API 请求让出连接；搜索完成后恢复
   */
  setTileConcurrency: (n: number) => void;
  /** 开启测距模式 */
  startMeasure: () => void;
  /** 退出测距模式并清除折线 */
  stopMeasure: () => void;
  /** 开启 Polygon 围栏绘制，并停止测距 */
  startGeofencePolygonDraw: () => void;
  /** 停止围栏绘制并清除临时图形 */
  stopGeofenceDraw: () => void;
  /** 自适应显示指定围栏 */
  fitGeofence: (id: string) => boolean;
}

/**
 * GIS 地图组件
 * 基于 OpenLayers 实现，支持设备标记、聚合、悬浮提示、统计面板
 */
const GISMap = forwardRef<GISMapRef, GISMapProps>(({
  devices = [],
  geofences = [],
  selectedGeofenceId,
  onGeofenceClick,
  onGeofenceDrawComplete,
  searchResultDevice = null,
  selectedDevice = null,
  antennaSectors = [],
  height = '100%',
  defaultCenter = MAP_CONFIG.defaultCenter,
  defaultZoom = MAP_CONFIG.defaultZoom,
  tileUrl,
  onDeviceClick,
  onViewportChange,
  onMapClick,
  onClusterShowList: _onClusterShowList,
  showStats = true,
  showControls = true,
  showMetadataTip = true,
  className,
  style,
  onAlarmClick,
  onAntennaPreviewChange,
  onAntennaCancel,
  onAntennaSave,
  antennaSaving,
}, ref) => {
  const intl = useIntl();
  const token = useThemeToken();

  // 定时器 refs，用于组件卸载时清理
  const highlightTimerRef = useRef<NodeJS.Timeout | null>(null);
  const highlightWithCardTimerRef = useRef<NodeJS.Timeout | null>(null);
  const highlightWithCardInnerTimerRef = useRef<NodeJS.Timeout | null>(null);
  // 记录最后一次已触发定位的设备对象引用，避免同一次 React 渲染被反复重启动画
  // 使用对象引用而非 id：用户重新点击同一设备时会创建新对象，可正常再次定位
  const lastHighlightedIdRef = useRef<MapDevice | null>(null);

  const [hoveredDevice, setHoveredDevice] = useState<MapDevice | null>(null);
  const [popupPosition, setPopupPosition] = useState<{ x: number; y: number } | null>(null);
  const [viewport, setViewport] = useState<MapViewport | null>(null);
  const [shouldShowMetadataAlert, setShouldShowMetadataAlert] = useState(false);
  // 点击锁定的设备（优先显示，支持复制）
  const [clickedDevice, setClickedDevice] = useState<MapDevice | null>(null);
  const [activeSectorNumber, setActiveSectorNumber] = useState<number | undefined>();
  const [mapLayoutRevision, setMapLayoutRevision] = useState(0);
  // 测距模式状态（仅用于 showControls=true 场景下的 MapControls 按钮联动）
  // 通过 mapRef.current?.startMeasure() 外部调用时不同步此 state，
  // 但 ESC 由调用方（如 GISMapView）自行监听处理。
  const [isMeasuring, setIsMeasuring] = useState(false);

  // 使用 OpenLayers Hook
  const {
    mapRef,
    mapInstanceRef,
    updateDevices,
    clearDevices,
    getViewport,
    flyTo,
    fitBounds,
    clearHighlight,
    isReady,
    updateSize,
    highlightAndSpiderfyIfNeeded,
    metadata,
    setTileConcurrency,
    startMeasure: startOLMeasure,
    stopMeasure: stopOLMeasure,
    updateAntennaSectors,
  } = useOLMap({
    center: defaultCenter,
    zoom: defaultZoom,
    tileUrl,
    onDeviceClick: (device) => {
      // 点击设备时锁定右侧详情
      setClickedDevice(device);
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
      onMapClick?.();
    },
  });

  const {
    startPolygonDraw,
    stopGeofenceDraw,
    fitGeofence,
    isGeofenceDrawing,
  } = useGeofenceLayer({
    mapInstanceRef,
    isReady,
    items: geofences,
    selectedId: selectedGeofenceId,
    onSelect: (item) => {
      setClickedDevice(null);
      onGeofenceClick?.(item);
    },
    onDrawComplete: onGeofenceDrawComplete,
  });

  const stopMeasure = useCallback(() => {
    stopOLMeasure();
    setIsMeasuring(false);
  }, [stopOLMeasure]);

  const startMeasure = useCallback(() => {
    stopGeofenceDraw();
    startOLMeasure();
    setIsMeasuring(true);
  }, [startOLMeasure, stopGeofenceDraw]);

  const startGeofencePolygonDraw = useCallback(() => {
    stopOLMeasure();
    setIsMeasuring(false);
    startPolygonDraw();
  }, [startPolygonDraw, stopOLMeasure]);

  // 合并主设备列表和搜索结果设备
  const mergedDevices = useMemo(() => {
    const baseDevices = devices || [];
    if (!searchResultDevice) {
      return baseDevices;
    }
    // 使用 Set 优化查找性能（O(n) -> O(1)）
    // 当前数据量下影响不大，但符合业内性能优化最佳实践
    const deviceIdSet = new Set(baseDevices.map(d => d.id));
    const exists = deviceIdSet.has(searchResultDevice.id);
    if (exists) {
      return baseDevices;
    }
    // 将搜索结果设备添加到列表中
    return [...baseDevices, searchResultDevice];
  }, [devices, searchResultDevice]);

  // 更新设备数据（使用合并后的列表）
  useEffect(() => {
    if (isReady) {
      updateDevices(mergedDevices);
    }
  }, [isReady, mergedDevices, updateDevices]);

  useEffect(() => {
    setActiveSectorNumber((current) => {
      if (current !== undefined && antennaSectors.some((sector) => sector.number === current)) return current;
      return antennaSectors[0]?.number;
    });
  }, [antennaSectors, selectedDevice?.id]);

  useEffect(() => {
    if (isReady) {
      updateAntennaSectors(selectedDevice, antennaSectors, activeSectorNumber);
    }
  }, [activeSectorNumber, antennaSectors, isReady, mapLayoutRevision, selectedDevice, updateAntennaSectors, viewport?.zoom]);

  useEffect(() => {
    if (!isReady) return;
    const frame = window.requestAnimationFrame(() => {
      updateSize();
      setMapLayoutRevision((revision) => revision + 1);
    });
    return () => window.cancelAnimationFrame(frame);
  }, [clickedDevice, isReady, updateSize]);

  const activeSectorRenderMode = useMemo(() => {
    const map = mapInstanceRef.current;
    const currentZoom = viewport?.zoom;
    const sector = antennaSectors.find((item) => item.number === activeSectorNumber);
    if (!map || currentZoom === undefined || currentZoom < 13 || !selectedDevice || !sector?.coverageAvailable
      || sector.azimuth === undefined
      || sector.horizontalBeamwidth === undefined || sector.farRadiusMeters === undefined) {
      return 'unavailable' as const;
    }
    const bearing = sector.azimuth! * Math.PI / 180;
    const halfBeam = sector.horizontalBeamwidth * Math.PI / 360;
    const outerStart = fromLonLat(offsetCoordinate(
      [selectedDevice.lng, selectedDevice.lat],
      sector.farRadiusMeters,
      bearing - halfBeam,
    ));
    const outerEnd = fromLonLat(offsetCoordinate(
      [selectedDevice.lng, selectedDevice.lat],
      sector.farRadiusMeters,
      bearing + halfBeam,
    ));
    const startPixel = map.getPixelFromCoordinate(outerStart);
    const endPixel = map.getPixelFromCoordinate(outerEnd);
    return resolveAntennaSectorRenderMode(
      sector,
      startPixel ? [startPixel[0], startPixel[1]] : undefined,
      endPixel ? [endPixel[0], endPixel[1]] : undefined,
    );
    // mapLayoutRevision 用于 updateSize() 后强制按新像素尺寸重新判定窄波束渲染模式。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeSectorNumber, antennaSectors, mapInstanceRef, mapLayoutRevision, selectedDevice, viewport?.zoom]);

  // 当搜索结果设备变化时，自动高亮并定位
  useEffect(() => {
    if (!searchResultDevice) {
      lastHighlightedIdRef.current = null;
      return;
    }
    if (!isReady) return;
    if (lastHighlightedIdRef.current === searchResultDevice) {
      // 完全相同的对象引用：effect 因其它 dep 变化重跑，无需重复定位
      return;
    }
    lastHighlightedIdRef.current = searchResultDevice;
    // 延迟执行，确保设备已添加到地图
    const timer = setTimeout(() => {
      // 搜索定位时需要飞行到目标位置（skipFlyTo = false）
      highlightAndSpiderfyIfNeeded(searchResultDevice, false);
    }, 100);
    return () => clearTimeout(timer);
  }, [searchResultDevice, isReady, highlightAndSpiderfyIfNeeded]);

  useEffect(() => {
    if (metadata && showMetadataTip) {
      // 显示短暂的元数据加载提示
      setShouldShowMetadataAlert(true);
      const timer = setTimeout(() => setShouldShowMetadataAlert(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [metadata, showMetadataTip]);

  // 窗口大小变化时更新地图
  useEffect(() => {
    const handleResize = () => {
      updateSize();
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [updateSize]);

  // 组件卸载时清理所有定时器
  useEffect(() => {
    return () => {
      const timer1 = highlightTimerRef.current;
      const timer2 = highlightWithCardTimerRef.current;
      const timer3 = highlightWithCardInnerTimerRef.current;

      if (timer1) clearTimeout(timer1);
      if (timer2) clearTimeout(timer2);
      if (timer3) clearTimeout(timer3);
    };
  }, []);

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

  // 切换测距模式
  const handleMeasureToggle = useCallback(() => {
    if (isMeasuring) {
      stopMeasure();
      setIsMeasuring(false);
    } else {
      // 进入测距模式时关闭锁定的弹窗，避免遮挡
      setClickedDevice(null);
      startMeasure();
      setIsMeasuring(true);
    }
  }, [isMeasuring, startMeasure, stopMeasure]);

  // ESC 退出测距模式
  useEffect(() => {
    if (!isMeasuring) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        stopMeasure();
        setIsMeasuring(false);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [isMeasuring, stopMeasure]);

  useEffect(() => {
    if (!isGeofenceDrawing) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        stopGeofenceDraw();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [isGeofenceDrawing, stopGeofenceDraw]);

  // 高亮设备（用于搜索定位）
  const highlightAndFlyTo = useCallback((device: MapDevice) => {
    // 清除之前的定时器
    if (highlightTimerRef.current) {
      clearTimeout(highlightTimerRef.current);
    }

    // 使用新函数：自动展开聚合并显示脉冲效果
    highlightAndSpiderfyIfNeeded(device);

    // 5秒后取消高亮（延长时间以便用户查看）
    highlightTimerRef.current = setTimeout(() => {
      clearHighlight();
      highlightTimerRef.current = null;
    }, 5000);
  }, [highlightAndSpiderfyIfNeeded, clearHighlight]);

  useEffect(() => {
    if (!isReady || !selectedDevice || selectedDevice.id !== searchResultDevice?.id) return;

    const timer = setTimeout(() => {
      setClickedDevice(selectedDevice);
    }, 400);
    return () => clearTimeout(timer);
  }, [isReady, searchResultDevice?.id, selectedDevice]);

  /**
   * 高亮设备并显示卡片（用于搜索定位）
   *
   * 业内最佳实践：从当前有瓦片的视图开始，平滑地缩放和平移到目标位置
   *
   * 执行流程：
   * 1. 清除当前的高亮和锁定的卡片
   * 2. 两阶段动画：
   *    - 阶段1：缩放到安全级别（保持在瓦片覆盖范围内）
   *    - 阶段2：平移到目标设备位置
   * 3. 添加脉冲标记效果，视觉反馈目标位置
   * 4. 动画完成后显示设备卡片
   *
   * @param device - 目标设备
   * @param options - 配置选项
   */
  const highlightAndFlyToWithCard = useCallback((
    device: MapDevice,
    options?: HighlightWithCardOptions
  ) => {
    const {
      autoCloseOldCard = true,
      animationMode = 'progressive', // 默认使用渐进式缩放
    } = options ?? {};

    // 1. 清除当前状态
    clearHighlight();
    if (autoCloseOldCard) {
      setClickedDevice(null);
    }

    const map = mapInstanceRef.current;
    if (!map) return;

    const view = map.getView();
    if (!view) return;

    // 2. 确定安全的缩放级别（不超出瓦片覆盖范围）
    const metadataMaxZoom = metadata?.zoom?.max ?? MAP_CONFIG.maxZoom;
    const safeTargetZoom = Math.min(
      ANIMATION_CONFIG.highlightZoom,
      metadataMaxZoom
    );

    // 3. 计算动画总时长（用于延迟显示卡片）
    let totalDuration: number;

    // 4. 根据动画模式执行定位
    if (animationMode === 'direct') {
      // 直接跳转模式
      view.setCenter(fromLonLat([device.lng, device.lat]));
      view.setZoom(safeTargetZoom);
      totalDuration = 50;
    } else if (animationMode === 'fast') {
      // 快速单阶段动画
      view.animate({
        center: fromLonLat([device.lng, device.lat]),
        zoom: safeTargetZoom,
        duration: 500,
      });
      totalDuration = 600;
    } else if (animationMode === 'smooth') {
      // smooth 模式：直接平滑动画到目标位置
      view.animate({
        center: fromLonLat([device.lng, device.lat]),
        zoom: safeTargetZoom,
        duration: 800,
        easing: (t) => {
          // easeOutCubic 缓动函数
          return 1 - Math.pow(1 - t, 3);
        },
      });
      totalDuration = 900;
    } else {
      // progressive 模式：渐进式缩放（国家→省→市→区→街道）
      // 使用 flyTo 的 progressive 选项
      flyTo(device.lng, device.lat, safeTargetZoom, {
        progressive: true,
        maxZoom: metadataMaxZoom,
        onComplete: () => {
          // 动画完成后执行高亮和显示卡片
          executeHighlightAndCard(0);
        },
      });
      // 对于 progressive 模式，不在这里执行后续逻辑
      // 而是在 onComplete 回调中执行
      return;
    }

    // 非 progressive 模式，延迟执行高亮和显示卡片
    executeHighlightAndCard(totalDuration);

    // 统一的高亮和显示卡片逻辑
    function executeHighlightAndCard(delay: number) {
      // 清除之前的定时器
      if (highlightWithCardTimerRef.current) {
        clearTimeout(highlightWithCardTimerRef.current);
      }
      if (highlightWithCardInnerTimerRef.current) {
        clearTimeout(highlightWithCardInnerTimerRef.current);
      }

      highlightWithCardTimerRef.current = setTimeout(() => {
        // 高亮设备（展开聚合、脉冲动画）
        // skipFlyTo: true 避免打断渐进式动画（动画由外层的 flyTo 完成）
        highlightAndSpiderfyIfNeeded(device, true);

        // 显示右侧设备详情
        highlightWithCardInnerTimerRef.current = setTimeout(() => {
          setClickedDevice(device);

          // 5秒后取消高亮（卡片保持显示）
          setTimeout(() => {
            clearHighlight();
          }, 5000);
        }, 100);
      }, delay);
    }
  }, [
    clearHighlight,
    highlightAndSpiderfyIfNeeded,
    mapInstanceRef,
    metadata,
    flyTo,
  ]);

  // 关闭当前锁定的卡片
  const closeClickedCard = useCallback(() => {
    setClickedDevice(null);
  }, []);

  // 暴露方法给父组件
  useImperativeHandle(ref, () => ({
    highlightAndFlyTo,
    highlightAndFlyToWithCard,
    flyTo,
    getViewport,
    fitBounds,
    clearDevices,
    closeClickedCard,
    setTileConcurrency,
    startMeasure,
    stopMeasure,
    startGeofencePolygonDraw,
    stopGeofenceDraw,
    fitGeofence,
  }), [highlightAndFlyTo, highlightAndFlyToWithCard, flyTo, getViewport, fitBounds, clearDevices, closeClickedCard, setTileConcurrency, startMeasure, stopMeasure, startGeofencePolygonDraw, stopGeofenceDraw, fitGeofence]);

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
    display: 'flex',
    width: '100%',
    height,
    background: token.colorBgLayout,
    overflow: 'hidden',
    borderRadius: 8,
    ...style,
  };

  const mapAreaStyle: React.CSSProperties = {
    position: 'relative',
    flex: 1,
    minWidth: 0,
    height: '100%',
    overflow: 'hidden',
  };

  const mapContainerStyle: React.CSSProperties = {
    width: '100%',
    height: '100%',
    overflow: 'hidden', // 地图容器内部保持裁剪
    borderRadius: 8,
    ...(isReady ? { opacity: 1, transition: 'opacity 0.5s ease-in' } : { opacity: 0 }),
  };

  const loadingStyle: React.CSSProperties = {
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    zIndex: 1000,
  };

  // 元数据信息提示
  const metadataAlert = useMemo(() => {
    if (!metadata || !shouldShowMetadataAlert) return null;
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
  }, [metadata, shouldShowMetadataAlert, intl]);

  return (
    <div
      style={containerStyle}
      className={`gis-map-container ${styles.gisMapContainer} ${className || ''}`}
    >
      <div style={mapAreaStyle}>
        {/* 地图容器 */}
        <div ref={mapRef} style={mapContainerStyle} />

        {/* 加载状态 */}
        {!isReady && (
          <div style={loadingStyle}>
            <Spin size="large" indicator={<LoadingOutlined spin />} />
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
              title={metadataAlert.message}
              description={metadataAlert.description}
              type="info"
              icon={<InfoCircleOutlined />}
              showIcon
              closable
              onClose={() => setShouldShowMetadataAlert(false)}
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
            isMeasuring={isMeasuring}
            onMeasureToggle={handleMeasureToggle}
          />
        )}

        {/* 统计面板 */}
        {showStats && isReady && <MapStatsPanel stats={stats} visible />}

        {/* hover 仅显示轻量设备提示，不承载天线编辑。 */}
        {!clickedDevice && hoveredDevice && popupPosition && (
          <MapPopup
            device={hoveredDevice}
            visible
            variant="tooltip"
            position={popupPosition}
            onAlarmClick={onAlarmClick}
          />
        )}
      </div>

      {clickedDevice && (
        <aside className={styles.deviceDetailsPanel} aria-label={intl.formatMessage({ id: 'gis.deviceDetails' })}>
          <MapPopup
            device={clickedDevice}
            visible
            variant="panel"
            onAlarmClick={onAlarmClick}
            antennaSectors={clickedDevice.id === selectedDevice?.id ? antennaSectors : []}
            activeSectorNumber={activeSectorNumber}
            activeSectorRenderMode={activeSectorRenderMode}
            onActiveSectorChange={setActiveSectorNumber}
            onAntennaPreviewChange={onAntennaPreviewChange}
            onAntennaCancel={onAntennaCancel}
            onAntennaSave={onAntennaSave}
            antennaSaving={antennaSaving}
            onClose={() => {
              setClickedDevice(null);
              setHoveredDevice(null);
              setPopupPosition(null);
              onMapClick?.();
            }}
          />
        </aside>
      )}
    </div>
  );
});

// 导出组件和类型
export default GISMap;
export type { GISMapProps, MapDevice, MapViewport, MapStats, GISMapRef, HighlightWithCardOptions };

// 导出子组件（可选）
export { default as MapPopup } from './MapPopup';
export { default as MapControls } from './MapControls';
export { default as MapStatsPanel } from './MapStatsPanel';
export { default as GroupTree } from './GroupTree';
export { default as DeviceSearch } from './DeviceSearch';

// 导出 Hook、常量和工具已在各模块中直接导出，无需从主组件重导出
