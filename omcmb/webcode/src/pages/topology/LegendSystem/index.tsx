import { Card, Col, Row, Tag, Tooltip, Typography } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Title, Text } = Typography;

type TFn = (id: string) => string;

interface LegendItem {
  labelKey: string;
  color?: string;
  bgColor?: string;
  borderColor?: string;
  dash?: string;
  shape?: 'circle' | 'rect' | 'diamond' | 'line';
  descKey: string;
}

const DEVICE_TYPE_ICONS: LegendItem[] = [
  { labelKey: 'gis.legend.deviceType.enb.label', color: '#1677ff', bgColor: '#e6f4ff', borderColor: '#91caff', shape: 'circle', descKey: 'gis.legend.deviceType.enb.desc' },
  { labelKey: 'gis.legend.deviceType.gnb.label', color: '#722ed1', bgColor: '#f9f0ff', borderColor: '#d3adf7', shape: 'circle', descKey: 'gis.legend.deviceType.gnb.desc' },
  { labelKey: 'gis.legend.deviceType.cpe.label', color: '#13c2c2', bgColor: '#e6fffb', borderColor: '#87e8de', shape: 'circle', descKey: 'gis.legend.deviceType.cpe.desc' },
  { labelKey: 'gis.legend.deviceType.egw.label', color: '#fa8c16', bgColor: '#fff7e6', borderColor: '#ffd591', shape: 'diamond', descKey: 'gis.legend.deviceType.egw.desc' },
  { labelKey: 'gis.legend.deviceType.rt.label', color: '#389e0d', bgColor: '#f6ffed', borderColor: '#b7eb8f', shape: 'rect', descKey: 'gis.legend.deviceType.rt.desc' },
  { labelKey: 'gis.legend.deviceType.sw.label', color: '#cf1322', bgColor: '#fff1f0', borderColor: '#ffa39e', shape: 'rect', descKey: 'gis.legend.deviceType.sw.desc' },
  { labelKey: 'gis.legend.deviceType.domain.label', color: '#8c8c8c', bgColor: '#f5f5f5', borderColor: '#d9d9d9', shape: 'circle', descKey: 'gis.legend.deviceType.domain.desc' },
  { labelKey: 'gis.legend.deviceType.site.label', color: '#d48806', bgColor: '#fffbe6', borderColor: '#ffe58f', shape: 'circle', descKey: 'gis.legend.deviceType.site.desc' },
];

const NODE_STATUS_COLORS: LegendItem[] = [
  { labelKey: 'gis.legend.nodeStatus.online.label', color: '#52c41a', bgColor: '#f6ffed', descKey: 'gis.legend.nodeStatus.online.desc' },
  { labelKey: 'gis.legend.nodeStatus.offline.label', color: '#8c8c8c', bgColor: '#f5f5f5', descKey: 'gis.legend.nodeStatus.offline.desc' },
  { labelKey: 'gis.legend.nodeStatus.alarm.label', color: '#f5222d', bgColor: '#fff1f0', descKey: 'gis.legend.nodeStatus.alarm.desc' },
  { labelKey: 'gis.legend.nodeStatus.maintenance.label', color: '#fa8c16', bgColor: '#fff7e6', descKey: 'gis.legend.nodeStatus.maintenance.desc' },
];

const ALARM_SEVERITY_COLORS: LegendItem[] = [
  { labelKey: 'gis.legend.severity.critical.label', color: '#520339', bgColor: '#9e1068', descKey: 'gis.legend.severity.critical.desc' },
  { labelKey: 'gis.legend.severity.major.label', color: '#fff0f6', bgColor: '#f5222d', descKey: 'gis.legend.severity.major.desc' },
  { labelKey: 'gis.legend.severity.minor.label', color: '#fff7e6', bgColor: '#fa8c16', descKey: 'gis.legend.severity.minor.desc' },
  { labelKey: 'gis.legend.severity.warning.label', color: '#feffe6', bgColor: '#d4b106', descKey: 'gis.legend.severity.warning.desc' },
  { labelKey: 'gis.legend.severity.notice.label', color: '#e6f7ff', bgColor: '#1677ff', descKey: 'gis.legend.severity.notice.desc' },
];

const EDGE_TYPES: LegendItem[] = [
  { labelKey: 'gis.legend.edge.normal.label', color: '#1677ff', shape: 'line', descKey: 'gis.legend.edge.normal.desc' },
  { labelKey: 'gis.legend.edge.degraded.label', color: '#fa8c16', dash: '8,4', shape: 'line', descKey: 'gis.legend.edge.degraded.desc' },
  { labelKey: 'gis.legend.edge.disconnected.label', color: '#8c8c8c', dash: '4,4', shape: 'line', descKey: 'gis.legend.edge.disconnected.desc' },
];

