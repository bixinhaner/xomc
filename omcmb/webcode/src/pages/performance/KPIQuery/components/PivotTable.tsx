/**
 * T-0174 PM 透视表（long→wide）：行=时间 / 列=N 指标 / 单元格=metric_value。
 *
 * 多设备 / 多 LDN 自动给列加 [SN / LDN] 后缀；缺采单元格显示 "-"。
 */

import { useMemo } from 'react';
import { Table, Tooltip, Empty, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { AggregatedRow } from '@core/types/pmDashboard';
import { pivotLongToWide, type PivotColumn } from '@core/utils/pmPivotTransform';

const { Text } = Typography;

interface PivotTableProps {
  rows: AggregatedRow[];
  loading?: boolean;
}

interface TableRowData {
  key: string;
  time: string;
  [columnKey: string]: string | number | null;
}

function formatNumber(v: number | null | undefined): string {
  if (v === null || v === undefined) return '-';
  if (Number.isInteger(v)) return String(v);
  return v.toFixed(4).replace(/\.?0+$/, '');
}

export default function PivotTable({ rows, loading }: PivotTableProps) {
  const pivoted = useMemo(() => pivotLongToWide(rows), [rows]);

  const dataSource: TableRowData[] = useMemo(
    () =>
      pivoted.rows.map((r) => ({
        key: r.time,
        time: r.time,
        ...r.cells,
      })),
    [pivoted.rows],
  );

  const columns: ColumnsType<TableRowData> = useMemo(() => {
    const timeCol: ColumnsType<TableRowData>[number] = {
      title: '时间',
      dataIndex: 'time',
      key: 'time',
      width: 180,
      fixed: 'left',
      render: (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm'),
    };
    const metricCols: ColumnsType<TableRowData> = pivoted.columns.map((c: PivotColumn) => ({
      title: (
        <Tooltip title={c.tooltipFull}>
          <span>{c.title}</span>
        </Tooltip>
      ),
      dataIndex: c.key,
      key: c.key,
      width: 160,
      align: 'right',
      render: (v: unknown) => formatNumber(v as number | null),
    }));
    return [timeCol, ...metricCols];
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
    <Table<TableRowData>
      rowKey="key"
      size="small"
      loading={loading}
      columns={columns}
      dataSource={dataSource}
      pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 行` }}
      scroll={{ x: 'max-content', y: 480 }}
      bordered
    />
  );
}
