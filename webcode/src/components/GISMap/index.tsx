import React, { useCallback, useState } from 'react';
import { Card, Tag, Tooltip, Typography } from 'antd';
import { EnvironmentOutlined, ExperimentOutlined } from '@ant-design/icons';
import { useThemeToken, useIsDark } from '@/hooks/useThemeToken';

export interface MapDevice {
  lat: number;
  lng: number;
  status: 'online' | 'offline' | 'warning' | 'error';
  name: string;
  sn?: string;
}

export interface GISMapProps {
  devices?: MapDevice[];
  height?: number | string;
  onDeviceClick?: (device: MapDevice) => void;
  style?: React.CSSProperties;
}

const STATUS_COLORS: Record<string, string> = {
  online: '#52C41A',
  offline: '#8C8C8C',
  warning: '#FA8C16',
  error: '#F5222D',
};

// China rough bounding box: lat [18, 53], lng [73, 135]
const MAP_LAT_MIN = 18;
const MAP_LAT_MAX = 53;
const MAP_LNG_MIN = 73;
const MAP_LNG_MAX = 135;

// China SVG path (simplified outline)
const CHINA_PATH =
  'M 220,30 L 280,20 L 340,35 L 380,25 L 420,45 L 460,30 L 500,50 L 520,80 ' +
  'L 510,120 L 530,150 L 510,180 L 520,210 L 500,240 L 470,260 L 450,290 ' +
  'L 420,310 L 400,340 L 370,360 L 340,370 L 310,350 L 280,360 L 250,340 ' +
  'L 230,310 L 200,290 L 180,260 L 160,230 L 150,200 L 170,170 L 160,140 ' +
  'L 180,110 L 190,80 L 210,55 Z';

// Taiwan island
const TAIWAN_PATH = 'M 490,270 L 498,265 L 502,280 L 495,285 Z';

// Hainan island
const HAINAN_PATH = 'M 360,360 L 380,358 L 378,375 L 358,374 Z';

const SVG_WIDTH = 680;
const SVG_HEIGHT = 420;

function projectToSVG(lat: number, lng: number): [number, number] {
  const x =
    ((lng - MAP_LNG_MIN) / (MAP_LNG_MAX - MAP_LNG_MIN)) * (SVG_WIDTH - 80) + 40;
  const y =
    ((MAP_LAT_MAX - lat) / (MAP_LAT_MAX - MAP_LAT_MIN)) * (SVG_HEIGHT - 80) + 30;
  return [x, y];
}

