/**
 * KPIPanel v2.0 - UI/UX优化版本
 *
 * 优化内容：
 * 1. 头部控制区分组布局
 * 2. 智能空状态提示
 * 3. 时间对比视觉反馈
 * 4. Panel图标化展示
 * 5. 快速操作功能
 */

import React, { useState, useMemo, useCallback } from 'react';
import { Card, Select, Button, Checkbox, Space, Spin, Empty, Typography, Dropdown, Segmented } from 'antd';
import {
  LoadingOutlined,
  FullscreenOutlined,
  DownloadOutlined,
  BarChartOutlined,
  WifiOutlined,
  CheckCircleOutlined,
  PhoneOutlined,
  SwapOutlined,
  DashboardOutlined,
  InfoCircleOutlined
} from '@ant-design/icons';
import LineChart from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import type {
  TechnologyType,
  PanelType,
  ViewMode,
  TimeRangeOption,
  KPIConfig,
  PanelConfig,
} from '@/pages/dashboard/kpi-config';
import { getPanelConfig } from '@/pages/dashboard/kpi-config';
import { useKPIPanelData } from './useKPIPanelData';

const { Text } = Typography;

// ============================================================================
// Panel Icon Mapping
// ============================================================================

const PANEL_ICONS: Record<PanelType, React.ReactNode> = {
  traffic: <BarChartOutlined />,
  availability: <CheckCircleOutlined />,
  utilization: <DashboardOutlined />,
  accessibility: <PhoneOutlined />,
  retainability: <WifiOutlined />,
  mobility: <SwapOutlined />,
};

const PANEL_COLORS: Record<PanelType, string> = {
  traffic: '#1677FF',
  availability: '#52C41A',
  utilization: '#FA8C16',
  accessibility: '#13C2C2',
  retainability: '#EB2F96',
  mobility: '#722ED1',
};

// ============================================================================
// Types
// ============================================================================

export interface KPIPanelProps {
  /** 当前制式 */
  technology: TechnologyType;
  /** Panel类型 */
  panelType: PanelType;
  /** 图表高度 */
  height?: number;
  /** 自定义className */
  className?: string;
  /** 是否显示快速操作按钮 */
  showQuickActions?: boolean;
  /** 全屏回调 */
  onFullscreen?: (panelType: PanelType) => void;
  /** 导出回调 */
  onExport?: (panelType: PanelType, indicator: string) => void;
}

// ============================================================================
// Helper Components
// ============================================================================

/**
 * 时间对比图例 - 显示当前选择的时间范围图例
 */
interface TimeRangeLegendProps {
  timeRangeSelection: TimeRangeOption[];
  selectedKPIConfig?: KPIConfig;
}

function TimeRangeLegend({ timeRangeSelection, selectedKPIConfig }: TimeRangeLegendProps) {
  const token = useThemeToken();
  const hasToday = timeRangeSelection.includes('today');
  const hasYesterday = timeRangeSelection.includes('yesterday');

  if (!hasToday && !hasYesterday) {
    return null;
  }

  return (
    <div style={{
      display: 'flex',
      justifyContent: 'center',
      gap: 24,
      padding: '8px 16px',
      borderBottom: `1px solid ${token.colorBorderSecondary}`,
      background: token.colorBgLayout,
    }}>
      {hasToday && (
        <Space size={6}>
          <span style={{
            width: 20,
            height: 3,
            background: selectedKPIConfig?.color || token.colorPrimary,
            borderRadius: 2,
          }} />
          <Text type="secondary" style={{ fontSize: 12 }}>今天</Text>
        </Space>
      )}
      {hasYesterday && (
        <Space size={6}>
          <span style={{
            width: 20,
            height: 3,
            background: token.colorBorder,
            borderStyle: 'dashed',
            borderWidth: 1,
            borderColor: token.colorBorder,
            borderRadius: 2,
          }} />
          <Text type="secondary" style={{ fontSize: 12 }}>昨天（对比）</Text>
        </Space>
      )}
    </div>
  );
}

/**
 * 智能空状态组件
 */
interface SmartEmptyStateProps {
  timeRangeSelection: TimeRangeOption[];
  viewMode: ViewMode;
  panelTitle: string;
  onSelectTimeRange: (selection: TimeRangeOption[]) => void;
}

