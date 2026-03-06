import { Card, Col, Row, Tag, Tooltip, Typography } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Title, Text } = Typography;

interface LegendItem {
  label: string;
  color?: string;
  bgColor?: string;
  borderColor?: string;
  dash?: string;
  shape?: 'circle' | 'rect' | 'diamond' | 'line';
  description: string;
}

const DEVICE_TYPE_ICONS: LegendItem[] = [
  { label: 'eNB', color: '#1677ff', bgColor: '#e6f4ff', borderColor: '#91caff', shape: 'circle', description: '4G基站 (eNodeB)' },
  { label: 'gNB', color: '#722ed1', bgColor: '#f9f0ff', borderColor: '#d3adf7', shape: 'circle', description: '5G基站 (gNodeB)' },
  { label: 'CPE', color: '#13c2c2', bgColor: '#e6fffb', borderColor: '#87e8de', shape: 'circle', description: '客户前置设备 (CPE)' },
  { label: 'eGW', color: '#fa8c16', bgColor: '#fff7e6', borderColor: '#ffd591', shape: 'diamond', description: '边缘网关 (eGateway)' },
  { label: 'RT', color: '#389e0d', bgColor: '#f6ffed', borderColor: '#b7eb8f', shape: 'rect', description: '路由器 (Router)' },
  { label: 'SW', color: '#cf1322', bgColor: '#fff1f0', borderColor: '#ffa39e', shape: 'rect', description: '交换机 (Switch)' },
  { label: '域', color: '#8c8c8c', bgColor: '#f5f5f5', borderColor: '#d9d9d9', shape: 'circle', description: '网络域节点' },
  { label: '站', color: '#d48806', bgColor: '#fffbe6', borderColor: '#ffe58f', shape: 'circle', description: '物理站点节点' },
];

const NODE_STATUS_COLORS: LegendItem[] = [
  { label: '在线', color: '#52c41a', bgColor: '#f6ffed', description: '设备正常运行' },
  { label: '离线', color: '#8c8c8c', bgColor: '#f5f5f5', description: '设备无法连接' },
  { label: '告警', color: '#f5222d', bgColor: '#fff1f0', description: '设备存在活动告警' },
  { label: '维护', color: '#fa8c16', bgColor: '#fff7e6', description: '设备处于维护状态' },
];

const ALARM_SEVERITY_COLORS: LegendItem[] = [
  { label: '严重', color: '#520339', bgColor: '#9e1068', description: '严重告警，需立即处理' },
  { label: '主要', color: '#fff0f6', bgColor: '#f5222d', description: '主要告警，需尽快处理' },
  { label: '次要', color: '#fff7e6', bgColor: '#fa8c16', description: '次要告警，需关注' },
  { label: '警告', color: '#feffe6', bgColor: '#d4b106', description: '警告，建议关注' },
  { label: '通知', color: '#e6f7ff', bgColor: '#1677ff', description: '普通通知' },
];

const EDGE_TYPES: LegendItem[] = [
  { label: '正常连接', color: '#1677ff', shape: 'line', description: '链路正常传输中' },
  { label: '降级连接', color: '#fa8c16', dash: '8,4', shape: 'line', description: '链路质量下降，带宽受限' },
  { label: '断开连接', color: '#8c8c8c', dash: '4,4', shape: 'line', description: '链路断开或不可用' },
];

function DeviceIconSwatch({ item }: { item: LegendItem }) {
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

  return (
    <Tooltip title={item.description}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 0', cursor: 'default' }}>
        <div style={getShape()}>
          <span style={{ color: item.color, fontWeight: 700, fontSize: 11, transform: item.shape === 'diamond' ? 'rotate(-45deg)' : undefined }}>
            {item.label}
          </span>
        </div>
        <div>
          <Text style={{ fontSize: 12, fontWeight: 500, display: 'block' }}>{item.label}</Text>
          <Text type="secondary" style={{ fontSize: 11 }}>{item.description}</Text>
        </div>
      </div>
    </Tooltip>
  );
}

function ColorSwatch({ item }: { item: LegendItem }) {
  return (
    <Tooltip title={item.description}>
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
          <Text style={{ fontSize: 12, fontWeight: 500, display: 'block' }}>{item.label}</Text>
          <Text type="secondary" style={{ fontSize: 11 }}>{item.description}</Text>
        </div>
      </div>
    </Tooltip>
  );
}

export default function LegendSystem() {
  const t = useT();
  return (
    <ListPageLayout title={t('nav.topology.legend')}>
      <Row gutter={[16, 16]}>
        {/* Device Type Icons */}
        <Col xs={24} md={12}>
          <Card size="small" title={<Title level={5} style={{ margin: 0 }}>设备类型图标</Title>}>
            <div style={{ columns: 2, columnGap: 24 }}>
              {DEVICE_TYPE_ICONS.map((item) => (
                <div key={item.label} style={{ breakInside: 'avoid' }}>
                  <DeviceIconSwatch item={item} />
                </div>
              ))}
            </div>
          </Card>
        </Col>

        {/* Node Status */}
        <Col xs={24} md={12}>
          <Card size="small" title={<Title level={5} style={{ margin: 0 }}>节点状态颜色</Title>}>
            {NODE_STATUS_COLORS.map((item) => (
              <ColorSwatch key={item.label} item={item} />
            ))}
          </Card>
        </Col>

        {/* Alarm Severity */}
        <Col xs={24} md={12}>
          <Card size="small" title={<Title level={5} style={{ margin: 0 }}>告警级别颜色</Title>}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {ALARM_SEVERITY_COLORS.map((item) => (
                <div key={item.label} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <Tag
                    color={item.bgColor}
                    style={{ color: item.color, minWidth: 48, textAlign: 'center', fontWeight: 600 }}
                  >
                    {item.label}
                  </Tag>
                  <Text type="secondary" style={{ fontSize: 12 }}>{item.description}</Text>
                </div>
              ))}
            </div>
          </Card>
        </Col>

        {/* Edge Types */}
        <Col xs={24} md={12}>
          <Card size="small" title={<Title level={5} style={{ margin: 0 }}>连线类型</Title>}>
            {EDGE_TYPES.map((item) => (
              <ColorSwatch key={item.label} item={item} />
            ))}

            <div style={{ borderTop: '1px solid #f0f0f0', marginTop: 16, paddingTop: 12 }}>
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 8 }}>其他标记说明:</Text>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {[
                  { symbol: '●', color: '#f5222d', label: '红色角标 - 存在严重/主要告警' },
                  { symbol: '●', color: '#fa8c16', label: '橙色角标 - 存在次要/警告告警' },
                  { symbol: '⟳', color: '#1677ff', label: '蓝色图标 - 正在维护操作' },
                ].map((s) => (
                  <div key={s.label} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ color: s.color, fontSize: 16, lineHeight: 1, width: 20, textAlign: 'center' }}>{s.symbol}</span>
                    <Text style={{ fontSize: 12 }}>{s.label}</Text>
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
