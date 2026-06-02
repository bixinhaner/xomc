import { useMemo, useState, useCallback, useEffect } from 'react';
import { Card, Col, Row, Typography, DatePicker, Radio, Space, Button, Switch, Tag, Tooltip, Spin } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { ReloadOutlined, SyncOutlined, ClockCircleOutlined } from '@ant-design/icons';
import PieChart, { type PieDataItem } from '@/components/Charts/PieChart';
import BarChart from '@/components/Charts/BarChart';
import EmptyState from '@/components/common/EmptyState';
import ReactECharts from 'echarts-for-react';
import { useAlarmCount } from '@core/hooks/api/useAlarms';
import { useAlarmTrend, useTopAlarmDevices } from '@core/hooks/api/useDashboard';
import { useT } from '@/hooks/useT';
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import EfficiencyCard from './EfficiencyCard';
import AlarmHeatmap from './AlarmHeatmap';
import { createAlarmTrendChartOption, createAlarmPieChartOption } from './chartConfig';

// 扩展的饼图数据项，包含严重度信息
interface AlarmPieDataItem extends PieDataItem {
  severity?: string;
}

const { Title } = Typography;

// 告警级别颜色 - 电信行业标准配色（符合3GPP TMF642规范）
/* eslint-disable react-refresh/only-export-components */
export const SEVERITY_COLORS = {
  critical: '#F5222D', // 紧急 - 标准红色
  major: '#FA8C16',    // 重要 - 橙色
  minor: '#FADB14',    // 次要 - 黄色
  warning: '#1677FF',  // 警告 - 蓝色
};

// 告警级别颜色数组（用于图表系列）
export const SEVERITY_COLOR_ARRAY = [
  SEVERITY_COLORS.critical,
  SEVERITY_COLORS.major,
  SEVERITY_COLORS.minor,
  SEVERITY_COLORS.warning,
];
/* eslint-enable react-refresh/only-export-components */

// 时间范围选项
type TimeRange = '7days' | '30days' | 'custom';

interface StatisticsFilters {
  timeRange: TimeRange;
  customStartDate?: Dayjs | null;
  customEndDate?: Dayjs | null;
}

// 生成时间标签
function generateDayLabels(count: number): string[] {
  const days: string[] = [];
  for (let i = count - 1; i >= 0; i--) {
    const d = dayjs().subtract(i, 'day');
    days.push(d.format('MM/DD'));
  }
  return days;
}

// 告警级别分布图组件
function AlarmDistributionChart({
  alarmCount,
  t,
  onDrillDown,
}: {
  alarmCount: ReturnType<typeof useAlarmCount>['data'];
  t: (key: string) => string;
  onDrillDown?: (severity: string) => void;
}) {
  const totalCount = useMemo(() => {
    if (!alarmCount) return 0;
    return (alarmCount.critical || 0) + (alarmCount.major || 0) + (alarmCount.minor || 0) + (alarmCount.warning || 0);
  }, [alarmCount]);

  const data = useMemo((): AlarmPieDataItem[] => {
    if (!alarmCount || totalCount === 0) return [];
    return [
      { name: t('alarm.severity.critical'), value: alarmCount.critical || 0, color: SEVERITY_COLORS.critical, severity: 'critical' },
      { name: t('alarm.severity.major'), value: alarmCount.major || 0, color: SEVERITY_COLORS.major, severity: 'major' },
      { name: t('alarm.severity.minor'), value: alarmCount.minor || 0, color: SEVERITY_COLORS.minor, severity: 'minor' },
      { name: t('alarm.severity.warning'), value: alarmCount.warning || 0, color: SEVERITY_COLORS.warning, severity: 'warning' },
    ];
  }, [alarmCount, totalCount, t]);

  const handlePieClick = useCallback((item: AlarmPieDataItem) => {
    if (onDrillDown && item.severity) {
      onDrillDown(item.severity);
    }
  }, [onDrillDown]);

  if (totalCount === 0) {
    return (
      <Card
        title={<span style={{ fontSize: 14, fontWeight: 500 }}>{t('alarm.stats.distribution')}</span>}
        size="small"
        styles={{ body: { padding: '16px', height: '280px', display: 'flex', alignItems: 'center', justifyContent: 'center' } }}
      >
        <EmptyState variant="no-data" style={{ padding: '20px 0' }} />
      </Card>
    );
  }

  return (
    <Card
      title={<span style={{ fontSize: 14, fontWeight: 500 }}>{t('alarm.stats.distribution')}</span>}
      size="small"
      styles={{ body: { padding: '16px', height: '280px', display: 'flex', flexDirection: 'column' } }}
    >
      <PieChart
        data={data}
        height="100%"
        donut
        showLegend
        centerText={totalCount.toString()}
        onClick={handlePieClick}
      />
    </Card>
  );
}

