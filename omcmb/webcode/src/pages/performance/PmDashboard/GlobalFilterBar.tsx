/**
 * G6-Gap-2：共享筛选条 — 时间窗 + 设备组多选 + 设备多选。
 *
 * 用法：嵌在 DashboardEditorPane 顶部；写入 usePmDashboardStore.globalFilter；
 * panel.config.inherit_global=true 时由 PanelRenderer 消费（替代 panel 自身的 timeRange / deviceSns / deviceGroupIds）。
 *
 * UI 简化：时间窗用 Segmented 给 4 个常用 offset；设备组 / 设备走 CSV 文本输入（同 PanelConfigDrawer），
 * 后续可替换为真实的设备选择器组件。
 */

import { Card, Input, Segmented, Space, Tag, Tooltip, Button } from 'antd';
import { FilterOutlined, ReloadOutlined } from '@ant-design/icons';
import { usePmDashboardStore } from '@core/store/pmDashboardStore';

const TIME_OPTIONS = [
  { label: '近 1h', value: '-1h' },
  { label: '近 24h', value: '-24h' },
  { label: '近 7d', value: '-7d' },
  { label: '近 30d', value: '-30d' },
];

function splitCsv(s: string): string[] {
  return s
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean);
}

export function GlobalFilterBar() {
  const filter = usePmDashboardStore((s) => s.globalFilter);
  const setFilter = usePmDashboardStore((s) => s.setGlobalFilter);

  const handleReset = () => {
    setFilter({ startOffset: '-24h', deviceGroupIds: [], deviceSns: [] });
  };

  const inheritCount =
    (filter.deviceGroupIds?.length ?? 0) + (filter.deviceSns?.length ?? 0);

  return (
    <Card
      size="small"
      style={{ marginBottom: 12 }}
      styles={{ body: { padding: '8px 12px' } }}
    >
      <Space size={12} wrap>
        <Space size={4}>
          <FilterOutlined />
          <span style={{ fontSize: 12, color: '#888' }}>全局筛选</span>
          <Tooltip title="panel 配置勾选「继承全局」时使用以下筛选；不勾时用 panel 自己的筛选">
            <Tag color="blue">inherit_global</Tag>
          </Tooltip>
        </Space>

        <span>
          <span style={{ fontSize: 12, color: '#888', marginRight: 4 }}>时间窗:</span>
          <Segmented
            size="small"
            options={TIME_OPTIONS}
            value={filter.startOffset ?? '-24h'}
            onChange={(v) => setFilter({ startOffset: String(v), absoluteStart: undefined, absoluteEnd: undefined })}
          />
        </span>

        <span>
          <span style={{ fontSize: 12, color: '#888', marginRight: 4 }}>设备组:</span>
          <Input
            size="small"
            style={{ width: 200 }}
            placeholder="uuid-1, uuid-2"
            value={(filter.deviceGroupIds ?? []).join(', ')}
            onChange={(e) => setFilter({ deviceGroupIds: splitCsv(e.target.value) })}
          />
        </span>

        <span>
          <span style={{ fontSize: 12, color: '#888', marginRight: 4 }}>设备 SN:</span>
          <Input
            size="small"
            style={{ width: 240 }}
            placeholder="SN-001, SN-002"
            value={(filter.deviceSns ?? []).join(', ')}
            onChange={(e) => setFilter({ deviceSns: splitCsv(e.target.value) })}
          />
        </span>

        {inheritCount > 0 && (
          <Tag color="purple">已选 {inheritCount} 项</Tag>
        )}

        <Button size="small" icon={<ReloadOutlined />} onClick={handleReset}>
          重置
        </Button>
      </Space>
    </Card>
  );
}
