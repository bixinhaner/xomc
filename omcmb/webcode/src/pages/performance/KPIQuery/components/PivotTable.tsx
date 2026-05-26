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

  // antd 默认 scroll.x 行为：当 container 宽度 > totalWidth 时，按列 width 比例拉伸填满容器；
  // 当 container 宽度 < totalWidth 时（指标列多），保留 width 严格值并出现横向滚动。
  return (
    <Table<PivotRow>
      rowKey="key"
      size="small"
      loading={loading}
      columns={columns}
      dataSource={pivoted.rows}
      pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 行` }}
      tableLayout="fixed"
      scroll={{ x: totalWidth, y: 480 }}
      bordered
    />
  );
}