// 告警趋势图组件
function AlarmTrendChart({
  trendData,
  days,
  t,
}: {
  trendData: ReturnType<typeof useAlarmTrend>['data'];
  days: number;
  t: (key: string) => string;
}) {
  const xData = useMemo(() => generateDayLabels(days), [days]);

  const hasData = useMemo(() => {
    return trendData && trendData.length > 0 && trendData.some(d =>
      (d.critical || 0) + (d.major || 0) + (d.minor || 0) + (d.warning || 0) > 0
    );
  }, [trendData]);

  const series = useMemo(() => {
    if (!trendData || trendData.length === 0) return [];

    return [
      {
        name: t('alarm.severity.critical'),
        data: trendData.map((d) => d.critical || 0),
        color: SEVERITY_COLORS.critical,
      },
      {
        name: t('alarm.severity.major'),
        data: trendData.map((d) => d.major || 0),
        color: SEVERITY_COLORS.major,
      },
      {
        name: t('alarm.severity.minor'),
        data: trendData.map((d) => d.minor || 0),
        color: SEVERITY_COLORS.minor,
      },
      {
        name: t('alarm.severity.warning'),
        data: trendData.map((d) => d.warning || 0),
        color: SEVERITY_COLORS.warning,
      },
    ];
  }, [trendData, t]);

  if (!hasData) {
    return (
      <Card
        title={<span style={{ fontSize: 14, fontWeight: 500 }}>{t('alarm.stats.trend')}</span>}
        size="small"
        styles={{ body: { padding: '16px', height: '280px', display: 'flex', alignItems: 'center', justifyContent: 'center' } }}
      >
        <EmptyState variant="no-data" style={{ padding: '20px 0' }} />
      </Card>
    );
  }

  return (
    <Card
      title={<span style={{ fontSize: 14, fontWeight: 500 }}>{t('alarm.stats.trend')}</span>}
      size="small"
      styles={{ body: { padding: '12px', height: '280px' } }}
    >
      <ReactECharts
        option={createAlarmTrendChartOption(xData, series, days, t)}
        style={{ height: 230, width: '100%' }}
        opts={{ renderer: 'canvas' }}
        notMerge={true}
      />
    </Card>
  );
}

// 高频告警设备排行图组件
function TopAlarmDevicesChart({
  devicesData,
  t,
  onDrillDown,
}: {
  devicesData: ReturnType<typeof useTopAlarmDevices>['data'];
  t: (key: string) => string;
  onDrillDown?: (deviceSN: string) => void;
}) {
  const topDevices = useMemo(() => {
    if (!devicesData || devicesData.length === 0) return [];
    // 取Top 10设备，按告警数量降序排序
    return [...devicesData]
      .sort((a, b) => (b.alarmCount || 0) - (a.alarmCount || 0))
      .slice(0, 10);
  }, [devicesData]);

  const deviceLabels = useMemo(() => {
    return topDevices.map((d) => {
      // 格式化设备标识：技术类型-设备SN后4位
      const snSuffix = d.deviceSN?.slice(-4) || '????';
      const techPrefix = d.technology === 'lte' ? 'LTE'
        : d.technology === 'nr' ? '5G'
        : d.technology === 'gsm' ? 'GSM'
        : d.technology?.toUpperCase() || 'UNK';
      return `${techPrefix}-${snSuffix}`;
    });
  }, [topDevices]);

  const series = useMemo(() => {
    return [{
      name: t('alarm.stats.alarmCount'),
      data: topDevices.map((d) => d.alarmCount || 0),
      color: SEVERITY_COLORS.major,
    }];
  }, [topDevices, t]);

  const handleBarClick = useCallback((index: number) => {
    if (onDrillDown && topDevices[index]?.deviceSN) {
      onDrillDown(topDevices[index].deviceSN!);
    }
  }, [onDrillDown, topDevices]);

  if (!topDevices || topDevices.length === 0) {
    return (
      <Card
        title={<span style={{ fontSize: 14, fontWeight: 500 }}>{t('alarm.stats.topDevices')}</span>}
        size="small"
        styles={{ body: { padding: '16px', height: '280px', display: 'flex', alignItems: 'center', justifyContent: 'center' } }}
      >
        <EmptyState variant="no-data" style={{ padding: '20px 0' }} />
      </Card>
    );
  }

  return (
    <Card
      title={<span style={{ fontSize: 14, fontWeight: 500 }}>{t('alarm.stats.topDevices')}</span>}
      size="small"
      styles={{ body: { padding: '16px', height: '280px', display: 'flex', flexDirection: 'column' } }}
    >
      <BarChart
        title=""
        xData={deviceLabels}
        series={series}
        height="100%"
        horizontal
        barWidth={16}
        borderRadius={4}
        onClick={handleBarClick}
      />
    </Card>
  );
}

