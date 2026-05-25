/**
 * G6 dashboard editor 内的 Panel 网格容器。
 *
 * v1 实现：基于 antd Row/Col 12 栅格的静态布局（不支持拖拽）。
 * v2 计划：加入 react-grid-layout 支持拖拽 + resize。
 *
 * 设计 trade-off：v1 不带依赖少 + 快速可用；v2 升级时只改本文件 + store 的
 * updatePanelLayout/replaceLayout 直接对接 react-grid-layout 的 onLayoutChange。
 *
 * G6-Gap-6：当 panel.granularities.length > 1 时，Card title 下方显示 Tab；
 * 切 Tab 只切换 PanelRenderer 的 activeGranularity prop（不重发 panel CRUD）。
 */

import { useState } from 'react';
import { Empty, Row, Col, Card, Button, Space, Tabs } from 'antd';
import { DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import type { Granularity, Panel } from '@core/types/pmDashboard';
import { usePmDashboardStore } from '@core/store/pmDashboardStore';
import { PanelRenderer } from './PanelRenderer';
import { ComparePanel } from './ComparePanel';

interface Props {
  panels: Panel[];
  editMode: boolean;
  onConfigPanel?: (panel: Panel) => void;
  onDeletePanel?: (panel: Panel) => void;
}

const GRAN_LABEL: Record<Granularity, string> = {
  '15min': '15 分钟',
  hourly: '小时',
  daily: '日',
  weekly: '周',
  monthly: '月',
};

function PanelCard({
  panel,
  editMode,
  onConfigPanel,
  onDeletePanel,
  span,
}: {
  panel: Panel;
  editMode: boolean;
  onConfigPanel?: (panel: Panel) => void;
  onDeletePanel?: (panel: Panel) => void;
  span: number;
}) {
  const grans = panel.granularities ?? [];
  const [active, setActive] = useState<Granularity>(grans[0] ?? 'hourly');
  const showTabs = grans.length > 1;

  return (
    <Col span={span}>
      <Card
        title={
          <Space size={6}>
            <span>{panel.title}</span>
            <ComparePanel mode={panel.compareMode} />
          </Space>
        }
        size="small"
        extra={
          editMode ? (
            <Space>
              <Button
                size="small"
                type="text"
                icon={<SettingOutlined />}
                onClick={() => onConfigPanel?.(panel)}
              />
              <Button
                size="small"
                type="text"
                danger
                icon={<DeleteOutlined />}
                onClick={() => onDeletePanel?.(panel)}
              />
            </Space>
          ) : null
        }
        styles={{ body: { minHeight: 200 } }}
      >
        {showTabs && (
          <Tabs
            size="small"
            activeKey={active}
            onChange={(k) => setActive(k as Granularity)}
            items={grans.map((g) => ({ key: g, label: GRAN_LABEL[g] }))}
            style={{ marginTop: -8, marginBottom: 8 }}
          />
        )}
        <PanelRenderer panel={panel} activeGranularity={active} />
      </Card>
    </Col>
  );
}

export function PanelGrid({ panels, editMode, onConfigPanel, onDeletePanel }: Props) {
  const layout = usePmDashboardStore((s) => s.currentDashboard?.layout.panels ?? []);

  if (panels.length === 0) {
    return <Empty description='还没有 panel，点上方"+ 添加 Panel"创建第一个' />;
  }

  // 按 layout 顺序渲染；找不到 layout 的 panel 也按 panel.id 顺序兜底。
  const layoutMap = new Map(layout.map((l) => [l.i, l]));
  const ordered = [...panels].sort((a, b) => {
    const ai = layoutMap.get(a.id);
    const bi = layoutMap.get(b.id);
    if (!ai && !bi) return 0;
    if (!ai) return 1;
    if (!bi) return -1;
    return ai.y - bi.y || ai.x - bi.x;
  });

  return (
    <Row gutter={[16, 16]}>
      {ordered.map((p) => {
        const item = layoutMap.get(p.id);
        // v1：col span 由 layout.w 映射（4 = 1/3 屏；6 = 1/2；12 = 全宽）
        const span = item?.w ? Math.max(4, Math.min(24, item.w * 2)) : 12;
        return (
          <PanelCard
            key={p.id}
            panel={p}
            editMode={editMode}
            onConfigPanel={onConfigPanel}
            onDeletePanel={onDeletePanel}
            span={span}
          />
        );
      })}
    </Row>
  );
}
