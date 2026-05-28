/**
 * G6 dashboard editor 内的 Panel 网格容器。
 *
 * v2 实现：基于 react-grid-layout 支持拖拽 + resize。
 * - canEdit=false 时禁用拖拽和 resize（内置 dashboard / 只读分享）
 * - 拖动 / resize 完成后通过 store.replaceLayout 落回 currentDashboard.layout
 * - 上层 DashboardEditorPane 的自动保存 effect 监听 unsavedChanges 触发 PUT
 *
 * G6-Gap-6：当 panel.granularities.length > 1 时，Card title 下方显示 Tab；
 * 切 Tab 只切换 PanelRenderer 的 activeGranularity prop（不重发 panel CRUD）。
 */

import { useMemo, useState } from 'react';
import { Empty, Card, Button, Space, Tabs, Tooltip } from 'antd';
import { DeleteOutlined, SettingOutlined, DownloadOutlined } from '@ant-design/icons';
import { Responsive, WidthProvider, type Layout } from 'react-grid-layout/legacy';
import type { Granularity, Panel, PanelGridItem } from '@core/types/pmDashboard';
import { usePmDashboardStore } from '@core/store/pmDashboardStore';
import { usePmPanelData } from '@core/hooks/api/usePmPanelData';
import { exportWorkbook } from '@core/utils/excelExport';
import { PanelRenderer } from './PanelRenderer';
import { ComparePanel } from './ComparePanel';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';
import './PanelGrid.css';

const ResponsiveGridLayout = WidthProvider(Responsive);

