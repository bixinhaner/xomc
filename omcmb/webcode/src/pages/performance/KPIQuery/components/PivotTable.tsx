/**
 * T-0174 PM 透视表（long→wide）。
 *
 * 列宽：默认按 max(标题, 内容) 估算；表头分隔线可拖拽手动调整，
 *       列宽偏好按用户存 localStorage（换页/重查/刷新后保留，不同用户互不影响）。
 * 固定左 5 列：开始时间 / 结束时间 / 设备 SN / Cell ID / PLMN（因为 (sn, time, object_ldn) 才是唯一键）。
 * 动态右列：用户选的 N 个指标。缺采单元格显示 "-"。
 */

import { useCallback, useMemo, useState } from 'react';
import { Table, Tooltip, Empty, Typography } from 'antd';
import type { ColumnsType, ColumnType } from 'antd/es/table';
import { Resizable, type ResizeCallbackData } from 'react-resizable';
import dayjs from 'dayjs';
import type { AggregatedRow } from '@core/types/pmDashboard';
import { pivotLongToWide, formatPivotNumber, type PivotColumn, type PivotRow } from '@core/utils/pmPivotTransform';
import { useUserStore } from '@core/store/userStore';

const { Text } = Typography;

interface PivotTableProps {
  rows: AggregatedRow[];
  loading?: boolean;
  /**
   * 空结果时的提示文案。缺省走通用「暂无数据，请选择查询条件后点击查询」。
   * gNB 查空时调用方传更明确文案（区分「未注册厂商指标库」与「时间窗内无采样」），
   * 见 #201：5G 真机样本厂商错配导致 KPI 算不出、后端返回 items=null 的盲点。
   */
  emptyDescription?: React.ReactNode;
}

// 固定 5 列默认宽度（开始时间 / 结束时间 / 设备 SN / Cell ID / PLMN）
const FIXED_COL_WIDTHS = {
  startTime: 160,
  endTime: 160,
  deviceSn: 180,
  cellId: 100,
  plmn: 80,
} as const;

// 拖拽时的最小列宽，避免拖没了
const MIN_COL_WIDTH = 60;

// 指标列标题宽度估算：字体 14px，英文/数字/标点 ≈ 8px，中文 ≈ 14px；左右 padding 共 32px。
function estimateMetricColWidth(title: string): number {
  let textWidth = 0;
  for (const ch of title) {
    textWidth += /[一-鿿]/.test(ch) ? 14 : 8;
  }
  return Math.max(120, textWidth + 32);
}

type ColWidthMap = Record<string, number>;

function storageKeyOf(userId: string | undefined): string {
  return `pm-pivot-colwidth:${userId ?? 'anon'}`;
}

function loadWidths(key: string): ColWidthMap {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as unknown;
    if (parsed && typeof parsed === 'object') return parsed as ColWidthMap;
  } catch {
    // localStorage 不可用 / 内容损坏 → 退回默认列宽
  }
  return {};
}

function saveWidths(key: string, widths: ColWidthMap): void {
  try {
    localStorage.setItem(key, JSON.stringify(widths));
  } catch {
    // 持久化失败不影响本次拖拽生效（仅丢失下次保留）
  }
}

// 可拖拽表头单元格：右侧分隔线作为拖拽把手。
interface ResizableTitleProps extends React.HTMLAttributes<HTMLTableCellElement> {
  width?: number;
  onResize?: (e: React.SyntheticEvent, data: ResizeCallbackData) => void;
}

function ResizableTitle({ width, onResize, ...restProps }: ResizableTitleProps) {
  if (width == null) {
    return <th {...restProps} />;
  }
  return (
    <Resizable
      width={width}
      height={0}
      handle={
        <span
          className="kpi-pivot-resize-handle"
          onClick={(e) => e.stopPropagation()}
          onMouseDown={(e) => e.stopPropagation()}
        />
      }
      onResize={onResize}
      draggableOpts={{ enableUserSelectHack: false }}
    >
      <th {...restProps} />
    </Resizable>
  );
}

