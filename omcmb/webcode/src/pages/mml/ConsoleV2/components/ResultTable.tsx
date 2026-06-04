import { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Dropdown,
  Empty,
  Input,
  Segmented,
  Space,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { MenuProps } from 'antd';
import {
  DownloadOutlined,
  DownOutlined,
  ProfileOutlined,
} from '@ant-design/icons';
import type { ExecMeta, ExecStatus, ExportFormat, ResultColumn, ResultRow } from '../types';
import { STATUS_META } from '../constants';
import { exportAll, exportOne } from '../download';
import ResultDetailModal from './ResultDetailModal';

const { Text } = Typography;

interface ResultTableProps {
  execMeta: ExecMeta | null;
  columns: ResultColumn[];
  rows: ResultRow[];
  running: boolean;
  hasExecuted: boolean;
}

type StatusFilter = 'all' | 'failed';

function StatusTag({ status }: { status: ExecStatus }) {
  const meta = STATUS_META[status];
  return <Tag color={meta.color}>{meta.text}</Tag>;
}

/**
 * 右侧执行结果表格（设计 §3.4，本次改版核心）—— 行=设备、列=参数路径（读）/状态+故障（写）。
 * 顶部汇总条（总数/成功/失败/用时 + 状态过滤 + 设备过滤 + 下载全部），失败行可展开看原始报文，
 * 行尾单设备下载，底部折叠保留全量原始报文。当前为 mock 数据驱动。
 */
export default function ResultTable({
  execMeta,
  columns,
  rows,
  running,
  hasExecuted,
}: ResultTableProps) {
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [snKeyword, setSnKeyword] = useState('');
  const [viewingRow, setViewingRow] = useState<ResultRow | null>(null);

  const stats = useMemo(() => {
    const total = rows.length;
    const success = rows.filter((r) => r.status === 'success').length;
    const failed = rows.filter((r) => r.status === 'failed').length;
    const elapsed = rows.reduce((m, r) => Math.max(m, r.elapsedMs), 0);
    return { total, success, failed, elapsed };
  }, [rows]);

  const filteredRows = useMemo(() => {
    const kw = snKeyword.trim().toLowerCase();
    return rows.filter((r) => {
      if (statusFilter === 'failed' && r.status !== 'failed') return false;
      if (kw && !r.deviceSn.toLowerCase().includes(kw)) return false;
      return true;
    });
  }, [rows, statusFilter, snKeyword]);

  const tableColumns: ColumnsType<ResultRow> = useMemo(() => {
    const base: ColumnsType<ResultRow> = [
      {
        title: '序号',
        key: 'idx',
        width: 60,
        fixed: 'left',
        render: (_v, _r, i) => i + 1,
      },
      {
        title: '设备SN',
        dataIndex: 'deviceSn',
        key: 'deviceSn',
        width: 190,
        fixed: 'left',
        ellipsis: true,
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 90,
        fixed: 'left',
        render: (s: ExecStatus) => <StatusTag status={s} />,
      },
    ];

    const dynamic: ColumnsType<ResultRow> = columns.map((c) => ({
      title: c.label,
      key: c.key,
      width: 140,
      ellipsis: true,
      render: (_v, r) => {
        const val = r.cells[c.path];
        if (r.status === 'failed') return <Text type="secondary">-</Text>;
        if (val === '✓') return <Tag color="success">✓</Tag>;
        return val ?? <Text type="secondary">-</Text>;
      },
    }));

    const tail: ColumnsType<ResultRow> = [
      {
        title: '故障码',
        dataIndex: 'faultCode',
        key: 'faultCode',
        width: 150,
        render: (v?: string) => (v ? <Text type="danger">{v}</Text> : <Text type="secondary">-</Text>),
      },
      {
        title: '操作',
        key: 'action',
        width: 110,
        fixed: 'right',
        align: 'center',
        render: (_v, r) => (
          <Space size={0}>
            <Button type="link" size="small" icon={<ProfileOutlined />} onClick={() => setViewingRow(r)}>
              查看
            </Button>
            <Tooltip title="下载该设备结果(CSV)">
              <Button
                type="text"
                size="small"
                icon={<DownloadOutlined />}
                onClick={() => exportOne(columns, r)}
              />
            </Tooltip>
          </Space>
        ),
      },
    ];

    return [...base, ...dynamic, ...tail];
  }, [columns]);

  const downloadMenu: MenuProps = {
    items: (['csv', 'xlsx', 'json'] as ExportFormat[]).map((f) => ({
      key: f,
      label: f.toUpperCase(),
    })),
    onClick: ({ key }) =>
      exportAll(key as ExportFormat, columns, rows, execMeta?.label ?? 'result'),
  };

  return (
    <Card
      title="执行结果"
      variant="borderless"
      style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
      styles={{ body: { padding: 16, flex: 1, minHeight: 0, overflow: 'auto' } }}
      extra={
        <Dropdown menu={downloadMenu} disabled={rows.length === 0}>
          <Button icon={<DownloadOutlined />}>
            下载全部 <DownOutlined />
          </Button>
        </Dropdown>
      }
    >
      {!hasExecuted ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="选择设备与命令后点击「执行」，结果将在此以表格呈现"
          style={{ marginTop: 80 }}
        />
      ) : (
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Space size={32} wrap>
            <Statistic title="设备总数" value={stats.total} />
            <Statistic title="成功" value={stats.success} valueStyle={{ color: '#52c41a' }} />
            <Statistic title="失败" value={stats.failed} valueStyle={{ color: '#ff4d4f' }} />
            <Statistic
              title="最长用时"
              value={(stats.elapsed / 1000).toFixed(1)}
              suffix="s"
            />
          </Space>

          <Space wrap style={{ justifyContent: 'space-between', width: '100%' }}>
            <Segmented<StatusFilter>
              value={statusFilter}
              onChange={(v) => setStatusFilter(v as StatusFilter)}
              options={[
                { label: '全部', value: 'all' },
                { label: `仅失败(${stats.failed})`, value: 'failed' },
              ]}
            />
            <Input.Search
              allowClear
              placeholder="按设备 SN 过滤"
              style={{ width: 240 }}
              value={snKeyword}
              onChange={(e) => setSnKeyword(e.target.value)}
            />
          </Space>

          <Table<ResultRow>
            rowKey="deviceSn"
            size="small"
            loading={running}
            columns={tableColumns}
            dataSource={filteredRows}
            scroll={{ x: 'max-content' }}
            sticky
            pagination={{ pageSize: 20, size: 'small', showTotal: (t) => `共 ${t} 行` }}
            expandable={{
              rowExpandable: (r) => r.status === 'failed',
              expandedRowRender: (r) => (
                <div style={{ padding: '4px 8px' }}>
                  <Text type="danger" strong>故障：{r.faultCode}</Text>
                  <div style={{ marginTop: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      点击「查看」可查看该设备的执行结果与格式化报文。
                    </Text>
                  </div>
                </div>
              ),
            }}
          />
        </Space>
      )}

      <ResultDetailModal
        open={!!viewingRow}
        row={viewingRow}
        execMeta={execMeta}
        columns={columns}
        onClose={() => setViewingRow(null)}
      />
    </Card>
  );
}