interface Props {
  panels: Panel[];
  /**
   * 是否允许编辑（= isOwner && !isBuiltin）。
   * 替代旧的 editMode：去编辑模式后只要 canEdit=true，panel 操作 (⚙/🗑) 与添加按钮即常驻显示。
   */
  canEdit: boolean;
  /** 内置 dashboard 标记 — 决定空状态文案分流（引导派生 vs 添加 Panel）。 */
  isBuiltin?: boolean;
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

// 12 列基础栅格（与 antd 习惯保持一致）；行高 50px，gap 8px
const GRID_COLS = 12;
const GRID_ROW_HEIGHT = 50;

function PanelCard({
  panel,
  canEdit,
  onConfigPanel,
  onDeletePanel,
}: {
  panel: Panel;
  canEdit: boolean;
  onConfigPanel?: (panel: Panel) => void;
  onDeletePanel?: (panel: Panel) => void;
}) {
  const grans = panel.granularities ?? [];
  const [active, setActive] = useState<Granularity>(grans[0] ?? 'hourly');
  const showTabs = grans.length > 1;

  const panelData = usePmPanelData(panel, active);
  const handleExportExcel = () => {
    const sheets = (panel.granularities ?? [active]).map((g) => {
      if (g !== active) {
        return { name: g, rows: [{ note: '切到该粒度 Tab 后再次点击导出可获取本粒度数据' }] };
      }
      const buckets = panelData.series[0]?.points.map((p) => p.label) ?? [];
      const rows = buckets.map((label, i) => {
        // 列名 / 缺采占位与页面表格 (PanelRenderer.TableRenderer) 保持一致
        const row: Record<string, string | number | null> = { 时间: label };
        panelData.series.forEach((s) => {
          const v = s.points[i]?.value;
          row[s.name] = v === null || v === undefined ? '缺采' : v;
        });
        return row;
      });
      return { name: g, rows };
    });
    exportWorkbook(`panel_${panel.title}_${panel.id.slice(0, 8)}`, sheets);
  };

  return (
    <Card
      title={
        <Space size={6}>
          <span>{panel.title}</span>
          <ComparePanel
            mode={panel.compareMode}
            windowOffset={'start_offset' in panel.timeRange ? panel.timeRange.start_offset : undefined}
          />
        </Space>
      }
      size="small"
      // 让标题区可作拖拽把手；按钮区域用 no-drag 防止误拖
      classNames={{ header: 'panel-drag-handle' }}
      extra={
        <Space className="panel-no-drag" onMouseDown={(e) => e.stopPropagation()}>
          <Tooltip title={`导出当前粒度 (${active}) 数据 Excel`}>
            <Button
              size="small"
              type="text"
              icon={<DownloadOutlined />}
              onClick={handleExportExcel}
            />
          </Tooltip>
          {canEdit && (
            <>
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
            </>
          )}
        </Space>
      }
      styles={{ body: { minHeight: 200, height: 'calc(100% - 40px)', overflow: 'auto' } }}
      style={{ height: '100%' }}
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
  );
}

function toRglLayout(items: PanelGridItem[]): Layout[] {
  return items.map((it) => ({
    i: it.i,
    x: Number.isFinite(it.x) ? it.x : 0,
    y: Number.isFinite(it.y) ? it.y : 0,
    w: Math.max(2, Math.min(GRID_COLS, it.w ?? 6)),
    h: Math.max(3, it.h ?? 4),
    minW: it.minW ?? 2,
    minH: it.minH ?? 3,
    static: it.static,
  }));
}

function fromRglLayout(layouts: Layout[]): PanelGridItem[] {
  return layouts.map((l) => ({
    i: l.i,
    x: l.x,
    y: l.y,
    w: l.w,
    h: l.h,
  }));
}

export function PanelGrid({ panels, canEdit, isBuiltin, onConfigPanel, onDeletePanel }: Props) {
  const layoutItems = usePmDashboardStore((s) => s.currentDashboard?.layout.panels ?? []);
  const replaceLayout = usePmDashboardStore((s) => s.replaceLayout);

  // panel 没有 layout 时补一个默认半宽布局：行内先填左半(x=0)再填右半(x=6)，
  // 每两个换行（而非一律塞 x=0 纵向堆在左列）。
  const effectiveLayout = useMemo(() => {
    const known = new Map(layoutItems.map((l) => [l.i, l] as const));
    const baseY = layoutItems.reduce((m, l) => Math.max(m, l.y + l.h), 0);
    const half = GRID_COLS / 2;
    const out: PanelGridItem[] = [];
    let slot = 0; // 新 panel 的填充序号：偶数=左半，奇数=右半
    panels.forEach((p) => {
      const existing = known.get(p.id);
      if (existing) {
        out.push(existing);
      } else {
        out.push({
          i: p.id,
          x: slot % 2 === 0 ? 0 : half,
          y: baseY + Math.floor(slot / 2) * 4,
          w: half,
          h: 4,
        });
        slot += 1;
      }
    });
    return out;
  }, [panels, layoutItems]);

  if (panels.length === 0) {
    if (isBuiltin) {
      return (
        <Empty
          description={
            <Space direction="vertical" align="center" size={4}>
              <span>系统内置 dashboard 只读</span>
              <span style={{ color: '#999', fontSize: 12 }}>请点击右上角"派生"复制后编辑</span>
            </Space>
          }
        />
      );
    }
    return <Empty description='还没有 panel，点上方"+ 添加 Panel"创建第一个' />;
  }

  const rglLayout = toRglLayout(effectiveLayout);

  const handleLayoutChange = (next: Layout[]) => {
    if (!canEdit) return;
    // 仅在确实有变化时落库，避免初次挂载就触发 unsavedChanges
    const prev = new Map(effectiveLayout.map((l) => [l.i, l]));
    const changed = next.some((n) => {
      const p = prev.get(n.i);
      return !p || p.x !== n.x || p.y !== n.y || p.w !== n.w || p.h !== n.h;
    });
    if (!changed) return;
    replaceLayout(fromRglLayout(next));
  };

  return (
    <ResponsiveGridLayout
      className="layout"
      layouts={{ lg: rglLayout, md: rglLayout, sm: rglLayout, xs: rglLayout, xxs: rglLayout }}
      breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
      cols={{ lg: GRID_COLS, md: GRID_COLS, sm: GRID_COLS, xs: 4, xxs: 2 }}
      rowHeight={GRID_ROW_HEIGHT}
      margin={[12, 12]}
      isDraggable={canEdit}
      isResizable={canEdit}
      draggableHandle=".panel-drag-handle"
      onLayoutChange={handleLayoutChange}
      compactType="vertical"
    >
      {panels.map((p) => (
        <div key={p.id}>
          <PanelCard
            panel={p}
            canEdit={canEdit}
            onConfigPanel={onConfigPanel}
            onDeletePanel={onDeletePanel}
          />
        </div>
      ))}
    </ResponsiveGridLayout>
  );
}
