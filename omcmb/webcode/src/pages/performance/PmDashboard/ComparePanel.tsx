/**
 * 对比模式视觉指示器 — 显示在 Panel 标题旁边。
 *
 * 实际的"周期对比 / 多设备对比"渲染逻辑由 PanelRenderer 内 series 多条数据承载；
 * 本组件仅提供 Tag + 问号 tooltip，让用户一眼看出 panel 对比模式 + hover 看动态说明。
 *
 * Tooltip 内容按 windowOffset 动态生成（如 -24h → "对比昨天同时段"），见 dashboardUtils.describeCompareMode。
 * 兼容旧数据：same_window_other_devices 显示"已废弃"红 Tag，不破老 panel。
 */

import { Tag, Tooltip } from 'antd';
import { HistoryOutlined, QuestionCircleOutlined, WarningOutlined } from '@ant-design/icons';
import type { CompareMode } from '@core/types/pmDashboard';
import { describeCompareMode } from './dashboardUtils';

interface Props {
  mode?: CompareMode;
  /** panel.timeRange.start_offset，用于按时间窗动态描述对比含义 */
  windowOffset?: string;
}

export function ComparePanel({ mode, windowOffset }: Props) {
  if (!mode) return null;
  const desc = describeCompareMode(mode, windowOffset);
  if (mode === 'same_window_other_devices') {
    return (
      <Tooltip title={desc}>
        <Tag icon={<WarningOutlined />} color="red" style={{ cursor: 'help' }}>
          对比已废弃 <QuestionCircleOutlined style={{ marginLeft: 2 }} />
        </Tag>
      </Tooltip>
    );
  }
  return (
    <Tooltip title={desc}>
      <Tag icon={<HistoryOutlined />} color="blue" style={{ cursor: 'help' }}>
        周期对比 <QuestionCircleOutlined style={{ marginLeft: 2 }} />
      </Tag>
    </Tooltip>
  );
}