function DeviceIconSwatch({ item, t }: { item: LegendItem; t: TFn }) {
  const getShape = () => {
    const base: React.CSSProperties = {
      width: 40,
      height: 40,
      background: item.bgColor,
      border: `2px solid ${item.borderColor ?? item.color}`,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      flexShrink: 0,
    };

    if (item.shape === 'circle') return { ...base, borderRadius: '50%' };
    if (item.shape === 'rect') return { ...base, borderRadius: 6 };
    if (item.shape === 'diamond') return { ...base, transform: 'rotate(45deg)', borderRadius: 4 };
    return base;
  };

  const label = t(item.labelKey);

  return (
    <Tooltip title={t(item.descKey)}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 0', cursor: 'default' }}>
        <div style={getShape()}>
          <span style={{ color: item.color, fontWeight: 700, fontSize: 11, transform: item.shape === 'diamond' ? 'rotate(-45deg)' : undefined }}>
            {label}
          </span>
        </div>
        <div>
          <Text style={{ fontSize: 12, fontWeight: 500, display: 'block' }}>{label}</Text>
          <Text type="secondary" style={{ fontSize: 11 }}>{t(item.descKey)}</Text>
        </div>
      </div>
    </Tooltip>
  );
}

function ColorSwatch({ item, t }: { item: LegendItem; t: TFn }) {
  return (
    <Tooltip title={t(item.descKey)}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '5px 0', cursor: 'default' }}>
        {item.shape === 'line' ? (
          <svg width={48} height={16} style={{ flexShrink: 0 }}>
            <line
              x1={0}
              y1={8}
              x2={48}
              y2={8}
              stroke={item.color}
              strokeWidth={2.5}
              strokeDasharray={item.dash}
            />
            <polygon points="44,5 48,8 44,11" fill={item.color} />
          </svg>
        ) : (
          <div style={{
            width: 20,
            height: 20,
            borderRadius: 4,
            background: item.bgColor ?? item.color,
            border: `2px solid ${item.color}`,
            flexShrink: 0,
          }} />
        )}
        <div>
          <Text style={{ fontSize: 12, fontWeight: 500, display: 'block' }}>{t(item.labelKey)}</Text>
          <Text type="secondary" style={{ fontSize: 11 }}>{t(item.descKey)}</Text>
        </div>
      </div>
    </Tooltip>
  );
}

export default function LegendSystem() {
  const t = useT();
  const otherMarkers = [
    { symbol: '●', color: '#f5222d', labelKey: 'gis.legend.marker.redBadge' },
    { symbol: '●', color: '#fa8c16', labelKey: 'gis.legend.marker.orangeBadge' },
    { symbol: '⟳', color: '#1677ff', labelKey: 'gis.legend.marker.maintenance' },
  ];
  return (
    <ListPageLayout title={t('nav.topology.legend')}>
      <Row gutter={[16, 16]}>
        {/* Device Type Icons */}
        <Col xs={24} md={12}>
          <Card size="small" title={<Title level={5} style={{ margin: 0 }}>{t('gis.legend.section.deviceType')}</Title>}>
            <div style={{ columns: 2, columnGap: 24 }}>
              {DEVICE_TYPE_ICONS.map((item) => (
                <div key={item.labelKey} style={{ breakInside: 'avoid' }}>
                  <DeviceIconSwatch item={item} t={t} />
                </div>
              ))}
            </div>
          </Card>
        </Col>

        {/* Node Status */}
        <Col xs={24} md={12}>
          <Card size="small" title={<Title level={5} style={{ margin: 0 }}>{t('gis.legend.section.nodeStatus')}</Title>}>
            {NODE_STATUS_COLORS.map((item) => (
              <ColorSwatch key={item.labelKey} item={item} t={t} />
            ))}
          </Card>
        </Col>

        {/* Alarm Severity */}
        <Col xs={24} md={12}>
          <Card size="small" title={<Title level={5} style={{ margin: 0 }}>{t('gis.legend.section.alarmSeverity')}</Title>}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {ALARM_SEVERITY_COLORS.map((item) => (
                <div key={item.labelKey} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <Tag
                    color={item.bgColor}
                    style={{ color: item.color, minWidth: 48, textAlign: 'center', fontWeight: 600 }}
                  >
                    {t(item.labelKey)}
                  </Tag>
                  <Text type="secondary" style={{ fontSize: 12 }}>{t(item.descKey)}</Text>
                </div>
              ))}
            </div>
          </Card>
        </Col>

        {/* Edge Types */}
        <Col xs={24} md={12}>
          <Card size="small" title={<Title level={5} style={{ margin: 0 }}>{t('gis.legend.section.edgeType')}</Title>}>
            {EDGE_TYPES.map((item) => (
              <ColorSwatch key={item.labelKey} item={item} t={t} />
            ))}

            <div style={{ borderTop: '1px solid #f0f0f0', marginTop: 16, paddingTop: 12 }}>
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 8 }}>{t('gis.legend.otherMarkers')}</Text>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {otherMarkers.map((s) => (
                  <div key={s.labelKey} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ color: s.color, fontSize: 16, lineHeight: 1, width: 20, textAlign: 'center' }}>{s.symbol}</span>
                    <Text style={{ fontSize: 12 }}>{t(s.labelKey)}</Text>
                  </div>
                ))}
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </ListPageLayout>
  );
}