function SmartEmptyState({
  timeRangeSelection,
  viewMode,
  panelTitle,
  onSelectTimeRange
}: SmartEmptyStateProps) {
  // 未选择时间范围
  if (timeRangeSelection.length === 0) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <Space direction="vertical" size={4} style={{ textAlign: 'center' }}>
            <Text>未选择时间范围</Text>
            <Text type="secondary" style={{ fontSize: 12 }}>
              请勾选"今天"或"昨天"以查看 {panelTitle} 数据
            </Text>
          </Space>
        }
      >
        <Button type="primary" size="small" onClick={() => onSelectTimeRange(['today'])}>
          查看今天数据
        </Button>
      </Empty>
    );
  }

  // 有选择但无数据
  return (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      description={
        <Space direction="vertical" size={4} style={{ textAlign: 'center' }}>
          <Text>该时间段暂无数据</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {viewMode === 'day' ? '今日' : '本周'}暂无 {panelTitle} 记录
          </Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            <InfoCircleOutlined /> 数据可能正在收集中
          </Text>
        </Space>
      }
    >
      <Space size={8}>
        <Button type="link" size="small" onClick={() => onSelectTimeRange(['yesterday'])}>
          切换到昨天
        </Button>
        <Button type="link" size="small" onClick={() => onSelectTimeRange(['today', 'yesterday'])}>
          启用对比
        </Button>
      </Space>
    </Empty>
  );
}

// ============================================================================
// Main Component
// ============================================================================

/**
 * KPIPanel v2.0 - UI/UX优化版本
 *
 * @example
 * ```tsx
 * <KPIPanel
 *   technology="lte"
 *   panelType="traffic"
 *   height={280}
 *   showQuickActions
 *   onFullscreen={(panelType) => console.log('Fullscreen:', panelType)}
 * />
 * ```
 */
