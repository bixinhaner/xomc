import { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  message,
  Segmented,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DownloadOutlined, ProfileOutlined } from '@ant-design/icons';
import { useExportTaskCSV, useExportTaskDeviceCSV } from '@core/hooks/api/useMmlConsole';
import { useT } from '@/hooks/useT';
import type {
  ExecMeta,
  ExecStatus,
  ResultColumn,
  ResultRow,
  UnverifiedReason,
} from '../types';
import { STATUS_META, UNVERIFIED_REASON_TEXT } from '../constants';
import { exportAll, exportOne, downloadFromUrl } from '../download';
import ResultDetailModal from './ResultDetailModal';

const { Text } = Typography;

interface ResultTableProps {
  execMeta: ExecMeta | null;
  /** 当前记录的命令 ID（= mml_tasks.id，传给详情页展示/深链） */
  commandId: string | null;
  columns: ResultColumn[];
  rows: ResultRow[];
  running: boolean;
  hasExecuted: boolean;
}

type StatusFilter = 'all' | 'success' | 'failed';

/** 状态 Tag；unverified 悬浮显示原因（只写/重启生效/查询失败，设计 §3.11.2）。 */
function StatusTag({ status, reason }: { status: ExecStatus; reason?: UnverifiedReason }) {
  const meta = STATUS_META[status];
  const tag = <Tag color={meta.color}>{meta.text}</Tag>;
  if (status === 'unverified' && reason) {
    return <Tooltip title={UNVERIFIED_REASON_TEXT[reason]}>{tag}</Tooltip>;
  }
  return tag;
}

/**
 * 右侧执行结果表格（设计 §3.4，本次改版核心）—— 行=设备、列=参数路径（读）/状态+故障（写）。
 * 顶部汇总条（总数/成功/失败/用时 + 状态过滤 + 设备过滤 + 下载全部），失败行可展开看原始报文，
 * 行尾单设备下载，底部折叠保留全量原始报文。当前为 mock 数据驱动。
 */
