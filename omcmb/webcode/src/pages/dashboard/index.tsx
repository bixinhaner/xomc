import { useCallback, useMemo, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Avatar,
  Badge,
  Card,
  Col,
  List,
  Row,
  Space,
  Tag,
  Typography,
} from 'antd';
import {
  AlertOutlined,
  AppstoreOutlined,
  ApartmentOutlined,
  CloudServerOutlined,
  DashboardOutlined,
  FileTextOutlined,
  MonitorOutlined,
  PlayCircleOutlined,
  RocketOutlined,
  SettingOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  UserOutlined,
  WifiOutlined,
} from '@ant-design/icons';
import KPICard from '@/components/KPICard';
import PieChart from '@/components/Charts/PieChart';
import BarChart from '@/components/Charts/BarChart';
import LineChart from '@/components/Charts/LineChart';
import GISMap from '@/components/GISMap';
import type { MapDevice } from '@/components/GISMap';
import { useDashboardData } from '@core/hooks/api/useDashboard';
import { useAlarmCount, useCurrentAlarms } from '@core/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { TiltCard } from '@/components/Effects';
import { useScrollReveal } from '@/hooks/useScrollReveal';

const { Title, Text } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: '#F5222D',
  major: '#FA8C16',
  minor: '#FADB14',
  warning: '#1677FF',
};