export function KPIPanelV2({
  technology,
  panelType,
  height = 280,
  className,
  showQuickActions = false,
  onFullscreen,
  onExport,
}: KPIPanelProps) {
  const t = useT();
  const token = useThemeToken();

  // 获取Panel配置
  const panelConfig: PanelConfig | undefined = useMemo(
    () => getPanelConfig(technology, panelType),
    [technology, panelType]
  );

  // Panel内部状态
  const [selectedIndicator, setSelectedIndicator] = useState<string>('');
  const [viewMode, setViewMode] = useState<ViewMode>('day');
  const [timeRangeSelection, setTimeRangeSelection] = useState<TimeRangeOption[]>(['today']);

  // 当panelConfig变化时，重置所有Panel内部状态为默认值
  React.useEffect(() => {
    if (panelConfig) {
      /* eslint-disable react-hooks/set-state-in-effect -- Intentional state reset when panelConfig changes */
      setSelectedIndicator(panelConfig.defaultIndicator);
      setViewMode('day');
      setTimeRangeSelection(['today']);
      /* eslint-enable react-hooks/set-state-in-effect */
    }
  }, [panelConfig]);

  // 获取选中指标的配置（用于获取颜色和单位）
  const selectedKPIConfig: KPIConfig | undefined = useMemo(() => {
    return panelConfig?.indicators.find(ind => ind.key === selectedIndicator);
  }, [panelConfig, selectedIndicator]);

  // 获取KPI数据
  const { data, isLoading } = useKPIPanelData(
    selectedIndicator,
    timeRangeSelection,
    viewMode,
    Boolean(selectedIndicator)
  );

  // 处理指标切换
  const handleIndicatorChange = useCallback((value: string) => {
    setSelectedIndicator(value);
  }, []);

  // 处理视图模式切换
  const handleViewModeChange = useCallback((mode: ViewMode) => {
    setViewMode(mode);
  }, []);

  // 处理时间范围选择变化
  const handleTimeRangeChange = useCallback((values: TimeRangeOption[]) => {
    setTimeRangeSelection(values);
  }, []);

  // 快速操作
  const quickActionItems = [
    {
      key: 'fullscreen',
      icon: <FullscreenOutlined />,
      label: '全屏查看',
      onClick: () => onFullscreen?.(panelType),
    },
    {
      key: 'export',
      icon: <DownloadOutlined />,
      label: '导出数据',
      onClick: () => onExport?.(panelType, selectedIndicator),
    },
  ];

  // 转换图表数据
  const chartData = useMemo(() => {
    if (!data || !selectedKPIConfig) {
      return { xData: [], series: [] };
    }

    const xData: string[] = [];
    const series: Array<{ name: string; data: number[]; color: string; lineStyle?: { type: string } }> = [];

    // 处理Today数据
    if (data.today && data.today.length > 0) {
      if (xData.length === 0) {
        data.today.forEach((point: { time: string; value: number }) => {
          const date = new Date(point.time);
          if (viewMode === 'day') {
            xData.push(`${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`);
          } else {
            xData.push(`${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`);
          }
        });
      }
      series.push({
        name: t('dashboard.timeRange.today'),
        data: data.today.map((point: { time: string; value: number }) => point.value),
        color: selectedKPIConfig.color,
      });
    }

    // 处理Yesterday数据
    if (data.yesterday && data.yesterday.length > 0) {
      if (xData.length === 0) {
        data.yesterday.forEach((point: { time: string; value: number }) => {
          const date = new Date(point.time);
          if (viewMode === 'day') {
            xData.push(`${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`);
          } else {
            xData.push(`${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`);
          }
        });
      }
      series.push({
        name: t('dashboard.timeRange.yesterday'),
        data: data.yesterday.map((point: { time: string; value: number }) => point.value),
        color: selectedKPIConfig.color,
        lineStyle: { type: 'dashed' },
      });
    }

    return { xData, series };
  }, [data, selectedKPIConfig, viewMode, t]);

  // 如果没有配置，不渲染
  if (!panelConfig) {
    return null;
  }

  const hasData = chartData.xData.length > 0;
  const hasSelection = timeRangeSelection.length > 0;
  const panelTitle = t(panelConfig.title);

  // 指标下拉选项
  const indicatorOptions = panelConfig.indicators.map(ind => ({
    label: t(ind.label),
    value: ind.key,
  }));

  return (
    <Card
      className={className}
      style={{ width: '100%', height: height }}
      styles={{
        body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' }
      }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          {/* Panel Icon */}
          <div style={{
            width: 32,
            height: 32,
            borderRadius: 8,
            background: `${PANEL_COLORS[panelType]}15`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 16,
            color: PANEL_COLORS[panelType],
          }}>
            {PANEL_ICONS[panelType]}
          </div>
          <Text style={{ fontSize: 15, fontWeight: 600, color: token.colorText }}>
            {panelTitle}
          </Text>
        </div>
      }
      extra={
        showQuickActions ? (
          <Dropdown
            menu={{ items: quickActionItems }}
            trigger={['click']}
          >
            <Button type="text" icon={< ThunderboardOutlined />} size="small" />
          </Dropdown>
        ) : null
      }
    >
      {/* 控制区域 - 优化布局 */}
      <div style={{
        padding: '12px 16px',
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
        background: token.colorBgContainer,
      }}>
        {/* 第一行：指标选择 + 视图切换 */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 8
        }}>
          <Select
            value={selectedIndicator}
            onChange={handleIndicatorChange}
            options={indicatorOptions}
            style={{ minWidth: 160 }}
            size="small"
            placeholder={t('dashboard.selectIndicator')}
          />

          <Segmented
            value={viewMode}
            onChange={handleViewModeChange}
            options={[
              { label: '天', value: 'day' },
              { label: '周', value: 'week' },
            ]}
            size="small"
          />
        </div>

        {/* 第二行：时间范围选择 */}
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Text type="secondary" style={{ fontSize: 12, marginRight: 8 }}>
            对比：
          </Text>
          <Checkbox.Group
            value={timeRangeSelection}
            onChange={handleTimeRangeChange}
            style={{ display: 'flex', gap: 12 }}
          >
            <Checkbox value="today" style={{ marginLeft: 0 }}>
              <Text style={{ fontSize: 12 }}>今天</Text>
            </Checkbox>
            <Checkbox value="yesterday">
              <Text style={{ fontSize: 12 }}>昨天</Text>
            </Checkbox>
          </Checkbox.Group>
        </div>
      </div>

      {/* 时间对比图例 */}
      {(hasSelection && hasData) && (
        <TimeRangeLegend
          timeRangeSelection={timeRangeSelection}
          selectedKPIConfig={selectedKPIConfig}
        />
      )}

      {/* 图表区域 */}
      <div style={{ flex: 1, minHeight: height - 140 }}>
        {isLoading ? (
          <div style={{ height: height - 140, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Spin indicator={<LoadingOutlined spin />} tip={t('common.loading')} />
          </div>
        ) : !hasSelection || !hasData ? (
          <div style={{ height: height - 140, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24 }}>
            <SmartEmptyState
              timeRangeSelection={timeRangeSelection}
              viewMode={viewMode}
              panelTitle={panelTitle}
              onSelectTimeRange={setTimeRangeSelection}
            />
          </div>
        ) : (
          <LineChart
            key={`${technology}-${panelType}-${selectedIndicator}-${chartData.xData.length}`}
            title=""
            xData={chartData.xData}
            series={chartData.series}
            height={height - 140}
            areaFill={timeRangeSelection.length === 1 && timeRangeSelection[0] === 'today'}
            smooth
            showLegend={timeRangeSelection.length > 1}
            unit={selectedKPIConfig?.unit ? t(selectedKPIConfig.unit) : ''}
          />
        )}
      </div>
    </Card>
  );
}

export default KPIPanelV2;
