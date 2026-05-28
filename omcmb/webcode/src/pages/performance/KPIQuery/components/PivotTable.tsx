/**
 * T-0174 PM 透视表（long→wide）。
 *
 * 列宽根据 max(标题, 内容) 自适应（tableLayout='auto'）。
 * 固定左 4 列：时间 / 设备 SN / Cell ID / PLMN（因为 (sn, time, object_ldn) 才是唯一键）。
 * 动态右列：用户选的 N 个指标。缺采单元格显示 "-"。
 */

import { useMemo } from 'react';
import { Table, Tooltip, Empty, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { AggregatedRow } from '@core/types/pmDashboard';
import { pivotLongToWide, formatPivotNumber, type PivotColumn, type PivotRow } from '@core/utils/pmPivotTransform';

const { Text } = Typography;

interface PivotTableProps {
  rows: AggregatedRow[];
  loading?: boolean;
}

// 固定 5 列宽度（开始时间 / 结束时间 / 设备 SN / Cell ID / PLMN）
const FIXED_COL_WIDTHS = {
  startTime: 160,
  endTime: 160,
  deviceSn: 180,
  cellId: 100,
  plmn: 80,
} as const;

const FIXED_COL_TOTAL =
  FIXED_COL_WIDTHS.startTime +
  FIXED_COL_WIDTHS.endTime +
  FIXED_COL_WIDTHS.deviceSn +
  FIXED_COL_WIDTHS.cellId +
  FIXED_COL_WIDTHS.plmn;

// 指标列标题宽度估算：字体 14px，英文/数字/标点 ≈ 8px，中文 ≈ 14px；左右 padding 共 32px。
function estimateMetricColWidth(title: string): number {
  let textWidth = 0;
  for (const ch of title) {
    textWidth += /[一-鿿]/.test(ch) ? 14 : 8;
  }
  return Math.max(120, textWidth + 32);
}

export default function PivotTable({ rows, loading }: PivotTableProps) {
  const pivoted = useMemo(() => pivotLongToWide(rows), [rows]);

  const columns: ColumnsType<PivotRow> = useMemo(() => {
    // 固定列：开始时间 / 结束时间 / 设备 SN / Cell ID / PLMN
    const fixed: ColumnsType<PivotRow> = [
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
    ];

    // 指标列：标题单行不换行不截断，列宽按列名字符宽度估算自适应
    const metricCols: ColumnsType<PivotRow> = pivoted.columns.map((c: PivotColumn) => ({
      title: <span style={{ whiteSpace: 'nowrap' }}>{c.title}</span>,
      key: c.key,
      width: estimateMetricColWidth(c.title),
      align: 'right',
      render: (_: unknown, row: PivotRow) => formatPivotNumber(row.cells[c.key]),
    }));

    return [...fixed, ...metricCols];
  }, [pivoted.columns]);

  // 总宽 = 固定 5 列 + 所有指标列累加
  const totalWidth = useMemo(
    () =>
      FIXED_COL_TOTAL +
      pivoted.columns.reduce((sum, c) => sum + estimateMetricColWidth(c.title), 0),
    [pivoted.columns],
  );

  if (!loading && pivoted.rows.length === 0) {
    return (
      <Empty
        description={<Text type="secondary">暂无数据，请选择查询条件后点击"查询"</Text>}
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
      `}</style>
      <Table<PivotRow>
        className="kpi-pivot-table"
        rowKey="key"
        size="small"
        loading={loading}
        columns={columns}
        dataSource={pivoted.rows}
        pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 行` }}
        tableLayout="fixed"
        scroll={{ x: totalWidth }}
        bordered
      />
    </>
  );
}
