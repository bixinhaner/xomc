import { Card, Radio, Select, Space, Tooltip, Spin } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { dashboardApi } from '@core/services/api/dashboardApi';
import { useT } from '@/hooks/useT';
import { useMemo, useState } from 'react';
import ReactECharts from 'echarts-for-react';
import type { CallbackDataParams } from 'echarts/types/dist/shared';
import { InfoCircleOutlined } from '@ant-design/icons';
import EmptyState from '@/components/common/EmptyState';
import {
  HEATMAP_COLORS_BY_SEVERITY,
  loadAlarmHeatmap,
  type HeatmapSeverity,
} from './alarmHeatmapModel';

const HOUR_LABELS = Array.from({ length: 24 }, (_, index) => `${index}:00`);

type HeatmapDay = {
  day: number;
  hours: number[];
};

type HeatmapPayload = {
  days_of_week?: HeatmapDay[];
  daysOfWeek?: HeatmapDay[];
  max_count?: number;
  maxCount?: number;
  data?: HeatmapPayload;
};

function normalizeHeatmap(payload?: HeatmapPayload) {
  if (!payload) return undefined;
  const source = payload.data || payload;
  const daysOfWeek = source.days_of_week || source.daysOfWeek || [];
  const totalCount = daysOfWeek.reduce(
    (sum, day) => sum + day.hours.reduce((hourSum, count) => hourSum + count, 0),
    0
  );
  return {
    daysOfWeek,
    maxCount: source.max_count ?? source.maxCount ?? 0,
    totalCount,
  };
}

export default function AlarmHeatmap() {
  const t = useT();
  const [days, setDays] = useState(30);
  const [severity, setSeverity] = useState<HeatmapSeverity>('all');

  const { data: heatmapData, isLoading } = useQuery({
    queryKey: ['dashboard', 'alarm-heatmap', days, severity],
    queryFn: () => loadAlarmHeatmap(dashboardApi, days, severity),
    refetchInterval: 300000, // 5分钟刷新
    staleTime: 60000,
  });

  const dayLabels = useMemo(() => [
    t('alarm.stats.heatmapMonday'),
    t('alarm.stats.heatmapTuesday'),
    t('alarm.stats.heatmapWednesday'),
    t('alarm.stats.heatmapThursday'),
    t('alarm.stats.heatmapFriday'),
    t('alarm.stats.heatmapSaturday'),
    t('alarm.stats.heatmapSunday'),
  ], [t]);
  const severityLabel = t(
    severity === 'all' ? 'alarm.statistics.all' : `alarm.severity.${severity}`,
  );
  const normalizedHeatmap = useMemo(() => normalizeHeatmap(heatmapData), [heatmapData]);

  const option = useMemo(() => {
    if (!normalizedHeatmap) return {};

    const data: [number, number, number][] = [];
    normalizedHeatmap.daysOfWeek.forEach((day) => {
      day.hours.forEach((count, hour) => {
        data.push([hour, day.day, count]);
      });
    });

    const maxCount = normalizedHeatmap.maxCount || 1;

    return {
      tooltip: {
        position: 'top',
        backgroundColor: 'rgba(0, 0, 0, 0.85)',
        borderColor: '#333',
        textStyle: { color: '#fff', fontSize: 12 },
        formatter: (params: CallbackDataParams) => {
          const [hour, day, count] = params.data as [number, number, number];
          const dayLabel = dayLabels[day] || t('alarm.stats.heatmapDayFallback', { day: day + 1 });
          return `
            <div style="padding: 8px; line-height: 1.6;">
              <div style="font-weight: 600; margin-bottom: 6px; font-size: 13px;">
                ${dayLabel} ${HOUR_LABELS[hour]}
              </div>
              <div style="color: #bbb; font-size: 12px;">
                ${t('alarm.severity.filter')}: <strong style="color: #fff;">${severityLabel}</strong>
              </div>
              <div style="color: #bbb; font-size: 12px;">
                ${t('alarm.stats.alarmCount')}: <strong style="color: #fff; font-size: 14px;">${count}</strong>
              </div>
            </div>
          `;
        },
      },
      grid: {
        height: '62%',
        top: '6%',
        left: '6%',
        right: '4%',
        bottom: '24%',
      },
      xAxis: {
        type: 'category',
        data: HOUR_LABELS,
        splitArea: {
          show: true,
        },
        axisLabel: {
          fontSize: 10,
          color: '#8c8c8c',
        },
        axisLine: { lineStyle: { color: '#e8e8e8' } },
      },
      yAxis: {
        type: 'category',
        data: dayLabels,
        splitArea: {
          show: true,
        },
        axisLabel: {
          fontSize: 11,
          color: '#595959',
        },
        axisLine: { lineStyle: { color: '#e8e8e8' } },
      },
      visualMap: {
        show: true,
        min: 0,
        max: maxCount,
        calculable: true,
        orient: 'horizontal',
        left: 'center',
        bottom: 0,
        itemWidth: 10,
        itemHeight: 100,
        precision: 0,
        text: [String(maxCount), '0'],
        textStyle: {
          color: '#8c8c8c',
          fontSize: 10,
        },
        inRange: {
          color: HEATMAP_COLORS_BY_SEVERITY[severity],
        },
      },
      series: [
        {
          name: t('alarm.stats.alarmCount'),
          type: 'heatmap',
          data: data,
          label: {
            show: false,
          },
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowColor: 'rgba(0, 0, 0, 0.3)',
              borderColor: '#333',
              borderWidth: 2,
            },
          },
          itemStyle: {
            borderWidth: 2,
            borderColor: '#fff',
            borderRadius: 2,
          },
        },
      ],
    };
  }, [normalizedHeatmap, dayLabels, severity, severityLabel, t]);

  return (
    <Card
      title={
        <Space size="small">
          <span style={{ fontSize: 14, fontWeight: 500 }}>{t('alarm.stats.heatmap')}</span>
          <Tooltip title={t('alarm.stats.heatmapTooltip')}>
            <InfoCircleOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
          </Tooltip>
        </Space>
      }
      size="small"
      styles={{ body: { padding: '12px 16px', height: '280px' } }}
      extra={
        <Space size={4}>
          <Select<HeatmapSeverity>
            value={severity}
            onChange={setSeverity}
            size="small"
            style={{ width: 88 }}
            options={[
              { value: 'all', label: t('alarm.statistics.all') },
              { value: 'critical', label: t('alarm.severity.critical') },
              { value: 'major', label: t('alarm.severity.major') },
              { value: 'minor', label: t('alarm.severity.minor') },
              { value: 'warning', label: t('alarm.severity.warning') },
            ]}
          />
          <Radio.Group
            value={days}
            onChange={(event) => setDays(event.target.value)}
            optionType="button"
            size="small"
          >
            <Radio.Button value={7}>{t('alarm.stats.heatmap7Days')}</Radio.Button>
            <Radio.Button value={30}>{t('alarm.stats.heatmap30Days')}</Radio.Button>
          </Radio.Group>
        </Space>
      }
    >
      {isLoading ? (
        <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Spin size="large" />
        </div>
      ) : !normalizedHeatmap || normalizedHeatmap.totalCount === 0 ? (
        <EmptyState variant="no-data" style={{ padding: '40px 0' }} />
      ) : (
        <ReactECharts option={option} style={{ height: '100%' }} />
      )}
    </Card>
  );
}