export default function PivotTable({ rows, loading, emptyDescription }: PivotTableProps) {
  const pivoted = useMemo(() => pivotLongToWide(rows), [rows]);

  const userId = useUserStore((s) => s.currentUser?.id);
  const storageKey = useMemo(() => storageKeyOf(userId), [userId]);
  const [widths, setWidths] = useState<ColWidthMap>(() => loadWidths(storageKey));

  // 切换用户时（storageKey 变）在渲染期重置为该用户的列宽偏好，避免用 effect 触发级联渲染
  const [prevStorageKey, setPrevStorageKey] = useState(storageKey);
  if (storageKey !== prevStorageKey) {
    setPrevStorageKey(storageKey);
    setWidths(loadWidths(storageKey));
  }

  const handleResize = useCallback(
    (key: string) => (_e: React.SyntheticEvent, { size }: ResizeCallbackData) => {
      setWidths((prev) => {
        const next = { ...prev, [key]: Math.max(MIN_COL_WIDTH, Math.round(size.width)) };
        saveWidths(storageKey, next);
        return next;
      });
    },
    [storageKey],
  );

  const columns: ColumnsType<PivotRow> = useMemo(() => {
    const base: ColumnsType<PivotRow> = [
      {
        title: '开始时间',
        dataIndex: 'startTime',
        key: 'startTime',
        width: FIXED_COL_WIDTHS.startTime,
        render: (t?: string, r?: PivotRow) => dayjs(t ?? r?.time).format('YYYY-MM-DD HH:mm'),
      },
      {
        title: '结束时间',
        dataIndex: 'endTime',
        key: 'endTime',
        width: FIXED_COL_WIDTHS.endTime,
        render: (t?: string) => (t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-'),
      },
      {
        title: '设备 SN',
        dataIndex: 'deviceSn',
        key: 'deviceSn',
        width: FIXED_COL_WIDTHS.deviceSn,
        render: (v?: string) => v ?? '-',
      },
      {
        title: 'Cell ID',
        dataIndex: 'cellId',
        key: 'cellId',
        width: FIXED_COL_WIDTHS.cellId,
        render: (v?: string, r?: PivotRow) =>
          v ?? (
            <Tooltip title={r?.objectLdn ? `原始 LDN: ${r.objectLdn}` : '无 LDN'}>
              <span>-</span>
            </Tooltip>
          ),
      },
      {
        title: 'PLMN',
        dataIndex: 'plmn',
        key: 'plmn',
        width: FIXED_COL_WIDTHS.plmn,
        render: (v?: string) => v ?? '-',
      },
      ...pivoted.columns.map((c: PivotColumn) => ({
        title: <span style={{ whiteSpace: 'nowrap' }}>{c.title}</span>,
        key: c.key,
        width: estimateMetricColWidth(c.title),
        align: 'right' as const,
        render: (_: unknown, row: PivotRow) => formatPivotNumber(row.cells[c.key]),
      })),
    ];

    // 套用已保存的用户列宽 + 挂拖拽回调
    return base.map((col) => {
      const key = String(col.key);
      const width = widths[key] ?? (col.width as number);
      return {
        ...col,
        width,
        onHeaderCell: (column: ColumnType<PivotRow>) => {
          const headerProps: ResizableTitleProps = {
            width: (column as { width?: number }).width,
            onResize: handleResize(key),
          };
          return headerProps;
        },
      };
    });
  }, [pivoted.columns, widths, handleResize]);

  // 总宽 = 各列有效宽度（含已保存的拖拽宽度）累加
  const totalWidth = useMemo(() => {
    const fixed = (Object.keys(FIXED_COL_WIDTHS) as Array<keyof typeof FIXED_COL_WIDTHS>).reduce(
      (sum, k) => sum + (widths[k] ?? FIXED_COL_WIDTHS[k]),
      0,
    );
    const metrics = pivoted.columns.reduce(
      (sum, c) => sum + (widths[c.key] ?? estimateMetricColWidth(c.title)),
      0,
    );
    return fixed + metrics;
  }, [pivoted.columns, widths]);

  if (!loading && pivoted.rows.length === 0) {
    return (
      <Empty
        description={
          emptyDescription ?? (
            <Text type="secondary">暂无数据，请选择查询条件后点击"查询"</Text>
          )
        }
        style={{ padding: '60px 0' }}
      />
    );
  }

  // antd 默认 scroll.x 行为：container > totalWidth 时按列 width 比例拉伸；container < totalWidth 时严格 width + 横滚。
  // macOS 默认 overlay scrollbar 太淡用户看不见，强制 webkit scrollbar 加深可见。
  return (
    <>
      <style>{`
        .kpi-pivot-table .ant-table-body::-webkit-scrollbar,
        .kpi-pivot-table .ant-table-content::-webkit-scrollbar {
          height: 12px;
          background: rgba(0, 0, 0, 0.04);
        }
        .kpi-pivot-table .ant-table-body::-webkit-scrollbar-thumb,
        .kpi-pivot-table .ant-table-content::-webkit-scrollbar-thumb {
          background: rgba(0, 0, 0, 0.35);
          border-radius: 6px;
        }
        .kpi-pivot-table .ant-table-body::-webkit-scrollbar-thumb:hover,
        .kpi-pivot-table .ant-table-content::-webkit-scrollbar-thumb:hover {
          background: rgba(0, 0, 0, 0.55);
        }
        .kpi-pivot-table .ant-table-thead th { position: relative; }
        .kpi-pivot-resize-handle {
          position: absolute;
          right: -5px;
          bottom: 0;
          z-index: 1;
          width: 10px;
          height: 100%;
          cursor: col-resize;
          touch-action: none;
        }
      `}</style>
      <Table<PivotRow>
        className="kpi-pivot-table"
        rowKey="key"
        size="small"
        loading={loading}
        columns={columns}
        components={{ header: { cell: ResizableTitle } }}
        dataSource={pivoted.rows}
        pagination={{ defaultPageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 行` }}
        tableLayout="fixed"
        scroll={{ x: totalWidth }}
        bordered
      />
    </>
  );
}