const GISMap: React.FC<GISMapProps> = ({
  devices = [],
  height = 480,
  onDeviceClick,
  style,
}) => {
  const [hoveredDevice, setHoveredDevice] = useState<MapDevice | null>(null);
  const [tooltipPos, setTooltipPos] = useState<{ x: number; y: number } | null>(null);
  const token = useThemeToken();
  const isDark = useIsDark();

  const overlayBg = isDark ? 'rgba(40,40,40,0.92)' : 'rgba(255,255,255,0.92)';

  const handleDotClick = useCallback(
    (device: MapDevice) => {
      onDeviceClick?.(device);
    },
    [onDeviceClick]
  );

  const handleDotMouseEnter = (device: MapDevice, x: number, y: number) => {
    setHoveredDevice(device);
    setTooltipPos({ x, y });
  };

  const handleDotMouseLeave = () => {
    setHoveredDevice(null);
    setTooltipPos(null);
  };

  return (
    <div
      style={{
        position: 'relative',
        height,
        background: '#EBF5FF',
        borderRadius: 8,
        overflow: 'hidden',
        ...style,
      }}
    >
      {/* Demo mode banner */}
      <Card
        size="small"
        style={{
          position: 'absolute',
          top: 12,
          left: 12,
          zIndex: 10,
          background: overlayBg,
          border: `1px solid ${token.colorBorder}`,
          borderRadius: 6,
          boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
        }}
        styles={{ body: { padding: '6px 12px' } }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <EnvironmentOutlined style={{ color: token.colorPrimary }} />
          <Typography.Text strong style={{ fontSize: 13 }}>
            GIS地图
          </Typography.Text>
          <Tag icon={<ExperimentOutlined />} color="blue" style={{ margin: 0, fontSize: 11 }}>
            Demo模式
          </Tag>
        </div>
      </Card>

      {/* Legend */}
      <Card
        size="small"
        style={{
          position: 'absolute',
          bottom: 12,
          right: 12,
          zIndex: 10,
          background: overlayBg,
          border: `1px solid ${token.colorBorder}`,
          borderRadius: 6,
        }}
        styles={{ body: { padding: '6px 10px' } }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {Object.entries(STATUS_COLORS).map(([status, color]) => (
            <div key={status} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <div
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: color,
                  flexShrink: 0,
                }}
              />
              <Typography.Text style={{ fontSize: 11, color: token.colorText }}>
                {status === 'online' ? '在线' : status === 'offline' ? '离线' : status === 'warning' ? '告警' : '故障'}
              </Typography.Text>
            </div>
          ))}
        </div>
      </Card>

      {/* Device count */}
      <div
        style={{
          position: 'absolute',
          top: 12,
          right: 12,
          zIndex: 10,
          background: overlayBg,
          border: `1px solid ${token.colorBorder}`,
          borderRadius: 6,
          padding: '6px 12px',
          fontSize: 13,
          color: token.colorText,
        }}
      >
        共 <strong style={{ color: token.colorPrimary }}>{devices.length}</strong> 台设备
      </div>

      {/* SVG Map */}
      <svg
        viewBox={`0 0 ${SVG_WIDTH} ${SVG_HEIGHT}`}
        width="100%"
        height="100%"
        style={{ display: 'block' }}
        preserveAspectRatio="xMidYMid meet"
      >
        {/* Ocean background */}
        <rect width={SVG_WIDTH} height={SVG_HEIGHT} fill="#DDEEFF" />

        {/* Grid lines */}
        {Array.from({ length: 6 }, (_, i) => (
          <line
            key={`h${i}`}
            x1={0}
            y1={(i * SVG_HEIGHT) / 5}
            x2={SVG_WIDTH}
            y2={(i * SVG_HEIGHT) / 5}
            stroke="#C5DCEF"
            strokeWidth={0.5}
          />
        ))}
        {Array.from({ length: 8 }, (_, i) => (
          <line
            key={`v${i}`}
            x1={(i * SVG_WIDTH) / 7}
            y1={0}
            x2={(i * SVG_WIDTH) / 7}
            y2={SVG_HEIGHT}
            stroke="#C5DCEF"
            strokeWidth={0.5}
          />
        ))}

        {/* China mainland */}
        <path d={CHINA_PATH} fill="#C8DDB0" stroke="#A8C890" strokeWidth={1.5} />

        {/* Taiwan */}
        <path d={TAIWAN_PATH} fill="#C8DDB0" stroke="#A8C890" strokeWidth={1} />

        {/* Hainan */}
        <path d={HAINAN_PATH} fill="#C8DDB0" stroke="#A8C890" strokeWidth={1} />

        {/* Province boundaries (very simplified) */}
        <path
          d="M 280,60 L 290,100 M 340,50 L 350,90 M 200,150 L 280,160 M 350,150 L 420,160 M 300,200 L 310,250"
          stroke="#B0C898"
          strokeWidth={0.8}
          fill="none"
          strokeDasharray="4,3"
        />

        {/* Device dots */}
        {devices.map((device, i) => {
          const [x, y] = projectToSVG(device.lat, device.lng);
          const color = STATUS_COLORS[device.status] ?? '#8C8C8C';
          return (
            <g key={`${device.sn ?? device.name}-${i}`}>
              {/* Pulse ring for online devices */}
              {device.status === 'online' && (
                <circle
                  cx={x}
                  cy={y}
                  r={10}
                  fill={color}
                  opacity={0.2}
                />
              )}
              {/* Main dot */}
              <circle
                cx={x}
                cy={y}
                r={6}
                fill={color}
                stroke="#fff"
                strokeWidth={1.5}
                style={{ cursor: onDeviceClick ? 'pointer' : 'default' }}
                onClick={() => handleDotClick(device)}
                onMouseEnter={() => handleDotMouseEnter(device, x, y)}
                onMouseLeave={handleDotMouseLeave}
              />
            </g>
          );
        })}

        {/* Tooltip (SVG foreignObject) */}
        {hoveredDevice && tooltipPos && (
          <foreignObject
            x={tooltipPos.x + 10}
            y={tooltipPos.y - 40}
            width={160}
            height={60}
            style={{ pointerEvents: 'none' }}
          >
            <div
              style={{
                background: overlayBg,
                border: `1px solid ${token.colorBorder}`,
                borderRadius: 4,
                padding: '6px 10px',
                boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
                fontSize: 12,
              }}
            >
              <div style={{ fontWeight: 600, color: token.colorTextHeading }}>{hoveredDevice.name}</div>
              {hoveredDevice.sn && (
                <div style={{ color: token.colorTextSecondary, fontFamily: 'monospace', fontSize: 11 }}>
                  {hoveredDevice.sn}
                </div>
              )}
              <div style={{ color: STATUS_COLORS[hoveredDevice.status] }}>
                {hoveredDevice.status === 'online'
                  ? '在线'
                  : hoveredDevice.status === 'offline'
                    ? '离线'
                    : hoveredDevice.status === 'warning'
                      ? '告警'
                      : '故障'}
              </div>
            </div>
          </foreignObject>
        )}
      </svg>
    </div>
  );
};

export default GISMap;
