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
import { pivotLongToWide, type PivotColumn, type PivotRow } from '@core/utils/pmPivotTransform';

const { Text } = Typography;

interface PivotTableProps {
  rows: AggregatedRow[];
  loading?: boolean;
}

function formatNumber(v: number | null | undefined): string {
  if (v === null || v === undefined) return '-';
  if (Number.isInteger(v)) return String(v);
  return v.toFixed(4).replace(/\.?0+$/, '');
}

export default function PivotTable({ rows, loading }: PivotTableProps) {
  const pivoted = useMemo(() => pivotLongToWide(rows), [rows]);

  const columns: ColumnsType<PivotRow> = useMemo(() => {
    // 紧凑宽度 — 匹配 (标题, 内容) 中较长者的实际 px 占用
    const fixed: ColumnsType<PivotRow> = [
      {
        title: '时间',
        dataIndex: 'time',
        key: 'time',
        width: 140, // "2026-05-26 19:00" ≈ 130px
        render: (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm'),
      },
      {
        title: '设备 SN',
        dataIndex: 'deviceSn',
        key: 'deviceSn',
        width: 180, // 19-char SN ≈ 165px
        ellipsis: true,
        render: (v?: string) => v ?? '-',
      },
      {
        title: 'Cell ID',
        dataIndex: 'cellId',
        key: 'cellId',
        width: 100, // 9-digit ID ≈ 80px
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
        width: 80, // 5-digit PLMN
        render: (v?: string) => v ?? '-',
      },
    ];

    const metricCols: ColumnsType<PivotRow> = pivoted.columns.map((c: PivotColumn) => ({
      title: <Tooltip title={c.title}><span style={{ display: 'inline-block', maxWidth: '100%', overflow: 'hidden', textOverflow: 'ellipsis' }}>{c.title}</span></Tooltip>,
      key: c.key,
      width: 160, // 平均指标路径宽度 ≈ 130-150px，留少量余量
      align: 'right',
      ellipsis: true,
      render: (_: unknown, row: PivotRow) => formatNumber(row.cells[c.key]),
    }));

    return [...fixed, ...metricCols];
  }, [pivoted.columns]);

  // 计算总宽度用于 scroll.x，超出 viewport 时横向滚动
  const totalWidth = 140 + 180 + 100 + 80 + pivoted.columns.length * 160;

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