// 主页面组件
export default function AlarmStatistics() {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  // 最后更新时间状态
  const [lastUpdateTime, setLastUpdateTime] = useState<Date>(new Date());
  const [timeAgoText, setTimeAgoText] = useState('');

  // 自动刷新状态
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval] = useState(60); // 固定60秒间隔
  const [nextRefreshIn, setNextRefreshIn] = useState(refreshInterval);

  // 手动刷新 loading 状态
  const [isRefreshing, setIsRefreshing] = useState(false);

  // 自动刷新倒计时
  useEffect(() => {
    let mounted = true;
    let timer: NodeJS.Timeout | undefined;
    let isInitialized = false;

    const tick = () => {
      if (!mounted) return;
      setNextRefreshIn((prev) => {
        // 首次执行时初始化
        if (!isInitialized) {
          isInitialized = true;
          return refreshInterval;
        }
        if (prev <= 1) return refreshInterval;
        return prev - 1;
      });
    };

    if (autoRefresh) {
      timer = setInterval(tick, 1000);
      // 立即执行一次以初始化
      tick();
    } else {
      // 停止时重置
      setTimeout(() => setNextRefreshIn(refreshInterval), 0);
    }

    return () => {
      mounted = false;
      if (timer) clearInterval(timer);
    };
  }, [autoRefresh, refreshInterval]);

  // 更新时间倒计时文案
  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      const diff = now.getTime() - lastUpdateTime.getTime();
      const seconds = Math.floor(diff / 1000);
      const minutes = Math.floor(seconds / 60);

      if (seconds < 60) {
        setTimeAgoText(t('alarm.stats.justNow'));
      } else if (minutes < 60) {
        setTimeAgoText(`${minutes}${t('common.minutes')}${t('alarm.stats.before')}`);
      } else {
        const hours = Math.floor(minutes / 60);
        setTimeAgoText(`${hours}${t('common.hours')}${t('alarm.stats.before')}`);
      }
    };

    updateTime();
    const interval = setInterval(updateTime, 10000); // 每10秒更新一次
    return () => clearInterval(interval);
  }, [lastUpdateTime, t]);

  // 自动刷新定时器
  useEffect(() => {
    if (!autoRefresh) return;

    const timer = setInterval(() => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      setLastUpdateTime(new Date());
    }, refreshInterval * 1000);

    return () => clearInterval(timer);
  }, [autoRefresh, refreshInterval, queryClient]);

  // 过滤器状态
  const [filters, setFilters] = useState<StatisticsFilters>({
    timeRange: '7days',
  });

  // 计算天数
  const days = useMemo(() => {
    if (filters.timeRange === '7days') return 7;
    if (filters.timeRange === '30days') return 30;
    if (filters.timeRange === 'custom' && filters.customStartDate && filters.customEndDate) {
      return Math.max(1, filters.customEndDate.diff(filters.customStartDate, 'day') + 1);
    }
    return 7;
  }, [filters]);

  // 数据hooks
  const { data: alarmCount, refetch: refetchCount } = useAlarmCount();
  const { data: trendData, refetch: refetchTrend } = useAlarmTrend(days);
  const { data: devicesData, refetch: refetchDevices } = useTopAlarmDevices();

  // 手动刷新
  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    try {
      // 并行刷新所有查询
      await Promise.all([
        refetchCount(),
        refetchTrend(),
        refetchDevices(),
      ]);
      setLastUpdateTime(new Date());
    } finally {
      // 短暂延迟确保 loading 效果可见
      setTimeout(() => setIsRefreshing(false), 500);
    }
  }, [refetchCount, refetchTrend, refetchDevices]);

  // 时间范围变化
  const handleTimeRangeChange = (value: TimeRange) => {
    setFilters((prev) => ({ ...prev, timeRange: value }));
  };

  // 自定义日期范围变化
  const handleCustomDateChange = (dates: [Dayjs | null, Dayjs | null] | null) => {
    if (dates && dates[0] && dates[1]) {
      setFilters((prev) => ({
        ...prev,
        customStartDate: dates[0],
        customEndDate: dates[1],
      }));
    }
  };

  // 钻取到告警详情
  const handleDrillDown = useCallback((severity?: string, deviceSN?: string) => {
    const params = new URLSearchParams();
    if (severity) params.set('severity', severity);
    if (deviceSN) params.set('deviceSN', deviceSN);
    navigate(`/alarm/current?${params.toString()}`);
  }, [navigate]);


  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', padding: '16px' }}>
      {/* 页面头部 - 优化布局 */}
      <div style={{ marginBottom: 24 }}>
        {/* 第一行：标题和操作 */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}>
          <div>
            <Title level={4} style={{ margin: 0, marginBottom: 8 }}>
              {t('nav.alarm.statistics')}
            </Title>
            <Space size={4} wrap>
              <Tag icon={<ClockCircleOutlined />} color="default" style={{ fontSize: 12, margin: 0 }}>
                {t('alarm.stats.lastUpdate')}: {timeAgoText || t('alarm.stats.justNow')}
              </Tag>
              {autoRefresh && (
                <Tag color="processing" style={{ fontSize: 12, margin: 0 }}>
                  {t('alarm.stats.autoRefresh')}: {nextRefreshIn}s
                </Tag>
              )}
            </Space>
          </div>

          {/* 操作区 */}
          <Space size="small">
            <Tooltip title={autoRefresh ? `${t('alarm.stats.nextRefresh')}: ${nextRefreshIn}s` : t('alarm.stats.enableAutoRefresh')}>
              <Switch
                checked={autoRefresh}
                onChange={setAutoRefresh}
                checkedChildren={<SyncOutlined spin />}
                unCheckedChildren={<ClockCircleOutlined />}
                size="small"
              />
            </Tooltip>
            {!autoRefresh && (
              <Button
                icon={<ReloadOutlined />}
                onClick={handleRefresh}
                size="small"
              >
                {t('common.refresh')}
              </Button>
            )}
          </Space>
        </div>

        {/* 过滤器区域 - 简化设计，移除Card容器 */}
        <Space size="middle" wrap>
          <span style={{ color: '#8c8c8c', fontSize: 14 }}>{t('alarm.stats.timeRange')}:</span>
          <Radio.Group
            value={filters.timeRange}
            onChange={(e) => handleTimeRangeChange(e.target.value)}
            optionType="button"
            size="small"
          >
            <Radio.Button value="7days">{t('alarm.stats.last7Days')}</Radio.Button>
            <Radio.Button value="30days">{t('alarm.stats.last30Days')}</Radio.Button>
            <Radio.Button value="custom">{t('alarm.stats.custom')}</Radio.Button>
          </Radio.Group>

          {filters.timeRange === 'custom' && (
            <DatePicker.RangePicker
              value={[filters.customStartDate || null, filters.customEndDate || null]}
              onChange={handleCustomDateChange}
              allowClear={false}
              size="small"
            />
          )}
        </Space>
      </div>

      {/* 图表区域 */}
      <Spin spinning={isRefreshing} tip={t('common.loading')} size="large">
        <Row gutter={[16, 16]}>
          {/* 第零行：告警效率指标 */}
          <Col xs={24}>
            <EfficiencyCard />
          </Col>

          {/* 第一行：告警级别分布 + 告警热度图 */}
          <Col xs={24} lg={12}>
            <AlarmDistributionChart
              alarmCount={alarmCount}
              t={t}
              onDrillDown={(severity) => handleDrillDown(severity)}
            />
          </Col>
          <Col xs={24} lg={12}>
            <AlarmHeatmap />
          </Col>

          {/* 第二行：告警趋势 + 高频告警设备 */}
          <Col xs={24} lg={12}>
            <AlarmTrendChart trendData={trendData} days={days} t={t} />
          </Col>
          <Col xs={24} lg={12}>
            <TopAlarmDevicesChart
              devicesData={devicesData}
              t={t}
              onDrillDown={(deviceSN) => handleDrillDown(undefined, deviceSN)}
            />
          </Col>
        </Row>
      </Spin>
    </div>
  );
}
