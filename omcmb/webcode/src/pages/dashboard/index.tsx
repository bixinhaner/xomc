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
import {
  useDashboardDeviceStats,
  useDashboardSummary,
  useDeviceStatusByType,
} from '@core/hooks/api/useDashboard';
import { DashboardKPIModules } from './DashboardKPIModules';
import type { TechnologyType } from './kpi-config';
import { useTechnologyDictionary } from '@core/hooks/api/useTechnologyDictionary';
import { useAppStore } from '@core/store/appStore';
import { useMenuStore } from '@core/store/menuStore';
import { useUserStore } from '@core/store/userStore';
import { useAlarmStore } from '@core/store/alarmStore';
import { useTabStore } from '@core/store/tabStore';
import { useShallow } from 'zustand/react/shallow';
import { isRouteAllowed } from '@core/utils/routeAccess';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { TiltCard } from '@/components/Effects';
import { isDynamicMenuEnabled } from '@/components/MenuBootstrap/featureFlag';
import { useScrollReveal } from '@/hooks/useScrollReveal';
import { formatTimeAgo } from '@core/utils/format';
import {
  createDashboardCardSnapshot,
  readDashboardCardSnapshot,
  resolveDashboardCardApiScope,
  resolveDashboardCardDisplay,
  writeDashboardCardSnapshot,
} from '@core/utils/dashboardCardSnapshot';
import { runDashboardSummaryRefresh } from '@core/utils/dashboardRefresh';

const { Title, Text } = Typography;

// Dashboard feature visibility configuration
const DASHBOARD_CONFIG = {
  showRunningTasks: false,        // 任务执行中
  showRefreshControls: false,     // 刷新控制栏
  showDeviceMap: false,           // 设备地图
  showUserProfileAndQuickAccess: false, // 用户信息与快速入口（暂不展示）
} as const;

