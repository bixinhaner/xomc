import { Card, Radio, Space, Tooltip, Spin } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { dashboardApi } from '@core/services/api/dashboardApi';
import { useT } from '@/hooks/useT';
import { useMemo, useState } from 'react';
import ReactECharts from 'echarts-for-react';
import { InfoCircleOutlined } from '@ant-design/icons';
import EmptyState from '@/components/common/EmptyState';

// 星期标签
const DAY_LABELS_ZH = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
const HOUR_LABELS = Array.from({ length: 24 }, (_, i) => `${i}:00`);

// 颜色主题 - 告警密度从低到高
const HEATMAP_COLORS = ['#e0f3f8', '#abd9e9', '#74add1', '#4575b4', '#313695'];

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

  const { data: heatmapData, isLoading } = useQuery({
    queryKey: ['dashboard', 'alarm-heatmap', days],
    queryFn: () => dashboardApi.getAlarmHeatmap({ days }),
    refetchInterval: 300000, // 5分钟刷新
    staleTime: 60000,
  });

  const dayLabels = DAY_LABELS_ZH; // 使用中文标签
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
        formatter: (params: any) => {
          const [hour, day, count] = params.data;
          const dayLabel = dayLabels[day] || `Day ${day}`;
          return `
            <div style="padding: 8px; line-height: 1.6;">
              <div style="font-weight: 600; margin-bottom: 6px; font-size: 13px;">
                ${dayLabel} ${HOUR_LABELS[hour]}
              </div>
              <div style="color: '#bbb'; font-size: 12px;">
                告警数量: <strong style="color: '#fff'; font-size: 14px;">${count}</strong>
              </div>
            </div>
          `;
        },
      },
      grid: {
        height: '72%',
        top: '10%',
        left: '6%',
        right: '4%',
        bottom: '12%',
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
        show: false,
        min: 0,
        max: maxCount,
        inRange: {
          color: HEATMAP_COLORS,
        },
      },
      series: [
        {
          name: '告警数量',
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
  }, [normalizedHeatmap, dayLabels]);

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
        <Radio.Group
          value={days}
          onChange={(e) => setDays(e.target.value)}
          optionType="button"
          size="small"
        >
          <Radio.Button value={7}>7天</Radio.Button>
          <Radio.Button value={30}>30天</Radio.Button>
        </Radio.Group>
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
