import { Card, Col, Row, Statistic, Typography, Space, Tooltip, Skeleton, Spin } from 'antd';
import {
  ClockCircleOutlined,
  CheckCircleOutlined,
  InfoCircleOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  MinusOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { dashboardApi } from '@core/services/api/dashboardApi';
import { useT } from '@/hooks/useT';
import { useCallback, useMemo } from 'react';
import type { EfficiencyMetrics } from '@core/types/dashboard';
import EmptyState from '@/components/common/EmptyState';

const { Text } = Typography;

// 趋势指示器组件
function TrendIndicator({ value, prefix = '' }: { value: number; prefix?: string }) {
  const valueStr = Math.abs(value).toFixed(1);

  if (value > 0) {
    return (
      <Space size={4}>
        <Text type="danger" style={{ fontSize: 12 }}>
          <ArrowUpOutlined /> {valueStr}%
        </Text>
      </Space>
    );
  }

  if (value < 0) {
    return (
      <Space size={4}>
        <Text type="success" style={{ fontSize: 12 }}>
          <ArrowDownOutlined /> {valueStr}%
        </Text>
      </Space>
    );
  }

  return <Text type="secondary" style={{ fontSize: 12 }}><MinusOutlined /> 0%</Text>;
}

// 迷你折线图组件（简化版，实际可以使用 ECharts）
function MiniTrendChart({ data }: { data: number[] }) {
  const max = Math.max(...data, 1);
  const min = Math.min(...data, 0);
  const range = max - min || 1;

  const points = useMemo(() => {
    return data.map((value, index) => {
      const x = (index / (data.length - 1)) * 100;
      const y = 100 - ((value - min) / range) * 100;
      return `${x},${y}`;
    }).join(' ');
  }, [data, max, min, range]);

  const fillArea = useMemo(() => {
    return `0,100 ${points} 100,100`;
  }, [points]);

  return (
    <svg
      width="100%"
      height="40"
      viewBox="0 0 100 40"
      preserveAspectRatio="none"
      style={{ display: 'block' }}
    >
      <polygon
        points={fillArea}
        fill="rgba(24, 144, 255, 0.1)"
        stroke="none"
      />
      <polyline
        points={points}
        fill="none"
        stroke="#1677FF"
        strokeWidth="2"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

export default function EfficiencyCard() {
  const t = useT();

  const { data: metrics, isLoading, isError } = useQuery({
    queryKey: ['dashboard', 'alarm-efficiency'],
    queryFn: () => dashboardApi.getAlarmEfficiency(),
    refetchInterval: 300000, // 5分钟刷新
    staleTime: 60000,
  });

  // 从趋势数据中提取 MTTR 趋势
  const mttrTrend = useMemo(() => {
    if (!metrics?.daily_trend) return [];
    return metrics.daily_trend.slice().reverse().map(d => d.avg_resolve_minutes ?? 0);
  }, [metrics]);

  // 计算趋势变化（最近1天 vs 前6天平均值）
  const mttrTrendChange = useMemo(() => {
    if (!metrics?.daily_trend || metrics.daily_trend.length < 2) return 0;

    // 当数据不足7天时，使用现有数据计算趋势
    const trendData = metrics.daily_trend.slice(0, Math.min(7, metrics.daily_trend.length));
    if (trendData.length < 2) return 0;

    // 最新一天 vs 前几天平均
    const latest = trendData[0];
    const previous = trendData.slice(1);
    const previousAvg = previous.reduce((sum, d) => sum + (d.avg_resolve_minutes ?? 0), 0) / previous.length;

    if (previousAvg === 0) return 0;
    return (((latest.avg_resolve_minutes ?? 0) - previousAvg) / previousAvg) * 100;
  }, [metrics]);

  if (isLoading) {
    return (
      <Card
        title={
          <Space>
            <span>{t('alarm.stats.efficiency')}</span>
            <Tooltip title={t('alarm.stats.efficiencyTooltip')}>
              <InfoCircleOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
            </Tooltip>
          </Space>
        }
        size="small"
        styles={{ body: { padding: '16px' } }}
      >
        <div style={{ minHeight: 140, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Spin size="large" />
        </div>
      </Card>
    );
  }

  if (isError) {
    return (
      <Card
        title={
          <Space>
            <span>{t('alarm.stats.efficiency')}</span>
            <Tooltip title={t('alarm.stats.efficiencyTooltip')}>
              <InfoCircleOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
            </Tooltip>
          </Space>
        }
        size="small"
        styles={{ body: { padding: '16px' } }}
      >
        <EmptyState variant="error" style={{ padding: '32px 0' }} />
      </Card>
    );
  }

  // 当没有数据时显示空状态
  if (!metrics) {
    return (
      <Card
        title={
          <Space>
            <span>{t('alarm.stats.efficiency')}</span>
            <Tooltip title={t('alarm.stats.efficiencyTooltip')}>
              <InfoCircleOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
            </Tooltip>
          </Space>
        }
        size="small"
        styles={{ body: { padding: '16px' } }}
      >
        <EmptyState variant="no-data" style={{ padding: '32px 0' }} />
      </Card>
    );
  }

  return (
    <Card
      title={
        <Space>
          <span>{t('alarm.stats.efficiency')}</span>
          <Tooltip title={t('alarm.stats.efficiencyTooltip')}>
            <InfoCircleOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
          </Tooltip>
        </Space>
      }
      size="small"
      styles={{ body: { padding: '16px' } }}
    >
      <Row gutter={[24, 16]}>
        {/* MTTA - 平均确认时间 */}
        <Col xs={12} sm={8}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 13, color: '#8c8c8c', marginBottom: 8 }}>
              <Tooltip title={t('alarm.stats.mttaTooltip')}>
                <Space size={4}>
                  {t('alarm.stats.mtta')}
                  <InfoCircleOutlined style={{ fontSize: 11 }} />
                </Space>
              </Tooltip>
            </div>
            <div
              style={{
                fontSize: 28,
                fontWeight: 600,
                color: (metrics.avg_acknowledge_minutes ?? 0) > 30 ? '#cf1322' : '#3f8600',
                fontFamily: 'SF Mono, Monaco, Consolas, monospace',
              }}
            >
              {(metrics.avg_acknowledge_minutes ?? 0).toFixed(1)}
            </div>
            <div style={{ fontSize: 12, color: '#8c8c8c', marginTop: 4 }}>
              {t('common.minute')}
            </div>
          </div>
        </Col>

        {/* MTTR - 平均解决时间 */}
        <Col xs={12} sm={8}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 13, color: '#8c8c8c', marginBottom: 8 }}>
              <Tooltip title={t('alarm.stats.mttrTooltip')}>
                <Space size={4}>
                  {t('alarm.stats.mttr')}
                  <InfoCircleOutlined style={{ fontSize: 11 }} />
                </Space>
              </Tooltip>
            </div>
            <div
              style={{
                fontSize: 28,
                fontWeight: 600,
                color: '#1677FF',
                fontFamily: 'SF Mono, Monaco, Consolas, monospace',
              }}
            >
              {(metrics.avg_resolve_minutes ?? 0).toFixed(1)}
            </div>
            <div style={{ fontSize: 12, color: '#8c8c8c', marginTop: 4 }}>
              {t('common.minute')}
            </div>
          </div>
        </Col>

        {/* 确认率和清除率 */}
        <Col xs={24} sm={8}>
          <Row gutter={[8, 8]} justify="center">
            <Col span={12}>
              <div style={{ textAlign: 'center', padding: '8px 0', background: '#f5f5f5', borderRadius: 6 }}>
                <div style={{ fontSize: 12, color: '#8c8c8c' }}>{t('alarm.stats.ackRate')}</div>
                <div style={{ fontSize: 18, fontWeight: 600, color: '#52c41a', marginTop: 4 }}>
                  {(metrics.acknowledge_rate ?? 0).toFixed(1)}%
                </div>
              </div>
            </Col>
            <Col span={12}>
              <div style={{ textAlign: 'center', padding: '8px 0', background: '#f5f5f5', borderRadius: 6 }}>
                <div style={{ fontSize: 12, color: '#8c8c8c' }}>{t('alarm.stats.clearRate')}</div>
                <div style={{ fontSize: 18, fontWeight: 600, color: '#52c41a', marginTop: 4 }}>
                  {(metrics.clear_rate ?? 0).toFixed(1)}%
                </div>
              </div>
            </Col>
          </Row>
        </Col>
      </Row>

      {/* 趋势图和总数统计 */}
      <Row gutter={16} style={{ marginTop: 8, paddingTop: 16, borderTop: '1px solid #f5f5f5' }}>
        {/* 7天趋势 */}
        <Col xs={24} md={12}>
          <div style={{ marginBottom: 8 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('alarm.stats.7DayTrend')}
            </Text>
            {mttrTrendChange !== 0 && <TrendIndicator value={mttrTrendChange} />}
          </div>
          <div style={{ height: 48 }}>
            {mttrTrend.length > 0 ? (
              <MiniTrendChart data={mttrTrend} />
            ) : (
              <div style={{ textAlign: 'center', paddingTop: 12, color: '#bfbfbf', fontSize: 12 }}>
                {t('common.noData')}
              </div>
            )}
          </div>
        </Col>

        {/* 总数统计 */}
        <Col xs={24} md={12}>
          <Row gutter={[8, 8]} justify="end">
            <Col>
              <Space size={16}>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('alarm.stats.totalAcknowledged')}: <Text strong>{metrics.acknowledged_count.toLocaleString()}</Text>
                </Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('alarm.stats.totalCleared')}: <Text strong>{metrics.cleared_count.toLocaleString()}</Text>
                </Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  总计: <Text strong>{metrics.total_count.toLocaleString()}</Text>
                </Text>
              </Space>
            </Col>
          </Row>
        </Col>
      </Row>
    </Card>
  );
}
