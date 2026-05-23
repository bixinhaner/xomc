/**
 * 对比模式视觉指示器 — 显示在 Panel 标题旁边。
 *
 * 实际的"多设备对比 / 上一周期对比"渲染逻辑由 PanelRenderer 内 series 多条数据承载；
 * 本组件仅提供 UI tag 让用户一眼看出 panel 当前是对比模式。
 */

import { Tag, Tooltip } from 'antd';
import { SwapOutlined, HistoryOutlined } from '@ant-design/icons';
import type { CompareMode } from '@core/types/pmDashboard';

interface Props {
  mode?: CompareMode;
}

export function ComparePanel({ mode }: Props) {
  if (!mode) return null;
  if (mode === 'same_window_other_devices') {
    return (
      <Tooltip title="同时段对比其它设备 / 设备组（多 series 渲染）">
        <Tag icon={<SwapOutlined />} color="purple">
          多设备对比
        </Tag>
      </Tooltip>
    );
  }
  return (
    <Tooltip title="对比上一周期同窗口（双 series 渲染）">
      <Tag icon={<HistoryOutlined />} color="blue">
        周期对比
      </Tag>
    </Tooltip>
  );
}
