import { Card, Radio, Space, Typography, Tooltip, Spin } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { dashboardApi } from '@core/services/api/dashboardApi';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useMemo, useState } from 'react';
import ReactECharts from 'echarts-for-react';
import type { HeatmapData } from '@core/types/dashboard';
import { InfoCircleOutlined } from '@ant-design/icons';
import EmptyState from '@/components/common/EmptyState';

const { Text } = Typography;

// 星期标签
const DAY_LABELS_ZH = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
const DAY_LABELS_EN = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const HOUR_LABELS = Array.from({ length: 24 }, (_, i) => `${i}:00`);

// 颜色主题 - 告警密度从低到高
const HEATMAP_COLORS = ['#e0f3f8', '#abd9e9', '#74add1', '#4575b4', '#313695'];

export default function AlarmHeatmap() {
  const t = useT();
  const token = useThemeToken();
  const [days, setDays] = useState(30);

  const { data: heatmapData, isLoading } = useQuery({
    queryKey: ['dashboard', 'alarm-heatmap', days],
    queryFn: () => dashboardApi.getAlarmHeatmap({ days }),
    refetchInterval: 300000, // 5分钟刷新
    staleTime: 60000,
  });

  const dayLabels = DAY_LABELS_ZH; // 使用中文标签

  const option = useMemo(() => {
    if (!heatmapData) return {};

    const data: [number, number, number][] = [];
    heatmapData.days_of_week.forEach((day) => {
      day.hours.forEach((count, hour) => {
        data.push([day.day, hour, count]);
      });
    });

    const maxCount = heatmapData.max_count || 1;

    return {
      tooltip: {
        position: 'top',
        backgroundColor: 'rgba(0, 0, 0, 0.85)',
        borderColor: '#333',
        textStyle: { color: '#fff', fontSize: 12 },
        formatter: (params: any) => {
          const [day, hour, count] = params.data;
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
        height: '65%',
        top: '10%',
        left: '6%',
        right: '12%',
        bottom: '18%',
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
        min: 0,
        max: maxCount,
        calculable: true,
        orient: 'horizontal',
        left: 'center',
        bottom: '4%',
        itemWidth: 12,
        itemHeight: 80,
        inRange: {
          color: HEATMAP_COLORS,
        },
        textStyle: {
          fontSize: 11,
          color: '#8c8c8c',
        },
        text: ['高', '低'],
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
  }, [heatmapData, t, dayLabels]);

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
          <Spin size="large" tip={t('common.loading')} />
        </div>
      ) : !heatmapData || heatmapData.max_count === 0 ? (
        <EmptyState variant="no-data" style={{ padding: '40px 0' }} />
      ) : (
        <ReactECharts option={option} style={{ height: '100%' }} />
      )}
    </Card>
  );
}
