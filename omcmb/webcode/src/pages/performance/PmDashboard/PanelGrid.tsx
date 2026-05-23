/**
 * G6 dashboard editor 内的 Panel 网格容器。
 *
 * v1 实现：基于 antd Row/Col 12 栅格的静态布局（不支持拖拽）。
 * v2 计划：加入 react-grid-layout 支持拖拽 + resize。
 *
 * 设计 trade-off：v1 不带依赖少 + 快速可用；v2 升级时只改本文件 + store 的
 * updatePanelLayout/replaceLayout 直接对接 react-grid-layout 的 onLayoutChange。
 */

import { Empty, Row, Col, Card, Button, Space } from 'antd';
import { DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import type { Panel } from '@core/types/pmDashboard';
import { usePmDashboardStore } from '@core/store/pmDashboardStore';
import { PanelRenderer } from './PanelRenderer';

interface Props {
  panels: Panel[];
  editMode: boolean;
  onConfigPanel?: (panel: Panel) => void;
  onDeletePanel?: (panel: Panel) => void;
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
          <Col key={p.id} span={span}>
            <Card
              title={p.title}
              size="small"
              extra={
                editMode ? (
                  <Space>
                    <Button
                      size="small"
                      type="text"
                      icon={<SettingOutlined />}
                      onClick={() => onConfigPanel?.(p)}
                    />
                    <Button
                      size="small"
                      type="text"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={() => onDeletePanel?.(p)}
                    />
                  </Space>
                ) : null
              }
              styles={{ body: { minHeight: 200 } }}
            >
              <PanelRenderer panel={p} />
            </Card>
          </Col>
        );
      })}
    </Row>
  );
}