export default function ResultTable({
  execMeta,
  commandId,
  columns,
  rows,
  running,
  hasExecuted,
}: ResultTableProps) {
  const t = useT();
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [snKeyword, setSnKeyword] = useState('');
  const [viewingRow, setViewingRow] = useState<ResultRow | null>(null);

  // CSV 导出走后端（落 MinIO + 记入 mml_tasks），拿预签名 URL 触发浏览器下载。
  const exportCsv = useExportTaskCSV();
  const exportDeviceCsv = useExportTaskDeviceCSV();

  const handleExportAllCsv = (): void => {
    if (!commandId) {
      // 无真实任务 ID（理论不达），回退客户端导出。
      exportAll('csv', columns, rows, execMeta?.label ?? 'result');
      return;
    }
    exportCsv.mutate(commandId, {
      onSuccess: ({ downloadUrl }) => {
        const name = `${execMeta?.commandName ?? execMeta?.label ?? 'mml-result'}-汇总.csv`;
        downloadFromUrl(downloadUrl, name)
          .then(() => void message.success('已下载汇总 CSV'))
          .catch((e) => void message.error(e instanceof Error ? e.message : '下载失败'));
      },
      onError: (e) => void message.error(e instanceof Error ? e.message : '导出失败'),
    });
  };

  const handleExportDeviceCsv = (deviceSn: string): void => {
    if (!commandId) {
      exportOne(columns, rows.find((r) => r.deviceSn === deviceSn)!);
      return;
    }
    exportDeviceCsv.mutate(
      { taskId: commandId, deviceSn },
      {
        onSuccess: ({ downloadUrl }) => {
          downloadFromUrl(downloadUrl, `${deviceSn}.csv`)
            .then(() => void message.success(`已下载设备 ${deviceSn} 的 CSV`))
            .catch((e) => void message.error(e instanceof Error ? e.message : '下载失败'));
        },
        onError: (e) => void message.error(e instanceof Error ? e.message : '导出失败'),
      },
    );
  };

  const stats = useMemo(() => {
    const total = rows.length;
    // 「执行中」= 待执行 + 执行中两态的设备行数（§3.11.7 需求②）
    const inProgress = rows.filter((r) => r.status === 'pending' || r.status === 'running').length;
    const success = rows.filter((r) => r.status === 'success').length;
    const unverified = rows.filter((r) => r.status === 'unverified').length;
    const mismatch = rows.filter((r) => r.status === 'mismatch').length;
    const failed = rows.filter((r) => r.status === 'failed').length;
    // 「问题行」= RPC 失败 + 核实未生效（mismatch）
    const problem = failed + mismatch;
    return { total, inProgress, success, unverified, mismatch, failed, problem };
  }, [rows]);

  // 写类（读后核实）才展示「未核实/未生效」统计；读类只有成功/失败。
  const isWrite = execMeta ? !execMeta.read : false;

  const filteredRows = useMemo(() => {
    const kw = snKeyword.trim().toLowerCase();
    return rows.filter((r) => {
      if (statusFilter === 'success' && r.status !== 'success') return false;
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
        width: 120,
        fixed: 'left',
        render: (s: ExecStatus, r) => <StatusTag status={s} reason={r.unverifiedReason} />,
      },
    ];

    const dynamic: ColumnsType<ResultRow> = columns.map((c) => ({
      title: c.label,
      key: c.key,
      width: 140,
      ellipsis: true,
      render: (_v, r) => {
        const val = r.cells[c.path];
        if (r.status === 'failed') {
          // 逐 PATH：失败行仍含成功 path 的读回值与失败 path 的「✗ 失败」标记，逐格呈现；
          // 整体下发失败（无单格值）回退「-」。
          if (val === '✗ 失败') return <Text type="danger">{val}</Text>;
          return val ? <Text>{val}</Text> : <Text type="secondary">-</Text>;
        }
        // 未核实（只写/重启生效）：无读回值，灰显占位
        if (r.status === 'unverified') {
          return (
            <Tooltip title="未核实，以读回为准时无值">
              <Text type="secondary">—</Text>
            </Tooltip>
          );
        }
        // 未生效（写成功但读回不符）：读回值标红
        if (r.status === 'mismatch') {
          return val ? <Text type="danger">{val}</Text> : <Text type="secondary">-</Text>;
        }
        if (val === '✓') return <Tag color="success">✓</Tag>;
        // 成功但读回值为空（设备返回空串/无值）：显示「空」（多语言），区别于「-」（无结果）。
        if (val === '' || val == null) {
          return <Text type="secondary">{t('mml.console.emptyValue')}</Text>;
        }
        return <Text>{val}</Text>;
      },
    }));

    const tail: ColumnsType<ResultRow> = [
      {
        title: '下发时间',
        dataIndex: 'dispatchedAt',
        key: 'dispatchedAt',
        width: 104,
        fixed: 'right',
        render: (v?: string) =>
          v ? <Text style={{ fontSize: 12 }}>{v}</Text> : <Text type="secondary">-</Text>,
      },
      {
        title: '响应时间',
        dataIndex: 'respondedAt',
        key: 'respondedAt',
        width: 104,
        fixed: 'right',
        render: (v?: string) =>
          v ? <Text style={{ fontSize: 12 }}>{v}</Text> : <Text type="secondary">-</Text>,
      },
      {
        title: '操作',
        key: 'action',
        width: 110,
        fixed: 'right',
        align: 'center',
        render: (_v, r) => (
          <Space size={0}>
            <Tooltip title="查看">
              <Button
                type="text"
                size="small"
                icon={<ProfileOutlined />}
                onClick={() => setViewingRow(r)}
              />
            </Tooltip>
            <Tooltip title="下载该设备结果">
              <Button
                type="text"
                size="small"
                icon={<DownloadOutlined />}
                loading={exportDeviceCsv.isPending && exportDeviceCsv.variables?.deviceSn === r.deviceSn}
                onClick={() => handleExportDeviceCsv(r.deviceSn)}
              />
            </Tooltip>
          </Space>
        ),
      },
    ];

    return [...base, ...dynamic, ...tail];
    // commandId / 导出 mutation 进依赖：切任务或导出 loading 变化时刷新「操作」列。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [columns, commandId, exportDeviceCsv.isPending, exportDeviceCsv.variables]);

  return (
    <Card
      title={
        // 「执行结果」标题 + 当前命令名 + 汇总统计同一行（设计 §3.11.1 / §3.11.7 修订）
        <Space size={20} wrap style={{ rowGap: 4 }}>
          <Space size={6}>
            <span>执行结果</span>
            {hasExecuted && execMeta && (
              <Text type="secondary" style={{ fontWeight: 400, fontSize: 14 }}>
                · {execMeta.commandName ?? execMeta.label}
              </Text>
            )}
          </Space>
          {hasExecuted && (
            <Space size={14} wrap style={{ fontWeight: 400, fontSize: 13 }}>
              <span>
                设备总数 <b>{stats.total}</b>
              </span>
              <span style={{ color: '#1677ff' }}>执行中 {stats.inProgress}</span>
              <span style={{ color: '#52c41a' }}>成功 {stats.success}</span>
              {isWrite && <span style={{ color: '#faad14' }}>未核实 {stats.unverified}</span>}
              {isWrite && <span style={{ color: '#ff4d4f' }}>未生效 {stats.mismatch}</span>}
              <span style={{ color: '#ff4d4f' }}>失败 {stats.failed}</span>
            </Space>
          )}
        </Space>
      }
      variant="borderless"
      style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
      styles={{ body: { padding: 16, flex: 1, minHeight: 0, overflow: 'auto' } }}
      extra={
        <Button
          icon={<DownloadOutlined />}
          disabled={rows.length === 0}
          loading={exportCsv.isPending}
          onClick={handleExportAllCsv}
        >
          下载全部
        </Button>
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
          <Space wrap style={{ justifyContent: 'space-between', width: '100%' }}>
            <Segmented<StatusFilter>
              value={statusFilter}
              onChange={(v) => setStatusFilter(v as StatusFilter)}
              options={[
                { label: `全部(${stats.total})`, value: 'all' },
                { label: `成功(${stats.success})`, value: 'success' },
                { label: `失败(${stats.failed})`, value: 'failed' },
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
          />
        </Space>
      )}

      <ResultDetailModal
        open={!!viewingRow}
        row={viewingRow}
        execMeta={execMeta}
        commandId={commandId}
        columns={columns}
        onClose={() => setViewingRow(null)}
      />
    </Card>
  );
}