function formatAlarmTime(value: string | undefined): string {
  if (!value) {
    return '--';
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return parsed.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

const QUICK_ACCESS_ITEMS = [
  { labelKey: 'nav.device.list',       icon: <AppstoreOutlined />,    path: '/device/list',               color: '#1677FF' },
  { labelKey: 'nav.alarm.current',     icon: <AlertOutlined />,       path: '/alarm/current',             color: '#F5222D' },
  { labelKey: 'nav.alarm.statistics',  icon: <DashboardOutlined />,   path: '/alarm/statistics',          color: '#FA8C16' },
  { labelKey: 'nav.device.ne',         icon: <ApartmentOutlined />,   path: '/device/ne',                 color: '#52C41A' },
  { labelKey: 'nav.device.monitor',    icon: <MonitorOutlined />,     path: '/device/monitor',            color: '#722ED1' },
  { labelKey: 'nav.device.commission', icon: <RocketOutlined />,      path: '/device/commission',         color: '#13C2C2' },
  { labelKey: 'nav.performance.kpiStandard', icon: <ThunderboltOutlined />, path: '/performance/kpi-standard', color: '#EB2F96' },
  { labelKey: 'nav.device.stats',      icon: <CloudServerOutlined />, path: '/device/stats',              color: '#2F54EB' },
  { labelKey: 'nav.alarm.rules',       icon: <SettingOutlined />,     path: '/alarm/rules',               color: '#8C8C8C' },
  { labelKey: 'nav.log.system',        icon: <FileTextOutlined />,    path: '/log/system',                color: '#595959' },
  { labelKey: 'nav.system.users',      icon: <TeamOutlined />,        path: '/system/users',              color: '#D46B08' },
  { labelKey: 'nav.mml.console',       icon: <PlayCircleOutlined />,  path: '/mml/console',               color: '#08979C' },
];

const MOCK_MAP_DEVICES: MapDevice[] = [
  { id: '1', lat: 39.9, lng: 116.4, status: 'onlineActive', name: '北京基站-001', sn: 'SN-BJ001' },
  { id: '2', lat: 31.2, lng: 121.5, status: 'onlineActive', name: '上海基站-002', sn: 'SN-SH002' },
  { id: '3', lat: 23.1, lng: 113.3, status: 'onlineActive', name: '广州基站-003', sn: 'SN-GZ003', alarmCount: 2 },
  { id: '4', lat: 22.5, lng: 114.1, status: 'onlineActive', name: '深圳基站-004', sn: 'SN-SZ004' },
  { id: '5', lat: 30.7, lng: 104.1, status: 'offline', name: '成都基站-005', sn: 'SN-CD005', alarmCount: 5 },
  { id: '6', lat: 36.1, lng: 103.8, status: 'offline', name: '兰州基站-006', sn: 'SN-LZ006' },
  { id: '7', lat: 34.3, lng: 108.9, status: 'onlineActive', name: '西安基站-007', sn: 'SN-XA007' },
  { id: '8', lat: 32.0, lng: 118.8, status: 'onlineActive', name: '南京基站-008', sn: 'SN-NJ008' },
  { id: '9', lat: 45.8, lng: 126.5, status: 'offline', name: '哈尔滨基站-009', sn: 'SN-HRB009' },
  { id: '10', lat: 25.0, lng: 102.7, status: 'onlineActive', name: '昆明基站-010', sn: 'SN-KM010', alarmCount: 1 },
];

export default function DashboardPage() {
  const navigate = useNavigate();
  const { data: dashboardData, isLoading } = useDashboardData();
  const { data: alarmCount } = useAlarmCount();
  const { data: currentAlarmData, isLoading: isCurrentAlarmsLoading } = useCurrentAlarms({
    page: 1,
    pageSize: 6,
  });
  const t = useT();
  const token = useThemeToken();

  // KPI values — use real data when available, fall back to sensible defaults
  const totalDevices = dashboardData?.summary?.deviceCounts?.total ?? 1284;
  const onlineDevices = dashboardData?.summary?.deviceCounts?.online ?? 1137;
  const activeAlarms = dashboardData?.summary?.alarmCounts?.total ?? 43;
  const runningTasks = dashboardData?.summary?.taskSummary?.running ?? 7;

  // Alarm severity counts
  const critical = alarmCount?.critical ?? 8;
  const major = alarmCount?.major ?? 15;
  const minor = alarmCount?.minor ?? 12;
  const warning = alarmCount?.warning ?? 8;

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  // Recent active alarms
  const recentAlarms = useMemo(
    () =>
      (currentAlarmData?.items || []).map((alarm) => ({
        id: alarm.id,
        alarmName: alarm.alarmName || alarm.alarmIdentifier,
        deviceName: alarm.deviceName || alarm.deviceSn,
        severity: alarm.severity,
        eventTime: formatAlarmTime(alarm.eventTime),
      })),
    [currentAlarmData]
  );

  // Donut chart data for alarm severity
  const alarmPieData = useMemo(
    () => [
      { name: t('alarm.severity.critical'), value: critical },
      { name: t('alarm.severity.major'), value: major },
      { name: t('alarm.severity.minor'), value: minor },
      { name: t('alarm.severity.warning'), value: warning },
    ],
    [critical, major, minor, warning, t]
  );

  // Device status bar chart data
  const deviceStatusXData = ['eNB', 'gNB', 'CPE', 'eGW'];
  const deviceStatusSeries = useMemo(() => [
    { name: t('dashboard.chart.online'), data: [432, 318, 265, 122], color: '#52C41A' },
    { name: t('dashboard.chart.offline'), data: [45, 28, 33, 14], color: '#8C8C8C' },
    { name: t('dashboard.chart.alarm'), data: [12, 8, 15, 8], color: '#FA8C16' },
  ], [t]);

  // 7-day alarm trend
  const trendXData = useMemo(() => {
    const days: string[] = [];
    for (let i = 6; i >= 0; i--) {
      const d = new Date();
      d.setDate(d.getDate() - i);
      days.push(`${d.getMonth() + 1}/${d.getDate()}`);
    }
    return days;
  }, []);

  const alarmTrendSeries = useMemo(() => [
    { name: t('alarm.severity.critical'), data: [5, 8, 6, 9, 7, 10, 8], color: '#F5222D' },
    { name: t('alarm.severity.major'), data: [12, 15, 11, 18, 14, 17, 15], color: '#FA8C16' },
    { name: t('alarm.severity.minor'), data: [8, 10, 9, 12, 11, 13, 12], color: '#FADB14' },
    { name: t('alarm.severity.warning'), data: [6, 7, 5, 8, 6, 9, 8], color: '#1677FF' },
  ], [t]);

  // TOP10 alarm devices horizontal bar chart
  const top10Devices = [
    '成都基站-005', '广州基站-003', '昆明基站-010', '哈尔滨-009',
    '北京-001', '上海-002', '西安-007', '南京-008', '深圳-004', '兰州-006',
  ];
  const top10Series = useMemo(() => [
    { name: t('dashboard.alarmCount'), data: [24, 21, 18, 16, 14, 12, 10, 8, 6, 4] },
  ], [t]);

  const dashboardRef = useRef<HTMLDivElement>(null);
  useScrollReveal(dashboardRef);

  const handleDeviceClick = useCallback(
    (device: MapDevice) => {
      if (device.sn) {
        void navigate(`/device/detail/${device.sn}`);
      }
    },
    [navigate]
  );

  return (
    <div ref={dashboardRef} style={{ padding: '0 0 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Row 1: KPI Cards */}
      <Row gutter={[16, 16]} className="omc-scroll-reveal" data-delay="0">
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.totalDevices')}
            value={totalDevices}
            icon={<AppstoreOutlined />}
            iconBgColor="#e6f4ff"
            iconColor={token.colorPrimary}
            loading={isLoading}
            trend="up"
            delta="+12"
            deltaLabel={t('dashboard.vsLastWeek')}
            onClick={() => void navigate('/device/list')}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.onlineDevices')}
            value={onlineDevices}
            icon={<WifiOutlined />}
            iconBgColor="#f6ffed"
            iconColor="#52C41A"
            loading={isLoading}
            trend="up"
            delta={`${Math.round((onlineDevices / totalDevices) * 100)}%`}
            deltaLabel={t('dashboard.onlineRate')}
            onClick={() => void navigate('/device/list')}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.activeAlarms')}
            value={activeAlarms}
            icon={<AlertOutlined />}
            iconBgColor="#fff2f0"
            iconColor="#F5222D"
            loading={isLoading}
            trend="down"
            delta="-5"
            deltaLabel={t('dashboard.vsYesterday')}
            onClick={() => void navigate('/alarm/current')}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.runningTasks')}
            value={runningTasks}
            icon={<PlayCircleOutlined />}
            iconBgColor="#f9f0ff"
            iconColor="#722ED1"
            loading={isLoading}
            onClick={() => void navigate('/ops/tasks')}
          />
        </Col>
      </Row>

      {/* Row 2: Alarm Summary */}
      <Row gutter={[16, 16]} className="omc-scroll-reveal" data-delay="1">
        <Col xs={24} lg={16}>
          <TiltCard maxTilt={6}>
          <Card
            title={t('dashboard.alarmSummary')}
            size="small"
            extra={
              <a onClick={() => void navigate('/alarm/current')} style={{ fontSize: 13 }}>
                {t('dashboard.viewAll')}
              </a>
            }
            styles={{ body: { padding: 0 } }}
          >
            {/* Severity count tags */}
            <div style={{ padding: '12px 16px', display: 'flex', gap: 12, flexWrap: 'wrap' }}>
              {(['critical', 'major', 'minor', 'warning'] as const).map((sev) => {
                const counts: Record<string, number> = { critical, major, minor, warning };
                return (
                  <div
                    key={sev}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      padding: '6px 16px',
                      background: `${SEVERITY_COLOR[sev]}10`,
                      border: `1px solid ${SEVERITY_COLOR[sev]}40`,
                      borderRadius: 8,
                      cursor: 'pointer',
                    }}
                    onClick={() => void navigate('/alarm/current')}
                  >
                    <span
                      style={{
                        display: 'inline-block',
                        width: 10,
                        height: 10,
                        borderRadius: '50%',
                        background: SEVERITY_COLOR[sev],
                      }}
                    />
                    <Text style={{ fontSize: 13, color: token.colorText }}>{SEVERITY_LABEL[sev]}</Text>
                    <Text strong style={{ fontSize: 18, color: SEVERITY_COLOR[sev] }}>
                      {counts[sev]}
                    </Text>
                  </div>
                );
              })}
            </div>

            {/* Recent alarm list */}
            <List
              size="small"
              loading={isCurrentAlarmsLoading}
              dataSource={recentAlarms}
              style={{ padding: '0 8px 8px' }}
              renderItem={(alarm) => (
                <List.Item
                  style={{ padding: '6px 8px', borderRadius: 4 }}
                  onClick={() => void navigate('/alarm/current')}
                >
                  <Space size={8} style={{ width: '100%', justifyContent: 'space-between' }}>
                    <Space size={8}>
                      <Tag
                        color={
                          alarm.severity === 'critical'
                            ? 'red'
                            : alarm.severity === 'major'
                            ? 'orange'
                            : alarm.severity === 'minor'
                            ? 'gold'
                            : 'blue'
                        }
                        style={{ margin: 0, fontSize: 11 }}
                      >
                        {SEVERITY_LABEL[alarm.severity]}
                      </Tag>
                      <Text style={{ fontSize: 13 }}>{alarm.alarmName}</Text>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {alarm.deviceName}
                      </Text>
                    </Space>
                    <Text type="secondary" style={{ fontSize: 12, flexShrink: 0 }}>
                      {alarm.eventTime}
                    </Text>
                  </Space>
                </List.Item>
              )}
            />
          </Card>
          </TiltCard>
        </Col>

        <Col xs={24} lg={8}>
          <TiltCard maxTilt={8}>
          <Card title={t('dashboard.alarmDistribution')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <PieChart
              title=""
              data={alarmPieData}
              height={220}
              donut
            />
          </Card>
          </TiltCard>
        </Col>
      </Row>

      {/* Row 3: Device Status Chart + Alarm Trend */}
      <Row gutter={[16, 16]} className="omc-scroll-reveal" data-delay="2">
        <Col xs={24} lg={12}>
          <TiltCard maxTilt={7}>
          <Card title={t('dashboard.deviceStatusByType')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <BarChart
              title=""
              xData={deviceStatusXData}
              series={deviceStatusSeries}
              height={260}
            />
          </Card>
          </TiltCard>
        </Col>
        <Col xs={24} lg={12}>
          <TiltCard maxTilt={7}>
          <Card title={t('dashboard.alarmTrend7d')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <LineChart
              title=""
              xData={trendXData}
              series={alarmTrendSeries}
              height={260}
              areaFill
            />
          </Card>
          </TiltCard>
        </Col>
      </Row>

      {/* Row 4: TOP10 Devices + GIS Map */}
      <Row gutter={[16, 16]} className="omc-scroll-reveal" data-delay="3">
        <Col xs={24} lg={10}>
          <TiltCard maxTilt={7}>
          <Card title={t('dashboard.top10AlarmDevices')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <BarChart
              title=""
              xData={top10Devices}
              series={top10Series}
              height={280}
              horizontal
            />
          </Card>
          </TiltCard>
        </Col>
        <Col xs={24} lg={14}>
          <TiltCard maxTilt={5}>
          <Card title={t('dashboard.deviceMap')} size="small" styles={{ body: { padding: 8 } }}>
            <GISMap
              devices={MOCK_MAP_DEVICES}
              height={280}
              showStats={false}
              onDeviceClick={handleDeviceClick}
            />
          </Card>
          </TiltCard>
        </Col>
      </Row>

      {/* Row 5: User Profile + Quick Access */}
      <Row gutter={[16, 16]} className="omc-scroll-reveal" data-delay="4">
        <Col xs={24} lg={6}>
          <TiltCard maxTilt={8}>
          <Card size="small" styles={{ body: { padding: '20px 16px' } }}>
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
              <Avatar size={64} icon={<UserOutlined />} style={{ background: token.colorPrimary }} />
              <Title level={5} style={{ margin: 0 }}>
                {t('dashboard.sysAdmin')}
              </Title>
              <Text type="secondary" style={{ fontSize: 13 }}>
                admin@omc.com
              </Text>
              <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
                <Badge color="green" text={t('status.online')} />
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('dashboard.lastLogin')} 09:00
                </Text>
              </div>
              <div
                style={{
                  width: '100%',
                  marginTop: 8,
                  padding: '8px 12px',
                  background: token.colorBgLayout,
                  borderRadius: 6,
                  display: 'grid',
                  gridTemplateColumns: '1fr 1fr',
                  gap: 8,
                }}
              >
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontWeight: 700, fontSize: 18, color: token.colorPrimary }}>156</div>
                  <div style={{ fontSize: 12, color: token.colorTextSecondary }}>{t('dashboard.todayOps')}</div>
                </div>
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontWeight: 700, fontSize: 18, color: '#52C41A' }}>23</div>
                  <div style={{ fontSize: 12, color: token.colorTextSecondary }}>{t('dashboard.processedAlarms')}</div>
                </div>
              </div>
            </div>
          </Card>
          </TiltCard>
        </Col>

        <Col xs={24} lg={18}>
          <TiltCard maxTilt={5}>
          <Card title={t('dashboard.quickAccess')} size="small">
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(100px, 1fr))',
                gap: 12,
              }}
            >
              {QUICK_ACCESS_ITEMS.map((item) => (
                <div
                  key={item.path}
                  onClick={() => void navigate(item.path)}
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    gap: 8,
                    padding: '16px 8px',
                    borderRadius: 8,
                    background: token.colorBgLayout,
                    cursor: 'pointer',
                    border: `1px solid ${token.colorBorderSecondary}`,
                    transition: 'all 0.2s',
                  }}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLDivElement).style.background = token.controlItemBgActiveHover;
                    (e.currentTarget as HTMLDivElement).style.borderColor = token.colorPrimaryBorderHover;
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLDivElement).style.background = token.colorBgLayout;
                    (e.currentTarget as HTMLDivElement).style.borderColor = token.colorBorderSecondary;
                  }}
                >
                  <div
                    style={{
                      width: 36,
                      height: 36,
                      borderRadius: 8,
                      background: `${item.color}15`,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 18,
                      color: item.color,
                    }}
                  >
                    {item.icon}
                  </div>
                  <Text style={{ fontSize: 12, textAlign: 'center' }}>{t(item.labelKey)}</Text>
                </div>
              ))}
            </div>
          </Card>
          </TiltCard>
        </Col>
      </Row>
    </div>
  );
}
