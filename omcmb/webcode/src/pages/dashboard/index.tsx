/**
 * Dashboard页面 - v2.0 Panel化设计
 *
 * 基于老系统Dashboard重构，支持：
 * - LTE (eNB): 6个Panel（2×3网格）
 * - NR (gNB): 2个Panel（1×2布局）
 * - GSM: 3个Panel（第一行2个，第二行1个占满）
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Avatar,
  Badge,
  Button,
  Card,
  Col,
  Row,
  Segmented,
  Space,
  Spin,
  Typography,
  message,
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
  ReloadOutlined,
  RocketOutlined,
  SettingOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  UserOutlined,
  WifiOutlined,
} from '@ant-design/icons';
import KPICard from '@/components/KPICard';
import BarChart from '@/components/Charts/BarChart';
import EmptyState from '@/components/common/EmptyState';
import { useDashboardData, useDeviceStatusByType } from '@core/hooks/api/useDashboard';
import { DashboardKPIModules } from './DashboardKPIModules';
import type { TechnologyType } from './kpi-config';
import { useTechnologyDictionary } from '@/components/dashboard/useTechnologyDictionary';
import { useUserStore } from '@core/store/userStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useQueryClient } from '@tanstack/react-query';
import { TiltCard } from '@/components/Effects';
import { useScrollReveal } from '@/hooks/useScrollReveal';
import { formatTimeAgo } from '@core/utils/format';

const { Title, Text } = Typography;

// Technology display name mapping
const TECH_DISPLAY_NAME: Record<string, string> = {
  lte: 'LTE',
  nr: '5G NR',
  gsm: 'GSM',
};

// Dashboard feature visibility configuration
const DASHBOARD_CONFIG = {
  showRunningTasks: false,        // 任务执行中
  showRefreshControls: false,     // 刷新控制栏
  showDeviceMap: false,           // 设备地图
} as const;

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
];

export default function DashboardPage() {
  const navigate = useNavigate();
  const { data: dashboardData, isLoading } = useDashboardData();
  const t = useT();
  const token = useThemeToken();
  const queryClient = useQueryClient();

  // 制式切换状态 - 默认使用LTE（符合验收标准：LTE 6个Panel作为主要展示）
  const [technology, setTechnology] = useState<TechnologyType>('lte');

  // 网络制式 Segmented 选项来自字典 `network_type`：字典有几项显示几项；
  // hook 内已过滤掉 LTE/NR/GSM 之外的 value（前端 KPI 静态契约暂未放开），
  // 也过滤 status=false 项，并按 sort 升序。字典空 / loading / error → 空数组，
  // UI 显示空 Segmented，让运维感知字典缺失（决策 D4）。
  const { options: techOptions } = useTechnologyDictionary();

  // 字典禁用了当前选中项 → 回退到 options 首项，避免下方图表区找不到 panel
  useEffect(() => {
    if (techOptions.length && !techOptions.some((o) => o.value === technology)) {
      setTechnology(techOptions[0].value);
    }
  }, [techOptions, technology]);

  // 刷新提示状态
  const [lastUpdateTime, setLastUpdateTime] = useState<Date>(new Date());
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [timeAgoText, setTimeAgoText] = useState('');

  // 更新时间倒计时文案
  useEffect(() => {
    const updateTime = () => {
      setTimeAgoText(formatTimeAgo(lastUpdateTime, t));
    };

    updateTime();
    const interval = setInterval(updateTime, 10000);

    return () => clearInterval(interval);
  }, [lastUpdateTime, t]);

  // 手动刷新处理
  const handleManualRefresh = useCallback(async () => {
    if (isRefreshing) return;
    setIsRefreshing(true);
    try {
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      setLastUpdateTime(new Date());
      message.success(t('dashboard.refreshSuccess'));
    } catch (error) {
      message.error(t('dashboard.refreshFailed'));
      console.error('Dashboard refresh failed:', error);
    } finally {
      setTimeout(() => setIsRefreshing(false), 500);
    }
  }, [queryClient, isRefreshing, t]);

  // 获取当前登录用户信息
  const currentUser = useUserStore((state) => state.currentUser);

  // 获取设备按技术类型分组的状态数据
  const { data: deviceStatusByTypeData, isLoading: isDeviceStatusLoading } = useDeviceStatusByType();

  // KPI values — 用真实数据；未就绪时回退 0（加载期由 KPICard 的 loading 态显示骨架，
  // 不再展示硬编码占位数字，避免硬刷新闪现假数。issue #370）
  const totalDevices = dashboardData?.summary?.deviceCounts?.total ?? 0;
  const onlineDevices = dashboardData?.summary?.deviceCounts?.online ?? 0;
  const activeAlarms = dashboardData?.summary?.alarmCounts?.total ?? 0;
  const runningTasks = dashboardData?.summary?.taskSummary?.running ?? 0;

  // KPI trend data — 用于卡片显示趋势
  const kpiDeltas = dashboardData?.summary?.kpiDeltas ?? {};
  const totalDevicesDelta = kpiDeltas['total_devices'];
  const activeAlarmsDelta = kpiDeltas['active_alarms'];
  const ueTrendDelta = kpiDeltas['UE_ACTIVE'];

  // UE 当前值 — 无数据时保持 undefined，UI 显示 '--' 区分"真 0"与"无数据"
  const kpiSummary = dashboardData?.summary?.kpiSummary ?? {};
  const ueRaw = kpiSummary['UE_ACTIVE'];
  const currentActiveUE = typeof ueRaw === 'number' ? Math.floor(ueRaw) : undefined;

  // Device status bar chart data - 按技术类型分组
  const deviceStatusData = useMemo(() => {
    if (!deviceStatusByTypeData || Object.keys(deviceStatusByTypeData).length === 0) {
      return { isEmpty: true, xData: [], series: [] };
    }

    const xData = Object.keys(deviceStatusByTypeData).map(
      key => TECH_DISPLAY_NAME[key] || key
    );

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

  // Alarm distribution bar chart data - 按告警等级分组统计
  const alarmDistributionData = useMemo(() => {
    const alarmCounts = dashboardData?.summary?.alarmCounts;

    if (!alarmCounts) {
      return { isEmpty: true, xData: [], series: [] };
    }

    const hasAlarms = alarmCounts.critical > 0 || alarmCounts.major > 0 ||
                      alarmCounts.minor > 0 || alarmCounts.warning > 0;

    if (!hasAlarms) {
      return { isEmpty: true, xData: [], series: [] };
    }

    const xData = [
      t('alarm.severity.critical'),
      t('alarm.severity.major'),
      t('alarm.severity.minor'),
      t('alarm.severity.warning'),
    ];

    const data = [
      { value: alarmCounts.critical, name: t('alarm.severity.critical') },
      { value: alarmCounts.major, name: t('alarm.severity.major') },
      { value: alarmCounts.minor, name: t('alarm.severity.minor') },
      { value: alarmCounts.warning, name: t('alarm.severity.warning') },
    ];

    const series = [
      {
        name: t('dashboard.alarmCount'),
        data: data as Array<{ value: number; name: string }>,
      },
    ];

    return { isEmpty: false, xData, series };
  }, [dashboardData, t]);

  const dashboardRef = useRef<HTMLDivElement>(null);
  useScrollReveal(dashboardRef);

  const handleAlarmChartClick = useCallback(
    (_index: number, _name: string) => {
      void navigate('/alarm/current');
    },
    [navigate]
  );

  return (
    <div ref={dashboardRef} style={{ padding: '0 0 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Dashboard Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('nav.dashboard')}
        </Title>
        {/* 刷新控制栏 */}
        {DASHBOARD_CONFIG.showRefreshControls && (
        <Space size="middle">
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            {t('dashboard.lastUpdate')}: {timeAgoText || formatTimeAgo(lastUpdateTime, t)}
          </Typography.Text>
          <Button
            icon={<ReloadOutlined />}
            loading={isRefreshing || isLoading}
            onClick={handleManualRefresh}
            size="small"
          >
            {t('dashboard.refresh')}
          </Button>
        </Space>
        )}
      </div>

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
            trend={totalDevicesDelta?.trend ?? 'stable'}
            delta={totalDevicesDelta?.changePercent !== undefined ? `${totalDevicesDelta.changePercent.toFixed(1)}%` : undefined}
            deltaLabel={totalDevicesDelta?.compareType === 'last_week' ? t('dashboard.vsLastWeek') : t('dashboard.vsYesterday')}
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
            delta={totalDevices > 0
              ? `${Math.round((onlineDevices / totalDevices) * 100)}%`
              : '--'}
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
            trend={activeAlarmsDelta?.trend ?? 'stable'}
            delta={activeAlarmsDelta?.changePercent !== undefined ? `${activeAlarmsDelta.changePercent.toFixed(1)}%` : undefined}
            deltaLabel={activeAlarmsDelta?.compareType === 'yesterday' ? t('dashboard.vsYesterday') : t('dashboard.vsLastWeek')}
            onClick={() => void navigate('/alarm/current')}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.activeUE')}
            value={currentActiveUE ?? '--'}
            icon={<TeamOutlined />}
            iconBgColor="#f6ffed"
            iconColor="#10B981"
            loading={isLoading}
            trend={ueTrendDelta?.trend ?? 'stable'}
            delta={ueTrendDelta?.changePercent !== undefined ? `${ueTrendDelta.changePercent.toFixed(1)}%` : undefined}
            deltaLabel={ueTrendDelta?.changePercent !== undefined ? t('dashboard.vsLastWeek') : undefined}
            minHeight={20}
          />
        </Col>
        {DASHBOARD_CONFIG.showRunningTasks && (
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.runningTasks')}
            value={runningTasks}
            icon={<PlayCircleOutlined />}
            iconBgColor="#f9f0ff"
            iconColor="#722ED1"
            loading={isLoading}
            trend="stable"
            minHeight={20}
            onClick={() => void navigate('/ops/tasks')}
          />
        </Col>
        )}
      </Row>

      {/* 制式切换栏 */}
      <Row gutter={[16, 16]} className="omc-scroll-reveal" data-delay="1">
        <Col span={24}>
          <Space size="middle" style={{ width: '100%', justifyContent: 'space-between', alignItems: 'center' }}>
            <Space size="middle">
              <Text type="secondary">{t('dashboard.networkTech')}:</Text>
              <Segmented
                value={technology}
                onChange={(value) => setTechnology(value as TechnologyType)}
                options={techOptions}
              />
            </Space>
          </Space>
        </Col>
      </Row>

      {/* KPI Panel区域 - v2.0 Panel化设计 */}
      <DashboardKPIModules technology={technology} />

      {/* Row 3: Device Status + Alarm Statistics */}
      <Row gutter={[16, 16]} align="stretch" className="omc-scroll-reveal" data-delay="2">
        <Col xs={24} lg={12} style={{ display: 'flex' }}>
          <TiltCard maxTilt={7} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.deviceStatusByType')}
            size="small"
            extra={<span style={{ visibility: 'hidden', fontSize: 13 }}>……</span>}
            styles={{ body: { padding: '8px 0 0', flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
            {isDeviceStatusLoading ? (
              <Spin
                spinning={isDeviceStatusLoading}
                style={{ height: 260, width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
              >
                <div style={{ height: 260 }} />
              </Spin>
            ) : deviceStatusData.isEmpty ? (
              <EmptyState description={t('common.noData')} />
            ) : (
              <BarChart
                title=""
                xData={deviceStatusData.xData}
                series={deviceStatusData.series}
                height={260}
                barWidth={32}
              />
            )}
          </Card>
          </TiltCard>
        </Col>

        <Col xs={24} lg={12} style={{ display: 'flex' }}>
          <TiltCard maxTilt={7} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.alarmLevelStatistics')}
            size="small"
            extra={
              <a onClick={() => void navigate('/alarm/current')} style={{ fontSize: 13 }}>
                {t('dashboard.viewAll')}
              </a>
            }
            styles={{ body: { padding: '8px 0 0', flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
            {isLoading ? (
              <Spin
                spinning={isLoading}
                style={{ height: 260, width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
              >
                <div style={{ height: 260 }} />
              </Spin>
            ) : alarmDistributionData.isEmpty ? (
              <EmptyState description={t('alarm.noActiveAlarms')} />
            ) : (
              <BarChart
                title=""
                xData={alarmDistributionData.xData}
                series={alarmDistributionData.series}
                height={260}
                barWidth={32}
                showLegend={false}
                onClick={handleAlarmChartClick}
              />
            )}
          </Card>
          </TiltCard>
        </Col>
      </Row>

      {/* Row 4: User Profile + Quick Access */}
      <Row gutter={[16, 16]} align="stretch" className="omc-scroll-reveal" data-delay="3">
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
              <div
                style={{
                  width: '60%',
                  height: 1,
                  background: `linear-gradient(90deg, transparent, ${token.colorBorder}, transparent)`,
                  marginTop: 8,
                }}
              />
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
