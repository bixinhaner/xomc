/**
 * T-0174 PM 透视表（long→wide）。
 *
 * 固定左列：时间 / 设备 SN / Cell ID / PLMN（始终显示，因为后端 (sn, time, object_ldn) 三元组才唯一）。
 * 动态列：用户选的 N 个指标。
 * 缺采单元格显示 "-"。
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
    const fixed: ColumnsType<PivotRow> = [
      {
        title: '时间',
        dataIndex: 'time',
        key: 'time',
        width: 160,
        fixed: 'left',
        render: (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm'),
      },
      {
        title: '设备 SN',
        dataIndex: 'deviceSn',
        key: 'deviceSn',
        width: 200,
        fixed: 'left',
        ellipsis: true,
        render: (v?: string) => v ?? '-',
      },
      {
        title: 'Cell ID',
        dataIndex: 'cellId',
        key: 'cellId',
        width: 120,
        fixed: 'left',
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
        width: 100,
        fixed: 'left',
        render: (v?: string) => v ?? '-',
      },
    ];

    const metricCols: ColumnsType<PivotRow> = pivoted.columns.map((c: PivotColumn) => ({
      title: c.title,
      key: c.key,
      width: 160,
      align: 'right',
      render: (_: unknown, row: PivotRow) => formatNumber(row.cells[c.key]),
    }));

    return [...fixed, ...metricCols];
  }, [pivoted.columns]);

  if (!loading && pivoted.rows.length === 0) {
    return (
      <Empty
        description={<Text type="secondary">暂无数据，请选择查询条件后点击"查询"</Text>}
        style={{ padding: '60px 0' }}
      />
    );
  }

  return (
    <Table<PivotRow>
      rowKey="key"
      size="small"
      loading={loading}
      columns={columns}
      dataSource={pivoted.rows}
      pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 行` }}
      scroll={{ x: 'max-content', y: 480 }}
      bordered
    />
  );
}