/** 格式化最后登录时间 */
function formatLastLogin(lastLoginTime: string | undefined, locale: 'zh-CN' | 'en-US'): string {
  if (!lastLoginTime) {
    return '--';
  }
  const parsed = new Date(lastLoginTime);
  if (Number.isNaN(parsed.getTime())) {
    return '--';
  }
  return parsed.toLocaleTimeString(locale, {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}

const QUICK_ACCESS_ITEMS = [
  { labelKey: 'nav.device.list',       icon: <AppstoreOutlined />,    path: '/device/list',               color: '#1677FF', productionReady: true },
  { labelKey: 'nav.alarm.current',     icon: <AlertOutlined />,       path: '/alarm/current',             color: '#F5222D', productionReady: true },
  { labelKey: 'nav.alarm.statistics',  icon: <DashboardOutlined />,   path: '/alarm/statistics',          color: '#FA8C16', productionReady: true },
  { labelKey: 'nav.device.ne',         icon: <ApartmentOutlined />,   path: '/device/ne',                 color: '#52C41A', productionReady: false },
  { labelKey: 'nav.device.monitor',    icon: <MonitorOutlined />,     path: '/device/monitor',            color: '#722ED1', productionReady: false },
  { labelKey: 'nav.device.commission', icon: <RocketOutlined />,      path: '/device/commission',         color: '#13C2C2', productionReady: false },
  {
    labelKey: 'nav.product.kpiLibrary',
    icon: <ThunderboltOutlined />,
    path: '/product/kpi-library',
    color: '#EB2F96',
    productionReady: true,
    requireSuperAdmin: true,
  },
  { labelKey: 'nav.device.stats',      icon: <CloudServerOutlined />, path: '/device/stats',              color: '#2F54EB', productionReady: false },
  { labelKey: 'nav.alarm.rules',       icon: <SettingOutlined />,     path: '/alarm/rules',               color: '#8C8C8C', productionReady: true },
  { labelKey: 'nav.log.system',        icon: <FileTextOutlined />,    path: '/log/system',                color: '#595959', productionReady: false },
  { labelKey: 'nav.system.users',      icon: <TeamOutlined />,        path: '/system/users',              color: '#D46B08', productionReady: true },
];

export default function DashboardPage() {
  const navigate = useNavigate();
  const openTab = useTabStore((s) => s.openTab);
  const currentUser = useUserStore((state) => state.currentUser);
  const currentUserId = currentUser?.id;
  const dashboardApiScope = resolveDashboardCardApiScope(
    import.meta.env.VITE_API_BASE_URL,
    import.meta.env.VITE_API_PROXY_TARGET,
    window.location.origin,
  );
  const {
    data: dashboardSummary,
    dataUpdatedAt: dashboardUpdatedAt,
    isFetching,
    isPending,
    refetch: refetchDashboardSummary,
  } = useDashboardSummary(dashboardApiScope, currentUserId);
  const {
    data: dashboardDeviceStats,
    isPending: isDeviceStatsPending,
    isError: isDeviceStatsError,
  } = useDashboardDeviceStats(dashboardApiScope, currentUserId);
  // 每次渲染读取一个小型快照，确保成功 effect 写入后，后续任意 Query 状态变化
  // 都会拿到最近值；避免额外 state/effect 级联渲染，也不会跨用户复用。
  const dashboardSnapshot = currentUserId
    ? readDashboardCardSnapshot(
      window.localStorage,
      dashboardApiScope,
      currentUserId,
    )
    : undefined;
  const alarmStoreCounts = useAlarmStore(useShallow((s) => s.counts));
  const t = useT();
  const locale = useAppStore((state) => state.locale);
  const token = useThemeToken();

  // 制式切换状态 - 默认使用LTE（符合验收标准：LTE 6个Panel作为主要展示）
  const [technology, setTechnology] = useState<TechnologyType>('lte');

  // 网络制式 Segmented 选项来自字典 `network_type`：字典有几项显示几项；
  // hook 内已过滤掉 LTE/NR/GSM 之外的 value（前端 KPI 静态契约暂未放开），
  // 也过滤 status=false 项，并按 sort 升序。字典空 / loading / error → 空数组，
  // UI 显示空 Segmented，让运维感知字典缺失（决策 D4）。
  const { options: techOptions } = useTechnologyDictionary();

  const effectiveTechnology = useMemo(() => {
    if (techOptions.length === 0) {
      return technology;
    }
    return techOptions.some((o) => o.value === technology) ? technology : techOptions[0].value;
  }, [techOptions, technology]);

	const latestPMSlotHealth = useMemo(() => {
	  const rows = dashboardSummary?.pmSlotHealth?.filter(
		(slot) => slot.technology.toLowerCase() === effectiveTechnology,
	  ) ?? [];
	  if (rows.length === 0) return undefined;
	  const expectedDevices = rows.reduce((sum, row) => sum + row.expectedDevices, 0);
	  const receivedDevices = rows.reduce((sum, row) => sum + row.receivedDevices, 0);
	  const coverageRatio = expectedDevices > 0 ? receivedDevices / expectedDevices : 0;
	  return {
		...rows[0],
		expectedDevices,
		receivedDevices,
		coverageRatio,
		status: rows.some((row) => row.status === 'bootstrap_ignored')
		  ? 'bootstrap_ignored' as const
		  : coverageRatio >= 0.98
			? 'complete' as const
			: receivedDevices > 0
			  ? 'partial' as const
			  : 'missing' as const,
	  };
	}, [dashboardSummary?.pmSlotHealth, effectiveTechnology]);
	const pmSlotBadgeStatus = latestPMSlotHealth?.status === 'complete'
	  ? 'success'
	  : latestPMSlotHealth?.status === 'bootstrap_ignored'
		? 'processing'
		: latestPMSlotHealth?.status === 'partial'
		  ? 'warning'
		  : 'error';

  // 刷新提示状态
  const lastUpdateTime = useMemo(
    () => {
      const lastUpdatedAt = dashboardUpdatedAt || dashboardSnapshot?.updatedAt;
      return lastUpdatedAt ? new Date(lastUpdatedAt) : undefined;
    },
    [dashboardSnapshot?.updatedAt, dashboardUpdatedAt]
  );
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [timeAgoText, setTimeAgoText] = useState('');

  // 只在 Summary 成功返回后更新卡片快照；pending/error 不用默认值覆盖历史成功值。
  // 当前接入 UE 不复用该 PM 快照，始终以设备全量统计为准。
  useEffect(() => {
    if (!dashboardSummary || !currentUserId) return;

    const updatedAt = dashboardUpdatedAt || Date.now();
    const nextSnapshot = createDashboardCardSnapshot(dashboardSummary, updatedAt);
    writeDashboardCardSnapshot(
      window.localStorage,
      dashboardApiScope,
      currentUserId,
      nextSnapshot
    );
  }, [currentUserId, dashboardApiScope, dashboardSummary, dashboardUpdatedAt]);

  // 更新时间倒计时文案
  useEffect(() => {
    if (!lastUpdateTime) {
      return undefined;
    }

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
      await runDashboardSummaryRefresh(refetchDashboardSummary);
      message.success(t('dashboard.refreshSuccess'));
    } catch (error) {
      message.error(t('dashboard.refreshFailed'));
      if (import.meta.env.DEV) {
        console.error('Dashboard refresh failed:', error);
      }
    } finally {
      setTimeout(() => setIsRefreshing(false), 500);
    }
  }, [isRefreshing, refetchDashboardSummary, t]);

  // 获取当前登录用户信息
  const menuLoaded = useMenuStore((state) => state.loaded);
  const routePaths = useMenuStore((state) => state.routePaths);

  // 获取设备按技术类型分组的状态数据
  const { data: deviceStatusByTypeData, isLoading: isDeviceStatusLoading } = useDeviceStatusByType();

  // KPI values — 用真实数据；未就绪时回退 0（加载期由 KPICard 的 loading 态显示骨架，
  // 不再展示硬编码占位数字，避免硬刷新闪现假数。issue #370）
  const cardDisplay = resolveDashboardCardDisplay(
    dashboardSummary,
    dashboardSnapshot,
    isPending
  );
  const totalDevices = dashboardDeviceStats?.total ?? cardDisplay.totalDevices;
  const onlineDevices = dashboardDeviceStats?.online_count ?? cardDisplay.onlineDevices;
  const activeAlarms = cardDisplay.activeAlarms;
  const currentUE = dashboardDeviceStats?.current_ue_count ?? 0;
  const deviceCardsLoading = !dashboardDeviceStats && cardDisplay.loading;
  const currentUECardLoading = !dashboardDeviceStats && isDeviceStatsPending;
  const summaryCardsLoading = cardDisplay.loading;
  const runningTasks = dashboardSummary?.taskSummary?.running ?? 0;

  // KPI trend data — 用于卡片显示趋势
  const kpiDeltas = dashboardSummary?.kpiDeltas ?? {};
  const totalDevicesDelta = kpiDeltas['total_devices'];
  const activeAlarmsDelta = kpiDeltas['active_alarms'];

  const quickAccessItems = useMemo(
    () => QUICK_ACCESS_ITEMS.filter(
      (item) =>
        item.productionReady
        && (!item.requireSuperAdmin || currentUser?.isSuperAdmin === true)
        && isRouteAllowed(item.path, {
          role: currentUser?.role,
          isSuperAdmin: currentUser?.isSuperAdmin,
          routePaths,
          dynamicEnabled: isDynamicMenuEnabled(),
          menuLoaded,
        }),
    ),
    [currentUser?.role, currentUser?.isSuperAdmin, routePaths, menuLoaded]
  );

  // Device status bar chart data - 按技术类型分组
  // X 轴文案与上方 Segmented 同源 useTechnologyDictionary（字典 network_type），
  // 字典未命中时 fallback 到 key.toUpperCase()，避免后端返回字典未配的 tech 时柱图轴标签为空。
  const deviceStatusData = useMemo(() => {
    if (!deviceStatusByTypeData || Object.keys(deviceStatusByTypeData).length === 0) {
      return { isEmpty: true, xData: [], series: [] };
    }

    const techLabelMap = new Map(techOptions.map((o) => [o.value, o.label]));
    const technologyKeys = Object.keys(deviceStatusByTypeData);
    const xData = technologyKeys.map((key) => techLabelMap.get(key as TechnologyType) ?? key.toUpperCase());

    const onlineData = technologyKeys.map(key => deviceStatusByTypeData[key]?.online ?? 0);
    const offlineData = technologyKeys.map(key => deviceStatusByTypeData[key]?.offline ?? 0);
    const alarmData = technologyKeys.map(key => deviceStatusByTypeData[key]?.alarm ?? 0);

    const series = [
      { name: t('dashboard.chart.online'), data: onlineData, color: '#52C41A' },
      { name: t('dashboard.chart.offline'), data: offlineData, color: '#8C8C8C' },
      { name: t('dashboard.chart.alarmDevices'), data: alarmData, color: '#FA8C16' },
    ];

    return { isEmpty: false, xData, series };
  }, [deviceStatusByTypeData, techOptions, t]);

  // Alarm distribution bar chart data - 按告警等级分组统计
  const alarmDistributionData = useMemo(() => {
    const alarmCounts = alarmStoreCounts;

    if (!alarmCounts) {
      return { isEmpty: true, xData: [], series: [] };
    }

    const knownTotal = (alarmCounts.critical ?? 0) + (alarmCounts.major ?? 0) +
      (alarmCounts.minor ?? 0) + (alarmCounts.warning ?? 0);
    // 从 global store 里可能没有 total，所以只用 knownTotal
    const overallTotal = alarmCounts.total_active ?? knownTotal;
    const otherCount = Math.max(overallTotal - knownTotal, 0);
    const hasAlarms = overallTotal > 0;

    if (!hasAlarms) {
      return { isEmpty: true, xData: [], series: [], otherCount: 0 };
    }

    const xData = [
      t('alarm.severity.critical'),
      t('alarm.severity.major'),
      t('alarm.severity.minor'),
      t('alarm.severity.warning'),
    ];

    const data = [
      { value: alarmCounts.critical, name: t('alarm.severity.critical'), itemStyle: { color: '#F5222D' } },
      { value: alarmCounts.major, name: t('alarm.severity.major'), itemStyle: { color: '#FA8C16' } },
      { value: alarmCounts.minor, name: t('alarm.severity.minor'), itemStyle: { color: '#FADB14' } },
      { value: alarmCounts.warning, name: t('alarm.severity.warning'), itemStyle: { color: '#1677FF' } },
    ];

    if (otherCount > 0) {
      xData.push(t('dashboard.alarmSeverityOther'));
      data.push({ value: otherCount, name: t('dashboard.alarmSeverityOther'), itemStyle: { color: '#8C8C8C' } });
    }

    const series = [
      {
        name: t('dashboard.alarmCountEvents'),
        data: data as Array<{ value: number; name: string; itemStyle: { color: string } }>,
      },
    ];

    return { isEmpty: false, xData, series, otherCount };
  }, [alarmStoreCounts, t]);

  const dashboardRef = useRef<HTMLDivElement>(null);
  useScrollReveal(dashboardRef);

  const handleAlarmChartClick = useCallback(
    (_index: number, _name: string) => {
      openTab({ key: '/alarm/current', label: 'nav.alarm.current', path: '/alarm/current', closable: true, labelRaw: false });
      void navigate('/alarm/current');
    },
    [navigate, openTab]
  );

  return (
    <div ref={dashboardRef} style={{ padding: '0 0 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Dashboard Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('nav.dashboard')}
        </Title>
        <Space size="middle">
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            {t('dashboard.lastUpdate')}:{' '}
            {lastUpdateTime
              ? timeAgoText || formatTimeAgo(lastUpdateTime, t)
              : '--'}
          </Typography.Text>
          {/* 刷新按钮 */}
          {DASHBOARD_CONFIG.showRefreshControls && (
          <Button
            icon={<ReloadOutlined />}
            loading={isRefreshing || isFetching}
            onClick={handleManualRefresh}
            size="small"
          >
            {t('dashboard.refresh')}
          </Button>
          )}
        </Space>
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
            loading={deviceCardsLoading}
            hasComparison={totalDevicesDelta?.hasComparison === true}
            unavailableText={t('dashboard.notComparable')}
            trend={totalDevicesDelta?.trend ?? 'stable'}
            delta={totalDevicesDelta?.changePercent !== undefined ? `${totalDevicesDelta.changePercent.toFixed(1)}%` : undefined}
            deltaLabel={totalDevicesDelta?.compareType === 'last_week' ? t('dashboard.vsLastWeek') : t('dashboard.vsYesterday')}
            onClick={() => {
              openTab({ key: '/device/list', label: 'nav.device.list', path: '/device/list', closable: true, labelRaw: false });
              void navigate('/device/list');
            }}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.onlineDevices')}
            value={onlineDevices}
            icon={<WifiOutlined />}
            iconBgColor="#f6ffed"
            iconColor="#52C41A"
            loading={deviceCardsLoading}
            trend="up"
            delta={totalDevices > 0
              ? `${Math.round((onlineDevices / totalDevices) * 100)}%`
              : '--'}
            deltaLabel={t('dashboard.onlineRate')}
            onClick={() => {
              openTab({ key: '/device/list', label: 'nav.device.list', path: '/device/list', closable: true, labelRaw: false });
              void navigate('/device/list');
            }}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.activeAlarmsEvents')}
            value={activeAlarms}
            icon={<AlertOutlined />}
            iconBgColor="#fff2f0"
            iconColor="#F5222D"
            loading={summaryCardsLoading}
            hasComparison={activeAlarmsDelta?.hasComparison === true}
            unavailableText={t('dashboard.notComparable')}
            trend={activeAlarmsDelta?.trend ?? 'stable'}
            delta={activeAlarmsDelta?.changePercent !== undefined ? `${activeAlarmsDelta.changePercent.toFixed(1)}%` : undefined}
            deltaLabel={activeAlarmsDelta?.compareType === 'yesterday' ? t('dashboard.vsYesterday') : t('dashboard.vsLastWeek')}
            onClick={() => {
              openTab({ key: '/alarm/current', label: 'nav.alarm.current', path: '/alarm/current', closable: true, labelRaw: false });
              void navigate('/alarm/current');
            }}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KPICard
            title={t('dashboard.currentUE')}
            value={isDeviceStatsError ? '--' : currentUE}
            icon={<TeamOutlined />}
            iconBgColor="#f6ffed"
            iconColor="#10B981"
            loading={currentUECardLoading}
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
            loading={isPending}
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
          <Space size="middle" style={{ width: '100%', alignItems: 'center' }}>
            <Text type="secondary">{t('dashboard.networkTech')}:</Text>
            <Segmented
              value={effectiveTechnology}
              onChange={(value) => setTechnology(value as TechnologyType)}
              options={techOptions}
            />
			{latestPMSlotHealth && (
			  <Space size="small">
				<Text type="secondary">{t('dashboard.pmSlotCoverage')}</Text>
				<Badge
				  status={pmSlotBadgeStatus}
				  text={`${latestPMSlotHealth.receivedDevices} / ${latestPMSlotHealth.expectedDevices} · ${(latestPMSlotHealth.coverageRatio * 100).toFixed(1)}%`}
				/>
			  </Space>
			)}
          </Space>
        </Col>
      </Row>

      {/* KPI Panel区域 - v2.0 Panel化设计 */}
      <DashboardKPIModules technology={effectiveTechnology} />

      {/* Row 3: Device Status + Alarm Statistics */}
      <Row gutter={[16, 16]} align="stretch" className="omc-scroll-reveal" data-delay="2">
        <Col xs={24} lg={12} style={{ display: 'flex' }}>
          <TiltCard maxTilt={7} style={{ width: '100%', display: 'flex', flexDirection: 'column' }}>
          <Card
            title={t('dashboard.deviceStatusByType')}
            size="small"
            extra={<Text type="secondary" style={{ fontSize: 12 }}>{t('dashboard.deviceStatusAlarmHint')}</Text>}
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
                minBarHeight={1}
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
              <Space size={8}>
                {(alarmDistributionData.otherCount ?? 0) > 0 && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('dashboard.alarmOtherHint', { count: alarmDistributionData.otherCount ?? 0 })}
                  </Text>
                )}
                <a onClick={() => {
                  openTab({ key: '/alarm/current', label: 'nav.alarm.current', path: '/alarm/current', closable: true, labelRaw: false });
                  void navigate('/alarm/current');
                }} style={{ fontSize: 13 }}>
                  {t('dashboard.viewAll')}
                </a>
              </Space>
            }
            styles={{ body: { padding: '8px 0 0', flex: 1, display: 'flex', flexDirection: 'column' } }}
            style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          >
            {isPending ? (
              <Spin
                spinning={isPending}
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
                minBarHeight={1}
                onClick={handleAlarmChartClick}
              />
            )}
          </Card>
          </TiltCard>
        </Col>
      </Row>

      {/* Row 4: User Profile + Quick Access */}
      {DASHBOARD_CONFIG.showUserProfileAndQuickAccess && (
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
                  {t('dashboard.lastLogin')} {formatLastLogin(currentUser?.lastLoginTime, locale)}
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
              {quickAccessItems.length === 0 ? (
                <EmptyState
                  variant="no-permission"
                  style={{ padding: '24px 16px', width: '100%' }}
                />
              ) : quickAccessItems.map((item) => (
                <div
                  key={item.path}
                  onClick={() => {
                    openTab({
                      key: item.path,
                      label: item.labelKey,
                      path: item.path,
                      closable: true,
                      labelRaw: false,
                    });
                    void navigate(item.path);
                  }}
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
      )}
    </div>
  );
}
