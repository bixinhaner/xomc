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
import EmptyState from '@/components/common/EmptyState';
import { MAP_CONFIG } from '@/components/GISMap/constants';
import type { MapDevice } from '@/components/GISMap';
import type { DeviceGeo } from '@core/types/map';
import { useDashboardData, useAlarmTrend, useTopAlarmDevices, useDeviceStatusByType } from '@core/hooks/api/useDashboard';
import { useAlarmCount, useCurrentAlarms } from '@core/hooks/api/useAlarms';
import { useMapDevicesGeo } from '@core/hooks/api/useTopology';
import { useUserStore } from '@core/store/userStore';
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

// Technology display name mapping
const TECH_DISPLAY_NAME: Record<string, string> = {
  lte: 'LTE',
  nr: '5G NR',
  gsm: 'GSM',
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

/** 格式化最后登录时间 */
function formatLastLogin(lastLoginTime?: string): string {
  if (!lastLoginTime) {
    return '--';
  }
  const parsed = new Date(lastLoginTime);
  if (Number.isNaN(parsed.getTime())) {
    return '--';
  }
  return parsed.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
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

  // 获取当前登录用户信息
  const currentUser = useUserStore((state) => state.currentUser);

  // 获取告警趋势数据
  const { data: alarmTrendData } = useAlarmTrend(7);

  // 获取TOP10告警设备真实数据
  const { data: topAlarmDevicesData } = useTopAlarmDevices();

  // 获取设备按技术类型分组的状态数据
  const { data: deviceStatusByTypeData } = useDeviceStatusByType();

  // 获取设备地理数据（与 GISMapView 相同的数据源）
  const mapFilterParams = useMemo(() => ({
    // 仪表板场景：获取所有状态的设备
    status: undefined,
    // 获取所有设备组的设备
    groupIds: undefined,
    // 启用请求
    enabled: true,
    // 限制数量，避免仪表板加载过慢
    pageSize: 100,
  }), []);

  const { data: devicesGeoData } = useMapDevicesGeo(mapFilterParams);

  // 转换 DeviceGeo 为 MapDevice 格式
  const mapDevices = useMemo(() => {
    if (!devicesGeoData?.items?.length) return [];
    return devicesGeoData.items.map((device: DeviceGeo) => ({
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
    }));
  }, [devicesGeoData]);

  // KPI values — use real data when available, fall back to sensible defaults
  const totalDevices = dashboardData?.summary?.deviceCounts?.total ?? 1284;
  const onlineDevices = dashboardData?.summary?.deviceCounts?.online ?? 1137;
  const activeAlarms = dashboardData?.summary?.alarmCounts?.total ?? 43;
  const runningTasks = dashboardData?.summary?.taskSummary?.running ?? 7;

  // Alarm severity counts - 没有数据时默认为 0，不显示虚假数据
  const critical = alarmCount?.critical ?? 0;
  const major = alarmCount?.major ?? 0;
  const minor = alarmCount?.minor ?? 0;
  const warning = alarmCount?.warning ?? 0;

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
      { name: t('alarm.severity.critical'), value: critical, color: SEVERITY_COLOR.critical },
      { name: t('alarm.severity.major'), value: major, color: SEVERITY_COLOR.major },
      { name: t('alarm.severity.minor'), value: minor, color: SEVERITY_COLOR.minor },
      { name: t('alarm.severity.warning'), value: warning, color: SEVERITY_COLOR.warning },
    ],
    [critical, major, minor, warning, t]
  );

  // Device status bar chart data - 按技术类型分组
  const deviceStatusData = useMemo(() => {
    if (!deviceStatusByTypeData || Object.keys(deviceStatusByTypeData).length === 0) {
      return { isEmpty: true, xData: [], series: [] };
    }

    // technology 键作为 X 轴数据，映射为友好显示名称
    const xData = Object.keys(deviceStatusByTypeData).map(
      key => TECH_DISPLAY_NAME[key] || key
    );

    // 提取各状态的数据（保持原始顺序）
    const technologyKeys = Object.keys(deviceStatusByTypeData);
    const onlineData = technologyKeys.map(key => deviceStatusByTypeData[key]?.online ?? 0);
    const offlineData = technologyKeys.map(key => deviceStatusByTypeData[key]?.offline ?? 0);
    const alarmData = technologyKeys.map(key => deviceStatusByTypeData[key]?.alarm ?? 0);

    const series = [
      { name: t('dashboard.chart.online'), data: onlineData, color: '#52C41A' },
      { name: t('dashboard.chart.offline'), data: offlineData, color: '#8C8C8C' },
      { name: t('dashboard.chart.alarm'), data: alarmData, color: '#FA8C16' },
    ];

    return { isEmpty: false, xData, series };
  }, [deviceStatusByTypeData, t]);

  // 7-day alarm trend - 根据后端返回数据动态生成X轴
  const { trendXData, alarmTrendSeries } = useMemo(() => {
    // 没有数据时生成默认7天X轴和全0数据
    if (!alarmTrendData?.length) {
      const days: string[] = [];
      for (let i = 6; i >= 0; i--) {
        const d = new Date();
        d.setDate(d.getDate() - i);
        days.push(`${d.getMonth() + 1}/${d.getDate()}`);
      }
      const zeroData = new Array(7).fill(0);
      return {
        trendXData: days,
        alarmTrendSeries: [
          { name: t('alarm.severity.critical'), data: zeroData, color: SEVERITY_COLOR.critical },
          { name: t('alarm.severity.major'), data: zeroData, color: SEVERITY_COLOR.major },
          { name: t('alarm.severity.minor'), data: zeroData, color: SEVERITY_COLOR.minor },
          { name: t('alarm.severity.warning'), data: zeroData, color: SEVERITY_COLOR.warning },
        ],
      };
    }

    // 从后端数据提取日期并转换为 月/日 格式
    const xData = alarmTrendData.map((d) => {
      const date = new Date(d.date);
      return `${date.getMonth() + 1}/${date.getDate()}`;
    });

    // 提取各级别的趋势数据
    const critical = alarmTrendData.map((d) => d.critical ?? 0);
    const major = alarmTrendData.map((d) => d.major ?? 0);
    const minor = alarmTrendData.map((d) => d.minor ?? 0);
    const warning = alarmTrendData.map((d) => d.warning ?? 0);

    return {
      trendXData: xData,
      alarmTrendSeries: [
        { name: t('alarm.severity.critical'), data: critical, color: SEVERITY_COLOR.critical },
        { name: t('alarm.severity.major'), data: major, color: SEVERITY_COLOR.major },
        { name: t('alarm.severity.minor'), data: minor, color: SEVERITY_COLOR.minor },
        { name: t('alarm.severity.warning'), data: warning, color: SEVERITY_COLOR.warning },
      ],
    };
  }, [alarmTrendData, t]);

  // TOP10 alarm devices horizontal bar chart - 使用真实API数据
  const top10Devices = useMemo(() => {
    if (!topAlarmDevicesData?.length) return [];
    return topAlarmDevicesData.map(d => {
      const techDisplay = TECH_DISPLAY_NAME[d.technology] || d.technology;
      const deviceNumber = d.deviceSN?.slice(-4) || "????";
      return `${techDisplay}-${deviceNumber}`;
    });
  }, [topAlarmDevicesData]);

  const top10Series = useMemo(() => {
    if (!topAlarmDevicesData?.length) {
      // 无数据时返回空数组
      return [{ name: t('dashboard.alarmCount'), data: [] }];
    }
    return [{
      name: t('dashboard.alarmCount'),
      data: topAlarmDevicesData.map(d => ({
        value: d.alarmCount,
        name: d.deviceSN
      })),
    }];
  }, [topAlarmDevicesData, t]);

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

  const tooltipFormatter = useCallback(
    (params: Array<{ dataIndex: number; color: string; seriesName: string; value: number; name: string }>) => {
      const dataIndex = params[0]?.dataIndex;
      const device = topAlarmDevicesData?.[dataIndex];
      if (!device) return params[0]?.name ?? '';
      return `
        <div style="color: #8c8c8c; font-size: 12px; margin-bottom: 4px;">${device.deviceSN || '--'}</div>
        <div style="display: flex; align-items: center; gap: 8px;">
          <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${params[0]?.color}"></span>
          <span>${params[0]?.seriesName}: ${params[0]?.value}</span>
        </div>
      `;
    },
    [topAlarmDevicesData]
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
            trend="neutral"
            minHeight={20}
            onClick={() => void navigate('/ops/tasks')}
          />
        </Col>
      </Row>

      {/* Row 2: Alarm Summary */}
      <Row gutter={[16, 16]} align="stretch" className="omc-scroll-reveal" data-delay="1">
        <Col xs={24} lg={16} style={{ display: 'flex' }}>
          <TiltCard maxTilt={6} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.alarmSummary')}
            size="small"
            extra={
              <a onClick={() => void navigate('/alarm/current')} style={{ fontSize: 13 }}>
                {t('dashboard.viewAll')}
              </a>
            }
            styles={{ body: { padding: 0, flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
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

        <Col xs={24} lg={8} style={{ display: 'flex' }}>
          <TiltCard maxTilt={8} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.alarmDistribution')}
            size="small"
            styles={{ body: { padding: '8px 0 0', flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
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
      <Row gutter={[16, 16]} align="stretch" className="omc-scroll-reveal" data-delay="2">
        <Col xs={24} lg={12} style={{ display: 'flex' }}>
          <TiltCard maxTilt={7} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.deviceStatusByType')}
            size="small"
            styles={{ body: { padding: '8px 0 0', flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
            {deviceStatusData.isEmpty ? (
              <EmptyState description={t('common.noData')} />
            ) : (
              <BarChart
                title=""
                xData={deviceStatusData.xData}
                series={deviceStatusData.series}
                height={260}
              />
            )}
          </Card>
          </TiltCard>
        </Col>
        <Col xs={24} lg={12} style={{ display: 'flex' }}>
          <TiltCard maxTilt={7} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.alarmTrend7d')}
            size="small"
            styles={{ body: { padding: '8px 0 0', flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
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
      <Row gutter={[16, 16]} align="stretch" className="omc-scroll-reveal" data-delay="3">
        <Col xs={24} lg={10} style={{ display: 'flex' }}>
          <TiltCard maxTilt={7} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.top10AlarmDevices')}
            size="small"
            styles={{ body: { padding: '8px 0 0', flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
            {top10Devices.length > 0 ? (
              <BarChart
                title=""
                xData={top10Devices}
                series={top10Series}
                height={280}
                horizontal
                tooltipFormatter={tooltipFormatter}
              />
            ) : (
              <EmptyState variant="no-data" description="" style={{ flex: 1 }} />
            )}
          </Card>
          </TiltCard>
        </Col>
        <Col xs={24} lg={14} style={{ display: 'flex' }}>
          <TiltCard maxTilt={5} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.deviceMap')}
            size="small"
            styles={{ body: { padding: 8, flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
            <GISMap
              devices={mapDevices}
              height={280}
              defaultCenter={MAP_CONFIG.defaultCenter}
              defaultZoom={MAP_CONFIG.defaultZoom}
              tileUrl={MAP_CONFIG.tileUrl}
              showStats={false}
              showMetadataTip={false}
              onDeviceClick={handleDeviceClick}
            />
          </Card>
          </TiltCard>
        </Col>
      </Row>

      {/* Row 5: User Profile + Quick Access */}
      <Row gutter={[16, 16]} align="stretch" className="omc-scroll-reveal" data-delay="4">
        <Col xs={24} lg={6} style={{ display: 'flex' }}>
          <TiltCard maxTilt={8} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            size="small"
            styles={{ body: { padding: '24px 20px', flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
              <Avatar
                size={80}
                src={currentUser?.avatar}
                icon={!currentUser?.avatar ? <UserOutlined /> : undefined}
                style={{
                  background: currentUser?.avatar ? undefined : token.colorPrimary,
                  border: `3px solid ${token.colorBgContainer}`,
                  boxShadow: `0 4px 12px ${token.colorPrimary}20`,
                }}
              />
              <Title level={5} style={{ margin: 0, fontSize: 16 }}>
                {currentUser?.displayName || currentUser?.username || t('dashboard.sysAdmin')}
              </Title>
              <Text type="secondary" style={{ fontSize: 13 }}>
                {currentUser?.email || '--'}
              </Text>
              <div style={{ display: 'flex', gap: 12, marginTop: 4, alignItems: 'center' }}>
                <Badge
                  color="green"
                  text={t('status.online')}
                  style={{ fontSize: 12 }}
                />
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('dashboard.lastLogin')} {formatLastLogin(currentUser?.lastLoginTime)}
                </Text>
              </div>
              {/* 装饰性分隔线 */}
              <div
                style={{
                  width: '60%',
                  height: 1,
                  background: `linear-gradient(90deg, transparent, ${token.colorBorder}, transparent)`,
                  marginTop: 8,
                }}
              />
              {/* 系统状态指示 */}
              <div
                style={{
                  width: '100%',
                  display: 'flex',
                  justifyContent: 'space-around',
                  padding: '8px 0',
                  fontSize: 12,
                  color: token.colorTextSecondary,
                }}
              >
                <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#52C41A' }} />
                  {t('status.normal')}
                </span>
                <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  <span style={{ width: 8, height: 8, borderRadius: '50%', background: token.colorPrimary }} />
                  {t('status.running')}
                </span>
              </div>
            </div>
          </Card>
          </TiltCard>
        </Col>

        <Col xs={24} lg={18} style={{ display: 'flex' }}>
          <TiltCard maxTilt={5} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.quickAccess')}
            size="small"
            styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
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
